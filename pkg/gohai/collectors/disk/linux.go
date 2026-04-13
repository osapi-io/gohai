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

import "context"

// Linux collects disk I/O counters on Linux. gopsutil's
// disk.IOCountersWithContext (/proc/diskstats) is swapped via the
// package-level ioCountersFn seam.
type Linux struct {
	base
}

// NewLinux returns a Linux variant.
func NewLinux() *Linux {
	return &Linux{}
}

// Collect returns per-device I/O counters.
func (l *Linux) Collect(
	ctx context.Context,
) (any, error) {
	devs, err := listIOCounters(ctx)
	if err != nil {
		return nil, err
	}
	return &Info{Devices: devs}, nil
}
