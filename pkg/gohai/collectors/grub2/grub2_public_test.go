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

package grub2_test

import (
	"context"
	"io/fs"
	"path"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/grub2"
)

var (
	_ collector.Collector = (*grub2.Linux)(nil)
	_ collector.Collector = (*grub2.Darwin)(nil)
)

// fsWith builds a memfs with the given path→content entries.
func fsWith(
	t require.TestingT,
	files map[string]string,
) avfs.VFS {
	f := memfs.New()
	for p, content := range files {
		dir := path.Dir(p)
		require.NoError(t, f.MkdirAll(dir, 0o755))
		require.NoError(t, f.WriteFile(p, []byte(content), fs.FileMode(0o644)))
	}
	return f
}

// grubenv is a representative GRUB2 environment block.
const grubenv = `# GRUB Environment Block
saved_entry=0
boot_success=1
boot_indeterminate=0
kernelopts=root=/dev/mapper/fedora-root ro
`

type Grub2PublicTestSuite struct {
	suite.Suite
}

func TestGrub2PublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(Grub2PublicTestSuite))
}

func (s *Grub2PublicTestSuite) TestNew() {
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
			c := grub2.New()
			s.Equal("grub2", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*grub2.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*grub2.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *Grub2PublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() avfs.VFS
		wantNil bool
		wantEnv map[string]string
	}{
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			wantNil: true,
		},
		{
			name:    "linux: no grubenv on any path — nil environment",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			wantEnv: nil,
		},
		{
			name:    "linux: grubenv at /boot/grub2/grubenv",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/boot/grub2/grubenv": grubenv,
				})
			},
			wantEnv: map[string]string{
				"saved_entry":        "0",
				"boot_success":       "1",
				"boot_indeterminate": "0",
				"kernelopts":         "root=/dev/mapper/fedora-root ro",
			},
		},
		{
			name:    "linux: grubenv at /boot/grub/grubenv (Debian path)",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/boot/grub/grubenv": "# GRUB Environment Block\nsaved_entry=ubuntu\n",
				})
			},
			wantEnv: map[string]string{
				"saved_entry": "ubuntu",
			},
		},
		{
			name:    "linux: grub2 path takes priority over grub path",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/boot/grub2/grubenv": "from=grub2\n",
					"/boot/grub/grubenv":  "from=grub\n",
				})
			},
			wantEnv: map[string]string{"from": "grub2"},
		},
		{
			name:    "linux: empty grubenv file",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/boot/grub2/grubenv": "",
				})
			},
			wantEnv: map[string]string{},
		},
		{
			name:    "linux: lines without equals sign are skipped",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(s.T(), map[string]string{
					"/boot/grub2/grubenv": "# comment\nno_equals_here\nkey=value\n",
				})
			},
			wantEnv: map[string]string{"key": "value"},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c grub2.Collector
			switch tt.variant {
			case "linux":
				c = &grub2.Linux{FS: tt.setupFS()}
			case "darwin":
				c = &grub2.Darwin{}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)

			if tt.wantNil {
				s.Nil(got)
				return
			}

			info, ok := got.(*grub2.Info)
			s.Require().True(ok)
			s.Equal(tt.wantEnv, info.Environment)
		})
	}
}
