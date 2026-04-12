# Virtualization

> **Status:** Implemented ✅

## Description

Detects the hypervisor or container runtime the host is running under
(KVM/VMware/Xen/Hyper-V/docker/lxc/…) and whether this host is the `host` or the
`guest` of that runtime.

Detection runs through gopsutil's heuristics: DMI / product name
(`/sys/class/dmi/id/product_name`, `sys_vendor`), CPUID flags (hypervisor leaf),
`/proc/1/cgroup` / `/proc/self/cgroup` for container runtimes, and
macOS-specific sysctls.

Consumers use this to:

- Branch metrics collection (disabling certain facts inside VMs / containers).
- Tag telemetry with the virtualization layer.
- Detect nested virtualization (a KVM guest that is also a docker host — current
  shape only reports one layer; multi-layer is a known gap).

## Collected Fields

| Field    | Type   | Description                                                                                     | OCSF mapping    |
| -------- | ------ | ----------------------------------------------------------------------------------------------- | --------------- |
| `system` | string | Runtime name: `"kvm"`, `"vmware"`, `"xen"`, `"hyperv"`, `"docker"`, `"lxc"`, `""` (bare metal). | No direct OCSF. |
| `role`   | string | `"host"`, `"guest"`, or `""` (unknown / not applicable).                                        | No direct OCSF. |

Empty `system` + empty `role` means "no virtualization detected" (bare metal) —
gopsutil didn't find hypervisor or container signatures.

## Platform Support

| Platform | Supported                                                         |
| -------- | ----------------------------------------------------------------- |
| Linux    | ✅ (DMI + CPUID + cgroup parse via gopsutil)                      |
| macOS    | ✅ (sysctl `kern.hv_vmm_present` / process ancestry via gopsutil) |

## Example Output

### KVM guest

```json
{
  "virtualization": {
    "system": "kvm",
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

### Docker host

```json
{
  "virtualization": {
    "system": "docker",
    "role": "host"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("virtualization"))
facts, _ := g.Collect(context.Background())

v := facts.Virtualization
switch {
case v.System == "":
    // bare metal
case v.Role == "guest":
    fmt.Printf("running under %s\n", v.System)
case v.Role == "host":
    fmt.Printf("hosting %s workloads\n", v.System)
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

| Platform | What we read                                                                                                                                                        | Ohai plugin                                                                                                                                                                                                          | Alignment                                                                                                                                                                                                                                                                                                                                                |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `host.Virtualization` — checks `/sys/class/dmi/id/product_name`/`sys_vendor`, CPUID hypervisor leaf, `/proc/1/cgroup`, `/proc/self/cgroup`, `/.dockerenv`. | [`linux/virtualization.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/virtualization.rb) — same DMI/cgroup/env checks plus `/proc/xen`, `/proc/vz`, `/proc/bc`, `lxc-ls`, `systemd-detect-virt`. | **Substantially same signal sources** (DMI + cgroup). Ohai additionally produces a `systems` map (e.g. `{"kvm": "guest", "docker": "host"}`) so nested runtimes surface both layers, and calls `systemd-detect-virt` as a fast-path. Our current shape reports only the outermost layer — multi-layer is tracked as a follow-up (issue #44 in the plan). |
| macOS    | gopsutil `host.Virtualization` — sysctl `kern.hv_vmm_present`, process ancestry.                                                                                    | Ohai's `darwin/virtualization.rb` has minimal macOS coverage (largely no-op).                                                                                                                                        | **Equivalent or better** — gopsutil's macOS detection is more complete than Ohai's.                                                                                                                                                                                                                                                                      |

**Known gaps vs. Ohai:**

- No `systems` map for nested / multi-layer virtualization.
- No explicit detection of `nspawn`, `podman`, `rkt`, `openvz` (gopsutil folds
  some of these into `lxc`/`docker`).
- No `systemd-detect-virt` fast-path.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3.
