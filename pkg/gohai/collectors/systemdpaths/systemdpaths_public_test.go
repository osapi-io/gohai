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
	"fmt"
	osexec "os/exec"
	"strings"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/systemdpaths"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*systemdpaths.Linux)(nil)
	_ collector.Collector = (*systemdpaths.Darwin)(nil)
)

// The command lines that the rows answer, as sourcetest.Commands keys them.
const (
	systemdPathLine = "systemd-path"
)

type SystemdPathsPublicTestSuite struct {
	suite.Suite
}

func TestSystemdPathsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(SystemdPathsPublicTestSuite))
}

func (s *SystemdPathsPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
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
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c systemdpaths.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := systemdpaths.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = systemdpaths.New()
			}
			s.Equal("systemd_paths", c.Name())
			s.Equal("linux", string(c.Category()))
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
		exec         func(*testing.T) source.Runner
		validateFunc func(any, error)
	}{
		{
			name:    "linux: a line over 64 KiB keeps the paths after it",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						systemdPathLine: "long: /" + strings.Repeat(
							"x",
							70000,
						) + "\nsystemd: /usr/lib/systemd\n",
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal("/usr/lib/systemd", info.Paths["systemd"])
			},
		},
		{
			name:    "linux: full output parsed",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{systemdPathLine: string(fullOutput)},
				}.Mock(
					t,
				)
			},
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{systemdPathLine: ""}}.Mock(t)
			},
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						systemdPathLine: "no_separator\nsystemd: /usr/lib/systemd\n",
					},
				}.Mock(
					t,
				)
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						systemdPathLine: ": /some/path\nsystemd: /usr/lib/systemd\n",
					},
				}.Mock(
					t,
				)
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
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						systemdPathLine: fmt.Errorf("exec: %w", osexec.ErrNotFound),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Paths)
			},
		},
		{
			name:    "linux: a failed run is an error",
			variant: "linux",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						systemdPathLine: errors.New("exit status 1: permission denied"),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: nil Exec yields empty paths, no error",
			variant: "linux",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*systemdpaths.Info)
				s.Require().True(ok)
				s.Equal(map[string]string{}, info.Paths)
			},
		},
		{
			name:    "darwin: unsupported",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
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
