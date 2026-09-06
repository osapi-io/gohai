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

package systemdpaths_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/mocks"
	"github.com/osapi-io/gohai/internal/platform"
	systemdpaths "github.com/osapi-io/gohai/pkg/gohai/collectors/systemd_paths"
)

var (
	_ collector.Collector = (*systemdpaths.Linux)(nil)
	_ collector.Collector = (*systemdpaths.Darwin)(nil)
)

type SystemdPathsPublicTestSuite struct {
	suite.Suite
}

func TestSystemdPathsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(SystemdPathsPublicTestSuite))
}

// systemdPathExec returns a MockExecutor that answers `systemd-path`
// with the provided output and error.
func systemdPathExec(
	t *testing.T,
	out []byte,
	err error,
) executor.Executor {
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
	m.EXPECT().
		Execute(gomock.Any(), "systemd-path").
		Return(out, err).
		AnyTimes()
	return m
}

func (s *SystemdPathsPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(systemdpaths.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c systemdpaths.Collector) {
				_, ok := c.(*systemdpaths.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c systemdpaths.Collector) {
				_, ok := c.(*systemdpaths.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c systemdpaths.Collector) {
				_, ok := c.(*systemdpaths.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c systemdpaths.Collector) {
				_, ok := c.(*systemdpaths.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c systemdpaths.Collector) {
				_, ok := c.(*systemdpaths.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := systemdpaths.New()
			s.Equal("systemd_paths", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *SystemdPathsPublicTestSuite) TestCollect() {
	fullOutput := []byte(`systemd: /usr/lib/systemd
systemd-search-system-unit: /etc/systemd/system.control
systemd-system-unit: /etc/systemd/system
user-configuration: /home/user/.config
user-runtime: /run/user/1000
`)

	tests := []struct {
		name         string
		variant      string
		exec         func(*testing.T) executor.Executor
		validateFunc func(any, error)
	}{
		{
			name:    "linux: full output parsed",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return systemdPathExec(t, fullOutput, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{
					"systemd":                    "/usr/lib/systemd",
					"systemd-search-system-unit": "/etc/systemd/system.control",
					"systemd-system-unit":        "/etc/systemd/system",
					"user-configuration":         "/home/user/.config",
					"user-runtime":               "/run/user/1000",
				}, info.Paths)
			},
		},
		{
			name:    "linux: empty output yields empty paths",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return systemdPathExec(t, []byte{}, nil) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Paths)
			},
		},
		{
			name:    "linux: line without colon-space skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return systemdPathExec(t,
					[]byte("no_separator\nsystemd: /usr/lib/systemd\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"systemd": "/usr/lib/systemd"}, info.Paths)
			},
		},
		{
			name:    "linux: empty key skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return systemdPathExec(t,
					[]byte(": /some/path\nsystemd: /usr/lib/systemd\n"),
					nil)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{"systemd": "/usr/lib/systemd"}, info.Paths)
			},
		},
		{
			name:    "linux: exec error yields empty paths, no error",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return systemdPathExec(t, nil, errors.New("not found")) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Paths)
			},
		},
		{
			name:    "linux: nil Exec yields empty paths, no error",
			variant: "linux",
			exec:    func(*testing.T) executor.Executor { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Paths)
			},
		},
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				if true {
					s.Nil(got)
					return
				}
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string(nil), info.Paths)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c systemdpaths.Collector
			switch tt.variant {
			case "linux":
				c = &systemdpaths.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = systemdpaths.NewDarwin()
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
