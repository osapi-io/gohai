# gohai — System Fact Collector Design Spec

## Overview

**gohai is an SDK first.** It is a Go library that OSAPI and other consumers
import to collect typed system facts, with a standalone CLI (`gohai`) shipped
as a thin wrapper over the same SDK. The SDK surface — not the CLI — is the
primary product.

gohai is inspired by Chef Ohai: pluggable collectors, comprehensive facts,
Ohai-compatible JSON output. Unlike Ohai (Ruby) or Facter (Ruby), gohai is a
Go-native library designed to be embedded in Go services.

### Primary consumers

- **OSAPI** — imports `pkg/gohai` to enable fact-based routing, guards,
  discovery, and inventory features.
- **`gohai` CLI** — ad-hoc human use; thin wrapper around the SDK.
- **Other Go services** — exporters, inventory tools, compliance scanners,
  agents that need typed system facts.

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
- **Leverage existing libraries**: wrap battle-tested Go system libraries
  rather than reimplement OS parsing from scratch. gohai's value is the
  unified API, Ohai-compatible output, and pluggable collector model — not
  re-solving `/proc` parsing.

## Library Strategy

Each collector chooses the backing library that best fits its domain. The
Collector interface is the abstraction boundary — consumers of the SDK never
see the backing library. The public `Info` structs are shaped to match Ohai's
output format, not the backing library's data model.

### Recommended libraries

| Library                                                  | Best for                                                 | License  |
| -------------------------------------------------------- | -------------------------------------------------------- | -------- |
| [gopsutil](https://github.com/shirou/gopsutil)           | CPU, memory, disk, filesystem, network, process, uptime, users, host, virtualization detection | BSD-3    |
| [ghw](https://github.com/jaypipes/ghw)                   | GPU, PCI, SCSI, block devices, DMI (BIOS/baseboard/chassis), NUMA topology | Apache-2 |
| [procfs](https://github.com/prometheus/procfs)           | Raw `/proc` and `/sys` parsing for Linux-specific facts (interrupts, ipc, mdadm, sysctl, block_device, etc.) | Apache-2 |
| [go-sysinfo](https://github.com/elastic/go-sysinfo)      | Alternative host info; rich per-process detail           | Apache-2 |
| [node_exporter source](https://github.com/prometheus/node_exporter) | Reference for tricky Linux parsing patterns — read, learn, rewrite in our style (don't import code directly) | Apache-2 |
| Cloud provider SDKs or `net/http` to IMDS endpoints      | ec2, gce, azure, digital_ocean, openstack, etc.          | varies   |
| Port Ohai's Ruby plugins                                 | Fallback when no Go library covers the domain            | Apache-2 (reference) |
| Roll our own thin parsers                                | selinux, packages (apt/yum/brew), services (systemctl), docker (docker info), languages (runtime --version probes), lsb | —        |

### Picking a backing strategy per collector

The recommended libraries above are a starting point, not a closed list. Each
collector is free to pull in whatever library best serves its domain —
especially for cloud providers (AWS SDK, Google Cloud SDK, Azure SDK, etc.),
container runtimes (Docker SDK, containerd, CRI), and specialized domains
(libvirt-go, go-smbios, etc.).

Decision order for each collector:

1. **Prefer a well-maintained Go library** when one exists and covers the
   data we need. Saves time, handles edge cases, typically cross-platform.
2. **Prefer an official provider SDK** for cloud collectors where feasible
   (aws-sdk-go for ec2, google.golang.org/cloud for gce, etc.). Alternatively
   plain `net/http` to IMDS endpoints for smaller binary size.
3. **Composite approach** — combine multiple sources, e.g., gopsutil for
   base data plus supplementary `/proc/self/cgroup` checks for fine detail.
4. **Roll our own thin parser** when the data is simple (single file or
   command) and a library would be over-engineering.
5. **Fall back to porting [Ohai's Ruby plugin](https://github.com/chef/ohai/tree/main/lib/ohai/plugins)**
   — when no Go library covers a specific data domain (especially
   Linux-specific `/proc`/`/sys` detail like `interrupts`, `mdadm`, `tc`,
   `livepatch`, or niche cloud providers), reproduce Ohai's Ruby
   implementation in Go. Ohai has already solved the edge cases for every
   major OS/distro; our job is translate, not re-discover.

The Collector interface and `Info` struct shape are the contract — whatever
backing strategy a collector uses, its output must match the typed struct
and Ohai-compatible JSON shape.

### Example: platform collector

```go
// internal/collector/platform/linux.go (or pkg/gohai/collectors/platform/)
import "github.com/shirou/gopsutil/v4/host"

func collect(ctx context.Context) (any, error) {
    info, err := host.InfoWithContext(ctx)
    if err != nil {
        return nil, fmt.Errorf("host.Info: %w", err)
    }
    return &Info{
        Name:         info.Platform,        // "ubuntu"
        Version:      info.PlatformVersion, // "24.04"
        Family:       info.PlatformFamily,  // "debian"
        Architecture: info.KernelArch,      // "arm64"
    }, nil
}
```

The same gopsutil `host.Info` call also feeds the `hostname`, `uptime`,
`kernel`, and `machine_id` collectors — each extracting its relevant fields
into its own typed `Info` struct.

## Architecture

### Package Structure

Collector sub-packages live under `pkg/gohai/collectors/` (not `internal/`) so
that external consumers — notably OSAPI — can import the typed `Info` structs
directly for type-safe fact access.

```
cmd/gohai/                          # CLI entrypoint (Cobra)
  main.go                           # Cobra root command, flag registration
  flags.go                          # Dynamic --collector.<name> flag registration
  output.go                         # JSON / pretty / flat output writers
pkg/gohai/                          # Public SDK
  gohai.go                          # Gohai struct, New(), Collect()
  facts.go                          # Facts struct with typed fields per collector
  options.go                        # Functional options (WithCollectors, etc.)
  registry.go                       # Public registry API
  collectors/                       # PUBLIC collector sub-packages
    platform/
      platform.go                   # Info struct (PUBLIC); Collector impl
      linux.go                      # Linux-specific collection (build-tagged)
      darwin.go                     # macOS-specific collection (build-tagged)
      other.go                      # Fallback for other OSes
      export_linux_test.go          # Exposes private symbols for linux tests
      export_darwin_test.go         # Exposes private symbols for darwin tests
      platform_public_test.go       # Package-level public tests
      linux_public_test.go          # Linux-specific public tests (build-tagged)
      darwin_public_test.go         # macOS-specific public tests (build-tagged)
    cpu/                            # Same pattern for every collector
      cpu.go
      linux.go
      darwin.go
      ...
    ...                             # One sub-package per collector
internal/collector/                 # Collector interface + registry plumbing
  collector.go                      # Collector interface, Tier type
  registry.go                       # Registry (register, resolve deps, run)
```

**Why `pkg/gohai/collectors/` and not `internal/`:** external consumers like
OSAPI need to import typed `Info` structs (e.g., `platform.Info`, `cpu.Info`)
to do type-safe fact access. Go's `internal/` rule would prevent that. Only
the interface/registry plumbing stays in `internal/`.

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

### Facts Struct and Data Layer

The public `Facts` struct in `pkg/gohai/facts.go` has a typed field for every
collector. Each collector emits its own JSON sub-document; gohai composes the
enabled sub-documents into one larger Ohai-style document.

```go
type Facts struct {
    Platform       *platform.Info       `json:"platform,omitempty"`
    CPU            *cpu.Info            `json:"cpu,omitempty"`
    Memory         *memory.Info         `json:"memory,omitempty"`
    Network        *network.Info        `json:"network,omitempty"`
    // ... one field per collector
    CollectTime    time.Time            `json:"collect_time"`
    CollectDuration time.Duration       `json:"collect_duration_ns"`
}
```

Fields for disabled collectors are `nil` and omitted from JSON (via
`omitempty`). Only enabled collectors contribute to the output document.

### Three ways for consumers to access facts

**(a) Typed Go access — the osapi integration path**

```go
facts, _ := g.Collect(ctx)
fmt.Println(facts.Platform.Name)    // "ubuntu"
fmt.Println(facts.CPU.Total)        // 8
fmt.Println(facts.Memory.Total)     // 34359738368
```

**(b) Full Ohai-compatible nested JSON — the CLI path**

```go
json, _ := facts.JSON()        // compact
pretty, _ := facts.PrettyJSON() // indented
```

**(c) Per-component JSON — for partial consumption**

```go
platformJSON, _ := facts.ComponentJSON("platform")
cpuJSON, _      := facts.ComponentJSON("cpu")
```

**(d) Flat dot-separated map — for quick lookups and templating**

```go
flat := facts.Flat()
flat["platform.name"]     // "ubuntu"
flat["cpu.total"]         // 8
facts.Get("platform.name") // shorthand for flat["platform.name"]
```

### Output composition

Each collector produces a self-contained JSON sub-document via its typed
`Info` struct's `json` tags. gohai composes the final document by including
only the sub-documents for collectors that are enabled and successfully ran.
This means:

- Disabled collectors → their field is `nil` → omitted via `omitempty`
- Failed collectors → their field is `nil` → omitted via `omitempty`
- Successful collectors → their field holds the typed `Info` struct →
  serialized as its JSON representation

### CLI

Cobra-based ([github.com/spf13/cobra](https://github.com/spf13/cobra)) with
node_exporter-style flags. Cobra is the project's standard CLI framework —
matches osapi-io conventions and provides dynamic flag registration from the
collector registry:

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
