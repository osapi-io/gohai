// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/executor"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
)

type MemoryDarwinPublicTestSuite struct {
	suite.Suite
}

func TestMemoryDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryDarwinPublicTestSuite))
}

// vmStatAppleSilicon represents a typical `vm_stat` output on a
// 16K-page Apple Silicon Mac.
const vmStatAppleSilicon = `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               42000.
Pages active:                             81920.
Pages inactive:                           65536.
Pages speculative:                         8192.
Pages throttled:                              0.
Pages wired down:                         32768.
Pages purgeable:                              0.
"Translation faults":                 123456789.
Pages copy-on-write:                      1000.
Pages zero filled:                         500.
Pages reactivated:                         100.
Pages purged:                                0.
File-backed pages:                       16384.
Anonymous pages:                         65536.
Pages stored in compressor:              24576.
Pages occupied by compressor:             8192.
Decompressions:                              0.
Compressions:                                0.
Pageins:                                  1024.
Pageouts:                                    0.
Swapins:                                     0.
Swapouts:                                    0.
`

func (s *MemoryDarwinPublicTestSuite) TestCollect() {
	okVM := func(context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{
			Total:     17179869184,
			Available: 8589934592,
			Used:      8589934592,
			Free:      4294967296,
			Active:    6000,
			Inactive:  2000,
			Wired:     3000,
		}, nil
	}
	okSwap := func(context.Context) (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{Total: 0}, nil
	}
	vmErr := func(context.Context) (*mem.VirtualMemoryStat, error) {
		return nil, errors.New("mach vm error")
	}

	tests := []struct {
		name            string
		vm              func(context.Context) (*mem.VirtualMemoryStat, error)
		swap            func(context.Context) (*mem.SwapMemoryStat, error)
		exec            executor.Executor
		wantErr         bool
		wantSpeculative uint64
		wantCompressed  uint64
		wantWired       uint64
	}{
		{
			name:            "apple silicon vm_stat parsed: speculative + compressed populated",
			vm:              okVM,
			swap:            okSwap,
			exec:            vmStatExec(s.T(), []byte(vmStatAppleSilicon), nil),
			wantSpeculative: 8192 * 16384,
			wantCompressed:  24576 * 16384,
			wantWired:       3000,
		},
		{
			name: "non-default page size still parsed correctly",
			vm:   okVM, swap: okSwap,
			exec: vmStatExec(s.T(), []byte(
				"Mach Virtual Memory Statistics: (page size of 4096 bytes)\n"+
					"Pages speculative:                         1000.\n"+
					"Pages stored in compressor:              2000.\n",
			), nil),
			wantSpeculative: 1000 * 4096,
			wantCompressed:  2000 * 4096,
			wantWired:       3000,
		},
		{
			name: "vm_stat without page-size header: defaults to 4096",
			vm:   okVM, swap: okSwap,
			exec: vmStatExec(s.T(), []byte(
				"Pages speculative:                         1000.\n",
			), nil),
			wantSpeculative: 1000 * 4096,
			wantWired:       3000,
		},
		{
			name: "vm_stat with non-numeric value: skipped",
			vm:   okVM, swap: okSwap,
			exec: vmStatExec(s.T(), []byte(
				"Mach Virtual Memory Statistics: (page size of 4096 bytes)\n"+
					"Pages speculative:                       abc.\n",
			), nil),
			wantWired: 3000,
		},
		{
			name: "vm_stat missing: extension skipped, gopsutil totals intact",
			vm:   okVM, swap: okSwap,
			exec:      vmStatExec(s.T(), nil, errors.New("not found")),
			wantWired: 3000,
		},
		{
			name: "nil Exec: extension skipped cleanly",
			vm:   okVM, swap: okSwap,
			exec:      nil,
			wantWired: 3000,
		},
		{
			name:    "gopsutil error propagated",
			vm:      vmErr,
			swap:    okSwap,
			exec:    vmStatExec(s.T(), []byte(vmStatAppleSilicon), nil),
			wantErr: true,
		},
		{
			name: "line without colon, zero-page-size header: both skipped",
			vm:   okVM, swap: okSwap,
			exec: vmStatExec(s.T(), []byte(
				"Mach Virtual Memory Statistics: (page size of 0 bytes)\n"+
					"no colon line here\n"+
					"Pages speculative:                         500.\n",
			), nil),
			wantSpeculative: 500 * 4096,
			wantWired:       3000,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer memory.SetVirtualMemoryFn(tt.vm)()
			defer memory.SetSwapMemoryFn(tt.swap)()

			c := &memory.Darwin{Exec: tt.exec}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*memory.Info)
			s.Require().True(ok)
			s.Equal(tt.wantSpeculative, info.Speculative)
			s.Equal(tt.wantCompressed, info.Compressed)
			s.Equal(tt.wantWired, info.Wired)
		})
	}
}
