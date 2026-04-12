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

package initd_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
)

var (
	_ collector.Collector = (*initd.Linux)(nil)
	_ collector.Collector = (*initd.Darwin)(nil)
)

type InitPublicTestSuite struct {
	suite.Suite
}

func TestInitPublicTestSuite(t *testing.T) {
	suite.Run(t, new(InitPublicTestSuite))
}

func (s *InitPublicTestSuite) TestNew() {
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
			c := initd.New()
			s.Equal("init", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*initd.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*initd.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *InitPublicTestSuite) TestCollectLinux() {
	tests := []struct {
		name string
		comm string
		err  error
		want initd.Info
	}{
		{"systemd host", "systemd\n", nil, initd.Info{Name: "systemd"}},
		{"upstart host", "upstart\n", nil, initd.Info{Name: "upstart"}},
		{"sysvinit (comm=init) normalized", "init\n", nil, initd.Info{Name: "sysvinit"}},
		{"openrc-init normalized to openrc", "openrc-init\n", nil, initd.Info{Name: "openrc"}},
		{"runit host", "runit\n", nil, initd.Info{Name: "runit"}},
		{"unknown comm passed through", "exoticinit\n", nil, initd.Info{Name: "exoticinit"}},
		{"read error soft-misses", "", errors.New("no /proc"), initd.Info{}},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &initd.Linux{
				ReadFileFn: func(string) ([]byte, error) {
					if tt.err != nil {
						return nil, tt.err
					}
					return []byte(tt.comm), nil
				},
			}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*initd.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *InitPublicTestSuite) TestCollectDarwin() {
	c := initd.NewDarwin()
	got, err := c.Collect(context.Background())
	s.Require().NoError(err)
	info, ok := got.(*initd.Info)
	s.Require().True(ok)
	s.Equal("launchd", info.Name)
}
