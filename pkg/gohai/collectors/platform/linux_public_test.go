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

	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

type PlatformLinuxPublicTestSuite struct {
	suite.Suite
}

func TestPlatformLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformLinuxPublicTestSuite))
}

// fsWith builds a memfs containing the given (path → contents) map.
// Parent directories are auto-created.
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

func dirOf(p string) string {
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

func (s *PlatformLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		hostInfo func(context.Context) (*host.InfoStat, error)
		fs       avfs.VFS
		wantErr  bool
		want     platform.Info
	}{
		{
			name: "ubuntu happy path: gopsutil populates everything",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "ubuntu", PlatformVersion: "24.04", PlatformFamily: "debian",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "ubuntu", Version: "24.04",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "centos 7 supplements minor version from /etc/redhat-release",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "centos", PlatformVersion: "7", PlatformFamily: "rhel",
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS Linux release 7.9.2009 (Core)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "centos", Version: "7.9.2009",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "rhel 9.3 already dotted: no supplement",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "rhel", PlatformVersion: "9.3", PlatformFamily: "rhel",
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "Red Hat Enterprise Linux release 9.99 (Plow)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "redhat", Version: "9.3",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "centos 7 missing /etc/redhat-release: version stays 7",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "centos", PlatformVersion: "7", PlatformFamily: "rhel",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "centos", Version: "7",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "debian testing: empty version supplemented from /etc/debian_version",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "debian", PlatformVersion: "", PlatformFamily: "debian",
				}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "trixie/sid\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "debian", Version: "trixie/sid",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy fallback /etc/redhat-release",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "CentOS release 6.10 (Final)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "centos", Version: "6.10",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy fallback /etc/SuSE-release with PATCHLEVEL",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "SUSE Linux Enterprise Server 11\nVERSION = 11\nPATCHLEVEL = 4\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "suse", Version: "11.4",
				Family: "suse", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy SuSE without PATCHLEVEL",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release": "VERSION = 12\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "suse", Version: "12",
				Family: "suse", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy fallback /etc/debian_version",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/debian_version": "11.7\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "debian", Version: "11.7",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy /etc/arch-release (rolling, no version)",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/arch-release": "",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "arch", Version: "",
				Family: "arch", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy /etc/system-release Amazon",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/system-release": "Amazon Linux release 2 (Karoo)\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "amazon", Version: "2",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: legacy /etc/gentoo-release",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/gentoo-release": "Gentoo Base System release 2.13\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "gentoo", Version: "2.13",
				Family: "gentoo", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: malformed legacy file skipped, next succeeds",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/redhat-release": "garbage with no version\n",
				"/etc/debian_version": "12\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "debian", Version: "12",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release: SuSE-release without VERSION line falls through",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), map[string]string{
				"/etc/SuSE-release":   "SUSE Linux Enterprise Server\n",
				"/etc/debian_version": "12\n",
			}),
			want: platform.Info{
				OS: runtime.GOOS, Name: "debian", Version: "12",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "no os-release + no legacy files: empty Info, no error",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Architecture: runtime.GOARCH,
			},
		},
		{
			name: "family fallback: rocky has empty PlatformFamily, table fills rhel",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "rocky", PlatformVersion: "9.3",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "rocky", Version: "9.3",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "family fallback: kali → debian",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "kali", PlatformVersion: "2023",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "kali", Version: "2023",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "remap: archarm → arch",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "archarm", PlatformVersion: "rolling",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "arch", Version: "rolling",
				Family: "arch", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "remap: cumulus-linux → cumulus",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "cumulus-linux", PlatformVersion: "5.0",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "cumulus", Version: "5.0",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "remap: sles_sap → suse",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "sles_sap", PlatformVersion: "15-SP5",
				}, nil
			},
			fs: fsWith(s.T(), nil),
			want: platform.Info{
				OS: runtime.GOOS, Name: "suse", Version: "15-SP5",
				Family: "suse", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "nil FS: supplement and legacy skipped, gopsutil + family fallback only",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "almalinux", PlatformVersion: "9.3",
				}, nil
			},
			fs: nil,
			want: platform.Info{
				OS: runtime.GOOS, Name: "almalinux", Version: "9.3",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "gopsutil error propagated",
			hostInfo: func(context.Context) (*host.InfoStat, error) {
				return nil, errors.New("host.Info failed")
			},
			fs:      fsWith(s.T(), nil),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer platform.SetHostInfoFn(tt.hostInfo)()
			c := &platform.Linux{FS: tt.fs}
			got, err := c.Collect(context.Background())
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

func (s *PlatformLinuxPublicTestSuite) TestReadPlatform() {
	tests := []struct {
		name       string
		fn         func(context.Context) (*host.InfoStat, error)
		wantErr    bool
		want       platform.Info
		wantKernel string
	}{
		{
			name: "ubuntu canonical",
			fn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "ubuntu", PlatformVersion: "24.04",
					PlatformFamily: "debian", KernelVersion: "6.1.0",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "ubuntu", Version: "24.04",
				Family: "debian", Architecture: runtime.GOARCH,
			},
			wantKernel: "6.1.0",
		},
		{
			name: "rhel remaps to redhat",
			fn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform: "rhel", PlatformVersion: "9.4", PlatformFamily: "rhel",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "redhat", Version: "9.4",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "nil info yields minimal Info",
			fn:   func(context.Context) (*host.InfoStat, error) { return nil, nil },
			want: platform.Info{OS: runtime.GOOS, Architecture: runtime.GOARCH},
		},
		{
			name:    "gopsutil error propagated",
			fn:      func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer platform.SetHostInfoFn(tt.fn)()
			got, kernel, err := platform.ReadPlatform(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, *got)
			s.Equal(tt.wantKernel, kernel)
		})
	}
}
