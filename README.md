<p align="center">
  <picture>
    <source srcset="asset/logo-dark.svg" media="(prefers-color-scheme: dark)">
    <source srcset="asset/logo-light.svg" media="(prefers-color-scheme: light)">
    <img src="asset/logo-dark.svg" alt="gohai" width="250">
  </picture>
</p>

<p align="center">An SDK-first Go library that collects system facts.</p>

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
Sixty-two collectors in ten categories. Each collector wraps a maintained
library and does not write the logic again. Each collector returns a typed
struct. Import the library, or use the CLI with the same collectors.
</p>

## 📦 Install

```bash
curl -fsSL https://github.com/osapi-io/gohai/raw/main/install.sh | bash
```

The script installs to `~/.local/bin`, or to `/usr/local/bin` as root. It
verifies the SHA-256 checksums. To change the directory, set
`GOHAI_INSTALL_DIR=/some/path`. To install one version, set
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

## ✨ Features

| Feature                        | Description                                                                                                                                                                              |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔌 Pluggable Collectors        | Enable or disable each fact collector                                                                                                                                                    |
| 🏗️ Typed Structs               | Strongly typed Go structs for all facts                                                                                                                                                  |
| 📄 JSON Output                 | Nested JSON output for the CLI and for programs                                                                                                                                          |
| 🗺️ Flat Map Access             | Key-value access with dot-separated keys                                                                                                                                                 |
| 🐧 Cross-Platform              | Linux first, macOS where possible                                                                                                                                                        |
| 🔗 Collector Dependencies      | gohai resolves the dependencies between facts                                                                                                                                            |
| ⚡ Concurrent Collection       | Collectors run at the same time. When a collector declares dependencies, the dependency graph sets the order.                                                                            |
| ⏱️ Per-Collector Timings       | The optional `--with-timings` / `WithTimings()` adds the duration, status and error message of each collector under `_timings` in the JSON output                                        |
| 📊 OCSF + OpenTelemetry + Ohai | Field names follow [OCSF](https://schema.ocsf.io/), then [OpenTelemetry](https://opentelemetry.io/docs/specs/semconv/resource/). The data sources are the same as in Chef Ohai's plugins |
| 🔄 Native OCSF Output          | `--format ocsf` makes an OCSF `inventory_info` event (class_uid 5001) that follows the standard. You can send it directly to SIEMs and data lakes                                        |
| 🔌 SDK Integration             | Import it as a Go package into a Go service                                                                                                                                              |

## 🧰 Collectors

62 collectors in 10 categories. The
[Collectors reference](docs/collectors/README.md) has the full catalog, the
implementation status, the default set, the schema mappings, and a page for each
collector. It lists 65 entries. The other three entries are deprecated and have
no implementation.

You enable or disable each collector with flags in the node_exporter style.
`--collector.<name>` enables a collector. `--no-collector.<name>` disables it.
SDK consumers use `gohai.WithEnabled(...)` / `gohai.WithDisabled(...)` /
`gohai.WithCollectors(...)`.

The defaults are optional. `gohai.New()` returns an empty registry. Pass
`gohai.WithDefaults()` to get the recommended set. This set has collectors that
are fast and work on almost all hosts: identity, base hardware, network, load
and virtualization detection. The CLI adds `WithDefaults()` automatically. To
use only the collectors that you name with `--collector.X` flags, pass
`--no-defaults`.

## 🚀 Usage

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

If you import the package, read the
[full API reference on pkg.go.dev][package documentation] for each `Option`,
`Facts` field and `Info` struct. That reference is the authoritative list. The
examples below show the two ways to use the package.

### Collect facts

The producer side:

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

### Read stored facts

The decoder side, for example a server that received a blob of facts from an
agent:

```go
var facts gohai.Facts
if err := json.Unmarshal(payload, &facts); err != nil {
    log.Fatal(err)
}

// Typed access on the decoded value — no map[string]any guessing.
fmt.Println(facts.Platform.Name, facts.CPU.Cores)
fmt.Println(facts.Network.DefaultInterface)
```

To add the timing and the error message of each collector to Facts, add
`gohai.WithTimings()` to `gohai.New(...)`. Use this to find slow collectors. Use
it also to see why a collector failed, without a stop of the run. Refer to the
`Timings` field on [pkg.go.dev][package documentation].

### Find the cloud of the host

Enable the cloud collectors (`WithCategory("cloud")`). Then switch on
`Facts.Cloud()`. It returns a `*Cloud` with `Name` set to a provider identifier.
If gohai found no cloud, it returns nil. Use the exported constants
`gohai.CloudAWS` / `CloudGCE` / `CloudAzure` and the others. Do not use raw
strings:

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

The full data of each provider is in its typed field, for example `Facts.Ec2` or
`Facts.Gce`. Refer to [docs/collectors/cloud.md](docs/collectors/cloud.md) for
the full pattern.

## 🧪 Examples

Each example is a standalone Go program that you can read and run.

| Example                                                   | What it shows                                                |
| --------------------------------------------------------- | ------------------------------------------------------------ |
| [default-collection](examples/default-collection/main.go) | Collect the default set and print it as JSON                 |
| [select-collectors](examples/select-collectors/main.go)   | Collect only from the named collectors                       |
| [categories](examples/categories/main.go)                 | Select by category, then remove one                          |
| [typed-facts](examples/typed-facts/main.go)               | Read results as typed fields, by key path, and as a flat map |
| [timings](examples/timings/main.go)                       | Show the duration and failure of each collector              |
| [list-collectors](examples/list-collectors/main.go)       | List the collectors without a collection                     |

## 📚 Documentation

- [Package documentation] on pkg.go.dev. The generated API reference. It
  documents each `Option`, `Facts` field and `Info` struct. This is the
  authoritative SDK reference.
- [Collectors reference](docs/collectors/README.md). One page for each
  collector, with the fields, the schema mappings (OCSF + OpenTelemetry), and
  the comparison with the Ohai sources.
- [Schemas](schemas/README.md). The JSON Schema, the rules for field names (OCSF
  \> OTel > convention), the OCSF gap analysis, and the canonical cloud overlay.
- [Contributing](CONTRIBUTING.md). The prerequisites, the setup, the tests and
  the commit conventions.
- [Collector methodology](docs/methodology.md). How to select a library, how to
  name fields, and the data sources.

## 🔗 Related works

gohai uses these projects in three ways. Some are methodology references. Some
are backing libraries that gohai wraps. Some are peers that solve related
problems.

### Fact collectors (direct peers)

- [Chef Ohai]. The canonical reference. A fact collector in Ruby, made of
  plugins. Each gohai collector compares with the related Ohai plugin for the
  data sources and for the special cases of each distribution.
- [Puppet Facter](https://github.com/puppetlabs/facter). The Puppet equivalent.
  It has a different JSON shape and many of the same facts.
- [osquery](https://github.com/osquery/osquery). Meta's endpoint visibility
  tool, based on SQL. It uses a different abstraction (SQL) for the same data.
  People often compare inventory tools with it.
- [Ansible setup](https://docs.ansible.com/ansible/latest/collections/ansible/builtin/setup_module.html)
  is the built-in fact collection of Ansible. Playbooks get the facts as
  `ansible_facts`.
- [Salt Grains](https://docs.saltproject.io/en/latest/topics/grains/). The
  static facts of SaltStack.

### Backing libraries (gohai imports these)

- [gopsutil]. The primary source for the dynamic runtime state: memory, network
  I/O, the process list and virtualization detection.
- [ghw]. The canonical source for the physical hardware topology: CPU NUMA,
  DIMMs, block devices, DMI, GPU and PCI.
- [procfs]. It parses Linux `/proc` and `/sys` when no other library has a
  field.
- [go-sysinfo](https://github.com/elastic/go-sysinfo). The Elastic alternative
  for host, platform and kernel facts.
- [avfs](https://github.com/avfs/avfs). A virtual filesystem abstraction. Each
  collector that reads files uses it, so tests can run against in-memory
  fixtures.

### Other Go libraries in the space

- [gosigar](https://github.com/cloudfoundry/gosigar). The Cloud Foundry Go port
  of Hyperic Sigar. An old reference for host metrics in Go.
- [go-ps](https://github.com/mitchellh/go-ps). A small library that lists
  processes. For gohai, gopsutil replaces it.
- [goprocinfo](https://github.com/c9s/goprocinfo). A small `/proc` parser. For
  gohai, gopsutil + procfs do the same work.

### Methodology references (gohai reads them, but does not import them)

- [node_exporter](https://github.com/prometheus/node_exporter). The reference
  for difficult parsing of Linux `/proc` and `/sys`. It is Apache-2, but gohai
  writes the code again in its own style and does not import it.
- [psutil](https://github.com/giampaolo/psutil). gopsutil is a port of this
  Python library. It is the original design reference for the dynamic-state
  facts.

## 🤝 Contributing

Refer to the [Contributing](CONTRIBUTING.md) guide for the prerequisites, the
setup, the conventions and the PR workflow.

## 📄 License

The [MIT] License.

[chef ohai]: https://docs.chef.io/ohai/
[ghw]: https://github.com/jaypipes/ghw
[gopsutil]: https://github.com/shirou/gopsutil
[mit]: LICENSE
[package documentation]: https://pkg.go.dev/github.com/osapi-io/gohai/pkg/gohai
[procfs]: https://github.com/prometheus/procfs
