# Filesystem

> **Status:** Implemented ✅

## Description

Enumerates mounted filesystems with capacity, usage, and inode data. Wraps
[gopsutil's `disk.Partitions` + `disk.Usage`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/disk).

## Collected Fields

Top-level: `mounts []Mount`.

Per-mount:

| Field          | Type     | Description                           |
| -------------- | -------- | ------------------------------------- |
| `device`       | string   | Block device path (e.g., `/dev/sda1`) |
| `mountpoint`   | string   | Mount point (e.g., `/`)               |
| `fstype`       | string   | Filesystem type (e.g., `ext4`)        |
| `opts`         | []string | Mount options                         |
| `total`        | uint64   | Total bytes                           |
| `used`         | uint64   | Used bytes                            |
| `free`         | uint64   | Free bytes                            |
| `used_percent` | float64  | Percent used                          |
| `inodes_total` | uint64   | Total inodes                          |
| `inodes_used`  | uint64   | Used inodes                           |
| `inodes_free`  | uint64   | Free inodes                           |

## Platform Support

| Platform | Source                                       | Supported |
| -------- | -------------------------------------------- | --------- |
| Linux    | `gopsutil/v4/disk.Partitions` + `disk.Usage` | ✅        |
| macOS    | `gopsutil/v4/disk.Partitions` + `disk.Usage` | ✅        |
| Other    | Returns `nil`                                | —         |

## Enable/Disable

```bash
gohai --collector.filesystem      # enable (default)
gohai --no-collector.filesystem   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil) —
BSD-3.
