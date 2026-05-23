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

package livepatch_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/livepatch"
)

var (
	_ collector.Collector = (*livepatch.Linux)(nil)
	_ collector.Collector = (*livepatch.Darwin)(nil)
)

// fsWith builds a memfs with the given path→content entries.
func fsWith(
	t require.TestingT,
	dirs []string,
	files map[string]string,
) avfs.VFS {
	f := memfs.New()
	for _, d := range dirs {
		require.NoError(t, f.MkdirAll(d, 0o755))
	}
	for path, content := range files {
		require.NoError(t, f.WriteFile(path, []byte(content), fs.FileMode(0o644)))
	}
	return f
}

type LivepatchPublicTestSuite struct {
	suite.Suite
}

func TestLivepatchPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(LivepatchPublicTestSuite))
}

func (s *LivepatchPublicTestSuite) TestNew() {
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
			c := livepatch.New()
			s.Equal("livepatch", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*livepatch.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*livepatch.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *LivepatchPublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		setupFS      func() avfs.VFS
		wantNil      bool
		wantErr      bool
		wantPatches  map[string]livepatch.Patch
		wantNilPatch bool
	}{
		{
			name:    "darwin: returns nil",
			variant: "darwin",
			setupFS: func() avfs.VFS { return memfs.New() },
			wantNil: true,
		},
		{
			name:    "linux: livepatch sysfs absent — nil patches",
			variant: "linux",
			setupFS: func() avfs.VFS { return memfs.New() },
			// /sys/kernel/livepatch does not exist — no livepatch support
			wantNilPatch: true,
		},
		{
			name:    "linux: livepatch dir exists but empty",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(
					s.T(),
					[]string{"/sys/kernel/livepatch"},
					map[string]string{},
				)
			},
			wantPatches: map[string]livepatch.Patch{},
		},
		{
			name:    "linux: one patch enabled not in transition",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(
					s.T(),
					[]string{"/sys/kernel/livepatch/lp_cve_2023_0001"},
					map[string]string{
						"/sys/kernel/livepatch/lp_cve_2023_0001/enabled":    "1\n",
						"/sys/kernel/livepatch/lp_cve_2023_0001/transition": "0\n",
					},
				)
			},
			wantPatches: map[string]livepatch.Patch{
				"lp_cve_2023_0001": {Enabled: true, Transition: false},
			},
		},
		{
			name:    "linux: one patch disabled in transition",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(
					s.T(),
					[]string{"/sys/kernel/livepatch/lp_test"},
					map[string]string{
						"/sys/kernel/livepatch/lp_test/enabled":    "0\n",
						"/sys/kernel/livepatch/lp_test/transition": "1\n",
					},
				)
			},
			wantPatches: map[string]livepatch.Patch{
				"lp_test": {Enabled: false, Transition: true},
			},
		},
		{
			name:    "linux: two patches",
			variant: "linux",
			setupFS: func() avfs.VFS {
				return fsWith(
					s.T(),
					[]string{
						"/sys/kernel/livepatch/lp_a",
						"/sys/kernel/livepatch/lp_b",
					},
					map[string]string{
						"/sys/kernel/livepatch/lp_a/enabled":    "1\n",
						"/sys/kernel/livepatch/lp_a/transition": "0\n",
						"/sys/kernel/livepatch/lp_b/enabled":    "1\n",
						"/sys/kernel/livepatch/lp_b/transition": "0\n",
					},
				)
			},
			wantPatches: map[string]livepatch.Patch{
				"lp_a": {Enabled: true, Transition: false},
				"lp_b": {Enabled: true, Transition: false},
			},
		},
		{
			name:    "linux: sysfs files absent — defaults to false",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// Patch dir exists but has no enabled/transition files.
				return fsWith(
					s.T(),
					[]string{"/sys/kernel/livepatch/lp_nofiles"},
					map[string]string{},
				)
			},
			wantPatches: map[string]livepatch.Patch{
				"lp_nofiles": {Enabled: false, Transition: false},
			},
		},
		{
			name:    "linux: non-directory entries in livepatch dir are skipped",
			variant: "linux",
			setupFS: func() avfs.VFS {
				// A regular file alongside a patch directory — must be skipped.
				return fsWith(
					s.T(),
					[]string{"/sys/kernel/livepatch/lp_real"},
					map[string]string{
						"/sys/kernel/livepatch/not_a_dir":       "junk\n",
						"/sys/kernel/livepatch/lp_real/enabled": "1\n",
					},
				)
			},
			wantPatches: map[string]livepatch.Patch{
				"lp_real": {Enabled: true, Transition: false},
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c livepatch.Collector
			switch tt.variant {
			case "linux":
				c = &livepatch.Linux{FS: tt.setupFS()}
			case "darwin":
				c = &livepatch.Darwin{}
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

			info, ok := got.(*livepatch.Info)
			s.Require().True(ok)

			if tt.wantNilPatch {
				s.Nil(info.Patches)
				return
			}

			s.Equal(tt.wantPatches, info.Patches)
		})
	}
}
