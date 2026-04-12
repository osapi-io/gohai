# CPU

> **Status:** Implemented ✅

## Description

Reports CPU topology and model identification — logical CPU count, physical
socket count, cores per socket, model name, family/model/ stepping, frequency,
cache size, and the feature-flag list.

Consumers use this to:

- Pick thread-pool / worker concurrency (`GOMAXPROCS`, task-queue size).
- Drive licensing / compliance checks that count physical sockets.
- Gate features on CPU capabilities (`avx2`, `aes`, `sse4_2`) without running
  probes.
- Identify fleet heterogeneity (different model names / core counts within the
  same tier).

Assumes homogeneous CPUs across sockets — the `model_name`, `vendor_id`,
`family`, `model`, `stepping`, `mhz`, `cache_size`, and `flags` come from the
first logical CPU gopsutil returns. True on ~99% of hosts; asymmetric-core SKUs
(Apple Silicon's P+E cores, Intel Alder Lake's hybrid) report the first core's
data.

## Collected Fields

| Field        | Type     | Description                                          | OCSF mapping                                                                     |
| ------------ | -------- | ---------------------------------------------------- | -------------------------------------------------------------------------------- |
| `total`      | int      | Logical CPU count (SMT-inclusive).                   | `device.cpu_count`.                                                              |
| `real`       | int      | Physical socket count (distinct `PhysicalID`).       | No direct OCSF — nearest is implicit in `device.cpu_count` / `device.cpu_cores`. |
| `cores`      | int      | Total physical cores across all sockets.             | `device.cpu_cores`.                                                              |
| `model_name` | string   | Human-readable CPU name.                             | Nearest: `device.hw_info.cpu_type`.                                              |
| `vendor_id`  | string   | CPU vendor (`"GenuineIntel"`, `"AuthenticAMD"`).     | No direct OCSF.                                                                  |
| `family`     | string   | CPU family number.                                   | No direct OCSF.                                                                  |
| `model`      | string   | Model number within family.                          | No direct OCSF.                                                                  |
| `stepping`   | int32    | Revision of the model.                               | No direct OCSF.                                                                  |
| `mhz`        | float64  | Current clock in MHz.                                | No direct OCSF.                                                                  |
| `cache_size` | int32    | L2/L3 cache size in KB (gopsutil exposes one value). | No direct OCSF.                                                                  |
| `flags`      | []string | CPU feature flags.                                   | No direct OCSF.                                                                  |

## Platform Support

| Platform | Supported                   |
| -------- | --------------------------- |
| Linux    | ✅ (parses `/proc/cpuinfo`) |
| macOS    | ✅ (sysctl `machdep.cpu.*`) |

## Example Output

### Linux dual-socket Xeon

```json
{
  "cpu": {
    "total": 64,
    "real": 2,
    "cores": 32,
    "model_name": "Intel(R) Xeon(R) Platinum 8375C CPU @ 2.90GHz",
    "vendor_id": "GenuineIntel",
    "family": "6",
    "model": "106",
    "stepping": 6,
    "mhz": 2900,
    "cache_size": 55296,
    "flags": ["fpu", "vme", "sse4_2", "avx2", "aes"]
  }
}
```

### macOS Apple M2 Pro

```json
{
  "cpu": {
    "total": 12,
    "real": 1,
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
fmt.Printf("%d cores across %d socket(s), %d logical\n", c.Cores, c.Real, c.Total)
```

## Enable/Disable

```bash
gohai --collector.cpu      # enable (default)
gohai --no-collector.cpu   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                                                       | Ohai plugin                                                                                                                     | Alignment                                                                                                                                                                                           |
| -------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `cpu.InfoWithContext` + `cpu.CountsWithContext` (parses `/proc/cpuinfo`). | [`linux/cpu.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/cpu.rb) — `/proc/cpuinfo` + `lscpu`.             | **Same primary source (`/proc/cpuinfo`).** Ohai additionally runs `lscpu` for NUMA topology, per-cache breakdown (L1d/L1i/L2/L3), mitigations, and per-logical-CPU maps — we report aggregate only. |
| macOS    | gopsutil `cpu.InfoWithContext` (sysctl).                                           | [`darwin/cpu.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/darwin/cpu.rb) — sysctl `hw.ncpu` / `hw.physicalcpu`. | **Equivalent** on the fields we capture.                                                                                                                                                            |

**Known gaps vs. Ohai:** NUMA map; per-cache-level sizes (L1d/L1i/L2/ L3
distinct); `vulnerabilities` map (Meltdown/Spectre mitigation state — Ohai reads
`/sys/devices/system/cpu/vulnerabilities/*`). Planned as follow-ups.

## Backing library

- [`github.com/shirou/gopsutil/v4/cpu`](https://github.com/shirou/gopsutil) —
  BSD-3.
