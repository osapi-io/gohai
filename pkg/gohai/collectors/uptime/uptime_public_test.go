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
	"io/fs"
	"testing"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
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
	orig := platform.Detect
	defer func() { platform.Detect = orig }()

	tests := []struct {
		name         string
		detect       string
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
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := uptime.New()
			s.Equal("uptime", c.Name())
			s.Equal("system", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *UptimePublicTestSuite) TestCollect() {
	okHost := func(context.Context) (*host.InfoStat, error) {
		return &host.InfoStat{Uptime: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000}, nil
	}
	buildFS := func(content string, writeFile bool) avfs.VFS {
		f := memfs.New()
		if writeFile {
			_ = f.MkdirAll("/proc", 0o755)
			_ = f.WriteFile("/proc/uptime", []byte(content), fs.FileMode(0o644))
		}
		return f
	}

	tests := []struct {
		name         string
		variant      string
		hostFn       func(context.Context) (*host.InfoStat, error)
		setupFS      func() avfs.VFS
		validateFunc func(any, error)
	}{
		{
			name:    "linux: 3h up + idle parsed",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67 9876.54\n", true) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Equal(uptime.Info{
					Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
					IdleSeconds: 9876, IdleHuman: "2h 44m 36s",
				}, *info)
			},
		},
		{
			name:    "linux: missing /proc/uptime omits idle",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("", false) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Equal(uptime.Info{
					Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
				}, *info)
			},
		},
		{
			name:    "linux: malformed /proc/uptime omits idle",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67\n", true) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Equal(uptime.Info{
					Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
				}, *info)
			},
		},
		{
			name:    "linux: unparseable idle field omits",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67 xyz\n", true) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Equal(uptime.Info{
					Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
				}, *info)
			},
		},
		{
			name:    "linux: negative idle omits",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67 -1.0\n", true) },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Equal(uptime.Info{
					Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
				}, *info)
			},
		},
		{
			name:    "linux: gopsutil error wrapped and returned",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			setupFS: func() avfs.VFS { return buildFS("1 1\n", true) },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
		{
			name:    "darwin: uptime returned",
			variant: "darwin",
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{Uptime: 7200, BootTime: 1_700_000_000}, nil
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*uptime.Info)
				s.Require().True(ok)
				s.Equal(
					uptime.Info{Seconds: 7200, BootTime: 1_700_000_000, Human: "2h 0m 0s"},
					*info,
				)
			},
		},
		{
			name:    "darwin: gopsutil error propagated",
			variant: "darwin",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			validateFunc: func(_ any, err error) {
				s.Error(err)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer uptime.SetHostInfoFn(tt.hostFn)()
			var c uptime.Collector
			switch tt.variant {
			case "linux":
				c = &uptime.Linux{FS: tt.setupFS()}
			case "darwin":
				c = &uptime.Darwin{}
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
