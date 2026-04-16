# Timezone

> **Status:** Implemented ✅

## Description

Reports the host's active timezone: IANA zone name (e.g. `America/Los_Angeles`),
current short abbreviation (`PDT`, `PST`, `UTC`), and current offset from UTC in
seconds. The zone name is resolved from `/etc/localtime`; abbreviation and
offset come from Go's `time.Now().Zone()` and reflect the DST rule active at
collection time.

Consumers use this to:

- Correlate logs and metrics across a fleet spanning regions.
- Detect misconfigured hosts — e.g. a production box still on the builder's home
  timezone.
- Drive cron/schedule computations that need host-local time.

The offset is a snapshot at collect time — a host on `America/Los_Angeles`
reports `-25200` in July (PDT) and `-28800` in January (PST).

## Collected Fields

| Field    | Type   | Description                                             | Schema mapping                                                                                                                                           |
| -------- | ------ | ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name`   | string | IANA timezone name.                                     | No direct schema mapping. OCSF's `device.location.tz_name` (within Location object) is the nearest analog but scoped to event location, not host config. |
| `abbrev` | string | Current short abbreviation (`PDT`, `PST`, `UTC`).       | No direct schema mapping — ambiguous (e.g. `IST` means India / Israel / Ireland) so OCSF deliberately prefers offsets and IANA names.                    |
| `offset` | int    | Current offset from UTC in seconds (positive for east). | Closest: OCSF's timestamps use ISO-8601 with offset, but there's no host-config field.                                                                   |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |
| Other    | —         |

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

None.

## Data Sources

On Linux:

1. Resolve the IANA `name` by calling `Readlink("/etc/localtime")` through the
   injected `avfs.VFS` and stripping the `/usr/share/zoneinfo/` prefix. This
   gives values like `America/Los_Angeles`.
2. When `/etc/localtime` isn't a symlink (Debian-style hosts where the timezone
   data is copied rather than symlinked, some container images), fall back to
   reading `/etc/timezone` through the VFS and trimming whitespace.
3. If both sources fail, `name` stays empty — `abbrev` and `offset` still
   populate from the Go runtime.
4. `abbrev` and `offset` come from `time.Now().Zone()` — the runtime's cached
   local-time rules. The offset is a snapshot at collect time, so a host on
   `America/Los_Angeles` reports `-25200` in July (PDT) and `-28800` in January
   (PST).

On macOS:

1. Resolve the IANA `name` by calling `Readlink("/etc/localtime")` and stripping
   macOS's `/var/db/timezone/zoneinfo/` prefix instead of Linux's
   `/usr/share/zoneinfo/`. No `/etc/timezone` fallback — Apple doesn't ship that
   file, so if the symlink is missing (rare, some CI sandboxes) `name` stays
   empty while `abbrev`/`offset` still populate.
2. `abbrev` and `offset` come from `time.Now().Zone()` — same as Linux.

Superset of Ohai's `time.rb`, which only reports `Time.now.getlocal.zone` (the
abbreviation). We additionally surface the IANA name (`America/Los_Angeles` vs.
just `PDT`) and the numeric offset.

## Backing library

- Go stdlib (`os.Readlink`, `os.ReadFile`, `time`) — no third-party dependency.
