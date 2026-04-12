# Memory

> **Status:** Implemented ✅

## Description

Collects virtual and swap memory usage. Wraps
[gopsutil's `mem`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/mem).

## Collected Fields

| Field          | Type    | Description                      |
| -------------- | ------- | -------------------------------- |
| `total`        | uint64  | Total memory (bytes)             |
| `available`    | uint64  | Available memory (bytes)         |
| `used`         | uint64  | Used memory (bytes)              |
| `used_percent` | float64 | Percent used (0-100)             |
| `free`         | uint64  | Free memory (bytes)              |
| `buffers`      | uint64  | Buffers (Linux; bytes)           |
| `cached`       | uint64  | Cached memory (Linux; bytes)     |
| `swap`         | object  | Swap memory (omitted if no swap) |

`swap` nested object fields: `total`, `used`, `free`, `used_percent`.

## Platform Support

| Platform | Source                                         | Supported |
| -------- | ---------------------------------------------- | --------- |
| Linux    | `gopsutil/v4/mem.VirtualMemory` + `SwapMemory` | ✅        |
| macOS    | `gopsutil/v4/mem.VirtualMemory` + `SwapMemory` | ✅        |
| Other    | Returns `nil`                                  | —         |

## Example Output

```json
{
  "memory": {
    "total": 17179869184,
    "available": 8589934592,
    "used": 8589934592,
    "used_percent": 50.0,
    "free": 4294967296,
    "swap": {
      "total": 2147483648,
      "used": 1024,
      "free": 2147482624,
      "used_percent": 0.1
    }
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("memory"))
facts, _ := g.Collect(context.Background())

m := facts.Memory
fmt.Printf("%.1f%% used (%d / %d bytes)\n", m.UsedPercent, m.Used, m.Total)
```

## Enable/Disable

```bash
gohai --collector.memory      # enable (default)
gohai --no-collector.memory   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/mem`](https://github.com/shirou/gopsutil) —
BSD-3.
