# Zpools

> **Status:** Implemented ✅

## Description

Reports ZFS pool status by running
`zpool list -H -o name,size,alloc,free,health,altroot`. Both Linux and macOS can
host ZFS via [OpenZFS][], so both variants attempt to run `zpool`. When `zpool`
is not found or returns a non-zero exit, an empty pool list is returned without
error.

The `-` sentinel that `zpool` reports for unavailable fields (e.g. `altroot`
when no alternate root is configured) is normalized to an empty string so
consumers don't need to special-case it.

The collector mirrors Ohai's `zpools` plugin methodology: the same
`zpool list -H` invocation is used on Linux, macOS, and BSDs.

Consumers use this to:

- Monitor ZFS pool health (`ONLINE`, `DEGRADED`, `FAULTED`) across a fleet.
- Track storage allocation and free space.
- Detect pools configured with alternate roots (e.g. during rescue operations).

## Collected Fields

| Field             | Type     | Description                                                                                                                  | Schema mapping   |
| ----------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `pools`           | `array`  | List of ZFS pools from `zpool list`.                                                                                         | gohai convention |
| `pools[].name`    | `string` | Pool name.                                                                                                                   | gohai convention |
| `pools[].size`    | `string` | Total pool size as reported by `zpool` (e.g. `"1.82T"`). Empty when `zpool` reports `-`.                                     | gohai convention |
| `pools[].alloc`   | `string` | Allocated storage space (e.g. `"672G"`). Empty when `zpool` reports `-`.                                                     | gohai convention |
| `pools[].free`    | `string` | Unallocated storage space (e.g. `"1.17T"`). Empty when `zpool` reports `-`.                                                  | gohai convention |
| `pools[].health`  | `string` | Pool health status: `"ONLINE"`, `"DEGRADED"`, `"FAULTED"`, `"OFFLINE"`, `"REMOVED"`, or `"UNAVAIL"`. Empty when unavailable. | gohai convention |
| `pools[].altroot` | `string` | Alternate root directory for the pool. Empty when none is configured (`zpool` reports `-`).                                  | gohai convention |

## Platform Support

| Platform | Supported                      |
| -------- | ------------------------------ |
| Linux    | ✅ (when OpenZFS is installed) |
| macOS    | ✅ (when OpenZFS is installed) |

Both variants return an empty list — not an error — when `zpool` is absent.

## Example Output

```json
{
  "zpools": {
    "pools": [
      {
        "name": "tank",
        "size": "1.82T",
        "alloc": "672G",
        "free": "1.17T",
        "health": "ONLINE"
      },
      {
        "name": "backup",
        "size": "931G",
        "alloc": "450G",
        "free": "481G",
        "health": "DEGRADED",
        "altroot": "/mnt/rescue"
      }
    ]
  }
}
```

### No ZFS installed

```json
{
  "zpools": {
    "pools": []
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("zpools"))
facts, _ := g.Collect(context.Background())
// facts.Zpools.Pools contains the list
```

## Enable/Disable

```bash
gohai --collector.zpools      # enable
gohai --no-collector.zpools   # disable
```

## Dependencies

None.

## Data Sources

On Linux and macOS (identical procedure):

1. Run `zpool list -H -o name,size,alloc,free,health,altroot` via the injected
   `executor.Executor`. `-H` suppresses the header line; `-o` selects exactly
   the six fields we want, tab-separated.
2. If the command fails (exit code non-zero, `zpool` not found in `PATH`, or the
   executor is `nil`), return an empty pool list without error — ZFS is not
   installed on this host.
3. Parse each output line, splitting on tabs. Lines that do not produce exactly
   six fields are skipped (defensive against unexpected output formats).
4. For each field, convert the `zpool` sentinel `"-"` to an empty string.

Ohai's `zpools` plugin uses the same `zpool list -H` invocation. We use a
slightly narrower field set (`-o name,size,alloc,free,health,altroot` versus
Ohai's `name,size,alloc,free,cap,dedup,health,version`) because `cap`, `dedup`,
and `version` are derivable from the core fields or rarely queried in fleet
inventory.

## Backing library

- `internal/executor` (`executor.New()` in production, gomock mock in tests) for
  the `zpool list` invocation.
- Go stdlib `bufio` and `strings` for output parsing.

[OpenZFS]: https://openzfs.org/
