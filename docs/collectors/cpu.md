# CPU

> **Status:** Implemented ✅

## Description

Collects CPU topology and feature facts: logical CPU count, physical cores,
model name, vendor, family, flags, cache size, and clock speed. Wraps
[gopsutil's `cpu`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/cpu)
package.

## Collected Fields

| Field        | Type     | Description                                                |
| ------------ | -------- | ---------------------------------------------------------- |
| `total`      | int      | Logical CPU count                                          |
| `cores`      | int      | Physical core count                                        |
| `model_name` | string   | Human-readable CPU name (e.g., `Intel Core i7-10700K`)     |
| `vendor_id`  | string   | CPU vendor (e.g., `GenuineIntel`, `AuthenticAMD`, `Apple`) |
| `family`     | string   | CPU family                                                 |
| `model`      | string   | CPU model number                                           |
| `stepping`   | int32    | Silicon stepping                                           |
| `mhz`        | float64  | Clock speed in MHz                                         |
| `cache_size` | int32    | Cache size (KB)                                            |
| `flags`      | []string | CPU feature flags (e.g., `sse`, `avx`, `aes`)              |

## Platform Support

| Platform | Source                                                  | Supported |
| -------- | ------------------------------------------------------- | --------- |
| Linux    | `gopsutil/v4/cpu.InfoWithContext` + `CountsWithContext` | ✅        |
| macOS    | `gopsutil/v4/cpu.InfoWithContext` + `CountsWithContext` | ✅        |
| Other    | Returns `nil`                                           | —         |

## Example Output

```json
{
  "cpu": {
    "total": 8,
    "cores": 8,
    "model_name": "Apple M1",
    "vendor_id": "Apple",
    "mhz": 3200.0
  }
}
```

## SDK Usage

```go
info := facts.CPU
fmt.Println(info.Total, "logical CPUs")
```

## Enable/Disable

```bash
gohai --collector.cpu      # enable (default)
gohai --no-collector.cpu   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/cpu`](https://github.com/shirou/gopsutil) —
BSD-3.
