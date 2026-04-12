# Timezone

> **Status:** Implemented ✅

## Description

Reports the host's active timezone — the IANA zone name (e.g.
`America/Los_Angeles`), its current short abbreviation (`PDT`, `PST`, `UTC`),
and the current offset from UTC in seconds. The zone name is resolved from the
`/etc/localtime` symlink on both Linux and macOS; the abbreviation and offset
come from Go's `time.Now().Zone()` and reflect whatever DST rule is active at
collection time.

Consumers use this to:

- Correlate logs and metrics across a fleet that spans regions (knowing whether
  a host emits timestamps in local time vs UTC).
- Detect misconfigured hosts — e.g. a production box still set to the builder's
  home timezone, or a container inheriting `UTC` when the policy says it
  shouldn't.
- Drive cron/schedule computations that need to know where the host thinks it
  is.

The offset is a snapshot at collect time, not a yearly constant — a host on
`America/Los_Angeles` will report `-25200` in July (PDT) and `-28800` in January
(PST).

## Collected Fields

| Field    | Type     | Description                                             |
| -------- | -------- | ------------------------------------------------------- |
| `name`   | `string` | IANA timezone name (e.g. `America/Los_Angeles`, `UTC`). |
| `abbrev` | `string` | Current short abbreviation (e.g. `PDT`, `PST`, `UTC`).  |
| `offset` | `int`    | Current offset from UTC in seconds (positive for east). |

## Platform Support

| Platform | Source                                         | Supported |
| -------- | ---------------------------------------------- | --------- |
| Linux    | `/etc/localtime` symlink + `time.Now().Zone()` | ✅        |
| macOS    | `/etc/localtime` symlink + `time.Now().Zone()` | ✅        |
| Other    | —                                              | `nil`     |

If `/etc/localtime` is not a symlink into the zoneinfo database, `name` falls
back to the abbreviation reported by `time`.

## Example Output

### Los Angeles host in summer

```json
{
  "timezone": {
    "name": "America/Los_Angeles",
    "abbrev": "PDT",
    "offset": -25200
  }
}
```

### Container set to UTC

```json
{
  "timezone": {
    "name": "UTC",
    "abbrev": "UTC",
    "offset": 0
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("timezone"))
facts, _ := g.Collect(context.Background())

tz := facts.Timezone
fmt.Println(tz.Name, tz.Abbrev, tz.Offset)
```

## Enable/Disable

```bash
gohai --collector.timezone      # enable (default)
gohai --no-collector.timezone   # disable
```

## Dependencies

None — Tier 1 core collector with no upstream collector dependencies.

## Backing library

- Go stdlib (`os.Readlink`, `time`) — no third-party dependency.
