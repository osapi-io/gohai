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

package cpu_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	gpcpu "github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

type CPULinuxPublicTestSuite struct {
	suite.Suite
}

func TestCPULinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CPULinuxPublicTestSuite))
}

// vulnReadErrorFS wraps memfs and forces ReadFile to error on a
// specific path — used to exercise the ReadFile-fails branch of
// readVulnerabilities.
type vulnReadErrorFS struct {
	avfs.VFS
	failPath string
}

func (v vulnReadErrorFS) ReadFile(p string) ([]byte, error) {
	if p == v.failPath {
		return nil, errors.New("read fail")
	}
	return v.VFS.ReadFile(p)
}

// newVulnFS returns a memfs with the vulnerabilities sysfs directory
// populated from the given map.
func newVulnFS(
	s *CPULinuxPublicTestSuite,
	entries map[string]string,
) avfs.VFS {
	f := memfs.New()
	if entries == nil {
		return f
	}
	s.Require().NoError(f.MkdirAll("/sys/devices/system/cpu/vulnerabilities", 0o755))
	for name, content := range entries {
		s.Require().NoError(
			f.WriteFile(
				"/sys/devices/system/cpu/vulnerabilities/"+name,
				[]byte(content),
				fs.FileMode(0o644),
			),
		)
	}
	return f
}

// noLscpuExec returns a mock Executor that returns an error for any
// Execute call — simulates `lscpu` absent from PATH.
func noLscpuExec(
	s *CPULinuxPublicTestSuite,
) executor.Executor {
	ctrl := gomock.NewController(s.T())
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lscpu").
		Return(nil, errors.New("not found")).
		AnyTimes()
	return m
}

// lscpuExec returns a mock Executor that returns canned lscpu output.
func lscpuExec(
	s *CPULinuxPublicTestSuite,
	out string,
) executor.Executor {
	ctrl := gomock.NewController(s.T())
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lscpu").
		Return([]byte(out), nil).
		AnyTimes()
	return m
}

func (s *CPULinuxPublicTestSuite) TestCollect() {
	baseInfo := func() *cpu.Info {
		return &cpu.Info{
			Total: 16, Sockets: 1, Cores: 8,
			ModelName: "Intel(R) Xeon(R) CPU",
			VendorID:  "GenuineIntel",
			Family:    "6", Model: "85",
			Stepping: 7, Mhz: 2400, CacheSize: 25600,
			Flags: []string{"fpu", "vme"},
		}
	}

	tests := []struct {
		name     string
		readFn   func(context.Context) (*cpu.Info, error)
		fs       avfs.VFS
		exec     executor.Executor
		wantErr  bool
		validate func(*cpu.Info)
	}{
		{
			name:   "base only — no vulns dir, no lscpu",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec:   noLscpuExec(s),
			validate: func(i *cpu.Info) {
				s.Nil(i.Vulnerabilities)
				s.Nil(i.Caches)
				s.Nil(i.NumaNodes)
				s.Equal(16, i.Total)
				s.Equal(8, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:   "vulnerabilities populated",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs: newVulnFS(s, map[string]string{
				"meltdown":   "Mitigation: PTI\n",
				"spectre_v2": "Mitigation: Retpolines\n",
				"mds":        "Not affected\n",
			}),
			exec: noLscpuExec(s),
			validate: func(i *cpu.Info) {
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.Equal("Mitigation: Retpolines", i.Vulnerabilities["spectre_v2"])
				s.Equal("Not affected", i.Vulnerabilities["mds"])
			},
		},
		{
			name:   "empty vulnerabilities directory yields nil map",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, map[string]string{}),
			exec:   noLscpuExec(s),
			validate: func(i *cpu.Info) {
				s.Nil(i.Vulnerabilities)
			},
		},
		{
			name:   "x86 lscpu: caches + numa_nodes populated, counts unchanged",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
Socket(s):           1
Core(s) per socket:  8
Thread(s) per core:  2
L1d cache:           32 KiB
L1i cache:           32 KiB
L2 cache:            256 KiB
L3 cache:            25 MiB
NUMA node(s):        1
NUMA node0 CPU(s):   0-15
`),
			validate: func(i *cpu.Info) {
				s.Require().NotNil(i.Caches)
				s.Equal("32 KiB", i.Caches.L1d)
				s.Equal("32 KiB", i.Caches.L1i)
				s.Equal("256 KiB", i.Caches.L2)
				s.Equal("25 MiB", i.Caches.L3)
				s.Require().NotNil(i.NumaNodes)
				s.Equal([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, i.NumaNodes[0])
				// x86 does NOT override gopsutil's counts
				s.Equal(16, i.Total)
				s.Equal(8, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:   "s390x lscpu: counts overridden",
			readFn: func(context.Context) (*cpu.Info, error) { return &cpu.Info{Total: 1, Sockets: 1, Cores: 1}, nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        s390x
Thread(s) per core:  1
Core(s) per socket:  3
Socket(s) per book:  2
Book(s) per drawer:  2
Drawer(s):           2
`),
			validate: func(i *cpu.Info) {
				// total = 2 * 3 * 1 * 2 * 2 = 24
				// cores = 2 * 3 * 2 * 2 = 24
				// sockets = socketsPerBook = 2
				s.Equal(24, i.Total)
				s.Equal(24, i.Cores)
				s.Equal(2, i.Sockets)
			},
		},
		{
			name:   "ppc64le lscpu: counts overridden",
			readFn: func(context.Context) (*cpu.Info, error) { return &cpu.Info{Total: 1, Sockets: 1, Cores: 1}, nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        ppc64le
Thread(s) per core:  4
Core(s) per socket:  20
Socket(s):           2
`),
			validate: func(i *cpu.Info) {
				// total = 2 * 20 * 4 = 160
				// cores = 2 * 20 = 40
				// sockets = 2
				s.Equal(160, i.Total)
				s.Equal(40, i.Cores)
				s.Equal(2, i.Sockets)
			},
		},
		{
			name:   "malformed lscpu: no-op extension, base Info preserved",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec:   lscpuExec(s, "no colons here\nmore garbage\n"),
			validate: func(i *cpu.Info) {
				s.Nil(i.Caches)
				s.Nil(i.NumaNodes)
				s.Equal(16, i.Total)
			},
		},
		{
			name:   "NUMA CPU list with range+singletons",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
NUMA node0 CPU(s):   0-3,8,10-11
`),
			validate: func(i *cpu.Info) {
				s.Equal([]int{0, 1, 2, 3, 8, 10, 11}, i.NumaNodes[0])
			},
		},
		{
			name:   "NUMA CPU list malformed yields no entry for that node",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
NUMA node0 CPU(s):   bad-data,x-y
`),
			validate: func(i *cpu.Info) {
				// parseCPURange returns nil on malformed → node0 entry is nil slice;
				// numaNodes map still has the key but with a nil value
				// (applyLscpuToInfo only surfaces the map when non-empty overall)
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:   "NUMA CPU list reversed range yields nil",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
NUMA node0 CPU(s):   5-2
`),
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:   "vulnerabilities dir with subdir: subdir skipped, files still read",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs: func() avfs.VFS {
				f := newVulnFS(s, map[string]string{"meltdown": "Mitigation: PTI\n"})
				s.Require().NoError(f.MkdirAll("/sys/devices/system/cpu/vulnerabilities/unrelated_subdir", 0o755))
				return f
			}(),
			exec: noLscpuExec(s),
			validate: func(i *cpu.Info) {
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.NotContains(i.Vulnerabilities, "unrelated_subdir")
			},
		},
		{
			name:   "vulnerabilities dir with unreadable file: that file skipped",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs: vulnReadErrorFS{
				VFS:      newVulnFS(s, map[string]string{"meltdown": "Mitigation: PTI\n", "bad": "x"}),
				failPath: "/sys/devices/system/cpu/vulnerabilities/bad",
			},
			exec: noLscpuExec(s),
			validate: func(i *cpu.Info) {
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.NotContains(i.Vulnerabilities, "bad")
			},
		},
		{
			name:   "lscpu with empty-value line skipped",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
L1d cache:
L1i cache:           32 KiB
`),
			validate: func(i *cpu.Info) {
				s.Require().NotNil(i.Caches)
				s.Equal("", i.Caches.L1d)
				s.Equal("32 KiB", i.Caches.L1i)
			},
		},
		{
			name:   "NUMA CPU list with empty parts (consecutive commas) parses remaining",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
NUMA node0 CPU(s):   0,,3
`),
			validate: func(i *cpu.Info) {
				s.Equal([]int{0, 3}, i.NumaNodes[0])
			},
		},
		{
			name:   "NUMA CPU list single non-numeric yields no entry",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
NUMA node0 CPU(s):   abc
`),
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:   "NUMA CPU list all-empty yields no numa_nodes",
			readFn: func(context.Context) (*cpu.Info, error) { return baseInfo(), nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        x86_64
NUMA node0 CPU(s):   ,,,
`),
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:   "s390x lscpu with missing drawers: nonZero fallback to 1",
			readFn: func(context.Context) (*cpu.Info, error) { return &cpu.Info{Total: 1, Sockets: 1, Cores: 1}, nil },
			fs:     newVulnFS(s, nil),
			exec: lscpuExec(s, `Architecture:        s390x
Thread(s) per core:  1
Core(s) per socket:  2
Socket(s) per book:  1
Book(s) per drawer:  1
`),
			validate: func(i *cpu.Info) {
				// drawers missing (0) → nonZero returns 1
				// total = 1 * 2 * 1 * 1 * 1 = 2, cores = 1 * 2 * 1 * 1 = 2
				s.Equal(2, i.Total)
				s.Equal(2, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:    "ReadFn error propagated",
			readFn:  func(context.Context) (*cpu.Info, error) { return nil, errors.New("cpuinfo error") },
			fs:      newVulnFS(s, nil),
			exec:    noLscpuExec(s),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &cpu.Linux{ReadFn: tt.readFn, FS: tt.fs, Exec: tt.exec}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*cpu.Info)
			s.Require().True(ok)
			if tt.validate != nil {
				tt.validate(info)
			}
		})
	}
}

func (s *CPULinuxPublicTestSuite) TestReadCPU() {
	okStats := []gpcpu.InfoStat{{
		PhysicalID: "0", Cores: 8, ModelName: "Intel(R) Xeon(R) CPU",
		VendorID: "GenuineIntel", Family: "6", Model: "85",
		Stepping: 7, Mhz: 2400, CacheSize: 25600,
		Flags: []string{"fpu", "vme"},
	}}

	tests := []struct {
		name        string
		infoFn      func(context.Context) ([]gpcpu.InfoStat, error)
		countsFn    func(context.Context, bool) (int, error)
		wantErr     bool
		wantTotal   int
		wantSockets int
		wantCores   int
	}{
		{
			name:        "success maps stats",
			infoFn:      func(context.Context) ([]gpcpu.InfoStat, error) { return okStats, nil },
			countsFn:    func(context.Context, bool) (int, error) { return 16, nil },
			wantTotal:   16,
			wantSockets: 1,
			wantCores:   8,
		},
		{
			name:        "counts error ignored, info still populated",
			infoFn:      func(context.Context) ([]gpcpu.InfoStat, error) { return okStats, nil },
			countsFn:    func(context.Context, bool) (int, error) { return 0, errors.New("counts failed") },
			wantTotal:   0,
			wantSockets: 1,
			wantCores:   8,
		},
		{
			name:        "empty info stats yields zero-value Info",
			infoFn:      func(context.Context) ([]gpcpu.InfoStat, error) { return nil, nil },
			countsFn:    func(context.Context, bool) (int, error) { return 4, nil },
			wantTotal:   4,
			wantSockets: 0,
			wantCores:   0,
		},
		{
			name:     "info error propagated",
			infoFn:   func(context.Context) ([]gpcpu.InfoStat, error) { return nil, errors.New("info failed") },
			countsFn: func(context.Context, bool) (int, error) { return 0, nil },
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restoreInfo := cpu.SetInfoFn(tt.infoFn)
			defer restoreInfo()
			restoreCounts := cpu.SetCountsFn(tt.countsFn)
			defer restoreCounts()
			got, err := cpu.ReadCPU(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.wantTotal, got.Total)
			s.Equal(tt.wantSockets, got.Sockets)
			s.Equal(tt.wantCores, got.Cores)
		})
	}
}
