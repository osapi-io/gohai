# Uptime

> **Status:** Implemented ✅

## Description

Reports system uptime (seconds since boot) and boot time (unix timestamp), plus
a human-readable uptime string. Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host).

## Collected Fields

| Field       | Type   | Description                                  |
| ----------- | ------ | -------------------------------------------- |
| `seconds`   | uint64 | Seconds since boot                           |
| `boot_time` | uint64 | Unix timestamp (seconds) of system boot      |
| `human`     | string | Human-readable uptime (e.g., `3d 4h 12m 5s`) |

## Platform Support

| Platform | Source                             | Supported |
| -------- | ---------------------------------- | --------- |
| Linux    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| macOS    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| Other    | Returns `nil`                      | —         |

## Example Output

```json
{
  "uptime": {
    "seconds": 360725,
    "boot_time": 1700000000,
    "human": "4d 4h 12m 5s"
  }
}
```

## SDK Usage

```go
info := facts.Data["uptime"].(*uptime.Info)
fmt.Println(info.Human)
```

## Enable/Disable

```bash
gohai --collector.uptime      # enable (default)
gohai --no-collector.uptime   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
BSD-3.
