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

> 🐧 **Linux-first.** macOS is supported with a narrower field surface
> (see per-collector docs for platform coverage); Windows is not supported.

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

| Feature                         | Description                                                                                                                                                           |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔌 Pluggable Collectors         | Enable/disable individual fact collectors                                                                                                                             |
| 🏗️ Typed Structs                | Strongly-typed Go structs for all facts                                                                                                                               |
| 📄 JSON Output                  | Nested JSON output for CLI and programmatic use                                                                                                                       |
| 🗺️ Flat Map Access              | Dot-separated key-value access                                                                                                                                        |
| 🐧 Cross-Platform               | Linux primary, macOS best-effort                                                                                                                                      |
| 🔗 Collector Dependencies       | Automatic dependency resolution between facts                                                                                                                         |
| ⚡ Concurrent Collection        | Collectors run concurrently; dependency graph resolves order when any collector declares deps.                                                                         |
| ⏱️ Per-Collector Timings        | Opt-in `--with-timings` / `WithTimings()` embeds per-collector durations, status, and error messages under `_timings` in the JSON output                              |
| 📊 OCSF + OpenTelemetry + Ohai  | Field names follow [OCSF](https://schema.ocsf.io/) then [OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/resource/); data sources mirror Chef Ohai's plugins |
| 🔌 SDK Integration              | Import as a Go package for OSAPI and others                                                                                                                           |

## 🔌 Collectors

65 collectors across 9 categories. See the **[Collectors
reference](docs/collectors/README.md)** for the full catalog —
implementation status, default membership, schema mappings, and
per-collector docs.

Collectors are individually toggled using node_exporter-style flags —
`--collector.<name>` to opt in, `--no-collector.<name>` to opt out. SDK
consumers use `gohai.WithEnabled(...)` / `gohai.WithDisabled(...)` /
`gohai.WithCollectors(...)`.

**Defaults are opt-in.** `gohai.New()` returns an empty registry. Pass
`gohai.WithDefaults()` for the recommended set (cheap + near-universal —
identity, base hardware, network, load, virt detect). The CLI wires
`WithDefaults()` automatically; pass `--no-defaults` to skip it and use only
explicit `--collector.X` flags.

## 🎯 Usage

### CLI

```bash
# Collect the recommended default collector set
gohai

# Pretty-printed JSON
gohai --pretty

# Flat key=value pairs instead of nested JSON
gohai --flat

# Enable specific collectors on top of the defaults
gohai --collector.process --collector.packages

# Disable specific collectors
gohai --no-collector.virtualization --no-collector.network

# Skip defaults entirely; run only what --collector.X turns on
gohai --no-defaults --collector.platform --collector.cpu

# Embed per-collector timings + errors under `_timings` in the JSON
gohai --with-timings --pretty

# List every registered collector and exit
gohai --list-collectors
```

### SDK

Importers should read the [full API reference on pkg.go.dev][Package
documentation] for every `Option`, `Facts` field, and `Info` struct —
that's the authoritative API surface. The examples below show the two
usage shapes.

**Collecting facts** (producer side):

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
        gohai.WithDefaults(),                      // the recommended set
        gohai.WithEnabled("process", "packages"),  // plus these two
    )
    if err != nil {
        log.Fatal(err)
    }

    facts, err := g.Collect(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Typed access — pkg.go.dev documents every Info struct's fields.
    fmt.Printf("OS:     %s %s\n", facts.Platform.Name, facts.Platform.Version)
    fmt.Printf("Cores:  %d\n", facts.CPU.Cores)
    fmt.Printf("Memory: %d bytes\n", facts.Memory.Total)

    // Serialize for transport / storage.
    b, _ := facts.PrettyJSON()
    fmt.Println(string(b))
}
```

**Consuming stored facts** (decoder side — e.g. a server that received
a fact blob from an agent):

```go
var facts gohai.Facts
if err := json.Unmarshal(payload, &facts); err != nil {
    log.Fatal(err)
}

// Typed access on the decoded value — no map[string]any guessing.
fmt.Println(facts.Platform.Name, facts.CPU.Cores)
fmt.Println(facts.Network.DefaultInterface)
```

Per-collector timings + error messages can be embedded in Facts by adding
`gohai.WithTimings()` to `gohai.New(...)` — useful for debugging slow
collectors or seeing why a collector failed without blocking the run.
See the `Timings` field on [pkg.go.dev][Package documentation].

**Detecting which cloud you're on.** Enable the cloud collectors
(`WithCategory("cloud")`) and switch on `Facts.Cloud()` — returns a
`*Cloud` with `Name` set to a provider identifier, or nil when no
cloud was detected. Use the exported `gohai.CloudAWS` / `CloudGCE` /
`CloudAzure` / etc. constants instead of raw strings:

```go
g, _ := gohai.New(gohai.WithCategory("cloud"))
facts, _ := g.Collect(ctx)

cloud := facts.Cloud()
if cloud == nil { return } // not on a supported cloud

switch cloud.Name {
case gohai.CloudAWS:
    fmt.Println(facts.Ec2.Region, facts.Ec2.IAMInfo.InstanceProfileArn)
case gohai.CloudGCE:
    fmt.Println(facts.Gce.ProjectID, facts.Gce.Zone)
}
```

Rich per-provider data lives on the typed `Facts.Ec2` / `Facts.Gce` /
etc. field. See [docs/collectors/cloud.md](docs/collectors/cloud.md)
for the full pattern.

## 📖 Documentation

- [Package documentation][] on pkg.go.dev — generated API reference. Every
  `Option`, `Facts` field, and `Info` struct is documented there. This is
  the authoritative SDK reference.
- [Collectors reference](docs/collectors/README.md) — one doc per collector
  with fields, schema mappings (OCSF + OpenTelemetry), and Ohai source
  alignment.
- [Development](docs/development.md) — prerequisites, setup, testing, commit
  conventions.
- [Contributing](docs/contributing.md) — PR workflow.

## 🤝 Contributing

See the [Development](docs/development.md) guide for prerequisites, setup,
and conventions. See the [Contributing](docs/contributing.md) guide before
submitting a PR.

## 🔗 Related Works

gohai stands on the shoulders of the following projects — as methodology
references, as backing libraries we wrap, or as peers solving adjacent
problems:

**Fact collectors (direct peers):**

- [Chef Ohai][] — the canonical reference. Ruby-based plugin-driven fact
  collector; every gohai collector cross-references the corresponding
  Ohai plugin for data sources and per-distro edge cases.
- [Puppet Facter](https://github.com/puppetlabs/facter) — Puppet's
  equivalent. Different JSON shape, overlapping fact surface.
- [osquery](https://github.com/osquery/osquery) — Meta's SQL-based
  endpoint visibility. Different abstraction (SQL), same data space;
  common reference point when evaluating an inventory tool.
- [Ansible setup](https://docs.ansible.com/ansible/latest/collections/ansible/builtin/setup_module.html) —
  Ansible's built-in fact gathering, exposed as `ansible_facts` in
  playbooks.
- [Salt Grains](https://docs.saltproject.io/en/latest/topics/grains/) —
  SaltStack's static facts.

**Backing libraries (we import these):**

- [gopsutil][] — primary source for dynamic runtime state (memory,
  network I/O, process enumeration, virtualization detection).
- [ghw][] — canonical for physical hardware topology (CPU NUMA,
  DIMMs, block devices, DMI, GPU, PCI).
- [procfs][] — Linux `/proc` and `/sys` parsing when a library doesn't
  cover a field.
- [go-sysinfo](https://github.com/elastic/go-sysinfo) — Elastic's
  alternative for host/platform/kernel facts.
- [avfs](https://github.com/avfs/avfs) — virtual filesystem
  abstraction used in every collector that reads files, so tests can
  run against in-memory fixtures.

**Other Go libraries in the space:**

- [gosigar](https://github.com/cloudfoundry/gosigar) — Cloud Foundry's
  Go port of Hyperic Sigar. Historical reference for Go-based host
  metrics.
- [go-ps](https://github.com/mitchellh/go-ps) — narrow process-listing
  library. gopsutil supersedes it for our use.
- [goprocinfo](https://github.com/c9s/goprocinfo) — lightweight `/proc`
  parser. gopsutil + procfs cover the same ground for us.

**Methodology references (we read, don't import):**

- [node_exporter](https://github.com/prometheus/node_exporter) — gold
  standard for tricky Linux `/proc` and `/sys` parsing. Apache-2, but
  we rewrite in our style rather than import.
- [psutil](https://github.com/giampaolo/psutil) — the Python library
  gopsutil is a port of; the original design reference for the
  dynamic-state facts.

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
