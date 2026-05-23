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

package blockdevice_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/avfs/avfs/vfs/memfs"
	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
	blockdevice "github.com/osapi-io/gohai/pkg/gohai/collectors/block_device"
)

var (
	_ collector.Collector = (*blockdevice.Linux)(nil)
	_ collector.Collector = (*blockdevice.Darwin)(nil)
)

type BlockDevicePublicTestSuite struct {
	suite.Suite
}

func TestBlockDevicePublicTestSuite(
	t *testing.T,
) {
	suite.Run(t, new(BlockDevicePublicTestSuite))
}

func (s *BlockDevicePublicTestSuite) TestNew() {
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
			c := blockdevice.New()
			s.Equal("block_device", c.Name())
			s.Equal("linux", c.Category())
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			switch tt.wantKind {
			case "darwin":
				_, ok := c.(*blockdevice.Darwin)
				s.True(ok)
			case "linux":
				_, ok := c.(*blockdevice.Linux)
				s.True(ok)
			}
		})
	}
}

func (s *BlockDevicePublicTestSuite) TestCollect() {
	tests := []struct {
		name    string
		variant string
		setupFS func() *memfs.MemFS
		wantNil bool
		want    []blockdevice.BlockDevice
	}{
		{
			name:    "linux: single disk with full sysfs tree",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/block/sda/device", 0o755)
				_ = f.MkdirAll("/sys/block/sda/queue", 0o755)
				_ = f.WriteFile("/sys/block/sda/size", []byte("976773168\n"), fs.FileMode(0o444))
				_ = f.WriteFile("/sys/block/sda/removable", []byte("0\n"), fs.FileMode(0o444))
				_ = f.WriteFile(
					"/sys/block/sda/queue/rotational",
					[]byte("1\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/block/sda/queue/physical_block_size",
					[]byte("512\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/block/sda/queue/logical_block_size",
					[]byte("512\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/block/sda/device/model",
					[]byte("WDC WD5000AAKX\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/block/sda/device/vendor",
					[]byte("ATA     \n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile("/sys/block/sda/device/rev", []byte("1H15\n"), fs.FileMode(0o444))
				_ = f.WriteFile(
					"/sys/block/sda/device/state",
					[]byte("running\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile("/sys/block/sda/device/timeout", []byte("30\n"), fs.FileMode(0o444))
				_ = f.WriteFile(
					"/sys/block/sda/device/queue_depth",
					[]byte("32\n"),
					fs.FileMode(0o444),
				)
				return f
			},
			want: []blockdevice.BlockDevice{
				{
					Name:              "sda",
					Size:              "976773168",
					Removable:         "0",
					Rotational:        "1",
					PhysicalBlockSize: "512",
					LogicalBlockSize:  "512",
					Model:             "WDC WD5000AAKX",
					Vendor:            "ATA",
					Rev:               "1H15",
					State:             "running",
					Timeout:           "30",
					QueueDepth:        "32",
				},
			},
		},
		{
			name:    "linux: multiple devices sorted by readdir order",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/block/nvme0n1/queue", 0o755)
				_ = f.WriteFile(
					"/sys/block/nvme0n1/size",
					[]byte("1000215216\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile("/sys/block/nvme0n1/removable", []byte("0\n"), fs.FileMode(0o444))
				_ = f.WriteFile(
					"/sys/block/nvme0n1/queue/rotational",
					[]byte("0\n"),
					fs.FileMode(0o444),
				)
				_ = f.MkdirAll("/sys/block/sda/queue", 0o755)
				_ = f.WriteFile("/sys/block/sda/size", []byte("500107862\n"), fs.FileMode(0o444))
				return f
			},
			want: []blockdevice.BlockDevice{
				{
					Name:       "nvme0n1",
					Size:       "1000215216",
					Removable:  "0",
					Rotational: "0",
				},
				{
					Name: "sda",
					Size: "500107862",
				},
			},
		},
		{
			name:    "linux: /sys/block absent returns empty list",
			variant: "linux",
			setupFS: func() *memfs.MemFS { return memfs.New() },
			want:    []blockdevice.BlockDevice{},
		},
		{
			name:    "linux: device with no optional sysfs files",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/block/vda", 0o755)
				return f
			},
			want: []blockdevice.BlockDevice{
				{Name: "vda"},
			},
		},
		{
			name:    "linux: firmware_rev populated",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/block/sdb/device", 0o755)
				_ = f.WriteFile(
					"/sys/block/sdb/device/firmware_rev",
					[]byte("GXA2\n"),
					fs.FileMode(0o444),
				)
				return f
			},
			want: []blockdevice.BlockDevice{
				{Name: "sdb", FirmwareRev: "GXA2"},
			},
		},
		{
			name:    "linux: non-directory entries in /sys/block are skipped",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/block", 0o755)
				// Write a regular file alongside a real device dir.
				_ = f.WriteFile("/sys/block/not_a_device", []byte("x"), fs.FileMode(0o444))
				_ = f.MkdirAll("/sys/block/sdc", 0o755)
				_ = f.WriteFile("/sys/block/sdc/size", []byte("1024\n"), fs.FileMode(0o444))
				return f
			},
			want: []blockdevice.BlockDevice{
				{Name: "sdc", Size: "1024"},
			},
		},
		{
			name:    "darwin returns nil",
			variant: "darwin",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var c blockdevice.Collector
			switch tt.variant {
			case "linux":
				c = &blockdevice.Linux{FS: tt.setupFS()}
			case "darwin":
				c = blockdevice.NewDarwin()
			}
			got, err := c.Collect(context.Background(), nil)
			s.Require().NoError(err)
			if tt.wantNil {
				s.Nil(got)
				return
			}
			info, ok := got.(*blockdevice.Info)
			s.Require().True(ok)
			s.Equal(tt.want, info.Devices)
		})
	}
}
