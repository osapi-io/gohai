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

package initd_test

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
	initd "github.com/osapi-io/gohai/pkg/gohai/collectors/init"
)

// readErrorFS forces ReadFile to return a non-ErrNotExist error so the
// "read error soft-misses" branch is exercised.
type readErrorFS struct {
	avfs.VFS
}

func (readErrorFS) ReadFile(
	string,
) ([]byte, error) {
	return nil, errors.New("no /proc")
}

var (
	_ collector.Collector = (*initd.Linux)(nil)
	_ collector.Collector = (*initd.Darwin)(nil)
)

type InitPublicTestSuite struct {
	suite.Suite
}

func TestInitPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(InitPublicTestSuite))
}

func (s *InitPublicTestSuite) TestNew() {
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
			c := initd.New()
			s.Equal("init", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*initd.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*initd.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *InitPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string // "linux" | "darwin"
		comm    string
		readErr bool
		want    initd.Info
	}{
		{
			name:    "linux: systemd host",
			variant: "linux",
			comm:    "systemd\n",
			want:    initd.Info{Name: "systemd"},
		},
		{
			name:    "linux: upstart host",
			variant: "linux",
			comm:    "upstart\n",
			want:    initd.Info{Name: "upstart"},
		},
		{
			name:    "linux: sysvinit (comm=init) normalized",
			variant: "linux",
			comm:    "init\n",
			want:    initd.Info{Name: "sysvinit"},
		},
		{
			name:    "linux: openrc-init normalized to openrc",
			variant: "linux",
			comm:    "openrc-init\n",
			want:    initd.Info{Name: "openrc"},
		},
		{
			name:    "linux: runit host",
			variant: "linux",
			comm:    "runit\n",
			want:    initd.Info{Name: "runit"},
		},
		{
			name:    "linux: unknown comm passed through",
			variant: "linux",
			comm:    "exoticinit\n",
			want:    initd.Info{Name: "exoticinit"},
		},
		{
			name:    "linux: missing file soft-misses",
			variant: "linux",
			want:    initd.Info{},
		},
		{
			name:    "linux: read error soft-misses",
			variant: "linux",
			readErr: true,
			want:    initd.Info{},
		},
		{
			name:    "darwin always reports launchd",
			variant: "darwin",
			want:    initd.Info{Name: "launchd"},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c initd.Collector
			switch tt.variant {
			case "linux":
				var vfs avfs.VFS
				switch {
				case tt.readErr:
					vfs = readErrorFS{VFS: memfs.New()}
				case tt.comm != "":
					f := memfs.New()
					_ = f.MkdirAll("/proc/1", 0o755)
					_ = f.WriteFile("/proc/1/comm", []byte(tt.comm), fs.FileMode(0o644))
					vfs = f
				default:
					vfs = memfs.New()
				}
				c = &initd.Linux{FS: vfs}
			case "darwin":
				c = initd.NewDarwin()
			}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*initd.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}
