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

package timezone_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
)

var (
	_ collector.Collector = (*timezone.Linux)(nil)
	_ collector.Collector = (*timezone.Darwin)(nil)
)

type TimezonePublicTestSuite struct {
	suite.Suite
}

func TestTimezonePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(TimezonePublicTestSuite))
}

func (s *TimezonePublicTestSuite) TestNew() {
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
			c := timezone.New()
			s.Equal("timezone", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*timezone.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*timezone.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *TimezonePublicTestSuite) TestCollect() {
	pdt := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PDT", -7*3600))
	}
	pst := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PST", -8*3600))
	}

	tests := []struct {
		name       string
		variant    string
		now        func() time.Time
		setupFS    func() avfs.VFS
		wantName   string
		wantAbbrev string
		wantOffset int
	}{
		{
			name:    "linux: symlink points to IANA zone",
			variant: "linux",
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/usr/share/zoneinfo/America", 0o755)
				_ = f.WriteFile(
					"/usr/share/zoneinfo/America/Los_Angeles",
					[]byte{},
					fs.FileMode(0o644),
				)
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.Symlink("/usr/share/zoneinfo/America/Los_Angeles", "/etc/localtime")
				return f
			},
			wantName:   "America/Los_Angeles",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:    "linux: target without zoneinfo prefix passed through",
			variant: "linux",
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.Symlink("UTC", "/etc/localtime")
				return f
			},
			wantName:   "UTC",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:    "linux: readlink fails, falls back to /etc/timezone",
			variant: "linux",
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.WriteFile("/etc/timezone", []byte("Europe/Berlin\n"), fs.FileMode(0o644))
				return f
			},
			wantName:   "Europe/Berlin",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:       "linux: both sources missing leaves name empty",
			variant:    "linux",
			now:        pdt,
			setupFS:    func() avfs.VFS { return memfs.New() },
			wantName:   "",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:    "darwin: macOS zoneinfo symlink",
			variant: "darwin",
			now:     pst,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.Symlink("/var/db/timezone/zoneinfo/America/Los_Angeles", "/etc/localtime")
				return f
			},
			wantName:   "America/Los_Angeles",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
		{
			name:    "darwin: target without prefix passed through",
			variant: "darwin",
			now:     pst,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.Symlink("UTC", "/etc/localtime")
				return f
			},
			wantName:   "UTC",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
		{
			name:       "darwin: readlink error leaves name empty",
			variant:    "darwin",
			now:        pst,
			setupFS:    func() avfs.VFS { return memfs.New() },
			wantName:   "",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer timezone.SetNowFn(tt.now)()
			var c timezone.Collector
			switch tt.variant {
			case "linux":
				c = &timezone.Linux{FS: tt.setupFS()}
			case "darwin":
				c = &timezone.Darwin{FS: tt.setupFS()}
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			info, ok := got.(*timezone.Info)
			s.Require().True(ok)
			s.Equal(tt.wantName, info.Name)
			s.Equal(tt.wantAbbrev, info.Abbrev)
			s.Equal(tt.wantOffset, info.Offset)
		})
	}
}
