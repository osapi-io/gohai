# Machine ID

> **Status:** Implemented ✅

## Description

Reports the unique machine identifier (from `/etc/machine-id` on Linux,
`IOPlatformUUID` on macOS). Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host).

## Collected Fields

| Field | Type   | Description                            |
| ----- | ------ | -------------------------------------- |
| `id`  | string | Machine identifier (typically a UUID)  |

## Platform Support

| Platform | Source                             | Supported |
| -------- | ---------------------------------- | --------- |
| Linux    | `gopsutil/v4/host.InfoWithContext` (reads `/etc/machine-id`) | ✅ |
| macOS    | `gopsutil/v4/host.InfoWithContext` (reads `IOPlatformUUID`) | ✅ |
| Other    | Returns `nil`                      | —         |

## Example Output

```json
{
  "machine_id": {
    "id": "abc12345-6789-def0-1234-56789abcdef0"
  }
}
```

## Enable/Disable

```bash
gohai --collector.machine_id      # enable (default)
gohai --no-collector.machine_id   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) — BSD-3.
