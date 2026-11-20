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

package disk

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

// procDiskstats has one line of I/O counters for each block device.
const procDiskstats = "/proc/diskstats"

// sectorSize is the unit of diskstats' sector counts: always 512 bytes,
// whatever the device's own sector size.
const sectorSize = 512

// Linux collects I/O counters from /proc/diskstats.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant that reads the host's filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns per-device I/O counters, in the order of
// /proc/diskstats.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	b, err := l.FS.ReadFile(procDiskstats)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procDiskstats, err)
	}
	devs := []Device{}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 14 {
			continue
		}
		n := func(i int) uint64 {
			v, _ := strconv.ParseUint(f[i], 10, 64)
			return v
		}
		// A device that has never read or written, such as an unused
		// loop device, is left out, as gopsutil left it out.
		if n(3) == 0 && n(7) == 0 {
			continue
		}
		devs = append(devs, Device{
			Name:       f[2],
			ReadCount:  n(3),
			ReadBytes:  n(5) * sectorSize,
			ReadTime:   n(6),
			WriteCount: n(7),
			WriteBytes: n(9) * sectorSize,
			WriteTime:  n(10),
			IoTime:     n(12),
		})
	}

	return &Info{Devices: devs}, nil
}
