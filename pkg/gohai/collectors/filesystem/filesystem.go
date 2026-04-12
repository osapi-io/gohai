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

// Package filesystem collects mounted filesystem data with capacity,
// usage, and inode stats.
package filesystem

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds mounted filesystem data.
type Info struct {
	Mounts []Mount `json:"mounts"`
}

// Mount represents a single mounted filesystem.
type Mount struct {
	Device      string   `json:"device"`
	Mountpoint  string   `json:"mountpoint"`
	Fstype      string   `json:"fstype"`
	Opts        []string `json:"opts,omitempty"`
	Total       uint64   `json:"total,omitempty"`
	Used        uint64   `json:"used,omitempty"`
	Free        uint64   `json:"free,omitempty"`
	UsedPercent float64  `json:"used_percent,omitempty"`
	InodesTotal uint64   `json:"inodes_total,omitempty"`
	InodesUsed  uint64   `json:"inodes_used,omitempty"`
	InodesFree  uint64   `json:"inodes_free,omitempty"`
}

// Collector is the public interface every filesystem variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "filesystem" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the filesystem variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// listMounts is the production bridge to gopsutil. Enumerates
// partitions and fetches usage (capacity + inodes) for each. Per-mount
// usage failures (permission denied, stale NFS, etc.) skip usage
// fields but keep the mount in the output.
func listMounts(
	ctx context.Context,
) ([]Mount, error) {
	parts, err := disk.PartitionsWithContext(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]Mount, 0, len(parts))
	for _, p := range parts {
		m := Mount{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			Opts:       p.Opts,
		}
		if u, err := disk.UsageWithContext(ctx, p.Mountpoint); err == nil {
			m.Total = u.Total
			m.Used = u.Used
			m.Free = u.Free
			m.UsedPercent = u.UsedPercent
			m.InodesTotal = u.InodesTotal
			m.InodesUsed = u.InodesUsed
			m.InodesFree = u.InodesFree
		}
		out = append(out, m)
	}
	return out, nil
}
