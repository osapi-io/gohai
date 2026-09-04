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
	"errors"
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

func TestPackageMgrPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackageMgrPublicTestSuite))
}

func (s *PackageMgrPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string // osLinux|osDarwin|platformDebian|platformRHEL
	}{
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Debian", platformDebian, platformDebian},
		{"rhel dispatches to RHEL", platformRHEL, platformRHEL},
		{"arch dispatches to Linux", "arch", osLinux},
		{"unknown dispatches to Linux", "", osLinux},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := packagemgr.New()
			s.Equal("package_mgr", c.Name())
			s.Equal("software", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*packagemgr.Darwin)
				s.True(ok)
			case platformDebian:
				_, ok := c.(*packagemgr.Debian)
				s.True(ok)
			case platformRHEL:
				_, ok := c.(*packagemgr.RHEL)
				s.True(ok)
			case osLinux:
				_, ok := c.(*packagemgr.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *PackageMgrPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string // osLinux | osDarwin | platformDebian | platformRHEL
		probed   map[string]string
		wantName string
		wantPath string
	}{
		{
			"debian with apt",
			platformDebian,
			map[string]string{"apt": "/usr/bin/apt"},
			"apt",
			"/usr/bin/apt",
		},
		{
			"debian with apt-get only",
			platformDebian,
			map[string]string{"apt-get": "/usr/bin/apt-get"},
			"apt-get",
			"/usr/bin/apt-get",
		},
		{
			"rhel with dnf wins over yum",
			platformRHEL,
			map[string]string{"dnf": "/usr/bin/dnf", "yum": "/usr/bin/yum"},
			"dnf",
			"/usr/bin/dnf",
		},
		{
			"rhel yum fallback",
			platformRHEL,
			map[string]string{"yum": "/usr/bin/yum"},
			"yum",
			"/usr/bin/yum",
		},
		{
			"darwin brew",
			osDarwin,
			map[string]string{"brew": "/opt/homebrew/bin/brew"},
			"brew",
			"/opt/homebrew/bin/brew",
		},
		{
			"darwin port fallback",
			osDarwin,
			map[string]string{"port": "/opt/local/bin/port"},
			"port",
			"/opt/local/bin/port",
		},
		{
			"linux arch with pacman",
			osLinux,
			map[string]string{"pacman": "/usr/bin/pacman"},
			"pacman",
			"/usr/bin/pacman",
		},
		{
			"linux alpine with apk",
			osLinux,
			map[string]string{"apk": "/sbin/apk"},
			"apk",
			"/sbin/apk",
		},
		{"none found returns empty", osLinux, map[string]string{}, "", ""},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer packagemgr.SetLookPathFn(func(name string) (string, error) {
				if p, ok := tt.probed[name]; ok {
					return p, nil
				}
				return "", errors.New("not found")
			})()
			var got any
			var err error
			switch tt.variant {
			case platformDebian:
				got, err = (&packagemgr.Debian{}).Collect(context.Background(), nil)
			case platformRHEL:
				got, err = (&packagemgr.RHEL{}).Collect(context.Background(), nil)
			case osDarwin:
				got, err = (&packagemgr.Darwin{}).Collect(context.Background(), nil)
			case osLinux:
				got, err = (&packagemgr.Linux{}).Collect(context.Background(), nil)
			default:
				s.Failf("unhandled case", "%v", tt.variant)
			}
			s.Require().NoError(err)
			info, ok := got.(*packagemgr.Info)
			s.Require().True(ok)
			s.Equal(tt.wantName, info.Name)
			s.Equal(tt.wantPath, info.Path)
		})
	}
}
