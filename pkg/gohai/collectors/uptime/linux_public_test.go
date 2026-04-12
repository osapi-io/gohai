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
	okHost := func(_ context.Context) (*host.InfoStat, error) {
		return &host.InfoStat{Uptime: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000}, nil
	}

	tests := []struct {
		name     string
		hostFn   func(context.Context) (*host.InfoStat, error)
		readFile func(string) ([]byte, error)
		wantErr  bool
		want     uptime.Info
	}{
		{
			name:     "3h up + idle parsed",
			hostFn:   okHost,
			readFile: func(string) ([]byte, error) { return []byte("12345.67 9876.54\n"), nil },
			want: uptime.Info{
				Seconds:     3*3600 + 12*60 + 5,
				BootTime:    1_700_000_000,
				Human:       "3h 12m 5s",
				IdleSeconds: 9876,
				IdleHuman:   "2h 44m 36s",
			},
		},
		{
			name:     "missing /proc/uptime omits idle",
			hostFn:   okHost,
			readFile: func(string) ([]byte, error) { return nil, errors.New("not found") },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:     "malformed /proc/uptime (one field) omits idle",
			hostFn:   okHost,
			readFile: func(string) ([]byte, error) { return []byte("12345.67\n"), nil },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:     "unparseable idle field omits",
			hostFn:   okHost,
			readFile: func(string) ([]byte, error) { return []byte("12345.67 xyz\n"), nil },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:     "negative idle omits",
			hostFn:   okHost,
			readFile: func(string) ([]byte, error) { return []byte("12345.67 -1.0\n"), nil },
			want: uptime.Info{
				Seconds: 3*3600 + 12*60 + 5, BootTime: 1_700_000_000, Human: "3h 12m 5s",
			},
		},
		{
			name:     "host.Info error propagated",
			hostFn:   func(_ context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			readFile: func(string) ([]byte, error) { return []byte("1 1\n"), nil },
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &uptime.Linux{HostInfoFn: tt.hostFn, ReadFileFn: tt.readFile}
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
