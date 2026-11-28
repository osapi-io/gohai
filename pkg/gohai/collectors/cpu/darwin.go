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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin collects CPU facts on macOS from sysctl: the model, the vendor,
// the family and the flags from machdep.cpu, the counts from hw. The speed
// is hw.cpufrequency_max or hw.cpufrequency, which Intel Macs have; Apple
// Silicon has neither, so the speed is the top of the performance cores'
// frequency table that IOKit holds, read with ioreg.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's sysctl and
// ioreg.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns the CPU Info: one entry, for the package, as gopsutil
// gave it on macOS. With no Exec it gives no fact, as every value comes
// from sysctl.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	count, ok := sysctlInt(ctx, d.Exec, "hw.logicalcpu")
	if !ok {
		return nil, fmt.Errorf("sysctl hw.logicalcpu failed")
	}
	c := CPU{
		ModelName: sysctlString(ctx, d.Exec, "machdep.cpu.brand_string"),
		VendorID:  sysctlString(ctx, d.Exec, "machdep.cpu.vendor"),
		Family:    "0",
		ModelID:   "0",
	}
	if v := sysctlString(ctx, d.Exec, "machdep.cpu.family"); v != "" {
		c.Family = v
	}
	if v := sysctlString(ctx, d.Exec, "machdep.cpu.model"); v != "" {
		c.ModelID = v
	}
	if n, ok := sysctlInt(ctx, d.Exec, "machdep.cpu.stepping"); ok {
		c.Stepping = int32(n)
	}
	for _, key := range []string{"machdep.cpu.features", "machdep.cpu.leaf7_features"} {
		c.Flags = append(
			c.Flags,
			strings.Fields(strings.ToLower(sysctlString(ctx, d.Exec, key)))...)
	}
	c.Cores = int32(count)
	if n, ok := sysctlInt(ctx, d.Exec, "hw.physicalcpu"); ok {
		c.Cores = int32(n)
	}
	if n, ok := sysctlInt64(ctx, d.Exec, "hw.cpufrequency_max"); ok {
		c.Speed = float64(n) / 1_000_000
	} else if n, ok := sysctlInt64(ctx, d.Exec, "hw.cpufrequency"); ok {
		c.Speed = float64(n) / 1_000_000
	} else {
		c.Speed = maxPmgrFrequency(ctx, d.Exec)
	}
	info := summarize(count, []CPU{c})
	if n, ok := sysctlInt(ctx, d.Exec, "hw.packages"); ok {
		info.Sockets = n
	}

	return info, nil
}

// voltageStatesRE is the performance cores' frequency table of the pmgr
// device in ioreg: pairs of little-endian uint32, a frequency in Hz and a
// voltage.
var voltageStatesRE = regexp.MustCompile(`"voltage-states5-sram" = <([0-9a-f]+)>`)

// maxPmgrFrequency is the highest frequency of the table, in MHz, or 0
// when ioreg fails or has no table.
func maxPmgrFrequency(
	ctx context.Context,
	exec source.Runner,
) float64 {
	out, err := exec.Execute(ctx, "ioreg", "-l", "-w0", "-c", "AppleARMIODevice", "-n", "pmgr")
	if err != nil {
		return 0
	}
	m := voltageStatesRE.FindSubmatch(out)
	if m == nil {
		return 0
	}
	b, err := hex.DecodeString(string(m[1]))
	if err != nil {
		return 0
	}
	var top uint32
	for i := 0; i+8 <= len(b); i += 8 {
		top = max(top, binary.LittleEndian.Uint32(b[i:]))
	}

	return float64(top) / 1_000_000
}

// sysctlString runs `sysctl -n <key>`, and is its output, or "" when it
// fails.
func sysctlString(
	ctx context.Context,
	exec source.Runner,
	key string,
) string {
	out, err := exec.Execute(ctx, "sysctl", "-n", key)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

// sysctlInt runs `sysctl -n <key>` and parses the output as int.
// Returns (0, false) on any exec error or parse failure.
func sysctlInt(
	ctx context.Context,
	exec source.Runner,
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
	exec source.Runner,
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
