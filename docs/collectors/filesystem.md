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

## Example Output

```json
{
  "filesystem": {
    "mounts": [
      {
        "device": "/dev/sda1",
        "mountpoint": "/",
        "fstype": "ext4",
        "opts": ["rw", "relatime"],
        "total": 107374182400,
        "used": 53687091200,
        "free": 53687091200,
        "used_percent": 50.0,
        "inodes_total": 6553600,
        "inodes_used": 129384,
        "inodes_free": 6424216
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

g, _ := gohai.New(gohai.WithCollectors("filesystem"))
facts, _ := g.Collect(context.Background())

for _, m := range facts.Filesystem.Mounts {
    fmt.Printf("%s on %s: %.1f%% used\n", m.Device, m.Mountpoint, m.UsedPercent)
}
```

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
