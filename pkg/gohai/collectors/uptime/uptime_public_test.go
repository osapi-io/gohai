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
			c := uptime.New()
			s.Equal("uptime", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*uptime.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*uptime.Linux)
				s.True(ok)
			}
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
		name    string
		variant string
		hostFn  func(context.Context) (*host.InfoStat, error)
		setupFS func() avfs.VFS
		wantErr bool
		want    uptime.Info
	}{
		{
			name:    "linux: 3h up + idle parsed",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67 9876.54\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
				IdleSeconds: 9876, IdleHuman: "2h 44m 36s",
			},
		},
		{
			name:    "linux: missing /proc/uptime omits idle",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("", false) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "linux: malformed /proc/uptime omits idle",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "linux: unparseable idle field omits",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67 xyz\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "linux: negative idle omits",
			variant: "linux",
			hostFn:  okHost,
			setupFS: func() avfs.VFS { return buildFS("12345.67 -1.0\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "linux: gopsutil error wrapped and returned",
			variant: "linux",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			setupFS: func() avfs.VFS { return buildFS("1 1\n", true) },
			wantErr: true,
		},
		{
			name:    "darwin: uptime returned",
			variant: "darwin",
			hostFn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{Uptime: 7200, BootTime: 1_700_000_000}, nil
			},
			want: uptime.Info{Seconds: 7200, BootTime: 1_700_000_000, Human: "2h 0m 0s"},
		},
		{
			name:    "darwin: gopsutil error propagated",
			variant: "darwin",
			hostFn:  func(context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
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
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*uptime.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
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
