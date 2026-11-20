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
	"fmt"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/osfs"
	"github.com/osapi-io/gohai/internal/collector"
)

const procMeminfoPath = "/proc/meminfo"

// Linux collects memory usage on Linux: every field of /proc/meminfo,
// read through the injected avfs.VFS, as Ohai reads it.
type Linux struct {
	base

	FS avfs.VFS
}

// NewLinux returns a Linux variant wired to the real OS filesystem.
func NewLinux() *Linux {
	return &Linux{FS: osfs.NewWithNoIdm()}
}

// Collect returns the memory Info, from /proc/meminfo.
func (l *Linux) Collect(
	_ context.Context,
	_ collector.PriorResults,
) (any, error) {
	b, err := l.FS.ReadFile(procMeminfoPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", procMeminfoPath, err)
	}

	return readMeminfo(b), nil
}

// readMeminfo is the Info of /proc/meminfo. The kernel reports kB, and
// Info holds bytes. Available, Used and Cached follow gopsutil: Cached
// includes SReclaimable, Available is Cached + Free on a kernel without
// MemAvailable, and Used is Total less Available.
func readMeminfo(
	raw []byte,
) *Info {
	kv := map[string]uint64{}
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		if key, val, ok := parseMeminfoLine(sc.Text()); ok {
			kv[key] = val
		}
	}
	info := &Info{
		Total:        kv["MemTotal"],
		Free:         kv["MemFree"],
		Available:    kv["MemAvailable"],
		Buffers:      kv["Buffers"],
		Cached:       kv["Cached"] + kv["SReclaimable"],
		Active:       kv["Active"],
		Inactive:     kv["Inactive"],
		ActiveAnon:   kv["Active(anon)"],
		InactiveAnon: kv["Inactive(anon)"],
		ActiveFile:   kv["Active(file)"],
		InactiveFile: kv["Inactive(file)"],
		Unevictable:  kv["Unevictable"],
		Dirty:        kv["Dirty"],
		WriteBack:    kv["Writeback"],
		WriteBackTmp: kv["WritebackTmp"],
		Shared:       kv["Shmem"],
		Shmem:        kv["Shmem"],
		Mapped:       kv["Mapped"],
		AnonPages:    kv["AnonPages"],
		Slab:         kv["Slab"],
		SReclaimable: kv["SReclaimable"],
		SUnreclaim:   kv["SUnreclaim"],
		KReclaimable: kv["KReclaimable"],
		PageTables:   kv["PageTables"],
		KernelStack:  kv["KernelStack"],
		PerCPU:       kv["Percpu"],
		HighTotal:    kv["HighTotal"],
		HighFree:     kv["HighFree"],
		LowTotal:     kv["LowTotal"],
		LowFree:      kv["LowFree"],
		NFSUnstable:  kv["NFS_Unstable"],
		Bounce:       kv["Bounce"],
		CommitLimit:  kv["CommitLimit"],
		CommittedAS:  kv["Committed_AS"],
		VmallocTotal: kv["VmallocTotal"],
		VmallocUsed:  kv["VmallocUsed"],
		VmallocChunk: kv["VmallocChunk"],
	}
	if _, ok := kv["MemAvailable"]; !ok {
		info.Available = info.Cached + info.Free
	}
	if info.Total > 0 && info.Available <= info.Total {
		info.Used = info.Total - info.Available
		info.UsedPercent = float64(info.Used) / float64(info.Total) * 100
	}
	_, hugetlb := kv["Hugetlb"]
	if kv["HugePages_Total"] > 0 || kv["Hugepagesize"] > 0 || kv["AnonHugePages"] > 0 || hugetlb {
		info.HugePages = &Hugepages{
			Total:         kv["HugePages_Total"],
			Free:          kv["HugePages_Free"],
			Reserved:      kv["HugePages_Rsvd"],
			Surplus:       kv["HugePages_Surp"],
			Size:          kv["Hugepagesize"],
			AnonHugePages: kv["AnonHugePages"],
			Hugetlb:       kv["Hugetlb"],
		}
	}
	for key, set := range map[string]func(*DirectMap, uint64){
		"DirectMap4k": func(d *DirectMap, v uint64) { d.Map4k = v },
		"DirectMap2M": func(d *DirectMap, v uint64) { d.Map2M = v },
		"DirectMap1G": func(d *DirectMap, v uint64) { d.Map1G = v },
	} {
		if v, ok := kv[key]; ok {
			set(ensureDirectMap(info), v)
		}
	}
	if total := kv["SwapTotal"]; total > 0 {
		free := kv["SwapFree"]
		info.Swap = &Swap{
			Total: total, Free: free, Used: total - free,
			UsedPercent: float64(total-free) / float64(total) * 100,
			Cached:      kv["SwapCached"],
		}
	}

	return info
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
