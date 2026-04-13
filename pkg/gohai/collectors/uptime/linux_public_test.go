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

	"github.com/osapi-io/gohai/pkg/gohai/collectors/uptime"
)

type UptimeLinuxPublicTestSuite struct {
	suite.Suite
}

func TestUptimeLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(UptimeLinuxPublicTestSuite))
}

func (s *UptimeLinuxPublicTestSuite) TestCollect() {
	okBase := func(context.Context) (*uptime.Info, error) {
		return &uptime.Info{
			Seconds:  3*3600 + 12*60 + 5,
			BootTime: 1_700_000_000,
			Human:    "3h 12m 5s",
		}, nil
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
		baseFn  func(context.Context) (*uptime.Info, error)
		setupFS func() avfs.VFS
		wantErr bool
		want    uptime.Info
	}{
		{
			name:    "3h up + idle parsed",
			baseFn:  okBase,
			setupFS: func() avfs.VFS { return buildFS("12345.67 9876.54\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
				IdleSeconds: 9876, IdleHuman: "2h 44m 36s",
			},
		},
		{
			name:    "missing /proc/uptime omits idle",
			baseFn:  okBase,
			setupFS: func() avfs.VFS { return buildFS("", false) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "malformed /proc/uptime omits idle",
			baseFn:  okBase,
			setupFS: func() avfs.VFS { return buildFS("12345.67\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "unparseable idle field omits",
			baseFn:  okBase,
			setupFS: func() avfs.VFS { return buildFS("12345.67 xyz\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "negative idle omits",
			baseFn:  okBase,
			setupFS: func() avfs.VFS { return buildFS("12345.67 -1.0\n", true) },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:    "BaseFn error propagated",
			baseFn:  func(context.Context) (*uptime.Info, error) { return nil, errors.New("boom") },
			setupFS: func() avfs.VFS { return buildFS("1 1\n", true) },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer uptime.SetReadBaseFn(tt.baseFn)()
			c := &uptime.Linux{FS: tt.setupFS()}
			got, err := c.Collect(context.Background())
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

func (s *UptimeLinuxPublicTestSuite) TestReadBase() {
	tests := []struct {
		name    string
		fn      func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    uptime.Info
	}{
		{
			name: "success maps gopsutil InfoStat",
			fn: func(context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{Uptime: 7200, BootTime: 1_700_000_000}, nil
			},
			want: uptime.Info{Seconds: 7200, BootTime: 1_700_000_000, Human: "2h 0m 0s"},
		},
		{
			name:    "gopsutil error wrapped and returned",
			fn:      func(context.Context) (*host.InfoStat, error) { return nil, errors.New("host.Info failed") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			restore := uptime.SetHostInfoFn(tt.fn)
			defer restore()
			got, err := uptime.ReadBase(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, *got)
		})
	}
}
