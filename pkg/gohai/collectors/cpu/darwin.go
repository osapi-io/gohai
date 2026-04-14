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

package cpu

import (
	"context"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Darwin collects CPU facts on macOS. gopsutil's cpu.Info (via the
// readCPU bridge) sources model / vendor / flags and the logical
// thread count correctly but gets physical cores
// (reports logical) and frequency (zero on Apple Silicon) wrong. We
// override those via direct sysctl reads through the injected Exec.
type Darwin struct {
	base

	Exec executor.Executor
}

// NewDarwin returns a Darwin variant wired to production dependencies.
func NewDarwin() *Darwin {
	return &Darwin{Exec: executor.New()}
}

// Collect returns the CPU Info with macOS-specific sysctl overrides
// merged on top of the gopsutil base:
//
//   - hw.physicalcpu   → Cores (gopsutil reports logical here).
//   - hw.packages      → Sockets.
//   - hw.cpufrequency_max / hw.cpufrequency → Mhz (both absent on
//     Apple Silicon; Mhz left as whatever gopsutil returned — usually 0
//     there).
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info, err := readCPU(ctx)
	if err != nil {
		return nil, err
	}
	if n, ok := sysctlInt(ctx, d.Exec, "hw.physicalcpu"); ok {
		info.Cores = n
	}
	if n, ok := sysctlInt(ctx, d.Exec, "hw.packages"); ok {
		info.Sockets = n
	}
	if n, ok := sysctlInt64(ctx, d.Exec, "hw.cpufrequency_max"); ok {
		info.Mhz = float64(n) / 1_000_000
	} else if n, ok := sysctlInt64(ctx, d.Exec, "hw.cpufrequency"); ok {
		info.Mhz = float64(n) / 1_000_000
	}
	return info, nil
}

// sysctlInt runs `sysctl -n <key>` and parses the output as int.
// Returns (0, false) on any exec error or parse failure.
func sysctlInt(
	ctx context.Context,
	exec executor.Executor,
	key string,
) (int, bool) {
	out, err := exec.Execute(ctx, "sysctl", "-n", key)
	if err != nil {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, false
	}
	return n, true
}

// sysctlInt64 runs `sysctl -n <key>` and parses the output as int64 —
// used for hw.cpufrequency which exceeds int32 on modern CPUs.
func sysctlInt64(
	ctx context.Context,
	exec executor.Executor,
	key string,
) (int64, bool) {
	out, err := exec.Execute(ctx, "sysctl", "-n", key)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
