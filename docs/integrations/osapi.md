# OSAPI Integration Guide

gohai is designed SDK-first, and [OSAPI][osapi] is the primary consumer. This
doc maps each field in OSAPI's `job.FactsRegistration` struct (the fact payload
agents send back to the server) to the gohai collector and typed struct field
that provides it.

OSAPI agents embed gohai, run `g.Collect(ctx)`, and project the result into
their `FactsRegistration` before registering with the server. This doc is the
contract between the two projects.

## OSAPI `FactsRegistration` → gohai mapping

| OSAPI field           | gohai collector                                     | gohai source                                      | Status         |
| --------------------- | --------------------------------------------------- | ------------------------------------------------- | -------------- |
| `hostname` (envelope) | [`hostname`](../collectors/hostname.md)             | `hostname.Info.Hostname`                          | ✅ implemented |
| `Architecture`        | [`platform`](../collectors/platform.md)             | `platform.Info.Architecture`                      | ✅ implemented |
| _(OS / os family)_    | [`platform`](../collectors/platform.md)             | `platform.Info.OS`, `platform.Info.Family`        | ✅ implemented |
| `KernelVersion`       | [`kernel`](../collectors/kernel.md)                 | `kernel.Info.Version`                             | ✅ implemented |
| `CPUCount`            | [`cpu`](../collectors/cpu.md)                       | `cpu.Info.Total`                                  | ✅ implemented |
| `FQDN`                | [`hostname`](../collectors/hostname.md)             | `hostname.Info.FQDN`                              | ✅ implemented |
| `ServiceMgr`          | [`init`](../collectors/init.md)                     | `init.Info.Name` (`systemd`, `openrc`, `launchd`) | 🚧 planned     |
| `PackageMgr`          | [`package_mgr`](../collectors/package_mgr.md)       | `package_mgr.Info.Name` (`apt`, `dnf`, `brew`, …) | 🚧 planned     |
| `Containerized`       | [`virtualization`](../collectors/virtualization.md) | `virtualization.Info.Role == "guest"`             | ✅ implemented |
| `Interfaces`          | [`network`](../collectors/network.md)               | `network.Info.Interfaces`                         | ✅ implemented |
| `PrimaryInterface`    | [`network`](../collectors/network.md)               | `network.Info.DefaultInterface`                   | 🚧 planned     |
| `Routes`              | [`network`](../collectors/network.md)               | `network.Info.Routes`                             | 🚧 planned     |
| `Facts` (custom map)  | —                                                   | OSAPI-layer concern (user-supplied)               | N/A            |

## Usage pattern

### In-process (OSAPI imports gohai)

```go
package facts

import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"

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
    if f.Platform != nil {
        reg.Architecture = f.Platform.Architecture
    }
    if f.Hostname != nil {
        reg.FQDN = f.Hostname.FQDN
    }
    if f.Kernel != nil {
        reg.KernelVersion = f.Kernel.Version
    }
    if f.CPU != nil {
        reg.CPUCount = f.CPU.Total
    }
    if f.Virtualization != nil {
        reg.Containerized = f.Virtualization.Role == "guest"
    }
    if f.Network != nil {
        reg.Interfaces = make([]job.NetworkInterface, 0, len(f.Network.Interfaces))
        // ... project network.Info into OSAPI's shape
    }
    return reg, nil
}
```

No collector sub-package imports — the typed fields on `gohai.Facts` give you
direct access. Each field is a `nil`-check before use (collector was disabled
or failed).

### Serialized handoff (agent → server)

Agents may serialize `gohai.Facts` as JSON, store it, and have the server
re-parse later. The JSON schema is stable (pinned by `TestUnmarshalStoredBlob`
in the gohai test suite):

```go
// Agent side
f, _ := g.Collect(ctx)
blob, _ := f.JSON()
// store blob in KV

// Server side — later
var f gohai.Facts
json.Unmarshal(blob, &f)
reg.FQDN = f.Hostname.FQDN  // typed access, no assertions
```

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
