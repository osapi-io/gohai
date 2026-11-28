# SCSI

> **Status:** Implemented ✅

## Description

The collector lists the SCSI devices that the host can see. It runs `lsscsi` and
parses the output. The method is the same as Ohai's `linux/scsi.rb`. The
collector reports one entry for each SCSI address. Each entry has the vendor,
the type, the make and model name, the firmware revision and the backing device
node. macOS has no equivalent interface, so the collector gives no fact there.

Consumers use this data to make a list of the attached disks, CD-ROM drives and
tape drives by SCSI address. Use it when you need the `lsscsi` view of the
hardware. For example, use it to find which logical device node is the drive in
a given drive bay of a server.

## Collected fields

| Field     | Type                | Description                         | Schema mapping            |
| --------- | ------------------- | ----------------------------------- | ------------------------- |
| `devices` | `map[string]Device` | The SCSI devices, keyed by address. | No direct schema mapping. |

### Device

| Field       | Type     | Description                                                                                                                                         |
| ----------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `scsi_addr` | `string` | The SCSI address, for example `0:0:0:0` or `5:0:0:0`.                                                                                               |
| `type`      | `string` | The device type, for example `disk`, `cd/dvd` or `tape`.                                                                                            |
| `vendor`    | `string` | The vendor column of `lsscsi`, for example `ATA`, `LSI` or `NECVMWar`. Ohai calls it `transport`, but `lsscsi` prints the transport only with `-t`. |
| `name`      | `string` | The vendor and model, from the middle tokens of the `lsscsi` row.                                                                                   |
| `revision`  | `string` | The firmware revision, from the second-to-last token.                                                                                               |
| `device`    | `string` | The backing device node, for example `/dev/sda` or `/dev/sr0`.                                                                                      |

## Platform support

| Platform | Supported                                                |
| -------- | -------------------------------------------------------- |
| Linux    | ✅ (`lsscsi` must be on the PATH)                        |
| macOS    | ❌ (Darwin has no `lsscsi`. The collector gives no fact) |

## Example output

```json
{
  "scsi": {
    "devices": {
      "0:0:0:0": {
        "scsi_addr": "0:0:0:0",
        "type": "disk",
        "vendor": "ATA",
        "name": "ST500DM002-1BD14",
        "revision": "KC48",
        "device": "/dev/sda"
      },
      "6:0:0:0": {
        "scsi_addr": "6:0:0:0",
        "type": "cd/dvd",
        "vendor": "NECVMWar",
        "name": "VMware IDE CDR10",
        "revision": "1.00",
        "device": "/dev/sr0"
      }
    }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("scsi"))
facts, _ := g.Collect(context.Background())

for addr, d := range facts.SCSI.Devices {
    fmt.Printf("%s  %s  %s\n", addr, d.Name, d.Device)
}
```

## Enable/disable

```bash
gohai collect --collector.scsi      # enable (opt-in)
gohai collect --no-collector.scsi   # disable (default)
gohai collect --category=hardware   # pulls this + all hardware collectors
```

## Dependencies

None.

## Data sources

On Linux:

1. The collector runs `lsscsi` through the shared `source.Runner`. `lsscsi`
   parses `/proc/scsi/scsi` and the sysfs SCSI tree. It writes one device on
   each row.
2. The collector splits each row that is not empty into tokens at the
   whitespace. It uses the algorithm of Ohai's `linux/scsi.rb`:
   - Token 0 is in brackets, for example `[0:0:0:0]`. It becomes `scsi_addr`
     without the brackets.
   - Token 1 → `type`.
   - Token 2 → `vendor`.
   - Last token → `device`.
   - Second-to-last token → `revision`.
   - Tokens 3 to `n-3`, with the two ends included, joined with single spaces →
     `name` (vendor and model).
3. The collector skips rows with fewer than 5 tokens. A row must have 5 tokens
   to hold the address, the type, the vendor, the revision and the device. The
   collector also skips rows with an empty address in the brackets.
4. If the binary is missing, the command fails, or the output is empty, the
   collector returns an empty `devices` map and no error. Thus a host without
   the util-linux `lsscsi` gives a clean result, not a missing-binary error.

On macOS, the collector gives no fact, as each collector does on a platform that
it does not support. macOS has no equivalent of `lsscsi`. It shows the storage
topology through IOKit, not through a SCSI layer. Ohai's scsi plugin is for
Linux only, and gohai has the same scope.

The collector is the same as Ohai's `linux/scsi.rb`. It uses the same command,
the same parse algorithm, the same field names and the same key for each
address. If `lsscsi` is missing, Ohai skips it, because the plugin has
`optional true`. The gohai collector does the same: it returns an empty result,
not an error. If `lsscsi` is there and fails, the collector returns the error.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `lsscsi`. Tests mock it with `go.uber.org/mock`.
