# Virtualization

> **Status:** Implemented ✅

## Description

Detects every hypervisor and container runtime the host participates in — as
**guest**, **host**, or both. A single host can legitimately report multiple
systems (a Docker host that is itself a KVM guest on EC2, an LXD host on bare
metal, etc.). Mirrors Ohai's linux/virtualization.rb and
darwin/virtualization.rb cascades.

Consumers use this to:

- Tag telemetry with every virtualization layer.
- Gate metric collection inside containers.
- Detect unexpected nesting (a container that shouldn't be running on a
  hypervisor guest, or vice versa).
- Distinguish host vs guest roles for the same runtime (a vbox host running on
  bare metal vs a vbox guest under VirtualBox).

## Collected Fields

| Field     | Type              | Description                                                                                                          | Schema mapping  |
| --------- | ----------------- | -------------------------------------------------------------------------------------------------------------------- | --------------- |
| `system`  | string            | Primary / most-recent positive detection (`"docker"`, `"kvm"`, `"lxc"`, `""` for bare metal).                        | No direct OCSF. |
| `role`    | string            | `"host"`, `"guest"`, or `""`.                                                                                        | No direct OCSF. |
| `systems` | map[string]string | Every detected layer: `{"kvm": "guest", "docker": "host"}`. Single entry on single-layer hosts; empty on bare metal. | No direct OCSF. |

Empty `system` + empty `systems` means "no virtualization detected" (bare
metal). When more than one layer is detected, `system`/`role` report the last
positive detection in the cascade order — consumers that care about every layer
should iterate `systems`.

## Platform Support

| Platform | Supported                                                                        |
| -------- | -------------------------------------------------------------------------------- |
| Linux    | ✅ (full Ohai cascade: systemd-detect-virt + DMI + cgroup + 12 file/exec probes) |
| macOS    | ✅ (Ohai cascade: PATH probes + sysctl + ioreg + system_profiler)                |

## Example Output

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

### Apple Virtualization.framework guest

```json
{
  "virtualization": {
    "system": "apple",
    "role": "guest",
    "systems": { "qemu": "guest", "apple": "guest" }
  }
}
```

## SDK Usage

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

## Enable/Disable

```bash
gohai --collector.virtualization      # enable (default)
gohai --no-collector.virtualization   # disable
```

## Dependencies

None.

## Data Sources

On Linux the collector cascades through every signal Ohai's
`linux/virtualization.rb` checks, populating `systems[<name>] = role` for each
positive hit. Order matters — the last positive detection sets primary
`system`/`role`, but every layer remains in `systems`:

1. **`systemd-detect-virt` fast-path:** when on PATH, run
   `systemd-detect-virt --vm` and `systemd-detect-virt --container`; each
   non-`none`/non-empty result registers as guest.
2. **Container-runtime hosts:** `command -v docker` / `command -v podman` /
   `command -v nova` → host.
3. **Xen:** `/proc/xen` exists → guest; `/proc/xen/capabilities` contains
   `control_d` → host.
4. **VirtualBox:** `/proc/modules` line `vboxdrv` → host; `vboxguest` → guest.
5. **KVM:** `/proc/cpuinfo` contains `QEMU Virtual CPU` or
   `Common KVM processor` → guest. `/sys/devices/virtual/misc/kvm` exists → host
   (or guest when the `hypervisor` cpuinfo flag is set).
6. **DMI:** `/sys/class/dmi/id/product_name` / `sys_vendor` / `bios_vendor`
   matched against vmware / hyperv / parallels / xen / qemu/kvm strings (Ohai's
   `guest_from_dmi_data` table).
7. **OpenVZ:** `/proc/bc/0` → host; `/proc/vz` → guest.
8. **Hyper-V:** `/var/lib/hyperv/.kvp_pool_3` → guest.
9. **linux-vserver:** `/proc/self/status` `s_context: 0` / `VxID: 0` → host;
   non-zero → guest.
10. **Containers via cgroup / environ:** `/proc/self/cgroup` regex for `docker`
    / `lxc` / `containerd` (containerd remaps to docker); `/proc/1/environ` for
    `container=lxc` / `container=systemd-nspawn` / `container=podman`.
11. **`.dockerenv` override:** `/.dockerenv` or `/.dockerinit` → docker guest
    (force overrides earlier registrations).
12. **LXD:** `/dev/lxd/sock` → guest; `/var/lib/lxd/devlxd` or
    `/var/snap/lxd/common/lxd/devlxd` → host.

On macOS the cascade matches Ohai's `darwin/virtualization.rb`:

1. **Hypervisor host binaries:** `command -v docker` → `systems[docker] = host`;
   `command -v VBoxManage` → `vbox` host; `command -v prlctl` → `parallels`
   host.
2. **VMware Fusion host:** `/Applications/VMware Fusion.app` exists.
3. **QEMU / Virtualization.framework guest:** `sysctl -n kern.hv_vmm_present`
   returns `1`.
4. **Parallels guest:** `ioreg -l` output contains `pci1ab8,4000`.
5. **VirtualBox / VMware / Apple-VM guest:**
   `system_profiler SPHardwareDataType` parsed for `Boot ROM Version` containing
   `VirtualBox` / `VMW`, and `Model Identifier` containing `VirtualMac`.

Bare-metal Macs with no hypervisor software produce an empty `systems` map and
empty `system`/`role`.

All file reads go through the injected `avfs.VFS`; all command invocations go
through the shared `internal/executor` runner. Tests mock both with `memfs` and
`go.uber.org/mock`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) — virtual filesystem
  for the dozen `/proc`, `/sys`, `/var/lib/...`, and `/Applications/...` probes.
  Tests inject `memfs` with canned fixtures.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction for `systemd-detect-virt`, `command -v <bin>`, `sysctl`, `ioreg`,
  `system_profiler`. Tests mock with `go.uber.org/mock`.
