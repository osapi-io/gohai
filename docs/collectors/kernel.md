# Kernel

> **Status:** Implemented ✅

## Description

Identifies the kernel OS name, release version, and architecture. Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host).

## Collected Fields

| Field     | Type   | Description                                         |
| --------- | ------ | --------------------------------------------------- |
| `os`      | string | Kernel OS name (e.g., `linux`, `darwin`)            |
| `version` | string | Kernel release (e.g., `6.8.0-31-generic`, `23.4.0`) |
| `arch`    | string | Kernel architecture (e.g., `x86_64`, `arm64`)       |

## Platform Support

| Platform | Source                             | Supported |
| -------- | ---------------------------------- | --------- |
| Linux    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| macOS    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| Other    | Returns `nil`                      | —         |

## Example Output

```json
{
  "kernel": {
    "os": "linux",
    "version": "6.8.0-31-generic",
    "arch": "x86_64"
  }
}
```

## SDK Usage

```go
info := facts.Data["kernel"].(*kernel.Info)
fmt.Println(info.Version)
```

## Enable/Disable

```bash
gohai --collector.kernel      # enable (default)
gohai --no-collector.kernel   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
BSD-3.
