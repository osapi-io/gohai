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

| Platform | What we read                                               | Ohai plugin                                                                                                                                                                    | Alignment                                                                                                                                                                                    |
| -------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `disk.IOCountersWithContext` (`/proc/diskstats`). | No Ohai equivalent — Ohai's `linux/block_device.rb` reports sysfs device metadata (model, rotational, block sizes) which is the future gohai `block_device` collector's scope. | **Gohai extension.** Ohai doesn't track I/O counters; we do so consumers can reason about storage workload without running `iostat`. Node_exporter's `diskstats` collector is the reference. |
| macOS    | gopsutil `disk.IOCountersWithContext` (IOKit).             | —                                                                                                                                                                              | Same gohai-native scope.                                                                                                                                                                     |

**Known gaps:** None at this collector's scope. Physical device metadata is out
of scope; see the planned `block_device` collector.

## Backing library

- [`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil) —
  BSD-3.
