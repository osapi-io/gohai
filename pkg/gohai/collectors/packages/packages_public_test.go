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

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
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

func mockExec(
	t *testing.T,
	cmd string,
	args []string,
	out []byte,
	execErr error,
) source.Runner {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockRunner(ctrl)
	extraArgs := make([]any, len(args))
	for i, a := range args {
		extraArgs[i] = a
	}
	m.EXPECT().Execute(gomock.Any(), cmd, extraArgs...).Return(out, execErr).AnyTimes()
	return m
}

func (s *PackagesPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
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
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := packages.New()
			s.Equal("packages", c.Name())
			s.Equal("software", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *PackagesPublicTestSuite) TestCollect() {
	origDetect := platform.Detect
	defer func() { platform.Detect = origDetect }()

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
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n",
					},
					dpkgOut,
					nil,
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
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n",
					},
					dpkgWithSkipped,
					nil,
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
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n",
					},
					[]byte("bash\t5.1\n"),
					nil,
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
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n",
					},
					[]byte("\t1.0\tamd64\tinstalled\nbash\t5.1\tamd64\tinstalled\n"),
					nil,
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
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n",
					},
					nil,
					fmt.Errorf("exec: %w", osexec.ErrNotFound),
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
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\t${binary:Summary}\\t${Installed-Size}\\n",
					},
					nil,
					errors.New("exit status 1: permission denied"),
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
				return mockExec(
					t,
					"rpm",
					[]string{
						"-qa",
						"--qf",
						"%{NAME}\\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n",
					},
					rpmOut,
					nil,
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
				return mockExec(
					t,
					"rpm",
					[]string{
						"-qa",
						"--qf",
						"%{NAME}\\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n",
					},
					nil,
					fmt.Errorf("exec: %w", osexec.ErrNotFound),
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
				return mockExec(
					t,
					"rpm",
					[]string{
						"-qa",
						"--qf",
						"%{NAME}\\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n",
					},
					nil,
					errors.New("exit status 1: permission denied"),
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
				return mockExec(
					t,
					"rpm",
					[]string{
						"-qa",
						"--qf",
						"%{NAME}\\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n",
					},
					[]byte("bash\n"),
					nil,
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
				return mockExec(
					t,
					"rpm",
					[]string{
						"-qa",
						"--qf",
						"%{NAME}\\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\t%{ARCH}\\t%{SUMMARY}\\t%{SIZE}\\n",
					},
					[]byte("\t1.0-1.fc36\tx86_64\nbash\t5.1-6.fc36\tx86_64\n"),
					nil,
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
				return mockExec(t, "brew", []string{"list", "--versions"}, brewOut, nil)
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
				return mockExec(
					t,
					"brew",
					[]string{"list", "--versions"},
					[]byte("\ngit 2.40.0\n\n"),
					nil,
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
				return mockExec(t, "brew", []string{"list", "--versions"}, []byte("git\n"), nil)
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
				return mockExec(
					t,
					"brew",
					[]string{"list", "--versions"},
					nil,
					fmt.Errorf("exec: %w", osexec.ErrNotFound),
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
				return mockExec(
					t,
					"brew",
					[]string{"list", "--versions"},
					nil,
					errors.New("exit status 1: permission denied"),
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
			if tt.detect != "" {
				platform.Detect = func() string { return tt.detect }
			} else {
				platform.Detect = origDetect
			}
			var c packages.Collector
			switch tt.variant {
			case "linux":
				l, ok := packages.New().(*packages.Linux)
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
