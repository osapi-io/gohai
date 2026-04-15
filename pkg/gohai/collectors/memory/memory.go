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

// Package memory reports system memory totals, usage buckets, kernel
// allocations, page-cache state, and hugepages / DirectMap layout on
// Linux; on macOS it adds the wired / speculative / compressed buckets
// that the Darwin VM reports. Consumers use the full picture to size
// workloads, debug OOM/overcommit events, detect kernel leaks (Slab
// growth), and audit hugepages configuration for databases or DPDK.
package memory

import (
	"context"

	"github.com/shirou/gopsutil/v4/mem"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// Info holds memory usage data. Values are native bytes; Ohai emits
// kB-suffixed strings, we chose bytes for Go ergonomics. Linux-only
// fields stay zero on macOS and vice versa — the struct is the union.
type Info struct {
	// Core totals (both platforms).
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available,omitempty"`
	Used        uint64  `json:"used,omitempty"`
	UsedPercent float64 `json:"used_percent,omitempty"`
	Free        uint64  `json:"free,omitempty"`

	// LRU buckets. macOS populates Active / Inactive / Wired via the
	// mach syscall; Linux populates Active / Inactive from
	// /proc/meminfo via gopsutil, and the finer anon/file splits from
	// our own /proc/meminfo extension (gopsutil hides those behind a
	// linux-only ExLinux type we can't reference cross-platform).
	Active       uint64 `json:"active,omitempty"`
	Inactive     uint64 `json:"inactive,omitempty"`
	ActiveAnon   uint64 `json:"active_anon,omitempty"`
	InactiveAnon uint64 `json:"inactive_anon,omitempty"`
	ActiveFile   uint64 `json:"active_file,omitempty"`
	InactiveFile uint64 `json:"inactive_file,omitempty"`
	Unevictable  uint64 `json:"unevictable,omitempty"`

	// macOS-specific buckets (Speculative + Compressed parsed from
	// vm_stat; Wired comes from gopsutil via the mach syscall).
	Wired       uint64 `json:"wired,omitempty"`
	Speculative uint64 `json:"speculative,omitempty"`
	Compressed  uint64 `json:"compressed,omitempty"`

	// Page-cache / writeback state.
	Buffers      uint64 `json:"buffers,omitempty"`
	Cached       uint64 `json:"cached,omitempty"`
	Dirty        uint64 `json:"dirty,omitempty"`
	WriteBack    uint64 `json:"writeback,omitempty"`
	WriteBackTmp uint64 `json:"writeback_tmp,omitempty"`
	Shared       uint64 `json:"shared,omitempty"`
	Mapped       uint64 `json:"mapped,omitempty"`

	// Kernel allocations.
	Slab         uint64 `json:"slab,omitempty"`
	SReclaimable uint64 `json:"s_reclaimable,omitempty"`
	SUnreclaim   uint64 `json:"s_unreclaim,omitempty"`
	KReclaimable uint64 `json:"k_reclaimable,omitempty"`
	PageTables   uint64 `json:"page_tables,omitempty"`
	KernelStack  uint64 `json:"kernel_stack,omitempty"`
	PerCPU       uint64 `json:"percpu,omitempty"`

	// 32-bit legacy high/low memory split (HighTotal > 0 only on
	// 32-bit kernels with >4GB RAM — parsed for Ohai parity; on 64-bit
	// kernels these stay zero).
	HighTotal uint64 `json:"high_total,omitempty"`
	HighFree  uint64 `json:"high_free,omitempty"`
	LowTotal  uint64 `json:"low_total,omitempty"`
	LowFree   uint64 `json:"low_free,omitempty"`

	// NFS and legacy DMA bounce buffers.
	NFSUnstable uint64 `json:"nfs_unstable,omitempty"`
	Bounce      uint64 `json:"bounce,omitempty"`

	// Anonymous + shared (Linux-specific splits from /proc/meminfo
	// that gopsutil doesn't expose on its cross-platform struct).
	AnonPages uint64 `json:"anon_pages,omitempty"`
	Shmem     uint64 `json:"shmem,omitempty"`

	// DirectMap: size of physical memory covered by each page-table
	// granularity (4k / 2M / 1G). Populated from /proc/meminfo.
	DirectMap *DirectMap `json:"direct_map,omitempty"`

	// Commit accounting.
	CommitLimit uint64 `json:"commit_limit,omitempty"`
	CommittedAS uint64 `json:"committed_as,omitempty"`

	// Vmalloc arena.
	VmallocTotal uint64 `json:"vmalloc_total,omitempty"`
	VmallocUsed  uint64 `json:"vmalloc_used,omitempty"`
	VmallocChunk uint64 `json:"vmalloc_chunk,omitempty"`

	// Hugepages.
	HugePages *Hugepages `json:"hugepages,omitempty"`

	// Swap.
	Swap *Swap `json:"swap,omitempty"`
}

// Hugepages holds the hugepage-configuration picture as reported by
// `/proc/meminfo`. Populated only when any hugepage field is present
// (kernels without hugepages support skip this cleanly).
type Hugepages struct {
	Total         uint64 `json:"total"`                    // HugePages_Total
	Free          uint64 `json:"free,omitempty"`           // HugePages_Free
	Reserved      uint64 `json:"reserved,omitempty"`       // HugePages_Rsvd
	Surplus       uint64 `json:"surplus,omitempty"`        // HugePages_Surp
	Size          uint64 `json:"size,omitempty"`           // Hugepagesize
	AnonHugePages uint64 `json:"anon_hugepages,omitempty"` // AnonHugePages
	Hugetlb       uint64 `json:"hugetlb,omitempty"`        // Hugetlb (total hugepage memory)
}

// DirectMap reports the physical memory covered by each page-table
// granularity. Populated on Linux from /proc/meminfo.
type DirectMap struct {
	Map4k uint64 `json:"map_4k,omitempty"`
	Map2M uint64 `json:"map_2m,omitempty"`
	Map1G uint64 `json:"map_1g,omitempty"`
}

// Swap holds swap memory usage.
type Swap struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent,omitempty"`
	Cached      uint64  `json:"cached,omitempty"` // SwapCached (Linux)
}

// Collector is the public interface every memory variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "memory" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the memory variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// Package-level injection seams. Kept private so importers don't
// transitively need gopsutil; swapped via the SetXFn helpers in
// export_test.go.
//
// NOTE: gopsutil also ships an ExLinux struct exposing
// Active(anon/file), Inactive(anon/file), Unevictable, Percpu,
// KernelStack — but it lives behind a `//go:build linux` tag, so
// referencing its type from our no-build-tags package breaks the
// darwin build. We surface the 27 fields that live on the
// cross-platform VirtualMemoryStat and leave the ex-only fields as a
// future follow-up (requires either build-tagged shim files or a
// procfs migration).
var (
	virtualMemoryFn = mem.VirtualMemoryWithContext
	swapMemoryFn    = mem.SwapMemoryWithContext
)

// readMemory is the production bridge to gopsutil. Combines
// VirtualMemory + SwapMemory + (Linux only) ExLinux.VirtualMemory into
// our Info. gopsutil parses /proc/meminfo once — we forward every
// field it exposes rather than reparse (library-first principle).
func readMemory(
	ctx context.Context,
) (*Info, error) {
	vm, err := virtualMemoryFn(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{
		Total:        vm.Total,
		Available:    vm.Available,
		Used:         vm.Used,
		UsedPercent:  vm.UsedPercent,
		Free:         vm.Free,
		Active:       vm.Active,
		Inactive:     vm.Inactive,
		Wired:        vm.Wired,
		Buffers:      vm.Buffers,
		Cached:       vm.Cached,
		Dirty:        vm.Dirty,
		WriteBack:    vm.WriteBack,
		WriteBackTmp: vm.WriteBackTmp,
		Shared:       vm.Shared,
		Mapped:       vm.Mapped,
		Slab:         vm.Slab,
		SReclaimable: vm.Sreclaimable,
		SUnreclaim:   vm.Sunreclaim,
		PageTables:   vm.PageTables,
		CommitLimit:  vm.CommitLimit,
		CommittedAS:  vm.CommittedAS,
		VmallocTotal: vm.VmallocTotal,
		VmallocUsed:  vm.VmallocUsed,
		VmallocChunk: vm.VmallocChunk,
	}
	if vm.HugePagesTotal > 0 || vm.HugePageSize > 0 || vm.AnonHugePages > 0 {
		info.HugePages = &Hugepages{
			Total:         vm.HugePagesTotal,
			Free:          vm.HugePagesFree,
			Reserved:      vm.HugePagesRsvd,
			Surplus:       vm.HugePagesSurp,
			Size:          vm.HugePageSize,
			AnonHugePages: vm.AnonHugePages,
		}
	}
	if sm, err := swapMemoryFn(ctx); err == nil && sm.Total > 0 {
		info.Swap = &Swap{
			Total:       sm.Total,
			Used:        sm.Used,
			Free:        sm.Free,
			UsedPercent: sm.UsedPercent,
			Cached:      vm.SwapCached,
		}
	}
	return info, nil
}
