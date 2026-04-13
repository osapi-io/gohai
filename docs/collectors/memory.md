# Memory

> **Status:** Implemented ✅

## Description

Reports virtual and swap memory usage — total, used, free, and the derived
percent. Linux additionally surfaces `buffers` and `cached` (page-cache
breakdowns from `/proc/meminfo`); swap is reported when a swap device is active.

All byte-valued fields are **native bytes** — Ohai emits kB-suffixed strings
(`"total": "16384000kB"`); we chose bytes for Go ergonomics. This is a
deliberate shape deviation, not a collection difference.

Consumers use this to:

- Pick in-memory cache sizes proportional to host RAM.
- Alert on `used_percent` crossing a threshold.
- Detect swap pressure (non-zero `swap.used` on hosts that shouldn't be
  swapping).

## Collected Fields

| Field               | Type    | Description                                | Schema mapping |
| ------------------- | ------- | ------------------------------------------ | -------------------------------- |
| `total`             | uint64  | Total physical memory in bytes.            | `device.hw_info.ram_size_bytes`. |
| `available`         | uint64  | Memory available to new allocations.       | No direct OCSF.                  |
| `used`              | uint64  | Used memory in bytes.                      | No direct OCSF.                  |
| `used_percent`      | float64 | Percent of `total` that is used (0–100).   | No direct OCSF.                  |
| `free`              | uint64  | Free memory in bytes.                      | No direct OCSF.                  |
| `buffers`           | uint64  | Linux buffer cache (bytes). Linux only.    | No direct OCSF.                  |
| `cached`            | uint64  | Linux page cache (bytes). Linux only.      | No direct OCSF.                  |
| `swap.total`        | uint64  | Total swap in bytes. Omitted when no swap. | No direct OCSF.                  |
| `swap.used`         | uint64  | Swap used in bytes.                        | No direct OCSF.                  |
| `swap.free`         | uint64  | Swap free in bytes.                        | No direct OCSF.                  |
| `swap.used_percent` | float64 | Percent of swap used.                      | No direct OCSF.                  |

## Platform Support

| Platform | Supported                                        |
| -------- | ------------------------------------------------ |
| Linux    | ✅ (`/proc/meminfo` via gopsutil)                |
| macOS    | ✅ (`host_statistics64` / `sysctl vm.swapusage`) |

## Example Output

### Linux

```json
{
  "memory": {
    "total": 16777216000,
    "available": 9437184000,
    "used": 6710886400,
    "used_percent": 40.0,
    "free": 629145600,
    "buffers": 134217728,
    "cached": 8053063680,
    "swap": {
      "total": 2147483648,
      "used": 0,
      "free": 2147483648,
      "used_percent": 0.0
    }
  }
}
```

### macOS

```json
{
  "memory": {
    "total": 17179869184,
    "available": 6442450944,
    "used": 10737418240,
    "used_percent": 62.5,
    "free": 2147483648
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("memory"))
facts, _ := g.Collect(context.Background())

m := facts.Memory
fmt.Printf("%.1f%% of %d GiB used\n", m.UsedPercent, m.Total/(1<<30))
if m.Swap != nil && m.Swap.Used > 0 {
    log.Println("swap in use")
}
```

## Enable/Disable

```bash
gohai --collector.memory      # enable (default)
gohai --no-collector.memory   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                                                                   | Ohai plugin                                                                                                            | Alignment                                                                                                                                                                                                                                                                               |
| -------- | ---------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `mem.VirtualMemory` + `mem.SwapMemory` (both read `/proc/meminfo`).                   | [`linux/memory.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/memory.rb) — `/proc/meminfo` parser. | **Same source of truth (`/proc/meminfo`).** Ohai surfaces many more derived fields: `active`, `inactive`, `dirty`, `writeback`, `slab`, `hugepages.{total,free,reserved,surplus,size}`, `swap.cached`. Those are tracked as follow-ups (issue #48 in the repo's in-progress task list). |
| macOS    | gopsutil `mem.VirtualMemory` + `mem.SwapMemory` (`host_statistics64` + `sysctl vm.swapusage`). | [`darwin/memory.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/darwin/memory.rb) — `vm_stat` + `sysctl`. | **Equivalent on what we collect.** Ohai additionally breaks out `active`/`inactive`/`wired`/`speculative` — deferred.                                                                                                                                                                   |

**Known gaps vs. Ohai:** `active`, `inactive`, `wired` (macOS), `dirty`,
`writeback`, `slab`, `hugepages.*`, `swap.cached`. Planned.

## Backing library

- [`github.com/shirou/gopsutil/v4/mem`](https://github.com/shirou/gopsutil) —
  BSD-3.
