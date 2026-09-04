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

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/shells"
)

var (
	_ collector.Collector = (*shells.Linux)(nil)
	_ collector.Collector = (*shells.Darwin)(nil)
)

// errorFS wraps a memfs and forces a non-ErrNotExist error from
// ReadFile. Used to exercise the "other read error" branch without
// needing real-FS permission manipulation.
type errorFS struct {
	avfs.VFS
}

func (errorFS) ReadFile(
	string,
) ([]byte, error) {
	return nil, errors.New("permission denied")
}

type ShellsPublicTestSuite struct {
	suite.Suite
}

func TestShellsPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(ShellsPublicTestSuite))
}

func (s *ShellsPublicTestSuite) TestNew() {
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name       string
		detect     string
		wantKind   string
		wantFSFrom func() avfs.VFS
	}{
		{
			"darwin dispatches to Darwin + wires FS",
			osDarwin,
			osDarwin,
			func() avfs.VFS { return shells.NewDarwin().FS },
		},
		{
			"debian dispatches to Linux + wires FS",
			"debian",
			osLinux,
			func() avfs.VFS { return shells.NewLinux().FS },
		},
		{"rhel dispatches to Linux", "rhel", osLinux, nil},
		{"arch dispatches to Linux", "arch", osLinux, nil},
		{"unknown dispatches to Linux", "", osLinux, nil},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := shells.New()
			s.Equal("shells", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*shells.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*shells.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
			if tt.wantFSFrom != nil {
				s.NotNil(tt.wantFSFrom())
			}
		})
	}
}

func (s *ShellsPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() avfs.VFS
		wantErr bool
		want    []string
	}{
		{
			name:    "linux: canonical /etc/shells",
			variant: osLinux,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.WriteFile(pathEtcShells,
					[]byte("# comment\n/bin/sh\n/bin/bash\n\n/usr/bin/zsh\n"),
					fs.FileMode(0o644))
				return f
			},
			want: []string{pathBinSh, "/bin/bash", "/usr/bin/zsh"},
		},
		{
			name:    "linux: non-absolute entries skipped",
			variant: osLinux,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.WriteFile(pathEtcShells,
					[]byte("/bin/sh\nnologin\nbash\n/bin/zsh\n"),
					fs.FileMode(0o644))
				return f
			},
			want: []string{pathBinSh, "/bin/zsh"},
		},
		{
			name:    "linux: whitespace trimmed",
			variant: osLinux,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.WriteFile(pathEtcShells,
					[]byte("  /bin/bash  \n\t/bin/sh\t\n"),
					fs.FileMode(0o644))
				return f
			},
			want: []string{"/bin/bash", pathBinSh},
		},
		{
			name:    "linux: empty file",
			variant: osLinux,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.WriteFile(pathEtcShells, []byte{}, fs.FileMode(0o644))
				return f
			},
			want: []string{},
		},
		{
			name:    "linux: missing file soft-misses",
			variant: osLinux,
			setupFS: func() avfs.VFS { return memfs.New() },
			want:    []string{},
		},
		{
			name:    "linux: other read error propagated",
			variant: osLinux,
			setupFS: func() avfs.VFS { return errorFS{memfs.New()} },
			wantErr: true,
		},
		{
			name:    "darwin: canonical macOS /etc/shells",
			variant: osDarwin,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.WriteFile(
					pathEtcShells,
					[]byte(
						"/bin/bash\n/bin/csh\n/bin/dash\n/bin/ksh\n/bin/sh\n/bin/tcsh\n/bin/zsh\n",
					),
					fs.FileMode(0o644),
				)
				return f
			},
			want: []string{
				"/bin/bash",
				"/bin/csh",
				"/bin/dash",
				"/bin/ksh",
				pathBinSh,
				"/bin/tcsh",
				"/bin/zsh",
			},
		},
		{
			name:    "darwin: missing file soft-misses",
			variant: osDarwin,
			setupFS: func() avfs.VFS { return memfs.New() },
			want:    []string{},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c shells.Collector
			switch tt.variant {
			case osLinux:
				c = &shells.Linux{FS: tt.setupFS()}
			case osDarwin:
				c = &shells.Darwin{FS: tt.setupFS()}
			default:
				s.Failf("unhandled case", "%v", tt.variant)
			}
			got, err := c.Collect(context.Background(), nil)
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
