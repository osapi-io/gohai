# Platform

> **Status:** Implemented ✅

## Description

Identifies the operating system platform: name, version, family, architecture,
and (on macOS) kernel build. Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host)
and reshapes the output into a typed `platform.Info` struct.

## Collected Fields

| Field          | Type   | Description                                               |
| -------------- | ------ | --------------------------------------------------------- |
| `os`           | string | Coarse OS (`runtime.GOOS`): `linux`, `darwin`, `windows`  |
| `name`         | string | Distro/product (e.g., `ubuntu`, `rhel`, `darwin`)         |
| `version`      | string | OS version (e.g., `24.04`, `14.4.1`)                      |
| `family`       | string | OS family (e.g., `debian`, `rhel`, `mac_os_x`)            |
| `architecture` | string | CPU architecture (e.g., `amd64`, `arm64`, `aarch64`)      |
| `build`        | string | Kernel build string (macOS only; maps to `KernelVersion`) |

## Platform Support

| Platform | Source                             | Supported |
| -------- | ---------------------------------- | --------- |
| Linux    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| macOS    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| Other    | Returns `nil` (no platform data)   | —         |

gopsutil internally reads `/etc/os-release` (with legacy fallbacks) on Linux and
calls `sw_vers` / `uname` on macOS. See
[gopsutil host docs](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host) for
the full list of distro quirks it handles.

## Example Output

### Linux (Ubuntu 24.04 on amd64)

```json
{
  "platform": {
    "os": "linux",
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "architecture": "x86_64"
  }
}
```

### macOS (14.4.1 on Apple Silicon)

```json
{
  "platform": {
    "os": "darwin",
    "name": "darwin",
    "version": "14.4.1",
    "family": "mac_os_x",
    "architecture": "arm64",
    "build": "23.4.0"
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
    "github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
)

g, _ := gohai.New(gohai.WithCollectors("platform"))
facts, _ := g.Collect(context.Background())

info := facts.Platform
fmt.Println(info.Name, info.Version) // "ubuntu" "24.04"
```

## Enable/Disable

```bash
gohai --collector.platform      # enable (default)
gohai --no-collector.platform   # disable
```

## Dependencies

None — `platform` is a Tier 1 core collector with no upstream collector
dependencies.

## Backing library

[`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
BSD-3 licensed.
