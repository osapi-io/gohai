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
	"path/filepath"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

const sysBlockDir = "/sys/block"

// Linux collects block device sysfs attributes on Linux. FS is the
// virtual filesystem the collector reads from — production is the real OS
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
// those return an empty device list without error.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	readFn := func(path string) ([]byte, error) {
		return l.FS.ReadFile(path)
	}

	entries, err := l.FS.ReadDir(sysBlockDir)
	if err != nil {
		// /sys/block absent — container or unsupported host.
		return &Info{Devices: []BlockDevice{}}, nil
	}

	devices := make([]BlockDevice, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		base := filepath.Join(sysBlockDir, name)
		dev := filepath.Join(base, "device")
		queue := filepath.Join(base, "queue")

		bd := BlockDevice{
			Name:              name,
			Size:              readSysFile(readFn, filepath.Join(base, "size")),
			Removable:         readSysFile(readFn, filepath.Join(base, "removable")),
			Rotational:        readSysFile(readFn, filepath.Join(queue, "rotational")),
			PhysicalBlockSize: readSysFile(readFn, filepath.Join(queue, "physical_block_size")),
			LogicalBlockSize:  readSysFile(readFn, filepath.Join(queue, "logical_block_size")),
			Model:             readSysFile(readFn, filepath.Join(dev, "model")),
			Vendor:            readSysFile(readFn, filepath.Join(dev, "vendor")),
			Rev:               readSysFile(readFn, filepath.Join(dev, "rev")),
			State:             readSysFile(readFn, filepath.Join(dev, "state")),
			Timeout:           readSysFile(readFn, filepath.Join(dev, "timeout")),
			QueueDepth:        readSysFile(readFn, filepath.Join(dev, "queue_depth")),
			FirmwareRev:       readSysFile(readFn, filepath.Join(dev, "firmware_rev")),
		}
		devices = append(devices, bd)
	}

	return &Info{Devices: devices}, nil
}
