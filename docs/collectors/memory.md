# Memory

> **Status:** Implemented ✅

## Description

Reports system memory totals, usage buckets, kernel allocations, page-cache
state, and hugepages layout on Linux; on macOS it adds the `wired` /
`speculative` / `compressed` buckets that the Darwin VM reports. Consumers use
the full picture to size workloads, debug OOM/overcommit events, detect kernel
leaks (Slab growth), and audit hugepages configuration for databases or DPDK
workloads.

All byte-valued fields are **native bytes** — Ohai emits kB-suffixed strings
(`"total": "16384000kB"`); we chose bytes for Go ergonomics. Deliberate shape
deviation, not a collection difference.

Consumers use this to:

- Pick in-memory cache sizes proportional to host RAM.
- Alert on `used_percent` crossing a threshold.
- Detect swap pressure (non-zero `swap.used` on hosts that shouldn't be
  swapping).
- Correlate kernel memory (Slab, PageTables) growth against leaks.
- Audit hugepages configuration for DPDK / Java / database workloads.
- On Apple Silicon, track `compressed` memory pressure as the primary signal
  (the compressor is aggressively used on macOS).

## Collected Fields

| Field                      | Type    | Description                                               | Schema mapping                   |
| -------------------------- | ------- | --------------------------------------------------------- | -------------------------------- |
| `total`                    | uint64  | Total physical memory in bytes.                           | `device.hw_info.ram_size_bytes`. |
| `available`                | uint64  | Memory available to new allocations.                      | No direct OCSF.                  |
| `used`                     | uint64  | Used memory in bytes.                                     | No direct OCSF.                  |
| `used_percent`             | float64 | Percent of `total` used (0–100).                          | No direct OCSF.                  |
| `free`                     | uint64  | Free memory in bytes.                                     | No direct OCSF.                  |
| `active`                   | uint64  | Active LRU pages (bytes).                                 | No direct OCSF.                  |
| `inactive`                 | uint64  | Inactive LRU pages (bytes).                               | No direct OCSF.                  |
| `active_anon`              | uint64  | Active anonymous pages (Linux).                           | No direct OCSF.                  |
| `inactive_anon`            | uint64  | Inactive anonymous pages (Linux).                         | No direct OCSF.                  |
| `active_file`              | uint64  | Active file-backed pages (Linux).                         | No direct OCSF.                  |
| `inactive_file`            | uint64  | Inactive file-backed pages (Linux).                       | No direct OCSF.                  |
| `unevictable`              | uint64  | Unevictable pages (mlock, ramfs) (Linux).                 | No direct OCSF.                  |
| `wired`                    | uint64  | Non-swappable pages (macOS primary signal). Linux: 0.     | No direct OCSF.                  |
| `speculative`              | uint64  | macOS speculative pages (prefetch-style cache). Linux: 0. | No direct OCSF.                  |
| `compressed`               | uint64  | macOS memory in the compressor. Linux: 0.                 | No direct OCSF.                  |
| `buffers`                  | uint64  | Block-layer buffers (Linux).                              | No direct OCSF.                  |
| `cached`                   | uint64  | Page cache (Linux).                                       | No direct OCSF.                  |
| `dirty`                    | uint64  | Pages awaiting writeback.                                 | No direct OCSF.                  |
| `writeback`                | uint64  | Pages being written back.                                 | No direct OCSF.                  |
| `writeback_tmp`            | uint64  | Temporary writeback memory used by FUSE.                  | No direct OCSF.                  |
| `shared`                   | uint64  | Shared memory (tmpfs + SysV shm).                         | No direct OCSF.                  |
| `mapped`                   | uint64  | mmap'd pages.                                             | No direct OCSF.                  |
| `slab`                     | uint64  | Total slab allocator.                                     | No direct OCSF.                  |
| `s_reclaimable`            | uint64  | Reclaimable slab.                                         | No direct OCSF.                  |
| `s_unreclaim`              | uint64  | Unreclaimable slab.                                       | No direct OCSF.                  |
| `k_reclaimable`            | uint64  | Other kernel-reclaimable memory (Linux).                  | No direct OCSF.                  |
| `page_tables`              | uint64  | Page-table allocations.                                   | No direct OCSF.                  |
| `kernel_stack`             | uint64  | Kernel stack allocations (Linux).                         | No direct OCSF.                  |
| `percpu`                   | uint64  | Per-CPU allocations (Linux).                              | No direct OCSF.                  |
| `anon_pages`               | uint64  | Anonymous (non-file-backed) pages (Linux).                | No direct OCSF.                  |
| `shmem`                    | uint64  | Shared memory (tmpfs + SysV shm) (Linux).                 | No direct OCSF.                  |
| `commit_limit`             | uint64  | Overcommit ceiling.                                       | No direct OCSF.                  |
| `committed_as`             | uint64  | Committed address space.                                  | No direct OCSF.                  |
| `vmalloc_total`            | uint64  | vmalloc arena size.                                       | No direct OCSF.                  |
| `vmalloc_used`             | uint64  | vmalloc in use.                                           | No direct OCSF.                  |
| `vmalloc_chunk`            | uint64  | Largest free vmalloc chunk.                               | No direct OCSF.                  |
| `hugepages.total`          | uint64  | `HugePages_Total` — configured pages.                     | No direct OCSF.                  |
| `hugepages.free`           | uint64  | `HugePages_Free`.                                         | No direct OCSF.                  |
| `hugepages.reserved`       | uint64  | `HugePages_Rsvd`.                                         | No direct OCSF.                  |
| `hugepages.surplus`        | uint64  | `HugePages_Surp`.                                         | No direct OCSF.                  |
| `hugepages.size`           | uint64  | `Hugepagesize` in bytes.                                  | No direct OCSF.                  |
| `hugepages.anon_hugepages` | uint64  | `AnonHugePages`.                                          | No direct OCSF.                  |
| `hugepages.hugetlb`        | uint64  | `Hugetlb` — total memory in hugepages (Linux).            | No direct OCSF.                  |
| `direct_map.map_4k`        | uint64  | Physical memory mapped with 4k pages (Linux).             | No direct OCSF.                  |
| `direct_map.map_2m`        | uint64  | Physical memory mapped with 2M pages (Linux).             | No direct OCSF.                  |
| `direct_map.map_1g`        | uint64  | Physical memory mapped with 1G pages (Linux).             | No direct OCSF.                  |
| `swap.total`               | uint64  | Total swap. Omitted when no swap.                         | No direct OCSF.                  |
| `swap.used`                | uint64  | Swap used.                                                | No direct OCSF.                  |
| `swap.free`                | uint64  | Swap free.                                                | No direct OCSF.                  |
| `swap.used_percent`        | float64 | Percent of swap used.                                     | No direct OCSF.                  |
| `swap.cached`              | uint64  | `SwapCached` from `/proc/meminfo` (Linux).                | No direct OCSF.                  |

## Platform Support

| Platform | Supported                                                |
| -------- | -------------------------------------------------------- |
| Linux    | ✅ (`/proc/meminfo` via gopsutil — 27+ forwarded fields) |
| macOS    | ✅ (`host_statistics64` + `vm_stat` via executor)        |

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
    "active": 4000000000,
    "inactive": 2000000000,
    "buffers": 134217728,
    "cached": 8053063680,
    "dirty": 1048576,
    "slab": 268435456,
    "s_reclaimable": 201326592,
    "s_unreclaim": 67108864,
    "page_tables": 16777216,
    "commit_limit": 10485760000,
    "committed_as": 6500000000,
    "hugepages": {
      "total": 512,
      "free": 512,
      "size": 2097152
    },
    "swap": {
      "total": 2147483648,
      "used": 0,
      "free": 2147483648,
      "used_percent": 0.0,
      "cached": 0
    }
  }
}
```

### macOS (Apple Silicon)

```json
{
  "memory": {
    "total": 17179869184,
    "available": 6442450944,
    "used": 10737418240,
    "used_percent": 62.5,
    "free": 2147483648,
    "active": 5000000000,
    "inactive": 2000000000,
    "wired": 3000000000,
    "speculative": 134217728,
    "compressed": 402653184
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
if m.Compressed > 0 {
    fmt.Printf("macOS compressor holds %d MiB\n", m.Compressed/(1<<20))
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

On Linux:

1. gopsutil `mem.VirtualMemory` reads `/proc/meminfo` and exposes 27+ fields on
   its `VirtualMemoryStat` struct (Total, Available, Used, Free, Active,
   Inactive, Buffers, Cached, Dirty, WriteBack, WriteBackTmp, Shared, Mapped,
   Slab, Sreclaimable, Sunreclaim, PageTables, SwapCached, CommitLimit,
   CommittedAS, VmallocTotal, VmallocUsed, VmallocChunk, HugePagesTotal,
   HugePagesFree, HugePagesRsvd, HugePagesSurp, HugePageSize, AnonHugePages). We
   forward every relevant field — library-first, no separate `/proc/meminfo`
   parse.
2. gopsutil `mem.SwapMemory` provides the `swap.*` totals. `swap.cached` comes
   from the `SwapCached` field on the virtual memory stat.
3. `hugepages.*` is populated only when any hugepage field is present (kernels
   without hugepages support skip it cleanly).
4. We additionally read `/proc/meminfo` through the injected `avfs.VFS` to pick
   up the fields gopsutil's cross-platform struct doesn't expose:
   `Active(anon)`, `Active(file)`, `Inactive(anon)`, `Inactive(file)`,
   `Unevictable`, `KernelStack`, `Percpu`, `KReclaimable`, `AnonPages`, `Shmem`,
   `Hugetlb`, and `DirectMap4k` / `DirectMap2M` / `DirectMap1G`. All values are
   kernel-reported kB; we multiply by 1024 to emit bytes. This closes the
   Ohai-methodology gap without rolling a separate parser for the 27 fields
   gopsutil already handles — extension on top of the library, per CLAUDE.md's
   library-first principle.

On macOS:

1. gopsutil `mem.VirtualMemory` calls `host_statistics64` for Total, Available,
   Used, Free, Active, Inactive, and Wired.
2. We additionally run `vm_stat` through the shared `internal/executor` runner.
   The first line's `page size of N bytes` header is parsed (4096 on Intel,
   16384 on Apple Silicon); each subsequent `Pages <label>: <int>.` line yields
   a byte count by multiplying pages × page-size. We populate:
   - `speculative` from `Pages speculative` (macOS-specific cache class; no
     Linux equivalent).
   - `compressed` from `Pages stored in compressor` (the aggressively-used
     compressor on Apple Silicon).
3. When `vm_stat` is absent or errors, the extended fields stay zero-valued; the
   gopsutil-sourced totals remain correct.

## Backing library

- [`github.com/shirou/gopsutil/v4/mem`](https://github.com/shirou/gopsutil) —
  BSD-3. Primary source on both platforms.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `vm_stat` on macOS. Tests mock it with
  `go.uber.org/mock`.
