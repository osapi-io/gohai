# CPU

> **Status:** Implemented ✅

## Description

Reports CPU topology, model, feature flags, cache layout, NUMA layout, and
hardware vulnerability mitigation status. The vulnerabilities map is the
canonical per-host signal for Meltdown / Spectre / MDS / Retbleed / SRBDS
audits. Non-x86 topology (s390x, ppc64le) is sourced from `lscpu` where
`/proc/cpuinfo` is insufficient. macOS core and frequency facts come from
`sysctl` rather than gopsutil, which reports logical cores as physical and zero
frequency on Apple Silicon.

Consumers use this to:

- Pick thread-pool / worker concurrency (`GOMAXPROCS`, task-queue size).
- Drive licensing / compliance checks that count physical sockets.
- Gate features on CPU capabilities (`avx2`, `aes`, `sse4_2`) without running
  probes.
- Audit Meltdown/Spectre mitigation posture across a fleet.
- Identify fleet heterogeneity (different model names / core counts within the
  same tier).

Assumes homogeneous CPUs across sockets — `model_name`, `vendor_id`, `family`,
`model`, `stepping`, `cache_size`, and `flags` come from the first logical CPU
gopsutil returns. True on ~99% of hosts; asymmetric-core SKUs (Apple Silicon's
P+E cores, Intel Alder Lake's hybrid) report the first core's data.

## Collected Fields

| Field                 | Type              | Description                                                                                                                                                                                                                                                                                         | Schema mapping                                                                                     |
| --------------------- | ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `count`               | int               | Logical CPU count (threads visible to the scheduler).                                                                                                                                                                                                                                               | OCSF `device.cpu_count` (leaf stripped: `cpu_` duplicates collector name).                         |
| `sockets`             | int               | Physical packages. macOS: `hw.packages`. Linux: gopsutil (x86) or `lscpu` (non-x86).                                                                                                                                                                                                                | No direct schema mapping.                                                                          |
| `cores`               | int               | Physical cores across all sockets. macOS: `hw.physicalcpu`. Linux: gopsutil or `lscpu`.                                                                                                                                                                                                             | OCSF `device.cpu_cores` (leaf matches our JSON key).                                               |
| `model_name`          | string            | Human-readable CPU name from `/proc/cpuinfo` or sysctl.                                                                                                                                                                                                                                             | OTel `host.cpu.model.name` (OCSF `device.hw_info.cpu_type` means architecture, not product model). |
| `vendor_id`           | string            | CPU vendor (`"GenuineIntel"`, `"AuthenticAMD"`).                                                                                                                                                                                                                                                    | OTel `host.cpu.vendor.id`.                                                                         |
| `family`              | string            | CPU family number.                                                                                                                                                                                                                                                                                  | OTel `host.cpu.family`.                                                                            |
| `model`               | string            | Model number within family.                                                                                                                                                                                                                                                                         | No direct schema mapping.                                                                          |
| `stepping`            | int32             | Revision of the model.                                                                                                                                                                                                                                                                              | OTel `host.cpu.stepping`.                                                                          |
| `mhz`                 | float64           | Current clock in MHz. macOS: `hw.cpufrequency_max` → `hw.cpufrequency`; absent on Apple Silicon.                                                                                                                                                                                                    | No direct schema mapping.                                                                          |
| `cache_size`          | int32             | Aggregate cache size in KB from `/proc/cpuinfo` (a single value — Linux only).                                                                                                                                                                                                                      | No direct schema mapping.                                                                          |
| `flags`               | []string          | CPU feature flags.                                                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                          |
| `caches.l1d`          | string            | L1 data cache size from `lscpu` (Linux).                                                                                                                                                                                                                                                            | No direct schema mapping.                                                                          |
| `caches.l1i`          | string            | L1 instruction cache size from `lscpu` (Linux).                                                                                                                                                                                                                                                     | No direct schema mapping.                                                                          |
| `caches.l2`           | string            | L2 cache size from `lscpu` (Linux).                                                                                                                                                                                                                                                                 | No direct schema mapping.                                                                          |
| `caches.l2d`          | string            | L2 data cache size (architectures that split L2).                                                                                                                                                                                                                                                   | No direct schema mapping.                                                                          |
| `caches.l2i`          | string            | L2 instruction cache size (architectures that split L2).                                                                                                                                                                                                                                            | No direct schema mapping.                                                                          |
| `caches.l3`           | string            | L3 cache size from `lscpu` (Linux).                                                                                                                                                                                                                                                                 | No direct schema mapping.                                                                          |
| `caches.l4`           | string            | L4 cache size from `lscpu` (high-end server CPUs).                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                          |
| `numa_nodes`          | map[int][]int     | NUMA node id → list of CPU indices, from `lscpu` (Linux).                                                                                                                                                                                                                                           | No direct schema mapping.                                                                          |
| `numa_nodes_count`    | int               | NUMA node count from `lscpu`'s `NUMA node(s):` line.                                                                                                                                                                                                                                                | No direct schema mapping.                                                                          |
| `cpus_online`         | int               | Online CPU count from `lscpu`'s `On-line CPU(s) list:` line.                                                                                                                                                                                                                                        | No direct schema mapping.                                                                          |
| `cpus_offline`        | int               | Offline CPU count from `lscpu`'s `Off-line CPU(s) list:` line.                                                                                                                                                                                                                                      | No direct schema mapping.                                                                          |
| `bios_vendor_id`      | string            | BIOS vendor (e.g. `"Dell Inc."`) from `lscpu`'s `BIOS Vendor ID:` line.                                                                                                                                                                                                                             | No direct schema mapping.                                                                          |
| `bios_model_name`     | string            | Machine model from `lscpu`'s `BIOS Model name:` line.                                                                                                                                                                                                                                               | No direct schema mapping.                                                                          |
| `machine_type`        | string            | s390x mainframe machine type.                                                                                                                                                                                                                                                                       | No direct schema mapping.                                                                          |
| `mhz_max`             | string            | Maximum clock frequency from `lscpu`.                                                                                                                                                                                                                                                               | No direct schema mapping.                                                                          |
| `mhz_min`             | string            | Minimum clock frequency from `lscpu`.                                                                                                                                                                                                                                                               | No direct schema mapping.                                                                          |
| `mhz_dynamic`         | string            | Dynamic (current) clock frequency from `lscpu`.                                                                                                                                                                                                                                                     | No direct schema mapping.                                                                          |
| `bogomips`            | string            | BogoMIPS value from `lscpu`.                                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                          |
| `cpu_opmodes`         | []string          | Supported operation modes (e.g. `["32-bit", "64-bit"]`).                                                                                                                                                                                                                                            | No direct schema mapping.                                                                          |
| `byte_order`          | string            | Endianness reported by `lscpu` (lowercased).                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                          |
| `address_sizes`       | []string          | Physical / virtual address sizes from `lscpu`.                                                                                                                                                                                                                                                      | No direct schema mapping.                                                                          |
| `virtualization`      | string            | Hardware virtualization capability (`"VT-x"`, `"AMD-V"`).                                                                                                                                                                                                                                           | No direct schema mapping.                                                                          |
| `virtualization_type` | string            | Virtualization type when a guest (`"full"`, `"para"`).                                                                                                                                                                                                                                              | No direct schema mapping.                                                                          |
| `hypervisor_vendor`   | string            | Hypervisor vendor (`"KVM"`, `"VMware"`) — signal the virtualization collector can fallback to.                                                                                                                                                                                                      | No direct schema mapping.                                                                          |
| `dispatching_mode`    | string            | s390x dispatching mode (`"horizontal"` / `"vertical"`).                                                                                                                                                                                                                                             | No direct schema mapping.                                                                          |
| `vulnerabilities`     | map[string]string | Mitigation name → status string from `/sys/devices/system/cpu/vulnerabilities/*` (Linux).                                                                                                                                                                                                           | No direct schema mapping.                                                                          |
| `cpus[]`              | []CPU             | Per-logical-CPU breakdown — `vendor_id`, `family`, `model`, `model_name`, `stepping`, `physical_id`, `core_id`, `cores`, `mhz`, `cache_size`, `flags`. Matters on hybrid-core CPUs (Alder Lake P+E, Apple Silicon P+E) where per-core values differ. Mirrors Ohai's `cpu["0"]`, `cpu["1"]` entries. | No direct schema mapping.                                                                          |

## Platform Support

| Platform | Supported                                                                                 |
| -------- | ----------------------------------------------------------------------------------------- |
| Linux    | ✅ (gopsutil `/proc/cpuinfo` + sysfs vulnerabilities + optional `lscpu`)                  |
| macOS    | ✅ (gopsutil sysctl + `hw.physicalcpu` / `hw.packages` / `hw.cpufrequency_max` overrides) |

## Example Output

### Linux dual-socket Xeon

```json
{
  "cpu": {
    "count": 64,
    "sockets": 2,
    "cores": 32,
    "model_name": "Intel(R) Xeon(R) Platinum 8375C CPU @ 2.90GHz",
    "vendor_id": "GenuineIntel",
    "family": "6",
    "model": "106",
    "stepping": 6,
    "mhz": 2900,
    "cache_size": 55296,
    "flags": ["fpu", "vme", "sse4_2", "avx2", "aes"],
    "caches": {
      "l1d": "1.5 MiB",
      "l1i": "1 MiB",
      "l2": "40 MiB",
      "l3": "54 MiB"
    },
    "numa_nodes": {
      "0": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]
    },
    "vulnerabilities": {
      "meltdown": "Not affected",
      "spectre_v1": "Mitigation: usercopy/swapgs barriers and __user pointer sanitization",
      "spectre_v2": "Mitigation: Enhanced IBRS, IBPB conditional, RSB filling"
    }
  }
}
```

### macOS Apple M2 Pro

```json
{
  "cpu": {
    "count": 12,
    "sockets": 1,
    "cores": 10,
    "model_name": "Apple M2 Pro"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("cpu"))
facts, _ := g.Collect(context.Background())

c := facts.CPU
fmt.Printf("%d cores across %d socket(s), %d logical\n", c.Cores, c.Sockets, c.Total)

// Security check: any mitigation reporting "Vulnerable"?
for mitigation, status := range c.Vulnerabilities {
    if strings.HasPrefix(status, "Vulnerable") {
        log.Printf("%s: %s", mitigation, status)
    }
}
```

## Enable/Disable

```bash
gohai --collector.cpu      # enable (default)
gohai --no-collector.cpu   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. gopsutil's `cpu.Info` parses `/proc/cpuinfo` for per-core model, flags, MHz,
   aggregate cache size, vendor, family/model/stepping; `cpu.Counts` gives the
   logical count and the distinct-`PhysicalID` count used for `sockets` on x86.
2. We walk `/sys/devices/system/cpu/vulnerabilities/` via the injected
   `avfs.VFS` and read each file; the basename becomes the map key, the trimmed
   contents become the value. Missing directory yields `vulnerabilities` absent
   from the output.
3. When `lscpu` is on PATH we run it via the injected `executor.Executor` and
   merge: per-level cache sizes (`caches.l1d` / `l1i` / `l2` / `l3`), NUMA node
   → CPU index lists (`numa_nodes`), and — on `s390x` / `ppc64le` only —
   authoritative socket / core / thread counts overriding gopsutil's x86-biased
   numbers. An `lscpu` exec error leaves the extension fields empty; the
   gopsutil-sourced fields remain correct.

On macOS:

1. gopsutil's `cpu.Info` sources `model_name`, `vendor_id`, `flags`, and the
   logical thread count (`total`) via the underlying `host_info()` mach call.
2. We override via `sysctl -n <key>` through the injected `executor.Executor`:
   - `hw.physicalcpu` → `cores` (gopsutil reports logical here).
   - `hw.packages` → `sockets`.
   - `hw.cpufrequency_max`, falling back to `hw.cpufrequency`, divided by 1e6 →
     `mhz`. Both sysctls are absent on Apple Silicon; `mhz` stays at whatever
     gopsutil returned (typically 0) rather than reporting a misleading
     frequency.

## Backing library

- [`github.com/shirou/gopsutil/v4/cpu`](https://github.com/shirou/gopsutil) —
  BSD-3. Primary source for `/proc/cpuinfo` on Linux and mach sysctls on macOS.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for the sysfs
  vulnerabilities walk (production = `osfs`; tests = `memfs`).
- `internal/executor` for `lscpu` on Linux and `sysctl` on macOS (tests use the
  generated gomock).
