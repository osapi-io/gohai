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
	"github.com/osapi-io/gohai/internal/sourcetest"
	"github.com/osapi-io/gohai/pkg/gohai/collectors/blockdevice"
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
		name         string
		detect       string
		validateFunc func(blockdevice.Collector)
	}{
		{
			name:   "darwin dispatches to Darwin",
			detect: "darwin",
			validateFunc: func(c blockdevice.Collector) {
				_, ok := c.(*blockdevice.Darwin)
				s.True(ok)
			},
		},
		{
			name:   "debian dispatches to Linux",
			detect: "debian",
			validateFunc: func(c blockdevice.Collector) {
				_, ok := c.(*blockdevice.Linux)
				s.True(ok)
			},
		},
		{
			name:   "rhel dispatches to Linux",
			detect: "rhel",
			validateFunc: func(c blockdevice.Collector) {
				_, ok := c.(*blockdevice.Linux)
				s.True(ok)
			},
		},
		{
			name:   "arch dispatches to Linux",
			detect: "arch",
			validateFunc: func(c blockdevice.Collector) {
				_, ok := c.(*blockdevice.Linux)
				s.True(ok)
			},
		},
		{
			name:   "unknown dispatches to Linux",
			detect: "",
			validateFunc: func(c blockdevice.Collector) {
				_, ok := c.(*blockdevice.Linux)
				s.True(ok)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			platform.Detect = func() string { return tt.detect }
			c := blockdevice.New()
			s.Equal("block_device", c.Name())
			s.Equal("linux", string(c.Category()))
			s.False(c.DefaultEnabled())
			s.Empty(c.Dependencies())
			tt.validateFunc(c)
		})
	}
}

func (s *BlockDevicePublicTestSuite) TestCollect() {
	tests := []struct {
		name         string
		variant      string
		setupFS      func() *memfs.MemFS
		validateFunc func(any, error)
	}{
		{
			name:    "linux: single disk with full sysfs tree",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices/block/sda/device", 0o755)
				_ = f.MkdirAll("/sys/devices/block/sda/queue", 0o755)
				_ = f.WriteFile(
					"/sys/devices/block/sda/size",
					[]byte("976773168\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/removable",
					[]byte("0\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/queue/rotational",
					[]byte("1\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/queue/physical_block_size",
					[]byte("512\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/queue/logical_block_size",
					[]byte("512\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/device/model",
					[]byte("WDC WD5000AAKX\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/device/vendor",
					[]byte("ATA     \n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/device/rev",
					[]byte("1H15\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/device/state",
					[]byte("running\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/device/timeout",
					[]byte("30\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/sda/device/queue_depth",
					[]byte("32\n"),
					fs.FileMode(0o444),
				)
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sda": "/sys/devices/block/sda",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{
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
				}, info.Devices)
			},
		},
		{
			name:    "linux: multiple devices sorted by readdir order",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices/block/nvme0n1/queue", 0o755)
				_ = f.WriteFile(
					"/sys/devices/block/nvme0n1/size",
					[]byte("1000215216\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/nvme0n1/removable",
					[]byte("0\n"),
					fs.FileMode(0o444),
				)
				_ = f.WriteFile(
					"/sys/devices/block/nvme0n1/queue/rotational",
					[]byte("0\n"),
					fs.FileMode(0o444),
				)
				_ = f.MkdirAll("/sys/devices/block/sda/queue", 0o755)
				_ = f.WriteFile(
					"/sys/devices/block/sda/size",
					[]byte("500107862\n"),
					fs.FileMode(0o444),
				)
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/nvme0n1": "/sys/devices/block/nvme0n1",
					"/sys/block/sda":     "/sys/devices/block/sda",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{
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
				}, info.Devices)
			},
		},
		{
			name:    "linux: /sys/block absent returns empty list",
			variant: "linux",
			setupFS: func() *memfs.MemFS { return memfs.New() },
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{}, info.Devices)
			},
		},
		{
			name:    "linux: device with no optional sysfs files",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices/block/vda", 0o755)
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/vda": "/sys/devices/block/vda",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{
					{Name: "vda"},
				}, info.Devices)
			},
		},
		{
			name:    "linux: firmware_rev populated",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices/block/sdb/device", 0o755)
				_ = f.WriteFile(
					"/sys/devices/block/sdb/device/firmware_rev",
					[]byte("GXA2\n"),
					fs.FileMode(0o444),
				)
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sdb": "/sys/devices/block/sdb",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{
					{Name: "sdb", FirmwareRev: "GXA2"},
				}, info.Devices)
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
				_ = f.MkdirAll("/sys/devices/block/sdc", 0o755)
				_ = f.WriteFile("/sys/devices/block/sdc/size", []byte("1024\n"), fs.FileMode(0o444))
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sdc": "/sys/devices/block/sdc",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{
					{Name: "sdc", Size: "1024"},
				}, info.Devices)
			},
		},
		{
			name:    "linux: a link in /sys/block to a missing device is skipped",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices/block/sdc", 0o755)
				_ = f.WriteFile("/sys/devices/block/sdc/size", []byte("1024\n"), fs.FileMode(0o444))
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sdc": "/sys/devices/block/sdc",
					"/sys/block/sdd": "/sys/devices/block/gone",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.Require().NoError(err)
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Equal([]blockdevice.BlockDevice{
					{Name: "sdc", Size: "1024"},
				}, info.Devices)
			},
		},
		{
			name:    "linux: a /sys/block that cannot be read is an error",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys", 0o755)
				_ = f.WriteFile("/sys/block", []byte("x"), fs.FileMode(0o444))
				return f
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "read /sys/block: ")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: a device that cannot be stat is an error",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices", 0o755)
				_ = f.WriteFile("/sys/devices/block", []byte("x"), fs.FileMode(0o444))
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sde": "/sys/devices/block/sde",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "stat /sys/block/sde: ")
				s.Exactly(nil, got)
			},
		},
		{
			name:    "linux: an attribute that cannot be read keeps the device and the others",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices/block/sdf/device", 0o755)
				_ = f.WriteFile("/sys/devices/block/sdf/size", []byte("1024\n"), fs.FileMode(0o444))
				sourcetest.FailRead(s.T(), f, "/sys/devices/block/sdf/removable")
				_ = f.MkdirAll("/sys/devices/block/sdg/device", 0o755)
				_ = f.WriteFile("/sys/devices/block/sdg/size", []byte("2048\n"), fs.FileMode(0o444))
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sdf": "/sys/devices/block/sdf",
					"/sys/block/sdg": "/sys/devices/block/sdg",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "read /sys/block/sdf/removable: ")
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Require().Len(info.Devices, 2)
				s.Equal("1024", info.Devices[0].Size)
				s.Equal("2048", info.Devices[1].Size)
			},
		},
		{
			name:    "linux: a device that cannot be stat keeps the others",
			variant: "linux",
			setupFS: func() *memfs.MemFS {
				f := memfs.New()
				_ = f.MkdirAll("/sys/devices", 0o755)
				_ = f.WriteFile("/sys/devices/block", []byte("x"), fs.FileMode(0o444))
				_ = f.MkdirAll("/sys/devices/disk/sdh/device", 0o755)
				_ = f.WriteFile("/sys/devices/disk/sdh/size", []byte("4096\n"), fs.FileMode(0o444))
				sourcetest.Links(s.T(), f, map[string]string{
					"/sys/block/sde": "/sys/devices/block/sde",
					"/sys/block/sdh": "/sys/devices/disk/sdh",
				})
				return f
			},
			validateFunc: func(got any, err error) {
				s.ErrorContains(err, "stat /sys/block/sde: ")
				info, ok := got.(*blockdevice.Info)
				s.Require().True(ok)
				s.Require().NotNil(info)
				s.Require().Len(info.Devices, 1)
				s.Equal("sdh", info.Devices[0].Name)
			},
		},
		{
			name:    "darwin: unsupported",
			variant: "darwin",
			validateFunc: func(got any, err error) {
				s.Require().ErrorIs(err, collector.ErrUnsupported)
				s.Exactly(nil, got)
			},
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
			tt.validateFunc(c.Collect(context.Background(), nil))
		})
	}
}
