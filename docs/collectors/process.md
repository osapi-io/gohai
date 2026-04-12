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

## Example Output

```json
{
  "process": {
    "count": 312,
    "processes": [
      {
        "pid": 1,
        "name": "systemd",
        "username": "root",
        "cmdline": "/sbin/init"
      },
      {
        "pid": 1247,
        "name": "nginx",
        "username": "www-data",
        "cmdline": "nginx: worker process"
      }
    ]
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("process"))
facts, _ := g.Collect(context.Background())

fmt.Printf("%d processes running\n", facts.Process.Count)
for _, p := range facts.Process.Processes {
    fmt.Printf("  pid=%d name=%s user=%s\n", p.PID, p.Name, p.Username)
}
```

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
