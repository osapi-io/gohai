# Filesystem

> **Status:** Implemented ✅

## Description

Enumerates mounted filesystems with capacity, usage, and inode statistics. Each
mount carries device path, mount point, filesystem type, mount options, and
per-mount space/inode counters.

Consumers use this to:

- Monitor disk-full conditions per mount (critical for `/var/log`,
  `/var/lib/docker`, etc.).
- Detect unexpected mount points (e.g. NFS shares that shouldn't be present).
- Feed inventory tooling that tracks filesystem layout across a fleet.

Per-mount usage failures (stale NFS, permission denied) skip the usage/inode
fields but keep the mount in the output — the device, mountpoint, fstype, and
opts still populate.

## Collected Fields

Top level: `mounts: []Mount`.

| Field per mount | Type     | Description                                      | Schema mapping |
| --------------- | -------- | ------------------------------------------------ | ------------------------------------------------- |
| `device`        | string   | Block device path (`/dev/sda1`, `/dev/disk3s1`). | No direct OCSF.                                   |
| `mountpoint`    | string   | Mount point path (`/`, `/boot`).                 | Nearest: `file.path` (event-scoped, not perfect). |
| `fstype`        | string   | Filesystem type (`ext4`, `xfs`, `apfs`).         | No direct OCSF.                                   |
| `opts`          | []string | Mount options (`rw`, `relatime`, `nosuid`).      | No direct OCSF.                                   |
| `total`         | uint64   | Total bytes.                                     | No direct OCSF.                                   |
| `used`          | uint64   | Used bytes.                                      | No direct OCSF.                                   |
| `free`          | uint64   | Free bytes.                                      | No direct OCSF.                                   |
| `used_percent`  | float64  | Percent used (0–100).                            | No direct OCSF.                                   |
| `inodes_total`  | uint64   | Total inodes.                                    | No direct OCSF.                                   |
| `inodes_used`   | uint64   | Used inodes.                                     | No direct OCSF.                                   |
| `inodes_free`   | uint64   | Free inodes.                                     | No direct OCSF.                                   |

Field-name choices follow node_exporter's `filesystem` collector (`mountpoint`
over Ohai's `mount`, `fstype` over Ohai's `fs_type`) — industry standard in the
Prometheus / Go ecosystem.

## Platform Support

| Platform | Supported                             |
| -------- | ------------------------------------- |
| Linux    | ✅ (parses `/proc/mounts` + `statfs`) |
| macOS    | ✅ (`getfsstat` syscall via gopsutil) |

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

## Data Sources

| Platform | What we read                                                     | Ohai plugin                                                                                                                                                                                        | Alignment                                                                                                                                                                                                                             |
| -------- | ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `disk.PartitionsWithContext` + `disk.UsageWithContext`. | [`filesystem.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/filesystem.rb) — cross-platform plugin combining `df -P`, `df -iP`, `mount`, `lsblk`, `blkid`, and btrfs sysfs on Linux. | **Same core data (mounts, capacity, inodes).** Ohai additionally surfaces `uuid` and `label` (via `blkid`/`lsblk`) and btrfs allocation detail. Field-name convention differs (Ohai uses `mount`/`fs_type`; we follow node_exporter). |
| macOS    | gopsutil `disk.PartitionsWithContext` + `disk.UsageWithContext`. | Same top-level [`filesystem.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/filesystem.rb) — `df`, `mount`, `diskutil` under the `:darwin` branch.                                    | **Same core data.** Ohai adds `diskutil`-derived UUID/label on darwin.                                                                                                                                                                |

**Known gaps vs. Ohai:**

- No `uuid` / `label` per mount (read `/dev/disk/by-uuid/*` and
  `/dev/disk/by-label/*` symlinks as a follow-up).
- No btrfs-allocation breakdown.
- No unmounted-volume discovery (`blkid` / `lsblk` enumerate unmounted
  filesystems too; gopsutil only lists currently-mounted).

## Backing library

- [`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil) —
  BSD-3.
