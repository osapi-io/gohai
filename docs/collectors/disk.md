# Disk

> **Status:** Implemented ✅

## Description

Per-device disk I/O counters (read/write counts, bytes, and times). Wraps
[gopsutil's `disk.IOCounters`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/disk).

Note: this collector reports I/O statistics only. For physical device metadata
(model, vendor, size) a future `block_device` collector will wrap `ghw/block`.

## Collected Fields

Top-level: `devices []Device`.

Per-device:

| Field         | Type   | Description                  |
| ------------- | ------ | ---------------------------- |
| `name`        | string | Device name (e.g., `sda`)    |
| `read_count`  | uint64 | Number of reads              |
| `write_count` | uint64 | Number of writes             |
| `read_bytes`  | uint64 | Bytes read                   |
| `write_bytes` | uint64 | Bytes written                |
| `read_time`   | uint64 | Time spent reading (ms)      |
| `write_time`  | uint64 | Time spent writing (ms)      |
| `io_time`     | uint64 | Time with in-flight I/O (ms) |

## Platform Support

| Platform | Source                                   | Supported |
| -------- | ---------------------------------------- | --------- |
| Linux    | `gopsutil/v4/disk.IOCountersWithContext` | ✅        |
| macOS    | `gopsutil/v4/disk.IOCountersWithContext` | ✅        |
| Other    | Returns `nil`                            | —         |

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
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("disk"))
facts, _ := g.Collect(context.Background())

for _, d := range facts.Disk.Devices {
    fmt.Printf("%s: %d reads, %d writes\n", d.Name, d.ReadCount, d.WriteCount)
}
```

## Enable/Disable

```bash
gohai --collector.disk      # enable (default)
gohai --no-collector.disk   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil) —
BSD-3.
