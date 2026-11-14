# Virtualization

> **Status:** Implemented ✅

## Description

The collector finds each hypervisor and container runtime that the host is part
of. It reports the host as **guest**, **host**, or both. One host can correctly
report more than one system. Examples are a Docker host that is also a KVM guest
on EC2, and an LXD host on bare metal. The checks are the same as the sequences
in Ohai's linux/virtualization.rb and darwin/virtualization.rb.

Consumers use this to:

- Tag telemetry with each virtualization layer.
- Stop or start metric collection inside containers.
- Find nesting that is not expected. An example is a container that must not run
  on a hypervisor guest, or the opposite.
- Tell the host role from the guest role for the same runtime. An example is a
  vbox host on bare metal and a vbox guest under VirtualBox.

## Collected fields

| Field             | Type              | Description                                                                                                                         | Schema mapping            |
| ----------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `system`          | string            | The primary system, that is, the last positive result (`"docker"`, `"kvm"`, `"lxc"`, `""` for bare metal).                          | No direct schema mapping. |
| `role`            | string            | `"host"`, `"guest"`, or `""`.                                                                                                       | No direct schema mapping. |
| `systems`         | map[string]string | Each layer that the collector finds: `{"kvm": "guest", "docker": "host"}`. One entry on a host with one layer. Empty on bare metal. | No direct schema mapping. |
| `hypervisor_host` | string            | The hostname of the hypervisor, if the guest can see it. On Linux, the value comes from the Hyper-V KVP pool.                       | No direct schema mapping. |

An empty `system` and an empty `systems` mean that the collector found no
virtualization (bare metal). If the collector finds more than one layer,
`system` and `role` give the last positive result in the order of the checks. If
a consumer needs each layer, it must read all of `systems`.

## Platform support

| Platform | Supported                                                                             |
| -------- | ------------------------------------------------------------------------------------- |
| Linux    | ✅ (all Ohai checks: systemd-detect-virt, DMI, cgroup and 12 file and command checks) |
| macOS    | ✅ (Ohai checks: PATH checks, sysctl, ioreg and system_profiler)                      |

## Example output

### Bare metal

```json
{ "virtualization": {} }
```

### KVM guest running Docker

```json
{
  "virtualization": {
    "system": "docker",
    "role": "host",
    "systems": {
      "kvm": "guest",
      "docker": "host"
    }
  }
}
```

### VMware Fusion host on macOS

```json
{
  "virtualization": {
    "system": "vmware",
    "role": "host",
    "systems": { "vmware": "host" }
  }
}
```

### Apple virtualization.framework guest

```json
{
  "virtualization": {
    "system": "apple",
    "role": "guest",
    "systems": { "qemu": "guest", "apple": "guest" }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("virtualization"))
facts, _ := g.Collect(context.Background())

v := facts.Virtualization
if len(v.Systems) == 0 {
    // bare metal
}
for name, role := range v.Systems {
    fmt.Printf("%s: %s\n", name, role)
}
```

## Enable/disable

```bash
gohai --collector.virtualization      # enable (default)
gohai --no-collector.virtualization   # disable
```

## Dependencies

None.

## Data sources

On Linux, the collector does each check that Ohai's `linux/virtualization.rb`
does, in sequence. For each positive result, it sets `systems[<name>] = role`.
The order is important. The last positive result sets the primary `system` and
`role`, but each layer stays in `systems`:

01. **`systemd-detect-virt` fast path.** If `systemd-detect-virt` is on PATH,
    the collector runs `systemd-detect-virt --vm` and
    `systemd-detect-virt --container`. Each result that is not `none` and not
    empty is a guest.

02. **Container-runtime hosts.** `command -v docker`, `command -v podman` or
    `command -v nova` → host.

03. **Xen.** `/proc/xen` is there → guest. `/proc/xen/capabilities` has
    `control_d` → host.

04. **VirtualBox.** A `vboxdrv` line in `/proc/modules` → host. `vboxguest` →
    guest.

05. **KVM.** `/proc/cpuinfo` has `QEMU Virtual CPU`, `Common KVM processor` or
    `Common 32-bit KVM processor` → guest. `/sys/devices/virtual/misc/kvm` is
    there → host. It is a guest if the `hypervisor` cpuinfo flag is set. Also,
    the collector reads the earlier result of the `cpu` collector (from
    `lscpu`). If it has `HypervisorVendor == "KVM"` and a `VirtualizationType`
    of `full` or `para`, the collector records a kvm guest. This check finds
    nested VMs that do not have the `/sys/devices/virtual/misc/kvm` node.

06. **DMI.** The collector first compares `/sys/class/dmi/id/sys_vendor` with
    the manufacturer table of Ohai's `guest_from_dmi_data`: `OpenStack`, `Xen`,
    `VMware`, `Microsoft` (with `Virtual Machine` in the product) → hyperv,
    `Amazon EC2`, `QEMU` → kvm, `Veertu`, `Parallels`. If no manufacturer
    agrees, the collector compares `/sys/class/dmi/id/product_name` with the
    product table: `VirtualBox` → vbox, `OpenStack` (the Red Hat variant), `KVM`
    or `RHEV` → kvm, `BHYVE`.

07. **OpenVZ.** `/proc/bc/0` → host. `/proc/vz` → guest.

08. **Hyper-V.** `/var/lib/hyperv/.kvp_pool_3` → guest.

09. **linux-vserver.** `s_context: 0` or `VxID: 0` in `/proc/self/status` →
    host. A value that is not zero → guest.

10. **Containers through cgroup and environ.** The collector compares
    `/proc/self/cgroup` with two regexes:

    - Direct: `/(docker|lxc|containerd)/…` → the runtime in the path. The
      collector records containerd as docker.
    - Nested (systemd-managed, docker-ce, cgroup v2): `/<parent>/docker-…` or
      `/<parent>/lxc-…`. This regex finds `/system.slice/docker-<hash>.scope`,
      `/docker-ce/docker/<hash>`, `/kubepods/.../docker-<hash>.scope` and the
      GitHub Actions runner layouts.

    The collector also reads `/proc/1/environ` for `container=lxc`,
    `container=systemd-nspawn` or `container=podman`.

11. **LXC host.** This check runs only if no earlier check set `system`, and
    each line of `/proc/self/cgroup` has the root path `/`. A root path of `/`
    shows that the process is not inside the cgroup namespace of a container. If
    `command -v lxc-version` or `command -v lxc-start` is successful, the
    collector records the host as an `lxc` host. This is the same as Ohai's
    OHAI-573 guard.

12. **`.dockerenv` override.** `/.dockerenv` or `/.dockerinit` → docker guest.
    This result replaces the earlier results.

13. **LXD.** `/dev/lxd/sock` → guest. `/var/lib/lxd/devlxd` or
    `/var/snap/lxd/common/lxd/devlxd` → host.

On macOS, the checks are the same as Ohai's `darwin/virtualization.rb`:

1. **Hypervisor host binaries.** `command -v docker` → `systems[docker] = host`.
   `command -v VBoxManage` → `vbox` host. `command -v prlctl` → `parallels`
   host.
2. **VMware Fusion host.** `/Applications/VMware Fusion.app` is there.
3. **QEMU or Virtualization.framework guest.** `sysctl -n kern.hv_vmm_present`
   returns `1`.
4. **Parallels guest.** The `ioreg -l` output has `pci1ab8,4000`.
5. **VirtualBox, VMware or Apple VM guest.** The collector parses
   `system_profiler SPHardwareDataType`. It looks for `VirtualBox` or `VMW` in
   `Boot ROM Version`, and for `VirtualMac` in `Model Identifier`.

A bare-metal Mac with no hypervisor software gives an empty `systems` map and an
empty `system` and `role`.

All file reads go through the injected `avfs.VFS`. All commands run through the
shared `internal/executor` runner. Tests mock the two with `memfs` and
`go.uber.org/mock`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs). The virtual filesystem
  for the checks of `/proc`, `/sys`, `/var/lib/...` and `/Applications/...`.
  Tests inject `memfs` with fixed fixtures.
- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `systemd-detect-virt`, `command -v <bin>`, `sysctl`,
  `ioreg` and `system_profiler`. Tests mock it with `go.uber.org/mock`.
