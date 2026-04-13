# Collectors

gohai collects system facts through 65 pluggable collectors across 9 categories.
Each collector gathers a specific category of information and returns a
strongly-typed Go struct.

Collectors are individually toggled using node_exporter-style flags:

```bash
gohai --collector.platform --no-collector.cloud
```

**Defaults are opt-in.** `gohai.New()` (SDK) returns an empty registry — nothing
runs unless you ask for it. Pass `gohai.WithDefaults()` for the recommended set,
or `gohai.WithCollectors(...)` / `gohai.WithEnabled(...)` to enumerate. The CLI
wires `WithDefaults()` automatically; pass `--no-defaults` to turn it off and
use only explicit `--collector.X` flags. The "Default" column below indicates
membership in the recommended set (`✅` = on when `WithDefaults()` is in effect,
`❌` = opt-in only).

**Schema:** Field names follow [OCSF][] (Open Cybersecurity Schema Framework) as
the primary schema and [OpenTelemetry Resource Semantic
Conventions][otel-semconv] as the secondary when OCSF is silent. Browse
[schema.ocsf.io][ocsf-schema] and the [OpenTelemetry resource
attributes][otel-semconv] to see canonical names and object shapes. Each
collector's doc includes a **Collected Fields** table with a schema mapping
column citing the canonical path. Collection logic (what to read, which distro
edge cases to handle) follows [Chef Ohai][]'s plugins.

[OCSF]: https://ocsf.io/
[ocsf-schema]: https://schema.ocsf.io/
[otel-semconv]: https://opentelemetry.io/docs/specs/semconv/resource/
[Chef Ohai]: https://github.com/chef/ohai

## 🖥️ System

| Collector                   | Key          | Description                            | Default |
| --------------------------- | ------------ | -------------------------------------- | ------- |
| [platform](platform.md)     | `platform`   | OS name, version, family, architecture | ✅      |
| [hostname](hostname.md)     | `hostname`   | FQDN, domain, hostname, machine name   | ✅      |
| [kernel](kernel.md)         | `kernel`     | Version, modules, parameters           | ✅      |
| [uptime](uptime.md)         | `uptime`     | Boot time, uptime duration, idle time  | ✅      |
| [timezone](timezone.md)     | `timezone`   | System timezone                        | ✅      |
| [os_release](os_release.md) | `os_release` | `/etc/os-release` fields               | ✅      |
| [init](init.md)             | `init`       | Init system detection                  | ✅      |
| [fips](fips.md)             | `fips`       | FIPS mode detection                    | ✅      |
| [machine_id](machine_id.md) | `machine_id` | Machine ID                             | ✅      |
| [root_group](root_group.md) | `root_group` | Root's primary group                   | ✅      |
| [shells](shells.md)         | `shells`     | Available shells                       | ✅      |
| [shard](shard.md)           | `shard`      | Deterministic shard seed               | ✅      |

## ⚙️ Hardware

| Collector                   | Key          | Description                              | Default |
| --------------------------- | ------------ | ---------------------------------------- | ------- |
| [cpu](cpu.md)               | `cpu`        | Model, cores, flags, cache, NUMA         | ✅      |
| [memory](memory.md)         | `memory`     | Total, free, swap, buffers, hugepages    | ✅      |
| [disk](disk.md)             | `disk`       | Block devices, I/O stats                 | ✅      |
| [filesystem](filesystem.md) | `filesystem` | Mounts, capacity, usage, inodes          | ✅      |
| [dmi](dmi.md)               | `dmi`        | BIOS, manufacturer, serial, UUID         | ❌      |
| [gpu](gpu.md)               | `gpu`        | GPU model, driver, memory                | ❌      |
| [pci](pci.md)               | `pci`        | PCI devices                              | ❌      |
| [scsi](scsi.md)             | `scsi`       | SCSI devices                             | ❌      |
| [hardware](hardware.md)     | `hardware`   | macOS hardware profile, battery, storage | ❌      |

## 🌐 Network

| Collector             | Key       | Description                                  | Default |
| --------------------- | --------- | -------------------------------------------- | ------- |
| [network](network.md) | `network` | Interfaces, IPs, MACs, routes, DNS, counters | ✅      |

## ☁️ Cloud

| Collector                         | Key             | Description                    | Default |
| --------------------------------- | --------------- | ------------------------------ | ------- |
| [cloud](cloud.md)                 | `cloud`         | Aggregated cloud provider info | ❌      |
| [ec2](ec2.md)                     | `ec2`           | AWS EC2 metadata               | ❌      |
| [gce](gce.md)                     | `gce`           | Google Compute Engine metadata | ❌      |
| [azure](azure.md)                 | `azure`         | Azure instance metadata        | ❌      |
| [digital_ocean](digital_ocean.md) | `digital_ocean` | DigitalOcean droplet metadata  | ❌      |
| [openstack](openstack.md)         | `openstack`     | OpenStack instance metadata    | ❌      |
| [alibaba](alibaba.md)             | `alibaba`       | Alibaba Cloud ECS metadata     | ❌      |
| [rackspace](rackspace.md)         | `rackspace`     | Rackspace server metadata      | ❌      |
| [linode](linode.md)               | `linode`        | Linode instance metadata       | ❌      |
| [oci](oci.md)                     | `oci`           | Oracle Cloud metadata          | ❌      |
| [scaleway](scaleway.md)           | `scaleway`      | Scaleway instance metadata     | ❌      |
| [softlayer](softlayer.md)         | `softlayer`     | IBM SoftLayer metadata         | ❌      |
| [eucalyptus](eucalyptus.md)       | `eucalyptus`    | Eucalyptus instance metadata   | ❌      |

## 🔮 Virtualization

| Collector                           | Key              | Description                        | Default |
| ----------------------------------- | ---------------- | ---------------------------------- | ------- |
| [virtualization](virtualization.md) | `virtualization` | Hypervisor and container detection | ✅      |
| [vmware](vmware.md)                 | `vmware`         | VMware guest tools data            | ❌      |
| [virtualbox](virtualbox.md)         | `virtualbox`     | VirtualBox guest additions data    | ❌      |
| [libvirt](libvirt.md)               | `libvirt`        | Libvirt domain information         | ❌      |

## 🔒 Security

| Collector             | Key       | Description                      | Default |
| --------------------- | --------- | -------------------------------- | ------- |
| [selinux](selinux.md) | `selinux` | SELinux status, policy, contexts | ❌      |
| [ssh](ssh.md)         | `ssh`     | Host keys (RSA, ECDSA, ED25519)  | ❌      |

## 📦 Software

| Collector                     | Key           | Description                                   | Default |
| ----------------------------- | ------------- | --------------------------------------------- | ------- |
| [package_mgr](package_mgr.md) | `package_mgr` | Active package manager (apt, dnf, brew, etc.) | ✅      |
| [packages](packages.md)       | `packages`    | Installed packages                            | ❌      |
| [languages](languages.md)     | `languages`   | Go, Python, Ruby, Node, etc.                  | ❌      |
| [docker](docker.md)           | `docker`      | Containers, images, Docker info               | ❌      |
| [services](services.md)       | `services`    | Systemd service states                        | ❌      |

## 👥 Users & Sessions

| Collector               | Key        | Description                     | Default |
| ----------------------- | ---------- | ------------------------------- | ------- |
| [users](users.md)       | `users`    | passwd/group data, current user | ❌      |
| [sessions](sessions.md) | `sessions` | Logged-in sessions              | ❌      |

## 🐧 Linux-Specific

| Collector                         | Key             | Description                   | Default |
| --------------------------------- | --------------- | ----------------------------- | ------- |
| [lsb](lsb.md)                     | `lsb`           | Linux Standard Base info      | ✅      |
| [hostnamectl](hostnamectl.md)     | `hostnamectl`   | `hostnamectl` output          | ❌      |
| [sysctl](sysctl.md)               | `sysctl`        | Kernel parameters             | ❌      |
| [systemd_paths](systemd_paths.md) | `systemd_paths` | Systemd path directories      | ❌      |
| [interrupts](interrupts.md)       | `interrupts`    | IRQ stats, SMP affinity       | ❌      |
| [ipc](ipc.md)                     | `ipc`           | IPC limits and status         | ❌      |
| [livepatch](livepatch.md)         | `livepatch`     | Kernel livepatch status       | ❌      |
| [mdadm](mdadm.md)                 | `mdadm`         | Software RAID arrays          | ❌      |
| [tc](tc.md)                       | `tc`            | Traffic control info          | ❌      |
| [grub2](grub2.md)                 | `grub2`         | GRUB2 environment             | ❌      |
| [zpools](zpools.md)               | `zpools`        | ZFS pool status               | ❌      |
| [rpm](rpm.md)                     | `rpm`           | RPM macros and config         | ❌      |
| [block_device](block_device.md)   | `block_device`  | Block device sysfs attributes | ❌      |

## 🔧 Miscellaneous

| Collector             | Key       | Description                               | Default |
| --------------------- | --------- | ----------------------------------------- | ------- |
| [process](process.md) | `process` | Process list (PID, name, user, cmdline)   | ❌      |
| [load](load.md)       | `load`    | Load averages (1/5/15-minute)             | ✅      |
| [command](command.md) | `command` | Full `ps` output (Ohai command/ps parity) | ❌      |
| [sysconf](sysconf.md) | `sysconf` | POSIX sysconf values                      | ❌      |

## Collector Dependencies

Some collectors depend on data from other collectors. Dependencies are resolved
automatically — enabling a collector also enables its dependencies.

| Collector        | Depends On                      |
| ---------------- | ------------------------------- |
| `platform`       | `kernel`                        |
| `package_mgr`    | `platform`                      |
| `cloud`          | `network`, `dmi`                |
| `ec2`            | `dmi`                           |
| `gce`            | —                               |
| `azure`          | —                               |
| `digital_ocean`  | `dmi`                           |
| `openstack`      | `virtualization`                |
| `virtualization` | `dmi`, `cpu`                    |
| `docker`         | `virtualization`                |
| `packages`       | `platform`                      |
| `services`       | `platform`                      |
| `shard`          | `hostname`, `dmi`, `machine_id` |
| `interrupts`     | `cpu`                           |
| `sessions`       | —                               |
