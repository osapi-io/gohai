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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	plat "github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
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

// swVersExec returns a MockRunner that canned-answers `sw_vers`.
func swVersExec(
	t *testing.T,
	out []byte, err error,
) source.Runner {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "sw_vers").
		Return(out, err).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("not found")).
		AnyTimes()
	m.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("not found")).
		AnyTimes()
	return m
}

// archExec returns a runner that answers the architecture reads with out,
// sw_vers with nothing, and fails each other line.
func archExec(
	t *testing.T,
	out map[string]string,
) source.Runner {
	r := sourcetest.Commands{Out: map[string]string{"sw_vers": ""}, Err: map[string]error{}}
	for k, v := range out {
		r.Out[k] = v
	}
	for _, line := range []string{"uname -m", "sysctl -n hw.optional.arm64"} {
		if _, ok := r.Out[line]; !ok {
			r.Err[line] = errors.New("not found")
		}
	}
	return r.Mock(t)
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
			s.Equal("system", string(c.Category()))
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
		osRelease    string
		fs           avfs.VFS
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:      "linux: ubuntu happy path, os-release populates everything",
			variant:   "linux",
			osRelease: "ID=ubuntu\nVERSION_ID=24.04\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "ubuntu", Version: "24.04",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: centos 7 supplements minor version from /etc/redhat-release",
			variant:   "linux",
			osRelease: "ID=centos\nVERSION_ID=7\n",
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS Linux release 7.9.2009 (Core)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "centos", Version: "7.9.2009",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: rhel 9.3 already dotted, no supplement",
			variant:   "linux",
			osRelease: "ID=rhel\nVERSION_ID=9.3\n",
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "Red Hat Enterprise Linux release 9.99 (Plow)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "redhat", Version: "9.3",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: centos 7 missing /etc/redhat-release version stays 7",
			variant:   "linux",
			osRelease: "ID=centos\nVERSION_ID=7\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "centos", Version: "7",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: debian testing empty version supplemented from /etc/debian_version",
			variant:   "linux",
			osRelease: "ID=debian\n",
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "trixie/sid\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "debian", Version: "trixie/sid",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/redhat-release from Red Hat",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "Red Hat Enterprise Linux Server release 7.9 (Maipo)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "redhat", Version: "7.9",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/system-release, a name Ohai has no family for",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/system-release": "Foo Linux release 1.0\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("rhel", info.Family, "the legacy file's own family")
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/redhat-release",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS release 6.10 (Final)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "centos", Version: "6.10",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/SuSE-release with PATCHLEVEL",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "SUSE Linux Enterprise Server 11\nVERSION = 11\nPATCHLEVEL = 4\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "suse", Version: "11.4",
					Family: "suse", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy SuSE without PATCHLEVEL",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "VERSION = 12\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "suse", Version: "12",
					Family: "suse", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/debian_version",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "11.7\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "debian", Version: "11.7",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/arch-release rolling no version",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/arch-release": "",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "arch", Version: "",
					Family: "arch", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/system-release Amazon",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/system-release": "Amazon Linux release 2 (Karoo)\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "amazon", Version: "2",
					Family: "amazon", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, legacy /etc/gentoo-release",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/gentoo-release": "Gentoo Base System release 2.13\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "gentoo", Version: "2.13",
					Family: "gentoo", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, malformed legacy file skipped next succeeds",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "garbage with no version\n",
				"/etc/debian_version": "12\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "debian", Version: "12",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release, SuSE-release without VERSION line falls through",
			variant: "linux",
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release":   "SUSE Linux Enterprise Server\n",
				"/etc/debian_version": "12\n",
			}),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "debian", Version: "12",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "linux: no os-release + no legacy files, empty Info no error",
			variant: "linux",
			fs:      fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: family fallback rocky → rhel (empty PlatformFamily)",
			variant:   "linux",
			osRelease: "ID=rocky\nVERSION_ID=9.3\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "rocky", Version: "9.3",
					Family: "rhel", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: family fallback kali → debian",
			variant:   "linux",
			osRelease: "ID=kali\nVERSION_ID=2023\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "kali", Version: "2023",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: remap archarm → arch",
			variant:   "linux",
			osRelease: "ID=archarm\nVERSION_ID=rolling\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "arch", Version: "rolling",
					Family: "arch", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: remap cumulus-linux → cumulus",
			variant:   "linux",
			osRelease: "ID=cumulus-linux\nVERSION_ID=5.0\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "cumulus", Version: "5.0",
					Family: "debian", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: remap sles_sap → suse",
			variant:   "linux",
			osRelease: "ID=sles_sap\nVERSION_ID=15-SP5\n",
			fs:        fsWith(s.T(), nil),
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", Name: "suse", Version: "15-SP5",
					Family: "suse", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:      "linux: nil FS reads nothing",
			variant:   "linux",
			osRelease: "ID=almalinux\nVERSION_ID=9.3\n",
			fs:        nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "linux", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "darwin: macOS with RSR patch, BuildVersion + ProductVersionExtra populate",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return swVersExec(t, []byte(swVersWithRSR), nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "darwin", Name: "darwin", Version: "14.4.1",
					VersionExtra: "(a)", Family: "darwin",
					CPUArchitecture: runtime.GOARCH, Build: "23E224",
				}, *info)
			},
		},
		{
			name:    "darwin: macOS without RSR, BuildVersion populates VersionExtra empty",
			variant: "darwin",
			exec:    func(t *testing.T) source.Runner { return swVersExec(t, []byte(swVersNoRSR), nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "darwin", Name: "darwin", Version: "13.5",
					Family: "darwin", CPUArchitecture: runtime.GOARCH, Build: "22G74",
				}, *info)
			},
		},
		{
			name:    "darwin: sw_vers error, no version or build",
			variant: "darwin",
			exec:    func(t *testing.T) source.Runner { return swVersExec(t, nil, errors.New("not found")) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "darwin", Name: "darwin",
					Family: "darwin", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
		{
			name:    "darwin: sw_vers output with no-colon line, skipped valid lines parsed",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return swVersExec(t,
					[]byte("no colon line\nBuildVersion:\t23A344\nProductVersionExtra:\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "darwin", Name: "darwin",
					Family: "darwin", CPUArchitecture: runtime.GOARCH, Build: "23A344",
				}, *info)
			},
		},
		{
			name:    "linux: uname's x86_64 is amd64",
			variant: "linux",
			fs:      memfs.New(),
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"uname -m": "x86_64\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("amd64", info.CPUArchitecture)
			},
		},
		{
			name:    "linux: uname's aarch64 is arm64",
			variant: "linux",
			fs:      memfs.New(),
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"uname -m": "aarch64\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("arm64", info.CPUArchitecture)
			},
		},
		{
			name:    "linux: a machine with no Go name is reported as uname gives it",
			variant: "linux",
			fs:      memfs.New(),
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"uname -m": "sparc64\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("sparc64", info.CPUArchitecture)
			},
		},
		{
			name:    "linux: an empty uname is the binary's architecture",
			variant: "linux",
			fs:      memfs.New(),
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"uname -m": "\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.GoArch(), info.CPUArchitecture)
			},
		},
		{
			name:    "linux: a failed uname is the binary's architecture",
			variant: "linux",
			fs:      memfs.New(),
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.GoArch(), info.CPUArchitecture)
			},
		},
		{
			name:    "darwin: Apple silicon under Rosetta is arm64",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"sysctl -n hw.optional.arm64": "1\n",
					"uname -m":                    "x86_64\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("arm64", info.CPUArchitecture)
			},
		},
		{
			name:    "darwin: an Intel Mac is amd64",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"sysctl -n hw.optional.arm64": "0\n",
					"uname -m":                    "x86_64\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("amd64", info.CPUArchitecture)
			},
		},
		{
			name:    "darwin: no arm64 key, uname decides",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return archExec(t, map[string]string{
					"uname -m": "x86_64\n",
				})
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal("amd64", info.CPUArchitecture)
			},
		},
		{
			name:    "darwin: nil Exec, no version or build",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*platform.Info)
				s.Require().True(ok)
				s.Equal(platform.Info{
					OS: "darwin", Name: "darwin",
					Family: "darwin", CPUArchitecture: runtime.GOARCH,
				}, *info)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			if tt.osRelease != "" && tt.fs != nil {
				s.Require().NoError(tt.fs.MkdirAll("/etc", 0o755))
				s.Require().NoError(tt.fs.WriteFile("/etc/os-release", []byte(tt.osRelease), 0o644))
			}
			var c platform.Collector
			switch tt.variant {
			case "linux":
				l := &platform.Linux{FS: tt.fs}
				if tt.exec != nil {
					l.Exec = tt.exec(s.T())
				}
				c = l
			case "darwin":
				c = &platform.Darwin{Exec: tt.exec(s.T())}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
