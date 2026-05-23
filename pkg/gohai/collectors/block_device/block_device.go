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

// Package blockdevice reports block device sysfs attributes for each
// device found under /sys/block.
package blockdevice

import (
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// BlockDevice holds the sysfs attributes for a single block device.
type BlockDevice struct {
	// Name is the kernel device name (e.g. sda, nvme0n1, vda).
	Name string `json:"name"`
	// Size is the device capacity in 512-byte sectors as reported by
	// /sys/block/<dev>/size.
	Size string `json:"size,omitempty"`
	// Removable is "0" or "1" from /sys/block/<dev>/removable.
	Removable string `json:"removable,omitempty"`
	// Rotational is "0" or "1" from /sys/block/<dev>/queue/rotational.
	// "0" means SSD/NVMe; "1" means spinning disk.
	Rotational string `json:"rotational,omitempty"`
	// PhysicalBlockSize is the physical sector size in bytes from
	// /sys/block/<dev>/queue/physical_block_size.
	PhysicalBlockSize string `json:"physical_block_size,omitempty"`
	// LogicalBlockSize is the logical sector size in bytes from
	// /sys/block/<dev>/queue/logical_block_size.
	LogicalBlockSize string `json:"logical_block_size,omitempty"`
	// Model is the device model string from /sys/block/<dev>/device/model.
	Model string `json:"model,omitempty"`
	// Vendor is the device vendor string from /sys/block/<dev>/device/vendor.
	Vendor string `json:"vendor,omitempty"`
	// Rev is the firmware revision from /sys/block/<dev>/device/rev.
	Rev string `json:"rev,omitempty"`
	// State is the device state from /sys/block/<dev>/device/state.
	State string `json:"state,omitempty"`
	// Timeout is the SCSI command timeout from /sys/block/<dev>/device/timeout.
	Timeout string `json:"timeout,omitempty"`
	// QueueDepth is the SCSI queue depth from /sys/block/<dev>/device/queue_depth.
	QueueDepth string `json:"queue_depth,omitempty"`
	// FirmwareRev is the alternate firmware revision field from
	// /sys/block/<dev>/device/firmware_rev.
	FirmwareRev string `json:"firmware_rev,omitempty"`
}

// Info holds the list of block devices discovered from /sys/block.
type Info struct {
	Devices []BlockDevice `json:"devices"`
}

// Collector is the public interface every block_device variant satisfies.
type Collector interface {
	collector.Collector
}

// base holds the fields every OS variant has in common.
type base struct{}

// Name returns "block_device".
func (base) Name() string { return "block_device" }

// Category returns "linux".
func (base) Category() string { return collector.CategoryLinux }

// DefaultEnabled returns false — block_device is opt-in.
func (base) DefaultEnabled() bool { return false }

// Dependencies returns no dependencies.
func (base) Dependencies() []string { return nil }

// New returns the block_device collector variant appropriate to the
// detected host OS.
func New() Collector {
	switch platform.Detect() {
	case "darwin":
		return NewDarwin()
	default:
		return NewLinux()
	}
}

// readSysFile reads a sysfs file via the provided readFn and returns its
// trimmed content. Returns empty string on any error (sysfs files may not
// exist for all devices).
func readSysFile(
	readFn func(string) ([]byte, error),
	path string,
) string {
	b, err := readFn(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
