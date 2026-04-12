//go:build darwin

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

func (s *TimezoneDarwinPublicTestSuite) TestCollectFromFuncs() {
	now := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PDT", -7*3600))
	}

	tests := []struct {
		name       string
		readlink   func(string) (string, error)
		wantName   string
		wantAbbrev string
		wantOffset int
	}{
		{
			name:       "symlink points to IANA zone",
			readlink:   func(string) (string, error) { return "/var/db/timezone/zoneinfo/America/Los_Angeles", nil },
			wantName:   "America/Los_Angeles",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:       "readlink error leaves name empty",
			readlink:   func(string) (string, error) { return "", errors.New("not found") },
			wantName:   "",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			info := timezone.CollectFromFuncs(tt.readlink, now)
			s.Equal(tt.wantName, info.Name)
			s.Equal(tt.wantAbbrev, info.Abbrev)
			s.Equal(tt.wantOffset, info.Offset)
		})
	}
}

func (s *TimezoneDarwinPublicTestSuite) TestCollectDefault() {
	got, err := timezone.Collect(context.Background())
	s.Require().NoError(err)
	info, ok := got.(*timezone.Info)
	s.Require().True(ok)
	s.NotNil(info)
}
