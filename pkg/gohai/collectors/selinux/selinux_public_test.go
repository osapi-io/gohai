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
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
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
		name         string
		detect       string
		validateFunc func(selinux.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c selinux.Collector) {
				_, ok := c.(*selinux.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c selinux.Collector) {
				_, ok := c.(*selinux.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c selinux.Collector) {
				_, ok := c.(*selinux.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c selinux.Collector) {
				_, ok := c.(*selinux.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c selinux.Collector) {
				_, ok := c.(*selinux.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := selinux.New()
			s.Equal("selinux", c.Name())
			s.Equal("security", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SelinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		setupFS      func() avfs.VFS
		setupExec    func(ctrl *gomock.Controller) *execmocks.MockExecutor
		wantErr      bool
		wantPolicy   string
		validateFunc func(any)
	}{
		{
			name:    "darwin: returns nil — no SELinux",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			validateFunc: func(got any) {
				s.Nil(got)
			},
		},
		{
			name:    "linux: no /etc/selinux/config — disabled",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			setupExec: func(ctrl *gomock.Controller) *execmocks.MockExecutor {
				return execmocks.NewMockExecutor(ctrl)
			},
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("disabled", info.Status)
				s.Equal("", info.CurrentMode)
				s.Equal("", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("disabled", info.Status)
				s.Equal("", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("33", info.MaxKernelPolicyVersion)
				s.Equal("33", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("permissive", info.CurrentMode)
				s.Equal("minimum", info.LoadedPolicyName)
				s.Equal("30", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("disabled", info.Status)
				s.Equal("", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
		},
		{
			name:    "linux: nil executor — falls back to config only",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: nil,
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("33", info.MaxKernelPolicyVersion)
				s.Equal("33", info.PolicyVersion)
			},
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
			validateFunc: func(got any) {
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
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

			tt.validateFunc(got)
		})
	}
}
