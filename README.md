[![release](https://img.shields.io/github/release/osapi-io/gohai.svg?style=for-the-badge)](https://github.com/osapi-io/gohai/releases/latest)
[![codecov](https://img.shields.io/codecov/c/github/osapi-io/gohai?style=for-the-badge)](https://codecov.io/gh/osapi-io/gohai)
[![go report card](https://goreportcard.com/badge/github.com/osapi-io/gohai?style=for-the-badge)](https://goreportcard.com/report/github.com/osapi-io/gohai)
[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](LICENSE)
[![build](https://img.shields.io/github/actions/workflow/status/osapi-io/gohai/go.yml?style=for-the-badge)](https://github.com/osapi-io/gohai/actions/workflows/go.yml)
[![powered by](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge)](https://github.com/goreleaser)
[![conventional commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
[![built with just](https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white)](https://just.systems)
![gitHub commit activity](https://img.shields.io/github/commit-activity/m/osapi-io/gohai?style=for-the-badge)
[![go reference](https://img.shields.io/badge/go-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://pkg.go.dev/github.com/osapi-io/gohai/pkg/gohai)

# gohai

**gohai is an SDK-first Go library** for collecting comprehensive system
facts, inspired by [Chef Ohai][]. Import it into your Go application for
typed access to system facts — or use the standalone `gohai` CLI, a thin
wrapper over the same SDK.

Each collector wraps a well-maintained backing source ([gopsutil][],
[ghw][], [procfs][], cloud SDKs) and reshapes its output into typed Go
structs. gohai's value is the unified API, typed structs, and pluggable
collector model — not reimplementing `/proc` parsing from scratch.

### Schema: OCSF + OpenTelemetry + Ohai

Fact naming and structure follow, in order of precedence:

1. **[OCSF][]** (Open Cybersecurity Schema Framework) — the primary
   schema. Backed by AWS and Splunk for asset, observability, and
   security data. Aligning means gohai output feeds SIEMs, data lakes,
   and inventory tools without translation. Browse
   [schema.ocsf.io][ocsf-schema] to see field names and object shapes.
2. **[OpenTelemetry Resource Semantic Conventions][otel-semconv]** —
   used when OCSF is silent. Widely adopted for observability
   telemetry; covers areas OCSF hasn't (per-CPU vendor/family/model,
   system load averages, process runtime, host uptime).

What we collect (which facts, which distro edge cases, which fallback
sources) draws on [Chef Ohai][]'s years of accumulated plugin logic.
What we call each field draws on OCSF + OpenTelemetry. We do **not**
pursue Ohai JSON shape parity — Ruby Mash ↔ Go struct translation
isn't worth pinning byte-for-byte.

### Primary consumer

gohai is built to be embedded in [OSAPI][] and other Go services that need
typed system facts for routing, guards, discovery, inventory, and
compliance. The CLI is a convenience — the SDK is the product.

## 📦 Install

```bash
go install github.com/osapi-io/gohai/cmd/gohai@latest
```

As a library dependency:

```bash
go get github.com/osapi-io/gohai
```

## ✨ Features

| Feature                                                       | Description                                    |
| ------------------------------------------------------------- | ---------------------------------------------- |
| [🔌 Pluggable Collectors](docs/features/collectors.md)        | Enable/disable individual fact collectors      |
| [🏗️ Typed Structs](docs/features/typed-structs.md)            | Strongly-typed Go structs for all facts        |
| [📄 JSON Output](docs/features/json-output.md)                | Nested JSON output for CLI and programmatic use |
| [🗺️ Flat Map Access](docs/features/flat-map.md)               | Dot-separated key-value access                 |
| [🐧 Cross-Platform](docs/features/cross-platform.md)          | Linux primary, macOS best-effort               |
| [🔗 Collector Dependencies](docs/features/dependencies.md)    | Automatic dependency resolution between facts  |
| [⚡ Concurrent Collection](docs/features/concurrency.md)      | Collectors run concurrently; dependency graph resolves order when any collector declares deps. |
| [🎛️ Profiles](docs/features/profiles.md)                      | Predefined collector sets (minimal, standard, full) |
| [📊 OCSF + OpenTelemetry + Ohai](docs/features/ocsf-ohai.md)   | Field names follow [OCSF](https://schema.ocsf.io/) then [OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/resource/); data sources mirror Chef Ohai's plugins |
| [🔌 SDK Integration](docs/features/sdk.md)                    | Import as a Go package for OSAPI and others    |

## 🔌 Collectors

65 collectors across 9 categories. Collectors are individually toggled using
node_exporter-style flags — `--collector.<name>` to opt in,
`--no-collector.<name>` to opt out. SDK consumers use
`gohai.WithEnabled(...)` / `gohai.WithDisabled(...)` / `gohai.WithCollectors(...)`.

**Defaults are opt-in.** `gohai.New()` returns an empty registry. Pass
`gohai.WithDefaults()` for the recommended set (cheap + near-universal —
identity, base hardware, network, load, virt detect). The CLI wires
`WithDefaults()` automatically; pass `--no-defaults` to skip it and use only
explicit `--collector.X` flags. The "Default" column reflects membership in the
recommended set.

Implementation status legend: ✅ = implemented and tested, ⚠️ = partial
support (e.g., Linux only), 🚧 = planned but not yet built.

### 🖥️ System

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [platform](docs/collectors/platform.md)            | OS name, version, family, architecture         | ✅      | ✅ |
| [hostname](docs/collectors/hostname.md)            | FQDN, domain, hostname, machine name           | ✅      | ✅ |
| [kernel](docs/collectors/kernel.md)                | Version, modules, parameters                   | ✅      | ✅ |
| [uptime](docs/collectors/uptime.md)                | Boot time, uptime duration, idle time          | ✅      | ✅ |
| [timezone](docs/collectors/timezone.md)            | System timezone                                | ✅      | ✅ |
| [os_release](docs/collectors/os_release.md)        | `/etc/os-release` fields (Linux)               | ✅      | ✅         |
| [init](docs/collectors/init.md)                    | Init system detection (systemd, init, etc.)    | ✅      | ✅         |
| [fips](docs/collectors/fips.md)                    | FIPS mode detection                            | ✅      | ✅ |
| [machine_id](docs/collectors/machine_id.md)        | Machine ID (`/etc/machine-id`)                 | ✅      | ✅ |
| [root_group](docs/collectors/root_group.md)        | Root user's primary group                      | ✅      | ✅ |
| [shells](docs/collectors/shells.md)                | Available shells from `/etc/shells`            | ✅      | ✅ |
| [shard](docs/collectors/shard.md)                  | Deterministic shard seed from machine identity | ✅      | ✅         |

### ⚙️ Hardware

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [cpu](docs/collectors/cpu.md)                      | Model, cores, flags, cache, NUMA topology      | ✅      | ✅ |
| [memory](docs/collectors/memory.md)                | Total, free, swap, buffers, cached, hugepages  | ✅      | 🚧          |
| [disk](docs/collectors/disk.md)                    | Block devices, I/O stats                       | ✅      | ✅ |
| [filesystem](docs/collectors/filesystem.md)        | Mounts, capacity, usage, inodes, fs type       | ✅      | ✅ |
| [dmi](docs/collectors/dmi.md)                      | BIOS, system manufacturer, serial, UUID        | ❌      | 🚧          |
| [gpu](docs/collectors/gpu.md)                      | GPU model, driver, memory                      | ❌      | 🚧          |
| [pci](docs/collectors/pci.md)                      | PCI devices (`lspci`)                          | ❌      | 🚧          |
| [scsi](docs/collectors/scsi.md)                    | SCSI devices                                   | ❌      | 🚧          |
| [hardware](docs/collectors/hardware.md)            | macOS hardware profile, battery, storage       | ❌      | 🚧          |

### 🌐 Network

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [network](docs/collectors/network.md)              | Interfaces, IPs, MACs, routes, neighbours, link details, counters | ✅      | ✅         |
| [dns](docs/collectors/dns.md)                      | `/etc/resolv.conf` nameservers + search        | ✅      | 🚧          |
| [ethtool](docs/collectors/ethtool.md)              | Per-NIC ethtool detail (link/duplex/driver)    | ❌      | 🚧          |

### ☁️ Cloud

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [cloud](docs/collectors/cloud.md)                  | Aggregated cloud provider info                 | ❌      | 🚧          |
| [ec2](docs/collectors/ec2.md)                      | AWS EC2 instance metadata (IMDS)               | ❌      | 🚧          |
| [gce](docs/collectors/gce.md)                      | Google Compute Engine metadata                 | ❌      | 🚧          |
| [azure](docs/collectors/azure.md)                  | Azure instance metadata (IMDS)                 | ❌      | 🚧          |
| [digital_ocean](docs/collectors/digital_ocean.md)  | DigitalOcean droplet metadata                  | ❌      | 🚧          |
| [openstack](docs/collectors/openstack.md)          | OpenStack instance metadata                    | ❌      | 🚧          |
| [alibaba](docs/collectors/alibaba.md)              | Alibaba Cloud ECS metadata                     | ❌      | 🚧          |
| [rackspace](docs/collectors/rackspace.md)          | Rackspace server metadata                      | ❌      | 🚧          |
| [linode](docs/collectors/linode.md)                | Linode instance metadata                       | ❌      | 🚧          |
| [oci](docs/collectors/oci.md)                      | Oracle Cloud instance metadata                 | ❌      | 🚧          |
| [scaleway](docs/collectors/scaleway.md)            | Scaleway instance metadata                     | ❌      | 🚧          |
| [softlayer](docs/collectors/softlayer.md)          | IBM SoftLayer metadata                         | ❌      | 🚧          |
| [eucalyptus](docs/collectors/eucalyptus.md)        | Eucalyptus instance metadata                   | ❌      | 🚧          |

### 🔮 Virtualization

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [virtualization](docs/collectors/virtualization.md) | Hypervisor and container runtime detection     | ✅      | ✅ |
| [vmware](docs/collectors/vmware.md)                | VMware guest tools data                        | ❌      | 🚧          |
| [virtualbox](docs/collectors/virtualbox.md)        | VirtualBox guest additions data                | ❌      | 🚧          |
| [libvirt](docs/collectors/libvirt.md)              | Libvirt domain information                     | ❌      | 🚧          |

### 🔒 Security

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [selinux](docs/collectors/selinux.md)              | SELinux status, policy, contexts               | ❌      | 🚧          |
| [ssh](docs/collectors/ssh.md)                      | Host keys (RSA, ECDSA, ED25519)                | ❌      | 🚧          |

### 📦 Software

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [package_mgr](docs/collectors/package_mgr.md)      | Active package manager (apt, dnf, brew, etc.)  | ✅      | ✅         |
| [packages](docs/collectors/packages.md)            | Installed packages (apt, yum, brew, etc.)      | ❌      | 🚧          |
| [languages](docs/collectors/languages.md)          | Go, Python, Ruby, Node, Rust, Java, etc.       | ❌      | 🚧          |
| [docker](docs/collectors/docker.md)                | Running containers, images, Docker info        | ❌      | 🚧          |
| [services](docs/collectors/services.md)            | Systemd service states                         | ❌      | 🚧          |

### 👥 Users & Sessions

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [users](docs/collectors/users.md)                  | passwd/group data, current user                | ❌      | ⚠️         |
| [sessions](docs/collectors/sessions.md)            | Logged-in sessions (`loginctl`)                | ❌      | 🚧          |

### 🐧 Linux-Specific

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [lsb](docs/collectors/lsb.md)                     | Linux Standard Base release info               | ✅      | ✅         |
| [hostnamectl](docs/collectors/hostnamectl.md)      | `hostnamectl` output                           | ❌      | 🚧          |
| [sysctl](docs/collectors/sysctl.md)               | Kernel parameters via `sysctl`                 | ❌      | 🚧          |
| [systemd_paths](docs/collectors/systemd_paths.md)  | Systemd path directories                       | ❌      | 🚧          |
| [interrupts](docs/collectors/interrupts.md)        | IRQ stats, SMP affinity                        | ❌      | 🚧          |
| [ipc](docs/collectors/ipc.md)                      | IPC limits and status                          | ❌      | 🚧          |
| [livepatch](docs/collectors/livepatch.md)          | Kernel livepatch status                        | ❌      | 🚧          |
| [mdadm](docs/collectors/mdadm.md)                  | Software RAID arrays                           | ❌      | 🚧          |
| [tc](docs/collectors/tc.md)                        | Traffic control (qdisc) info                   | ❌      | 🚧          |
| [grub2](docs/collectors/grub2.md)                  | GRUB2 environment variables                    | ❌      | 🚧          |
| [zpools](docs/collectors/zpools.md)                | ZFS pool status                                | ❌      | 🚧          |
| [rpm](docs/collectors/rpm.md)                      | RPM macros and config                          | ❌      | 🚧          |
| [block_device](docs/collectors/block_device.md)    | Block device attributes from sysfs             | ❌      | 🚧          |

### 🔧 Miscellaneous

| Collector                                          | Description                                    | Default | Implemented |
| -------------------------------------------------- | ---------------------------------------------- | ------- | ----------- |
| [process](docs/collectors/process.md)              | Process list (PID, name, user, cmdline)        | ❌      | ✅ |
| [load](docs/collectors/load.md)                    | Load averages (1/5/15-minute)                  | ✅      | ✅         |
| [command](docs/collectors/command.md)              | Full `ps` output (Ohai command/ps parity)      | ❌      | 🚧          |
| [sysconf](docs/collectors/sysconf.md)             | POSIX sysconf values                           | ❌      | 🚧          |

## 🎯 Usage

### CLI

```bash
# Collect all default facts
gohai

# Enable specific collectors only
gohai --collector.platform --collector.cpu --collector.memory

# Disable specific collectors
gohai --no-collector.cloud --no-collector.gpu

# Output as pretty-printed JSON
gohai --pretty

# Output flat key-value pairs
gohai --flat
```

### SDK

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
    g, err := gohai.New(
        gohai.WithCollectors("platform", "cpu", "memory"),
    )
    if err != nil {
        log.Fatal(err)
    }

    facts, err := g.Collect(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Access typed structs
    fmt.Printf("OS: %s %s\n", facts.Platform.Name, facts.Platform.Version)
    fmt.Printf("CPUs: %d\n", facts.CPU.Total)
    fmt.Printf("Memory: %s\n", facts.Memory.Total)

    // Or get as JSON
    json, _ := facts.JSON()
    fmt.Println(string(json))

    // Or get as flat map
    flat := facts.Flat()
    fmt.Println(flat["cpu.total"])
}
```


## 🔗 Integrations

Primary consumers of the gohai SDK:

| Integration                                       | What it consumes                                               |
| ------------------------------------------------- | -------------------------------------------------------------- |
| [OSAPI](docs/integrations/osapi.md)               | Maps `job.FactsRegistration` → gohai collector fields          |

The OSAPI integration doc is the authoritative mapping of OSAPI fact fields
to gohai collectors — keep it in sync with any `Info` struct changes.

## 📖 Documentation

- [Collectors reference](docs/collectors/README.md) — one doc per collector
  with fields, schema mappings (OCSF + OpenTelemetry), and Ohai source
  alignment.
- [Features](docs/features/README.md) — SDK surface, concurrency model,
  dependency resolution, OCSF + OpenTelemetry + Ohai schema, profiles.
- [Integrations](docs/integrations/osapi.md) — how downstream services consume
  the SDK.
- [Development](docs/development.md) — prerequisites, setup, testing, commit
  conventions.
- [Contributing](docs/contributing.md) — PR workflow.
- [Package documentation][] on pkg.go.dev — generated API reference.

## 🤝 Contributing

See the [Development](docs/development.md) guide for prerequisites, setup,
and conventions. See the [Contributing](docs/contributing.md) guide before
submitting a PR.

## 📄 License

The [MIT][] License.

[Chef Ohai]: https://docs.chef.io/ohai/
[OSAPI]: https://github.com/osapi-io/osapi
[gopsutil]: https://github.com/shirou/gopsutil
[ghw]: https://github.com/jaypipes/ghw
[procfs]: https://github.com/prometheus/procfs
[OCSF]: https://ocsf.io/
[ocsf-schema]: https://schema.ocsf.io/
[otel-semconv]: https://opentelemetry.io/docs/specs/semconv/resource/
[package documentation]: https://pkg.go.dev/github.com/osapi-io/gohai/pkg/gohai
[MIT]: LICENSE
