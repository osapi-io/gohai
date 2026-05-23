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

package selinux_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/selinux"
)

var (
	_ collector.Collector = (*selinux.Linux)(nil)
	_ collector.Collector = (*selinux.Darwin)(nil)
)

// fsWith builds a memfs with the given path→content entries.
func fsWith(
	t require.TestingT,
	files map[string]string,
) avfs.VFS {
	f := memfs.New()
	for path, content := range files {
		_ = f.MkdirAll("/etc/selinux", 0o755)
		require.NoError(t, f.WriteFile(path, []byte(content), fs.FileMode(0o644)))
	}
	return f
}

// sestatusEnforcing is a minimal sestatus output for an enforcing host.
const sestatusEnforcing = `SELinux status:                 enabled
SELinuxfs mount:                /sys/fs/selinux
SELinux mount point:            /sys/fs/selinux
Loaded policy name:             targeted
Current mode:                   enforcing
Mode from config file:          enforcing
Policy MLS status:              enabled
Policy deny_unknown status:     allowed
Memory protection checking:     actual (secure)
Max kernel policy version:      33
Policy version:                 33
`

// sestatusPermissive is a minimal sestatus output for a permissive host.
const sestatusPermissive = `SELinux status:                 enabled
Loaded policy name:             minimum
Current mode:                   permissive
Max kernel policy version:      30
`

// sestatusDisabled is sestatus output when SELinux is disabled at
// runtime but the config file still exists.
const sestatusDisabled = `SELinux status:                 disabled
`

type SelinuxPublicTestSuite struct {
	suite.Suite
}

func TestSelinuxPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SelinuxPublicTestSuite))
}

func (s *SelinuxPublicTestSuite) TestNew() {
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
			c := selinux.New()
			s.Equal("selinux", c.Name())
			s.Equal("security", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*selinux.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*selinux.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *SelinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name       string
		variant    string
		setupFS    func() avfs.VFS
		setupExec  func(ctrl *gomock.Controller) *execmocks.MockExecutor
		wantNil    bool
		wantErr    bool
		wantStatus string
		wantMode   string
		wantPolicy string
		wantMaxKV  string
		wantPV     string
		wantLoaded string
	}{
		{
			name:    "darwin: returns nil — no SELinux",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			wantNil: true,
		},
		{
			name:    "linux: no /etc/selinux/config — disabled",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				return execmocks.NewMockExecutor(ctrl)
			},
			wantStatus: "disabled",
		},
		{
			name:    "linux: config SELINUX=disabled — disabled, no sestatus",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "# comment\nSELINUX=disabled\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				return execmocks.NewMockExecutor(ctrl)
			},
			wantStatus: "disabled",
			wantLoaded: "targeted",
		},
		{
			name:    "linux: enforcing with sestatus",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusEnforcing), nil)
				return m
			},
			wantStatus: "enabled",
			wantMode:   "enforcing",
			wantLoaded: "targeted",
			wantMaxKV:  "33",
			wantPV:     "33",
		},
		{
			name:    "linux: permissive with sestatus",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=permissive\nSELINUXTYPE=minimum\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusPermissive), nil)
				return m
			},
			wantStatus: "enabled",
			wantMode:   "permissive",
			wantLoaded: "minimum",
			wantMaxKV:  "30",
		},
		{
			name:    "linux: sestatus fails — status derived from config",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return(nil, errors.New("sestatus: command not found"))
				return m
			},
			wantStatus: "enabled",
			wantLoaded: "targeted",
		},
		{
			name:    "linux: sestatus returns disabled status",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusDisabled), nil)
				return m
			},
			wantStatus: "disabled",
			wantLoaded: "targeted",
		},
		{
			name:    "linux: nil executor — falls back to config only",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec:  nil,
			wantStatus: "enabled",
			wantLoaded: "targeted",
		},
		{
			// config line with no '=' separator exercises the !ok branch
			// in parseConfigFile.
			name:    "linux: config line without equals sign skipped",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nthis line has no equals\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusEnforcing), nil)
				return m
			},
			wantStatus: "enabled",
			wantMode:   "enforcing",
			wantLoaded: "targeted",
			wantMaxKV:  "33",
			wantPV:     "33",
		},
		{
			// sestatus output line with no ':' exercises the !ok branch
			// in parseSestatus.
			name:    "linux: sestatus line without colon skipped",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				m := execmocks.NewMockExecutor(ctrl)
				// Include a line with no colon to exercise the !ok branch.
				out := "SELinux status: enabled\nno colon here\nCurrent mode: enforcing\n"
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(out), nil)
				return m
			},
			wantStatus: "enabled",
			wantMode:   "enforcing",
			wantLoaded: "targeted",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctrl := gomock.NewController(s.T())

			var c selinux.Collector
			switch tt.variant {
			case "linux":
				l := &selinux.Linux{FS: tt.setupFS()}
				if tt.setupExec != nil {
					l.Exec = tt.setupExec(ctrl)
				}
				c = l
			case "darwin":
				c = &selinux.Darwin{}
			}

			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)

			if tt.wantNil {
				s.Nil(got)
				return
			}

			info, ok := got.(*selinux.Info)
			s.Require().True(ok)
			s.Equal(tt.wantStatus, info.Status)
			s.Equal(tt.wantMode, info.CurrentMode)
			s.Equal(tt.wantLoaded, info.LoadedPolicyName)
			s.Equal(tt.wantMaxKV, info.MaxKernelPolicyVersion)
			s.Equal(tt.wantPV, info.PolicyVersion)
		})
	}
}
