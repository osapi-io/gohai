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

	osrelease "github.com/osapi-io/gohai/pkg/gohai/collectors/os_release"
)

// readErrorFS forces ReadFile to return a non-ErrNotExist error so the
// "other read error propagated" branch is exercised.
type readErrorFS struct {
	avfs.VFS
}

func (readErrorFS) ReadFile(string) ([]byte, error) {
	return nil, errors.New("permission denied")
}

type OSReleaseLinuxPublicTestSuite struct {
	suite.Suite
}

func TestOSReleaseLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(OSReleaseLinuxPublicTestSuite))
}

func (s *OSReleaseLinuxPublicTestSuite) TestCollect() {
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
		content string
		missing bool
		readErr bool
		wantErr bool
		want    osrelease.Info
	}{
		{
			name:    "ubuntu 24.04",
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
			name:    "missing file soft-misses",
			missing: true,
			want:    osrelease.Info{},
		},
		{
			name:    "other read error propagated",
			readErr: true,
			wantErr: true,
		},
		{
			name:    "comments and blanks and malformed lines",
			content: "# some comment\n\nmalformed\nID=\"quoted\"\n",
			want:    osrelease.Info{ID: "quoted"},
		},
		{
			name: "build/variant fields populated (Fedora-style)",
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
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
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
			c := &osrelease.Linux{FS: vfs}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*osrelease.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
