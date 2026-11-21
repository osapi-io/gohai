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

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
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

func (s *MemoryPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(memory.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c memory.Collector) {
				_, ok := c.(*memory.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c memory.Collector) {
				_, ok := c.(*memory.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c memory.Collector) {
				_, ok := c.(*memory.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c memory.Collector) {
				_, ok := c.(*memory.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c memory.Collector) {
				_, ok := c.(*memory.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := memory.New()
			s.Equal("memory", c.Name())
			s.Equal("hardware", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *MemoryPublicTestSuite) TestCollect() {
	const meminfo = `MemTotal:        1000 kB
MemFree:          200 kB
MemAvailable:     500 kB
Buffers:           10 kB
Cached:           100 kB
SwapCached:         5 kB
Active:           300 kB
Inactive:         150 kB
Active(anon):     120 kB
Inactive(anon):    30 kB
Active(file):     180 kB
Inactive(file):   120 kB
Unevictable:        1 kB
Dirty:              2 kB
Writeback:          3 kB
WritebackTmp:       4 kB
AnonPages:         50 kB
Mapped:            60 kB
Shmem:             70 kB
KReclaimable:      40 kB
Slab:              80 kB
SReclaimable:      40 kB
SUnreclaim:        40 kB
KernelStack:        6 kB
PageTables:         7 kB
NFS_Unstable:       8 kB
Bounce:             9 kB
CommitLimit:      900 kB
Committed_AS:     800 kB
VmallocTotal:     100 kB
VmallocUsed:       20 kB
VmallocChunk:      30 kB
Percpu:            11 kB
HighTotal:         12 kB
HighFree:          13 kB
LowTotal:          14 kB
LowFree:           15 kB
AnonHugePages:     16 kB
HugePages_Total:     2
HugePages_Free:      1
HugePages_Rsvd:      0
HugePages_Surp:      0
Hugepagesize:    2048 kB
Hugetlb:         4096 kB
DirectMap4k:      100 kB
DirectMap2M:      200 kB
DirectMap1G:      300 kB
SwapTotal:        400 kB
SwapFree:         100 kB
no colon here
Broken:
NotANumber:       abc kB
`
	const vmStat = `Mach Virtual Memory Statistics: (page size of 4096 bytes)
Pages free:                               10.
Pages active:                             20.
Pages inactive:                           30.
Pages speculative:                         5.
Pages wired down:                         40.
Pages stored in compressor:                6.
no colon line
Pages bogus:                         lots.
`
	const swapUsage = "total = 2048.00M  used = 512.00M  free = 1536.00M  (encrypted)\n"
	const kb = 1024

	tests := []struct {
		name    string
		variant string
		files   map[string]string
		runner  sourcetest.Commands
		check   func(*memory.Info)
		wantErr string
	}{
		{
			name:    "linux: every field of /proc/meminfo",
			variant: "linux",
			files:   map[string]string{"/proc/meminfo": meminfo},
			check: func(i *memory.Info) {
				s.Equal(uint64(1000*kb), i.Total)
				s.Equal(uint64(200*kb), i.Free)
				s.Equal(uint64(500*kb), i.Available)
				s.Equal(uint64(140*kb), i.Cached, "Cached includes SReclaimable, as in gopsutil")
				s.Equal(uint64(500*kb), i.Used, "Used is Total less Available")
				s.InDelta(50.0, i.UsedPercent, 0.001)
				s.Equal(uint64(70*kb), i.Shared)
				s.Equal(uint64(70*kb), i.Shmem)
				s.Equal(uint64(120*kb), i.ActiveAnon)
				s.Equal(uint64(800*kb), i.CommittedAS)
				s.Equal(uint64(15*kb), i.LowFree)
				s.Equal(&memory.Hugepages{
					Total: 2, Free: 1, Size: 2048 * kb, AnonHugePages: 16 * kb, Hugetlb: 4096 * kb,
				}, i.HugePages)
				s.Equal(
					&memory.DirectMap{Map4k: 100 * kb, Map2M: 200 * kb, Map1G: 300 * kb},
					i.DirectMap,
				)
				s.Equal(&memory.Swap{
					Total:       400 * kb,
					Used:        300 * kb,
					Free:        100 * kb,
					UsedPercent: 75,
					Cached:      5 * kb,
				}, i.Swap)
			},
		},
		{
			name:    "linux: an old kernel with no MemAvailable",
			variant: "linux",
			files: map[string]string{"/proc/meminfo": "MemTotal: 1000 kB\nMemFree: 200 kB\n" +
				"Cached: 100 kB\nSReclaimable: 50 kB\n"},
			check: func(i *memory.Info) {
				s.Equal(uint64(350*kb), i.Available, "Cached plus Free")
				s.Equal(uint64(650*kb), i.Used)
				s.Nil(i.HugePages)
				s.Nil(i.DirectMap)
				s.Nil(i.Swap)
			},
		},
		{
			name:    "linux: no MemTotal",
			variant: "linux",
			files:   map[string]string{"/proc/meminfo": "MemFree: 200 kB\n"},
			check: func(i *memory.Info) {
				s.Zero(i.Used)
				s.Zero(i.UsedPercent)
			},
		},
		{name: "linux: no /proc/meminfo", variant: "linux", wantErr: "read /proc/meminfo"},
		{
			name:    "darwin: hw.memsize, vm_stat and vm.swapusage",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n hw.memsize":   "409600\n",
				"vm_stat":                vmStat,
				"sysctl -n vm.swapusage": swapUsage,
			}},
			check: func(i *memory.Info) {
				s.Equal(uint64(409600), i.Total)
				s.Equal(uint64(10*4096), i.Free)
				s.Equal(uint64(20*4096), i.Active)
				s.Equal(uint64(30*4096), i.Inactive)
				s.Equal(uint64(40*4096), i.Wired)
				s.Equal(uint64(5*4096), i.Speculative)
				s.Equal(uint64(6*4096), i.Compressed)
				s.Equal(uint64(40*4096), i.Available, "free plus inactive, as in gopsutil")
				s.Equal(uint64(409600-40*4096), i.Used)
				s.Equal(&memory.Swap{
					Total: 2048 << 20, Used: 512 << 20, Free: 1536 << 20, UsedPercent: 25,
				}, i.Swap)
			},
		},
		{
			name:    "darwin: vm_stat and vm.swapusage fail",
			variant: "darwin",
			runner: sourcetest.Commands{
				Out: map[string]string{"sysctl -n hw.memsize": "409600\n"},
			},
			check: func(i *memory.Info) {
				s.Equal(&memory.Info{Total: 409600}, i)
			},
		},
		{
			name:    "darwin: more available than the total, and no swap",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n hw.memsize":   "1\n",
				"vm_stat":                vmStat,
				"sysctl -n vm.swapusage": "total = 0.00M  used = 0.00M  free = 0.00M\n",
			}},
			check: func(i *memory.Info) {
				s.Zero(i.Used)
				s.Nil(i.Swap)
			},
		},
		{
			name:    "darwin: hw.memsize fails",
			variant: "darwin",
			runner: sourcetest.Commands{
				Err: map[string]error{"sysctl -n hw.memsize": errors.New("boom")},
			},
			wantErr: "sysctl hw.memsize: boom",
		},
		{
			name:    "darwin: hw.memsize is not a number",
			variant: "darwin",
			runner:  sourcetest.Commands{Out: map[string]string{"sysctl -n hw.memsize": "lots\n"}},
			wantErr: "sysctl hw.memsize",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c memory.Collector
			switch tt.variant {
			case "linux":
				c = &memory.Linux{FS: sourcetest.FS(s.T(), tt.files)}
			case "darwin":
				c = &memory.Darwin{Exec: tt.runner.Mock(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			info, ok := got.(*memory.Info)
			s.Require().True(ok)
			tt.check(info)
		})
	}
}
