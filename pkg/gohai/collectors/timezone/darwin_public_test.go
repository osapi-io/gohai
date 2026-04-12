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
	"errors"
	"testing"
	"time"

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
		readlink   func(string) (string, error)
		wantName   string
		wantAbbrev string
		wantOffset int
	}{
		{
			name:       "macOS zoneinfo symlink",
			readlink:   func(string) (string, error) { return "/var/db/timezone/zoneinfo/America/Los_Angeles", nil },
			wantName:   "America/Los_Angeles",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
		{
			name:       "target without prefix passed through",
			readlink:   func(string) (string, error) { return "UTC", nil },
			wantName:   "UTC",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
		{
			name:       "readlink error leaves name empty",
			readlink:   func(string) (string, error) { return "", errors.New("not a symlink") },
			wantName:   "",
			wantAbbrev: "PST",
			wantOffset: -8 * 3600,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &timezone.Darwin{ReadlinkFn: tt.readlink, NowFn: now}
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

func (s *TimezoneDarwinPublicTestSuite) TestNewDarwinWiresUpStdlib() {
	c := timezone.NewDarwin()
	s.NotNil(c.ReadlinkFn)
	s.NotNil(c.NowFn)
	s.False(c.NowFn().IsZero())
}
