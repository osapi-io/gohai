# Mdadm

> **Status:** Implemented ✅

## Description

The collector reports the Linux software RAID arrays that the MD (Multiple
Device) driver manages. It reads `/proc/mdstat` to find all arrays. Then, for
each array, it runs `mdadm --detail /dev/<device>` to get the RAID level, the
health state, the UUID and the device counts.

If `mdadm` is not installed, the collector returns the arrays with only the
devices from `/proc/mdstat`. If `/proc/mdstat` is missing, the collector returns
an empty array list and no error. If `mdadm --detail` fails for an array, such
as an inactive array, that array keeps its `/proc/mdstat` devices and reports
the failure in `detail_error`. The other arrays keep their details. The
collector returns the arrays and the joined error of each failed detail, so its
status is `partial`. A cancelled context is an error with no fact.

The method of the collector is the same as Ohai's `mdadm` plugin. It reads
`/proc/mdstat` first. Then it calls `mdadm --detail` for each array to get the
detail fields.

Consumers use this data to:

- Audit the RAID health of a fleet (`state: "degraded"` or `"failed"`).
- Monitor the count of spare disks, to find arrays that run without redundancy.
- Make sure that the RAID level and the UUID are correct for change management.

## Collected fields

| Field                    | Type       | Description                                                                                                                                   | Schema mapping   |
| ------------------------ | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `arrays`                 | `array`    | The list of MD RAID arrays in `/proc/mdstat`.                                                                                                 | gohai convention |
| `arrays[].device`        | `string`   | The kernel device name, for example `"md0"` or `"md127"`.                                                                                     | gohai convention |
| `arrays[].level`         | `string`   | The RAID level, from `mdadm --detail`, for example `"raid1"` or `"raid5"`. Empty when it is not available.                                    | gohai convention |
| `arrays[].state`         | `string`   | The array state as `mdadm --detail` prints it, a list separated by `, `, for example `"clean"`, `"active, resyncing"` or `"clean, degraded"`. | gohai convention |
| `arrays[].uuid`          | `string`   | The array UUID, from `mdadm --detail`, for example `"a5d3:1234:dead:beef"`.                                                                   | gohai convention |
| `arrays[].active_disks`  | `int`      | The count of active member disks, from `mdadm --detail`.                                                                                      | gohai convention |
| `arrays[].total_disks`   | `int`      | The total count of configured member slots, from `mdadm --detail`.                                                                            | gohai convention |
| `arrays[].spare_disks`   | `int`      | The count of spare disks, from `mdadm --detail`.                                                                                              | gohai convention |
| `arrays[].failed_disks`  | `int`      | The count of failed member disks, from the `Failed Devices` line of `mdadm --detail`.                                                         | gohai convention |
| `arrays[].working_disks` | `int`      | The count of working member disks, from the `Working Devices` line of `mdadm --detail`.                                                       | gohai convention |
| `arrays[].members`       | `[]string` | The device names of the active members, from `/proc/mdstat`, for example `["sda1","sdb1"]`.                                                   | gohai convention |
| `arrays[].spares`        | `[]string` | The device names of the spare members, from `/proc/mdstat` (devices with the `(S)` tag).                                                      | gohai convention |
| `arrays[].faulty`        | `[]string` | The device names that the kernel has failed out of the array, from `/proc/mdstat` (devices with the `(F)` tag).                               | gohai convention |
| `arrays[].journal`       | `[]string` | The device names of the write journal, from `/proc/mdstat` (devices with the `(J)` tag). A journal device is not a data member.               | gohai convention |
| `arrays[].detail_error`  | `string`   | The failure of `mdadm --detail` for this array, for example on an inactive array. Empty when the detail step ran or `mdadm` is not installed. | gohai convention |

## Platform support

| Platform | Supported      |
| -------- | -------------- |
| Linux    | ✅             |
| macOS    | nil (no mdadm) |

macOS has no MD RAID stack. The Darwin variant gives no fact, with the status
`unsupported`.

## Example output

```json
{
  "mdadm": {
    "arrays": [
      {
        "device": "md0",
        "level": "raid1",
        "state": "clean",
        "uuid": "a5d3:1234:dead:beef",
        "active_disks": 2,
        "total_disks": 2,
        "spare_disks": 0,
        "failed_disks": 0,
        "working_disks": 2,
        "members": ["sda1", "sdb1"],
        "spares": [],
        "faulty": [],
        "journal": []
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

g, _ := gohai.New(gohai.WithCollectors("mdadm"))
facts, _ := g.Collect(context.Background())
// facts.Mdadm.Arrays contains the list
```

## Enable/disable

```bash
gohai collect --collector.mdadm      # enable
gohai collect --no-collector.mdadm   # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/mdadm.rb` reads `/proc/mdstat` to find the arrays. It parses the
lists of member and spare disks. gohai uses the same method for `/proc/mdstat`.
gohai also runs `mdadm --detail`, if it is available, to get the UUID, the state
and the disk counts. Ohai does the same (`mdadm --detail /dev/<device>`). If
`mdadm` is not installed, gohai still reports the arrays with the data from
`/proc/mdstat` only.

On Linux:

1. The collector reads `/proc/mdstat` through the injected `avfs.VFS`. If the
   file is missing, the collector returns an empty array list and no error. The
   file is missing if the kernel has no MD support, or on a host in a container.
2. The collector parses `/proc/mdstat` line by line. An array line agrees with
   `<name> : active|inactive ...` and has a device name that starts with `md`.
   On each array line, a regex finds the member tokens. The form `<dev>[N]` is
   an active member. The form `<dev>[N](S)` is a spare, `<dev>[N](F)` is a
   faulty device, and `<dev>[N](J)` is a journal device. Only the active form is
   in `members`, because a faulty device and a journal device hold no data or
   parity for the array.
3. The collector sorts the arrays by device name, so the output is always in the
   same order. For each array, it runs `mdadm --detail /dev/<device>` through
   the injected `source.Runner`. It parses these lines of the output:
   - `Raid Level : <level>`. The RAID level.
   - `State : <state>`. The array state.
   - `UUID : <uuid>`. The array UUID.
   - `Active Devices : <n>`, `Total Devices : <n>`, `Spare Devices : <n>`. The
     device counts.
4. If `mdadm` is not installed, the collector skips the detail step. It still
   returns each array with the devices from `/proc/mdstat`. This is the same as
   Ohai, which always reports the data from `/proc/mdstat`, also when `mdadm` is
   missing. If `mdadm --detail` fails for an array, the array reports the
   failure in `detail_error`, so a failed read is not reported as an array with
   no detail, and the other arrays keep their details. `mdadm` exits non-zero
   for `--detail` on an inactive array and for a user that can't open the
   device.

On macOS, the collector gives no fact, with the status `unsupported`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for the `/proc/mdstat`
  read. Production uses `osfs`. Tests use `memfs`.
- `source.Runner` to run `mdadm --detail`. Production uses `source.Commands()`.
  Tests use a gomock mock.
- The Go standard library `bufio`, `regexp` and `strconv` to parse the data.
