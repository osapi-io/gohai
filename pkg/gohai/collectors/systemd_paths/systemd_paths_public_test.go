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
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
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
			c := systemdpaths.New()
			s.Equal("systemd_paths", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*systemdpaths.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*systemdpaths.Linux)
				s.True(ok)
			}
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
		name      string
		variant   string
		exec      func(*testing.T) executor.Executor
		wantNil   bool
		wantPaths map[string]string
	}{
		{
			name:    "linux: full output parsed",
			variant: "linux",
			exec:    func(t *testing.T) executor.Executor { return systemdPathExec(t, fullOutput, nil) },
			wantPaths: map[string]string{
				"systemd":                    "/usr/lib/systemd",
				"systemd-search-system-unit": "/etc/systemd/system.control",
				"systemd-system-unit":        "/etc/systemd/system",
				"user-configuration":         "/home/user/.config",
				"user-runtime":               "/run/user/1000",
			},
		},
		{
			name:      "linux: empty output yields empty paths",
			variant:   "linux",
			exec:      func(t *testing.T) executor.Executor { return systemdPathExec(t, []byte{}, nil) },
			wantPaths: map[string]string{},
		},
		{
			name:    "linux: line without colon-space skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return systemdPathExec(t,
					[]byte("no_separator\nsystemd: /usr/lib/systemd\n"),
					nil)
			},
			wantPaths: map[string]string{"systemd": "/usr/lib/systemd"},
		},
		{
			name:    "linux: empty key skipped",
			variant: "linux",
			exec: func(t *testing.T) executor.Executor {
				return systemdPathExec(t,
					[]byte(": /some/path\nsystemd: /usr/lib/systemd\n"),
					nil)
			},
			wantPaths: map[string]string{"systemd": "/usr/lib/systemd"},
		},
		{
			name:      "linux: exec error yields empty paths, no error",
			variant:   "linux",
			exec:      func(t *testing.T) executor.Executor { return systemdPathExec(t, nil, errors.New("not found")) },
			wantPaths: map[string]string{},
		},
		{
			name:      "linux: nil Exec yields empty paths, no error",
			variant:   "linux",
			exec:      func(*testing.T) executor.Executor { return nil },
			wantPaths: map[string]string{},
		},
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			wantNil: true,
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
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*systemdpaths.Info)
			s.Require().True(ok)
			s.Equal(tt.wantPaths, info.Paths)
		})
	}
}
