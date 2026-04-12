# Process

> **Status:** Implemented ✅

## Description

Snapshot of running processes with PID, name, username, and command line. Wraps
[gopsutil's `process`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/process).

Processes the caller doesn't have permission to introspect will appear in the
list with empty name/username/cmdline fields (only `pid` populated).

## Collected Fields

Top-level:

| Field       | Type      | Description               |
| ----------- | --------- | ------------------------- |
| `count`     | int       | Total number of processes |
| `processes` | []Process | Snapshot entries          |

Per-process:

| Field      | Type   | Description                         |
| ---------- | ------ | ----------------------------------- |
| `pid`      | int32  | Process ID                          |
| `name`     | string | Executable name (empty on denied)   |
| `username` | string | Owning user (empty on denied)       |
| `cmdline`  | string | Full command line (empty on denied) |

## Platform Support

| Platform | Source                                     | Supported |
| -------- | ------------------------------------------ | --------- |
| Linux    | `gopsutil/v4/process.ProcessesWithContext` | ✅        |
| macOS    | `gopsutil/v4/process.ProcessesWithContext` | ✅        |
| Other    | Returns `nil`                              | —         |

## Enable/Disable

```bash
gohai --collector.process      # enable (default)
gohai --no-collector.process   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/process`](https://github.com/shirou/gopsutil) —
BSD-3.
