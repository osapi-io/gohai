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
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/host"
)

// procUptimePath is /proc/uptime. Two fields: uptime seconds and
// aggregate idle seconds across all CPUs.
const procUptimePath = "/proc/uptime"

// Linux collects uptime + idle on Linux. Wraps gopsutil.host.Info for
// uptime/boot time and extends with /proc/uptime for aggregate CPU
// idle time (which gopsutil doesn't expose). Matches Ohai's idletime.
type Linux struct {
	base

	// HostInfoFn is gopsutil's host.InfoWithContext.
	HostInfoFn func(context.Context) (*host.InfoStat, error)
	// ReadFileFn reads a file. Wired to os.ReadFile; tests inject a
	// stub that returns canned /proc/uptime content.
	ReadFileFn func(string) ([]byte, error)
}

// NewLinux returns a Linux variant wired to gopsutil + os.ReadFile.
// No closures — both injected fields are bare function references, so
// tests cover the wiring without any real file system interaction.
func NewLinux() *Linux {
	return &Linux{
		HostInfoFn: host.InfoWithContext,
		ReadFileFn: os.ReadFile,
	}
}

// Collect returns uptime facts. Uses gopsutil for Seconds/BootTime
// (works on any OS) and our own /proc/uptime parse for IdleSeconds
// (Linux-specific; silently omitted on other OSes since the file
// doesn't exist).
func (l *Linux) Collect(ctx context.Context) (any, error) {
	info, err := l.HostInfoFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Info: %w", err)
	}
	out := &Info{
		Seconds:  info.Uptime,
		BootTime: info.BootTime,
		Human:    HumanDuration(info.Uptime),
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
