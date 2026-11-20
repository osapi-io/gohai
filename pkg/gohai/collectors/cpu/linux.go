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
	"fmt"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/executor"
)

// Linux collects CPU facts on Linux. /proc/cpuinfo, read through FS, is
// the primary source, with the count from /proc/stat when it has no
// processor blocks. Two extensions go on top:
//
//   - vulnerability mitigation status via /sys/devices/system/cpu/vulnerabilities/*
//     (read through FS — an avfs.VFS, so tests can inject memfs content).
//   - NUMA topology, per-level cache sizes, and — on s390x / ppc64le —
//     authoritative core/socket/thread counts via `lscpu` (run through
//     Exec — an executor.Executor, so tests can mock the command).
type Linux struct {
	base

	FS   avfs.VFS
	Exec executor.Executor
}

// NewLinux returns a Linux variant wired to production dependencies:
// the real OS filesystem for sysfs, and a real `os/exec` wrapper for
// `lscpu`.
func NewLinux() *Linux {
	return &Linux{
		FS:   osfs.NewWithNoIdm(),
		Exec: executor.New(),
	}
}

// Collect returns the CPU Info with Linux-specific extensions merged
// on top of /proc/cpuinfo.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	b, err := l.FS.ReadFile(procCPUInfo)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procCPUInfo, err)
	}
	cpus := parseCPUInfo(l.FS, b)
	count := len(cpus)
	if count == 0 {
		count = countStatCPUs(l.FS)
	}
	info := summarize(count, cpus)
	if v := readVulnerabilities(l.FS); v != nil {
		info.Vulnerabilities = v
	}
	if l.Exec == nil {
		return info, nil
	}
	if out, err := l.Exec.Execute(ctx, "lscpu"); err == nil {
		applyLscpuToInfo(info, parseLscpu(out))
	}
	return info, nil
}

// procCPUInfo has a block of facts for each logical CPU.
const procCPUInfo = "/proc/cpuinfo"

// implementers is the vendor of an ARM "CPU implementer" code.
var implementers = map[uint64]string{
	0x41: "ARM", 0x42: "Broadcom", 0x43: "Cavium", 0x44: "DEC", 0x46: "Fujitsu",
	0x48: "HiSilicon", 0x49: "Infineon", 0x4d: "Motorola/Freescale", 0x4e: "NVIDIA",
	0x50: "APM", 0x51: "Qualcomm", 0x56: "Marvell", 0x61: "Apple", 0x69: "Intel",
	0xc0: "Ampere",
}

// parseCPUInfo is an entry for each logical CPU of /proc/cpuinfo, read as
// gopsutil read it: a block starts at a numbered "processor" line, each
// entry counts as one core, and the speed is the CPU's maximum from
// cpufreq when the kernel has it.
func parseCPUInfo(
	fs avfs.VFS,
	raw []byte,
) []CPU {
	cpus := []CPU{}
	var cur *CPU
	var index []int
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key == "processor" || key == "cpu number" {
			n, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			cpus = append(cpus, CPU{Cores: 1})
			index = append(index, n)
			cur = &cpus[len(cpus)-1]

			continue
		}
		if cur == nil {
			continue
		}
		switch key {
		case "vendor_id", "vendorId":
			cur.VendorID = value
		case "CPU implementer":
			if v, err := strconv.ParseUint(value, 0, 16); err == nil {
				cur.VendorID = implementers[v]
			}
		case "cpu family":
			cur.Family = value
		case "model", "CPU part":
			cur.ModelID = value
		case "model name", "Model Name", "cpu":
			cur.ModelName = value
		case "stepping", "CPU revision":
			if n, err := strconv.ParseInt(value, 10, 32); err == nil {
				cur.Stepping = int32(n)
			}
		case "cpu MHz":
			if f, err := strconv.ParseFloat(value, 64); err == nil {
				cur.Speed = f
			}
		case "cache size":
			if n, err := strconv.ParseInt(strings.TrimSuffix(value, " KB"), 10, 32); err == nil {
				cur.CacheSize = int32(n)
			}
		case "physical id":
			cur.PhysicalID = value
		case "core id":
			cur.CoreID = value
		case "flags", "Features":
			cur.Flags = strings.FieldsFunc(value, func(r rune) bool { return r == ' ' || r == ',' })
		}
	}
	for i := range cpus {
		finishCPU(fs, index[i], &cpus[i])
	}

	return cpus
}

// finishCPU fills c from sysfs, for CPU n: its core id when cpuinfo has
// none, and its maximum speed from cpufreq, in kHz, or in Hz on a kernel
// that gives a figure over 9,999 MHz.
func finishCPU(
	fs avfs.VFS,
	n int,
	c *CPU,
) {
	dir := "/sys/devices/system/cpu/cpu" + strconv.Itoa(n)
	if c.CoreID == "" {
		if b, err := fs.ReadFile(dir + "/topology/core_id"); err == nil {
			c.CoreID = strings.TrimSpace(string(b))
		}
	}
	if b, err := fs.ReadFile(dir + "/cpufreq/cpuinfo_max_freq"); err == nil {
		if f, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); err == nil {
			c.Speed = f / 1000
			if c.Speed > 9999 {
				c.Speed /= 1000
			}
		}
	}
}

// countStatCPUs is the number of cpuN lines of /proc/stat, the logical
// count when cpuinfo has no numbered processor lines.
func countStatCPUs(
	fs avfs.VFS,
) int {
	b, err := fs.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(string(b), "\n") {
		if len(line) > 3 && strings.HasPrefix(line, "cpu") && line[3] >= '0' && line[3] <= '9' {
			n++
		}
	}

	return n
}
