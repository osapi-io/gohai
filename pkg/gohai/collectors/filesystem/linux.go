//go:build linux

// Copyright (c) 2024 John Dewey

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

package filesystem

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
)

var (
	partitionsFn = disk.PartitionsWithContext
	usageFn      = disk.UsageWithContext
)

func collect(
	ctx context.Context,
) (any, error) {
	return collectFromGopsutil(ctx, partitionsFn, usageFn)
}

func collectFromGopsutil(
	ctx context.Context,
	partFn func(context.Context, bool) ([]disk.PartitionStat, error),
	usageFnArg func(context.Context, string) (*disk.UsageStat, error),
) (any, error) {
	parts, err := partFn(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("disk.Partitions: %w", err)
	}
	mounts := make([]Mount, 0, len(parts))
	for _, p := range parts {
		m := Mount{Device: p.Device, Mountpoint: p.Mountpoint, Fstype: p.Fstype, Opts: p.Opts}
		if u, uerr := usageFnArg(ctx, p.Mountpoint); uerr == nil && u != nil {
			m.Total = u.Total
			m.Used = u.Used
			m.Free = u.Free
			m.UsedPercent = u.UsedPercent
			m.InodesTotal = u.InodesTotal
			m.InodesUsed = u.InodesUsed
			m.InodesFree = u.InodesFree
		}
		mounts = append(mounts, m)
	}
	return &Info{Mounts: mounts}, nil
}
