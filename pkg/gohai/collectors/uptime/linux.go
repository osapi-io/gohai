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
	"os"
	"strconv"
	"strings"
)

// procUptimePath is /proc/uptime. Two fields: uptime seconds and
// aggregate idle seconds across all CPUs.
const procUptimePath = "/proc/uptime"

// Linux collects uptime + idle on Linux. BaseFn is typed in our *Info
// so importers don't need gopsutil; ReadFileFn is stdlib.
type Linux struct {
	base

	BaseFn     func(context.Context) (*Info, error)
	ReadFileFn func(string) ([]byte, error)
}

// NewLinux returns a Linux variant wired to the production bridges.
func NewLinux() *Linux {
	return &Linux{
		BaseFn:     readBase,
		ReadFileFn: os.ReadFile,
	}
}

// Collect returns uptime facts. Uses BaseFn for Seconds/BootTime and
// layers idle on top via /proc/uptime.
func (l *Linux) Collect(ctx context.Context) (any, error) {
	out, err := l.BaseFn(ctx)
	if err != nil {
		return nil, err
	}
	if idle, ok := readIdleSeconds(l.ReadFileFn); ok {
		out.IdleSeconds = idle
		out.IdleHuman = HumanDuration(idle)
	}
	return out, nil
}

// readIdleSeconds parses the second field of /proc/uptime (aggregate
// idle seconds across all CPU cores). Returns (0, false) on any failure
// — idle is best-effort.
func readIdleSeconds(
	read func(string) ([]byte, error),
) (uint64, bool) {
	b, err := read(procUptimePath)
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(b))
	if len(fields) < 2 {
		return 0, false
	}
	f, err := strconv.ParseFloat(fields[1], 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return uint64(f), true
}
