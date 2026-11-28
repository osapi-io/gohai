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

package packagemgr_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packagemgr"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*packagemgr.Linux)(nil)
	_ collector.Collector = (*packagemgr.Darwin)(nil)
	_ collector.Collector = (*packagemgr.Debian)(nil)
	_ collector.Collector = (*packagemgr.RHEL)(nil)
)

type PackageMgrPublicTestSuite struct {
	suite.Suite
}

func TestPackageMgrPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageMgrPublicTestSuite))
}

func (s *PackageMgrPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
		validateFunc func(packagemgr.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c packagemgr.Collector) {
				_, ok := c.(*packagemgr.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Debian",
			detect: "debian",
			validateFunc: func(c packagemgr.Collector) {
				_, ok := c.(*packagemgr.Debian)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to RHEL",
			detect: "rhel",
			validateFunc: func(c packagemgr.Collector) {
				_, ok := c.(*packagemgr.RHEL)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c packagemgr.Collector) {
				_, ok := c.(*packagemgr.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c packagemgr.Collector) {
				_, ok := c.(*packagemgr.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := packagemgr.New()
			s.Equal("package_mgr", c.Name())
			s.Equal("software", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *PackageMgrPublicTestSuite) TestCollect() {
	found := func(paths map[string]string) sourcetest.Commands {
		out := map[string]string{}
		for name, p := range paths {
			out["sh -c command -v "+name] = p + "\n"
		}
		return sourcetest.Commands{Out: out}
	}
	tests := []struct {
		name     string
		c        func(source.Runner) packagemgr.Collector
		runner   sourcetest.Commands
		nilExec  bool
		wantName string
		wantPath string
	}{
		{
			name: "debian: apt",
			c:    func(e source.Runner) packagemgr.Collector { return &packagemgr.Debian{Exec: e} },
			runner: found(
				map[string]string{"apt": "/usr/bin/apt", "apt-get": "/usr/bin/apt-get"},
			),
			wantName: "apt",
			wantPath: "/usr/bin/apt",
		},
		{
			name:     "debian: apt-get when apt is missing",
			c:        func(e source.Runner) packagemgr.Collector { return &packagemgr.Debian{Exec: e} },
			runner:   found(map[string]string{"apt-get": "/usr/bin/apt-get"}),
			wantName: "apt-get",
			wantPath: "/usr/bin/apt-get",
		},
		{
			name:     "rhel: dnf",
			c:        func(e source.Runner) packagemgr.Collector { return &packagemgr.RHEL{Exec: e} },
			runner:   found(map[string]string{"dnf": "/usr/bin/dnf"}),
			wantName: "dnf",
			wantPath: "/usr/bin/dnf",
		},
		{
			name:     "linux: apk",
			c:        func(e source.Runner) packagemgr.Collector { return &packagemgr.Linux{Exec: e} },
			runner:   found(map[string]string{"apk": "/sbin/apk"}),
			wantName: "apk",
			wantPath: "/sbin/apk",
		},
		{
			name:     "darwin: brew on the PATH",
			c:        func(e source.Runner) packagemgr.Collector { return &packagemgr.Darwin{Exec: e} },
			runner:   found(map[string]string{"brew": "/opt/homebrew/bin/brew"}),
			wantName: "brew",
			wantPath: "/opt/homebrew/bin/brew",
		},
		{
			name: "darwin: an alias is not a path",
			c:    func(e source.Runner) packagemgr.Collector { return &packagemgr.Darwin{Exec: e} },
			runner: sourcetest.Commands{Out: map[string]string{
				"sh -c command -v brew": "alias brew='echo'\n",
			}},
		},
		{
			name: "linux: none found",
			c:    func(e source.Runner) packagemgr.Collector { return &packagemgr.Linux{Exec: e} },
		},
		{
			name:    "linux: nil Exec",
			c:       func(e source.Runner) packagemgr.Collector { return &packagemgr.Linux{Exec: e} },
			nilExec: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var exec source.Runner = tt.runner.Mock(s.T())
			if tt.nilExec {
				exec = nil
			}
			got, err := tt.c(exec).Collect(context.Background(), nil)
			s.Require().NoError(err)
			s.Equal(&packagemgr.Info{Name: tt.wantName, Path: tt.wantPath}, got)
		})
	}
}
