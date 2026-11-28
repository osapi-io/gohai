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
	"path/filepath"
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
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
		name         string
		detect       string
		validateFunc func(timezone.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c timezone.Collector) {
				_, ok := c.(*timezone.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c timezone.Collector) {
				_, ok := c.(*timezone.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c timezone.Collector) {
				_, ok := c.(*timezone.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c timezone.Collector) {
				_, ok := c.(*timezone.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c timezone.Collector) {
				_, ok := c.(*timezone.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := timezone.New()
			s.Equal("timezone", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

// tzdata is a zone file from testdata, copied from Go's zoneinfo.zip.
func (s *TimezonePublicTestSuite) tzdata(
	name string,
) []byte {
	b, err := osfs.NewWithNoIdm().ReadFile(filepath.Join("testdata", name))
	s.Require().NoError(err)
	return b
}

func (s *TimezonePublicTestSuite) TestCollect() {
	pdt := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PDT", -7*3600))
	}
	pst := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PST", -8*3600))
	}
	utc := func() time.Time { return time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC) }
	tokyo := s.tzdata("Asia_Tokyo")

	tests := []struct {
		name         string
		variant      string
		now          func() time.Time
		setupFS      func() avfs.VFS
		validateFunc func(any, error)
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("America/Los_Angeles", info.Name)
				s.Equal("PDT", info.Abbrev)
				s.Equal(-7*3600, info.Offset)
			},
		},
		{
			name:    "linux: a relative link, as Ubuntu has, resolves from /etc",
			variant: "linux",
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/etc", 0o755)
				_ = f.Symlink("../usr/share/zoneinfo/Etc/UTC", "/etc/localtime")
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("Etc/UTC", info.Name)
			},
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("UTC", info.Name)
				s.Equal("PDT", info.Abbrev)
				s.Equal(-7*3600, info.Offset)
			},
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("Europe/Berlin", info.Name)
				s.Equal("PDT", info.Abbrev)
				s.Equal(-7*3600, info.Offset)
			},
		},
		{
			name:    "linux: both sources missing leaves name empty",
			variant: "linux",
			now:     pdt,
			setupFS: func() avfs.VFS { return memfs.New() },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("PDT", info.Abbrev)
				s.Equal(-7*3600, info.Offset)
			},
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("America/Los_Angeles", info.Name)
				s.Equal("PST", info.Abbrev)
				s.Equal(-8*3600, info.Offset)
			},
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
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("UTC", info.Name)
				s.Equal("PST", info.Abbrev)
				s.Equal(-8*3600, info.Offset)
			},
		},
		{
			name:    "darwin: readlink error leaves name empty",
			variant: "darwin",
			now:     pst,
			setupFS: func() avfs.VFS { return memfs.New() },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("PST", info.Abbrev)
				s.Equal(-8*3600, info.Offset)
			},
		},
		{
			name:    "linux: a copied /etc/localtime gives the zone's abbreviation and offset",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.WriteFile("/etc/localtime", tokyo, 0o644))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("JST", info.Abbrev)
				s.Equal(9*3600, info.Offset)
			},
		},
		{
			name:    "linux: a copied /etc/localtime and /etc/sysconfig/clock name the zone",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.WriteFile("/etc/localtime", tokyo, 0o644))
				s.Require().NoError(f.MkdirAll("/etc/sysconfig", 0o755))
				s.Require().
					NoError(f.WriteFile("/etc/sysconfig/clock", []byte("# comment\nZONE=\"Asia/Tokyo\"\nUTC=true\n"), 0o644))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("Asia/Tokyo", info.Name)
				s.Equal("JST", info.Abbrev)
				s.Equal(9*3600, info.Offset)
			},
		},
		{
			name:    "linux: /etc/timezone that is not a name gives way to /etc/sysconfig/clock",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.WriteFile("/etc/timezone", []byte("not a zone\n"), 0o644))
				s.Require().NoError(f.MkdirAll("/etc/sysconfig", 0o755))
				s.Require().
					NoError(f.WriteFile("/etc/sysconfig/clock", []byte("ZONE=Europe/Paris\n"), 0o644))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("Europe/Paris", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
		{
			name:    "linux: /etc/sysconfig/clock with no ZONE line gives no name",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc/sysconfig", 0o755))
				s.Require().
					NoError(f.WriteFile("/etc/sysconfig/clock", []byte("UTC=true\n"), 0o644))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
		{
			name:    "linux: a /etc/localtime that is not zone data uses the clock's zone",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.WriteFile("/etc/localtime", []byte("garbage"), 0o644))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
		{
			name:    "linux: NixOS's link under /etc/zoneinfo gives the name",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.Symlink("/etc/zoneinfo/Europe/Paris", "/etc/localtime"))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("Europe/Paris", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
		{
			name:    "linux: a link to a path with no zoneinfo directory gives no name",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.Symlink("/opt/tz/custom.file", "/etc/localtime"))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
		{
			name:    "linux: a link whose zoneinfo part is not a name gives no name",
			variant: "linux",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().NoError(f.Symlink("/usr/share/zoneinfo/not a zone", "/etc/localtime"))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
		{
			name:    "darwin: a link under a versioned zoneinfo directory gives the name",
			variant: "darwin",
			now:     utc,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				s.Require().NoError(f.MkdirAll("/etc", 0o755))
				s.Require().
					NoError(f.Symlink("/var/db/timezone/tz/2024a.1.0/zoneinfo/Asia/Tokyo", "/etc/localtime"))
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*timezone.Info)
				s.Require().True(ok)
				s.Equal("Asia/Tokyo", info.Name)
				s.Equal("UTC", info.Abbrev)
				s.Equal(0, info.Offset)
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c timezone.Collector
			switch tt.variant {
			case "linux":
				c = &timezone.Linux{FS: tt.setupFS(), Now: tt.now}
			case "darwin":
				c = &timezone.Darwin{FS: tt.setupFS(), Now: tt.now}
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
