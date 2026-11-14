# Collectors

gohai collects system facts with 62 collectors in 10 categories. You can enable
or disable each collector. This catalogue has 65 entries. The other three are
deprecated and gohai will not implement them. The tables mark them with 🪦. Each
collector gathers one category of information and returns a typed Go struct.

You enable or disable each collector with a flag in the node_exporter style:

```bash
gohai --collector.platform --no-collector.cloud
```

To enable all collectors of a **category**, use `--category` (CLI) or
`WithCategory(...)` (SDK):

```bash
gohai --category=cloud --category=hardware
```

```go
gohai.New(gohai.WithCategory("cloud"))
```

The categories are the section headings on this page: `system`, `hardware`,
`network`, `cloud`, `virtualization`, `security`, `software`, `users`, `linux`,
`misc`. gohai adds the dependencies automatically. For example, when you enable
`cloud`, gohai also enables `dmi`, because all cloud collectors depend on it.

**Defaults are opt-in.** `gohai.New()` (SDK) returns an empty registry. No
collector runs unless you ask for it. To get the recommended set, pass
`gohai.WithDefaults()`. To name each collector, pass `gohai.WithCollectors(...)`
or `gohai.WithEnabled(...)`. The CLI adds `WithDefaults()` automatically. To
turn it off, pass `--no-defaults`. Then the CLI uses only the `--collector.X`
flags that you give.

The "Default" column shows if a collector is in the recommended set:

- `✅`: the collector runs when `WithDefaults()` is in effect.
- `❌`: the collector runs only when you enable it.

The "Implemented" column shows the status of the collector:

- `✅`: implemented and tested.
- `⚠️`: partial.
- `🪦`: deprecated. gohai will not implement it, because the demand is low or the
  upstream project is archived.

**Schema.** The field names follow a naming ladder with three tiers:

1. [OCSF] (Open Cybersecurity Schema Framework) is the primary authority. It
   gives the names of 107 fields.
2. [OpenTelemetry Resource Semantic Conventions][otel-semconv] give the name
   when OCSF has no name. This applies to 91 fields.
3. A documented gohai convention gives the names of the remaining 752 fields.
   The Tier column of `schemas/field-mapping.md` gives these counts.

To see the canonical names, read [schema.ocsf.io][ocsf-schema] and the
[OpenTelemetry resource attributes][otel-semconv]. The full mapping of each
field, with citations, is in
[`schemas/field-mapping.md`](../../schemas/field-mapping.md).
[`schemas/ocsf-gaps.md`](../../schemas/ocsf-gaps.md) lists the fields that OCSF
has no name for. Each one is a candidate for a contribution to OCSF.

The collection logic follows the plugins of [Chef Ohai]. This logic decides what
to read and which special cases of each distribution to handle.

## System

| Collector                           | Key              | Description                            | Default | Implemented | Depends On        |
| ----------------------------------- | ---------------- | -------------------------------------- | ------- | ----------- | ----------------- |
| [platform](platform.md)             | `platform`       | OS name, version, family, architecture | ✅      | ✅          | None              |
| [hostname](hostname.md)             | `hostname`       | FQDN, domain, hostname, machine name   | ✅      | ✅          | None              |
| [kernel](kernel.md)                 | `kernel`         | Kernel identity (uname + Rosetta)      | ✅      | ✅          | None              |
| [kernel_modules](kernel_modules.md) | `kernel_modules` | Loaded kernel modules / kexts          | ❌      | ✅          | None              |
| [uptime](uptime.md)                 | `uptime`         | Boot time, uptime duration, idle time  | ✅      | ✅          | None              |
| [timezone](timezone.md)             | `timezone`       | System timezone                        | ✅      | ✅          | None              |
| [os_release](os_release.md)         | `os_release`     | `/etc/os-release` fields               | ✅      | ✅          | None              |
| [init](init.md)                     | `init`           | Detected init system                   | ✅      | ✅          | None              |
| [fips](fips.md)                     | `fips`           | FIPS mode status                       | ✅      | ✅          | None              |
| [machine_id](machine_id.md)         | `machine_id`     | Machine ID                             | ✅      | ✅          | None              |
| [root_group](root_group.md)         | `root_group`     | Root's primary group                   | ✅      | ✅          | None              |
| [shells](shells.md)                 | `shells`         | Available shells                       | ✅      | ✅          | None              |
| [shard](shard.md)                   | `shard`          | Deterministic shard seed               | ✅      | ✅          | `hostname`, `dmi` |

## Hardware

| Collector                   | Key          | Description                              | Default | Implemented | Depends On |
| --------------------------- | ------------ | ---------------------------------------- | ------- | ----------- | ---------- |
| [cpu](cpu.md)               | `cpu`        | Model, cores, flags, cache, NUMA         | ✅      | ✅          | None       |
| [memory](memory.md)         | `memory`     | Total, free, swap, buffers, hugepages    | ✅      | ✅          | None       |
| [disk](disk.md)             | `disk`       | Block devices, I/O stats                 | ✅      | ✅          | None       |
| [filesystem](filesystem.md) | `filesystem` | Mounts, capacity, usage, inodes          | ✅      | ✅          | None       |
| [dmi](dmi.md)               | `dmi`        | BIOS, manufacturer, serial, UUID         | ❌      | ✅          | None       |
| [gpu](gpu.md)               | `gpu`        | GPU model, vendor, cores (macOS)         | ❌      | ✅          | None       |
| [pci](pci.md)               | `pci`        | PCI devices                              | ❌      | ✅          | None       |
| [scsi](scsi.md)             | `scsi`       | SCSI devices                             | ❌      | ✅          | None       |
| [hardware](hardware.md)     | `hardware`   | macOS hardware profile, battery, storage | ❌      | ✅          | None       |

## Network

| Collector             | Key       | Description                                  | Default | Implemented | Depends On |
| --------------------- | --------- | -------------------------------------------- | ------- | ----------- | ---------- |
| [network](network.md) | `network` | Interfaces, IPs, MACs, routes, DNS, counters | ✅      | ✅          | None       |

## Cloud

| Collector                         | Key             | Description                    | Default | Implemented | Depends On |
| --------------------------------- | --------------- | ------------------------------ | ------- | ----------- | ---------- |
| [ec2](ec2.md)                     | `ec2`           | AWS EC2 metadata               | ❌      | ✅          | `dmi`      |
| [gce](gce.md)                     | `gce`           | Google Compute Engine metadata | ❌      | ✅          | `dmi`      |
| [azure](azure.md)                 | `azure`         | Azure instance metadata        | ❌      | ✅          | None       |
| [digital_ocean](digital_ocean.md) | `digital_ocean` | DigitalOcean droplet metadata  | ❌      | ✅          | `dmi`      |
| [openstack](openstack.md)         | `openstack`     | OpenStack instance metadata    | ❌      | ✅          | `dmi`      |
| [alibaba](alibaba.md)             | `alibaba`       | Alibaba Cloud ECS metadata     | ❌      | ✅          | `dmi`      |
| [linode](linode.md)               | `linode`        | Linode instance metadata       | ❌      | ✅          | `hostname` |
| [oci](oci.md)                     | `oci`           | Oracle Cloud metadata          | ❌      | ✅          | `dmi`      |
| [scaleway](scaleway.md)           | `scaleway`      | Scaleway instance metadata     | ❌      | ✅          | None       |
| rackspace                         | `rackspace`     | Rackspace server metadata      | ❌      | 🪦          | None       |
| softlayer                         | `softlayer`     | IBM SoftLayer metadata         | ❌      | 🪦          | None       |
| eucalyptus                        | `eucalyptus`    | Eucalyptus instance metadata   | ❌      | 🪦          | None       |

There is no `cloud` collector. gohai has no aggregator for all cloud providers.
[cloud.md](cloud.md) shows how to use the SDK to find the provider of a host.

## Virtualization

| Collector                           | Key              | Description                       | Default | Implemented | Depends On |
| ----------------------------------- | ---------------- | --------------------------------- | ------- | ----------- | ---------- |
| [virtualization](virtualization.md) | `virtualization` | Detected hypervisor and container | ✅      | ✅          | `cpu`      |
| [vmware](vmware.md)                 | `vmware`         | VMware guest tools data           | ❌      | ✅          | None       |
| [virtualbox](virtualbox.md)         | `virtualbox`     | VirtualBox guest additions data   | ❌      | ✅          | None       |
| [libvirt](libvirt.md)               | `libvirt`        | Libvirt domain information        | ❌      | ✅          | None       |

## Security

| Collector             | Key       | Description                      | Default | Implemented | Depends On |
| --------------------- | --------- | -------------------------------- | ------- | ----------- | ---------- |
| [selinux](selinux.md) | `selinux` | SELinux status, policy, contexts | ❌      | ✅          | None       |
| [ssh](ssh.md)         | `ssh`     | Host keys (RSA, ECDSA, ED25519)  | ❌      | ✅          | None       |

## Software

| Collector                     | Key           | Description                                         | Default | Implemented | Depends On |
| ----------------------------- | ------------- | --------------------------------------------------- | ------- | ----------- | ---------- |
| [package_mgr](package_mgr.md) | `package_mgr` | Active package manager (for example apt, dnf, brew) | ✅      | ✅          | None       |
| [packages](packages.md)       | `packages`    | Installed packages                                  | ❌      | ✅          | None       |
| [languages](languages.md)     | `languages`   | Go, Python, Ruby, Node and more                     | ❌      | ✅          | None       |
| [docker](docker.md)           | `docker`      | Containers, images, Docker info                     | ❌      | ✅          | None       |
| [services](services.md)       | `services`    | Systemd service states                              | ❌      | ✅          | None       |

## Users & sessions

| Collector               | Key        | Description                     | Default | Implemented | Depends On |
| ----------------------- | ---------- | ------------------------------- | ------- | ----------- | ---------- |
| [users](users.md)       | `users`    | passwd/group data, current user | ❌      | ✅          | None       |
| [sessions](sessions.md) | `sessions` | Logged-in sessions              | ❌      | ✅          | None       |

## Linux-specific

| Collector                         | Key             | Description                   | Default | Implemented | Depends On |
| --------------------------------- | --------------- | ----------------------------- | ------- | ----------- | ---------- |
| [lsb](lsb.md)                     | `lsb`           | Linux Standard Base info      | ✅      | ✅          | None       |
| [hostnamectl](hostnamectl.md)     | `hostnamectl`   | `hostnamectl` output          | ❌      | ✅          | None       |
| [sysctl](sysctl.md)               | `sysctl`        | Kernel parameters             | ❌      | ✅          | None       |
| [systemd_paths](systemd_paths.md) | `systemd_paths` | Systemd path directories      | ❌      | ✅          | None       |
| [interrupts](interrupts.md)       | `interrupts`    | IRQ stats, SMP affinity       | ❌      | ✅          | None       |
| [ipc](ipc.md)                     | `ipc`           | IPC limits and status         | ❌      | ✅          | None       |
| [livepatch](livepatch.md)         | `livepatch`     | Kernel livepatch status       | ❌      | ✅          | None       |
| [mdadm](mdadm.md)                 | `mdadm`         | Software RAID arrays          | ❌      | ✅          | None       |
| [tc](tc.md)                       | `tc`            | Traffic control info          | ❌      | ✅          | None       |
| [grub2](grub2.md)                 | `grub2`         | GRUB2 environment             | ❌      | ✅          | None       |
| [zpools](zpools.md)               | `zpools`        | ZFS pool status               | ❌      | ✅          | None       |
| [rpm](rpm.md)                     | `rpm`           | RPM macros and config         | ❌      | ✅          | None       |
| [block_device](block_device.md)   | `block_device`  | Block device sysfs attributes | ❌      | ✅          | None       |

## Miscellaneous

| Collector             | Key       | Description                               | Default | Implemented | Depends On |
| --------------------- | --------- | ----------------------------------------- | ------- | ----------- | ---------- |
| [process](process.md) | `process` | Process list (PID, name, user, cmdline)   | ❌      | ✅          | None       |
| [load](load.md)       | `load`    | Load averages (1/5/15-minute)             | ✅      | ✅          | None       |
| [command](command.md) | `command` | Full `ps` output (Ohai command/ps parity) | ❌      | ✅          | None       |
| [sysconf](sysconf.md) | `sysconf` | POSIX sysconf values                      | ❌      | ✅          | None       |

## Collector dependencies

The `Depends On` column of each category table lists the collectors that each
entry depends on. gohai resolves the dependencies automatically. When you enable
a collector, gohai also enables its dependencies.

[chef ohai]: https://github.com/chef/ohai
[ocsf]: https://ocsf.io/
[ocsf-schema]: https://schema.ocsf.io/
[otel-semconv]: https://opentelemetry.io/docs/specs/semconv/resource/
