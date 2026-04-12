# Virtualization

> **Status:** Implemented ✅

## Description

Detects hypervisor / container runtime presence. Reports whether the host is a
guest (inside a VM or container) or a host (running a hypervisor). Wraps
[gopsutil's `host.Info`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host)
which handles detection across kvm, xen, vmware, virtualbox, docker, lxc,
podman, and more.

## Collected Fields

| Field    | Type   | Description                                                      |
| -------- | ------ | ---------------------------------------------------------------- |
| `system` | string | Detected system (e.g., `docker`, `kvm`, `xen`, `vmware`, `vbox`) |
| `role`   | string | `host` or `guest`; empty on bare metal                           |

Both fields are empty when no virtualization is detected.

## Platform Support

| Platform | Source                             | Supported |
| -------- | ---------------------------------- | --------- |
| Linux    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| macOS    | `gopsutil/v4/host.InfoWithContext` | ✅        |
| Other    | Returns `nil`                      | —         |

## Example Output

### Docker container (guest)

```json
{
  "virtualization": {
    "system": "docker",
    "role": "guest"
  }
}
```

### Bare metal

```json
{
  "virtualization": {}
}
```

## SDK Usage

```go
info := facts.Virtualization
containerized := info.Role == "guest"
```

## Enable/Disable

```bash
gohai --collector.virtualization      # enable (default)
gohai --no-collector.virtualization   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
BSD-3.
