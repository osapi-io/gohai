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
	packagemgr "github.com/osapi-io/gohai/pkg/gohai/collectors/package_mgr"
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

func TestPackageMgrPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PackageMgrPublicTestSuite))
}

func (s *PackageMgrPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string // "linux"|"darwin"|"debian"|"rhel"
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Debian", "debian", "debian"},
		{"rhel dispatches to RHEL", "rhel", "rhel"},
		{"arch dispatches to Linux", "arch", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := packagemgr.New()
			s.Equal("package_mgr", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*packagemgr.Darwin)
				s.True(ok)
			case "debian":
				_, ok := c.(*packagemgr.Debian)
				s.True(ok)
			case "rhel":
				_, ok := c.(*packagemgr.RHEL)
				s.True(ok)
			case "linux":
				_, ok := c.(*packagemgr.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *PackageMgrPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string // "linux" | "darwin" | "debian" | "rhel"
		probed   map[string]string
		wantName string
		wantPath string
	}{
		{
			"debian with apt",
			"debian",
			map[string]string{"apt": "/usr/bin/apt"},
			"apt",
			"/usr/bin/apt",
		},
		{
			"debian with apt-get only",
			"debian",
			map[string]string{"apt-get": "/usr/bin/apt-get"},
			"apt-get",
			"/usr/bin/apt-get",
		},
		{
			"rhel with dnf wins over yum",
			"rhel",
			map[string]string{"dnf": "/usr/bin/dnf", "yum": "/usr/bin/yum"},
			"dnf",
			"/usr/bin/dnf",
		},
		{
			"rhel yum fallback",
			"rhel",
			map[string]string{"yum": "/usr/bin/yum"},
			"yum",
			"/usr/bin/yum",
		},
		{
			"darwin brew",
			"darwin",
			map[string]string{"brew": "/opt/homebrew/bin/brew"},
			"brew",
			"/opt/homebrew/bin/brew",
		},
		{
			"darwin port fallback",
			"darwin",
			map[string]string{"port": "/opt/local/bin/port"},
			"port",
			"/opt/local/bin/port",
		},
		{
			"linux arch with pacman",
			"linux",
			map[string]string{"pacman": "/usr/bin/pacman"},
			"pacman",
			"/usr/bin/pacman",
		},
		{
			"linux alpine with apk",
			"linux",
			map[string]string{"apk": "/sbin/apk"},
			"apk",
			"/sbin/apk",
		},
		{"none found returns empty", "linux", map[string]string{}, "", ""},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			probe := func(name string) string { return tt.probed[name] }
			defer packagemgr.SetProbeFn(probe)()
			var got any
			var err error
			switch tt.variant {
			case "debian":
				got, err = (&packagemgr.Debian{}).Collect(context.Background())
			case "rhel":
				got, err = (&packagemgr.RHEL{}).Collect(context.Background())
			case "darwin":
				got, err = (&packagemgr.Darwin{}).Collect(context.Background())
			case "linux":
				got, err = (&packagemgr.Linux{}).Collect(context.Background())
			}
			s.Require().NoError(err)
			info, ok := got.(*packagemgr.Info)
			s.Require().True(ok)
			s.Equal(tt.wantName, info.Name)
			s.Equal(tt.wantPath, info.Path)
		})
	}
}

func (s *PackageMgrPublicTestSuite) TestProbe() {
	tests := []struct {
		name    string
		binary  string
		wantAbs bool
	}{
		{"present binary resolves to absolute path", "sh", true},
		{"missing binary returns empty", "definitely-not-a-real-binary-xyz123", false},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := packagemgr.Probe(tt.binary)
			if tt.wantAbs {
				s.NotEmpty(got)
			} else {
				s.Empty(got)
			}
		})
	}
}
