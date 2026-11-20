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

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	"github.com/osapi-io/gohai/internal/sourcetest"
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
		name         string
		detect       string
		validateFunc func(disk.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c disk.Collector) {
				_, ok := c.(*disk.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c disk.Collector) {
				_, ok := c.(*disk.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c disk.Collector) {
				_, ok := c.(*disk.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c disk.Collector) {
				_, ok := c.(*disk.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c disk.Collector) {
				_, ok := c.(*disk.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := disk.New()
			s.Equal("disk", c.Name())
			s.Equal("hardware", c.Category())
			s.True(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *DiskPublicTestSuite) TestCollect() {
	const ioregLine = "ioreg -r -c IOBlockStorageDriver -d 2 -l -w0"
	const ioreg = `+-o IOBlockStorageDriver  <class IOBlockStorageDriver, id 0x1>
      "Statistics" = {"Operations (Read)"=0,"Bytes (Read)"=0}
+-o IOBlockStorageDriver  <class IOBlockStorageDriver, id 0x2>
  |   "Statistics" = {"Operations (Write)"=20,"Bytes (Read)"=4096,"Total Time (Read)"=3000000,` +
		`"Total Time (Write)"=5000000,"Bytes (Write)"=8192,"Operations (Read)"=10}
  +-o APPLE SSD Media  <class IOMedia, id 0x3>
        "BSD Name" = "disk0"
        "BSD Name" = "disk0s1"
`
	tests := []struct {
		name    string
		variant string
		files   map[string]string
		runner  sourcetest.Runner
		want    *disk.Info
		wantErr string
	}{
		{
			name:    "linux: /proc/diskstats, in file order",
			variant: "linux",
			files: map[string]string{"/proc/diskstats": "" +
				"   8       0 sda 100 5 2000 30 50 6 4000 70 0 90 100 0 0 0 0\n" +
				"   8       1 sda1 1 0 8 1 2 0 16 2 0 3 3\n" +
				"short line\n"},
			want: &disk.Info{Devices: []disk.Device{
				{
					Name: "sda", ReadCount: 100, ReadBytes: 2000 * 512, ReadTime: 30,
					WriteCount: 50, WriteBytes: 4000 * 512, WriteTime: 70, IoTime: 90,
				},
				{
					Name: "sda1", ReadCount: 1, ReadBytes: 8 * 512, ReadTime: 1,
					WriteCount: 2, WriteBytes: 16 * 512, WriteTime: 2, IoTime: 3,
				},
			}},
		},
		{name: "linux: no /proc/diskstats", variant: "linux", wantErr: "read /proc/diskstats"},
		{
			name:    "darwin: drivers with media, times in ms",
			variant: "darwin",
			runner:  sourcetest.Runner{Out: map[string]string{ioregLine: ioreg}},
			want: &disk.Info{Devices: []disk.Device{{
				Name: "disk0", ReadCount: 10, WriteCount: 20, ReadBytes: 4096, WriteBytes: 8192,
				ReadTime: 3, WriteTime: 5, IoTime: 8,
			}}},
		},
		{
			name:    "darwin: ioreg fails",
			variant: "darwin",
			runner:  sourcetest.Runner{Err: map[string]error{ioregLine: errors.New("boom")}},
			wantErr: "ioreg IOBlockStorageDriver: boom",
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c disk.Collector
			switch tt.variant {
			case "linux":
				c = &disk.Linux{FS: sourcetest.FS(s.T(), tt.files)}
			case "darwin":
				c = &disk.Darwin{Exec: tt.runner}
			}
			got, err := c.Collect(context.Background(), nil)
			if tt.wantErr != "" {
				s.ErrorContains(err, tt.wantErr)

				return
			}
			s.Require().NoError(err)
			s.Equal(tt.want, got)
		})
	}
}
