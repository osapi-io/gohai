# Libvirt

> **Status:** Implemented ✅

## Description

Reports libvirt domain information from a KVM/QEMU hypervisor host. Uses the
`virsh` CLI (part of `libvirt-client` / `libvirt-bin`) rather than the
`ruby-libvirt` gem used by Ohai, since there is no maintained Go libvirt binding
with equivalent coverage. When `virsh` is absent or cannot connect to the
daemon, `Collect` returns `nil` with no error.

Only Linux is implemented — KVM requires Linux and the libvirt daemon
(`libvirtd`) is a Linux service. macOS always returns `nil`.

## Collected Fields

### Top-level `Info`

| Field     | Type     | Description                                     | Schema mapping            |
| --------- | -------- | ----------------------------------------------- | ------------------------- |
| `uri`     | string   | libvirt connection URI (e.g. `qemu:///system`). | No direct schema mapping. |
| `version` | string   | libvirt daemon version (e.g. `10.0.0`).         | No direct schema mapping. |
| `domains` | []Domain | List of all domains (running and stopped).      | No direct schema mapping. |

### `Domain`

| Field        | Type   | Description                                            | Schema mapping                     |
| ------------ | ------ | ------------------------------------------------------ | ---------------------------------- |
| `name`       | string | Domain name.                                           | `process.name` (closest analogue). |
| `uuid`       | string | Domain UUID.                                           | No direct schema mapping.          |
| `state`      | string | Domain state (`running`, `shut off`, `paused`, etc.).  | No direct schema mapping.          |
| `vcpus`      | int    | Number of virtual CPUs.                                | No direct schema mapping.          |
| `max_memory` | string | Maximum memory allocation (e.g. `2097152 KiB`).        | No direct schema mapping.          |
| `autostart`  | bool   | Whether the domain autostarts with the libvirt daemon. | No direct schema mapping.          |

## Platform Support

| Platform | Supported                                                                           |
| -------- | ----------------------------------------------------------------------------------- |
| Linux    | ✅ (`virsh version`, `virsh uri`, `virsh list --all`, `virsh dominfo` via executor) |
| macOS    | `nil` (KVM not supported on Darwin)                                                 |

## Example Output

### KVM host with two domains

```json
{
  "libvirt": {
    "uri": "qemu:///system",
    "version": "10.0.0",
    "domains": [
      {
        "name": "myvm",
        "uuid": "aaaabbbb-cccc-dddd-eeee-ffffffffffff",
        "state": "running",
        "vcpus": 4,
        "max_memory": "2097152 KiB",
        "autostart": true
      },
      {
        "name": "stopped-vm",
        "uuid": "11112222-3333-4444-5555-666677778888",
        "state": "shut off",
        "vcpus": 2,
        "max_memory": "1048576 KiB",
        "autostart": false
      }
    ]
  }
}
```

### Non-KVM host or virsh absent

```json
{
  "libvirt": null
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("libvirt"))
facts, _ := g.Collect(context.Background())
if lv := facts.Libvirt; lv != nil {
    for _, d := range lv.Domains {
        fmt.Println(d.Name, d.State, d.VCPUs)
    }
}
```

## Enable/Disable

```bash
gohai --collector.libvirt    # enable (opt-in)
gohai --no-collector.libvirt # disable
```

DefaultEnabled: `false` — libvirt is only present on KVM hypervisor hosts;
callers must opt in explicitly.

## Dependencies

None.

## Data Sources

On Linux the collector runs four `virsh` commands in sequence:

1. **`virsh version`** — probe and version extraction. If this fails (virsh
   absent or daemon down), `Collect` returns `nil` immediately. The version is
   extracted from the "Running against daemon:" line first (daemon version);
   when absent, falls back to "Using library: libvirt \<ver\>" (library
   version). Matches Ohai's `libvirt_version` reporting.

2. **`virsh uri`** — connection URI (e.g. `qemu:///system`). Errors are silently
   ignored; `uri` stays empty rather than failing the whole collection.

3. **`virsh list --all`** — enumerates all domains regardless of state. Errors
   yield an empty domain list, not a failure. The output table is parsed by
   skipping the header line and separator, then extracting name (column 2) and
   state (columns 3+, joined with spaces to handle "shut off").

4. **`virsh dominfo <name>`** — enriches each domain with UUID, vCPU count,
   maximum memory, and autostart status. Per-domain errors are silently ignored;
   the domain is kept in the list with only name and state populated. Lines
   without a colon separator are skipped.

Ohai's libvirt plugin uses the `ruby-libvirt` gem for direct API access. We use
the `virsh` CLI instead — it is always available when libvirt is installed and
does not require native bindings. The information surfaced is a subset of Ohai's
full output (we do not enumerate networks, storage pools, or node hardware
topology).

macOS is not covered — `Collect` always returns `nil` on Darwin.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction for all `virsh` invocations. Tests mock it with
  `go.uber.org/mock`.
