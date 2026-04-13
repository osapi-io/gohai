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

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/memory"
)

type MemoryLinuxPublicTestSuite struct {
	suite.Suite
}

func TestMemoryLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryLinuxPublicTestSuite))
}

// vmStatExec returns a MockExecutor that canned-answers `vm_stat`.
// Shared with darwin tests because both run the same command (linux
// file hosts the helper per project convention).
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
	s *MemoryLinuxPublicTestSuite,
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

// canonical /proc/meminfo excerpt covering exactly the keys our
// extension parses. Values chosen for easy verification (1024 kB →
// 1048576 bytes).
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

// fullGopsutilVM is the "canonical Linux host" gopsutil response —
// populates the cross-platform fields TestCollect asserts.
func fullGopsutilVM(context.Context) (*mem.VirtualMemoryStat, error) {
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

func swapOK(context.Context) (*mem.SwapMemoryStat, error) {
	return &mem.SwapMemoryStat{Total: 2147483648, Used: 1024, Free: 2147482624, UsedPercent: 0.0001}, nil
}

func zeroVM(context.Context) (*mem.VirtualMemoryStat, error) {
	return &mem.VirtualMemoryStat{Total: 100}, nil
}

func zeroSwap(context.Context) (*mem.SwapMemoryStat, error) {
	return &mem.SwapMemoryStat{Total: 0}, nil
}

func (s *MemoryLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		vmFn     func(context.Context) (*mem.VirtualMemoryStat, error)
		swapFn   func(context.Context) (*mem.SwapMemoryStat, error)
		fs       avfs.VFS
		wantErr  bool
		validate func(*memory.Info)
	}{
		{
			name:   "canonical host: gopsutil + extension fields all populated",
			vmFn:   fullGopsutilVM,
			swapFn: swapOK,
			fs:     meminfoFS(s, procMeminfoSample),
			validate: func(i *memory.Info) {
				// gopsutil-sourced.
				s.Equal(uint64(17179869184), i.Total)
				s.Equal(uint64(6000), i.Active)
				s.Equal(uint64(456), i.Slab)
				s.Equal(uint64(15000000000), i.CommittedAS)
				s.Require().NotNil(i.HugePages)
				s.Equal(uint64(100), i.HugePages.Total)
				s.Equal(uint64(2097152), i.HugePages.Size)
				s.Require().NotNil(i.Swap)
				s.Equal(uint64(512), i.Swap.Cached)
				// Extension-sourced (kB × 1024 = bytes).
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
			name:   "missing /proc/meminfo leaves extension fields zero",
			vmFn:   zeroVM,
			swapFn: zeroSwap,
			fs:     meminfoFS(s, ""),
			validate: func(i *memory.Info) {
				s.Zero(i.ActiveAnon)
				s.Nil(i.DirectMap)
			},
		},
		{
			name:   "Hugetlb without gopsutil hugepages still allocates struct",
			vmFn:   zeroVM,
			swapFn: zeroSwap,
			fs:     meminfoFS(s, "Hugetlb: 2048 kB\n"),
			validate: func(i *memory.Info) {
				s.Require().NotNil(i.HugePages)
				s.Equal(uint64(2048*1024), i.HugePages.Hugetlb)
			},
		},
		{
			name:   "malformed lines skipped; valid lines still parsed",
			vmFn:   zeroVM,
			swapFn: zeroSwap,
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
			name:   "value without kB unit treated as bytes",
			vmFn:   zeroVM,
			swapFn: zeroSwap,
			fs:     meminfoFS(s, "KernelStack: 4096\n"),
			validate: func(i *memory.Info) {
				s.Equal(uint64(4096), i.KernelStack)
			},
		},
		{
			name:   "nil FS: extension skipped cleanly",
			vmFn:   zeroVM,
			swapFn: zeroSwap,
			fs:     nil,
			validate: func(i *memory.Info) {
				s.Zero(i.ActiveAnon)
			},
		},
		{
			name: "gopsutil error propagated",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return nil, errors.New("vm failed")
			},
			swapFn:  zeroSwap,
			fs:      meminfoFS(s, ""),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer memory.SetVirtualMemoryFn(tt.vmFn)()
			defer memory.SetSwapMemoryFn(tt.swapFn)()

			c := &memory.Linux{FS: tt.fs}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*memory.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}

func (s *MemoryLinuxPublicTestSuite) TestReadMemory() {
	tests := []struct {
		name      string
		vmFn      func(context.Context) (*mem.VirtualMemoryStat, error)
		swapFn    func(context.Context) (*mem.SwapMemoryStat, error)
		wantErr   bool
		wantTotal uint64
		wantSwap  bool
		wantHuge  bool
	}{
		{
			name: "virtual + swap populated, no hugepages",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{Total: 100, Used: 50, Free: 50}, nil
			},
			swapFn: func(context.Context) (*mem.SwapMemoryStat, error) {
				return &mem.SwapMemoryStat{Total: 10, Used: 1, Free: 9}, nil
			},
			wantTotal: 100,
			wantSwap:  true,
		},
		{
			name: "hugepages surfaced when any hugepage field present",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{Total: 100, HugePagesTotal: 5}, nil
			},
			swapFn:    zeroSwap,
			wantTotal: 100,
			wantHuge:  true,
		},
		{
			name: "swap error tolerated silently",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return &mem.VirtualMemoryStat{Total: 100}, nil
			},
			swapFn: func(context.Context) (*mem.SwapMemoryStat, error) {
				return nil, errors.New("swap failed")
			},
			wantTotal: 100,
		},
		{
			name: "virtual memory error propagated",
			vmFn: func(context.Context) (*mem.VirtualMemoryStat, error) {
				return nil, errors.New("vm failed")
			},
			swapFn:  func(context.Context) (*mem.SwapMemoryStat, error) { return nil, nil },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer memory.SetVirtualMemoryFn(tt.vmFn)()
			defer memory.SetSwapMemoryFn(tt.swapFn)()
			got, err := memory.ReadMemory(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantTotal, got.Total)
			if tt.wantSwap {
				s.NotNil(got.Swap)
			} else {
				s.Nil(got.Swap)
			}
			if tt.wantHuge {
				s.NotNil(got.HugePages)
			} else {
				s.Nil(got.HugePages)
			}
		})
	}
}
