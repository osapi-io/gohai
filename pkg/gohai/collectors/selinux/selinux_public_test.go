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
	"fmt"
	osexec "os/exec"
	"strings"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/selinux"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*selinux.Linux)(nil)
	_ collector.Collector = (*selinux.Darwin)(nil)
)

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

func TestSelinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SelinuxPublicTestSuite))
}

func (s *SelinuxPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
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
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c selinux.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := selinux.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = selinux.New()
			}
			s.Equal("selinux", c.Name())
			s.Equal("security", string(c.Category()))
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
		setupExec    func(ctrl *gomock.Controller) *mocks.MockRunner
		validateFunc func(any, error)
	}{
		{
			name:    "darwin: unsupported",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: with no /etc/selinux/config SELinux is disabled",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				return mocks.NewMockRunner(ctrl)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
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
			name:    "linux: SELINUX=disabled in the config is disabled, and sestatus does not run",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/selinux/config": "# comment\nSELINUX=disabled\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				return mocks.NewMockRunner(ctrl)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
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
			name:    "linux: a config line over 64 KiB keeps the config lines after it",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config": "# " + strings.Repeat(
						"x",
						70000,
					) + "\nSELINUX=permissive\nSELINUXTYPE=mls\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return(nil, fmt.Errorf("exec sestatus: %w", osexec.ErrNotFound))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("permissive", info.ConfigMode)
				s.Equal("mls", info.LoadedPolicyName)
			},
		},
		{
			name:    "linux: a line over 64 KiB keeps the config and sestatus lines after it",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config": "# " + strings.Repeat(
						"x",
						70000,
					) + "\nSELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte("Note: "+strings.Repeat("x", 70000)+"\n"+sestatusEnforcing), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("33", info.PolicyVersion)
			},
		},
		{
			name:    "linux: enforcing with sestatus",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config":     "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusEnforcing), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
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
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "0\n",
					"/etc/selinux/config":     "SELINUX=permissive\nSELINUXTYPE=minimum\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusPermissive), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
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
			name:    "linux: no sestatus: the kernel gives the status and the mode",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config":     "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return(nil, fmt.Errorf("sestatus: %w", osexec.ErrNotFound))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
		},
		{
			name:    "linux: the kernel decides over a sestatus that says disabled",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "0\n",
					"/etc/selinux/config":     "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusDisabled), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("permissive", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
		},
		{
			name:    "linux: with a nil executor only the config is read",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config":     "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: nil,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
				s.Equal("targeted", info.LoadedPolicyName)
				s.Equal("", info.MaxKernelPolicyVersion)
				s.Equal("", info.PolicyVersion)
			},
		},
		{
			name:    "linux: config says enforcing, the kernel has no selinuxfs: disabled",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/etc/selinux/config": "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				return mocks.NewMockRunner(ctrl)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("disabled", info.Status)
				s.Equal("enforcing", info.ConfigMode)
				s.Equal("", info.CurrentMode)
			},
		},
		{
			name:    "linux: no config file, the kernel has SELinux on: enabled",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{"/sys/fs/selinux/enforce": "1\n"})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusEnforcing), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
				s.Equal("", info.ConfigMode)
			},
		},
		{
			name:    "linux: a sestatus that fails keeps the kernel and config facts",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{"/sys/fs/selinux/enforce": "1\n"})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return(nil, errors.New("signal: killed"))
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "sestatus: signal: killed")
				info, ok := got.(*selinux.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("enabled", info.Status)
				s.Equal("enforcing", info.CurrentMode)
			},
		},
		{
			name:    "linux: an enforce file that cannot be read is an error",
			variant: "linux",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/sys/fs/selinux/enforce", 0o755))
				return f
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				return mocks.NewMockRunner(ctrl)
			},
			validateFunc: func(got any, err error) {
				s.Require().ErrorContains(err, "read /sys/fs/selinux/enforce")
				s.Exactly(nil, got)
			},
		},
		{
			// config line with no '=' separator exercises the !ok branch
			// in parseConfigFile.
			name:    "linux: config line without equals sign skipped",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config":     "SELINUX=enforcing\nthis line has no equals\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(sestatusEnforcing), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
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
				return sourcetest.FS(s.T(), map[string]string{
					"/sys/fs/selinux/enforce": "1\n",
					"/etc/selinux/config":     "SELINUX=enforcing\nSELINUXTYPE=targeted\n",
				})
			},
			setupExec: func(ctrl *gomock.Controller) *mocks.MockRunner {
				m := mocks.NewMockRunner(ctrl)
				// Include a line with no colon to exercise the !ok branch.
				out := "SELinux status: enabled\nno colon here\nCurrent mode: enforcing\n"
				m.EXPECT().
					Execute(gomock.Any(), "sestatus").
					Return([]byte(out), nil)
				return m
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
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

			tt.validateFunc(got, err)
		})
	}
}
