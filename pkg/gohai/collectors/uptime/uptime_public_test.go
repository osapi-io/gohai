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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

var (
	_ collector.Collector = (*uptime.Linux)(nil)
	_ collector.Collector = (*uptime.Darwin)(nil)
)

type UptimePublicTestSuite struct {
	suite.Suite
}

func TestUptimePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(UptimePublicTestSuite))
}

func (s *UptimePublicTestSuite) TestNew() {
	tests := []struct {
		name         string
		detect       string
		real         bool // New, on the real host
		validateFunc func(uptime.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c uptime.Collector) {
				_, ok := c.(*uptime.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c uptime.Collector) {
				_, ok := c.(*uptime.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c uptime.Collector) {
				_, ok := c.(*uptime.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c uptime.Collector) {
				_, ok := c.(*uptime.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c uptime.Collector) {
				_, ok := c.(*uptime.Linux)
				s.True(ok)
			},
		},
		{
			name: "New reads the real host",
			real: true,
			validateFunc: func(c uptime.Collector) {
				s.NotNil(c)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := uptime.NewWith(source.Sources{Family: tt.detect, FS: memfs.New()})
			if tt.real {
				c = uptime.New()
			}
			s.Equal("uptime", c.Name())
			s.Equal("system", string(c.Category()))
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *UptimePublicTestSuite) TestCollect() {
	now := func() time.Time { return time.Unix(1767322845, 0) }
	tests := []struct {
		name         string
		variant      string
		files        map[string]string
		failRead     []string
		runner       sourcetest.Commands
		nilExec      bool
		validateFunc func(any, error)
	}{
		{
			name:    "linux: uptime, idle and boot time",
			variant: "linux",
			files: map[string]string{
				"/proc/uptime": "93900.42 180001.15\n",
				"/proc/stat":   "cpu  1 2 3\nbtime 1767229145\nprocesses 9\n",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&uptime.Info{
					Seconds: 93900, BootTime: 1767229145, Human: "1d 2h 5m 0s",
					IdleSeconds: 180001, IdleHuman: "2d 2h 0m 1s",
				}, got)
			},
		},
		{
			name:    "linux: no idle field, and no /proc/stat leaves boot_time out",
			variant: "linux",
			files:   map[string]string{"/proc/uptime": "65\n"},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&uptime.Info{Seconds: 65, Human: "1m 5s"}, got)
			},
		},
		{
			name:    "linux: a /proc/stat with no btime leaves boot_time out",
			variant: "linux",
			files: map[string]string{
				"/proc/uptime": "65\n",
				"/proc/stat":   "cpu  1 2 3\n",
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&uptime.Info{Seconds: 65, Human: "1m 5s"}, got)
			},
		},
		{
			name:    "linux: a btime that is not a number keeps the uptime, with the error",
			variant: "linux",
			files: map[string]string{
				"/proc/uptime": "65 x\n",
				"/proc/stat":   "btime soon\n",
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "/proc/stat: btime:")
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal(uint64(65), info.Seconds)
				s.Zero(info.BootTime)
			},
		},
		{
			name:     "linux: a /proc/stat that does not read keeps the uptime, with the error",
			variant:  "linux",
			files:    map[string]string{"/proc/uptime": "93900.42 180001.15\n"},
			failRead: []string{"/proc/stat"},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "read /proc/stat")
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Equal("1d 2h 5m 0s", info.Human)
				s.Equal(uint64(180001), info.IdleSeconds)
				s.Zero(info.BootTime)
			},
		},
		{name: "linux: no /proc/uptime", variant: "linux", validateFunc: func(got any, err error) {
			s.ErrorContains(err, "read /proc/uptime")
			s.Exactly(nil, got)
		}},
		{
			name: "linux: empty /proc/uptime", variant: "linux",
			files: map[string]string{"/proc/uptime": ""}, validateFunc: func(got any, err error) {
				s.ErrorContains(err, "is empty")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an uptime of NaN is refused",
			variant: "linux",
			files: map[string]string{
				"/proc/uptime": "NaN 1",
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "is not a number")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an uptime of +Inf is refused",
			variant: "linux",
			files: map[string]string{
				"/proc/uptime": "+Inf 1",
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "is not a number")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an idle of NaN is left out",
			variant: "linux",
			files:   map[string]string{"/proc/uptime": "65 NaN\n"},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&uptime.Info{Seconds: 65, Human: "1m 5s"}, got)
			},
		},
		{
			name:    "linux: uptime not a number",
			variant: "linux",
			files: map[string]string{
				"/proc/uptime": "x 1",
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "is not a number")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: boot time and the clock",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n kern.boottime": "{ sec = 1767229145, usec = 0 } Thu Jan  1 00:59:05 2026\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(
					&uptime.Info{Seconds: 93700, BootTime: 1767229145, Human: "1d 2h 1m 40s"},
					got,
				)
			},
		},
		{
			name:    "darwin: a boot time after the clock",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n kern.boottime": "{ sec = 1999999999, usec = 0 }\n",
			}},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Equal(&uptime.Info{Seconds: 0, BootTime: 1999999999, Human: "0s"}, got)
			},
		},
		{
			name:    "darwin: a boot time too large for an int64 is an error",
			variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{
				"sysctl -n kern.boottime": "{ sec = 99999999999999999999, usec = 0 }\n",
			}},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "sysctl kern.boottime:")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "darwin: sysctl fails",
			variant: "darwin",
			runner: sourcetest.Commands{
				Err: map[string]error{"sysctl -n kern.boottime": errors.New("boom")},
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "sysctl kern.boottime: boom")
				s.Exactly(nil, got)
			},
		},
		{
			name: "darwin: no Exec gives no fact", variant: "darwin",
			nilExec: true,
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				s.Exactly(nil, got)
			},
		},
		{
			name: "darwin: no seconds", variant: "darwin",
			runner: sourcetest.Commands{Out: map[string]string{"sysctl -n kern.boottime": "nope"}},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "no seconds")
				s.Exactly(nil, got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c uptime.Collector
			switch tt.variant {
			case "linux":
				f := sourcetest.FS(s.T(), tt.files)
				c = &uptime.Linux{FS: sourcetest.FailRead(s.T(), f, tt.failRead...)}
			case "darwin":
				d := &uptime.Darwin{Exec: tt.runner.Mock(s.T()), Now: now}
				if tt.nilExec {
					d.Exec = nil
				}
				c = d
			}
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}

func (s *UptimePublicTestSuite) TestHumanDuration() {
	tests := []struct {
		name         string
		seconds      uint64
		validateFunc func(string)
	}{
		{
			name:    "zero seconds",
			seconds: 0,
			validateFunc: func(got string) {
				s.Equal("0s", got)
			},
		},
		{
			name:    "seconds only",
			seconds: 45,
			validateFunc: func(got string) {
				s.Equal("45s", got)
			},
		},
		{
			name:    "minutes and seconds",
			seconds: 75,
			validateFunc: func(got string) {
				s.Equal("1m 15s", got)
			},
		},
		{
			name:    "hours/minutes/seconds",
			seconds: 3*3600 + 12*60 + 5,
			validateFunc: func(got string) {
				s.Equal("3h 12m 5s", got)
			},
		},
		{
			name:    "days+hours+minutes+seconds",
			seconds: 2*86400 + 5*3600 + 12*60 + 5,
			validateFunc: func(got string) {
				s.Equal("2d 5h 12m 5s", got)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			tt.validateFunc(uptime.HumanDuration(tt.seconds))
		})
	}
}
