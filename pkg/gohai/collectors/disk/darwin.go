//go:build darwin

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

package disk

import (
	"context"
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/v4/disk"
)

var ioCountersFn = disk.IOCountersWithContext

func collect(
	ctx context.Context,
) (any, error) {
	return collectFromGopsutil(ctx, ioCountersFn)
}

func collectFromGopsutil(
	ctx context.Context,
	fn func(context.Context, ...string) (map[string]disk.IOCountersStat, error),
) (any, error) {
	counters, err := fn(ctx)
	if err != nil {
		return nil, fmt.Errorf("disk.IOCounters: %w", err)
	}
	names := make([]string, 0, len(counters))
	for n := range counters {
		names = append(names, n)
	}
	sort.Strings(names)
	devices := make([]Device, 0, len(names))
	for _, n := range names {
		c := counters[n]
		devices = append(devices, Device{
			Name:       c.Name,
			ReadCount:  c.ReadCount,
			WriteCount: c.WriteCount,
			ReadBytes:  c.ReadBytes,
			WriteBytes: c.WriteBytes,
			ReadTime:   c.ReadTime,
			WriteTime:  c.WriteTime,
			IoTime:     c.IoTime,
		})
	}
	return &Info{Devices: devices}, nil
}
