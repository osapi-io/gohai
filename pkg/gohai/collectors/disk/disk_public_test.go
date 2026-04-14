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

package disk_test

import (
	"context"
	"errors"
	"testing"

	gpdisk "github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/disk"
)

var (
	_ collector.Collector = (*disk.Linux)(nil)
	_ collector.Collector = (*disk.Darwin)(nil)
)

type DiskPublicTestSuite struct {
	suite.Suite
}

func TestDiskPublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(DiskPublicTestSuite))
}

func (s *DiskPublicTestSuite) TestNew() {
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
			c := disk.New()
			s.Equal("disk", c.Name())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*disk.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*disk.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *DiskPublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		fn      func(context.Context, ...string) (map[string]gpdisk.IOCountersStat, error)
		wantErr bool
		wantLen int
	}{
		{
			name:    "linux: sda snapshot",
			variant: "linux",
			fn: func(context.Context, ...string) (map[string]gpdisk.IOCountersStat, error) {
				return map[string]gpdisk.IOCountersStat{
					"sda": {
						Name:       "sda",
						ReadCount:  100,
						WriteCount: 50,
						ReadBytes:  102400,
						WriteBytes: 51200,
					},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:    "linux: empty devices",
			variant: "linux",
			fn: func(context.Context, ...string) (map[string]gpdisk.IOCountersStat, error) {
				return map[string]gpdisk.IOCountersStat{}, nil
			},
			wantLen: 0,
		},
		{
			name:    "linux: gopsutil error propagated",
			variant: "linux",
			fn: func(context.Context, ...string) (map[string]gpdisk.IOCountersStat, error) {
				return nil, errors.New("boom")
			},
			wantErr: true,
		},
		{
			name:    "darwin: disk0 snapshot",
			variant: "darwin",
			fn: func(context.Context, ...string) (map[string]gpdisk.IOCountersStat, error) {
				return map[string]gpdisk.IOCountersStat{
					"disk0": {Name: "disk0", ReadCount: 200, WriteCount: 100},
				}, nil
			},
			wantLen: 1,
		},
		{
			name:    "darwin: iokit error propagated",
			variant: "darwin",
			fn: func(context.Context, ...string) (map[string]gpdisk.IOCountersStat, error) {
				return nil, errors.New("iokit unavailable")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer disk.SetIOCountersFn(tt.fn)()
			var c disk.Collector
			switch tt.variant {
			case "linux":
				c = &disk.Linux{}
			case "darwin":
				c = &disk.Darwin{}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.Require().NoError(err)
			info, ok := got.(*disk.Info)
			s.Require().True(ok)
			s.Len(info.Devices, tt.wantLen)
		})
	}
}
