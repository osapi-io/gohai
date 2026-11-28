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

package packages_test

import (
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
	"github.com/osapi-io/gohai/pkg/gohai/source"
	"github.com/osapi-io/gohai/pkg/gohai/source/mocks"
)

var (
	_ collector.Collector = (*packages.Linux)(nil)
	_ collector.Collector = (*packages.Darwin)(nil)
)

type PackagesPublicTestSuite struct {
	suite.Suite
}

func TestPackagesPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(PackagesPublicTestSuite))
}

func (s *PackagesPublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(packages.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c packages.Collector) {
				_, ok := c.(*packages.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("dpkg", l.Manager)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("rpm", l.Manager)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("", l.Manager)
			},
		},
		{
			name:   "fedora dispatches to Linux with rpm",
			detect: "fedora",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("rpm", l.Manager)
			},
		},
		{
			name:   "suse dispatches to Linux with rpm",
			detect: "suse",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("rpm", l.Manager)
			},
		},
		{
			name:   "amazon dispatches to Linux with rpm",
			detect: "amazon",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("rpm", l.Manager)
			},
		},
		{
			name:   "alpine dispatches to Linux with no database",
			detect: "alpine",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("", l.Manager)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c packages.Collector) {
				l, ok := c.(*packages.Linux)
				s.True(ok)
				s.Equal("", l.Manager)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c packages.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := packages.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = packages.New()
			}
			s.Equal("packages", c.Name())
			s.Equal("software", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *PackagesPublicTestSuite) TestCollect() {
	// The command lines of the package databases, as Commands keys them.
	const (
		dpkgLine = "dpkg-query -W -f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n"
		rpmLine  = "rpm -qa --qf %{NAME}\\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n"
		brewLine = "brew list --versions"
	)
	dpkgOut := []byte(
		"bash\t5.1-6\tamd64\tinstalled\tGNU Bourne Again SHell\t7164\nlibc6\t2.35-0\tamd64\tinstalled\n",
	)
	dpkgWithSkipped := []byte(
		"bash\t5.1-6\tamd64\tinstalled\npartial-pkg\t1.0\tamd64\thalf-installed\n\t\t\t\n",
	)
	rpmOut := []byte(
		"bash\t5.1-6.fc36\tx86_64\tThe GNU Bourne Again shell\t8204000\nlibc\t2.35-1.fc36\tx86_64\n",
	)
	brewOut := []byte("git 2.40.0\nzsh 5.9\ncurl 8.1.2 8.1.1\n")

	tests := []struct {
		name    string
		variant string
		detect  string
		exec    func(*testing.T) source.Runner
		// noManager runs the Linux variant with no package manager.
		noManager    bool
		validateFunc func(any, error)
	}{
		{
			name:    "linux arch: no package database, an empty list",
			variant: "linux",
			detect:  "arch",
			exec: func(t *testing.T) source.Runner {
				return mocks.NewMockRunner(gomock.NewController(t))
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "linux debian: dpkg-query returns packages",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{dpkgLine: string(dpkgOut)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "bash",
						Version:        "5.1-6",
						Architecture:   "amd64",
						PackageManager: "dpkg",
						Description:    "GNU Bourne Again SHell",
						Size:           7164 * 1024,
					},
					{
						Name:           "libc6",
						Version:        "2.35-0",
						Architecture:   "amd64",
						PackageManager: "dpkg",
					},
				}, info.Packages)
			},
		},
		{
			name:    "linux debian: dpkg skips non-installed and malformed lines",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{dpkgLine: string(dpkgWithSkipped)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{Name: "bash", Version: "5.1-6", Architecture: "amd64", PackageManager: "dpkg"},
				}, info.Packages)
			},
		},
		{
			name:    "linux debian: dpkg skips short lines",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{dpkgLine: string([]byte("bash\t5.1\n"))},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "linux debian: dpkg skips empty-name entries",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						dpkgLine: string(
							[]byte("\t1.0\tamd64\tinstalled\nbash\t5.1\tamd64\tinstalled\n"),
						),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{Name: "bash", Version: "5.1", Architecture: "amd64", PackageManager: "dpkg"},
				}, info.Packages)
			},
		},
		{
			name:    "linux debian: dpkg-query fails, empty list",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{dpkgLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "linux debian: a failed run is an error",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{dpkgLine: errors.New("exit status 1: permission denied")},
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
			name:    "linux rhel: rpm returns packages",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{rpmLine: string(rpmOut)}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "bash",
						Version:        "5.1-6.fc36",
						Architecture:   "x86_64",
						PackageManager: "rpm",
						Description:    "The GNU Bourne Again shell",
						Size:           8204000,
					},
					{
						Name:           "libc",
						Version:        "2.35-1.fc36",
						Architecture:   "x86_64",
						PackageManager: "rpm",
					},
				}, info.Packages)
			},
		},
		{
			name:    "linux rhel: rpm fails, empty list",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{rpmLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "linux rhel: a failed run is an error",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{rpmLine: errors.New("exit status 1: permission denied")},
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
			name:    "linux rhel: rpm skips short lines",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{rpmLine: string([]byte("bash\n"))},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "linux rhel: rpm skips empty-name entries",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						rpmLine: string([]byte("\t1.0-1.fc36\tx86_64\nbash\t5.1-6.fc36\tx86_64\n")),
					},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "bash",
						Version:        "5.1-6.fc36",
						Architecture:   "x86_64",
						PackageManager: "rpm",
					},
				}, info.Packages)
			},
		},
		{
			name:    "linux: nil Exec returns empty list",
			variant: "linux",
			detect:  "debian",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "darwin: brew list returns packages",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{brewLine: string(brewOut)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{Name: "git", Version: "2.40.0", PackageManager: "brew"},
					{Name: "zsh", Version: "5.9", PackageManager: "brew"},
					{Name: "curl", Version: "8.1.1", PackageManager: "brew"},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: brew output with blank lines skipped",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{brewLine: string([]byte("\ngit 2.40.0\n\n"))},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{Name: "git", Version: "2.40.0", PackageManager: "brew"},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: brew output with only-name line skipped",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{brewLine: string([]byte("git\n"))},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "darwin: brew fails, empty list",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{brewLine: fmt.Errorf("exec: %w", osexec.ErrNotFound)},
				}.Mock(
					t,
				)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:    "darwin: a failed run is an error",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{brewLine: errors.New("exit status 1: permission denied")},
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
			name:    "darwin: nil Exec returns empty list",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
		{
			name:      "linux: with no package manager the list is empty",
			variant:   "linux",
			detect:    "debian",
			exec:      func(*testing.T) source.Runner { return source.Commands() },
			noManager: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{}, info.Packages)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c packages.Collector
			switch tt.variant {
			case "linux":
				src := source.Sources{Family: tt.detect, FS: memfs.New()}
				l, ok := packages.NewWith(src).(*packages.Linux)
				s.Require().True(ok)
				l.Exec = tt.exec(s.T())
				if tt.noManager {
					l.Manager = ""
				}
				c = l
			case "darwin":
				c = &packages.Darwin{Exec: tt.exec(s.T())}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
