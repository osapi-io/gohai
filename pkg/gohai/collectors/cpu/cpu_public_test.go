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

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
)

var (
	_ collector.Collector = (*cpu.Linux)(nil)
	_ collector.Collector = (*cpu.Darwin)(nil)
)

type CPUPublicTestSuite struct {
	suite.Suite
}

func TestCPUPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(CPUPublicTestSuite))
}

// vulnReadErrorFS wraps memfs and forces ReadFile to error on a
// specific path.
type vulnReadErrorFS struct {
	avfs.VFS
	failPath string
}

func (v vulnReadErrorFS) ReadFile(
	p string,
) ([]byte, error) {
	if p == v.failPath {
		return nil, errors.New("read fail")
	}
	return v.VFS.ReadFile(p)
}

func newVulnFS(
	s *CPUPublicTestSuite,
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

func noLscpuExec(
	t *testing.T,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lscpu").
		Return(nil, errors.New("not found")).
		AnyTimes()
	return m
}

func lscpuExec(
	t *testing.T,
	out string,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lscpu").
		Return([]byte(out), nil).
		AnyTimes()
	return m
}

func (s *CPUPublicTestSuite) TestNew() {
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
			c := cpu.New()
			s.Equal("cpu", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*cpu.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*cpu.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *CPUPublicTestSuite) TestCollect() {
	// linuxBaseStats mirrors the old baseInfo() result: Xeon w/ 16 logical / 8 cores / 1 socket.
	linuxBaseStats := []gpcpu.InfoStat{{
		PhysicalID: "0", Cores: 8,
		ModelName: "Intel(R) Xeon(R) CPU",
		VendorID:  "GenuineIntel",
		Family:    "6", Model: "85",
		Stepping: 7, Mhz: 2400, CacheSize: 25600,
		Flags: []string{"fpu", "vme"},
	}}
	linuxBaseInfo := func(context.Context) ([]gpcpu.InfoStat, error) { return linuxBaseStats, nil }
	linuxBaseCounts := func(context.Context, bool) (int, error) { return 16, nil }

	// darwinBaseStats mirrors the old darwin baseInfo(): 12 logical / 12 cores / 1 socket / 0 Mhz.
	darwinBaseStats := []gpcpu.InfoStat{{
		PhysicalID: "0", Cores: 12,
		ModelName: "Apple M2 Pro",
	}}
	darwinBaseInfo := func(context.Context) ([]gpcpu.InfoStat, error) { return darwinBaseStats, nil }
	darwinBaseCounts := func(context.Context, bool) (int, error) { return 12, nil }

	// onecoreStats: 1 logical / 1 core / 1 socket — used by s390x/ppc64le tests whose counts
	// are fully overridden by lscpu.
	onecoreStats := []gpcpu.InfoStat{{PhysicalID: "0", Cores: 1}}
	onecoreInfo := func(context.Context) ([]gpcpu.InfoStat, error) { return onecoreStats, nil }
	onecoreCounts := func(context.Context, bool) (int, error) { return 1, nil }

	type sysctlRet struct {
		out string
		err error
	}
	darwinExec := func(t *testing.T, sysctls map[string]sysctlRet) executor.Executor {
		ctrl := gomock.NewController(t)
		m := execmocks.NewMockExecutor(ctrl)
		for key, ret := range sysctls {
			call := m.EXPECT().Execute(gomock.Any(), "sysctl", "-n", key)
			if ret.err != nil {
				call.Return(nil, ret.err).AnyTimes()
			} else {
				call.Return([]byte(ret.out), nil).AnyTimes()
			}
		}
		return m
	}

	tests := []struct {
		name     string
		variant  string
		infoFn   func(context.Context) ([]gpcpu.InfoStat, error)
		countsFn func(context.Context, bool) (int, error)
		fs       avfs.VFS
		exec     func(*testing.T) executor.Executor
		wantErr  bool
		validate func(*cpu.Info)
	}{
		{
			name:     "linux: base only, no vulns dir no lscpu",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec:     noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Nil(i.Vulnerabilities)
				s.Nil(i.Caches)
				s.Nil(i.NumaNodes)
				s.Equal(16, i.Count)
				s.Equal(8, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:     "linux: vulnerabilities populated",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs: newVulnFS(s, map[string]string{
				"meltdown":   "Mitigation: PTI\n",
				"spectre_v2": "Mitigation: Retpolines\n",
				"mds":        "Not affected\n",
			}),
			exec: noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.Equal("Mitigation: Retpolines", i.Vulnerabilities["spectre_v2"])
				s.Equal("Not affected", i.Vulnerabilities["mds"])
			},
		},
		{
			name:     "linux: empty vulnerabilities directory yields nil map",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, map[string]string{}),
			exec:     noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Nil(i.Vulnerabilities)
			},
		},
		{
			name:     "linux: x86 lscpu populates caches + numa_nodes, counts unchanged",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
Socket(s):           1
Core(s) per socket:  8
Thread(s) per core:  2
L1d cache:           32 KiB
L1i cache:           32 KiB
L2 cache:            256 KiB
L3 cache:            25 MiB
NUMA node(s):        1
NUMA node0 CPU(s):   0-15
`)
			},
			validate: func(i *cpu.Info) {
				s.Require().NotNil(i.Caches)
				s.Equal("32 KiB", i.Caches.L1d)
				s.Equal("32 KiB", i.Caches.L1i)
				s.Equal("256 KiB", i.Caches.L2)
				s.Equal("25 MiB", i.Caches.L3)
				s.Require().NotNil(i.NumaNodes)
				s.Equal([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, i.NumaNodes[0])
				s.Equal(16, i.Count)
				s.Equal(8, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:     "linux: s390x lscpu overrides counts",
			variant:  "linux",
			infoFn:   onecoreInfo,
			countsFn: onecoreCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        s390x
Thread(s) per core:  1
Core(s) per socket:  3
Socket(s) per book:  2
Book(s) per drawer:  2
Drawer(s):           2
`)
			},
			validate: func(i *cpu.Info) {
				s.Equal(24, i.Count)
				s.Equal(24, i.Cores)
				s.Equal(2, i.Sockets)
			},
		},
		{
			name:     "linux: ppc64le lscpu overrides counts",
			variant:  "linux",
			infoFn:   onecoreInfo,
			countsFn: onecoreCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        ppc64le
Thread(s) per core:  4
Core(s) per socket:  20
Socket(s):           2
`)
			},
			validate: func(i *cpu.Info) {
				s.Equal(160, i.Count)
				s.Equal(40, i.Cores)
				s.Equal(2, i.Sockets)
			},
		},
		{
			name:     "linux: malformed lscpu no-ops, base Info preserved",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, "no colons here\nmore garbage\n")
			},
			validate: func(i *cpu.Info) {
				s.Nil(i.Caches)
				s.Nil(i.NumaNodes)
				s.Equal(16, i.Count)
			},
		},
		{
			name:     "linux: NUMA CPU list with range+singletons",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   0-3,8,10-11
`)
			},
			validate: func(i *cpu.Info) {
				s.Equal([]int{0, 1, 2, 3, 8, 10, 11}, i.NumaNodes[0])
			},
		},
		{
			name:     "linux: NUMA CPU list malformed yields nil entry",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   bad-data,x-y
`)
			},
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:     "linux: NUMA CPU list reversed range yields nil",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   5-2
`)
			},
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:     "linux: vulnerabilities dir with subdir, subdir skipped files still read",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs: func() avfs.VFS {
				f := newVulnFS(s, map[string]string{"meltdown": "Mitigation: PTI\n"})
				s.Require().NoError(
					f.MkdirAll("/sys/devices/system/cpu/vulnerabilities/unrelated_subdir", 0o755),
				)
				return f
			}(),
			exec: noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.NotContains(i.Vulnerabilities, "unrelated_subdir")
			},
		},
		{
			name:     "linux: vulnerabilities dir with unreadable file skipped",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs: vulnReadErrorFS{
				VFS: newVulnFS(s, map[string]string{
					"meltdown": "Mitigation: PTI\n", "bad": "x",
				}),
				failPath: "/sys/devices/system/cpu/vulnerabilities/bad",
			},
			exec: noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.NotContains(i.Vulnerabilities, "bad")
			},
		},
		{
			name:     "linux: lscpu with empty-value line skipped",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
L1d cache:
L1i cache:           32 KiB
`)
			},
			validate: func(i *cpu.Info) {
				s.Require().NotNil(i.Caches)
				s.Equal("", i.Caches.L1d)
				s.Equal("32 KiB", i.Caches.L1i)
			},
		},
		{
			name:     "linux: NUMA CPU list with empty parts parses remaining",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   0,,3
`)
			},
			validate: func(i *cpu.Info) {
				s.Equal([]int{0, 3}, i.NumaNodes[0])
			},
		},
		{
			name:     "linux: NUMA CPU list single non-numeric yields no entry",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   abc
`)
			},
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:     "linux: NUMA CPU list all-empty yields nil entry",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   ,,,
`)
			},
			validate: func(i *cpu.Info) {
				if i.NumaNodes != nil {
					s.Nil(i.NumaNodes[0])
				}
			},
		},
		{
			name:     "linux: s390x lscpu with missing drawers falls back to 1",
			variant:  "linux",
			infoFn:   onecoreInfo,
			countsFn: onecoreCounts,
			fs:       newVulnFS(s, nil),
			exec: func(t *testing.T) executor.Executor {
				return lscpuExec(t, `Architecture:        s390x
Thread(s) per core:  1
Core(s) per socket:  2
Socket(s) per book:  1
Book(s) per drawer:  1
`)
			},
			validate: func(i *cpu.Info) {
				s.Equal(2, i.Count)
				s.Equal(2, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:     "linux: gopsutil info error propagated",
			variant:  "linux",
			infoFn:   func(context.Context) ([]gpcpu.InfoStat, error) { return nil, errors.New("cpuinfo error") },
			countsFn: linuxBaseCounts,
			fs:       newVulnFS(s, nil),
			exec:     noLscpuExec,
			wantErr:  true,
		},
		{
			name:     "linux: gopsutil counts error ignored (Count zero), info still populated",
			variant:  "linux",
			infoFn:   linuxBaseInfo,
			countsFn: func(context.Context, bool) (int, error) { return 0, errors.New("counts failed") },
			fs:       newVulnFS(s, nil),
			exec:     noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Equal(0, i.Count)
				s.Equal(1, i.Sockets)
				s.Equal(8, i.Cores)
			},
		},
		{
			name:     "linux: empty gopsutil info stats leaves zero-value derived fields",
			variant:  "linux",
			infoFn:   func(context.Context) ([]gpcpu.InfoStat, error) { return nil, nil },
			countsFn: func(context.Context, bool) (int, error) { return 4, nil },
			fs:       newVulnFS(s, nil),
			exec:     noLscpuExec,
			validate: func(i *cpu.Info) {
				s.Equal(4, i.Count)
				s.Equal(0, i.Sockets)
				s.Equal(0, i.Cores)
				s.Empty(i.ModelName)
			},
		},
		{
			name:     "darwin: Intel Mac with hyperthreading, all sysctls override",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "6\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {out: "2600000000\n"},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(12, i.Count)
				s.Equal(6, i.Cores)
				s.Equal(1, i.Sockets)
				s.Equal(2600.0, i.Mhz)
			},
		},
		{
			name:     "darwin: Apple Silicon, cpufrequency_max and cpufrequency both absent",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "10\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(10, i.Cores)
				s.Equal(1, i.Sockets)
				s.Equal(0.0, i.Mhz)
			},
		},
		{
			name:     "darwin: Apple Silicon fallback, cpufrequency_max absent but cpufrequency present",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "8\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {out: "3200000000\n"},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(3200.0, i.Mhz)
			},
		},
		{
			name:     "darwin: hw.physicalcpu fails, Cores unchanged",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {err: errors.New("unknown oid")},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(12, i.Cores)
			},
		},
		{
			name:     "darwin: hw.packages fails, Sockets unchanged",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "6\n"},
					"hw.packages":         {err: errors.New("unknown oid")},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:     "darwin: hw.physicalcpu non-numeric, Cores unchanged",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "xyz\n"},
					"hw.packages":         {err: errors.New("unknown oid")},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(12, i.Cores)
			},
		},
		{
			name:     "darwin: hw.cpufrequency_max non-numeric falls through to hw.cpufrequency",
			variant:  "darwin",
			infoFn:   darwinBaseInfo,
			countsFn: darwinBaseCounts,
			exec: func(t *testing.T) executor.Executor {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "6\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {out: "not-a-number\n"},
					"hw.cpufrequency":     {out: "2800000000\n"},
				})
			},
			validate: func(i *cpu.Info) {
				s.Equal(2800.0, i.Mhz)
			},
		},
		{
			name:     "darwin: gopsutil error propagated",
			variant:  "darwin",
			infoFn:   func(context.Context) ([]gpcpu.InfoStat, error) { return nil, errors.New("sysctl error") },
			countsFn: darwinBaseCounts,
			exec:     func(t *testing.T) executor.Executor { return darwinExec(t, map[string]sysctlRet{}) },
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer cpu.SetInfoFn(tt.infoFn)()
			defer cpu.SetCountsFn(tt.countsFn)()
			var c cpu.Collector
			switch tt.variant {
			case "linux":
				c = &cpu.Linux{FS: tt.fs, Exec: tt.exec(s.T())}
			case "darwin":
				c = &cpu.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
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
