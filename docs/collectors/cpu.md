# CPU

> **Status:** Implemented ✅

## Description

The collector reports the CPU topology, the model, the feature flags, the cache
layout, the NUMA layout, and the mitigation status of hardware vulnerabilities.
The vulnerabilities map is the canonical signal of each host for Meltdown,
Spectre, MDS, Retbleed and SRBDS audits. For s390x and ppc64le, the topology
comes from `lscpu`, because `/proc/cpuinfo` does not have all of it. On macOS,
the core and frequency facts come from `sysctl`, not from gopsutil. On Apple
Silicon, gopsutil reports logical cores as physical cores and reports a
frequency of zero.

Consumers use this data to:

- Select the concurrency of a thread pool or of workers (`GOMAXPROCS`, the size
  of a task queue).
- Do licensing and compliance checks that count physical sockets.
- Enable features from the CPU capabilities (`avx2`, `aes`, `sse4_2`) without
  probes.
- Audit the Meltdown and Spectre mitigations of a fleet.
- Find differences in a fleet, for example different model names or core counts
  in the same tier.

The collector assumes that all sockets have the same CPU. `model_name`,
`vendor_id`, `family`, `model`, `stepping`, `cache_size` and `flags` come from
the first logical CPU that gopsutil returns. This is correct on approximately
99% of hosts. On SKUs with different types of cores (the P and E cores of Apple
Silicon, the hybrid cores of Intel Alder Lake), these fields give the data of
the first core.

## Collected fields

| Field                 | Type              | Description                                                                                                                                                                                                                                                                                                                          | Schema mapping                                                                                         |
| --------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `count`               | int               | The number of logical CPUs (the threads that the scheduler sees).                                                                                                                                                                                                                                                                    | OCSF `device.cpu_count`. The `cpu_` prefix is removed because it repeats the collector name.           |
| `sockets`             | int               | The physical packages. macOS: `hw.packages`. Linux: gopsutil (x86) or `lscpu` (other architectures).                                                                                                                                                                                                                                 | No direct schema mapping.                                                                              |
| `cores`               | int               | The physical cores of all sockets. macOS: `hw.physicalcpu`. Linux: gopsutil or `lscpu`.                                                                                                                                                                                                                                              | OCSF `device.cpu_cores`. The leaf is the same as the gohai JSON key.                                   |
| `model_name`          | string            | The CPU name that a person can read, from `/proc/cpuinfo` or sysctl.                                                                                                                                                                                                                                                                 | OTel `host.cpu.model.name`. OCSF `device.hw_info.cpu_type` is the architecture, not the product model. |
| `vendor_id`           | string            | The CPU vendor (`"GenuineIntel"`, `"AuthenticAMD"`).                                                                                                                                                                                                                                                                                 | OTel `host.cpu.vendor.id`.                                                                             |
| `family`              | string            | The CPU family number.                                                                                                                                                                                                                                                                                                               | OTel `host.cpu.family`.                                                                                |
| `model_id`            | string            | The model number in the family.                                                                                                                                                                                                                                                                                                      | No direct schema mapping.                                                                              |
| `stepping`            | int32             | The revision of the model.                                                                                                                                                                                                                                                                                                           | OTel `host.cpu.stepping`.                                                                              |
| `speed`               | float64           | The current clock in MHz. macOS: `hw.cpufrequency_max`, then `hw.cpufrequency`. Missing on Apple Silicon.                                                                                                                                                                                                                            | No direct schema mapping.                                                                              |
| `cache_size`          | int32             | The total cache size in KB, from `/proc/cpuinfo`. One value, Linux only.                                                                                                                                                                                                                                                             | No direct schema mapping.                                                                              |
| `flags`               | []string          | The CPU feature flags.                                                                                                                                                                                                                                                                                                               | No direct schema mapping.                                                                              |
| `caches.l1d`          | string            | The L1 data cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                              |
| `caches.l1i`          | string            | The L1 instruction cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                 | No direct schema mapping.                                                                              |
| `caches.l2`           | string            | The L2 cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                             | No direct schema mapping.                                                                              |
| `caches.l2d`          | string            | The L2 data cache size, on architectures that split L2.                                                                                                                                                                                                                                                                              | No direct schema mapping.                                                                              |
| `caches.l2i`          | string            | The L2 instruction cache size, on architectures that split L2.                                                                                                                                                                                                                                                                       | No direct schema mapping.                                                                              |
| `caches.l3`           | string            | The L3 cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                             | No direct schema mapping.                                                                              |
| `caches.l4`           | string            | The L4 cache size, from `lscpu` (high-end server CPUs).                                                                                                                                                                                                                                                                              | No direct schema mapping.                                                                              |
| `numa_nodes`          | map[int][]int     | The NUMA node id → the list of CPU indices, from `lscpu` (Linux).                                                                                                                                                                                                                                                                    | No direct schema mapping.                                                                              |
| `numa_nodes_count`    | int               | The number of NUMA nodes, from the `NUMA node(s):` line of `lscpu`.                                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                              |
| `online`              | int               | The number of online CPUs, from the `On-line CPU(s) list:` line of `lscpu`.                                                                                                                                                                                                                                                          | No direct schema mapping.                                                                              |
| `offline`             | int               | The number of offline CPUs, from the `Off-line CPU(s) list:` line of `lscpu`.                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                              |
| `bios_vendor_id`      | string            | The BIOS vendor, for example `"Dell Inc."`, from the `BIOS Vendor ID:` line of `lscpu`.                                                                                                                                                                                                                                              | No direct schema mapping.                                                                              |
| `bios_model_name`     | string            | The machine model, from the `BIOS Model name:` line of `lscpu`.                                                                                                                                                                                                                                                                      | No direct schema mapping.                                                                              |
| `machine_type`        | string            | The s390x mainframe machine type.                                                                                                                                                                                                                                                                                                    | No direct schema mapping.                                                                              |
| `mhz_max`             | string            | The maximum clock frequency, from `lscpu`.                                                                                                                                                                                                                                                                                           | No direct schema mapping.                                                                              |
| `mhz_min`             | string            | The minimum clock frequency, from `lscpu`.                                                                                                                                                                                                                                                                                           | No direct schema mapping.                                                                              |
| `mhz_dynamic`         | string            | The dynamic (current) clock frequency, from `lscpu`.                                                                                                                                                                                                                                                                                 | No direct schema mapping.                                                                              |
| `bogomips`            | string            | The BogoMIPS value, from `lscpu`.                                                                                                                                                                                                                                                                                                    | No direct schema mapping.                                                                              |
| `opmodes`             | []string          | The supported operation modes, for example `["32-bit", "64-bit"]`.                                                                                                                                                                                                                                                                   | No direct schema mapping.                                                                              |
| `byte_order`          | string            | The endianness that `lscpu` reports, in lowercase.                                                                                                                                                                                                                                                                                   | No direct schema mapping.                                                                              |
| `address_sizes`       | []string          | The physical and virtual address sizes, from `lscpu`.                                                                                                                                                                                                                                                                                | No direct schema mapping.                                                                              |
| `virtualization`      | string            | The hardware virtualization capability (`"VT-x"`, `"AMD-V"`).                                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                              |
| `virtualization_type` | string            | The virtualization type, if the host is a guest (`"full"`, `"para"`).                                                                                                                                                                                                                                                                | No direct schema mapping.                                                                              |
| `hypervisor_vendor`   | string            | The hypervisor vendor (`"KVM"`, `"VMware"`). The virtualization collector can use this signal as a fallback.                                                                                                                                                                                                                         | No direct schema mapping.                                                                              |
| `dispatching_mode`    | string            | The s390x dispatching mode (`"horizontal"` or `"vertical"`).                                                                                                                                                                                                                                                                         | No direct schema mapping.                                                                              |
| `vulnerabilities`     | map[string]string | The mitigation name → the status string, from `/sys/devices/system/cpu/vulnerabilities/*` (Linux).                                                                                                                                                                                                                                   | No direct schema mapping.                                                                              |
| `cpus[]`              | []CPU             | The data of each logical CPU: `vendor_id`, `family`, `model_id`, `model_name`, `stepping`, `physical_id`, `core_id`, `cores`, `speed`, `cache_size`, `flags`. This is important on hybrid-core CPUs (Alder Lake P+E, Apple Silicon P+E), where the values of each core are different. Same as Ohai's `cpu["0"]`, `cpu["1"]` entries. | No direct schema mapping.                                                                              |

## Platform support

| Platform | Supported                                                                                 |
| -------- | ----------------------------------------------------------------------------------------- |
| Linux    | ✅ (gopsutil `/proc/cpuinfo` + sysfs vulnerabilities + optional `lscpu`)                  |
| macOS    | ✅ (gopsutil sysctl + `hw.physicalcpu` / `hw.packages` / `hw.cpufrequency_max` overrides) |

## Example output

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
    "model_id": "106",
    "stepping": 6,
    "speed": 2900,
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

## SDK usage

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

## Enable/disable

```bash
gohai --collector.cpu      # enable (default)
gohai --no-collector.cpu   # disable
```

## Dependencies

None.

## Data sources

On Linux, Ohai's `cpu.rb` uses two sources. It parses `/proc/cpuinfo` directly
for the identity of each core (model, flags, MHz, vendor, family, stepping). It
uses `lscpu` for the topology (sockets, NUMA nodes, caches, vulnerabilities,
byte order, op-modes). gohai uses the same two sources. gopsutil reads
`/proc/cpuinfo` for the base fields. gohai runs `lscpu` through Executor for the
extended topology fields. On macOS, Ohai reads `sysctl` for the CPU data. gohai
does the same with `sysctl -n` for the cores, the sockets and the frequency.

On Linux:

1. gopsutil's `cpu.Info` parses `/proc/cpuinfo`. It gives the model, the flags,
   the MHz, the total cache size, the vendor, and the family, model and stepping
   of each core. `cpu.Counts` gives the logical count. It also gives the number
   of different `PhysicalID` values, which is `sockets` on x86. The collector
   also puts each `cpu.Info` entry into the `cpus[]` slice. Thus the fields of
   each logical CPU stay in the output. This is important on hybrid-core CPUs,
   where the cores are not all the same. It is the same as Ohai's `cpu["0"]`,
   `cpu["1"]` entries.

2. The collector reads each file in `/sys/devices/system/cpu/vulnerabilities/`
   through the injected `avfs.VFS`. The basename is the map key. The trimmed
   contents are the value. If the directory is missing, the output has no
   `vulnerabilities` field.

3. If `lscpu` is on PATH, the collector runs it through the injected
   `executor.Executor` and merges these fields:

   - The cache size of each level (`caches.l1d` / `l1i` / `l2` / `l3`).
   - The NUMA node → CPU index lists (`numa_nodes`).
   - On `s390x` and `ppc64le` only, the socket, core and thread counts. These
     replace the gopsutil numbers, which are correct only for x86.

   If `lscpu` fails, the extension fields stay empty. The fields from gopsutil
   stay correct.

On macOS:

1. gopsutil's `cpu.Info` gives `model_name`, `vendor_id`, `flags` and the
   logical thread count (`total`) through the `host_info()` mach call. The
   collector puts each entry into the `cpus[]` slice, as on Linux. The summary
   fields, which assume that all CPUs are the same, come from the first entry.
2. The collector replaces these values with `sysctl -n <key>` through the
   injected `executor.Executor`:
   - `hw.physicalcpu` → `cores`. gopsutil reports logical cores here.
   - `hw.packages` → `sockets`.
   - `hw.cpufrequency_max`, or `hw.cpufrequency` if it is missing, divided by
     1e6 → `mhz`. Apple Silicon has neither sysctl. Then `mhz` keeps the value
     from gopsutil (usually 0). The collector does not report an incorrect
     frequency.

## Backing library

- [`github.com/shirou/gopsutil/v4/cpu`](https://github.com/shirou/gopsutil),
  BSD-3. The primary source for `/proc/cpuinfo` on Linux and for the mach
  sysctls on macOS.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read the sysfs
  vulnerabilities. Production uses `osfs`. Tests use `memfs`.
- `internal/executor` to run `lscpu` on Linux and `sysctl` on macOS. Tests use
  the generated gomock.
