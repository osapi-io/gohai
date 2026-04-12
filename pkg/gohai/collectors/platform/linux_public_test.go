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

package platform_test

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/shirou/gopsutil/v4/host"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

type PlatformLinuxPublicTestSuite struct {
	suite.Suite
}

func TestPlatformLinuxPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformLinuxPublicTestSuite))
}

func (s *PlatformLinuxPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		hostFn  func(context.Context) (*host.InfoStat, error)
		wantErr bool
		want    platform.Info
	}{
		{
			name: "ubuntu reports as ubuntu (no remap)",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "ubuntu",
					PlatformVersion: "24.04",
					PlatformFamily:  "debian",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "ubuntu", Version: "24.04",
				Family: "debian", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "amzn remaps to amazon",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "amzn",
					PlatformVersion: "2023",
					PlatformFamily:  "rhel",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "amazon", Version: "2023",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "rhel remaps to redhat",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "rhel",
					PlatformVersion: "9.4",
					PlatformFamily:  "rhel",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "redhat", Version: "9.4",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "sles remaps to suse",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "sles",
					PlatformVersion: "15-SP5",
					PlatformFamily:  "suse",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "suse", Version: "15-SP5",
				Family: "suse", Architecture: runtime.GOARCH,
			},
		},
		{
			name: "ol remaps to oracle",
			hostFn: func(_ context.Context) (*host.InfoStat, error) {
				return &host.InfoStat{
					Platform:        "ol",
					PlatformVersion: "8.9",
					PlatformFamily:  "rhel",
				}, nil
			},
			want: platform.Info{
				OS: runtime.GOOS, Name: "oracle", Version: "8.9",
				Family: "rhel", Architecture: runtime.GOARCH,
			},
		},
		{
			name:   "nil info yields minimal Info",
			hostFn: func(_ context.Context) (*host.InfoStat, error) { return nil, nil },
			want:   platform.Info{OS: runtime.GOOS, Architecture: runtime.GOARCH},
		},
		{
			name:    "host.Info error propagated",
			hostFn:  func(_ context.Context) (*host.InfoStat, error) { return nil, errors.New("boom") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			c := &platform.Linux{HostInfoFn: tt.hostFn}
			got, err := c.Collect(context.Background())
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*platform.Info)
			s.Require().True(ok)
			s.Equal(tt.want, *info)
		})
	}
}

func (s *PlatformLinuxPublicTestSuite) TestNewLinuxWiresUp() {
	c := platform.NewLinux()
	s.NotNil(c.HostInfoFn)
}
