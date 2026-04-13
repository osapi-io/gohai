# Filesystem

> **Status:** Implemented ✅

## Description

Enumerates filesystems known to the host, both mounted and unmounted. Mounted
filesystems carry capacity, usage, and inode counters; every filesystem (mounted
or not) carries UUID and label where available so consumers can correlate mounts
to disks, volumes, or cloud snapshots.

Consumers use this to:

- Monitor disk-full conditions per mount (critical for `/var/log`,
  `/var/lib/docker`, etc.).
- Detect unexpected mount points (e.g. NFS shares that shouldn't be present).
- Correlate a mount to its disk UUID — the standard join key across storage,
  asset-inventory, and cloud-snapshot tooling.
- Surface block devices that carry a filesystem but aren't mounted — LUKS
  containers, inactive LVs, btrfs device members.

Per-mount usage failures (stale NFS, permission denied) skip the usage/inode
fields but keep the mount in the output.

## Collected Fields

Top level: `mounts: []Mount`, plus (Linux only) `unmounted: []Filesystem`.

| Field per mount | Type     | Description                                 | Schema mapping                            |
| --------------- | -------- | ------------------------------------------- | ----------------------------------------- |
| `device`        | string   | Block-device path (`/dev/sda1`).            | OCSF `device.path`.                       |
| `mountpoint`    | string   | Mount path (`/`, `/boot`).                  | Nearest: OCSF `file.path` (event-scoped). |
| `fstype`        | string   | Filesystem type (`ext4`, `xfs`, `apfs`).    | No direct OCSF.                           |
| `opts`          | []string | Mount options (`rw`, `relatime`, `nosuid`). | No direct OCSF.                           |
| `total`         | uint64   | Total bytes.                                | No direct OCSF.                           |
| `used`          | uint64   | Used bytes.                                 | No direct OCSF.                           |
| `free`          | uint64   | Free bytes.                                 | No direct OCSF.                           |
| `used_percent`  | float64  | Percent used (0–100).                       | No direct OCSF.                           |
| `inodes_total`  | uint64   | Total inodes.                               | No direct OCSF.                           |
| `inodes_used`   | uint64   | Used inodes.                                | No direct OCSF.                           |
| `inodes_free`   | uint64   | Free inodes.                                | No direct OCSF.                           |
| `uuid`          | string   | Filesystem UUID from `lsblk`.               | OCSF `device.uid`.                        |
| `label`         | string   | Filesystem label from `lsblk`.              | No direct OCSF.                           |
| `part_uuid`     | string   | GPT partition UUID from `lsblk`.            | No direct OCSF.                           |
| `part_label`    | string   | GPT partition label from `lsblk`.           | No direct OCSF.                           |

Top-level `unmounted[]` (Linux only): block devices that `lsblk` reports with a
non-empty filesystem type and empty mountpoint. Each entry carries `device`,
`fstype`, `uuid`, `label`, `part_uuid`, `part_label` — capacity/usage are
omitted because `statfs` requires a mount.

Field-name choices follow node_exporter's `filesystem` collector (`mountpoint`
over Ohai's `mount`, `fstype` over Ohai's `fs_type`).

## Platform Support

| Platform | Supported                                       |
| -------- | ----------------------------------------------- |
| Linux    | ✅ (gopsutil + `lsblk` enrichment when on PATH) |
| macOS    | ✅ (`getfsstat` syscall via gopsutil)           |

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
        "inodes_free": 6424216,
        "uuid": "a1b2c3d4-0000-0000-0000-000000000001",
        "label": "root",
        "part_uuid": "11111111-2222-3333-4444-555555555555",
        "part_label": ""
      }
    ],
    "unmounted": [
      {
        "device": "/dev/sdb1",
        "fstype": "crypto_LUKS",
        "uuid": "deadbeef-0000-0000-0000-000000000002",
        "label": "data"
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
for _, u := range facts.Filesystem.Unmounted {
    fmt.Printf("unmounted %s (%s) uuid=%s\n", u.Device, u.Fstype, u.UUID)
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

On Linux:

1. gopsutil `disk.Partitions(true)` enumerates the mount table from
   `/proc/mounts`; gopsutil `disk.Usage(mountpoint)` per row provides
   capacity/used/available and inode counters.
2. When `lsblk` is on PATH we run
   `lsblk -J -o NAME,UUID,LABEL,FSTYPE,MOUNTPOINT,PARTUUID,PARTLABEL` via the
   shared `internal/executor` runner and merge the result into the mount table
   by device path, populating `uuid`, `label`, `part_uuid`, `part_label`.
3. Block devices that `lsblk` reports with a non-empty `FSTYPE` and empty
   `MOUNTPOINT` populate `unmounted[]` so LUKS, inactive LVs, and btrfs devices
   are still visible.
4. When `lsblk` is missing (minimal containers, Alpine without util-linux) or
   its output is malformed, we skip the enrichment silently; capacity and inode
   data remain from gopsutil.

On macOS we use gopsutil's mount enumeration backed by `getfsstat(2)`.

## Backing library

- [`github.com/shirou/gopsutil/v4/disk`](https://github.com/shirou/gopsutil) —
  BSD-3. Primary source for mounts, capacity, inodes.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `lsblk` on Linux. Tests mock it with
  `go.uber.org/mock`.
