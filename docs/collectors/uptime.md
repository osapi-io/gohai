# Uptime

> **Status:** Implemented ✅

## Description

Reports how long the system has been running (wall-clock seconds since boot),
the boot timestamp, and — on Linux only — aggregate CPU idle seconds since boot.
The idle counter is a sum across all logical CPUs, so on an 8-core host with a
day of uptime you can see ~8 days of idle — that's the expected Linux semantic,
matching `cat /proc/uptime`.

Consumers use this to:

- Spot recently-rebooted hosts (low uptime → maintenance window or unplanned
  reboot).
- Detect hosts that have been up too long (patching compliance).
- Correlate incidents with boot events via boot_time.

## Collected Fields

| Field          | Type   | Description                                               | Schema mapping                                                                                          |
| -------------- | ------ | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `seconds`      | uint64 | Seconds since boot.                                       | No direct schema mapping. OCSF records individual event times; host uptime isn't a first-class concept. |
| `boot_time`    | uint64 | Unix timestamp of boot.                                   | OCSF `device.first_seen_time` captures the same idea for asset-inventory use cases.                     |
| `human`        | string | Human-readable uptime (e.g. `3d 4h 12m 5s`).              | N/A (presentation).                                                                                     |
| `idle_seconds` | uint64 | Aggregate CPU idle seconds across all cores (Linux only). | No direct schema mapping.                                                                               |
| `idle_human`   | string | Human-readable idle duration (Linux only).                | N/A (presentation).                                                                                     |

## Platform Support

| Platform | Supported                                         |
| -------- | ------------------------------------------------- |
| Linux    | ✅                                                |
| macOS    | ✅ (uptime/boot only — no idle counter available) |
| Other    | —                                                 |

## Example Output

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

## SDK Usage

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

## Enable/Disable

```bash
gohai --collector.uptime      # enable (default)
gohai --no-collector.uptime   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. gopsutil's `host.InfoWithContext` reads `/proc/uptime` (field 1) for
   `seconds` and synthesizes `boot_time` as `now - uptime`. We forward both
   verbatim and render `human` via `HumanDuration(seconds)`.
2. We additionally read `/proc/uptime` through the injected `avfs.VFS` and parse
   field 2 (aggregate idle time across all CPUs — can exceed wall-clock uptime
   on multi-core systems) into `idle_seconds`; `idle_human` renders via the same
   helper. Missing file, malformed content, or a negative value leaves the idle
   fields zero-valued.

On macOS:

1. gopsutil's `host.InfoWithContext` reads `kern.boottime` via sysctl and
   computes `seconds` as `now - boot_epoch`. Same mapping to `seconds` /
   `boot_time` / `human` as Linux. macOS has no aggregate idle counter, so
   `idle_seconds` / `idle_human` are omitted from the output.

Mirrors Ohai's `uptime.rb` — same `/proc/uptime` parse on Linux, same
`kern.boottime`-derived uptime on Darwin. Our `boot_time` field is an additional
surface not in Ohai. Human-rendering format is compact (`1d 2h 3m 4s`) rather
than Ohai's verbose pluralized form (`1 day 02 hours 03 minutes 04 seconds`) —
presentation choice.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  uptime + boot time. Linux idle seconds is our own `/proc/uptime` parse
  (gopsutil doesn't expose the idle field).
