<p align="center">
  <picture>
    <source srcset="asset/logo-dark.svg" media="(prefers-color-scheme: dark)">
    <source srcset="asset/logo-light.svg" media="(prefers-color-scheme: light)">
    <img src="asset/logo-dark.svg" alt="gohai" width="250">
  </picture>
</p>

<p align="center">An SDK-first Go library for collecting system facts.</p>

<p align="center">
  <a href="https://github.com/osapi-io/gohai/releases/latest"><img alt="release" src="https://img.shields.io/github/release/osapi-io/gohai.svg?style=for-the-badge"></a>
  <a href="https://codecov.io/gh/osapi-io/gohai"><img alt="codecov" src="https://img.shields.io/codecov/c/github/osapi-io/gohai?style=for-the-badge"></a>
  <a href="LICENSE"><img alt="license" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
  <a href="https://github.com/osapi-io/gohai/actions/workflows/go.yml"><img alt="build" src="https://img.shields.io/github/actions/workflow/status/osapi-io/gohai/go.yml?style=for-the-badge"></a>
  <a href="https://github.com/goreleaser"><img alt="powered by" src="https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge"></a>
  <a href="https://conventionalcommits.org"><img alt="conventional commits" src="https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge"></a>
  <a href="https://just.systems"><img alt="built with just" src="https://img.shields.io/badge/Built_with-Just-black?style=for-the-badge&logo=just&logoColor=white"></a>
  <img alt="gitHub commit activity" src="https://img.shields.io/github/commit-activity/m/osapi-io/gohai?style=for-the-badge">
  <a href="https://pkg.go.dev/github.com/osapi-io/gohai/pkg/gohai"><img alt="go reference" src="https://img.shields.io/badge/go-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white"></a>
</p>

<p align="center">
<b>SDK first, CLI second, one contract for both.</b>
</p>

<p align="center">
Sixty-two collectors across ten categories, each wrapping a well-maintained
library rather than reimplementing it, and each returning a typed struct.
Import it, or use the CLI over the same collectors.
</p>

## Install

```bash
curl -fsSL https://github.com/osapi-io/gohai/raw/main/install.sh | bash
```

Installs to `~/.local/bin` (or `/usr/local/bin` as root). SHA-256 checksums
verified. Override with `GOHAI_INSTALL_DIR=/some/path` or pin a version with
`GOHAI_VERSION=1.0.0`.

<details>
<summary>Other install methods</summary>

### Go install

```bash
go install github.com/osapi-io/gohai@latest
```

### As a library dependency

```bash
go get github.com/osapi-io/gohai
```

### Build from source

```bash
git clone https://github.com/osapi-io/gohai.git
cd gohai
go build -o gohai .
```

</details>

## Features

| Feature                        | Description                                                                                                                                                             |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔌 Pluggable Collectors        | Enable/disable individual fact collectors                                                                                                                               |
| 🏗️ Typed Structs               | Strongly-typed Go structs for all facts                                                                                                                                 |
| 📄 JSON Output                 | Nested JSON output for CLI and programmatic use                                                                                                                         |
| 🗺️ Flat Map Access             | Dot-separated key-value access                                                                                                                                          |
| 🐧 Cross-Platform              | Linux primary, macOS best-effort                                                                                                                                        |
| 🔗 Collector Dependencies      | Automatic dependency resolution between facts                                                                                                                           |
| ⚡ Concurrent Collection       | Collectors run concurrently; dependency graph resolves order when any collector declares deps.                                                                          |
| ⏱️ Per-Collector Timings       | Opt-in `--with-timings` / `WithTimings()` embeds per-collector durations, status, and error messages under `_timings` in the JSON output                                |
| 📊 OCSF + OpenTelemetry + Ohai | Field names follow [OCSF](https://schema.ocsf.io/) then [OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/resource/); data sources mirror Chef Ohai's plugins |
| 🔄 Native OCSF Output          | `--format ocsf` produces a standards-compliant OCSF `inventory_info` event (class_uid 5001), feed directly into SIEMs and data lakes                                    |
| 🔌 SDK Integration             | Import as a Go package into any Go service                                                                                                                              |

## Collectors

62 collectors across 10 categories. See the
**[Collectors reference](docs/collectors/README.md)** for the full catalog,
implementation status, default membership, schema mappings, and per-collector
docs — it lists 65 entries, the other three being deprecated and never
implemented.

Collectors are individually toggled using node_exporter-style flags.
`--collector.<name>` to opt in, `--no-collector.<name>` to opt out. SDK
consumers use `gohai.WithEnabled(...)` / `gohai.WithDisabled(...)` /
`gohai.WithCollectors(...)`.

**Defaults are opt-in.** `gohai.New()` returns an empty registry. Pass
`gohai.WithDefaults()` for the recommended set (cheap + near-universal,
identity, base hardware, network, load, virt detect). The CLI wires
`WithDefaults()` automatically; pass `--no-defaults` to skip it and use only
explicit `--collector.X` flags.

## Usage

### CLI

```bash
gohai collect --pretty                          # default collectors, pretty JSON
gohai collect --format ocsf --pretty            # OCSF inventory_info event
gohai collect --flat                            # flat key=value pairs
gohai collect --no-defaults --collector.cpu     # specific collectors only
gohai collect --pretty | gohai validate          # validate against schema
gohai version                                   # build info
```

### SDK

Importers should read the
[full API reference on pkg.go.dev][package documentation] for every `Option`,
`Facts` field, and `Info` struct. That is the authoritative list. The examples
below show the two usage shapes.

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

**Consuming stored facts** (decoder side, e.g. a server that received a fact
blob from an agent):

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
`gohai.WithTimings()` to `gohai.New(...)`, useful for debugging slow collectors
or seeing why a collector failed without blocking the run. See the `Timings`
field on [pkg.go.dev][package documentation].

**Detecting which cloud you're on.** Enable the cloud collectors
(`WithCategory("cloud")`) and switch on `Facts.Cloud()`, which returns a
`*Cloud` with `Name` set to a provider identifier, or nil when no cloud was
detected. Use the exported `gohai.CloudAWS` / `CloudGCE` / `CloudAzure` / etc.
constants instead of raw strings:

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

Rich per-provider data lives on the typed `Facts.Ec2` / `Facts.Gce` / etc.
field. See [docs/collectors/cloud.md](docs/collectors/cloud.md) for the full
pattern.

## Examples

Each example is a standalone Go program you can read and run.

| Example                                                   | What it shows                                                |
| --------------------------------------------------------- | ------------------------------------------------------------ |
| [default-collection](examples/default-collection/main.go) | Collect the default set and print it as JSON                 |
| [select-collectors](examples/select-collectors/main.go)   | Restrict collection to named collectors                      |
| [categories](examples/categories/main.go)                 | Select by category, then subtract one                        |
| [typed-facts](examples/typed-facts/main.go)               | Read results as typed fields, by key path, and as a flat map |
| [timings](examples/timings/main.go)                       | Surface per-collector duration and failure                   |
| [list-collectors](examples/list-collectors/main.go)       | Enumerate collectors without collecting                      |

## Documentation

- [Package documentation] on pkg.go.dev. Generated API reference. Every
  `Option`, `Facts` field, and `Info` struct is documented there. This is the
  authoritative SDK reference.
- [Collectors reference](docs/collectors/README.md). One doc per collector with
  fields, schema mappings (OCSF + OpenTelemetry), and Ohai source alignment.
- [Schemas](schemas/README.md). JSON Schema, field-naming strategy (OCSF > OTel
  \> convention), OCSF gap analysis, and cloud canonical overlay.
- [Contributing](CONTRIBUTING.md). Prerequisites, setup, testing, commit
  conventions.
- [Collector methodology](docs/methodology.md). Library selection, field naming,
  and data sources.

## Related works

gohai builds on the following projects, as methodology references, as backing
libraries we wrap, or as peers solving adjacent problems:

**Fact collectors (direct peers):**

- [Chef Ohai]. The canonical reference. Ruby-based plugin-driven fact collector;
  every gohai collector cross-references the corresponding Ohai plugin for data
  sources and per-distro edge cases.
- [Puppet Facter](https://github.com/puppetlabs/facter). Puppet's equivalent.
  Different JSON shape, overlapping fact surface.
- [osquery](https://github.com/osquery/osquery). Meta's SQL-based endpoint
  visibility. Different abstraction (SQL), same data space; common reference
  point when evaluating an inventory tool.
- [Ansible setup](https://docs.ansible.com/ansible/latest/collections/ansible/builtin/setup_module.html)
  is Ansible's built-in fact gathering, exposed as `ansible_facts` in playbooks.
- [Salt Grains](https://docs.saltproject.io/en/latest/topics/grains/).
  SaltStack's static facts.

**Backing libraries (we import these):**

- [gopsutil]. Primary source for dynamic runtime state (memory, network I/O,
  process enumeration, virtualization detection).
- [ghw]. Canonical for physical hardware topology (CPU NUMA, DIMMs, block
  devices, DMI, GPU, PCI).
- [procfs]. Linux `/proc` and `/sys` parsing when a library doesn't cover a
  field.
- [go-sysinfo](https://github.com/elastic/go-sysinfo). Elastic's alternative for
  host/platform/kernel facts.
- [avfs](https://github.com/avfs/avfs). Virtual filesystem abstraction used in
  every collector that reads files, so tests can run against in-memory fixtures.

**Other Go libraries in the space:**

- [gosigar](https://github.com/cloudfoundry/gosigar). Cloud Foundry's Go port of
  Hyperic Sigar. Historical reference for Go-based host metrics.
- [go-ps](https://github.com/mitchellh/go-ps). Narrow process-listing library.
  gopsutil supersedes it for our use.
- [goprocinfo](https://github.com/c9s/goprocinfo). Lightweight `/proc` parser.
  gopsutil + procfs cover the same ground for us.

**Methodology references (we read, don't import):**

- [node_exporter](https://github.com/prometheus/node_exporter). The reference we
  check against for tricky Linux `/proc` and `/sys` parsing. Apache-2, but we
  rewrite in our style rather than import.
- [psutil](https://github.com/giampaolo/psutil). The Python library gopsutil is
  a port of; the original design reference for the dynamic-state facts.

## Contributing

See the [Contributing](CONTRIBUTING.md) guide for prerequisites, setup,
conventions, and the PR workflow.

## License

The [MIT] License.

[chef ohai]: https://docs.chef.io/ohai/
[ghw]: https://github.com/jaypipes/ghw
[gopsutil]: https://github.com/shirou/gopsutil
[mit]: LICENSE
[package documentation]: https://pkg.go.dev/github.com/osapi-io/gohai/pkg/gohai
[procfs]: https://github.com/prometheus/procfs
