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
