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
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
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
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
