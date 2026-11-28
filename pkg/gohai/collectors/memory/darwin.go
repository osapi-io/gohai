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

package memory

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/pkg/gohai/source"
)

// Darwin collects memory usage on macOS: the total from sysctl
// hw.memsize, the page counts from vm_stat, and swap from sysctl
// vm.swapusage. Available and Used follow gopsutil: Available is the
// free and inactive pages, and Used is Total less Available.
type Darwin struct {
	base

	Exec source.Runner
}

// NewDarwin returns a Darwin variant that runs the host's sysctl and
// vm_stat.
func NewDarwin() *Darwin {
	return &Darwin{Exec: source.Commands()}
}

// Collect returns the memory Info. With no Exec it gives no fact, as
// every value comes from sysctl and vm_stat.
func (d *Darwin) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	if d.Exec == nil {
		return nil, nil
	}
	out, err := d.Exec.Execute(ctx, "sysctl", "-n", "hw.memsize")
	if err != nil {
		return nil, fmt.Errorf("sysctl hw.memsize: %w", err)
	}
	total, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("sysctl hw.memsize: %w", err)
	}
	info := &Info{Total: total}
	// Without vm_stat, Available and Used would be zero, which reads as
	// a host with no free memory. Without vm.swapusage, Swap would be nil,
	// which reads as a host with no swap. Both are errors instead.
	out, err = d.Exec.Execute(ctx, "vm_stat")
	if err != nil {
		return nil, fmt.Errorf("vm_stat: %w", err)
	}
	applyVMStat(info, out)
	out, err = d.Exec.Execute(ctx, "sysctl", "-n", "vm.swapusage")
	if err != nil {
		return nil, fmt.Errorf("sysctl vm.swapusage: %w", err)
	}
	info.Swap = parseSwapUsage(string(out))

	return info, nil
}

// swapUsageRE is one figure of sysctl vm.swapusage:
// "total = 3072.00M  used = 1818.81M  free = 1253.19M  (encrypted)".
var swapUsageRE = regexp.MustCompile(`(total|used|free) = ([\d.]+)([KMG])`)

// parseSwapUsage is the swap of vm.swapusage, or nil when it has no
// total.
func parseSwapUsage(
	out string,
) *Swap {
	unit := map[string]float64{"K": 1 << 10, "M": 1 << 20, "G": 1 << 30}
	v := map[string]uint64{}
	for _, m := range swapUsageRE.FindAllStringSubmatch(out, -1) {
		n, _ := strconv.ParseFloat(m[2], 64)
		v[m[1]] = uint64(n * unit[m[3]])
	}
	if v["total"] == 0 {
		return nil
	}

	return &Swap{
		Total: v["total"], Used: v["used"], Free: v["free"],
		UsedPercent: float64(v["total"]-v["free"]) / float64(v["total"]) * 100,
	}
}

// vmStatPageSize matches the first line's page-size declaration.
// `Mach Virtual Memory Statistics: (page size of 16384 bytes)` on
// Apple Silicon; 4096 on Intel.
var vmStatPageSize = regexp.MustCompile(`page size of (\d+) bytes`)

// applyVMStat parses `vm_stat` output into info's page buckets, and
// derives Available, Used and UsedPercent from them and info.Total.
func applyVMStat(
	info *Info,
	out []byte,
) {
	pageSize := uint64(4096)
	for rawLine := range strings.Lines(string(out)) {
		line := strings.TrimRight(rawLine, "\r\n")
		if m := vmStatPageSize.FindStringSubmatch(line); m != nil {
			if ps, err := strconv.ParseUint(m[1], 10, 64); err == nil && ps > 0 {
				pageSize = ps
			}
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line[idx+1:]), "."))
		pages, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			continue
		}
		bytes := pages * pageSize
		switch key {
		case "Pages free":
			info.Free = bytes
		case "Pages active":
			info.Active = bytes
		case "Pages inactive":
			info.Inactive = bytes
		case "Pages wired down":
			info.Wired = bytes
		case "Pages speculative":
			info.Speculative = bytes
		case "Pages stored in compressor":
			info.Compressed = bytes
		}
	}
	info.Available = info.Free + info.Inactive
	if info.Total > 0 && info.Available <= info.Total {
		info.Used = info.Total - info.Available
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}
}
