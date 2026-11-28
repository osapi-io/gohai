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

package family_test

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/family"
)

type FamilyPublicTestSuite struct {
	suite.Suite
}

func (s *FamilyPublicTestSuite) TestPlatform() {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{name: "when the ID is remapped", id: "amzn", want: "amazon"},
		{name: "when rhel is remapped to redhat", id: "rhel", want: "redhat"},
		{name: "when the ID is in upper case", id: "OL", want: "oracle"},
		{name: "when the ID is its own platform", id: "ubuntu", want: "ubuntu"},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Equal(tc.want, family.Platform(tc.id))
		})
	}
}

// TestOf holds the table to Ohai's platform_family_from_platform: one
// platform per match, and the cases where Ohai's order decides.
func (s *FamilyPublicTestSuite) TestOf() {
	tests := []struct {
		platform string
		want     string
	}{
		{platform: "ubuntu", want: family.Debian},
		{platform: "debian", want: family.Debian},
		{platform: "linuxmint", want: family.Debian},
		{platform: "raspbian", want: family.Debian},
		{platform: "cumulus", want: family.Debian},
		{platform: "kali", want: family.Debian},
		{platform: "pop", want: family.Debian},
		{platform: "centos", want: family.RHEL},
		{platform: "redhat", want: family.RHEL},
		{platform: "oracle", want: family.RHEL},
		{platform: "almalinux", want: family.RHEL},
		{platform: "rocky", want: family.RHEL},
		{platform: "scientific", want: family.RHEL},
		{platform: "enterpriseenterprise", want: family.RHEL},
		{platform: "xenserver", want: family.RHEL},
		{platform: "xcp-ng", want: family.RHEL},
		{platform: "cloudlinux", want: family.RHEL},
		{platform: "alibabalinux", want: family.RHEL},
		{platform: "sangoma", want: family.RHEL},
		{platform: "clearos", want: family.RHEL},
		{platform: "parallels", want: family.RHEL},
		{platform: "ibm_powerkvm", want: family.RHEL},
		{platform: "bigip", want: family.RHEL},
		{platform: "virtuozzo", want: family.RHEL},
		// centos comes before nexus in Ohai's order.
		{platform: "nexus_centos", want: family.RHEL},
		{platform: "amazon", want: family.Amazon},
		{platform: "suse", want: family.SUSE},
		{platform: "opensuseleap", want: family.SUSE},
		{platform: "sles", want: family.SUSE},
		{platform: "sled", want: family.SUSE},
		{platform: "sle-micro", want: family.SUSE},
		{platform: "sle_hpc", want: family.SUSE},
		{platform: "fedora", want: family.Fedora},
		{platform: "arista_eos", want: family.Fedora},
		{platform: "nexus", want: family.WRLinux},
		{platform: "ios_xr", want: family.WRLinux},
		{platform: "gentoo", want: family.Gentoo},
		{platform: "arch", want: family.Arch},
		{platform: "manjaro", want: family.Arch},
		{platform: "exherbo", want: family.Exherbo},
		{platform: "alpine", want: family.Alpine},
		{platform: "clearlinux", want: family.ClearLinux},
		{platform: "mangeia", want: family.Mandriva},
		{platform: "mageia", want: family.Mandriva},
		{platform: "slackware", want: family.Slackware},
		{platform: "darwin", want: family.Darwin},
		{platform: "Ubuntu", want: family.Debian},
		{platform: "nixos", want: ""},
	}

	for _, tt := range tests {
		s.Run(tt.platform, func() {
			s.Equal(tt.want, family.Of(tt.platform))
		})
	}
}

func (s *FamilyPublicTestSuite) TestDetect() {
	tests := []struct {
		name         string
		validateFunc func(string)
	}{
		{
			name: "it is the family of the real filesystem",
			validateFunc: func(got string) {
				s.Equal(family.FromFS(osfs.NewWithNoIdm()), got)
			},
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(family.Detect())
		})
	}
}

func (s *FamilyPublicTestSuite) TestReadOSRelease() {
	tests := []struct {
		name         string
		files        map[string]string
		dirs         []string
		validateFunc func([]byte, error)
	}{
		{
			name: "when /etc/os-release exists, it is read and /usr/lib is not",
			files: map[string]string{
				"/etc/os-release":     "ID=ubuntu\n",
				"/usr/lib/os-release": "ID=fedora\n",
			},
			validateFunc: func(b []byte, err error) {
				s.Require().NoError(err)
				s.Equal("ID=ubuntu\n", string(b))
			},
		},
		{
			name:  "when only /usr/lib/os-release exists, it is read",
			files: map[string]string{"/usr/lib/os-release": "ID=fedora\n"},
			validateFunc: func(b []byte, err error) {
				s.Require().NoError(err)
				s.Equal("ID=fedora\n", string(b))
			},
		},
		{
			name: "when neither exists, the error is not-exist",
			validateFunc: func(b []byte, err error) {
				s.Require().ErrorIs(err, fs.ErrNotExist)
				s.Nil(b)
			},
		},
		{
			name:  "when /etc/os-release cannot be read, the error names it",
			files: map[string]string{"/usr/lib/os-release": "ID=fedora\n"},
			dirs:  []string{"/etc/os-release"},
			validateFunc: func(b []byte, err error) {
				s.Require().ErrorContains(err, "read /etc/os-release")
				s.Nil(b)
			},
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			vfs := memfs.New()
			for _, dir := range tc.dirs {
				s.Require().NoError(vfs.MkdirAll(dir, 0o755))
			}
			for path, content := range tc.files {
				s.Require().NoError(vfs.MkdirAll(filepath.Dir(path), 0o755))
				s.Require().NoError(vfs.WriteFile(path, []byte(content), 0o644))
			}
			tc.validateFunc(family.ReadOSRelease(vfs))
		})
	}
}

func (s *FamilyPublicTestSuite) TestParseOSRelease() {
	tests := []struct {
		name         string
		data         string
		validateFunc func(map[string]string)
	}{
		{
			name: "pairs, with blank lines, comments and a line with no = skipped",
			data: "# a comment\n\nID=ubuntu\nnot a pair\n  VERSION_ID=24.04  \n",
			validateFunc: func(got map[string]string) {
				s.Equal(map[string]string{"ID": "ubuntu", "VERSION_ID": "24.04"}, got)
			},
		},
		{
			name: "double quotes lose their quotes, and a backslash escapes the next character",
			data: `PRETTY_NAME="Ubuntu \"Noble\" 24.04"` + "\n" + `HOME_URL="a\\b"` + "\n",
			validateFunc: func(got map[string]string) {
				s.Equal(`Ubuntu "Noble" 24.04`, got["PRETTY_NAME"])
				s.Equal(`a\b`, got["HOME_URL"])
			},
		},
		{
			name: "single quotes lose their quotes, and a backslash is kept",
			data: `NAME='Rocky \Linux'` + "\n",
			validateFunc: func(got map[string]string) {
				s.Equal(`Rocky \Linux`, got["NAME"])
			},
		},
		{
			name: "an unmatched quote is kept, and a repeated key keeps its last value",
			data: "ID=\"fedora\nID=rocky\nVARIANT='x\n",
			validateFunc: func(got map[string]string) {
				s.Equal("rocky", got["ID"])
				s.Equal("'x", got["VARIANT"])
			},
		},
		{
			name: "empty content has no pairs",
			validateFunc: func(got map[string]string) {
				s.Empty(got)
			},
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			tc.validateFunc(family.ParseOSRelease([]byte(tc.data)))
		})
	}
}

func (s *FamilyPublicTestSuite) TestFromFS() {
	tests := []struct {
		name  string
		files map[string]string
		dirs  []string
		want  string
	}{
		{
			name: "when the filesystem is macOS's",
			files: map[string]string{
				"/System/Library/CoreServices/SystemVersion.plist": "<plist/>",
			},
			want: family.Darwin,
		},
		{
			name: "when the filesystem has both, os-release wins",
			files: map[string]string{
				"/System/Library/CoreServices/SystemVersion.plist": "<plist/>",
				"/etc/os-release": "ID=ubuntu\n",
			},
			want: family.Debian,
		},
		{
			name:  "when the filesystem names a Linux platform",
			files: map[string]string{"/etc/os-release": "ID=fedora\n"},
			want:  family.Fedora,
		},
		{
			name:  "when the ID is quoted with an escape, the escape is read",
			files: map[string]string{"/etc/os-release": `ID="rock\y"` + "\n"},
			want:  family.RHEL,
		},
		{
			name:  "when only /usr/lib/os-release exists, it names the platform",
			files: map[string]string{"/usr/lib/os-release": "ID=fedora\n"},
			want:  family.Fedora,
		},
		{
			name:  "when /etc/os-release cannot be read, /usr/lib is not read",
			files: map[string]string{"/usr/lib/os-release": "ID=fedora\n"},
			dirs:  []string{"/etc/os-release"},
			want:  "",
		},
		{
			name:  "when the ID is not quoted",
			files: map[string]string{"/etc/os-release": "ID=ubuntu\nID_LIKE=debian\n"},
			want:  family.Debian,
		},
		{
			name: "when a derivative names its parent in ID_LIKE",
			files: map[string]string{
				"/etc/os-release": "ID=elementary\nID_LIKE=\"ubuntu debian\"\n",
			},
			want: family.Debian,
		},
		{
			name:  "when ID_LIKE names no parent with a family",
			files: map[string]string{"/etc/os-release": "ID=mystery\nID_LIKE=other\n"},
			want:  "mystery",
		},
		{
			name:  "when the platform has no family",
			files: map[string]string{"/etc/os-release": "ID=NixOS\n"},
			want:  "nixos",
		},
		{
			name:  "when the file has no ID",
			files: map[string]string{"/etc/os-release": "NAME=Something\n"},
			want:  "",
		},
		{
			name: "when the platform has a family",
			files: map[string]string{
				"/etc/os-release": "NAME=\"Amazon Linux\"\nID=\"amzn\"\nVERSION_ID=\"2023\"\n",
			},
			want: family.Amazon,
		},
		{name: "when the filesystem has neither", want: ""},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			fs := memfs.New()
			for _, dir := range tc.dirs {
				s.Require().NoError(fs.MkdirAll(dir, 0o755))
			}
			for path, content := range tc.files {
				s.Require().NoError(fs.MkdirAll(filepath.Dir(path), 0o755))
				s.Require().NoError(fs.WriteFile(path, []byte(content), 0o644))
			}
			s.Equal(tc.want, family.FromFS(fs))
		})
	}
}

func TestFamilyPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(FamilyPublicTestSuite))
}
