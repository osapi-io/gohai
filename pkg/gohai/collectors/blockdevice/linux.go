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

package blockdevice

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

const sysBlockDir = "/sys/block"

// Linux collects block device sysfs attributes on Linux. FS is the
// virtual filesystem the collector reads from. Production uses the real OS
// FS via avfs/osfs; tests inject an avfs memfs with canned content.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect reads /sys/block and returns one BlockDevice per entry. The
// sysfs directory may be absent on container-only or embedded hosts;
// those return an empty device list without error. A read that fails
// for any other reason is an error.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	entries, err := l.FS.ReadDir(sysBlockDir)
	if collector.Absent(err) {
		// /sys/block is absent: a container or an unsupported host.
		return &Info{Devices: []BlockDevice{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sysBlockDir, err)
	}

	devices := make([]BlockDevice, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		base := filepath.Join(sysBlockDir, name)
		// Each entry of /sys/block is a symbolic link to the device's
		// directory under /sys/devices, so ReadDir does not report it as a
		// directory. Stat follows the link.
		info, err := l.FS.Stat(base)
		if collector.Absent(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", base, err)
		}
		if !info.IsDir() {
			continue
		}
		dev := filepath.Join(base, "device")
		queue := filepath.Join(base, "queue")

		r := sysReader{fs: l.FS}
		bd := BlockDevice{
			Name:              name,
			Size:              r.read(filepath.Join(base, "size")),
			Removable:         r.read(filepath.Join(base, "removable")),
			Rotational:        r.read(filepath.Join(queue, "rotational")),
			PhysicalBlockSize: r.read(filepath.Join(queue, "physical_block_size")),
			LogicalBlockSize:  r.read(filepath.Join(queue, "logical_block_size")),
			Model:             r.read(filepath.Join(dev, "model")),
			Vendor:            r.read(filepath.Join(dev, "vendor")),
			Rev:               r.read(filepath.Join(dev, "rev")),
			State:             r.read(filepath.Join(dev, "state")),
			Timeout:           r.read(filepath.Join(dev, "timeout")),
			QueueDepth:        r.read(filepath.Join(dev, "queue_depth")),
			FirmwareRev:       r.read(filepath.Join(dev, "firmware_rev")),
		}
		if r.err != nil {
			return nil, r.err
		}
		devices = append(devices, bd)
	}

	return &Info{Devices: devices}, nil
}
