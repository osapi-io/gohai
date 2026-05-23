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
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
	execmocks "github.com/osapi-io/gohai/internal/executor/gen"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/packages"
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
) executor.Executor {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := execmocks.NewMockExecutor(ctrl)
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
			c := packages.New()
			s.Equal("packages", c.Name())
			s.Equal("software", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*packages.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*packages.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *PackagesPublicTestSuite) TestCollect() {
	origDetect := platform.Detect
	defer func() { platform.Detect = origDetect }()

	dpkgOut := []byte("bash\t5.1-6\tamd64\tinstalled\nlibc6\t2.35-0\tamd64\tinstalled\n")
	dpkgWithSkipped := []byte(
		"bash\t5.1-6\tamd64\tinstalled\npartial-pkg\t1.0\tamd64\thalf-installed\n\t\t\t\n",
	)
	rpmOut := []byte("bash\t5.1-6.fc36\tx86_64\nlibc\t2.35-1.fc36\tx86_64\n")
	brewOut := []byte("git 2.40.0\nzsh 5.9\ncurl 8.1.2 8.1.1\n")

	tests := []struct {
		name    string
		variant string
		detect  string
		exec    func(*testing.T) executor.Executor
		want    []packages.Package
	}{
		{
			name:    "linux debian: dpkg-query returns packages",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\n",
					},
					dpkgOut,
					nil,
				)
			},
			want: []packages.Package{
				{Name: "bash", Version: "5.1-6", Arch: "amd64", Source: "dpkg"},
				{Name: "libc6", Version: "2.35-0", Arch: "amd64", Source: "dpkg"},
			},
		},
		{
			name:    "linux debian: dpkg skips non-installed and malformed lines",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\n",
					},
					dpkgWithSkipped,
					nil,
				)
			},
			want: []packages.Package{
				{Name: "bash", Version: "5.1-6", Arch: "amd64", Source: "dpkg"},
			},
		},
		{
			name:    "linux debian: dpkg skips short lines",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\n",
					},
					[]byte("bash\t5.1\n"),
					nil,
				)
			},
			want: []packages.Package{},
		},
		{
			name:    "linux debian: dpkg skips empty-name entries",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\n",
					},
					[]byte("\t1.0\tamd64\tinstalled\nbash\t5.1\tamd64\tinstalled\n"),
					nil,
				)
			},
			want: []packages.Package{
				{Name: "bash", Version: "5.1", Arch: "amd64", Source: "dpkg"},
			},
		},
		{
			name:    "linux debian: dpkg-query fails, empty list",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"dpkg-query",
					[]string{
						"-W",
						"-f=${Package}\\t${Version}\\t${Architecture}\\t${db:Status-Status}\\n",
					},
					nil,
					errors.New("not found"),
				)
			},
			want: []packages.Package{},
		},
		{
			name:    "linux rhel: rpm returns packages",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"rpm",
					[]string{"-qa", "--qf", "%{NAME}\\t%{VERSION}-%{RELEASE}\\t%{ARCH}\\n"},
					rpmOut,
					nil,
				)
			},
			want: []packages.Package{
				{Name: "bash", Version: "5.1-6.fc36", Arch: "x86_64", Source: "rpm"},
				{Name: "libc", Version: "2.35-1.fc36", Arch: "x86_64", Source: "rpm"},
			},
		},
		{
			name:    "linux rhel: rpm fails, empty list",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"rpm",
					[]string{"-qa", "--qf", "%{NAME}\\t%{VERSION}-%{RELEASE}\\t%{ARCH}\\n"},
					nil,
					errors.New("not found"),
				)
			},
			want: []packages.Package{},
		},
		{
			name:    "linux rhel: rpm skips short lines",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"rpm",
					[]string{"-qa", "--qf", "%{NAME}\\t%{VERSION}-%{RELEASE}\\t%{ARCH}\\n"},
					[]byte("bash\n"),
					nil,
				)
			},
			want: []packages.Package{},
		},
		{
			name:    "linux rhel: rpm skips empty-name entries",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"rpm",
					[]string{"-qa", "--qf", "%{NAME}\\t%{VERSION}-%{RELEASE}\\t%{ARCH}\\n"},
					[]byte("\t1.0-1.fc36\tx86_64\nbash\t5.1-6.fc36\tx86_64\n"),
					nil,
				)
			},
			want: []packages.Package{
				{Name: "bash", Version: "5.1-6.fc36", Arch: "x86_64", Source: "rpm"},
			},
		},
		{
			name:    "linux: nil Exec returns empty list",
			variant: "linux",
			detect:  "debian",
			exec:    func(*testing.T) executor.Executor { return nil },
			want:    []packages.Package{},
		},
		{
			name:    "darwin: brew list returns packages",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(t, "brew", []string{"list", "--versions"}, brewOut, nil)
			},
			want: []packages.Package{
				{Name: "git", Version: "2.40.0", Source: "brew"},
				{Name: "zsh", Version: "5.9", Source: "brew"},
				{Name: "curl", Version: "8.1.1", Source: "brew"},
			},
		},
		{
			name:    "darwin: brew output with blank lines skipped",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"brew",
					[]string{"list", "--versions"},
					[]byte("\ngit 2.40.0\n\n"),
					nil,
				)
			},
			want: []packages.Package{
				{Name: "git", Version: "2.40.0", Source: "brew"},
			},
		},
		{
			name:    "darwin: brew output with only-name line skipped",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(t, "brew", []string{"list", "--versions"}, []byte("git\n"), nil)
			},
			want: []packages.Package{},
		},
		{
			name:    "darwin: brew fails, empty list",
			variant: "darwin",
			exec: func(t *testing.T) executor.Executor {
				return mockExec(
					t,
					"brew",
					[]string{"list", "--versions"},
					nil,
					errors.New("not found"),
				)
			},
			want: []packages.Package{},
		},
		{
			name:    "darwin: nil Exec returns empty list",
			variant: "darwin",
			exec:    func(*testing.T) executor.Executor { return nil },
			want:    []packages.Package{},
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
				c = &packages.Linux{Exec: tt.exec(s.T())}
			case "darwin":
				c = &packages.Darwin{Exec: tt.exec(s.T())}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*packages.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Packages)
		})
	}
}
