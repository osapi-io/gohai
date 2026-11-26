# Filesystem

> **Status:** Implemented ✅

## Description

The collector lists the filesystems that the host knows, mounted and not
mounted. A mounted filesystem has capacity, usage and inode counters. Each
filesystem, mounted or not, has a UUID and a label where they are available.
Consumers use them to relate mounts to disks, volumes or cloud snapshots.

Consumers use this data to:

- Monitor each mount for a full disk. This is critical for `/var/log`,
  `/var/lib/docker` and similar mounts.
- Find unexpected mount points, for example NFS shares that must not be there.
- Relate a mount to its disk UUID. The UUID is the usual join key for storage,
  asset-inventory and cloud-snapshot tools.
- Find block devices that have a filesystem but are not mounted: LUKS
  containers, inactive LVs and btrfs device members.

If the usage data of one mount fails (stale NFS, permission denied), the
collector leaves out the usage and inode fields. The mount stays in the output.

## Collected fields

The top level has `mounts: []Mount`. On Linux only, it also has
`unmounted: []Filesystem`.

| Field per mount                     | Type     | Description                                                                                                                                                                                                                                | Schema mapping                               |
| ----------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------- |
| `device`                            | string   | The block-device path (`/dev/sda1`).                                                                                                                                                                                                       | OCSF `device.path`.                          |
| `mountpoint`                        | string   | The mount path (`/`, `/boot`).                                                                                                                                                                                                             | Nearest: OCSF `file.path` (for events only). |
| `type`                              | string   | The filesystem type (`ext4`, `xfs`, `apfs`).                                                                                                                                                                                               | No direct schema mapping.                    |
| `opts`                              | []string | The mount options (`rw`, `relatime`, `nosuid`).                                                                                                                                                                                            | No direct schema mapping.                    |
| `total`                             | uint64   | The total bytes.                                                                                                                                                                                                                           | No direct schema mapping.                    |
| `used`                              | uint64   | The used bytes.                                                                                                                                                                                                                            | No direct schema mapping.                    |
| `free`                              | uint64   | The free bytes.                                                                                                                                                                                                                            | No direct schema mapping.                    |
| `used_percent`                      | float64  | The percentage used, 0 to 100.                                                                                                                                                                                                             | No direct schema mapping.                    |
| `inodes_total`                      | uint64   | The total inodes.                                                                                                                                                                                                                          | No direct schema mapping.                    |
| `inodes_used`                       | uint64   | The used inodes.                                                                                                                                                                                                                           | No direct schema mapping.                    |
| `inodes_free`                       | uint64   | The free inodes.                                                                                                                                                                                                                           | No direct schema mapping.                    |
| `inodes_used_percent`               | float64  | The percentage of inodes used, 0 to 100.                                                                                                                                                                                                   | No direct schema mapping.                    |
| `uuid`                              | string   | The filesystem UUID, from `lsblk`.                                                                                                                                                                                                         | OCSF `device.uid`.                           |
| `label`                             | string   | The filesystem label, from `lsblk`.                                                                                                                                                                                                        | No direct schema mapping.                    |
| `part_uuid`                         | string   | The GPT partition UUID, from `lsblk`.                                                                                                                                                                                                      | No direct schema mapping.                    |
| `part_label`                        | string   | The GPT partition label, from `lsblk`.                                                                                                                                                                                                     | No direct schema mapping.                    |
| `btrfs.raid`                        | string   | The Btrfs RAID profile (`single`, `dup`, `raid0`, `raid1`, `raid1c3`, `raid1c4`, `raid5`, `raid6`, `raid10`). Linux only. The field has a value only when the mount is btrfs and `/sys/fs/btrfs/<uuid>/allocation/<bg>/<profile>/` exists. | No direct schema mapping.                    |
| `btrfs.allocation.<bg>.total_bytes` | uint64   | The total bytes allocated to a block-group type (`data`, `metadata`, `system`).                                                                                                                                                            | No direct schema mapping.                    |
| `btrfs.allocation.<bg>.bytes_used`  | uint64   | The bytes in use now in that block-group type.                                                                                                                                                                                             | No direct schema mapping.                    |

The top-level `unmounted[]` (Linux only) has the block devices that `lsblk`
reports with a filesystem type and no mountpoint. Each entry has `device`,
`type`, `uuid`, `label`, `part_uuid` and `part_label`. It has no capacity or
usage, because `statfs` needs a mount.

The top-level `zfs_datasets[]` (Linux only, when `zfs` is on the PATH) has each
ZFS dataset that the ZFS kernel module knows: filesystems, volumes, snapshots
and bookmarks. `zfs get -p -H all` reports them. This is the same data as Ohai's
`zfs_properties`, `zfs_parents` and `zfs_zpool` keys.

| Field per dataset     | Type                | Description                                                                                                                                                    |
| --------------------- | ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`                | string              | The full dataset path (`tank`, `tank/home`, `tank/home/john`).                                                                                                 |
| `mountpoint`          | string              | The mount path, when the `mountpoint` property is absolute. Empty when the property is `none`, `legacy` or `-`, and for snapshots.                             |
| `is_pool`             | bool                | True when this dataset IS the zpool root (it has no proper ancestors). The same as Ohai's `zfs_zpool`.                                                         |
| `parents`             | []string            | The paths of the proper ancestor datasets, the shallowest first. For `tank/home/john`, the value is `["tank", "tank/home"]`. The same as Ohai's `zfs_parents`. |
| `properties`          | map[string]Property | All properties from `zfs get all`. The keys are property names. Each value has the property value and the source, as the rows below show.                      |
| `properties.*.value`  | string              | The property value, exactly as zfs gives it.                                                                                                                   |
| `properties.*.source` | string              | `"-"` (read-only), `"default"`, `"local"`, `"inherited from <ancestor>"`, `"received"`.                                                                        |

The field names are the same as in node_exporter's `filesystem` collector. gohai
uses `mountpoint`, not Ohai's `mount`, and `type`, not Ohai's `fs_type`.

## Platform support

| Platform | Supported                                               |
| -------- | ------------------------------------------------------- |
| Linux    | ✅ (`/proc/mounts`, `df`, and `lsblk` when it is there) |
| macOS    | ✅ (`mount` and `df`)                                   |

## Example output

```json
{
  "filesystem": {
    "mounts": [
      {
        "device": "/dev/sda1",
        "mountpoint": "/",
        "type": "ext4",
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
        "type": "crypto_LUKS",
        "uuid": "deadbeef-0000-0000-0000-000000000002",
        "label": "data"
      }
    ]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("filesystem"))
facts, _ := g.Collect(context.Background())
for _, m := range facts.Filesystem.Mounts {
    fmt.Printf("%s on %s: %.1f%% used\n", m.Device, m.Mountpoint, m.UsedPercent)
}
for _, u := range facts.Filesystem.Unmounted {
    fmt.Printf("unmounted %s (%s) uuid=%s\n", u.Device, u.Type, u.UUID)
}
```

## Enable/disable

```bash
gohai collect --collector.filesystem      # enable (default)
gohai collect --no-collector.filesystem   # disable
```

## Dependencies

None.

## Data sources

Ohai's `filesystem.rb` combines the output of `mount`, `df` and `lsblk`. For
each device, it shows the mount points, the capacity and the filesystem types.
gohai reads `/proc/mounts` and runs `df`. The data is the same. gohai also adds
btrfs allocation data, which it reads through VFS. It also adds ZFS dataset
properties from the `zfs` command.

On Linux:

1. `/proc/mounts`, read through the filesystem of the collector, is the mount
   table. The collector changes the octal escapes of the file, such as `\040`
   for a space, back to characters. `df -akP` gives the capacity, used and
   available bytes of each mount. `df -aiP` gives the inode counts
   (`inodes_total`, `inodes_used`, `inodes_available`). When `total > 0`, the
   collector calculates `inodes_used_percent` as `used / total * 100`. Thus
   gohai also has Ohai's `filesystem[mount][:inodes_percent_used]` field.
2. When `lsblk` is on the PATH, the collector runs
   `lsblk -J -o NAME,UUID,LABEL,FSTYPE,MOUNTPOINT,PARTUUID,PARTLABEL` through
   the shared `source.Runner`. It merges the result into the mount table by
   device path, and fills `uuid`, `label`, `part_uuid` and `part_label`.
3. The block devices that `lsblk` reports with an `FSTYPE` and no `MOUNTPOINT`
   go into `unmounted[]`. Thus LUKS containers, inactive LVs and btrfs devices
   are still visible.
4. `lsblk` can be missing, for example in minimal containers or in Alpine
   without util-linux. Its output can also be not valid. In these cases, the
   collector skips the `lsblk` data and reports no error. The capacity and inode
   data from `df` stay.
5. **Btrfs allocation data**. For each mount with `type == "btrfs"` and a UUID,
   the collector reads `/sys/fs/btrfs/<uuid>/allocation/` through the injected
   `avfs.VFS`. It fills the `btrfs` sub-record of the mount: the byte counters
   (`total_bytes`, `bytes_used`) of each block group (`data`, `metadata`,
   `system`), and the set of RAID profile flags (`single`, `raid0`, `raid1`,
   `raid10`, `raid5`, `raid6`, `dup`). If the sysfs paths are missing (older
   kernels, containers without a mounted /sys), the collector skips this data
   and reports no error.
6. **ZFS datasets**. When `zfs` is on the PATH, the collector runs
   `zfs get -p -H all` through the shared `source.Runner`. It parses the
   tab-separated rows (dataset, property, value, source) into `ZFSDatasets`,
   with the dataset name as the key. `-p` gives byte values as numbers. `-H`
   removes the header. On a host without ZFS (the binary is missing, or the run
   fails), the collector returns zero datasets and no error. This does not
   depend on `lsblk`.

On macOS, `mount` gives the mount table: the device, the mount point, the type
and the options. `df -ki` gives the capacity and the inode counts of each mount
in one run. The btrfs and ZFS data are on Linux only.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `df` on the two platforms, `mount` on macOS, and
  `lsblk` and `zfs` on Linux. Tests mock it with `go.uber.org/mock`.
