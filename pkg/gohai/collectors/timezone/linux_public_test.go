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

type TimezoneLinuxPublicTestSuite struct {
	suite.Suite
}

func TestTimezoneLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(TimezoneLinuxPublicTestSuite))
}

func (s *TimezoneLinuxPublicTestSuite) TestCollect() {
	now := func() time.Time {
		return time.Date(2026, 4, 12, 12, 0, 0, 0, time.FixedZone("PDT", -7*3600))
	}
	missingReadlink := func(string) (string, error) { return "", errors.New("not a symlink") }
	missingReadFile := func(string) ([]byte, error) { return nil, errors.New("not found") }

	tests := []struct {
		name       string
		readlink   func(string) (string, error)
		readFile   func(string) ([]byte, error)
		wantName   string
		wantAbbrev string
		wantOffset int
	}{
		{
			name:       "symlink points to IANA zone",
			readlink:   func(string) (string, error) { return "/usr/share/zoneinfo/America/Los_Angeles", nil },
			readFile:   missingReadFile,
			wantName:   "America/Los_Angeles",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:       "target without zoneinfo prefix passed through",
			readlink:   func(string) (string, error) { return "UTC", nil },
			readFile:   missingReadFile,
			wantName:   "UTC",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:       "readlink fails, falls back to /etc/timezone",
			readlink:   missingReadlink,
			readFile:   func(string) ([]byte, error) { return []byte("Europe/Berlin\n"), nil },
			wantName:   "Europe/Berlin",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
		{
			name:       "both sources missing leaves name empty",
			readlink:   missingReadlink,
			readFile:   missingReadFile,
			wantName:   "",
			wantAbbrev: "PDT",
			wantOffset: -7 * 3600,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &timezone.Linux{ReadlinkFn: tt.readlink, ReadFileFn: tt.readFile, NowFn: now}
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
