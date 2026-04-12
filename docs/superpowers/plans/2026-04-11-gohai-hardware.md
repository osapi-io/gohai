# gohai Hardware Section Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the 9 collectors in the 🖥️ Hardware section of the README, following the patterns established by the `platform` collector. Each collector wraps a well-maintained Go library (gopsutil or ghw) and reshapes output into a typed `Info` struct.

**Architecture:** Each collector lives in `pkg/gohai/collectors/<name>/`. Collector implements the `collector.Collector` interface. Linux-first, macOS best-effort. Registered in `pkg/gohai/gohai.go`'s `builtinCollectors()`. 100% test coverage on all non-`cmd/` packages.

**Reference implementation:** `pkg/gohai/collectors/platform/` — copy this pattern exactly for every new collector in this plan.

**Spec:** [docs/superpowers/specs/2026-04-11-gohai-design.md](../specs/2026-04-11-gohai-design.md)

**Methodology:** See [CLAUDE.md § Adding a New Collector](../../../CLAUDE.md#adding-a-new-collector) for the 9-step per-collector process.

---

## Scope

Collectors in this plan (in implementation order):

| Order | Collector    | Backing library                                            | Tier         |
| ----- | ------------ | ---------------------------------------------------------- | ------------ |
| 1     | `cpu`        | gopsutil `cpu`                                             | TierCore     |
| 2     | `memory`     | gopsutil `mem`                                             | TierCore     |
| 3     | `filesystem` | gopsutil `disk` (Partitions + Usage)                       | TierCore     |
| 4     | `disk`       | gopsutil `disk` (IOCounters) + ghw `block`                 | TierExtended |
| 5     | `dmi`        | ghw `baseboard` + `bios` + `chassis` + `product`           | TierExtended |
| 6     | `gpu`        | ghw `gpu`                                                  | TierExtended |
| 7     | `pci`        | ghw `pci`                                                  | TierExtended |
| 8     | `scsi`       | `/sys/class/scsi_host/` parsing (Linux); nil on darwin     | TierExtended |
| 9     | `hardware`   | gopsutil `host` extras + macOS `system_profiler` wrapper   | TierExtended |

The `hardware` collector is macOS-only extras (battery, storage, machine
model) — returns `nil` on Linux since `dmi`/`cpu`/`memory` already cover
Linux hardware.

---

## Common per-collector checklist

For **every** collector in this plan, complete each step below. The step
numbering matches [CLAUDE.md § Adding a New Collector](../../../CLAUDE.md#adding-a-new-collector).

- [ ] **Step 1: Create sub-package** at `pkg/gohai/collectors/<name>/`
- [ ] **Step 2: Write `<name>.go`** — MIT header, package doc, `Info` struct with JSON tags, `Collector` struct, `New()`, `Name()`, `Tier()`, `Dependencies()`, `Collect()` that calls `collect(ctx)`
- [ ] **Step 3: Write `linux.go`** (build-tagged) — testable `collect(ctx)` + `collectWithX(ctx, stubbableFn)` pattern
- [ ] **Step 4: Write `darwin.go`** (build-tagged) — same pattern; if identical to linux, factor into `unix.go` with `//go:build linux || darwin`
- [ ] **Step 5: Write `other.go`** (build-tagged `!linux && !darwin`) — returns `nil, nil`
- [ ] **Step 6: Write `export_linux_test.go` and `export_darwin_test.go`** — expose `Collect` and `CollectWithX` aliases
- [ ] **Step 7: Write `<name>_public_test.go`** — tests `New()` metadata + interface satisfaction + real `Collect()` smoke test
- [ ] **Step 8: Write `linux_public_test.go` and `darwin_public_test.go`** (build-tagged) — table-driven tests stubbing the library call, covering happy path and error propagation
- [ ] **Step 9: Register in `pkg/gohai/gohai.go`** — add `<name>.New()` to `builtinCollectors()` slice and import the sub-package
- [ ] **Step 10: Run tests and verify 100% coverage** — `go test ./... -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | grep -v '100.0%'` should return nothing
- [ ] **Step 11: Run lint** — `go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run` should return "0 issues"
- [ ] **Step 12: Update `docs/collectors/<name>.md`** — replace stub with: Status ✅, Description, Collected Fields table, Platform Support table, Example Output (JSON, both OSes), SDK Usage, Enable/Disable, Dependencies, Backing library (with license). Use `docs/collectors/platform.md` as template.
- [ ] **Step 13: Update `README.md`** — flip the collector's "Implemented" cell from 🚧 to ✅ (with library name, e.g., `✅ (gopsutil)`)
- [ ] **Step 14: Smoke test the CLI** — `go run . --collector.<name> --pretty` should output the collector's fact
- [ ] **Step 15: Commit** — `feat(<name>): add <name> collector wrapping <library>` with MIT footer

---

## Task 1: `cpu` collector

**Library:** [gopsutil/v4/cpu](https://pkg.go.dev/github.com/shirou/gopsutil/v4/cpu)

**Info struct:**

```go
type Info struct {
    Total     int      `json:"total"`      // logical CPUs
    Cores     int      `json:"cores"`      // physical cores
    Sockets   int      `json:"sockets,omitempty"`
    ModelName string   `json:"model_name,omitempty"`
    VendorID  string   `json:"vendor_id,omitempty"`
    Family    string   `json:"family,omitempty"`
    Model     string   `json:"model,omitempty"`
    Stepping  int32    `json:"stepping,omitempty"`
    Mhz       float64  `json:"mhz,omitempty"`
    CacheSize int32    `json:"cache_size,omitempty"`
    Flags     []string `json:"flags,omitempty"`
}
```

**Tier:** `TierCore`, **Dependencies:** none

**Backing calls:** `cpu.InfoWithContext(ctx)` returns `[]InfoStat`; `cpu.CountsWithContext(ctx, true)` returns logical count; `cpu.CountsWithContext(ctx, false)` returns physical count.

Follow the common per-collector checklist above. Commit when done.

---

## Task 2: `memory` collector

**Library:** [gopsutil/v4/mem](https://pkg.go.dev/github.com/shirou/gopsutil/v4/mem)

**Info struct:**

```go
type Info struct {
    Total       uint64  `json:"total"`         // bytes
    Available   uint64  `json:"available"`
    Used        uint64  `json:"used"`
    UsedPercent float64 `json:"used_percent"`
    Free        uint64  `json:"free"`
    Buffers     uint64  `json:"buffers,omitempty"`
    Cached      uint64  `json:"cached,omitempty"`
    Swap        *Swap   `json:"swap,omitempty"`
}

type Swap struct {
    Total       uint64  `json:"total"`
    Used        uint64  `json:"used"`
    Free        uint64  `json:"free"`
    UsedPercent float64 `json:"used_percent"`
}
```

**Tier:** `TierCore`, **Dependencies:** none

**Backing calls:** `mem.VirtualMemoryWithContext(ctx)`, `mem.SwapMemoryWithContext(ctx)`.

Follow the common per-collector checklist. Commit when done.

---

## Task 3: `filesystem` collector

**Library:** [gopsutil/v4/disk](https://pkg.go.dev/github.com/shirou/gopsutil/v4/disk)

**Info struct:**

```go
type Info struct {
    Mounts []Mount `json:"mounts"`
}

type Mount struct {
    Device     string  `json:"device"`
    Mountpoint string  `json:"mountpoint"`
    Fstype     string  `json:"fstype"`
    Opts       []string `json:"opts,omitempty"`
    Total      uint64  `json:"total,omitempty"`
    Used       uint64  `json:"used,omitempty"`
    Free       uint64  `json:"free,omitempty"`
    InodesTotal uint64 `json:"inodes_total,omitempty"`
    InodesUsed  uint64 `json:"inodes_used,omitempty"`
}
```

**Tier:** `TierCore`, **Dependencies:** none

**Backing calls:** `disk.PartitionsWithContext(ctx, false)` for physical partitions; per-partition `disk.UsageWithContext(ctx, mountpoint)` for size/inodes.

Follow the common per-collector checklist. Commit when done.

---

## Task 4: `disk` collector

**Libraries:** [gopsutil/v4/disk](https://pkg.go.dev/github.com/shirou/gopsutil/v4/disk) for I/O counters, [ghw/block](https://pkg.go.dev/github.com/jaypipes/ghw/pkg/block) for block device detail

**Info struct:**

```go
type Info struct {
    Devices []Device `json:"devices"`
}

type Device struct {
    Name        string `json:"name"`
    Model       string `json:"model,omitempty"`
    SizeBytes   uint64 `json:"size_bytes,omitempty"`
    Vendor      string `json:"vendor,omitempty"`
    SerialNumber string `json:"serial_number,omitempty"`
    BusPath     string `json:"bus_path,omitempty"`
    IOCounters  *IO    `json:"io_counters,omitempty"`
}

type IO struct {
    ReadCount  uint64 `json:"read_count"`
    WriteCount uint64 `json:"write_count"`
    ReadBytes  uint64 `json:"read_bytes"`
    WriteBytes uint64 `json:"write_bytes"`
}
```

**Tier:** `TierExtended`, **Dependencies:** none

**Backing calls:** `ghw.Block()` for device list; `disk.IOCountersWithContext(ctx)` for counters keyed by device name.

**macOS:** ghw's block support is Linux-only. On darwin, return devices with name/size from `diskutil list` output or fall back to gopsutil's `disk.Partitions()`.

Follow the common per-collector checklist. Commit when done.

---

## Task 5: `dmi` collector

**Library:** [ghw](https://pkg.go.dev/github.com/jaypipes/ghw) — `Baseboard()`, `BIOS()`, `Chassis()`, `Product()`

**Info struct:**

```go
type Info struct {
    BIOS      *BIOSInfo      `json:"bios,omitempty"`
    Baseboard *BaseboardInfo `json:"baseboard,omitempty"`
    Chassis   *ChassisInfo   `json:"chassis,omitempty"`
    Product   *ProductInfo   `json:"product,omitempty"`
}

// Each sub-struct mirrors ghw's corresponding fields (Vendor, Version,
// SerialNumber, etc.) with our JSON tags.
```

**Tier:** `TierExtended`, **Dependencies:** none

**Backing calls:** `ghw.Baseboard()`, `ghw.BIOS()`, `ghw.Chassis()`, `ghw.Product()`. Each may return a not-present error on systems without SMBIOS — handle gracefully (set the field to nil).

**macOS:** ghw DMI is Linux-only. On darwin, return `&Info{}` with all nil fields and a note in the doc.

Follow the common per-collector checklist. Commit when done.

---

## Task 6: `gpu` collector

**Library:** [ghw/gpu](https://pkg.go.dev/github.com/jaypipes/ghw/pkg/gpu)

**Info struct:**

```go
type Info struct {
    Cards []Card `json:"cards"`
}

type Card struct {
    Address string `json:"address"`          // PCI address
    Vendor  string `json:"vendor,omitempty"`
    Product string `json:"product,omitempty"`
    Driver  string `json:"driver,omitempty"`
}
```

**Tier:** `TierExtended`, **Dependencies:** none

**Backing calls:** `ghw.GPU()`.

**macOS:** ghw GPU is Linux-only. On darwin, return empty `Cards` list (or wrap `system_profiler SPDisplaysDataType` — deferred to a later plan).

Follow the common per-collector checklist. Commit when done.

---

## Task 7: `pci` collector

**Library:** [ghw/pci](https://pkg.go.dev/github.com/jaypipes/ghw/pkg/pci)

**Info struct:**

```go
type Info struct {
    Devices []Device `json:"devices"`
}

type Device struct {
    Address  string `json:"address"`
    Vendor   string `json:"vendor,omitempty"`
    Product  string `json:"product,omitempty"`
    Class    string `json:"class,omitempty"`
    Driver   string `json:"driver,omitempty"`
}
```

**Tier:** `TierExtended`, **Dependencies:** none

**Backing calls:** `ghw.PCI()`.

**macOS:** Linux-only. Return empty list.

Follow the common per-collector checklist. Commit when done.

---

## Task 8: `scsi` collector

**Library:** None — parse `/sys/class/scsi_host/` and `/sys/class/scsi_device/` on Linux.

**Info struct:**

```go
type Info struct {
    Hosts []Host `json:"hosts"`
}

type Host struct {
    Name        string `json:"name"`         // e.g. "host0"
    ProcName    string `json:"proc_name,omitempty"`
    State       string `json:"state,omitempty"`
    Devices     []Device `json:"devices,omitempty"`
}

type Device struct {
    ID     string `json:"id"`       // e.g. "0:0:0:0"
    Vendor string `json:"vendor,omitempty"`
    Model  string `json:"model,omitempty"`
    State  string `json:"state,omitempty"`
    Type   string `json:"type,omitempty"`
}
```

**Tier:** `TierExtended`, **Dependencies:** none

**Linux implementation:** Walk `/sys/class/scsi_host/host*`, read each file. Reference node_exporter's `scsi_tape.go` for parsing patterns. Make the sysfs root path a parameter for testability — tests write a fake sysfs tree to a temp dir and call the parser.

**macOS:** Return empty `Hosts` list.

Follow the common per-collector checklist. Commit when done.

---

## Task 9: `hardware` collector (macOS extras)

**Library:** `gopsutil/v4/host` on both OSes; `system_profiler SPHardwareDataType` on macOS for extras.

**Info struct:**

```go
type Info struct {
    Serial         string  `json:"serial,omitempty"`
    MachineModel   string  `json:"machine_model,omitempty"`
    ChipType       string  `json:"chip_type,omitempty"`      // macOS only
    NumberProcessors int   `json:"number_processors,omitempty"`
    PhysicalMemory uint64  `json:"physical_memory,omitempty"`
    Battery        *Battery `json:"battery,omitempty"`       // macOS only
}

type Battery struct {
    CurrentCapacity int    `json:"current_capacity"`
    MaxCapacity     int    `json:"max_capacity"`
    FullyCharged    bool   `json:"fully_charged"`
    IsCharging      bool   `json:"is_charging"`
    CycleCount      int    `json:"cycle_count,omitempty"`
    Health          string `json:"health,omitempty"`
}
```

**Tier:** `TierExtended`, **Dependencies:** none

**Linux:** return a minimal `Info` populated from `host.Info()` fields (serial, physical memory).

**macOS:** shell out to `system_profiler SPHardwareDataType -json` and parse, then `pmset -g batt` for battery. Parsing functions take the command output as a string so tests can exercise them without executing commands.

Follow the common per-collector checklist. Commit when done.

---

## Done — Hardware section complete

After all 9 tasks:

- Every hardware-section row in README.md shows ✅ in the Implemented column
- All hardware collector docs in `docs/collectors/*.md` show Status ✅
- `go test ./... -coverprofile=...` reports 100% for all non-`cmd/` packages
- Lint is clean (0 issues)
- `go run . --pretty` outputs JSON with `platform`, `cpu`, `memory`, `filesystem`, `disk`, `dmi`, `gpu`, `pci`, `scsi`, `hardware` (where applicable per OS)

Next plan: 🌐 Network section (1 collector: `network`) — then ☁️ Cloud (13), 🔮 Virtualization (4), and so on.
