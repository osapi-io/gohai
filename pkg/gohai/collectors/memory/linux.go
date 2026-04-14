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
	"bufio"
	"context"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/osapi-io/gohai/internal/collector"
)

const procMeminfoPath = "/proc/meminfo"

// Linux collects memory usage on Linux. gopsutil's mem.VirtualMemory
// parses /proc/meminfo and exposes 27+ fields on its cross-platform
// struct; we forward every relevant field. Fields gopsutil hides
// behind its linux-only ExLinux type (Active(anon/file),
// Inactive(anon/file), Unevictable, KernelStack, Percpu) and fields
// it doesn't parse at all (Hugetlb, DirectMap4k/2M/1G, AnonPages,
// KReclaimable, Shmem) come from our own /proc/meminfo pass read
// through the injected avfs.VFS.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the memory Info. gopsutil provides the base; our
// /proc/meminfo extension fills the gaps.
func (l *Linux) Collect(
	ctx context.Context,
	_ collector.PriorResults,
) (any, error) {
	info, err := readMemory(ctx)
	if err != nil {
		return nil, err
	}
	if l.FS != nil {
		if b, err := l.FS.ReadFile(procMeminfoPath); err == nil {
			applyMeminfoExtension(info, b)
		}
	}
	return info, nil
}

// applyMeminfoExtension parses /proc/meminfo for the fields gopsutil
// doesn't expose on its cross-platform VirtualMemoryStat struct. Keys
// we already populate from gopsutil are ignored to avoid double-work.
// All values are reported in kB by the kernel; we multiply by 1024.
func applyMeminfoExtension(
	info *Info,
	raw []byte,
) {
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		key, val, ok := parseMeminfoLine(sc.Text())
		if !ok {
			continue
		}
		switch key {
		case "Active(anon)":
			info.ActiveAnon = val
		case "Inactive(anon)":
			info.InactiveAnon = val
		case "Active(file)":
			info.ActiveFile = val
		case "Inactive(file)":
			info.InactiveFile = val
		case "Unevictable":
			info.Unevictable = val
		case "KernelStack":
			info.KernelStack = val
		case "Percpu":
			info.PerCPU = val
		case "KReclaimable":
			info.KReclaimable = val
		case "AnonPages":
			info.AnonPages = val
		case "Shmem":
			info.Shmem = val
		case "Hugetlb":
			if info.HugePages == nil {
				info.HugePages = &Hugepages{}
			}
			info.HugePages.Hugetlb = val
		case "DirectMap4k":
			ensureDirectMap(info).Map4k = val
		case "DirectMap2M":
			ensureDirectMap(info).Map2M = val
		case "DirectMap1G":
			ensureDirectMap(info).Map1G = val
		}
	}
}

// ensureDirectMap lazy-allocates the DirectMap sub-struct.
func ensureDirectMap(
	info *Info,
) *DirectMap {
	if info.DirectMap == nil {
		info.DirectMap = &DirectMap{}
	}
	return info.DirectMap
}

// parseMeminfoLine turns a "Key: Value [unit]" line into (key, bytes).
// Kernel always reports kB; we multiply by 1024. Returns ok=false on
// malformed input.
func parseMeminfoLine(
	line string,
) (key string, bytes uint64, ok bool) {
	i := strings.Index(line, ":")
	if i < 0 {
		return "", 0, false
	}
	key = line[:i]
	rest := strings.TrimSpace(line[i+1:])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", 0, false
	}
	n, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return "", 0, false
	}
	if len(fields) >= 2 && strings.EqualFold(fields[1], "kB") {
		n *= 1024
	}
	return key, n, true
}
