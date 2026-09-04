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
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Linux", platformDebian, osLinux},
		{"rhel dispatches to Linux", platformRHEL, osLinux},
		{"arch dispatches to Linux", platformArch, osLinux},
		{"unknown dispatches to Linux", "", osLinux},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			plat.Detect = func() string { return tt.detect }
			c := platform.New()
			s.Equal("platform", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*platform.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*platform.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *PlatformPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		hostInfo func(context.Context) (*host.InfoStat, error)
		fs       avfs.VFS
		exec     func(*testing.T) executor.Executor
		wantErr  bool
		want     platform.Info
	}{
		{
			name:    "linux: ubuntu happy path, gopsutil populates everything",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "ubuntu", PlatformVersion: "24.04", PlatformFamily: platformDebian,
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "ubuntu", Version: "24.04",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: centos 7 supplements minor version from /etc/redhat-release",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: platformCentOS, PlatformVersion: "7", PlatformFamily: platformRHEL,
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS Linux release 7.9.2009 (Core)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformCentOS, Version: "7.9.2009",
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: rhel 9.3 already dotted, no supplement",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: platformRHEL, PlatformVersion: value93, PlatformFamily: platformRHEL,
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "Red Hat Enterprise Linux release 9.99 (Plow)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "redhat", Version: value93,
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: centos 7 missing /etc/redhat-release version stays 7",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: platformCentOS, PlatformVersion: "7", PlatformFamily: platformRHEL,
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformCentOS, Version: "7",
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: debian testing empty version supplemented from /etc/debian_version",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: platformDebian, PlatformVersion: "", PlatformFamily: platformDebian,
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "trixie/sid\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformDebian, Version: "trixie/sid",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/redhat-release",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS release 6.10 (Final)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformCentOS, Version: "6.10",
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/SuSE-release with PATCHLEVEL",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "SUSE Linux Enterprise Server 11\nVERSION = 11\nPATCHLEVEL = 4\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformSUSE, Version: "11.4",
				Family: platformSUSE, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy SuSE without PATCHLEVEL",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "VERSION = 12\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformSUSE, Version: "12",
				Family: platformSUSE, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/debian_version",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "11.7\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformDebian, Version: "11.7",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/arch-release rolling no version",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/arch-release": "",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformArch, Version: "",
				Family: platformArch, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/system-release Amazon",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/system-release": "Amazon Linux release 2 (Karoo)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "amazon", Version: "2",
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/gentoo-release",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/gentoo-release": "Gentoo Base System release 2.13\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "gentoo", Version: "2.13",
				Family: "gentoo", CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, malformed legacy file skipped next succeeds",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "garbage with no version\n",
				"/etc/debian_version": "12\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformDebian, Version: "12",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release, SuSE-release without VERSION line falls through",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release":   "SUSE Linux Enterprise Server\n",
				"/etc/debian_version": "12\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformDebian, Version: "12",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: no os-release + no legacy files, empty Info no error",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: family fallback rocky → rhel (empty PlatformFamily)",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "rocky", PlatformVersion: value93,
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "rocky", Version: value93,
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: family fallback kali → debian",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "kali", PlatformVersion: "2023",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "kali", Version: "2023",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: remap archarm → arch",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "archarm", PlatformVersion: "rolling",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformArch, Version: "rolling",
				Family: platformArch, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: remap cumulus-linux → cumulus",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "cumulus-linux", PlatformVersion: "5.0",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "cumulus", Version: "5.0",
				Family: platformDebian, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: remap sles_sap → suse",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "sles_sap", PlatformVersion: "15-SP5",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: platformSUSE, Version: "15-SP5",
				Family: platformSUSE, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:    "linux: nil FS skips supplement and legacy, gopsutil + family fallback only",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "almalinux", PlatformVersion: value93,
				}, nil
			},
			fs: nil,
			want: platform.Info{
				OS: runtime.GOOS, Name: "almalinux", Version: value93,
				Family: platformRHEL, CPUArchitecture: runtime.GOARCH,
			},
		},
		{
			name:     "linux: nil info yields minimal Info (no gopsutil data)",
			variant:  osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) { return nil, nil },
			fs:       fsWith(s.T(), nil),
			want:     platform.Info{OS: runtime.GOOS, CPUArchitecture: runtime.GOARCH},
		},
		{
			name:    "linux: gopsutil error propagated",
			variant: osLinux,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return nil, errors.New("host.Info failed")
			},
			fs:      fsWith(s.T(), nil),
			wantErr: true,
		},
		{
			name:    "darwin: macOS with RSR patch, BuildVersion + ProductVersionExtra populate",
			variant: osDarwin,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: osDarwin, PlatformVersion: "14.4.1",
					PlatformFamily: "Standalone Workstation",
					KernelVersion:  "fallback-kernel",
				}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return swVersExec(t, []byte(swVersWithRSR), nil)
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: osDarwin, Version: "14.4.1",
				VersionExtra: "(a)", Family: "Standalone Workstation",
				CPUArchitecture: runtime.GOARCH, Build: "23E224",
			},
		},
		{
			name:    "darwin: macOS without RSR, BuildVersion populates VersionExtra empty",
			variant: osDarwin,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: osDarwin, PlatformVersion: "13.5",
					KernelVersion: "fallback-kernel",
				}, nil
			},
			exec: func(t *testing.T) executor.Executor { return swVersExec(t, []byte(swVersNoRSR), nil) },
			want: platform.Info{
				OS: runtime.GOOS, Name: osDarwin, Version: "13.5",
				CPUArchitecture: runtime.GOARCH, Build: "22G74",
			},
		},
		{
			name:    "darwin: sw_vers error, Build falls back to gopsutil KernelVersion",
			variant: osDarwin,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: osDarwin, PlatformVersion: "12.6",
					KernelVersion: "21G115",
				}, nil
			},
			exec: func(t *testing.T) executor.Executor { return swVersExec(t, nil, errors.New("not found")) },
			want: platform.Info{
				OS: runtime.GOOS, Name: osDarwin, Version: "12.6",
				CPUArchitecture: runtime.GOARCH, Build: "21G115",
			},
		},
		{
			name:    "darwin: sw_vers output with no-colon line, skipped valid lines parsed",
			variant: osDarwin,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{Platform: osDarwin, PlatformVersion: value140}, nil
			},
			exec: func(t *testing.T) executor.Executor {
				return swVersExec(t,
					[]byte("no colon line\nBuildVersion:\t23A344\nProductVersionExtra:\n"),
					nil)
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: osDarwin, Version: value140,
				CPUArchitecture: runtime.GOARCH, Build: "23A344",
			},
		},
		{
			name:    "darwin: nil Exec, extension skipped Build from KernelVersion fallback",
			variant: osDarwin,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: osDarwin, PlatformVersion: value140,
					KernelVersion: "23A344",
				}, nil
			},
			exec: func(*testing.T) executor.Executor { return nil },
			want: platform.Info{
				OS: runtime.GOOS, Name: osDarwin, Version: value140,
				CPUArchitecture: runtime.GOARCH, Build: "23A344",
			},
		},
		{
			name:    "darwin: gopsutil error propagated",
			variant: osDarwin,
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return nil, errors.New("boom")
			},
			exec:    func(t *testing.T) executor.Executor { return swVersExec(t, nil, nil) },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer platform.SetHostInfoFn(tt.hostInfo)()
			var c platform.Collector
			switch tt.variant {
			case osLinux:
				c = &platform.Linux{FS: tt.fs}
			case osDarwin:
				c = &platform.Darwin{Exec: tt.exec(s.T())}
			default:
				s.Failf("unhandled case", "%v", tt.variant)
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*platform.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
