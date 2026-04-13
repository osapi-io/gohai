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

| Platform | What we read                                                                                                     | Ohai plugin ([`time.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/time.rb)) | Alignment                                                                                                                                                                                                                                                                   |
| -------- | ---------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `/etc/localtime` symlink (IANA name) → `/etc/timezone` file fallback → `time.Now().Zone()` for abbrev+offset     | Ohai reads only `Time.now.getlocal.zone` (abbreviation only); no IANA name.                | **Richer than Ohai.** We capture IANA name and offset on top of Ohai's abbreviation. Debian/container fallback covers hosts where `/etc/localtime` is a copied file rather than a symlink — Ohai doesn't have this problem because it doesn't try to resolve the IANA name. |
| macOS    | `/etc/localtime` symlink stripped of `/var/db/timezone/zoneinfo/` prefix → `time.Now().Zone()` for abbrev+offset | Same as Linux — only the abbreviation.                                                     | **Richer than Ohai.**                                                                                                                                                                                                                                                       |

**Known gaps:** On macOS we don't ship a `/etc/timezone` fallback because Apple
doesn't set that file. If `/etc/localtime` isn't a symlink on a mac (rare — seen
in some CI sandboxes), `name` stays empty while `abbrev`/`offset` still
populate.

## Backing library

- Go stdlib (`os.Readlink`, `os.ReadFile`, `time`) — no third-party dependency.
