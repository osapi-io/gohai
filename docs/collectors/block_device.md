# Block device

> **Status:** Implemented ✅

## Description

The collector reports the sysfs attributes of each block device under
`/sys/block` on Linux. It reports one entry for each device name, for example
`sda`, `nvme0n1`, `vda` or `loop0`. The attributes come from three sysfs
subtrees of each device:

- `/sys/block/<dev>/`. The top-level attributes: `size`, `removable`.
- `/sys/block/<dev>/queue/`. The I/O scheduler and the geometry: `rotational`,
  `physical_block_size`, `logical_block_size`.
- `/sys/block/<dev>/device/`. The SCSI or NVMe device attributes: `model`,
  `vendor`, `rev`, `state`, `timeout`, `queue_depth`, `firmware_rev`.

If a sysfs file is missing, its field is an empty string. Not all devices have
all attributes. Loop devices, virtual block devices, and NVMe and SCSI devices
each have a different set.

Consumers use this data to:

- Tell spinning disks from SSD and NVMe devices with `rotational`.
- Read the hardware metadata (`model`, `vendor`) without root access.
- Find removable media.

Ohai's `block_device` plugin reads the same sysfs paths in the same layout for
each device. gohai reports the same set of attributes as Ohai.

## Collected fields

| Field                           | Type     | Description                                                                             | Schema mapping                                                                                                |
| ------------------------------- | -------- | --------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| `devices`                       | `array`  | The list of block devices under `/sys/block`.                                           | gohai convention. OCSF `device_hw_info` gives a summary of disks, but not the sysfs data of each block device |
| `devices[].name`                | `string` | The kernel device name, for example `sda`, `nvme0n1` or `vda`.                          | gohai convention                                                                                              |
| `devices[].size`                | `string` | The device capacity in 512-byte sectors, from `/sys/block/<dev>/size`.                  | gohai convention. The name has no `_bytes` suffix because the unit is sectors, not bytes                      |
| `devices[].removable`           | `string` | `"0"` (fixed) or `"1"` (removable), from `/sys/block/<dev>/removable`.                  | gohai convention                                                                                              |
| `devices[].rotational`          | `string` | `"0"` (SSD or NVMe) or `"1"` (spinning disk), from `/sys/block/<dev>/queue/rotational`. | gohai convention                                                                                              |
| `devices[].physical_block_size` | `string` | The physical sector size in bytes, from `/sys/block/<dev>/queue/physical_block_size`.   | gohai convention                                                                                              |
| `devices[].logical_block_size`  | `string` | The logical sector size in bytes, from `/sys/block/<dev>/queue/logical_block_size`.     | gohai convention                                                                                              |
| `devices[].model`               | `string` | The device model, from `/sys/block/<dev>/device/model`.                                 | gohai convention                                                                                              |
| `devices[].vendor`              | `string` | The device vendor, from `/sys/block/<dev>/device/vendor`.                               | gohai convention                                                                                              |
| `devices[].rev`                 | `string` | The firmware revision, from `/sys/block/<dev>/device/rev`.                              | gohai convention                                                                                              |
| `devices[].state`               | `string` | The device state, from `/sys/block/<dev>/device/state`, for example `"running"`.        | gohai convention                                                                                              |
| `devices[].timeout`             | `string` | The SCSI command timeout in seconds, from `/sys/block/<dev>/device/timeout`.            | gohai convention                                                                                              |
| `devices[].queue_depth`         | `string` | The SCSI queue depth, from `/sys/block/<dev>/device/queue_depth`.                       | gohai convention                                                                                              |
| `devices[].firmware_rev`        | `string` | An alternative firmware revision, from `/sys/block/<dev>/device/firmware_rev`.          | gohai convention                                                                                              |

## Platform support

| Platform | Supported             |
| -------- | --------------------- |
| Linux    | ✅                    |
| macOS    | nil (no `/sys/block`) |

macOS has no `/sys/block` tree. The Darwin variant returns `nil`. This shows
that the data is not available on this platform.

## Example output

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

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("block_device"))
facts, _ := g.Collect(context.Background())
// facts.BlockDevice.Devices contains the list
```

## Enable/disable

```bash
gohai collect --collector.block_device      # enable
gohai collect --no-collector.block_device   # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/block_device.rb` reads the same `/sys/block/` tree. It reads
`size` and `removable` at the top level. It reads `rotational`,
`physical_block_size` and `logical_block_size` from `queue/`. It reads `model`,
`rev`, `state`, `timeout`, `vendor`, `queue_depth` and `firmware_rev` from
`device/`. gohai reads the same sysfs paths in the same order. The method is the
same as Ohai's.

On Linux:

1. The collector reads the `/sys/block` directory through the injected
   `avfs.VFS`. If the directory is not there, the collector returns an empty
   device list and no error. Containers that do not bind-mount `/sys` and
   minimal images have no `/sys/block`. A `/sys/block` that is there and cannot
   be read is an error.
2. For each device subdirectory, the collector reads the sysfs attributes in
   this layout. It skips regular files. The read order is the same as in Ohai's
   `block_device` plugin:
   - `/sys/block/<dev>/size` and `/sys/block/<dev>/removable`, at the top level.
   - `/sys/block/<dev>/queue/rotational`, `physical_block_size` and
     `logical_block_size`, in the queue subtree.
   - `/sys/block/<dev>/device/model`, `vendor`, `rev`, `state`, `timeout`,
     `queue_depth` and `firmware_rev`, in the device subtree.
3. If one file is missing, its field is an empty string. Not all devices have
   all paths. Loop devices have no `device/` subtree. NVMe devices have
   `firmware_rev` but not `rev`. A file that is there and cannot be read is an
   error, so that an empty field always means that the device has no such
   attribute.

On macOS, the collector returns `nil`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for all sysfs reads.
  Production uses `osfs`. Tests use `memfs`.
- The Go standard library `path/filepath` to make the paths.
