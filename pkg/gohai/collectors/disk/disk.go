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

// Package disk collects per-device disk I/O counters (node_exporter-style).
// Ohai has no equivalent plugin — block-device *metadata* lives under
// linux/block_device.rb in Ohai, which we target with a future gohai
// `block_device` collector. This collector is I/O counters only.
package disk

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds per-device disk I/O counters.
type Info struct {
	Devices []Device `json:"devices"`
}

// Device represents I/O counters for a single block device.
type Device struct {
	Name       string `json:"name"`
	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadTime   uint64 `json:"read_time,omitempty"`
	WriteTime  uint64 `json:"write_time,omitempty"`
	IoTime     uint64 `json:"io_time,omitempty"`
}

// Collector is the public interface every disk variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "disk" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the disk variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// ioCountersFn is the injection seam for gopsutil's
// disk.IOCountersWithContext. Kept private so importers don't
// transitively need gopsutil. Swapped via SetIOCountersFn.
var ioCountersFn = disk.IOCountersWithContext

// listIOCounters is the production bridge to gopsutil. Named function
// so factories can assign by reference (no closure body to cover).
func listIOCounters(
	ctx context.Context,
) ([]Device, error) {
	m, err := ioCountersFn(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(m))
	for _, s := range m {
		out = append(out, Device{
			Name:       s.Name,
			ReadCount:  s.ReadCount,
			WriteCount: s.WriteCount,
			ReadBytes:  s.ReadBytes,
			WriteBytes: s.WriteBytes,
			ReadTime:   s.ReadTime,
			WriteTime:  s.WriteTime,
			IoTime:     s.IoTime,
		})
	}
	return out, nil
}
