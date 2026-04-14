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

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
)

var (
	_ collector.Collector = (*memory.Linux)(nil)
	_ collector.Collector = (*memory.Darwin)(nil)
)

type MemoryPublicTestSuite struct {
	suite.Suite
}

func TestMemoryPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(MemoryPublicTestSuite))
}

// vmStatExec returns a MockExecutor that canned-answers `vm_stat`.
func vmStatExec(
	t *testing.T,
	out []byte, err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "vm_stat").
		Return(out, err).
		AnyTimes()
	return m
}

// meminfoFS returns a memfs containing /proc/meminfo with the given
// content. Empty content means the file is absent.
func meminfoFS(
	s *MemoryPublicTestSuite,
	content string,
) avfs.VFS {
	fs := memfs.New()
	if content == "" {
		return fs
	}
	s.Require().NoError(fs.MkdirAll("/proc", 0o755))
	s.Require().NoError(fs.WriteFile("/proc/meminfo", []byte(content), 0o644))
	return fs
}

const procMeminfoSample = `MemTotal:       16777216 kB
Active(anon):       1000 kB
Inactive(anon):      500 kB
Active(file):       2000 kB
Inactive(file):     1500 kB
Unevictable:         100 kB
KernelStack:         200 kB
Percpu:               50 kB
KReclaimable:        400 kB
AnonPages:          3000 kB
Shmem:              1234 kB
Hugetlb:          262144 kB
DirectMap4k:       10000 kB
DirectMap2M:      500000 kB
DirectMap1G:     1048576 kB
`

const vmStatAppleSilicon = `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               42000.
Pages active:                             81920.
Pages inactive:                           65536.
Pages speculative:                         8192.
Pages wired down:                         32768.
Pages stored in compressor:              24576.
`

func (s *MemoryPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"rhel dispatches to Linux", "rhel", "linux"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := memory.New()
			s.Equal("memory", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*memory.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*memory.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *MemoryPublicTestSuite) TestCollect() {
	fullGopsutilVM := func(context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{
			Total:          17179869184,
			Available:      8589934592,
			Used:           8589934592,
			UsedPercent:    50.0,
			Free:           4294967296,
			Active:         6000,
			Inactive:       2000,
			Buffers:        1048576,
			Cached:         4194304,
			Dirty:          123,
			Slab:           456,
			Sreclaimable:   300,
			Sunreclaim:     156,
			PageTables:     789,
			CommitLimit:    20000000000,
			CommittedAS:    15000000000,
			VmallocTotal:   1024,
			HugePagesTotal: 100,
			HugePagesFree:  80,
			HugePageSize:   2097152,
			SwapCached:     512,
		}, nil
	}
	swapOK := func(context.Context) (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{
			Total:       2147483648,
			Used:        1024,
			Free:        2147482624,
			UsedPercent: 0.0001,
		}, nil
	}
	zeroVM := func(context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Total: 100}, nil
	}
	zeroSwap := func(context.Context) (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{Total: 0}, nil
	}
	darwinVM := func(context.Context) (*mem.VirtualMemoryStat, error) {
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
	hugepagesVM := func(context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Total: 100, HugePagesTotal: 5}, nil
	}

	tests := []struct {
		name     string
		variant  string
		vmFn     func(context.Context) (*mem.VirtualMemoryStat, error)
		swapFn   func(context.Context) (*mem.SwapMemoryStat, error)
		fs       avfs.VFS
		exec     func(*testing.T) executor.Executor
		wantErr  bool
		validate func(*memory.Info)
	}{
		{
			name:    "linux: canonical host, gopsutil + extension fields populated",
			variant: "linux",
			vmFn:    fullGopsutilVM,
			swapFn:  swapOK,
			fs:      meminfoFS(s, procMeminfoSample),
			validate: func(i *memory.Info) {
				s.Equal(uint64(17179869184), i.Total)
				s.Equal(uint64(6000), i.Active)
				s.Equal(uint64(456), i.Slab)
				s.Equal(uint64(15000000000), i.CommittedAS)
				s.Require().NotNil(i.HugePages)
				s.Equal(uint64(100), i.HugePages.Total)
				s.Equal(uint64(2097152), i.HugePages.Size)
				s.Require().NotNil(i.Swap)
				s.Equal(uint64(512), i.Swap.Cached)
				s.Equal(uint64(1000*1024), i.ActiveAnon)
				s.Equal(uint64(500*1024), i.InactiveAnon)
				s.Equal(uint64(2000*1024), i.ActiveFile)
				s.Equal(uint64(1500*1024), i.InactiveFile)
				s.Equal(uint64(100*1024), i.Unevictable)
				s.Equal(uint64(200*1024), i.KernelStack)
				s.Equal(uint64(50*1024), i.PerCPU)
				s.Equal(uint64(400*1024), i.KReclaimable)
				s.Equal(uint64(3000*1024), i.AnonPages)
				s.Equal(uint64(1234*1024), i.Shmem)
				s.Equal(uint64(262144*1024), i.HugePages.Hugetlb)
				s.Require().NotNil(i.DirectMap)
				s.Equal(uint64(10000*1024), i.DirectMap.Map4k)
				s.Equal(uint64(500000*1024), i.DirectMap.Map2M)
				s.Equal(uint64(1048576*1024), i.DirectMap.Map1G)
			},
		},
		{
			name:    "linux: missing /proc/meminfo leaves extension fields zero",
			variant: "linux",
			vmFn:    zeroVM,
			swapFn:  zeroSwap,
			fs:      meminfoFS(s, ""),
			validate: func(i *memory.Info) {
				s.Zero(i.ActiveAnon)
				s.Nil(i.DirectMap)
			},
		},
		{
			name:    "linux: Hugetlb without gopsutil hugepages still allocates struct",
			variant: "linux",
			vmFn:    zeroVM,
			swapFn:  zeroSwap,
			fs:      meminfoFS(s, "Hugetlb: 2048 kB\n"),
			validate: func(i *memory.Info) {
				s.Require().NotNil(i.HugePages)
				s.Equal(uint64(2048*1024), i.HugePages.Hugetlb)
			},
		},
		{
			name:    "linux: hugepages surfaced when gopsutil reports any hugepage field",
			variant: "linux",
			vmFn:    hugepagesVM,
			swapFn:  zeroSwap,
			fs:      meminfoFS(s, ""),
			validate: func(i *memory.Info) {
				s.Require().NotNil(i.HugePages)
			},
		},
		{
			name:    "linux: malformed lines skipped, valid lines still parsed",
			variant: "linux",
			vmFn:    zeroVM,
			swapFn:  zeroSwap,
			fs: meminfoFS(s, "no colon here\n"+
				"AnonPages: notanumber kB\n"+
				"Shmem:\n"+
				"KReclaimable:      42 kB\n"),
			validate: func(i *memory.Info) {
				s.Zero(i.AnonPages)
				s.Zero(i.Shmem)
				s.Equal(uint64(42*1024), i.KReclaimable)
			},
		},
		{
			name:    "linux: value without kB unit treated as bytes",
			variant: "linux",
			vmFn:    zeroVM,
			swapFn:  zeroSwap,
			fs:      meminfoFS(s, "KernelStack: 4096\n"),
			validate: func(i *memory.Info) {
				s.Equal(uint64(4096), i.KernelStack)
			},
		},
		{
			name:    "linux: nil FS, extension skipped cleanly",
			variant: "linux",
			vmFn:    zeroVM,
			swapFn:  zeroSwap,
			fs:      nil,
			validate: func(i *memory.Info) {
				s.Zero(i.ActiveAnon)
			},
		},
		{
			name:    "linux: swap error tolerated silently (Swap stays nil)",
			variant: "linux",
			vmFn:    zeroVM,
			swapFn: func(context.Context) (*mem.SwapMemoryStat, error) {
				return nil, errors.New("swap failed")
			},
			fs: meminfoFS(s, ""),
			validate: func(i *memory.Info) {
				s.Nil(i.Swap)
			},
		},
		{
			name:    "linux: gopsutil error propagated",
			variant: "linux",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return nil, errors.New("vm failed")
			},
			swapFn:  zeroSwap,
			fs:      meminfoFS(s, ""),
			wantErr: true,
		},
		{
			name:    "darwin: apple silicon vm_stat parsed, speculative + compressed populated",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, []byte(vmStatAppleSilicon), nil)
			},
			validate: func(i *memory.Info) {
				s.Equal(uint64(8192*16384), i.Speculative)
				s.Equal(uint64(24576*16384), i.Compressed)
				s.Equal(uint64(3000), i.Wired)
			},
		},
		{
			name:    "darwin: non-default page size still parsed correctly",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, []byte(
					"Mach Virtual Memory Statistics: (page size of 4096 bytes)\n"+
						"Pages speculative:                         1000.\n"+
						"Pages stored in compressor:              2000.\n",
				), nil)
			},
			validate: func(i *memory.Info) {
				s.Equal(uint64(1000*4096), i.Speculative)
				s.Equal(uint64(2000*4096), i.Compressed)
				s.Equal(uint64(3000), i.Wired)
			},
		},
		{
			name:    "darwin: vm_stat without page-size header defaults to 4096",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, []byte("Pages speculative: 1000.\n"), nil)
			},
			validate: func(i *memory.Info) {
				s.Equal(uint64(1000*4096), i.Speculative)
				s.Equal(uint64(3000), i.Wired)
			},
		},
		{
			name:    "darwin: vm_stat with non-numeric value skipped",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, []byte(
					"Mach Virtual Memory Statistics: (page size of 4096 bytes)\n"+
						"Pages speculative:                       abc.\n",
				), nil)
			},
			validate: func(i *memory.Info) {
				s.Equal(uint64(3000), i.Wired)
			},
		},
		{
			name:    "darwin: vm_stat missing, extension skipped, gopsutil totals intact",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, nil, errors.New("not found"))
			},
			validate: func(i *memory.Info) {
				s.Equal(uint64(3000), i.Wired)
			},
		},
		{
			name:    "darwin: nil Exec, extension skipped cleanly",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec:    func(*testing.T) executor.Executor { return nil },
			validate: func(i *memory.Info) {
				s.Equal(uint64(3000), i.Wired)
			},
		},
		{
			name:    "darwin: gopsutil error propagated",
			variant: "darwin",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return nil, errors.New("mach vm error")
			},
			swapFn: zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, []byte(vmStatAppleSilicon), nil)
			},
			wantErr: true,
		},
		{
			name:    "darwin: line without colon and zero-page-size header both skipped",
			variant: "darwin",
			vmFn:    darwinVM,
			swapFn:  zeroSwap,
			exec: func(t *testing.T) executor.Executor {
				return vmStatExec(t, []byte(
					"Mach Virtual Memory Statistics: (page size of 0 bytes)\n"+
						"no colon line here\n"+
						"Pages speculative:                         500.\n",
				), nil)
			},
			validate: func(i *memory.Info) {
				s.Equal(uint64(500*4096), i.Speculative)
				s.Equal(uint64(3000), i.Wired)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer memory.SetVirtualMemoryFn(tt.vmFn)()
			defer memory.SetSwapMemoryFn(tt.swapFn)()
			var c memory.Collector
			switch tt.variant {
			case "linux":
				c = &memory.Linux{FS: tt.fs}
			case "darwin":
				c = &memory.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*memory.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}
