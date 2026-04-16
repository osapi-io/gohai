# SCSI

> **Status:** Implemented ✅

## Description

Enumerates SCSI devices visible to the host by running `lsscsi` and parsing its
output. Mirrors Ohai's `linux/scsi.rb` methodology: one entry per SCSI address,
with transport, type, make+model name, firmware revision, and the backing device
node. macOS has no equivalent interface and returns an empty Info.

Consumers use this to inventory attached disks / CD-ROMs / tape drives by SCSI
address when `lsscsi`'s particular view is the needed abstraction (e.g. matching
hardware in a server's drive bays to logical device nodes).

## Collected Fields

| Field     | Type                | Description                    | Schema mapping            |
| --------- | ------------------- | ------------------------------ | ------------------------- |
| `devices` | `map[string]Device` | SCSI devices keyed by address. | No direct schema mapping. |

### Device

| Field       | Type     | Description                                                |
| ----------- | -------- | ---------------------------------------------------------- |
| `scsi_addr` | `string` | SCSI address (e.g. `0:0:0:0`, `5:0:0:0`).                  |
| `type`      | `string` | Device type (`disk`, `cd/dvd`, `tape`, etc.).              |
| `transport` | `string` | Transport / controller (`ATA`, `LSI`, `iSCSI`, etc.).      |
| `name`      | `string` | Vendor + model string (middle tokens of the `lsscsi` row). |
| `revision`  | `string` | Firmware revision (second-to-last token).                  |
| `device`    | `string` | Backing device node (`/dev/sda`, `/dev/sr0`, etc.).        |

## Platform Support

| Platform | Supported                                        |
| -------- | ------------------------------------------------ |
| Linux    | ✅ (requires `lsscsi` on PATH)                   |
| macOS    | ❌ (no `lsscsi` on Darwin; returns empty `Info`) |

## Example Output

```json
{
  "scsi": {
    "devices": {
      "0:0:0:0": {
        "scsi_addr": "0:0:0:0",
        "type": "disk",
        "transport": "ATA",
        "name": "ST500DM002-1BD14",
        "revision": "KC48",
        "device": "/dev/sda"
      },
      "6:0:0:0": {
        "scsi_addr": "6:0:0:0",
        "type": "cd/dvd",
        "transport": "NECVMWar",
        "name": "VMware IDE CDR10",
        "revision": "1.00",
        "device": "/dev/sr0"
      }
    }
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("scsi"))
facts, _ := g.Collect(context.Background())

for addr, d := range facts.SCSI.Devices {
    fmt.Printf("%s  %s  %s\n", addr, d.Name, d.Device)
}
```

## Enable/Disable

```bash
gohai --collector.scsi      # enable (opt-in)
gohai --no-collector.scsi   # disable (default)
gohai --category=hardware   # pulls this + all hardware collectors
```

## Dependencies

None.

## Data Sources

On Linux:

1. Run `lsscsi` through the shared `internal/executor` runner. `lsscsi` parses
   `/proc/scsi/scsi` and the sysfs SCSI tree and formats one device per row.
2. Each non-empty row is whitespace-split into tokens. Following Ohai's
   `linux/scsi.rb` algorithm:
   - Token 0 (bracketed, e.g. `[0:0:0:0]`) → `scsi_addr` with brackets stripped.
   - Token 1 → `type`.
   - Token 2 → `transport`.
   - Last token → `device`.
   - Second-to-last token → `revision`.
   - Tokens 3 through `n-3` (inclusive) joined with single spaces → `name`
     (vendor + model).
3. Rows with fewer than 5 tokens (the minimum needed to carry address, type,
   transport, revision, device) are skipped. Rows with an empty bracketed
   address are also skipped.
4. Missing binary, exec error, or empty output yield an empty `devices` map with
   no error — hosts without util-linux's `lsscsi` return cleanly rather than
   surfacing a missing-binary error.

On macOS the collector returns an empty `Info{Devices: {}}`. macOS has no
`lsscsi` equivalent and surfaces storage topology through IOKit, not a
SCSI-abstraction layer. Ohai's scsi plugin is Linux-only and we match that
scope.

Mirrors Ohai's `linux/scsi.rb` exactly — same command, same parsing algorithm,
same field names, same per-address keying. Ohai tolerates a missing `lsscsi` by
simply skipping (`optional true` plugin); our collector matches that by
returning empty rather than erroring.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `lsscsi`. Tests mock it with `go.uber.org/mock`.
