// Copyright (c) 2024 John Dewey

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

package uptime_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
)

type UptimePublicTestSuite struct {
	suite.Suite
}

func TestUptimePublicTestSuite(t *testing.T) {
	suite.Run(t, new(UptimePublicTestSuite))
}

func (s *UptimePublicTestSuite) TestNew() {
	c := uptime.New()
	s.Equal("uptime", c.Name())
	s.Equal(collector.TierCore, c.Tier())
	s.Empty(c.Dependencies())
}

func (s *UptimePublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = uptime.New()
}

func (s *UptimePublicTestSuite) TestCollect() {
	c := uptime.New()
	got, err := c.Collect(context.Background())
	s.Require().NoError(err)
	if got == nil {
		return
	}
	info, ok := got.(*uptime.Info)
	s.Require().True(ok)
	s.NotZero(info.Seconds + info.BootTime) // at least one populated
}

func (s *UptimePublicTestSuite) TestHumanDuration() {
	tests := []struct {
		name    string
		seconds uint64
		want    string
	}{
		{"zero", 0, "0s"},
		{"seconds only", 45, "45s"},
		{"minutes", 3*60 + 7, "3m 7s"},
		{"hours", 2*3600 + 5*60 + 9, "2h 5m 9s"},
		{"days", 4*86400 + 6*3600 + 15*60 + 2, "4d 6h 15m 2s"},
		{"exactly one day", 86400, "1d 0h 0m 0s"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, uptime.HumanDuration(tt.seconds))
		})
	}
}
