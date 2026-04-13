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

package shells_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
)

type ShellsLinuxPublicTestSuite struct {
	suite.Suite
}

func TestShellsLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsLinuxPublicTestSuite))
}

// errorFS wraps a memfs and forces a non-ErrNotExist error from
// ReadFile. Used to exercise the "other read error" branch without
// needing real-FS permission manipulation.
type errorFS struct {
	avfs.VFS
}

func (errorFS) ReadFile(string) ([]byte, error) {
	return nil, errors.New("permission denied")
}

func (s *ShellsLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		setupFS func() avfs.VFS
		wantErr bool
		want    []string
	}{
		{
			name: "canonical /etc/shells",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.WriteFile("/etc/shells",
					[]byte("# comment\n/bin/sh\n/bin/bash\n\n/usr/bin/zsh\n"),
					fs.FileMode(0o644))
				return f
			},
			want: []string{"/bin/sh", "/bin/bash", "/usr/bin/zsh"},
		},
		{
			name: "non-absolute entries skipped",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.WriteFile("/etc/shells",
					[]byte("/bin/sh\nnologin\nbash\n/bin/zsh\n"),
					fs.FileMode(0o644))
				return f
			},
			want: []string{"/bin/sh", "/bin/zsh"},
		},
		{
			name: "whitespace trimmed",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.WriteFile("/etc/shells",
					[]byte("  /bin/bash  \n\t/bin/sh\t\n"),
					fs.FileMode(0o644))
				return f
			},
			want: []string{"/bin/bash", "/bin/sh"},
		},
		{
			name: "empty file",
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.WriteFile("/etc/shells", []byte{}, fs.FileMode(0o644))
				return f
			},
			want: []string{},
		},
		{
			name: "missing file soft-misses",
			setupFS: func() avfs.VFS {
				return memfs.New()
			},
			want: []string{},
		},
		{
			name:    "other read error propagated",
			setupFS: func() avfs.VFS { return errorFS{memfs.New()} },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &shells.Linux{FS: tt.setupFS()}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			got, err := c.Collect(ctx)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*shells.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Paths)
		})
	}
}
