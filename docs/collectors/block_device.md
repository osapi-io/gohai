# Block Device

> **Status:** Implemented ✅

## Description

Reports sysfs attributes for each block device found under `/sys/block` on
Linux. One entry is emitted per device name (e.g. `sda`, `nvme0n1`, `vda`,
`loop0`). Attributes come from three sysfs subtrees per device:

- `/sys/block/<dev>/` — top-level attributes: `size`, `removable`.
- `/sys/block/<dev>/queue/` — I/O scheduler and geometry: `rotational`,
  `physical_block_size`, `logical_block_size`.
- `/sys/block/<dev>/device/` — SCSI/NVMe device attributes: `model`, `vendor`,
  `rev`, `state`, `timeout`, `queue_depth`, `firmware_rev`.

Missing sysfs files yield empty strings for their fields — not all devices
expose all attributes (loop devices, virtual block devices, NVMe vs SCSI).

Consumers use this to:

- Distinguish spinning disks from SSDs/NVMe via `rotational`.
- Read hardware metadata (`model`, `vendor`) without requiring root access.
- Detect removable media.

Ohai's `block_device` plugin reads the same sysfs paths on the same
device-relative layout; we mirror its attribute set exactly.

## Collected Fields

| Field                           | Type     | Description                                                                         | Schema mapping                                                                                              |
| ------------------------------- | -------- | ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `devices`                       | `array`  | List of block devices discovered under `/sys/block`.                                | gohai convention — OCSF `device_hw_info` covers disks at a high level but not per-block-device sysfs detail |
| `devices[].name`                | `string` | Kernel device name (e.g. `sda`, `nvme0n1`, `vda`).                                  | gohai convention                                                                                            |
| `devices[].size`                | `string` | Device capacity in 512-byte sectors from `/sys/block/<dev>/size`.                   | gohai convention (`_bytes` suffix omitted because the unit is sectors, not bytes)                           |
| `devices[].removable`           | `string` | `"0"` (fixed) or `"1"` (removable) from `/sys/block/<dev>/removable`.               | gohai convention                                                                                            |
| `devices[].rotational`          | `string` | `"0"` (SSD/NVMe) or `"1"` (spinning disk) from `/sys/block/<dev>/queue/rotational`. | gohai convention                                                                                            |
| `devices[].physical_block_size` | `string` | Physical sector size in bytes from `/sys/block/<dev>/queue/physical_block_size`.    | gohai convention                                                                                            |
| `devices[].logical_block_size`  | `string` | Logical sector size in bytes from `/sys/block/<dev>/queue/logical_block_size`.      | gohai convention                                                                                            |
| `devices[].model`               | `string` | Device model string from `/sys/block/<dev>/device/model`.                           | gohai convention                                                                                            |
| `devices[].vendor`              | `string` | Device vendor string from `/sys/block/<dev>/device/vendor`.                         | gohai convention                                                                                            |
| `devices[].rev`                 | `string` | Firmware revision from `/sys/block/<dev>/device/rev`.                               | gohai convention                                                                                            |
| `devices[].state`               | `string` | Device state from `/sys/block/<dev>/device/state` (e.g. `"running"`).               | gohai convention                                                                                            |
| `devices[].timeout`             | `string` | SCSI command timeout in seconds from `/sys/block/<dev>/device/timeout`.             | gohai convention                                                                                            |
| `devices[].queue_depth`         | `string` | SCSI queue depth from `/sys/block/<dev>/device/queue_depth`.                        | gohai convention                                                                                            |
| `devices[].firmware_rev`        | `string` | Alternate firmware revision from `/sys/block/<dev>/device/firmware_rev`.            | gohai convention                                                                                            |

## Platform Support

| Platform | Supported             |
| -------- | --------------------- |
| Linux    | ✅                    |
| macOS    | nil (no `/sys/block`) |

macOS does not expose a `/sys/block` hierarchy. The Darwin variant returns
`nil`, indicating the data is not available on this platform.

## Example Output

```json
{
  "block_device": {
    "devices": [
      {
        "name": "sda",
        "size": "976773168",
        "removable": "0",
        "rotational": "1",
        "physical_block_size": "512",
        "logical_block_size": "512",
        "model": "WDC WD5000AAKX",
        "vendor": "ATA",
        "rev": "1H15",
        "state": "running",
        "timeout": "30",
        "queue_depth": "32"
      },
      {
        "name": "nvme0n1",
        "size": "1000215216",
        "removable": "0",
        "rotational": "0",
        "physical_block_size": "512",
        "logical_block_size": "512",
        "model": "Samsung SSD 970 EVO",
        "firmware_rev": "2B2QEXE7"
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

g, _ := gohai.New(gohai.WithCollectors("block_device"))
facts, _ := g.Collect(context.Background())
// facts.BlockDevice.Devices contains the list
```

## Enable/Disable

```bash
gohai --collector.block_device      # enable
gohai --no-collector.block_device   # disable
```

## Dependencies

None.

## Data Sources

Ohai's `linux/block_device.rb` reads the same `/sys/block/` tree — top-level
`size` and `removable`, `queue/` for `rotational`, `physical_block_size`,
`logical_block_size`, and `device/` for `model`, `rev`, `state`, `timeout`,
`vendor`, `queue_depth`, `firmware_rev`. gohai reads the identical sysfs paths
in the same order. No methodology deviation from Ohai.

On Linux:

1. Read the `/sys/block` directory via the injected `avfs.VFS`. If the directory
   is absent (containers that don't bind-mount `/sys`, minimal images), return
   an empty device list without error.
2. For each directory entry (skipping regular files — only device subdirectories
   are relevant), read sysfs attributes using the following layout, matching
   Ohai's `block_device` plugin read order:
   - `/sys/block/<dev>/size` and `/sys/block/<dev>/removable` — top-level.
   - `/sys/block/<dev>/queue/rotational`, `physical_block_size`,
     `logical_block_size` — queue sub-tree.
   - `/sys/block/<dev>/device/model`, `vendor`, `rev`, `state`, `timeout`,
     `queue_depth`, `firmware_rev` — device sub-tree.
3. Missing individual files are soft-missed to an empty string — not all devices
   populate all paths (loop devices have no `device/` sub-tree; NVMe devices
   expose `firmware_rev` but not `rev`).

On macOS: returns `nil`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for all sysfs reads.
- Go stdlib `path/filepath` for path construction.
