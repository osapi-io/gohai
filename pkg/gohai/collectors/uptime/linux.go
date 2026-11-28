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

package uptime

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
)

const (
	// procUptimePath has two fields: the seconds since boot, and the idle
	// seconds summed over the CPUs.
	procUptimePath = "/proc/uptime"
	// procStatPath has btime, the boot time as a unix timestamp.
	procStatPath = "/proc/stat"
)

// Linux collects uptime and idle time from /proc/uptime, and the boot time
// from /proc/stat.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant that reads the host's filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns uptime facts.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	b, err := l.FS.ReadFile(procUptimePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procUptimePath, err)
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return nil, fmt.Errorf("%s is empty", procUptimePath)
	}
	up, err := strconv.ParseFloat(fields[0], 64)
	// NaN and +Inf parse as floats, and their uint64 conversion is not
	// defined, so they are refused with a negative value.
	if err != nil || up < 0 || math.IsNaN(up) || math.IsInf(up, 0) {
		return nil, fmt.Errorf(
			"%s: uptime %q is not a number of seconds",
			procUptimePath,
			fields[0],
		)
	}
	out := &Info{Seconds: uint64(up), Human: HumanDuration(uint64(up))}
	out.BootTime = readBootTime(l.FS)
	if len(fields) > 1 {
		if idle, err := strconv.ParseFloat(fields[1], 64); err == nil && idle >= 0 &&
			!math.IsNaN(idle) && !math.IsInf(idle, 0) {
			out.IdleSeconds = uint64(idle)
			out.IdleHuman = HumanDuration(out.IdleSeconds)
		}
	}

	return out, nil
}

// readBootTime is the btime line of /proc/stat, or 0 when it cannot be
// read.
func readBootTime(
	fs avfs.VFS,
) uint64 {
	b, err := fs.ReadFile(procStatPath)
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
			if err == nil {
				return n
			}
		}
	}

	return 0
}
