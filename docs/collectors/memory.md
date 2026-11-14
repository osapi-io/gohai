# Memory

> **Status:** Implemented ✅

## Description

On Linux, the collector reports the system memory totals, the usage buckets, the
kernel allocations, the page-cache state and the hugepages layout. On macOS, it
adds the `wired`, `speculative` and `compressed` buckets that the Darwin VM
reports. Consumers use all of this data to size workloads, to debug OOM and
overcommit events, to find kernel leaks (Slab growth), and to audit the
hugepages configuration for databases or DPDK workloads.

All fields with a byte value are in **native bytes**. Ohai reports strings with
a kB suffix (`"total": "16384000kB"`). gohai uses bytes because they are easier
to use in Go. This is an intentional difference in shape, not a difference in
how the collector gets the data.

Consumers use this data to:

- Select in-memory cache sizes in proportion to the host RAM.
- Send an alert when `used_percent` goes above a threshold.
- Find swap pressure: a non-zero `swap.used` on hosts that must not swap.
- Compare the growth of kernel memory (Slab, PageTables) with leaks.
- Audit the hugepages configuration for DPDK, Java and database workloads.
- On Apple Silicon, monitor the `compressed` memory pressure as the primary
  signal. macOS uses the compressor much.

## Collected fields

| Field                      | Type    | Description                                                                         | Schema mapping                   |
| -------------------------- | ------- | ----------------------------------------------------------------------------------- | -------------------------------- |
| `total`                    | uint64  | The total physical memory in bytes.                                                 | `device.hw_info.ram_size_bytes`. |
| `available`                | uint64  | The memory that is available for new allocations.                                   | No direct schema mapping.        |
| `used`                     | uint64  | The used memory in bytes.                                                           | No direct schema mapping.        |
| `used_percent`             | float64 | The percent of `total` that is used (0 to 100).                                     | No direct schema mapping.        |
| `free`                     | uint64  | The free memory in bytes.                                                           | No direct schema mapping.        |
| `active`                   | uint64  | The active LRU pages, in bytes.                                                     | No direct schema mapping.        |
| `inactive`                 | uint64  | The inactive LRU pages, in bytes.                                                   | No direct schema mapping.        |
| `active_anon`              | uint64  | The active anonymous pages (Linux).                                                 | No direct schema mapping.        |
| `inactive_anon`            | uint64  | The inactive anonymous pages (Linux).                                               | No direct schema mapping.        |
| `active_file`              | uint64  | The active pages that a file backs (Linux).                                         | No direct schema mapping.        |
| `inactive_file`            | uint64  | The inactive pages that a file backs (Linux).                                       | No direct schema mapping.        |
| `unevictable`              | uint64  | The pages that the kernel cannot evict, for example mlock and ramfs (Linux).        | No direct schema mapping.        |
| `wired`                    | uint64  | The pages that cannot go to swap. This is the primary signal on macOS. On Linux, 0. | No direct schema mapping.        |
| `speculative`              | uint64  | The macOS speculative pages, a cache similar to prefetch. On Linux, 0.              | No direct schema mapping.        |
| `compressed`               | uint64  | The macOS memory in the compressor. On Linux, 0.                                    | No direct schema mapping.        |
| `buffers`                  | uint64  | The block-layer buffers (Linux).                                                    | No direct schema mapping.        |
| `cached`                   | uint64  | The page cache (Linux).                                                             | No direct schema mapping.        |
| `dirty`                    | uint64  | The pages that wait for writeback.                                                  | No direct schema mapping.        |
| `writeback`                | uint64  | The pages that the kernel writes back now.                                          | No direct schema mapping.        |
| `writeback_tmp`            | uint64  | The temporary writeback memory that FUSE uses.                                      | No direct schema mapping.        |
| `shared`                   | uint64  | The shared memory (tmpfs and SysV shm).                                             | No direct schema mapping.        |
| `mapped`                   | uint64  | The pages that mmap maps.                                                           | No direct schema mapping.        |
| `slab`                     | uint64  | The total memory of the slab allocator.                                             | No direct schema mapping.        |
| `s_reclaimable`            | uint64  | The slab memory that the kernel can reclaim.                                        | No direct schema mapping.        |
| `s_unreclaim`              | uint64  | The slab memory that the kernel cannot reclaim.                                     | No direct schema mapping.        |
| `k_reclaimable`            | uint64  | Other kernel memory that the kernel can reclaim (Linux).                            | No direct schema mapping.        |
| `page_tables`              | uint64  | The page-table allocations.                                                         | No direct schema mapping.        |
| `kernel_stack`             | uint64  | The kernel stack allocations (Linux).                                               | No direct schema mapping.        |
| `percpu`                   | uint64  | The allocations for each CPU (Linux).                                               | No direct schema mapping.        |
| `anon_pages`               | uint64  | The anonymous pages, which no file backs (Linux).                                   | No direct schema mapping.        |
| `shmem`                    | uint64  | The shared memory (tmpfs and SysV shm) (Linux).                                     | No direct schema mapping.        |
| `high_total`               | uint64  | The total high memory of a 32-bit kernel. Zero on 64-bit.                           | No direct schema mapping.        |
| `high_free`                | uint64  | The free high memory of a 32-bit kernel. Zero on 64-bit.                            | No direct schema mapping.        |
| `low_total`                | uint64  | The total low memory of a 32-bit kernel. Zero on 64-bit.                            | No direct schema mapping.        |
| `low_free`                 | uint64  | The free low memory of a 32-bit kernel. Zero on 64-bit.                             | No direct schema mapping.        |
| `nfs_unstable`             | uint64  | The NFS pages that wait for a commit to the server.                                 | No direct schema mapping.        |
| `bounce`                   | uint64  | The bounce-buffer memory (legacy DMA).                                              | No direct schema mapping.        |
| `commit_limit`             | uint64  | The overcommit limit.                                                               | No direct schema mapping.        |
| `committed_as`             | uint64  | The committed address space.                                                        | No direct schema mapping.        |
| `vmalloc_total`            | uint64  | The size of the vmalloc area.                                                       | No direct schema mapping.        |
| `vmalloc_used`             | uint64  | The vmalloc memory in use.                                                          | No direct schema mapping.        |
| `vmalloc_chunk`            | uint64  | The largest free vmalloc chunk.                                                     | No direct schema mapping.        |
| `hugepages.total`          | uint64  | `HugePages_Total`, the configured pages.                                            | No direct schema mapping.        |
| `hugepages.free`           | uint64  | `HugePages_Free`.                                                                   | No direct schema mapping.        |
| `hugepages.reserved`       | uint64  | `HugePages_Rsvd`.                                                                   | No direct schema mapping.        |
| `hugepages.surplus`        | uint64  | `HugePages_Surp`.                                                                   | No direct schema mapping.        |
| `hugepages.size`           | uint64  | `Hugepagesize` in bytes.                                                            | No direct schema mapping.        |
| `hugepages.anon_hugepages` | uint64  | `AnonHugePages`.                                                                    | No direct schema mapping.        |
| `hugepages.hugetlb`        | uint64  | `Hugetlb`, the total memory in hugepages (Linux).                                   | No direct schema mapping.        |
| `direct_map.map_4k`        | uint64  | The physical memory that 4k pages map (Linux).                                      | No direct schema mapping.        |
| `direct_map.map_2m`        | uint64  | The physical memory that 2M pages map (Linux).                                      | No direct schema mapping.        |
| `direct_map.map_1g`        | uint64  | The physical memory that 1G pages map (Linux).                                      | No direct schema mapping.        |
| `swap.total`               | uint64  | The total swap. Omitted when there is no swap.                                      | No direct schema mapping.        |
| `swap.used`                | uint64  | The used swap.                                                                      | No direct schema mapping.        |
| `swap.free`                | uint64  | The free swap.                                                                      | No direct schema mapping.        |
| `swap.used_percent`        | float64 | The percent of swap that is used.                                                   | No direct schema mapping.        |
| `swap.cached`              | uint64  | `SwapCached` from `/proc/meminfo` (Linux).                                          | No direct schema mapping.        |

## Platform support

| Platform | Supported                                               |
| -------- | ------------------------------------------------------- |
| Linux    | ✅ (`/proc/meminfo` through gopsutil, 27+ fields)       |
| macOS    | ✅ (`host_statistics64` and `vm_stat` through executor) |

## Example output

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

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.memory      # enable (default)
gohai collect --no-collector.memory   # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/memory.rb` reads `/proc/meminfo` line by line and reports each
field. gohai uses gopsutil, which reads the same `/proc/meminfo` file and gives
the same fields as typed Go values. The two report the same fields: total, free,
available, buffers, cached, active, inactive, swap, hugepages, directmap, and
all detail fields of `/proc/meminfo`.

On Linux:

1. gopsutil `mem.VirtualMemory` reads `/proc/meminfo` and gives 27+ fields in
   its `VirtualMemoryStat` struct (Total, Available, Used, Free, Active,
   Inactive, Buffers, Cached, Dirty, WriteBack, WriteBackTmp, Shared, Mapped,
   Slab, Sreclaimable, Sunreclaim, PageTables, SwapCached, CommitLimit,
   CommittedAS, VmallocTotal, VmallocUsed, VmallocChunk, HugePagesTotal,
   HugePagesFree, HugePagesRsvd, HugePagesSurp, HugePageSize, AnonHugePages).
   The collector reports each relevant field. It uses the library first and does
   not parse `/proc/meminfo` separately for these fields.
2. gopsutil `mem.SwapMemory` gives the `swap.*` totals. `swap.cached` comes from
   the `SwapCached` field of the virtual memory stat.
3. The collector sets `hugepages.*` only when one or more hugepage fields are
   there. On kernels without hugepages support, the collector skips it with no
   error.
4. The collector also reads `/proc/meminfo` through the injected `avfs.VFS` to
   get the fields that the cross-platform struct of gopsutil does not give:
   `Active(anon)`, `Active(file)`, `Inactive(anon)`, `Inactive(file)`,
   `Unevictable`, `KernelStack`, `Percpu`, `KReclaimable`, `AnonPages`, `Shmem`,
   `Hugetlb`, and `DirectMap4k`, `DirectMap2M` and `DirectMap1G`. The kernel
   reports all values in kB. The collector multiplies them by 1024 to report
   bytes. This removes the difference from the Ohai method. It does not add a
   separate parser for the 27 fields that gopsutil already reads. It extends the
   library, as the library-first principle of CONTRIBUTING.md tells.

On macOS:

1. gopsutil `mem.VirtualMemory` calls `host_statistics64` for Total, Available,
   Used, Free, Active, Inactive and Wired.
2. The collector also runs `vm_stat` through the shared `internal/executor`
   runner. It parses the `page size of N bytes` header on the first line. The
   page size is 4096 on Intel and 16384 on Apple Silicon. Each line after it has
   the form `Pages <label>: <int>.`. The collector multiplies the pages by the
   page size to get a byte count. It sets these fields:
   - `speculative`, from `Pages speculative`. This cache class is specific to
     macOS. Linux has no equivalent.
   - `compressed`, from `Pages stored in compressor`. Apple Silicon uses the
     compressor much.
3. If `vm_stat` is missing or fails, the extra fields stay at zero. The totals
   from gopsutil stay correct.

## Backing library

- [`github.com/shirou/gopsutil/v4/mem`](https://github.com/shirou/gopsutil),
  BSD-3. The primary source on the two platforms.
- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `vm_stat` on macOS. Tests mock it with
  `go.uber.org/mock`.
