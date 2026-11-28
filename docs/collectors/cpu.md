# CPU

> **Status:** Implemented ✅

## Description

The collector reports the CPU topology, the model, the feature flags, the cache
layout, the NUMA layout, and the mitigation status of hardware vulnerabilities.
The vulnerabilities map is the canonical signal of each host for Meltdown,
Spectre, MDS, Retbleed and SRBDS audits. For s390x and ppc64le, the topology
comes from `lscpu`, because `/proc/cpuinfo` does not have all of it. On macOS,
the facts come from `sysctl`, and on Apple Silicon the frequency comes from the
frequency table that `ioreg` shows.

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
the first logical CPU. This is correct on approximately 99% of hosts. Some SKUs
have different types of cores, for example the P and E cores of Apple Silicon or
the hybrid cores of Intel Alder Lake. On these SKUs, the fields give the data of
the first core.

## Collected fields

| Field                 | Type              | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | Schema mapping                                                                                         |
| --------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `count`               | int               | The number of logical CPUs (the threads that the scheduler sees).                                                                                                                                                                                                                                                                                                                                                                                                                                                   | OCSF `device.cpu_count`. The `cpu_` prefix is removed because it repeats the collector name.           |
| `sockets`             | int               | The physical packages. macOS: `hw.packages`. Linux: `physical id` (x86) or `lscpu` (other architectures).                                                                                                                                                                                                                                                                                                                                                                                                           | No direct schema mapping.                                                                              |
| `cores`               | int               | The physical cores of all sockets. macOS: `hw.physicalcpu`. Linux: `/proc/cpuinfo` or `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                      | OCSF `device.cpu_cores`. The leaf is the same as the gohai JSON key.                                   |
| `model_name`          | string            | The CPU name that a person can read, from `/proc/cpuinfo` or sysctl.                                                                                                                                                                                                                                                                                                                                                                                                                                                | OTel `host.cpu.model.name`. OCSF `device.hw_info.cpu_type` is the architecture, not the product model. |
| `vendor_id`           | string            | The CPU vendor (`"GenuineIntel"`, `"AuthenticAMD"`).                                                                                                                                                                                                                                                                                                                                                                                                                                                                | OTel `host.cpu.vendor.id`.                                                                             |
| `family`              | string            | The CPU family number.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | OTel `host.cpu.family`.                                                                                |
| `model_id`            | string            | The model number in the family.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | No direct schema mapping.                                                                              |
| `stepping`            | int32             | The revision of the model.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | OTel `host.cpu.stepping`.                                                                              |
| `speed`               | float64           | The clock in MHz: the maximum when the OS gives one, else the current clock. Linux: cpufreq `cpuinfo_max_freq`, then `cpu MHz` of `/proc/cpuinfo`. macOS: `hw.cpufrequency_max`, then `hw.cpufrequency`. Missing on Apple Silicon.                                                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                              |
| `cache_size`          | int32             | The cache size in KB of the first logical CPU, from `/proc/cpuinfo`. One value, Linux only.                                                                                                                                                                                                                                                                                                                                                                                                                         | No direct schema mapping.                                                                              |
| `flags`               | []string          | The CPU feature flags.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | No direct schema mapping.                                                                              |
| `caches.l1d`          | string            | The L1 data cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | No direct schema mapping.                                                                              |
| `caches.l1i`          | string            | The L1 instruction cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                                                                                                                                                                                                | No direct schema mapping.                                                                              |
| `caches.l2`           | string            | The L2 cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | No direct schema mapping.                                                                              |
| `caches.l2d`          | string            | The L2 data cache size, on architectures that split L2.                                                                                                                                                                                                                                                                                                                                                                                                                                                             | No direct schema mapping.                                                                              |
| `caches.l2i`          | string            | The L2 instruction cache size, on architectures that split L2.                                                                                                                                                                                                                                                                                                                                                                                                                                                      | No direct schema mapping.                                                                              |
| `caches.l3`           | string            | The L3 cache size, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | No direct schema mapping.                                                                              |
| `caches.l4`           | string            | The L4 cache size, from `lscpu` (high-end server CPUs).                                                                                                                                                                                                                                                                                                                                                                                                                                                             | No direct schema mapping.                                                                              |
| `numa_nodes`          | map[int][]int     | The NUMA node id → the list of CPU indices, from `lscpu` (Linux).                                                                                                                                                                                                                                                                                                                                                                                                                                                   | No direct schema mapping.                                                                              |
| `numa_nodes_count`    | int               | The number of NUMA nodes, from the `NUMA node(s):` line of `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                 | No direct schema mapping.                                                                              |
| `online`              | int               | The number of online CPUs, from the `On-line CPU(s) list:` line of `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                         | No direct schema mapping.                                                                              |
| `offline`             | int               | The number of offline CPUs, from the `Off-line CPU(s) list:` line of `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                       | No direct schema mapping.                                                                              |
| `bios_vendor_id`      | string            | The BIOS vendor, for example `"Dell Inc."`, from the `BIOS Vendor ID:` line of `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                             | No direct schema mapping.                                                                              |
| `bios_model_name`     | string            | The machine model, from the `BIOS Model name:` line of `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                     | No direct schema mapping.                                                                              |
| `machine_type`        | string            | The s390x mainframe machine type.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | No direct schema mapping.                                                                              |
| `mhz_max`             | string            | The maximum clock frequency, from `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | No direct schema mapping.                                                                              |
| `mhz_min`             | string            | The minimum clock frequency, from `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | No direct schema mapping.                                                                              |
| `mhz_dynamic`         | string            | The dynamic (current) clock frequency, from `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                | No direct schema mapping.                                                                              |
| `bogomips`            | string            | The BogoMIPS value, from `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | No direct schema mapping.                                                                              |
| `opmodes`             | []string          | The supported operation modes, for example `["32-bit", "64-bit"]`.                                                                                                                                                                                                                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                              |
| `byte_order`          | string            | The endianness that `lscpu` reports, in lowercase.                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                              |
| `address_sizes`       | []string          | The physical and virtual address sizes, from `lscpu`.                                                                                                                                                                                                                                                                                                                                                                                                                                                               | No direct schema mapping.                                                                              |
| `virtualization`      | string            | The hardware virtualization capability (`"VT-x"`, `"AMD-V"`).                                                                                                                                                                                                                                                                                                                                                                                                                                                       | No direct schema mapping.                                                                              |
| `virtualization_type` | string            | The virtualization type, if the host is a guest (`"full"`, `"para"`).                                                                                                                                                                                                                                                                                                                                                                                                                                               | No direct schema mapping.                                                                              |
| `hypervisor_vendor`   | string            | The hypervisor vendor (`"KVM"`, `"VMware"`). The virtualization collector can use this signal as a fallback.                                                                                                                                                                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                              |
| `dispatching_mode`    | string            | The s390x dispatching mode (`"horizontal"` or `"vertical"`).                                                                                                                                                                                                                                                                                                                                                                                                                                                        | No direct schema mapping.                                                                              |
| `vulnerabilities`     | map[string]string | The mitigation name → the status string, from `/sys/devices/system/cpu/vulnerabilities/*` (Linux).                                                                                                                                                                                                                                                                                                                                                                                                                  | No direct schema mapping.                                                                              |
| `cpus[]`              | []CPU             | The data of each logical CPU: `vendor_id`, `family`, `model_id`, `model_name`, `stepping`, `physical_id`, `core_id`, `cores`, `speed`, `cache_size`, `flags`. This is important on hybrid-core CPUs such as Alder Lake P+E, where the values of each core are different. On macOS, `cpus[]` has one entry for the package, not one for each P or E core. On Linux, `cores` is 1 for each logical CPU, as gopsutil gives it, so a sum of `cores` is the thread count. Same as Ohai's `cpu["0"]`, `cpu["1"]` entries. | No direct schema mapping.                                                                              |

## Platform support

| Platform | Supported                                                                   |
| -------- | --------------------------------------------------------------------------- |
| Linux    | ✅ (`/proc/cpuinfo` + sysfs cpufreq and vulnerabilities + optional `lscpu`) |
| macOS    | ✅ (`sysctl`, and `ioreg` for the Apple Silicon frequency)                  |

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
fmt.Printf("%d cores across %d socket(s), %d logical\n", c.Cores, c.Sockets, c.Count)

// Security check: any mitigation reporting "Vulnerable"?
for mitigation, status := range c.Vulnerabilities {
    if strings.HasPrefix(status, "Vulnerable") {
        log.Printf("%s: %s", mitigation, status)
    }
}
```

## Enable/disable

```bash
gohai collect --collector.cpu      # enable (default)
gohai collect --no-collector.cpu   # disable
```

## Dependencies

None.

## Data sources

On Linux, Ohai's `cpu.rb` uses two sources. It parses `/proc/cpuinfo` directly
for the identity of each core (model, flags, MHz, vendor, family, stepping). It
uses `lscpu` for the topology (sockets, NUMA nodes, caches, vulnerabilities,
byte order, op-modes). gohai uses the same two sources. gohai reads
`/proc/cpuinfo` for the base fields, and runs `lscpu` through Executor for the
extended topology fields. On macOS, Ohai reads `sysctl` for the CPU data. gohai
does the same with `sysctl -n` for the cores, the sockets and the frequency.

On Linux:

1. The collector reads `/proc/cpuinfo` through its filesystem. Each numbered
   `processor` line starts the entry of one logical CPU. The entry gives the
   model, the flags, the MHz, the cache size, the vendor, and the family, model
   and stepping. On ARM, `CPU implementer`, `CPU part` and `CPU revision` give
   the vendor, the model and the stepping. If
   `/sys/devices/system/cpu/cpuN/cpufreq/cpuinfo_max_freq` is there, it gives
   the speed, and `topology/core_id` gives the core. The number of entries is
   `count`. If the file has no `processor` lines, the `cpuN` lines of
   `/proc/stat` give `count`. The number of different `physical id` values is
   `sockets` on x86. The number of different (`physical id`, `core id`) pairs is
   `cores`, because each thread of a core is an entry of its own. The collector
   puts each entry into the `cpus[]` slice. Thus the fields of each logical CPU
   stay in the output. This is important on hybrid-core CPUs, where the cores
   are not all the same. It is the same as Ohai's `cpu["0"]`, `cpu["1"]`
   entries.

2. The collector reads each file in `/sys/devices/system/cpu/vulnerabilities/`
   through the injected `avfs.VFS`. The basename is the map key. The trimmed
   contents are the value. If the directory is missing, the output has no
   `vulnerabilities` field. A directory or a file that is there and cannot be
   read adds its error. The collector keeps the facts of `/proc/cpuinfo`, so the
   status is `partial`.

3. If `lscpu` is on PATH, the collector runs it through the injected
   `source.Runner` and merges these fields:

   - The cache size of each level (`caches.l1d` / `l1i` / `l2` / `l3`).
   - The NUMA node → CPU index lists (`numa_nodes`). A node whose CPU list does
     not parse is left out.
   - `Model name`, when `/proc/cpuinfo` gives none, as on arm64.
   - `cores`, as `Socket(s)` times `Core(s) per socket`, when `lscpu` gives
     both. arm64's `/proc/cpuinfo` has no `physical id` or `core id`.
   - On `s390x` and `ppc64le`, the socket and logical CPU counts too. These
     replace the numbers from `/proc/cpuinfo`, which are wrong there.

   If `lscpu` is not on the PATH, the extension fields stay empty, and the
   fields from `/proc/cpuinfo` stay correct. An `lscpu` that is on the PATH and
   fails keeps the facts of `/proc/cpuinfo` with the error, so the status is
   `partial`, and empty extension fields with the status `ok` always mean that
   the host has no `lscpu`. A cancelled context gives no fact.

On macOS:

1. The collector runs `sysctl -n` through its command runner for each key:
   - `hw.logicalcpu` → `count`. If it fails, the collector gives an error.
   - `machdep.cpu.brand_string`, `machdep.cpu.vendor`, `machdep.cpu.family`,
     `machdep.cpu.model` and `machdep.cpu.stepping` → the model, vendor, family,
     model and stepping. Apple Silicon has no family or model, and they are
     empty.
   - `machdep.cpu.features` and `machdep.cpu.leaf7_features`, in lower case →
     `flags`. Apple Silicon has neither.
   - `hw.physicalcpu` → `cores`. If it fails, `cores` is `count`.
   - `hw.packages` → `sockets`.
   - `hw.cpufrequency_max`, or `hw.cpufrequency` if it is missing, divided by
     1e6 → `speed`.
2. Apple Silicon has neither frequency sysctl. Then the collector runs
   `ioreg -l -w0 -c AppleARMIODevice -n pmgr`. The `voltage-states5-sram`
   property is the frequency table of the performance cores: pairs of
   little-endian 32-bit numbers, a frequency in Hz and a voltage. The highest
   frequency, divided by 1e6, is `speed`. If there is no table, `speed` is 0.
3. The collector puts the one entry into the `cpus[]` slice, as on Linux.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read
  `/proc/cpuinfo`, `/proc/stat` and sysfs. Production uses `osfs`. Tests use
  `memfs`.
- `source.Runner` to run `lscpu` on Linux, and `sysctl` and `ioreg` on macOS.
  Tests use the generated gomock.
