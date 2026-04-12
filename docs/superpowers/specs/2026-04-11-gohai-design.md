# gohai — System Fact Collector Design Spec

## Overview

gohai is a Go-based system information collector inspired by Chef Ohai. It
collects comprehensive system facts via a pluggable collector architecture
with node_exporter-style flag toggling. It is both a standalone CLI tool and
a Go library/SDK for integration with OSAPI and other consumers.

## Goals

- **Full Ohai parity**: 65 collectors across 9 categories covering system,
  hardware, network, cloud, virtualization, security, software, users, and
  Linux-specific facts.
- **Dual-use**: standalone `gohai` CLI binary AND importable `pkg/gohai` SDK
  with strongly-typed Go structs.
- **node_exporter-style flags**: `--collector.<name>` / `--no-collector.<name>`
  to toggle individual collectors.
- **Cross-platform**: Linux as primary target, macOS best-effort.
- **100% test coverage**: all packages must reach 100% coverage with
  testify/suite and table-driven patterns.
- **OSAPI conventions**: follows osapi-orchestrator code standards (multi-line
  signatures, public/internal test split, golangci-lint, conventional commits).

## Architecture

### Package Structure

```
cmd/gohai/                     # CLI entrypoint (Cobra)
  main.go                      # Cobra root command, flag registration
pkg/gohai/                     # Public SDK
  gohai.go                     # Gohai struct, New(), Collect()
  facts.go                     # Facts struct with typed fields for all collectors
  options.go                   # Functional options (WithCollectors, etc.)
  registry.go                  # Public registry API (collector enable/disable)
internal/collector/            # Collector interface + implementations
  collector.go                 # Collector interface, Tier type
  registry.go                  # Internal registry (register, resolve deps, run)
  platform/                    # One sub-package per collector
    platform.go                # Info struct + Collector implementation
    platform_linux.go          # Linux-specific collection (build-tagged)
    platform_darwin.go         # macOS-specific collection (build-tagged)
    platform_test.go           # Internal tests
    platform_public_test.go    # Public interface tests
  ...                          # Same pattern for all 65 collectors
```

### Collector Interface

Every collector implements a common interface:

```go
type Collector interface {
    Name() string
    Tier() Tier
    Dependencies() []string
    Collect(ctx context.Context) (any, error)
}
```

- `Name()` returns the unique key (e.g., `"platform"`, `"cpu"`)
- `Tier()` returns TierCore, TierExtended, or TierOptIn
- `Dependencies()` returns names of collectors that must run first
- `Collect()` gathers facts and returns a strongly-typed struct

### Registry

The internal registry handles:

1. **Registration**: each collector package calls `Register()` in an `init()`
   function
2. **Enable/Disable**: node_exporter-style — TierCore and TierExtended are
   enabled by default, TierOptIn is disabled by default
3. **Dependency Resolution**: topological sort ensures collectors run after
   their dependencies
4. **Concurrent Execution**: collectors at the same dependency level run in
   parallel

### Facts Struct

The public `Facts` struct in `pkg/gohai/facts.go` has a typed field for every
collector:

```go
type Facts struct {
    Platform       *platform.Info       `json:"platform,omitempty"`
    CPU            *cpu.Info            `json:"cpu,omitempty"`
    Memory         *memory.Info         `json:"memory,omitempty"`
    // ... one field per collector
    CollectTime    time.Time            `json:"collect_time"`
    CollectDuration time.Duration       `json:"collect_duration"`
}

func (f *Facts) JSON() ([]byte, error)        // nested JSON
func (f *Facts) PrettyJSON() ([]byte, error)  // indented JSON
func (f *Facts) Flat() map[string]any         // dot-separated keys
```

### CLI

Cobra-based with node_exporter-style flags:

```
gohai                                    # all default collectors
gohai --collector.docker                 # enable opt-in collector
gohai --no-collector.cloud               # disable default collector
gohai --collector.platform --collector.cpu  # only these two
gohai --pretty                           # pretty-print JSON
gohai --flat                             # flat key=value output
gohai --list-collectors                  # list all collectors and status
```

Flags are dynamically registered from the collector registry.

### Platform-Specific Code

Each collector uses Go build tags for platform-specific implementations:

- `platform_linux.go` — `//go:build linux`
- `platform_darwin.go` — `//go:build darwin`

If a collector has no implementation for the current OS, it returns `nil`
without error (fact is omitted from output).

## Collector Catalog

### 🖥️ System (12 collectors)

| Collector    | Key            | Ohai Equivalent          | Description                                |
| ------------ | -------------- | ------------------------ | ------------------------------------------ |
| platform     | `platform`     | `platform.rb`            | OS name, version, family, arch             |
| hostname     | `hostname`     | `hostname.rb`            | FQDN, domain, hostname, machinename        |
| kernel       | `kernel`       | `kernel.rb`              | Version, modules, parameters               |
| uptime       | `uptime`       | `uptime.rb`              | Boot time, duration, idle time             |
| timezone     | `timezone`     | `timezone.rb`            | System timezone abbreviation               |
| os_release   | `os_release`   | `linux/os_release.rb`    | `/etc/os-release` fields                   |
| init         | `init`         | `init_package.rb`        | Init system (systemd, init)                |
| fips         | `fips`         | `fips.rb`                | FIPS kernel mode                           |
| machine_id   | `machine_id`   | `linux/machineid.rb`     | `/etc/machine-id`                          |
| root_group   | `root_group`   | `root_group.rb`          | Root's primary group                       |
| shells       | `shells`       | `shells.rb`              | Available shells                           |
| shard        | `shard`        | `shard.rb`               | Deterministic shard seed                   |

### ⚙️ Hardware (9 collectors)

| Collector    | Key            | Ohai Equivalent          | Description                                |
| ------------ | -------------- | ------------------------ | ------------------------------------------ |
| cpu          | `cpu`          | `cpu.rb`                 | Model, cores, flags, cache, NUMA           |
| memory       | `memory`       | `linux/memory.rb`        | Total, free, swap, buffers, hugepages      |
| disk         | `disk`         | `linux/block_device.rb`  | Block devices, I/O stats                   |
| filesystem   | `filesystem`   | `filesystem.rb`          | Mounts, capacity, inodes, fs type          |
| dmi          | `dmi`          | `dmi.rb`                 | BIOS, manufacturer, serial, UUID           |
| gpu          | `gpu`          | —                        | GPU model, driver, memory                  |
| pci          | `pci`          | `linux/lspci.rb`         | PCI devices                                |
| scsi         | `scsi`         | `scsi.rb`                | SCSI devices                               |
| hardware     | `hardware`     | `darwin/hardware.rb`     | macOS hardware profile, battery, storage   |

### 🌐 Network (1 collector)

| Collector    | Key            | Ohai Equivalent          | Description                                |
| ------------ | -------------- | ------------------------ | ------------------------------------------ |
| network      | `network`      | `network.rb` + platform  | Interfaces, IPs, MACs, routes, DNS         |

### ☁️ Cloud (13 collectors)

| Collector    | Key              | Ohai Equivalent        | Description                              |
| ------------ | ---------------- | ---------------------- | ---------------------------------------- |
| cloud        | `cloud`          | `cloud.rb`             | Aggregated cloud provider info           |
| ec2          | `ec2`            | `ec2.rb`               | AWS EC2 IMDS metadata                    |
| gce          | `gce`            | `gce.rb`               | Google Compute Engine metadata           |
| azure        | `azure`          | `azure.rb`             | Azure IMDS metadata                      |
| digital_ocean| `digital_ocean`  | `digital_ocean.rb`     | DigitalOcean droplet metadata            |
| openstack    | `openstack`      | `openstack.rb`         | OpenStack instance metadata              |
| alibaba      | `alibaba`        | `alibaba.rb`           | Alibaba Cloud ECS metadata              |
| rackspace    | `rackspace`      | `rackspace.rb`         | Rackspace server metadata                |
| linode       | `linode`         | `linode.rb`            | Linode instance metadata                 |
| oci          | `oci`            | `oci.rb`               | Oracle Cloud metadata                    |
| scaleway     | `scaleway`       | `scaleway.rb`          | Scaleway instance metadata               |
| softlayer    | `softlayer`      | `softlayer.rb`         | IBM SoftLayer metadata                   |
| eucalyptus   | `eucalyptus`     | `eucalyptus.rb`        | Eucalyptus instance metadata             |

### 🔮 Virtualization (4 collectors)

| Collector       | Key              | Ohai Equivalent              | Description                        |
| --------------- | ---------------- | ---------------------------- | ---------------------------------- |
| virtualization  | `virtualization` | `linux/virtualization.rb`    | Hypervisor and container detection |
| vmware          | `vmware`         | `vmware.rb`                  | VMware guest tools data            |
| virtualbox      | `virtualbox`     | `virtualbox.rb`              | VirtualBox guest additions data    |
| libvirt         | `libvirt`        | `libvirt.rb`                 | Libvirt domain info                |

### 🔒 Security (2 collectors)

| Collector    | Key        | Ohai Equivalent          | Description                              |
| ------------ | ---------- | ------------------------ | ---------------------------------------- |
| selinux      | `selinux`  | `linux/selinux.rb`       | SELinux status, policy, contexts         |
| ssh          | `ssh`      | `ssh_host_key.rb`        | Host keys (RSA, ECDSA, ED25519)          |

### 📦 Software (4 collectors)

| Collector    | Key         | Ohai Equivalent          | Description                              |
| ------------ | ----------- | ------------------------ | ---------------------------------------- |
| packages     | `packages`  | `packages.rb`            | Installed packages                       |
| languages    | `languages` | `languages.rb` + subs    | Go, Python, Ruby, Node, Java, etc.       |
| docker       | `docker`    | `docker.rb`              | Containers, images, Docker info          |
| services     | `services`  | —                        | Systemd service states                   |

### 👥 Users & Sessions (2 collectors)

| Collector    | Key        | Ohai Equivalent          | Description                              |
| ------------ | ---------- | ------------------------ | ---------------------------------------- |
| users        | `users`    | `passwd.rb`              | passwd/group data, current user          |
| sessions     | `sessions` | `linux/sessions.rb`      | Logged-in sessions                       |

### 🐧 Linux-Specific (13 collectors)

| Collector      | Key              | Ohai Equivalent            | Description                          |
| -------------- | ---------------- | -------------------------- | ------------------------------------ |
| lsb            | `lsb`            | `linux/lsb.rb`             | Linux Standard Base info             |
| hostnamectl    | `hostnamectl`    | `linux/hostnamectl.rb`     | `hostnamectl` output                 |
| sysctl         | `sysctl`         | `linux/sysctl.rb`          | Kernel parameters                    |
| systemd_paths  | `systemd_paths`  | `linux/systemd_paths.rb`   | Systemd path directories             |
| interrupts     | `interrupts`     | `linux/interrupts.rb`      | IRQ stats, SMP affinity              |
| ipc            | `ipc`            | `linux/ipc.rb`             | IPC limits and status                |
| livepatch      | `livepatch`      | `linux/livepatch.rb`       | Kernel livepatch status              |
| mdadm          | `mdadm`          | `linux/mdadm.rb`           | Software RAID arrays                 |
| tc             | `tc`             | `linux/tc.rb`              | Traffic control info                 |
| grub2          | `grub2`          | `grub2.rb`                 | GRUB2 environment                    |
| zpools         | `zpools`         | `zpools.rb`                | ZFS pool status                      |
| rpm            | `rpm`            | `rpm.rb`                   | RPM macros and config                |
| block_device   | `block_device`   | `linux/block_device.rb`    | Block device sysfs attributes        |

### 🔧 Miscellaneous (2 collectors)

| Collector    | Key        | Ohai Equivalent          | Description                              |
| ------------ | ---------- | ------------------------ | ---------------------------------------- |
| command      | `command`  | `command.rb` + `ps.rb`   | Process snapshot                         |
| sysconf      | `sysconf`  | `sysconf.rb`             | POSIX sysconf values                     |

## Implementation Strategy

### Phase 1: Foundation + 3 Representative Collectors

Build the full framework and implement one collector from each priority tier
to validate the architecture end-to-end before filling in the rest:

1. **Collector interface and registry** — `internal/collector/`
2. **Public SDK** — `pkg/gohai/` with `New()`, `Collect()`, `Facts`
3. **CLI** — `cmd/gohai/` with Cobra and dynamic flag registration
4. **platform collector** (Tier 1 representative) — simplest, exercises OS
   detection, validates basic collector contract
5. **virtualization collector** (Tier 2 representative) — moderate complexity,
   detection logic, optional data patterns
6. **docker collector** (Tier 3 representative) — external runtime, nested
   output, opt-in disabled-by-default pattern

### Phase 2–N: Remaining Collectors

After Phase 1 validates the architecture, implement remaining collectors
grouped by category. Each collector follows the same pattern established
in Phase 1.

## Testing Strategy

- **100% test coverage** on all packages
- **Public tests**: `*_public_test.go` in test package — verify exported API
- **Internal tests**: `*_test.go` in same package — verify parsing, edge cases
- **testify/suite** with table-driven patterns
- **One suite method per function** — all scenarios as table rows
- **Platform-specific tests**: build-tagged test files matching implementations
- **Mocking**: collectors that read from `/proc`, `/sys`, or run commands
  should accept an abstraction (e.g., filesystem interface or command runner)
  that can be stubbed in tests

## Dependencies

Decision on specific libraries (gopsutil, go-sysinfo, node_exporter, or
direct syscall/procfs) deferred to implementation. The collector interface
abstracts this — each collector can use whatever approach works best for
its specific domain.

## Output Format

### Nested JSON (default CLI output)

```json
{
  "platform": {
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "architecture": "amd64"
  },
  "cpu": {
    "total": 8,
    "cores": 4,
    "model_name": "Intel Core i7-10700K"
  },
  "collect_time": "2026-04-11T10:30:00Z",
  "collect_duration": "1.234s"
}
```

### Flat map (SDK + `--flat` flag)

```
platform.name=ubuntu
platform.version=24.04
platform.family=debian
cpu.total=8
cpu.cores=4
```

### Typed Go structs (SDK)

```go
facts.Platform.Name      // "ubuntu"
facts.CPU.Total          // 8
facts.Memory.Total       // "32GB"
```
