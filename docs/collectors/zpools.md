# Zpools

> **Status:** Implemented ✅

## Description

The collector reports the ZFS pool status. It runs
`zpool list -H -o name,size,alloc,free,health,altroot`. Linux and macOS can both
have ZFS through [OpenZFS]. Thus the two variants try to run `zpool`. If the
collector does not find `zpool`, or if `zpool` exits with a non-zero code, the
collector returns an empty pool list and no error.

`zpool` reports the `-` sentinel for fields that have no value. An example is
`altroot` when no alternative root is set. The collector changes `-` to an empty
string. Thus consumers do not need a special case for it.

The method is the same as Ohai's `zpools` plugin. The same `zpool list -H`
command runs on Linux, macOS and the BSDs.

Consumers use this to:

- Monitor the ZFS pool health (`ONLINE`, `DEGRADED`, `FAULTED`) of a fleet.
- Monitor the allocated storage and the free space.
- Find pools that have an alternative root, for example during a rescue.

## Collected fields

| Field             | Type     | Description                                                                                                                         | Schema mapping   |
| ----------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| `pools`           | `array`  | The list of ZFS pools from `zpool list`.                                                                                            | gohai convention |
| `pools[].name`    | `string` | The pool name.                                                                                                                      | gohai convention |
| `pools[].size`    | `string` | The total pool size, as `zpool` reports it (for example `"1.82T"`). Empty if `zpool` reports `-`.                                   | gohai convention |
| `pools[].alloc`   | `string` | The allocated storage space (for example `"672G"`). Empty if `zpool` reports `-`.                                                   | gohai convention |
| `pools[].free`    | `string` | The storage space that is not allocated (for example `"1.17T"`). Empty if `zpool` reports `-`.                                      | gohai convention |
| `pools[].health`  | `string` | The pool health status: `"ONLINE"`, `"DEGRADED"`, `"FAULTED"`, `"OFFLINE"`, `"REMOVED"` or `"UNAVAIL"`. Empty if there is no value. | gohai convention |
| `pools[].altroot` | `string` | The alternative root directory of the pool. Empty if no alternative root is set (`zpool` reports `-`).                              | gohai convention |

## Platform support

| Platform | Supported                    |
| -------- | ---------------------------- |
| Linux    | ✅ (if OpenZFS is installed) |
| macOS    | ✅ (if OpenZFS is installed) |

If `zpool` is missing, the two variants return an empty list, not an error. If
`zpool` is there and fails, they return the error.

## Example output

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

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("zpools"))
facts, _ := g.Collect(context.Background())
// facts.Zpools.Pools contains the list
```

## Enable/disable

```bash
gohai collect --collector.zpools      # enable
gohai collect --no-collector.zpools   # disable
```

## Dependencies

None.

## Data sources

On Linux and macOS, the method is the same:

1. The collector runs `zpool list -H -o name,size,alloc,free,health,altroot`
   through the injected `source.Runner`. `-H` removes the header line. `-o`
   selects the six fields that gohai needs, with tabs between them.
2. The command can fail. The exit code can be non-zero, `zpool` can be missing
   from `PATH`, or the executor can be `nil`. In these conditions, the collector
   returns an empty pool list and no error. This shows that ZFS is not installed
   on the host.
3. The collector parses each output line and splits it on tabs. It skips lines
   that do not have exactly six fields. This prevents errors from an output
   format that is not expected.
4. For each field, the collector changes the `zpool` sentinel `"-"` to an empty
   string.

Ohai's `zpools` plugin uses the same `zpool list -H` command. gohai uses a
smaller set of fields: `-o name,size,alloc,free,health,altroot`, not Ohai's
`name,size,alloc,free,cap,dedup,health,version`. gohai does not read `cap`,
`dedup` and `version`. They come from the core fields, or fleet inventory seldom
uses them.

## Backing library

- `source.Runner` to run `zpool list`. Production uses `source.Commands()`.
  Tests use a gomock mock.
- The Go standard library `bufio` and `strings` to parse the output.

[openzfs]: https://openzfs.org/
