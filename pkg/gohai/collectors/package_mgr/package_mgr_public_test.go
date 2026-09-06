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
			s.Equal("software", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *PackageMgrPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string // "linux" | "darwin" | "debian" | "rhel"
		probed       map[string]string
		wantName     string
		validateFunc func(string)
	}{
		{
			name:     "debian with apt",
			variant:  "debian",
			probed:   map[string]string{"apt": "/usr/bin/apt"},
			wantName: "apt",
			validateFunc: func(got string) {
				s.Equal("/usr/bin/apt", got)
			},
		},
		{
			name:     "debian with apt-get only",
			variant:  "debian",
			probed:   map[string]string{"apt-get": "/usr/bin/apt-get"},
			wantName: "apt-get",
			validateFunc: func(got string) {
				s.Equal("/usr/bin/apt-get", got)
			},
		},
		{
			name:     "rhel with dnf wins over yum",
			variant:  "rhel",
			probed:   map[string]string{"dnf": "/usr/bin/dnf", "yum": "/usr/bin/yum"},
			wantName: "dnf",
			validateFunc: func(got string) {
				s.Equal("/usr/bin/dnf", got)
			},
		},
		{
			name:     "rhel yum fallback",
			variant:  "rhel",
			probed:   map[string]string{"yum": "/usr/bin/yum"},
			wantName: "yum",
			validateFunc: func(got string) {
				s.Equal("/usr/bin/yum", got)
			},
		},
		{
			name:     "darwin brew",
			variant:  "darwin",
			probed:   map[string]string{"brew": "/opt/homebrew/bin/brew"},
			wantName: "brew",
			validateFunc: func(got string) {
				s.Equal("/opt/homebrew/bin/brew", got)
			},
		},
		{
			name:     "darwin port fallback",
			variant:  "darwin",
			probed:   map[string]string{"port": "/opt/local/bin/port"},
			wantName: "port",
			validateFunc: func(got string) {
				s.Equal("/opt/local/bin/port", got)
			},
		},
		{
			name:     "linux arch with pacman",
			variant:  "linux",
			probed:   map[string]string{"pacman": "/usr/bin/pacman"},
			wantName: "pacman",
			validateFunc: func(got string) {
				s.Equal("/usr/bin/pacman", got)
			},
		},
		{
			name:     "linux alpine with apk",
			variant:  "linux",
			probed:   map[string]string{"apk": "/sbin/apk"},
			wantName: "apk",
			validateFunc: func(got string) {
				s.Equal("/sbin/apk", got)
			},
		},
		{
			name:     "none found returns empty",
			variant:  "linux",
			probed:   map[string]string{},
			wantName: "",
			validateFunc: func(got string) {
				s.Equal("", got)
			},
		},
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
			case "debian":
				got, err = (&packagemgr.Debian{}).Collect(context.Background(), nil)
			case "rhel":
				got, err = (&packagemgr.RHEL{}).Collect(context.Background(), nil)
			case "darwin":
				got, err = (&packagemgr.Darwin{}).Collect(context.Background(), nil)
			case "linux":
				got, err = (&packagemgr.Linux{}).Collect(context.Background(), nil)
			}
			s.Require().NoError(err)
			info, ok := got.(*packagemgr.Info)
			s.Require().True(ok)
			s.Equal(tt.wantName, info.Name)
			tt.validateFunc(info.Path)
		})
	}
}
