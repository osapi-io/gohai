# Timezone

> **Status:** Implemented ✅

## Description

The collector reports the active timezone of the host. It reports the IANA zone
name (for example `America/Los_Angeles`), the current short abbreviation (`PDT`,
`PST`, `UTC`) and the current offset from UTC in seconds. The collector gets the
zone name from `/etc/localtime`. The abbreviation and the offset come from the
zone data in `/etc/localtime` itself, applied to the current time. They show the
DST rule that is active when the collector runs, and a change of zone shows at
the next collection.

Consumers use this to:

- Compare logs and metrics across a fleet that is in many regions.
- Find hosts with an incorrect configuration. An example is a production host
  that still uses the home timezone of the person who built it.
- Calculate cron and schedule times that need the local time of the host.

The offset is the value at the time of collection. A host on
`America/Los_Angeles` reports `-25200` in July (PDT) and `-28800` in January
(PST).

## Collected fields

| Field    | Type   | Description                                                      | Schema mapping                                                                                                                                                                       |
| -------- | ------ | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `name`   | string | The IANA timezone name.                                          | No direct schema mapping. OCSF's `device.location.tz_name` (in the Location object) is the nearest equivalent. But it is about the location of an event, not the host configuration. |
| `abbrev` | string | The current short abbreviation (`PDT`, `PST`, `UTC`).            | No direct schema mapping. Abbreviations can have more than one meaning. For example, `IST` means India, Israel or Ireland. Thus OCSF uses offsets and IANA names.                    |
| `offset` | int    | The current offset from UTC in seconds. It is positive for east. | Nearest: OCSF's timestamps use ISO-8601 with an offset, but OCSF has no host configuration field.                                                                                    |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |
| Other    | None      |

## Example output

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

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.timezone      # enable (default)
gohai collect --no-collector.timezone   # disable
```

## Dependencies

None.

## Data sources

Ohai's `timezone.rb` reports only `Time.now.getlocal.zone`. This is the short
abbreviation, for example "EST". gohai reports more than Ohai. It adds the IANA
name, for example "America/New_York", from the target of the `/etc/localtime`
symlink. It also adds the numeric UTC offset in seconds. Thus consumers get a
timezone value that a person can read and a value that a program can parse.

On Linux:

1. The collector gets the IANA `name`. It calls `Readlink("/etc/localtime")`
   through the injected `avfs.VFS`, and takes the part after the last
   `/zoneinfo/` of the target. This gives values like `America/Los_Angeles` from
   `/usr/share/zoneinfo/`, and also from NixOS's `/etc/zoneinfo/`. A relative
   link, such as `../usr/share/zoneinfo/Etc/UTC` on Ubuntu, is relative to
   `/etc`, so it also gives `Etc/UTC`. A bare target that is an IANA name, such
   as `UTC`, is that name. Any other target, such as a path with no zoneinfo
   directory, gives no name: `name` is never a path.
2. Sometimes `/etc/localtime` is not a symlink. On Debian-style hosts, RHEL,
   Alpine and some container images, the timezone data is a copy. In this
   condition, the collector reads `/etc/timezone`, then the `ZONE=` line of
   `/etc/sysconfig/clock` (RHEL and SUSE), through the VFS. It takes the first
   one that is an IANA name.
3. If no source gives a name, `name` stays empty.
4. `abbrev` and `offset` come from the zone data in `/etc/localtime`, read
   through the VFS and applied to the current time. The offset is the value at
   the time of collection. Thus a host on `America/Los_Angeles` reports `-25200`
   in July (PDT) and `-28800` in January (PST). When the file cannot be read or
   parsed, they come from the process's local zone.

On macOS:

1. The collector gets the IANA `name` from `Readlink("/etc/localtime")`, the
   same way: the part after the last `/zoneinfo/`, which also handles Apple's
   versioned `/var/db/timezone/tz/<version>/zoneinfo/`. There is no second
   source like `/etc/timezone`, because Apple does not supply that file. If the
   symlink is missing, `name` stays empty, but `abbrev` and `offset` still have
   values. A missing symlink is rare. Some CI sandboxes have it.
2. `abbrev` and `offset` come from the zone data in `/etc/localtime`, the same
   as on Linux.

## Backing library

- `avfs.VFS` (`Readlink`, `ReadFile`) for the files, and the Go standard
  library's `time` (`LoadLocationFromTZData`) for the zone data.
