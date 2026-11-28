# Libvirt

> **Status:** Implemented ✅

## Description

The collector reports the libvirt domain data from a KVM or QEMU hypervisor
host. It uses the `virsh` CLI, which is part of `libvirt-client` or
`libvirt-bin`. Ohai uses the `ruby-libvirt` gem. gohai does not use a Go
binding, because no maintained Go libvirt binding has equivalent coverage. If
`virsh` is missing or cannot connect to the daemon, `Collect` returns `nil` and
no error.

The collector supports only Linux. KVM needs Linux, and the libvirt daemon
(`libvirtd`) is a Linux service. On macOS, the collector always returns `nil`.

## Collected fields

### Top-level `Info`

| Field     | Type     | Description                                               | Schema mapping            |
| --------- | -------- | --------------------------------------------------------- | ------------------------- |
| `uri`     | string   | The libvirt connection URI, for example `qemu:///system`. | No direct schema mapping. |
| `version` | string   | The libvirt daemon version, for example `10.0.0`.         | No direct schema mapping. |
| `domains` | []Domain | The list of all domains, running and stopped.             | No direct schema mapping. |

### `Domain`

| Field        | Type   | Description                                                      | Schema mapping                      |
| ------------ | ------ | ---------------------------------------------------------------- | ----------------------------------- |
| `name`       | string | The domain name.                                                 | `process.name` (the nearest match). |
| `uuid`       | string | The domain UUID.                                                 | No direct schema mapping.           |
| `state`      | string | The domain state, for example `running`, `shut off` or `paused`. | No direct schema mapping.           |
| `vcpus`      | int    | The number of virtual CPUs.                                      | No direct schema mapping.           |
| `max_memory` | string | The maximum memory allocation, for example `2097152 KiB`.        | No direct schema mapping.           |
| `autostart`  | bool   | True if the domain starts automatically with the libvirt daemon. | No direct schema mapping.           |

## Platform support

| Platform | Supported                                                                                   |
| -------- | ------------------------------------------------------------------------------------------- |
| Linux    | ✅ (`virsh version`, `virsh uri`, `virsh list --all`, `virsh dominfo` through the executor) |
| macOS    | `nil` (Darwin does not support KVM)                                                         |

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("libvirt"))
facts, _ := g.Collect(context.Background())
if lv := facts.Libvirt; lv != nil {
    for _, d := range lv.Domains {
        fmt.Println(d.Name, d.State, d.VCPUs)
    }
}
```

## Enable/disable

```bash
gohai collect --collector.libvirt    # enable (opt-in)
gohai collect --no-collector.libvirt # disable
```

`DefaultEnabled` is `false`, because only KVM hypervisor hosts have libvirt.
Callers must enable the collector explicitly.

## Dependencies

None.

## Data sources

On Linux, the collector runs four `virsh` commands in sequence:

1. **`virsh version`**. The probe, and the source of the version. If this
   command fails, `Collect` returns `nil` immediately. The command fails if
   `virsh` is missing or the daemon is down. The collector first gets the daemon
   version from the "Running against daemon:" line. If that line is not there,
   it uses the library version from the "Using library: libvirt \<ver>" line.
   This is the same as the `libvirt_version` that Ohai reports.

2. **`virsh uri`**. The connection URI, for example `qemu:///system`. The
   collector returns an error if this command fails.

3. **`virsh list --all`**. The list of all domains, in all states. If this
   command fails, the collector returns an error. To parse the output table, the
   collector skips the header line and the separator. Then it gets the name from
   column 2 and the state from columns 3 and higher. It joins the state columns
   with spaces, so that "shut off" stays one value.

4. **`virsh dominfo <name>`**. The collector adds the UUID, the vCPU count, the
   maximum memory and the autostart status to each domain. A domain can go away
   between the list and its `dominfo`. Thus, if `dominfo` fails, the domain
   stays in the list with only its name and state. The collector skips lines
   that have no colon separator.

Ohai's libvirt plugin uses the `ruby-libvirt` gem for direct API access. gohai
uses the `virsh` CLI. `virsh` is always there when libvirt is installed, and it
does not need native bindings. The collector reports a subset of the full Ohai
output. It does not list networks, storage pools or the hardware topology of the
node.

The collector does not support macOS. On Darwin, `Collect` always returns `nil`.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run all `virsh` commands. Tests mock it with
  `go.uber.org/mock`.
