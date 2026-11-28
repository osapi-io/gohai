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
	"fmt"
	"io/fs"
	"os/exec"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/cpu"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*cpu.Linux)(nil)
	_ collector.Collector = (*cpu.Darwin)(nil)
)

type CPUPublicTestSuite struct {
	suite.Suite
}

func TestCPUPublicTestSuite(t *testing.T) {
	suite.Run(t, new(CPUPublicTestSuite))
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
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "lscpu").
		Return(nil, fmt.Errorf("lscpu: %w", exec.ErrNotFound)).
		AnyTimes()
	return m
}

func lscpuExec(
	t *testing.T,
	out string,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
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
		name         string
		detect       string
		validateFunc func(cpu.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c cpu.Collector) {
				_, ok := c.(*cpu.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c cpu.Collector) {
				_, ok := c.(*cpu.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c cpu.Collector) {
				_, ok := c.(*cpu.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c cpu.Collector) {
				_, ok := c.(*cpu.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c cpu.Collector) {
				_, ok := c.(*cpu.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := cpu.New()
			s.Equal("cpu", c.Name())
			s.Equal("hardware", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

// TestCollect covers each variant: x86 and ARM cpuinfo, the sysfs speed
// and core id, lscpu's counts and fields, and the macOS sysctls, flags
// and frequency table.
func (s *CPUPublicTestSuite) TestCollect() {
	const pmgr = "ioreg -l -w0 -c AppleARMIODevice -n pmgr"
	// Two states: 600 MHz and 3228 MHz, each a frequency and a voltage.
	table := "\"voltage-states5-sram\" = <0046c32300030000005f67c000030000>"
	// block is one processor block of /proc/cpuinfo.
	block := func(n int, fields string) string {
		return fmt.Sprintf("processor\t: %d\n%s\n", n, fields)
	}
	xeon := "vendor_id\t: GenuineIntel\ncpu family\t: 6\nmodel\t\t: 85\n" +
		"model name\t: Intel(R) Xeon(R) CPU\nstepping\t: 7\ncpu MHz\t\t: 2400.000\n" +
		"cache size\t: 25600 KB\nphysical id\t: 0\nflags\t\t: fpu vme\n"
	// linuxCPUInfo is a Xeon with 16 logical CPUs on one socket: 8 cores,
	// each with two threads, as a Xeon with hyperthreading gives them.
	var linuxCPUInfo string
	for n := range 16 {
		linuxCPUInfo += block(n, xeon+fmt.Sprintf("core id\t\t: %d\n", n/2))
	}
	// oneCPUInfo is one logical CPU, for the s390x and ppc64le rows whose
	// counts lscpu overrides.
	oneCPUInfo := block(0, "physical id\t: 0\n")

	type sysctlRet struct {
		out string
		err error
	}
	darwinExec := func(_ *testing.T, sysctls map[string]sysctlRet) source.Runner {
		r := sourcetest.Commands{Out: map[string]string{}, Err: map[string]error{}}
		defaults := map[string]sysctlRet{
			"hw.logicalcpu":            {out: "12\n"},
			"machdep.cpu.brand_string": {out: "Apple M2 Pro\n"},
		}
		for key, ret := range defaults {
			if _, ok := sysctls[key]; !ok {
				sysctls[key] = ret
			}
		}
		for key, ret := range sysctls {
			line := "sysctl -n " + key
			if ret.err != nil {
				r.Err[line] = ret.err
			} else {
				r.Out[line] = ret.out
			}
		}
		return r.Mock(s.T())
	}

	tests := []struct {
		name         string
		variant      string
		cpuinfo      string // /proc/cpuinfo; "" is no file
		stat         string // /proc/stat
		fs           avfs.VFS
		runner       sourcetest.Commands // the commands, when exec is nil
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: ARM cpuinfo, with sysfs core ids, a speed in kHz and one in Hz, and an implementer with no name",
			variant: "linux",
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/cpuinfo": "Processor\t: AArch64\nno colon\nprocessor\t: x\n" +
					"processor\t: 0\nCPU implementer\t: 0x41\nCPU part\t: 0xd0c\nCPU revision\t: 1\n" +
					"Features\t: fp asimd,evtstrm\n" +
					"processor\t: 1\nCPU implementer\t: 0xff\ncpu MHz\t: fast\ncache size\t: big\n",
				"/sys/devices/system/cpu/cpu0/topology/core_id":         "7\n",
				"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq": "3000000\n",
				"/sys/devices/system/cpu/cpu1/cpufreq/cpuinfo_max_freq": "3200000000\n",
				"/sys/devices/system/cpu/cpu1/topology/core_id":         "1\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(2, i.Count)
				s.Require().Len(i.CPUs, 2)
				s.Equal("ARM", i.CPUs[0].VendorID)
				s.Equal("0xd0c", i.CPUs[0].ModelID)
				s.Equal(int32(1), i.CPUs[0].Stepping)
				s.Equal([]string{"fp", "asimd", "evtstrm"}, i.CPUs[0].Flags)
				s.Equal("7", i.CPUs[0].CoreID)
				s.Equal(float64(3000), i.CPUs[0].Speed)
				s.Empty(i.CPUs[1].VendorID)
				s.Equal(float64(3200), i.CPUs[1].Speed)
			},
		},
		{
			name:    "linux: a speed that is not a number, and no /proc/stat",
			variant: "linux",
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/cpuinfo": "model name\t: before any processor\n",
				"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq": "soon\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(0, i.Count)
				s.Empty(i.CPUs)
			},
		},
		{
			name:    "linux: a sysfs speed that is not a number, with no Exec for lscpu",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/cpuinfo": "processor\t: 0\ncpu MHz\t: 1200\n",
				"/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq": "soon\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(float64(1200), i.Speed)
			},
		},
		{
			name:    "darwin: no Exec gives no fact",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: an Intel Mac's family, model, flags and frequency, with the logical count as cores when hw.physicalcpu fails",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n hw.logicalcpu":              "8\n",
				"sysctl -n machdep.cpu.brand_string":   "Intel(R) Core(TM) i9\n",
				"sysctl -n machdep.cpu.vendor":         "GenuineIntel\n",
				"sysctl -n machdep.cpu.family":         "6\n",
				"sysctl -n machdep.cpu.model":          "158\n",
				"sysctl -n machdep.cpu.stepping":       "13\n",
				"sysctl -n machdep.cpu.features":       "FPU VME\n",
				"sysctl -n machdep.cpu.leaf7_features": "AVX2\n",
				"sysctl -n hw.cpufrequency":            "2400000000\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal("GenuineIntel", i.VendorID)
				s.Equal("6", i.Family)
				s.Equal("158", i.ModelID)
				s.Equal(int32(13), i.Stepping)
				s.Equal([]string{"fpu", "vme", "avx2"}, i.Flags)
				s.Equal(float64(2400), i.Speed)
				s.Equal(8, i.Cores)
			},
		},
		{
			name:    "darwin: Apple Silicon's frequency table",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n hw.logicalcpu": "10\n", pmgr: "  | " + table + "\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(float64(3228), i.Speed)
			},
		},
		{
			name:    "darwin: no table in ioreg's output",
			variant: "darwin",
			runner: sourcetest.Commands{
				Out: map[string]string{"sysctl -n hw.logicalcpu": "10\n", pmgr: "{}\n"},
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Zero(i.Speed)
			},
		},
		{
			name:    "darwin: a table that is not hex",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n hw.logicalcpu": "10\n", pmgr: "\"voltage-states5-sram\" = <abc>\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Zero(i.Speed)
			},
		},
		{
			name:    "linux: base only, no vulns dir no lscpu; the cores are the distinct core ids, not the threads",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec:    noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.Vulnerabilities)
				s.Nil(i.Caches)
				s.Nil(i.NumaNodes)
				s.Equal(16, i.Count)
				s.Equal(8, i.Cores)
				s.Equal(1, i.Sockets)
				// One entry for each processor block of /proc/cpuinfo.
				s.Require().Len(i.CPUs, 16)
				s.Equal("GenuineIntel", i.CPUs[0].VendorID)
				s.Equal("0", i.CPUs[0].PhysicalID)
				s.Equal(int32(1), i.CPUs[0].Cores)
				s.Equal(float64(2400), i.Speed)
				s.Equal(int32(25600), i.CacheSize)
				s.Equal([]string{"fpu", "vme"}, i.Flags)
			},
		},
		{
			name:    "linux: a hybrid CPU's P-core and E-core have their own model and speed, and count as two cores",
			variant: "linux",
			cpuinfo: block(
				0,
				"vendor_id\t: GenuineIntel\nmodel name\t: P-core\ncpu MHz\t\t: 5200\n"+
					"physical id\t: 0\ncore id\t\t: 0\n",
			) +
				block(
					1,
					"vendor_id\t: GenuineIntel\nmodel name\t: E-core\ncpu MHz\t\t: 3800\n"+
						"physical id\t: 0\ncore id\t\t: 8\n",
				),
			fs:   newVulnFS(s, nil),
			exec: noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Require().Len(i.CPUs, 2)
				s.Equal("P-core", i.CPUs[0].ModelName)
				s.Equal(float64(5200), i.CPUs[0].Speed)
				s.Equal("E-core", i.CPUs[1].ModelName)
				s.Equal(float64(3800), i.CPUs[1].Speed)
				s.Equal(2, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:    "linux: arm64 has no model name in cpuinfo, so lscpu's is used, with its cores",
			variant: "linux",
			fs: sourcetest.FS(s.T(), map[string]string{
				"/proc/cpuinfo": "processor\t: 0\nCPU implementer\t: 0x41\nCPU part\t: 0xd0c\n" +
					"processor\t: 1\nCPU implementer\t: 0x41\nCPU part\t: 0xd0c\n",
			}),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(
					t,
					"Architecture:        aarch64\nModel name:          Neoverse-N1\n"+
						"Socket(s):           1\nCore(s) per socket:  2\nThread(s) per core:  1\n",
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal("Neoverse-N1", i.ModelName)
				s.Equal(2, i.Cores)
				s.Equal(2, i.Count)
			},
		},
		{
			name:    "linux: vulnerabilities populated",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs: newVulnFS(s, map[string]string{
				"meltdown":   "Mitigation: PTI\n",
				"spectre_v2": "Mitigation: Retpolines\n",
				"mds":        "Not affected\n",
			}),
			exec: noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.Equal("Mitigation: Retpolines", i.Vulnerabilities["spectre_v2"])
				s.Equal("Not affected", i.Vulnerabilities["mds"])
			},
		},
		{
			name:    "linux: empty vulnerabilities directory yields nil map",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, map[string]string{}),
			exec:    noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.Vulnerabilities)
			},
		},
		{
			name:    "linux: x86 lscpu populates caches and numa_nodes, and its sockets times cores per socket are the cores",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
CPU op-mode(s):      32-bit, 64-bit
Byte Order:          Little Endian
Address sizes:       46 bits physical, 48 bits virtual
CPU(s):              16
On-line CPU(s) list: 0-15
Off-line CPU(s) list: 16-31
Socket(s):           1
Core(s) per socket:  8
Thread(s) per core:  2
Vendor ID:           GenuineIntel
BIOS Vendor ID:      Dell Inc.
BIOS Model name:     PowerEdge R750
CPU max MHz:         3200.0000
CPU min MHz:         800.0000
CPU dynamic MHz:     2400
BogoMIPS:            4600.00
Virtualization:      VT-x
Virtualization type: full
Hypervisor vendor:   KVM
L1d cache:           32 KiB
L1i cache:           32 KiB
L2 cache:            256 KiB
L2d cache:           128 KiB
L2i cache:           128 KiB
L3 cache:            25 MiB
L4 cache:            128 MiB
NUMA node(s):        2
NUMA node0 CPU(s):   0-15
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Require().NotNil(i.Caches)
				s.Equal("32 KiB", i.Caches.L1d)
				s.Equal("32 KiB", i.Caches.L1i)
				s.Equal("256 KiB", i.Caches.L2)
				s.Equal("128 KiB", i.Caches.L2d)
				s.Equal("128 KiB", i.Caches.L2i)
				s.Equal("25 MiB", i.Caches.L3)
				s.Equal("128 MiB", i.Caches.L4)
				s.Require().NotNil(i.NumaNodes)
				s.Equal([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, i.NumaNodes[0])
				s.Equal(2, i.NumaNodesCount)
				s.Equal(16, i.Online)
				s.Equal(16, i.Offline)
				s.Equal("Dell Inc.", i.BIOSVendorID)
				s.Equal("PowerEdge R750", i.BIOSModelName)
				s.Equal("3200.0000", i.MhzMax)
				s.Equal("800.0000", i.MhzMin)
				s.Equal("2400", i.MhzDynamic)
				s.Equal("4600.00", i.Bogomips)
				s.Equal([]string{"32-bit", "64-bit"}, i.Opmodes)
				s.Equal("little endian", i.ByteOrder)
				s.Equal([]string{"46 bits physical", "48 bits virtual"}, i.AddressSizes)
				s.Equal("VT-x", i.Virtualization)
				s.Equal("full", i.VirtualizationType)
				s.Equal("KVM", i.HypervisorVendor)
				s.Equal(16, i.Count)
				s.Equal(8, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:    "linux: s390x lscpu populates machine_type + dispatching_mode",
			variant: "linux",
			cpuinfo: oneCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        s390x
Machine type:        3906
Dispatching mode:    horizontal
Thread(s) per core:  1
Core(s) per socket:  3
Socket(s) per book:  2
Book(s) per drawer:  2
Drawer(s):           2
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal("3906", i.MachineType)
				s.Equal("horizontal", i.DispatchingMode)
				s.Equal(24, i.Count)
			},
		},
		{
			name:    "linux: s390x lscpu overrides counts",
			variant: "linux",
			cpuinfo: oneCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        s390x
Thread(s) per core:  1
Core(s) per socket:  3
Socket(s) per book:  2
Book(s) per drawer:  2
Drawer(s):           2
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(24, i.Count)
				s.Equal(24, i.Cores)
				s.Equal(2, i.Sockets)
			},
		},
		{
			name:    "linux: ppc64le lscpu overrides counts",
			variant: "linux",
			cpuinfo: oneCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        ppc64le
Thread(s) per core:  4
Core(s) per socket:  20
Socket(s):           2
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(160, i.Count)
				s.Equal(40, i.Cores)
				s.Equal(2, i.Sockets)
			},
		},
		{
			name:    "linux: lscpu whitespace-only CPU op-mode(s) leaves slice nil",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, "CPU op-mode(s):      ,   ,\n")
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.Opmodes)
			},
		},
		{
			name:    "linux: malformed lscpu no-ops, base Info preserved",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, "no colons here\nmore garbage\n")
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.Caches)
				s.Nil(i.NumaNodes)
				s.Equal(16, i.Count)
			},
		},
		{
			name:    "linux: NUMA CPU list with range+singletons",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   0-3,8,10-11
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal([]int{0, 1, 2, 3, 8, 10, 11}, i.NumaNodes[0])
			},
		},
		{
			name:    "linux: a NUMA node whose CPU list is malformed is left out",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   bad-data,x-y
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.NumaNodes)
			},
		},
		{
			name:    "linux: a NUMA node whose CPU range is reversed is left out",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   5-2
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.NumaNodes)
			},
		},
		{
			name:    "linux: vulnerabilities dir with subdir, subdir skipped files still read",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs: func() avfs.VFS {
				f := newVulnFS(s, map[string]string{"meltdown": "Mitigation: PTI\n"})
				s.Require().NoError(
					f.MkdirAll("/sys/devices/system/cpu/vulnerabilities/unrelated_subdir", 0o755),
				)
				return f
			}(),
			exec: noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal("Mitigation: PTI", i.Vulnerabilities["meltdown"])
				s.NotContains(i.Vulnerabilities, "unrelated_subdir")
			},
		},
		{
			name:    "linux: a vulnerabilities directory that cannot be read is partial",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs: sourcetest.FS(s.T(), map[string]string{
				"/sys/devices/system/cpu/vulnerabilities": "not a directory",
			}),
			exec: noLscpuExec,
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "read /sys/devices/system/cpu/vulnerabilities: ")
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Require().NotNil(i)
				s.Equal(16, i.Count)
				s.Nil(i.Vulnerabilities)
			},
		},
		{
			name:    "linux: a vulnerabilities link that points at nothing is left out",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs: sourcetest.Links(
				s.T(),
				newVulnFS(s, map[string]string{"meltdown": "Mitigation: PTI\n"}),
				map[string]string{"/sys/devices/system/cpu/vulnerabilities/gone": "/nowhere"},
			),
			exec: noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"meltdown": "Mitigation: PTI"}, i.Vulnerabilities)
			},
		},
		{
			name:    "linux: a vulnerabilities file that cannot be read is partial",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs: sourcetest.ReadError(
				newVulnFS(s, map[string]string{
					"meltdown": "Mitigation: PTI\n", "bad": "x",
				}),
				errors.New("read fail"),
				"/sys/devices/system/cpu/vulnerabilities/bad",
			),
			exec: noLscpuExec,
			validateFunc: func(got any, err error) {
				s.EqualError(err, "read /sys/devices/system/cpu/vulnerabilities/bad: read fail")
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Require().NotNil(i)
				s.Equal(16, i.Count)
			},
		},

		{
			name:    "linux: a failed lscpu keeps the cpuinfo facts",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{"lscpu": errors.New("exit status 1")},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.EqualError(err, "lscpu: exit status 1")
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Require().NotNil(i)
				s.Equal(16, i.Count)
			},
		},
		{
			name:    "linux: a cancelled context is an error",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Err: map[string]error{"lscpu": context.Canceled}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: lscpu with empty-value line skipped",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
L1d cache:
L1i cache:           32 KiB
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Require().NotNil(i.Caches)
				s.Equal("", i.Caches.L1d)
				s.Equal("32 KiB", i.Caches.L1i)
			},
		},
		{
			name:    "linux: NUMA CPU list with empty parts parses remaining",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   0,,3
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal([]int{0, 3}, i.NumaNodes[0])
			},
		},
		{
			name:    "linux: a NUMA node whose CPU list is not a number is left out",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   abc
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.NumaNodes)
			},
		},
		{
			name:    "linux: a NUMA node whose CPU list is all empty is left out",
			variant: "linux",
			cpuinfo: linuxCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        x86_64
NUMA node0 CPU(s):   ,,,
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Nil(i.NumaNodes)
			},
		},
		{
			name:    "linux: s390x lscpu with missing drawers falls back to 1",
			variant: "linux",
			cpuinfo: oneCPUInfo,
			fs:      newVulnFS(s, nil),
			exec: func(t *testing.T) source.Runner {
				return lscpuExec(t, `Architecture:        s390x
Thread(s) per core:  1
Core(s) per socket:  2
Socket(s) per book:  1
Book(s) per drawer:  1
`)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(2, i.Count)
				s.Equal(2, i.Cores)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:    "linux: no /proc/cpuinfo",
			variant: "linux",
			fs:      newVulnFS(s, nil),
			exec:    noLscpuExec,
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "linux: no processor blocks, the count from /proc/stat",
			variant: "linux",
			cpuinfo: "\n",
			stat:    "cpu  1 2 3\ncpu0 1\ncpu1 1\ncpu2 1\ncpu3 1\nintr 5\n",
			fs:      newVulnFS(s, nil),
			exec:    noLscpuExec,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(4, i.Count)
				s.Equal(0, i.Sockets)
				s.Equal(0, i.Cores)
				s.Empty(i.ModelName)
			},
		},
		{
			name:    "darwin: Intel Mac with hyperthreading, all sysctls override",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "6\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {out: "2600000000\n"},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(12, i.Count)
				s.Equal(6, i.Cores)
				s.Equal(1, i.Sockets)
				s.Equal(2600.0, i.Speed)
			},
		},
		{
			name:    "darwin: Apple Silicon, cpufrequency_max and cpufrequency both absent",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "10\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(10, i.Cores)
				s.Equal(1, i.Sockets)
				s.Equal(0.0, i.Speed)
			},
		},
		{
			name:    "darwin: Apple Silicon fallback, cpufrequency_max absent but cpufrequency present",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "8\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {out: "3200000000\n"},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(3200.0, i.Speed)
			},
		},
		{
			name:    "darwin: hw.physicalcpu fails, Cores unchanged",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {err: errors.New("unknown oid")},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(12, i.Cores)
			},
		},
		{
			name:    "darwin: hw.packages fails, Sockets unchanged",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "6\n"},
					"hw.packages":         {err: errors.New("unknown oid")},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(1, i.Sockets)
			},
		},
		{
			name:    "darwin: hw.physicalcpu non-numeric, Cores unchanged",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "xyz\n"},
					"hw.packages":         {err: errors.New("unknown oid")},
					"hw.cpufrequency_max": {err: errors.New("unknown oid")},
					"hw.cpufrequency":     {err: errors.New("unknown oid")},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(12, i.Cores)
			},
		},
		{
			name:    "darwin: hw.cpufrequency_max non-numeric falls through to hw.cpufrequency",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"hw.physicalcpu":      {out: "6\n"},
					"hw.packages":         {out: "1\n"},
					"hw.cpufrequency_max": {out: "not-a-number\n"},
					"hw.cpufrequency":     {out: "2800000000\n"},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Equal(2800.0, i.Speed)
			},
		},
		{
			name:    "darwin: hw.logicalcpu fails",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(
					t,
					map[string]sysctlRet{"hw.logicalcpu": {err: errors.New("sysctl error")}},
				)
			},
			validateFunc: func(got any, err error) {
				s.EqualError(err, "sysctl hw.logicalcpu: sysctl error")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: a cancelled context keeps its cause",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{"hw.logicalcpu": {err: context.Canceled}})
			},
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, context.Canceled)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: an hw.logicalcpu that is not a number is an error",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{"hw.logicalcpu": {out: "many\n"}})
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "parse sysctl hw.logicalcpu: ")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: Apple Silicon has no family or model, and they stay empty",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return darwinExec(t, map[string]sysctlRet{
					"machdep.cpu.family": {err: errors.New("unknown oid")},
					"machdep.cpu.model":  {err: errors.New("unknown oid")},
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				i, ok := got.(*cpu.Info)
				s.Require().True(ok)
				s.Empty(i.Family)
				s.Empty(i.ModelID)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c cpu.Collector
			var exec source.Runner = tt.runner.Mock(s.T())
			if tt.exec != nil {
				exec = tt.exec(s.T())
			}
			switch tt.variant {
			case "linux":
				fs := tt.fs
				if fs == nil {
					fs = memfs.New()
				}
				for path, content := range map[string]string{"/proc/cpuinfo": tt.cpuinfo, "/proc/stat": tt.stat} {
					if content != "" {
						_ = fs.MkdirAll("/proc", 0o755)
						s.Require().NoError(fs.WriteFile(path, []byte(content), 0o644))
					}
				}
				c = &cpu.Linux{FS: fs, Exec: exec}
			case "darwin":
				c = &cpu.Darwin{Exec: exec}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
