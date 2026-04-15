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

// Package cpu collects CPU topology, model, feature flags, cache layout,
// NUMA layout, and hardware vulnerability mitigation status.
package cpu

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/avfs/avfs"
	"github.com/shirou/gopsutil/v4/cpu"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/platform"
)

// vulnerabilitiesDir is the sysfs directory whose files enumerate
// per-mitigation status (meltdown, spectre_v1, spectre_v2, mds, l1tf,
// srbds, retbleed, tsx_async_abort, itlb_multihit, ...). Each file's
// basename is the mitigation name and its contents are one line of
// status text.
const vulnerabilitiesDir = "/sys/devices/system/cpu/vulnerabilities"

// Info holds CPU information. Includes aggregate facts (logical count,
// physical cores, socket count), model/feature data from the first CPU
// (assumed homogeneous), per-level cache sizes, NUMA topology, and the
// vulnerability mitigation status map on Linux. Extended fields
// (BIOS identity, frequency range, virtualization, execution modes,
// additional cache levels) come from `lscpu` on Linux to match Ohai's
// full cpu plugin coverage.
type Info struct {
	Count           int               `json:"count"`                // logical CPU count (OCSF: device.cpu_count)
	Sockets         int               `json:"sockets"`              // physical packages
	Cores           int               `json:"cores"`                // physical core count
	ModelName       string            `json:"model_name,omitempty"` // human-readable CPU name
	VendorID        string            `json:"vendor_id,omitempty"`
	Family          string            `json:"family,omitempty"`
	Model           string            `json:"model,omitempty"`
	Stepping        int32             `json:"stepping,omitempty"`
	Mhz             float64           `json:"mhz,omitempty"`
	CacheSize       int32             `json:"cache_size,omitempty"` // KB — aggregate from /proc/cpuinfo
	Flags           []string          `json:"flags,omitempty"`
	Caches          *Caches           `json:"caches,omitempty"`           // per-level sizes from lscpu (Linux)
	NumaNodes       map[int][]int     `json:"numa_nodes,omitempty"`       // node id → CPU indices (Linux)
	NumaNodesCount  int               `json:"numa_nodes_count,omitempty"` // NUMA node count from "NUMA node(s):" line
	Vulnerabilities map[string]string `json:"vulnerabilities,omitempty"`  // mitigation → status (Linux)

	// CPU availability (Linux, from lscpu).
	CPUsOnline  int `json:"cpus_online,omitempty"`
	CPUsOffline int `json:"cpus_offline,omitempty"`

	// BIOS / machine identity (Linux, from lscpu).
	BIOSVendorID  string `json:"bios_vendor_id,omitempty"`
	BIOSModelName string `json:"bios_model_name,omitempty"`
	MachineType   string `json:"machine_type,omitempty"` // s390x mainframes

	// Frequency range (Linux, from lscpu). Strings mirror lscpu output
	// (e.g. `3200.0000`); same convention Ohai follows.
	MhzMax     string `json:"mhz_max,omitempty"`
	MhzMin     string `json:"mhz_min,omitempty"`
	MhzDynamic string `json:"mhz_dynamic,omitempty"`
	Bogomips   string `json:"bogomips,omitempty"`

	// Execution mode metadata (Linux, from lscpu).
	CPUOpmodes   []string `json:"cpu_opmodes,omitempty"`
	ByteOrder    string   `json:"byte_order,omitempty"`
	AddressSizes []string `json:"address_sizes,omitempty"`

	// Virtualization capabilities (Linux, from lscpu). `HypervisorVendor`
	// is the key signal our virtualization collector would consume when
	// /sys/devices/virtual/misc/kvm is absent (Ohai does the same).
	Virtualization     string `json:"virtualization,omitempty"`
	VirtualizationType string `json:"virtualization_type,omitempty"`
	HypervisorVendor   string `json:"hypervisor_vendor,omitempty"`
	DispatchingMode    string `json:"dispatching_mode,omitempty"` // s390x
}

// Caches carries the per-level cache sizes reported by `lscpu`. Strings
// mirror lscpu's output verbatim (e.g. `32 KiB`, `1 MiB`) — Ohai does
// the same; keeps units visible without a unit-conversion policy.
type Caches struct {
	L1d string `json:"l1d,omitempty"`
	L1i string `json:"l1i,omitempty"`
	L2  string `json:"l2,omitempty"`
	L2d string `json:"l2d,omitempty"`
	L2i string `json:"l2i,omitempty"`
	L3  string `json:"l3,omitempty"`
	L4  string `json:"l4,omitempty"`
}

// Collector is the public interface every cpu variant satisfies.
type Collector interface {
	collector.Collector
}

type base struct{}

func (base) Name() string           { return "cpu" }
func (base) Category() string       { return collector.CategoryHardware }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the cpu variant for the host OS.
func New() Collector {
	if platform.Detect() == "darwin" {
		return NewDarwin()
	}
	return NewLinux()
}

// infoFn is the injection seam for gopsutil's cpu.InfoWithContext.
// Kept private so importers don't transitively need gopsutil. Swapped
// in tests via SetInfoFn (export_test.go).
var infoFn = cpu.InfoWithContext

// countsFn is the injection seam for gopsutil's cpu.CountsWithContext.
// Kept private alongside infoFn. Swapped via SetCountsFn.
var countsFn = cpu.CountsWithContext

// readCPU is the production bridge to gopsutil. Returns the base Info
// populated from `/proc/cpuinfo` (Linux) or sysctl (Darwin). Per-OS
// Collect wrappers layer extensions on top.
func readCPU(
	ctx context.Context,
) (*Info, error) {
	stats, err := infoFn(ctx)
	if err != nil {
		return nil, err
	}
	info := &Info{}
	if logical, err := countsFn(ctx, true); err == nil {
		info.Count = logical
	}
	sockets := map[string]struct{}{}
	var totalCores int32
	for _, s := range stats {
		sockets[s.PhysicalID] = struct{}{}
		totalCores += s.Cores
	}
	info.Sockets = len(sockets)
	info.Cores = int(totalCores)
	if len(stats) > 0 {
		s := stats[0]
		info.ModelName = s.ModelName
		info.VendorID = s.VendorID
		info.Family = s.Family
		info.Model = s.Model
		info.Stepping = s.Stepping
		info.Mhz = s.Mhz
		info.CacheSize = s.CacheSize
		info.Flags = s.Flags
	}
	return info, nil
}

// readVulnerabilities walks /sys/devices/system/cpu/vulnerabilities/
// via the injected avfs and returns a mitigation-name → status map.
// Missing directory yields nil — most modern Linux kernels have it,
// but containers / stripped /sys do not, and an absent directory is
// not an error.
func readVulnerabilities(
	fs avfs.VFS,
) map[string]string {
	entries, err := fs.ReadDir(vulnerabilitiesDir)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := fs.ReadFile(vulnerabilitiesDir + "/" + e.Name())
		if err != nil {
			continue
		}
		out[e.Name()] = strings.TrimSpace(string(b))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// lscpuSummary holds the fields we care about from `lscpu` output.
// Populated by parseLscpu; consumed by the per-OS Collect layering.
type lscpuSummary struct {
	architecture    string
	sockets         int
	coresPerSocket  int
	threadsPerCore  int
	socketsPerBook  int
	booksPerDrawer  int
	drawers         int
	caches          Caches
	numaNodes       map[int][]int
	numaNodesCount  int
	cpusOnline      int
	cpusOffline     int
	haveLscpuLayout bool // true once any of the topology fields was parsed

	// Additional flat fields — merged onto Info verbatim.
	biosVendorID       string
	biosModelName      string
	machineType        string
	mhzMax             string
	mhzMin             string
	mhzDynamic         string
	bogomips           string
	cpuOpmodes         []string
	byteOrder          string
	addressSizes       []string
	virtualization     string
	virtualizationType string
	hypervisorVendor   string
	dispatchingMode    string
}

// parseLscpu parses `lscpu` line-oriented output. Mirrors the key set
// Ohai's parse_lscpu uses, trimmed to what our Info surfaces:
//
//   - Architecture (triggers s390x / ppc64le count-override path)
//   - Socket(s), Core(s) per socket, Thread(s) per core
//   - Socket(s) per book, Book(s) per drawer, Drawer(s) (s390x)
//   - L1d / L1i / L2 / L3 cache
//   - NUMA node<N> CPU(s) — per node CPU index list
//
// Line format is `Field:\s+Value`. Unrecognized fields are ignored.
func parseLscpu(
	out []byte,
) lscpuSummary {
	s := lscpuSummary{numaNodes: map[int][]int{}}
	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// key: value (value starts at first non-space after the colon)
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if val == "" {
			continue
		}
		switch {
		case key == "Architecture":
			s.architecture = val
			s.haveLscpuLayout = true
		case key == "Socket(s)":
			s.sockets = atoi(val)
			s.haveLscpuLayout = true
		case key == "Core(s) per socket", key == "Core(s) per cluster":
			s.coresPerSocket = atoi(val)
			s.haveLscpuLayout = true
		case key == "Thread(s) per core":
			s.threadsPerCore = atoi(val)
			s.haveLscpuLayout = true
		case key == "Socket(s) per book":
			s.socketsPerBook = atoi(val)
			s.haveLscpuLayout = true
		case key == "Book(s) per drawer":
			s.booksPerDrawer = atoi(val)
			s.haveLscpuLayout = true
		case key == "Drawer(s)":
			s.drawers = atoi(val)
			s.haveLscpuLayout = true
		case key == "L1d cache":
			s.caches.L1d = val
		case key == "L1i cache":
			s.caches.L1i = val
		case key == "L2 cache":
			s.caches.L2 = val
		case key == "L2d cache":
			s.caches.L2d = val
		case key == "L2i cache":
			s.caches.L2i = val
		case key == "L3 cache":
			s.caches.L3 = val
		case key == "L4 cache":
			s.caches.L4 = val
		case key == "NUMA node(s)":
			s.numaNodesCount = atoi(val)
		case key == "On-line CPU(s) list":
			s.cpusOnline = len(parseCPURange(val))
		case key == "Off-line CPU(s) list":
			s.cpusOffline = len(parseCPURange(val))
		case key == "BIOS Vendor ID":
			s.biosVendorID = val
		case key == "BIOS Model name":
			s.biosModelName = val
		case key == "Machine type":
			s.machineType = val
		case key == "CPU max MHz":
			s.mhzMax = val
		case key == "CPU min MHz":
			s.mhzMin = val
		case key == "CPU dynamic MHz":
			s.mhzDynamic = val
		case key == "BogoMIPS":
			s.bogomips = val
		case key == "CPU op-mode(s)":
			s.cpuOpmodes = splitCSV(val)
		case key == "Byte Order":
			s.byteOrder = strings.ToLower(val)
		case key == "Address sizes":
			s.addressSizes = splitCSV(val)
		case key == "Virtualization":
			s.virtualization = val
		case key == "Virtualization type":
			s.virtualizationType = val
		case key == "Hypervisor vendor":
			s.hypervisorVendor = val
		case key == "Dispatching mode":
			s.dispatchingMode = val
		case strings.HasPrefix(key, "NUMA node") && strings.HasSuffix(key, "CPU(s)"):
			// "NUMA node0 CPU(s)" → node 0
			mid := strings.TrimPrefix(key, "NUMA node")
			mid = strings.TrimSuffix(mid, " CPU(s)")
			mid = strings.TrimSpace(mid)
			if node, err := strconv.Atoi(mid); err == nil {
				s.numaNodes[node] = parseCPURange(val)
			}
		}
	}
	if len(s.numaNodes) == 0 {
		s.numaNodes = nil
	}
	return s
}

// atoi is an error-suppressing Atoi for lscpu values that are
// structurally guaranteed to be integers. Returns 0 on parse failure —
// which preserves the "unset" invariant callers rely on.
func atoi(
	v string,
) int {
	n, _ := strconv.Atoi(v)
	return n
}

// splitCSV splits a comma-separated lscpu value ("32-bit, 64-bit",
// "48 bits physical, 48 bits virtual") into trimmed tokens. Empty
// entries are dropped; fully-empty input returns nil.
func splitCSV(
	v string,
) []string {
	out := []string{}
	for _, part := range strings.Split(v, ",") {
		t := strings.TrimSpace(part)
		if t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseCPURange converts an lscpu-style CPU list like `0-3,8,10-11`
// into a sorted []int of CPU indices. Returns nil on malformed input
// rather than partial results — a NUMA node with unparseable CPU list
// is reported as absent rather than half-populated.
func parseCPURange(
	v string,
) []int {
	out := []int{}
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil || lo > hi {
				return nil
			}
			for i := lo; i <= hi; i++ {
				out = append(out, i)
			}
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Ints(out)
	return out
}

// applyLscpuToInfo merges a parsed lscpu summary into the Info built
// from gopsutil. Caches and NumaNodes are always surfaced when
// lscpu provides them. Socket / core / thread counts are only
// overridden on architectures where /proc/cpuinfo (and therefore
// gopsutil) is known to mis-count — s390x and ppc64le.
func applyLscpuToInfo(
	info *Info,
	s lscpuSummary,
) {
	if s.caches != (Caches{}) {
		c := s.caches
		info.Caches = &c
	}
	if len(s.numaNodes) > 0 {
		info.NumaNodes = s.numaNodes
	}
	if s.numaNodesCount > 0 {
		info.NumaNodesCount = s.numaNodesCount
	}
	if s.cpusOnline > 0 {
		info.CPUsOnline = s.cpusOnline
	}
	if s.cpusOffline > 0 {
		info.CPUsOffline = s.cpusOffline
	}
	info.BIOSVendorID = s.biosVendorID
	info.BIOSModelName = s.biosModelName
	info.MachineType = s.machineType
	info.MhzMax = s.mhzMax
	info.MhzMin = s.mhzMin
	info.MhzDynamic = s.mhzDynamic
	info.Bogomips = s.bogomips
	info.CPUOpmodes = s.cpuOpmodes
	info.ByteOrder = s.byteOrder
	info.AddressSizes = s.addressSizes
	info.Virtualization = s.virtualization
	info.VirtualizationType = s.virtualizationType
	info.HypervisorVendor = s.hypervisorVendor
	info.DispatchingMode = s.dispatchingMode
	if !s.haveLscpuLayout {
		return
	}
	switch s.architecture {
	case "s390x":
		// total = sockets_per_book * cores_per_socket * threads_per_core * books_per_drawer * drawers
		total := nonZero(s.socketsPerBook) *
			nonZero(s.coresPerSocket) *
			nonZero(s.threadsPerCore) *
			nonZero(s.booksPerDrawer) *
			nonZero(s.drawers)
		cores := nonZero(s.socketsPerBook) *
			nonZero(s.coresPerSocket) *
			nonZero(s.booksPerDrawer) *
			nonZero(s.drawers)
		if total > 0 {
			info.Count = total
		}
		if cores > 0 {
			info.Cores = cores
		}
		if s.socketsPerBook > 0 {
			info.Sockets = s.socketsPerBook
		}
	case "ppc64le":
		// total = sockets * cores_per_socket * threads_per_core
		total := nonZero(s.sockets) *
			nonZero(s.coresPerSocket) *
			nonZero(s.threadsPerCore)
		cores := nonZero(s.sockets) * nonZero(s.coresPerSocket)
		if total > 0 {
			info.Count = total
		}
		if cores > 0 {
			info.Cores = cores
		}
		if s.sockets > 0 {
			info.Sockets = s.sockets
		}
	}
}

// nonZero returns n if n > 0, else 1. Used for the arch-override
// multiplications where a missing factor should not zero out the
// whole result — Ohai uses the same 1-identity trick.
func nonZero(
	n int,
) int {
	if n > 0 {
		return n
	}
	return 1
}
