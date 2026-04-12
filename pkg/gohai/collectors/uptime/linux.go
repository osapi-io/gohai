//go:build linux

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

var (
	hostInfoFn       = host.InfoWithContext
	readProcUptimeFn = func() ([]byte, error) { return os.ReadFile("/proc/uptime") }
)

func collect(
	ctx context.Context,
) (any, error) {
	return collectWithHost(ctx, hostInfoFn, readProcUptimeFn)
}

func collectWithHost(
	ctx context.Context,
	fn func(context.Context) (*host.InfoStat, error),
	readProcUptime func() ([]byte, error),
) (any, error) {
	info, err := fn(ctx)
	if err != nil {
		return nil, fmt.Errorf("host.Info: %w", err)
	}
	out := &Info{
		Seconds:  info.Uptime,
		BootTime: info.BootTime,
		Human:    HumanDuration(info.Uptime),
	}
	if idle, ok := readIdleSeconds(readProcUptime); ok {
		out.IdleSeconds = idle
		out.IdleHuman = HumanDuration(idle)
	}
	return out, nil
}

// readIdleSeconds parses the second field of /proc/uptime, which records
// the aggregate seconds all CPU cores have spent idle since boot. On
// multi-core systems this can exceed wall-clock uptime (it's summed
// across cores). Returns (0, false) if /proc/uptime is unreadable or
// malformed — idle is a best-effort signal.
func readIdleSeconds(
	readProcUptime func() ([]byte, error),
) (uint64, bool) {
	b, err := readProcUptime()
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
