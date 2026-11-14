# Uptime

> **Status:** Implemented ✅

## Description

The collector reports how long the system has run, in wall-clock seconds since
boot. It also reports the boot timestamp. On Linux only, it reports the total
CPU idle seconds since boot. The idle counter is the sum of all logical CPUs.
Thus an 8-core host with one day of uptime can show up to 8 days of idle time.
This is the usual Linux meaning, and it is the same as `cat /proc/uptime`.

Consumers use this to:

- Find hosts that rebooted recently. A low uptime shows a maintenance window or
  a reboot that was not planned.
- Find hosts that have run for too long without a reboot, for patch compliance.
- Compare incidents with boot events through `boot_time`.

## Collected fields

| Field          | Type   | Description                                                  | Schema mapping                                                                                                |
| -------------- | ------ | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------- |
| `seconds`      | uint64 | The seconds since boot.                                      | No direct schema mapping. OCSF records the times of single events. Host uptime is not a primary OCSF concept. |
| `boot_time`    | uint64 | The Unix timestamp of boot.                                  | OCSF `device.first_seen_time` records the same idea for asset inventory.                                      |
| `human`        | string | The uptime as text for a person, for example `3d 4h 12m 5s`. | N/A (presentation).                                                                                           |
| `idle_seconds` | uint64 | The total CPU idle seconds of all cores (Linux only).        | No direct schema mapping.                                                                                     |
| `idle_human`   | string | The idle time as text for a person (Linux only).             | N/A (presentation).                                                                                           |

## Platform support

| Platform | Supported                                  |
| -------- | ------------------------------------------ |
| Linux    | ✅                                         |
| macOS    | ✅ (uptime and boot only, no idle counter) |
| Other    | None                                       |

## Example output

### Linux with idle

```json
{
  "uptime": {
    "seconds": 259200,
    "boot_time": 1712572800,
    "human": "3d 0h 0m 0s",
    "idle_seconds": 1036800,
    "idle_human": "12d 0h 0m 0s"
  }
}
```

### macOS

```json
{
  "uptime": {
    "seconds": 7200,
    "boot_time": 1712825600,
    "human": "2h 0m 0s"
  }
}
```

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("uptime"))
facts, _ := g.Collect(context.Background())

u := facts.Uptime
fmt.Printf("up %s since %d\n", u.Human, u.BootTime)
```

## Enable/disable

```bash
gohai --collector.uptime      # enable (default)
gohai --no-collector.uptime   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. gopsutil's `host.InfoWithContext` reads field 1 of `/proc/uptime` for
   `seconds`. It calculates `boot_time` as `now - uptime`. gohai reports the two
   values with no change. It makes `human` with `HumanDuration(seconds)`.
2. gohai also reads `/proc/uptime` through the injected `avfs.VFS`. It parses
   field 2 into `idle_seconds`. Field 2 is the total idle time of all CPUs. On
   multi-core systems, it can be more than the wall-clock uptime. The same
   helper makes `idle_human`. If the file is missing, if its content is not
   valid, or if the value is negative, the idle fields stay at zero.

On macOS:

1. gopsutil's `host.InfoWithContext` reads `kern.boottime` through sysctl. It
   calculates `seconds` as `now - boot_epoch`. The mapping to `seconds`,
   `boot_time` and `human` is the same as on Linux. macOS has no total idle
   counter. Thus the output does not have `idle_seconds` and `idle_human`.

The method is the same as Ohai's `uptime.rb`. On Linux, it parses `/proc/uptime`
in the same way. On Darwin, it gets the uptime from `kern.boottime` in the same
way. The `boot_time` field is an addition that Ohai does not have. The text
format is short (`1d 2h 3m 4s`). Ohai uses a long plural form
(`1 day 02 hours 03 minutes 04 seconds`). This is only a choice of presentation.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  the uptime and the boot time. gohai parses `/proc/uptime` itself for the Linux
  idle seconds, because gopsutil does not give the idle field.
