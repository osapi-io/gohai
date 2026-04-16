# Disk

> **Status:** Implemented ✅

## Description

Reports per-device disk I/O counters (read/write counts, bytes, and latency).
Equivalent to what `iostat` shows at the block-device layer — useful for
detecting saturated disks, unbalanced I/O across NVMe devices, and trending
storage workload over time.

Note: this collector reports **I/O counters only**. Physical device metadata
(model, vendor, serial, rotational, block sizes) will come from a forthcoming
`block_device` collector that wraps `ghw/block` — tracked separately. Ohai
splits these the same way.

## Collected Fields

| Field per device | Type   | Description                                   | Schema mapping            |
| ---------------- | ------ | --------------------------------------------- | ------------------------- |
| `name`           | string | Device name (e.g. `sda`, `nvme0n1`, `disk0`). | No direct schema mapping. |
| `read_count`     | uint64 | Number of completed reads.                    | No direct schema mapping. |
| `write_count`    | uint64 | Number of completed writes.                   | No direct schema mapping. |
| `read_bytes`     | uint64 | Bytes read.                                   | No direct schema mapping. |
| `write_bytes`    | uint64 | Bytes written.                                | No direct schema mapping. |
| `read_time`      | uint64 | Time spent reading (ms).                      | No direct schema mapping. |
| `write_time`     | uint64 | Time spent writing (ms).                      | No direct schema mapping. |
| `io_time`        | uint64 | Time with in-flight I/O (ms).                 | No direct schema mapping. |

Top level: `devices: []Device`.

## Platform Support

| Platform | Supported                     |
| -------- | ----------------------------- |
| Linux    | ✅ (parses `/proc/diskstats`) |
| macOS    | ✅ (IOKit via gopsutil)       |

## Example Output

```json
{
  "disk": {
    "devices": [
      {
        "name": "sda",
        "read_count": 152341,
        "write_count": 98234,
        "read_bytes": 9823746048,
        "write_bytes": 4123498752,
        "read_time": 482013,
        "write_time": 182947,
        "io_time": 623481
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("disk"))
facts, _ := g.Collect(context.Background())
for _, d := range facts.Disk.Devices {
    fmt.Printf("%s: %d reads / %d writes\n", d.Name, d.ReadCount, d.WriteCount)
}
```

## Enable/Disable

```bash
gohai --collector.disk      # enable (default)
gohai --no-collector.disk   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. gopsutil's `disk.IOCountersWithContext` reads `/proc/diskstats` and returns
   per-device read/write counts, bytes, and time spent. We forward each device
   as a `Device` row without transformation.

On macOS:

1. gopsutil's `disk.IOCountersWithContext` calls IOKit to produce the same
   shape. Device names match what `iostat` reports (e.g. `disk0`).

Physical device metadata (model, vendor, serial, rotational, block sizes) is out
of this collector's scope — tracked in the planned `block_device` collector
backed by `ghw/block`. Ohai splits the two concerns the same way (its
`diskstats`-style data doesn't exist; `linux/block_device.rb` covers sysfs
device metadata only).

## Backing library

- [`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil) —
  BSD-3.
