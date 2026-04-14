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

package osrelease_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
)

var (
	_ collector.Collector = (*osrelease.Linux)(nil)
	_ collector.Collector = (*osrelease.Darwin)(nil)
)

// readErrorFS forces ReadFile to return a non-ErrNotExist error so the
// "other read error propagated" branch is exercised.
type readErrorFS struct {
	avfs.VFS
}

func (readErrorFS) ReadFile(
	string,
) ([]byte, error) {
	return nil, errors.New("permission denied")
}

type OSReleasePublicTestSuite struct {
	suite.Suite
}

func TestOSReleasePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(OSReleasePublicTestSuite))
}

func (s *OSReleasePublicTestSuite) TestNew() {
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
			c := osrelease.New()
			s.Equal("os_release", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*osrelease.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*osrelease.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *OSReleasePublicTestSuite) TestCollect() {
	const ubuntu = `NAME="Ubuntu"
VERSION="24.04 LTS (Noble Numbat)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 24.04 LTS"
VERSION_ID="24.04"
VERSION_CODENAME=noble
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
UBUNTU_CODENAME=noble
`

	tests := []struct {
		name    string
		variant string
		content string
		missing bool
		readErr bool
		wantErr bool
		wantNil bool
		want    osrelease.Info
	}{
		{
			name:    "linux: ubuntu 24.04",
			variant: "linux",
			content: ubuntu,
			want: osrelease.Info{
				ID: "ubuntu", IDLike: []string{"debian"},
				Name: "Ubuntu", PrettyName: "Ubuntu 24.04 LTS",
				Version: "24.04 LTS (Noble Numbat)", VersionID: "24.04",
				VersionCodename: "noble",
				HomeURL:         "https://www.ubuntu.com/",
				SupportURL:      "https://help.ubuntu.com/",
				BugReportURL:    "https://bugs.launchpad.net/ubuntu/",
				Extra:           map[string]string{"UBUNTU_CODENAME": "noble"},
			},
		},
		{
			name:    "linux: missing file soft-misses",
			variant: "linux",
			missing: true,
			want:    osrelease.Info{},
		},
		{
			name:    "linux: other read error propagated",
			variant: "linux",
			readErr: true,
			wantErr: true,
		},
		{
			name:    "linux: comments and blanks and malformed lines",
			variant: "linux",
			content: "# some comment\n\nmalformed\nID=\"quoted\"\n",
			want:    osrelease.Info{ID: "quoted"},
		},
		{
			name:    "linux: build/variant fields populated (Fedora-style)",
			variant: "linux",
			content: `ID=fedora
BUILD_ID=40.20240416.0
VARIANT="Workstation Edition"
VARIANT_ID=workstation
`,
			want: osrelease.Info{
				ID:        "fedora",
				BuildID:   "40.20240416.0",
				Variant:   "Workstation Edition",
				VariantID: "workstation",
			},
		},
		{
			name:    "darwin returns nil",
			variant: "darwin",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c osrelease.Collector
			switch tt.variant {
			case "linux":
				var vfs avfs.VFS
				switch {
				case tt.readErr:
					vfs = readErrorFS{VFS: memfs.New()}
				case tt.missing:
					vfs = memfs.New()
				default:
					f := memfs.New()
					_ = f.MkdirAll("/etc", 0o755)
					_ = f.WriteFile("/etc/os-release", []byte(tt.content), fs.FileMode(0o644))
					vfs = f
				}
				c = &osrelease.Linux{FS: vfs}
			case "darwin":
				c = osrelease.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*osrelease.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
