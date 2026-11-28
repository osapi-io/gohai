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
	"runtime"
	"strings"
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

func TestPackagesPublicTestSuite(t *testing.T) {
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
				// New picks the variant for the host the test runs on.
				_, darwin := c.(*packages.Darwin)
				s.Equal(runtime.GOOS == "darwin", darwin)
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
		brewLine = "brew list --formula --versions"
		caskLine = "brew list --cask --versions"
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
		// files are the files of the Darwin variant's host, such as a brew
		// executable at one of its install paths.
		files map[string]string
		// noManager runs the Linux variant with no package manager.
		noManager    bool
		validateFunc func(any, error)
	}{
		{
			name:    "linux arch: no package database, so packages are unsupported",
			variant: "linux",
			detect:  "arch",
			exec: func(t *testing.T) source.Runner {
				return mocks.NewMockRunner(gomock.NewController(t))
			},
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux alpine: no package database, so packages are unsupported",
			variant: "linux",
			detect:  "alpine",
			exec: func(t *testing.T) source.Runner {
				return mocks.NewMockRunner(gomock.NewController(t))
			},
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
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
			name:    "linux rhel: a gpg-pubkey, whose architecture is (none), has none",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{
					rpmLine: "gpg-pubkey\t8b1220fd-61c44f9c\t(none)\tgpg(Fedora (36) <fedora-36-primary@fedoraproject.org>)\t0\n",
				}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{{
					Name:           "gpg-pubkey",
					Version:        "8b1220fd-61c44f9c",
					PackageManager: "rpm",
					Description:    "gpg(Fedora (36) <fedora-36-primary@fedoraproject.org>)",
				}}, info.Packages)
			},
		},
		{
			name:    "linux rhel: a line over 64 KiB keeps the packages after it",
			variant: "linux",
			detect:  "rhel",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{
					rpmLine: "huge\t1.0\tx86_64\t" + strings.Repeat("x", 70000) + "\t1\n" +
						"bash\t5.1-6.fc36\tx86_64\tshell\t2\n",
				}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Require().Len(info.Packages, 2)
				s.Equal("bash", info.Packages[1].Name)
			},
		},
		{
			name:    "linux debian: a line over 64 KiB keeps the packages after it",
			variant: "linux",
			detect:  "debian",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{Out: map[string]string{
					dpkgLine: "huge\t1.0\tamd64\tinstalled\t" + strings.Repeat("x", 70000) + "\n" +
						"bash\t5.1-6\tamd64\tinstalled\n",
				}}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Require().Len(info.Packages, 2)
				s.Equal("bash", info.Packages[1].Name)
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
			name:    "linux: nil Exec gives no fact",
			variant: "linux",
			detect:  "debian",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
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
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
					{
						Name:           "zsh",
						Version:        "5.9",
						Versions:       []string{"5.9"},
						PackageManager: "brew",
						Kind:           "formula",
					},
					{
						Name:           "curl",
						Version:        "8.1.2",
						Versions:       []string{"8.1.2", "8.1.1"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a brew line over 64 KiB keeps the packages after it",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "huge " + strings.Repeat("1.", 35000) + "0\ngit 2.40.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Require().Len(info.Packages, 2)
				s.Equal("git", info.Packages[1].Name)
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
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
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
			name:    "darwin: a cask that brew refuses keeps the formulae",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{brewLine: "git 2.40.0\nzsh 5.9\n"},
					Err: map[string]error{
						caskLine: errors.New(
							"exit status 1: Error: Refusing to load cask tap/cask/neovim-nightly from untrusted tap tap/cask",
						),
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "untrusted tap")
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
					{
						Name:           "zsh",
						Version:        "5.9",
						Versions:       []string{"5.9"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a failed formula list keeps the casks",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{caskLine: "firefox 120.0\n"},
					Err: map[string]error{brewLine: errors.New("exit status 1: permission denied")},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "firefox",
						Version:        "120.0",
						Versions:       []string{"120.0"},
						PackageManager: "brew",
						Kind:           "cask",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: formulae and casks both fail, no fact",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Err: map[string]error{
						brewLine: errors.New("exit status 1: permission denied"),
						caskLine: errors.New("exit status 1: untrusted tap"),
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().Error(err)
				s.Contains(err.Error(), "permission denied")
				s.Contains(err.Error(), "untrusted tap")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: formulae and casks are both listed",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 2.40.0\n",
						caskLine: "firefox 120.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
					{
						Name:           "firefox",
						Version:        "120.0",
						Versions:       []string{"120.0"},
						PackageManager: "brew",
						Kind:           "cask",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a formula and a cask with the same name differ in kind",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "docker 28.5.2\n",
						caskLine: "docker 4.52.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "docker",
						Version:        "28.5.2",
						Versions:       []string{"28.5.2"},
						PackageManager: "brew",
						Kind:           "formula",
					},
					{
						Name:           "docker",
						Version:        "4.52.0",
						Versions:       []string{"4.52.0"},
						PackageManager: "brew",
						Kind:           "cask",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: brew at the Apple silicon path, not on the PATH",
			variant: "darwin",
			files:   map[string]string{"/opt/homebrew/bin/brew": ""},
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"/opt/homebrew/bin/brew list --formula --versions": "git 2.40.0\n",
						"id -u": "501\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: brew at the Intel path, not on the PATH",
			variant: "darwin",
			files:   map[string]string{"/usr/local/bin/brew": ""},
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"/usr/local/bin/brew list --formula --versions": "git 2.40.0\n",
						"id -u": "501\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: as root, brew runs as the owner of its executable",
			variant: "darwin",
			files:   map[string]string{"/opt/homebrew/bin/brew": ""},
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"id -u":                                "0\n",
						"stat -L -f %u /opt/homebrew/bin/brew": "501\n",
						"sudo -u #501 -H /opt/homebrew/bin/brew list --formula --versions": "git 2.40.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: as root, a brew found only on the PATH runs as it is",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"id -u":              "0\n",
						"stat -L -f %u brew": "501\n",
						brewLine:             "git 2.40.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: as root, a root-owned brew runs as it is",
			variant: "darwin",
			files:   map[string]string{"/opt/homebrew/bin/brew": ""},
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"id -u":                                "0\n",
						"stat -L -f %u /opt/homebrew/bin/brew": "0\n",
						"/opt/homebrew/bin/brew list --formula --versions": "git 2.40.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: as root, an owner that stat cannot read runs brew as it is",
			variant: "darwin",
			files:   map[string]string{"/opt/homebrew/bin/brew": ""},
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"id -u": "0\n",
						"/opt/homebrew/bin/brew list --formula --versions": "git 2.40.0\n",
					},
					Err: map[string]error{
						"stat -L -f %u /opt/homebrew/bin/brew": errors.New("exit status 1"),
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a user ID that is not a number runs brew as it is",
			variant: "darwin",
			files:   map[string]string{"/opt/homebrew/bin/brew": ""},
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						"id -u": "nobody\n",
						"/opt/homebrew/bin/brew list --formula --versions": "git 2.40.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.40.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: the newest version is the version, in any order",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 2.9.1 2.40.0 2.10\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "2.40.0",
						Versions:       []string{"2.9.1", "2.40.0", "2.10"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a release is newer than its pre-release",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 1.2.0 1.2.0rc1\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1.2.0",
						Versions:       []string{"1.2.0", "1.2.0rc1"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a pre-release is older than its release",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 1.2.0rc1 1.2.0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1.2.0",
						Versions:       []string{"1.2.0rc1", "1.2.0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a longer version is newer",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 1.2 1.2.1\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1.2.1",
						Versions:       []string{"1.2", "1.2.1"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a shorter version is older",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 1.2.1 1.2\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1.2.1",
						Versions:       []string{"1.2.1", "1.2"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: a number is newer than letters",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git a1 1a b2\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1a",
						Versions:       []string{"a1", "1a", "b2"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: letters compare in order",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 1b 1a_2 1c\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1c",
						Versions:       []string{"1b", "1a_2", "1c"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: equal versions keep the first",
			variant: "darwin",
			exec: func(t *testing.T) source.Runner {
				return sourcetest.Commands{
					Out: map[string]string{
						brewLine: "git 1.0 1_0\n",
					},
				}.Mock(t)
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*packages.Info)
				s.Require().True(ok)
				s.Equal([]packages.Package{
					{
						Name:           "git",
						Version:        "1.0",
						Versions:       []string{"1.0", "1_0"},
						PackageManager: "brew",
						Kind:           "formula",
					},
				}, info.Packages)
			},
		},
		{
			name:    "darwin: nil Exec gives no fact",
			variant: "darwin",
			exec:    func(*testing.T) source.Runner { return nil },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name:      "linux: with no package manager, packages are unsupported",
			variant:   "linux",
			detect:    "debian",
			exec:      func(*testing.T) source.Runner { return source.Commands() },
			noManager: true,
			validateFunc: func(got any, err error) {
				s.ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
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
				c = &packages.Darwin{
					FS:   sourcetest.FS(s.T(), tt.files),
					Exec: tt.exec(s.T()),
				}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
