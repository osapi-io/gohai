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

package uptime_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
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
	s.True(c.DefaultEnabled())
	s.Empty(c.Dependencies())
}

func (s *UptimePublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = uptime.NewLinux()
	var _ collector.Collector = uptime.NewDarwin()
}

func (s *UptimePublicTestSuite) TestNewDispatch() {
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
			got := uptime.New()
			switch tt.wantKind {
			case "darwin":
				_, ok := got.(*uptime.Darwin)
				s.True(ok)
			case "linux":
				_, ok := got.(*uptime.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *UptimePublicTestSuite) TestHumanDuration() {
	tests := []struct {
		name    string
		seconds uint64
		want    string
	}{
		{"zero seconds", 0, "0s"},
		{"seconds only", 45, "45s"},
		{"minutes and seconds", 75, "1m 15s"},
		{"hours/minutes/seconds", 3*3600 + 12*60 + 5, "3h 12m 5s"},
		{"days+hours+minutes+seconds", 2*86400 + 5*3600 + 12*60 + 5, "2d 5h 12m 5s"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, uptime.HumanDuration(tt.seconds))
		})
	}
}
