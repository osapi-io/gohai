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

package platform_test

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	plat "github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

var (
	_ collector.Collector = (*platform.Linux)(nil)
	_ collector.Collector = (*platform.Darwin)(nil)
)

type PlatformPublicTestSuite struct {
	suite.Suite
}

func TestPlatformPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

// fsWith builds a memfs containing the given (path → contents) map.
func fsWith(
	t require.TestingT,
	files map[string]string,
) avfs.VFS {
	fs := memfs.New()
	for path, content := range files {
		require.NoError(t, fs.MkdirAll(dirOf(path), 0o755))
		require.NoError(t, fs.WriteFile(path, []byte(content), 0o644))
	}
	return fs
}

func dirOf(
	p string,
) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "/"
}

// swVersExec returns a MockExecutor that canned-answers `sw_vers`.
func swVersExec(
	t *testing.T,
	out []byte, err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "sw_vers").
		Return(out, err).
		AnyTimes()
	return m
}

const swVersWithRSR = `ProductName:		macOS
ProductVersion:		14.4.1
ProductVersionExtra:	(a)
BuildVersion:		23E224
`

const swVersNoRSR = `ProductName:		macOS
ProductVersion:		13.5
BuildVersion:		22G74
`

func (s *PlatformPublicTestSuite) TestNew() {
	orig := plat.Detect
	defer func() { plat.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(platform.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c platform.Collector) {
				_, ok := c.(*platform.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c platform.Collector) {
				_, ok := c.(*platform.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c platform.Collector) {
				_, ok := c.(*platform.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c platform.Collector) {
				_, ok := c.(*platform.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c platform.Collector) {
				_, ok := c.(*platform.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			plat.Detect = func() string { return tt.detect }
			c := platform.New()
			s.Equal("platform", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *PlatformPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		hostInfo     func(context.Context) (*host.InfoStat, error)
		fs           avfs.VFS
		exec         func(*testing.T) executor.Executor
		validateFunc func(any, error)
	}{
		{
			name:    "linux: ubuntu happy path, gopsutil populates everything",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "ubuntu", PlatformVersion: "24.04", PlatformFamily: "debian",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "ubuntu", Version: "24.04",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: centos 7 supplements minor version from /etc/redhat-release",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "centos", PlatformVersion: "7", PlatformFamily: "rhel",
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS Linux release 7.9.2009 (Core)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "centos", Version: "7.9.2009",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: rhel 9.3 already dotted, no supplement",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "rhel", PlatformVersion: "9.3", PlatformFamily: "rhel",
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "Red Hat Enterprise Linux release 9.99 (Plow)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "redhat", Version: "9.3",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: centos 7 missing /etc/redhat-release version stays 7",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "centos", PlatformVersion: "7", PlatformFamily: "rhel",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "centos", Version: "7",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: debian testing empty version supplemented from /etc/debian_version",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "debian", PlatformVersion: "", PlatformFamily: "debian",
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "trixie/sid\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "debian", Version: "trixie/sid",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/redhat-release",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS release 6.10 (Final)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "centos", Version: "6.10",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/SuSE-release with PATCHLEVEL",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "SUSE Linux Enterprise Server 11\nVERSION = 11\nPATCHLEVEL = 4\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "suse", Version: "11.4",
					Family: "suse", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy SuSE without PATCHLEVEL",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "VERSION = 12\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "suse", Version: "12",
					Family: "suse", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/debian_version",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "11.7\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "debian", Version: "11.7",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/arch-release rolling no version",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/arch-release": "",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "arch", Version: "",
					Family: "arch", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/system-release Amazon",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/system-release": "Amazon Linux release 2 (Karoo)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "amazon", Version: "2",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/gentoo-release",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/gentoo-release": "Gentoo Base System release 2.13\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "gentoo", Version: "2.13",
					Family: "gentoo", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, malformed legacy file skipped next succeeds",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "garbage with no version\n",
				"/etc/debian_version": "12\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "debian", Version: "12",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, SuSE-release without VERSION line falls through",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release":   "SUSE Linux Enterprise Server\n",
				"/etc/debian_version": "12\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "debian", Version: "12",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release + no legacy files, empty Info no error",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: family fallback rocky → rhel (empty PlatformFamily)",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "rocky", PlatformVersion: "9.3",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "rocky", Version: "9.3",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: family fallback kali → debian",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "kali", PlatformVersion: "2023",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "kali", Version: "2023",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: remap archarm → arch",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "archarm", PlatformVersion: "rolling",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "arch", Version: "rolling",
					Family: "arch", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: remap cumulus-linux → cumulus",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "cumulus-linux", PlatformVersion: "5.0",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "cumulus", Version: "5.0",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: remap sles_sap → suse",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "sles_sap", PlatformVersion: "15-SP5",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "suse", Version: "15-SP5",
					Family: "suse", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: nil FS skips supplement and legacy, gopsutil + family fallback only",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "almalinux", PlatformVersion: "9.3",
				}, nil
			},
			fs: nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "almalinux", Version: "9.3",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:     "linux: nil info yields minimal Info (no gopsutil data)",
			variant:  "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) { return nil, nil },
			fs:       fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{OS: runtime.GOOS, CPUArchitecture: runtime.GOARCH}, *info)
			},
		},
		{
			name:    "linux: gopsutil error propagated",
			variant: "linux",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return nil, errors.New("host.Info failed")
			},
			fs: fsWith(s.T(), nil),
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: macOS with RSR patch, BuildVersion + ProductVersionExtra populate",
			variant: "darwin",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "14.4.1",
					PlatformFamily: "Standalone Workstation",
					KernelVersion:  "fallback-kernel",
				}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return swVersExec(t, []byte(swVersWithRSR), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "darwin", Version: "14.4.1",
					VersionExtra: "(a)", Family: "Standalone Workstation",
					CPUArchitecture: runtime.GOARCH, Build: "23E224",
				}, *info)
			},
		},
		{
			name:    "darwin: macOS without RSR, BuildVersion populates VersionExtra empty",
			variant: "darwin",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "13.5",
					KernelVersion: "fallback-kernel",
				}, nil
			},
			exec: func(t *testing.T) executor.Executor { return swVersExec(t, []byte(swVersNoRSR), nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "darwin", Version: "13.5",
					CPUArchitecture: runtime.GOARCH, Build: "22G74",
				}, *info)
			},
		},
		{
			name:    "darwin: sw_vers error, Build falls back to gopsutil KernelVersion",
			variant: "darwin",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "12.6",
					KernelVersion: "21G115",
				}, nil
			},
			exec: func(t *testing.T) executor.Executor { return swVersExec(t, nil, errors.New("not found")) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "darwin", Version: "12.6",
					CPUArchitecture: runtime.GOARCH, Build: "21G115",
				}, *info)
			},
		},
		{
			name:    "darwin: sw_vers output with no-colon line, skipped valid lines parsed",
			variant: "darwin",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{Platform: "darwin", PlatformVersion: "14.0"}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return swVersExec(t,
					[]byte("no colon line\nBuildVersion:\t23A344\nProductVersionExtra:\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "darwin", Version: "14.0",
					CPUArchitecture: runtime.GOARCH, Build: "23A344",
				}, *info)
			},
		},
		{
			name:    "darwin: nil Exec, extension skipped Build from KernelVersion fallback",
			variant: "darwin",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "darwin", PlatformVersion: "14.0",
					KernelVersion: "23A344",
				}, nil
			},
			exec: func(*testing.T) executor.Executor { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: runtime.GOOS, Name: "darwin", Version: "14.0",
					CPUArchitecture: runtime.GOARCH, Build: "23A344",
				}, *info)
			},
		},
		{
			name:    "darwin: gopsutil error propagated",
			variant: "darwin",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return nil, errors.New("boom")
			},
			exec: func(t *testing.T) executor.Executor { return swVersExec(t, nil, nil) },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer platform.SetHostInfoFn(tt.hostInfo)()
			var c platform.Collector
			switch tt.variant {
			case "linux":
				c = &platform.Linux{FS: tt.fs}
			case "darwin":
				c = &platform.Darwin{Exec: tt.exec(s.T())}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
