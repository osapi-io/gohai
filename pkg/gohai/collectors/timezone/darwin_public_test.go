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
	"testing"
	"time"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/timezone"
)

type TimezoneDarwinPublicTestSuite struct {
	suite.Suite
}

func TestTimezoneDarwinPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TimezoneDarwinPublicTestSuite))
}

func (s *TimezoneDarwinPublicTestSuite) TestCollect() {
	now := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PST", -8*3600))
	}

	tests := []struct {
		name       string
		setupFS    func() avfs.VFS
		wantName   string
		wantAbbrev string
		wantOffset int
	}{
		{
			name: "macOS zoneinfo symlink",
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
			name: "target without prefix passed through",
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
			name:       "readlink error leaves name empty",
			setupFS:    func() avfs.VFS { return memfs.New() },
			wantName:   "",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer timezone.SetNowFn(now)()
			c := &timezone.Darwin{FS: tt.setupFS()}
			got, err := c.Collect(context.Background())
			s.Require().NoError(err)
			info, ok := got.(*timezone.Info)
			s.Require().True(ok)
			s.Equal(tt.wantName, info.Name)
			s.Equal(tt.wantAbbrev, info.Abbrev)
			s.Equal(tt.wantOffset, info.Offset)
		})
	}
}
