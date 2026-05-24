# Mdadm

> **Status:** Implemented ✅

## Description

Reports Linux software RAID arrays managed by the MD (Multiple Device) driver.
Discovery uses `/proc/mdstat` to enumerate all arrays; each array is then
enriched with `mdadm --detail /dev/<device>` to obtain the RAID level, health
state, UUID, and device counts.

When `mdadm` is not installed, arrays are returned with only the members and
spares parsed from `/proc/mdstat`. A missing `/proc/mdstat` returns an empty
array list without error.

The collector mirrors Ohai's `mdadm` plugin methodology: it reads `/proc/mdstat`
first, then calls `mdadm --detail` per array to extract the detail fields.

Consumers use this to:

- Audit RAID health across a fleet (`state: "degraded"` or `"failed"`).
- Track spare disk counts to detect arrays running without redundancy.
- Verify RAID level and UUID for change management.

## Collected Fields

| Field                   | Type       | Description                                                                           | Schema mapping   |
| ----------------------- | ---------- | ------------------------------------------------------------------------------------- | ---------------- |
| `arrays`                | `array`    | List of MD RAID arrays discovered via `/proc/mdstat`.                                 | gohai convention |
| `arrays[].device`       | `string`   | Kernel device name (e.g. `"md0"`, `"md127"`).                                         | gohai convention |
| `arrays[].level`        | `string`   | RAID level from `mdadm --detail` (e.g. `"raid1"`, `"raid5"`). Empty when unavailable. | gohai convention |
| `arrays[].state`        | `string`   | Array state from `mdadm --detail` (e.g. `"clean"`, `"active"`, `"degraded"`).         | gohai convention |
| `arrays[].uuid`         | `string`   | Array UUID from `mdadm --detail` (e.g. `"a5d3:1234:dead:beef"`).                      | gohai convention |
| `arrays[].active_disks` | `int`      | Count of active member disks from `mdadm --detail`.                                   | gohai convention |
| `arrays[].total_disks`  | `int`      | Total configured member slots from `mdadm --detail`.                                  | gohai convention |
| `arrays[].spare_disks`  | `int`      | Count of spare disks from `mdadm --detail`.                                           | gohai convention |
| `arrays[].members`      | `[]string` | Active member device names from `/proc/mdstat` (e.g. `["sda1","sdb1"]`).              | gohai convention |
| `arrays[].spares`       | `[]string` | Spare member device names from `/proc/mdstat` (devices tagged `(S)`).                 | gohai convention |

## Platform Support

| Platform | Supported      |
| -------- | -------------- |
| Linux    | ✅             |
| macOS    | nil (no mdadm) |

macOS has no MD RAID stack. The Darwin variant returns `nil`.

## Example Output

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
        "members": ["sda1", "sdb1"],
        "spares": []
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

g, _ := gohai.New(gohai.WithCollectors("mdadm"))
facts, _ := g.Collect(context.Background())
// facts.Mdadm.Arrays contains the list
```

## Enable/Disable

```bash
gohai --collector.mdadm      # enable
gohai --no-collector.mdadm   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. Read `/proc/mdstat` via the injected `avfs.VFS`. If the file is absent (no MD
   kernel support, containerized host), return an empty array list without
   error.
2. Parse `/proc/mdstat` line by line. Lines matching
   `<name> : active|inactive ...` with a device name starting with `md` are
   recognized as array lines. For each such line, member tokens of the form
   `<dev>[N]` (active) and `<dev>[N](S)` (spare) are extracted via regex.
   Journal devices (`(J)`) are recognized and silently skipped — they are an
   implementation detail of the MD layer, not a data or parity member.
3. For each discovered array (sorted by device name for deterministic output),
   run `mdadm --detail /dev/<device>` via the injected `executor.Executor`.
   Parse the output for:
   - `Raid Level : <level>` — RAID level string.
   - `State : <state>` — array state.
   - `UUID : <uuid>` — array UUID.
   - `Active Devices : <n>`, `Total Devices : <n>`, `Spare Devices : <n>` —
     device counts.
4. If `mdadm` is not installed or returns a non-zero exit, the enrichment step
   is silently skipped for that array. The array is still returned with members
   and spares from `/proc/mdstat`, matching Ohai's behavior of always emitting
   what `/proc/mdstat` tells us even when `mdadm` is missing.

On macOS: returns `nil`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for the `/proc/mdstat` read.
- `internal/executor` (`executor.New()` in production, gomock mock in tests) for
  the `mdadm --detail` invocation.
- Go stdlib `bufio`, `regexp`, and `strconv` for parsing.
