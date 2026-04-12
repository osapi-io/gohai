# OSAPI Integration Guide

gohai is designed SDK-first, and [OSAPI][osapi] is the primary consumer. This
doc maps each field in OSAPI's `job.FactsRegistration` struct (the fact payload
agents send back to the server) to the gohai collector and typed struct field
that provides it.

OSAPI agents embed gohai, run `g.Collect(ctx)`, and project the result into
their `FactsRegistration` before registering with the server. This doc is the
contract between the two projects.

## OSAPI `FactsRegistration` → gohai mapping

| OSAPI field           | gohai collector                                     | gohai source                                      | Status                     |
| --------------------- | --------------------------------------------------- | ------------------------------------------------- | -------------------------- |
| `hostname` (envelope) | [`hostname`](../collectors/hostname.md)             | `hostname.Info.Hostname`                          | ✅ implemented             |
| `Architecture`        | [`platform`](../collectors/platform.md)             | `platform.Info.Architecture`                      | ✅ implemented             |
| _(OS / os family)_    | [`platform`](../collectors/platform.md)             | `platform.Info.OS`, `platform.Info.Family`        | ✅ implemented             |
| `KernelVersion`       | [`kernel`](../collectors/kernel.md)                 | `kernel.Info.Version`                             | ✅ implemented             |
| `CPUCount`            | [`cpu`](../collectors/cpu.md)                       | `cpu.Info.Total`                                  | ✅ implemented             |
| `FQDN`                | [`hostname`](../collectors/hostname.md)             | `hostname.Info.FQDN`                              | ✅ implemented             |
| `ServiceMgr`          | [`init`](../collectors/init.md)                     | `init.Info.Name` (`systemd`, `openrc`, `launchd`) | 🚧 planned                 |
| `PackageMgr`          | [`package_mgr`](../collectors/package_mgr.md)       | `package_mgr.Info.Name` (`apt`, `dnf`, `brew`, …) | 🚧 planned                 |
| `Containerized`       | [`virtualization`](../collectors/virtualization.md) | `virtualization.Info.Role == "guest"`             | ✅ implemented             |
| `Interfaces`          | [`network`](../collectors/network.md)               | `network.Info.Interfaces`                         | 🚧 planned                 |
| `PrimaryInterface`    | [`network`](../collectors/network.md)               | `network.Info.DefaultInterface`                   | 🚧 planned                 |
| `Routes`              | [`network`](../collectors/network.md)               | `network.Info.Routes`                             | 🚧 planned                 |
| `Facts` (custom map)  | —                                                   | OSAPI-layer concern (user-supplied)               | N/A                        |

## Usage pattern

```go
package facts

import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
    "github.com/osapi-io/gohai/pkg/gohai/collectors/platform"
    // ... other collector sub-packages as needed
    "github.com/osapi-io/osapi/internal/job"
)

func Collect(
    ctx context.Context,
) (*job.FactsRegistration, error) {
    g, err := gohai.New()
    if err != nil {
        return nil, err
    }
    f, err := g.Collect(ctx)
    if err != nil {
        return nil, err
    }

    reg := &job.FactsRegistration{}
    if p, ok := f.Data["platform"].(*platform.Info); ok {
        reg.Architecture = p.Architecture
    }
    // ... project other collectors similarly as they become available
    return reg, nil
}
```

The typed cast is safe because gohai exports each collector's `Info` struct from
`pkg/gohai/collectors/<name>/`.

## Known gaps

The fields below are tracked in the README's collector tables (🚧 in the
Implemented column) and in the
[Hardware implementation plan](../superpowers/plans/2026-04-11-gohai-hardware.md).
Additional plans (Network, Software, etc.) will follow the same format.

- Everything except `Architecture` is still 🚧 as of today.
- `package_mgr` and `init` weren't on the original Ohai-parity list; they were
  added specifically to fill OSAPI's `PackageMgr` / `ServiceMgr` fields with a
  single typed fact per concept.
- `Containerized` is a derived bool from `virtualization.Info.Role`; OSAPI must
  project the richer `virtualization.Info` down to the bool when populating
  `FactsRegistration`.

## Keeping this doc in sync

Any change to OSAPI's `FactsRegistration` struct or gohai's collector `Info`
structs should update the mapping table above. This doc is the authoritative
reference for OSAPI integration.

[osapi]: https://github.com/osapi-io/osapi
