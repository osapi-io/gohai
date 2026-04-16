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

package users_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/users"
)

var (
	_ collector.Collector = (*users.Linux)(nil)
	_ collector.Collector = (*users.Darwin)(nil)
)

const passwdFixture = `# comment
root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
john:x:1000:1000:John Doe,,,:/home/john:/bin/zsh
dup:x:2:2:first:/home/a:/bin/sh
dup:x:99:99:second:/home/b:/bin/sh
malformed-line-missing-fields
`

const groupFixture = `# comment
root:x:0:
wheel:x:10:john,root
staff:x:20:
users:x:1000:john,daemon
malformed:x:`

func newFS(
	files map[string]string,
) avfs.VFS {
	f := memfs.New()
	_ = f.MkdirAll("/etc", 0o755)
	for path, content := range files {
		_ = f.WriteFile(path, []byte(content), fs.FileMode(0o644))
	}
	return f
}

type UsersPublicTestSuite struct {
	suite.Suite
}

func TestUsersPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UsersPublicTestSuite))
}

func (s *UsersPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name     string
		detect   string
		wantKind string
	}{
		{"darwin dispatches to Darwin", "darwin", "darwin"},
		{"debian dispatches to Linux", "debian", "linux"},
		{"unknown dispatches to Linux", "", "linux"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := users.New()
			s.Equal("users", c.Name())
			s.Equal("users", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*users.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*users.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *UsersPublicTestSuite) TestCollect() {
	tests := []struct {
		name     string
		variant  string
		files    map[string]string
		euid     int
		validate func(*users.Info)
	}{
		{
			name:    "linux: full fixture parses passwd + group + current_user",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": passwdFixture,
				"/etc/group":  groupFixture,
			},
			euid: 1000,
			validate: func(info *users.Info) {
				s.Require().Len(info.Passwd, 4)
				root := info.Passwd["root"]
				s.Equal(0, root.UID)
				s.Equal("/bin/bash", root.Shell)
				s.Equal("root", root.GECOS)
				john := info.Passwd["john"]
				s.Equal(1000, john.UID)
				s.Equal("John Doe,,,", john.GECOS)
				s.Equal("/home/john", john.Dir)
				// Duplicate "dup" entry kept first.
				dup := info.Passwd["dup"]
				s.Equal(2, dup.UID)
				s.Equal("/home/a", dup.Dir)

				wheel := info.Group["wheel"]
				s.Equal(10, wheel.GID)
				s.Equal([]string{"john", "root"}, wheel.Members)
				staff := info.Group["staff"]
				s.Equal(20, staff.GID)
				s.Empty(staff.Members)

				s.Equal("john", info.CurrentUser)
			},
		},
		{
			name:    "linux: missing files produces empty maps, no error",
			variant: "linux",
			files:   map[string]string{},
			euid:    0,
			validate: func(info *users.Info) {
				s.Empty(info.Passwd)
				s.Empty(info.Group)
				s.Empty(info.CurrentUser)
			},
		},
		{
			name:    "linux: current user lookup misses when euid absent",
			variant: "linux",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/root:/bin/bash\n",
			},
			euid: 9999,
			validate: func(info *users.Info) {
				s.Equal("", info.CurrentUser)
			},
		},
		{
			name:    "darwin: same POSIX parsing",
			variant: "darwin",
			files: map[string]string{
				"/etc/passwd": "root:x:0:0:root:/var/root:/bin/zsh\njohn:x:501:20:John:/Users/john:/bin/zsh\n",
				"/etc/group":  "admin:x:80:root,john\n",
			},
			euid: 501,
			validate: func(info *users.Info) {
				s.Require().Len(info.Passwd, 2)
				s.Equal(501, info.Passwd["john"].UID)
				s.Equal([]string{"root", "john"}, info.Group["admin"].Members)
				s.Equal("john", info.CurrentUser)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer users.SetGeteuidFn(func() int { return tt.euid })()
			var c users.Collector
			switch tt.variant {
			case "linux":
				c = &users.Linux{FS: newFS(tt.files)}
			case "darwin":
				c = &users.Darwin{FS: newFS(tt.files)}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*users.Info)
			s.Require().True(ok)
			tt.validate(info)
		})
	}
}
