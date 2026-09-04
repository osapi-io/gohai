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
		{"darwin dispatches to Darwin", osDarwin, osDarwin},
		{"debian dispatches to Linux", "debian", osLinux},
		{"rhel dispatches to Linux", "rhel", osLinux},
		{"arch dispatches to Linux", "arch", osLinux},
		{"unknown dispatches to Linux", "", osLinux},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := timezone.New()
			s.Equal("timezone", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case osDarwin:
				_, ok := c.(*timezone.Darwin)
				s.True(ok)
			case osLinux:
				_, ok := c.(*timezone.Linux)
				s.True(ok)
			default:
				s.Failf("unhandled case", "%v", tt.wantKind)
			}
		})
	}
}

func (s *TimezonePublicTestSuite) TestCollect() {
	pdt := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone(zonePDT, -7*3600))
	}
	pst := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone(zonePST, -8*3600))
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
			variant: osLinux,
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll("/usr/share/zoneinfo/America", 0o755)
				_ = f.WriteFile(
					"/usr/share/zoneinfo/America/Los_Angeles",
					[]byte{},
					fs.FileMode(0o644),
				)
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.Symlink("/usr/share/zoneinfo/America/Los_Angeles", pathEtcLocaltime)
				return f
			},
			wantName:   "America/Los_Angeles",
			wantAbbrev: zonePDT,
			wantOffset: -7 * 3600,
		},
		{
			name:    "linux: target without zoneinfo prefix passed through",
			variant: osLinux,
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.Symlink(zoneUTC, pathEtcLocaltime)
				return f
			},
			wantName:   zoneUTC,
			wantAbbrev: zonePDT,
			wantOffset: -7 * 3600,
		},
		{
			name:    "linux: readlink fails, falls back to /etc/timezone",
			variant: osLinux,
			now:     pdt,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.WriteFile("/etc/timezone", []byte("Europe/Berlin\n"), fs.FileMode(0o644))
				return f
			},
			wantName:   "Europe/Berlin",
			wantAbbrev: zonePDT,
			wantOffset: -7 * 3600,
		},
		{
			name:       "linux: both sources missing leaves name empty",
			variant:    osLinux,
			now:        pdt,
			setupFS:    func() avfs.VFS { return memfs.New() },
			wantName:   "",
			wantAbbrev: zonePDT,
			wantOffset: -7 * 3600,
		},
		{
			name:    "darwin: macOS zoneinfo symlink",
			variant: osDarwin,
			now:     pst,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.Symlink("/var/db/timezone/zoneinfo/America/Los_Angeles", pathEtcLocaltime)
				return f
			},
			wantName:   "America/Los_Angeles",
			wantAbbrev: zonePST,
			wantOffset: -8 * 3600,
		},
		{
			name:    "darwin: target without prefix passed through",
			variant: osDarwin,
			now:     pst,
			setupFS: func() avfs.VFS {
				f := memfs.New()
				_ = f.MkdirAll(pathEtc, 0o755)
				_ = f.Symlink(zoneUTC, pathEtcLocaltime)
				return f
			},
			wantName:   zoneUTC,
			wantAbbrev: zonePST,
			wantOffset: -8 * 3600,
		},
		{
			name:       "darwin: readlink error leaves name empty",
			variant:    osDarwin,
			now:        pst,
			setupFS:    func() avfs.VFS { return memfs.New() },
			wantName:   "",
			wantAbbrev: zonePST,
			wantOffset: -8 * 3600,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer timezone.SetNowFn(tt.now)()
			var c timezone.Collector
			switch tt.variant {
			case osLinux:
				c = &timezone.Linux{FS: tt.setupFS()}
			case osDarwin:
				c = &timezone.Darwin{FS: tt.setupFS()}
			default:
				s.Failf("unhandled case", "%v", tt.variant)
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
