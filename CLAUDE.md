# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

**gohai is an SDK first.** It is a Go library that [OSAPI][] and other Go
services import to collect typed system facts, with a standalone CLI
(`gohai`) shipped as a thin wrapper over the same SDK. When making design
decisions, always optimize for the importer's experience first; the CLI is
secondary.

Inspired by [Chef Ohai][] — pluggable collectors, comprehensive facts, and
Ohai-compatible JSON output — but designed to be embedded in Go services
rather than run as a standalone agent.

## Implementation Methodology

**We are a wrapper and aggregator, not a re-implementor.** Each collector's
job is to wrap a well-maintained backing source (Go library, provider SDK, or
a thin file/command parser) and reshape its output into our typed `Info`
struct for Ohai-compatible JSON.

**Decision order for each collector:**

1. **Prefer a well-maintained Go library** — saves time, handles edge cases.
   Standard picks: [gopsutil][] (CPU/memory/disk/network/host/process),
   [ghw][] (hardware detail), [procfs][] (raw `/proc`, `/sys`),
   [go-sysinfo][] (alternative host info).
2. **Prefer official provider SDKs** for cloud collectors (aws-sdk-go,
   google.golang.org/cloud, azure-sdk-for-go) or plain `net/http` to IMDS
   endpoints for smaller binaries.
3. **Composite approach** when needed — combine multiple sources.
4. **Roll our own thin parser** when the data is simple (single file or
   command) and a library would be over-engineering.
5. **Fall back to porting [Ohai's Ruby plugin][ohai-plugins]** when no Go
   library covers the domain. Ohai has solved the edge cases for every major
   OS/distro; our job is translate, not re-discover.

**We learn from, but don't directly import, [node_exporter][]** — their
collectors are a gold reference for tricky Linux `/proc` and `/sys` parsing
(Apache-2 licensed). Read, understand, rewrite in our style.

The Collector interface and `Info` struct shape are the contract — whatever
backing strategy a collector uses, its output must match the typed struct
and consumer expectations.

### Field naming

Collector JSON field names use `snake_case`. For the specific names,
follow this precedence:

1. **[OCSF](https://ocsf.io/)** (Open Cybersecurity Schema Framework) —
   when a field we collect has a canonical OCSF name, use it. OCSF is
   the industry schema for asset/observability/security data, backed by
   AWS and Splunk, and aligning means our output feeds SIEMs and
   inventory tools without translation. Examples: `process.cmd_line`
   (not `cmdline`), `network_interface.mac` (not `hardware_addr`),
   `os.kernel_release` (not `kernel_version`).
2. **Industry standard** — when OCSF doesn't cover the field, use
   whatever node_exporter / systemd / Prometheus exporters have
   standardized on. Example: filesystem `mountpoint` / `fstype` follow
   node_exporter, not Ohai's `mount` / `fs_type`.
3. **Ohai's name** — only when OCSF and industry standards are silent
   AND Ohai has a clear, meaningful name.
4. **Our own name** — last resort. Use a Go-idiomatic snake_case form.

**Do not mirror Ohai's JSON shape.** Ohai is for data-source reference
(collection approach, distro edge cases) — not field names or struct
layout. Ruby Mash ↔ Go struct translation isn't worthwhile to pin
byte-for-byte.

### MANDATORY: Cross-reference Ohai's data sources before implementing

Before writing code for a new collector (or modifying an existing one),
**read Ohai's corresponding plugin and spec** — but the goal is to match
their **collection approach** (what file/command/library they read, what
edge cases they handle, how they detect per-distro differences), **not**
their JSON output shape. Ruby Mash ↔ Go struct translation isn't
worthwhile to pin byte-for-byte; Go-native JSON shape is fine.

What matters: Ohai has years of accumulated bug fixes and distro-specific
quirks. Leverage that. If they read `/proc/X` plus fall back to `cmd Y`
on SUSE, we should too. If they have special handling for Amazon Linux
vs RHEL, we need to think about it too.

Fetch both files with `gh api`:

```bash
gh api repos/chef/ohai/contents/lib/ohai/plugins/<name>.rb --jq .content | base64 -d
gh api repos/chef/ohai/contents/spec/unit/plugins/<name>_spec.rb --jq .content | base64 -d
```

Filenames occasionally differ — many plugins live under OS subdirs
(`linux/`, `darwin/`, `windows/`). Browse
`repos/chef/ohai/contents/lib/ohai/plugins` if the direct path 404s.

Every collector doc **must** carry a standard **"Data Sources"** section.
Complex collectors that emit multiple derived facts answering different
questions also get a **"Signals"** section.

**Data Sources** (required on every doc):

```md
## Data Sources

| Platform | What we read | Ohai equivalent | Alignment |
| --- | --- | --- | --- |
| Linux   | `/path/or/command` | Ohai plugin + what it reads | equivalent / we deviate because … |
| macOS   | …                  | …                          | …                                |

**Known gaps:** any edge cases Ohai handles that we don't yet, with a
one-line plan (add-now or tracked follow-up).
```

**Signals** (required on complex collectors like `fips` where multiple
fields answer different consumer questions; omit for simple collectors
like `shells` or `root_group` where the fact is a single value).

Use a prose list immediately after the Description section:

```md
The collector reports N related signals:

- `<field>` — what it means, what source it comes from, what question
  it answers for the consumer.
- `<field>` — same, including when this signal and the one above can
  disagree and what that disagreement tells you.
```

Signals are about **meaning**, not structure. Use them whenever a
consumer can reasonably ask "which of these fields should I look at for
X?" — the Signals section answers that before they have to read the
field table.

This keeps docs consistent and makes it obvious at a glance whether
we're leveraging Ohai's hard-won knowledge or flying solo. If Ohai has
coverage we lack, either add it in the same PR or open a tracked issue
— don't silently drop it.

[Reference PR adding this rule: chef/ohai#1754]

## Development Reference

For setup, prerequisites, and contributing guidelines:

- @docs/development.md - Prerequisites, setup, code style, testing, commits
- @docs/contributing.md - PR workflow and contribution guidelines
- @docs/collectors/README.md - Per-collector reference

## Quick Reference

```bash
just fetch / just deps / just test / just go::unit / just go::vet / just go::fmt
```

## Package Structure

- **`main.go`** — repo-root entry point; just calls `cmd.Execute()`
- **`cmd/`** — Cobra CLI: `root.go`, `flags.go`, `output.go`
- **`pkg/gohai/`** — Public SDK
  - `gohai.go` — `Gohai` struct, `New()`, `Collect()`
  - `facts.go` — `Facts` struct with `Data map[string]any` and JSON/Flat methods
  - `options.go` — functional options (`WithEnabled`, `WithDisabled`, `WithCollectors`)
  - `registry.go` — `PublicRegistry` used by CLI for flag enumeration
- **`pkg/gohai/collectors/<name>/`** — Public per-collector sub-packages
  - `<name>.go` — `Info` struct and `Collector` implementation
  - `linux.go` / `darwin.go` / `other.go` — build-tagged OS implementations.
    **Always separate files** — keep `darwin.go` distinct from `linux.go`
    even when code is byte-identical. Uniform layout > tiny duplication.
  - `linux_export_test.go` / `darwin_export_test.go` — exposes private
    symbols to public tests (`var X = x`). Requires an explicit
    `//go:build linux` or `//go:build darwin` tag (filename does not
    auto-tag because the suffix isn't `_<GOOS>`).
  - `<name>_public_test.go` — package-level public tests
  - `linux_public_test.go` / `darwin_public_test.go` — OS-specific public
    tests (filename auto-tags; explicit `//go:build` tag also present)
- **`internal/collector/`** — Collector interface + registry plumbing
  - `collector.go` — `Collector` interface, `Tier` type
  - `registry.go` — `Registry` (register, resolve deps, run concurrently)

## Code Standards (MANDATORY)

### File Headers

Every `.go` file MUST start with the MIT license header — see any existing
Go file in the repo for the exact format. Build-tagged files put `//go:build`
on line 1, blank line, then the header.

### Function Signatures

Functions with parameters MUST use multi-line format:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
}
```

Zero-parameter functions stay single-line:

```go
func (t Tier) String() string {
    return "core"
}
```

### Testing

- Public tests: `*_public_test.go` in test package
  (`package gohai_test` or `package collector_test`) for exported functions
- Internal tests: `*_test.go` in same package (`package gohai`)
  for private functions
- Suite naming: `*_public_test.go` → `{Name}PublicTestSuite`,
  `*_test.go` → `{Name}TestSuite`
- Use `testify/suite` with table-driven patterns
- One suite method per function under test — all scenarios (success, errors,
  edge cases) as rows in one table
- Target 100% test coverage on all packages

### Go Patterns

- Error wrapping: `fmt.Errorf("context: %w", err)`
- Early returns over nested if-else
- Unused parameters: rename to `_`
- Import order: stdlib, third-party, local (blank-line separated)

### Linting

golangci-lint with: errcheck, errname, goimports, govet, prealloc,
predeclared, revive, staticcheck. Generated files (`*.gen.go`, `*.pb.go`)
are excluded from formatting.

### Branching

See @docs/development.md#branching for full conventions.

When committing changes via `/commit`, create a feature branch first if
currently on `main`. Branch names use the pattern `type/short-description`
(e.g., `feat/add-cpu-collector`, `fix/memory-parsing`, `docs/update-readme`).

### Task Tracking

Implementation planning and execution uses the superpowers plugin workflows
(`writing-plans` and `executing-plans`). Plans live in `docs/superpowers/`.

### Commit Messages

See @docs/development.md#commit-messages for full conventions.

Follow [Conventional Commits](https://www.conventionalcommits.org/) with the
50/72 rule. Format: `type(scope): description`.

When committing via Claude Code, end with:

- `🤖 Generated with [Claude Code](https://claude.ai/code)`
- `Co-Authored-By: Claude <noreply@anthropic.com>`

## Adding a New Collector

When adding a new collector (e.g., `cpu`), follow these steps. The `platform`
collector at `pkg/gohai/collectors/platform/` is the reference implementation
— copy its file layout exactly.

### Step 1: Create the sub-package

Path: `pkg/gohai/collectors/<name>/` (public — consumers like OSAPI need to
import `Info` structs directly).

### Step 2: `<name>.go` — Info struct and Collector

```go
// (MIT header)
// Package cpu collects CPU topology and feature facts.
package cpu

import (
    "context"

    "github.com/osapi-io/gohai/internal/collector"
)

// Info holds CPU information collected from the system.
type Info struct {
    Total     int      `json:"total"`
    Cores     int      `json:"cores"`
    ModelName string   `json:"model_name,omitempty"`
    Flags     []string `json:"flags,omitempty"`
}

// Collector implements collector.Collector for CPU facts.
type Collector struct{}

// New returns a new CPU Collector.
func New() *Collector {
    return &Collector{}
}

// Name returns "cpu".
func (c *Collector) Name() string {
    return "cpu"
}

// Tier returns the collector's tier (TierCore, TierExtended, TierOptIn).
func (c *Collector) Tier() collector.Tier {
    return collector.TierCore
}

// Dependencies returns upstream collectors this one depends on.
func (c *Collector) Dependencies() []string {
    return nil
}

// Collect gathers CPU facts. Implementation lives in linux.go / darwin.go.
func (c *Collector) Collect(
    ctx context.Context,
) (any, error) {
    return collect(ctx)
}
```

### Step 3: OS-specific implementations

Follow the library decision order (see "Implementation Methodology" above).
Wrap gopsutil / ghw / procfs / cloud SDK and reshape the output.

#### Linux distro-family branching

Go build tags split by `GOOS` (linux/darwin/windows) — they **cannot** split
by Linux distro family (debian/rhel/suse/arch/alpine). Family branching
happens at **runtime** by reading `platform.Family` (debian, rhel, fedora,
amazon, suse, arch, alpine, mac_os_x).

**Default: runtime switch inside `linux.go`.** Readable top-down, easy to
test by injecting a probe/parse function.

```go
//go:build linux

func collect(
    ctx context.Context,
) (any, error) {
    family := detectFamily(ctx) // read /etc/os-release or call gopsutil
    name, path := detectLinuxManager(family, probeBinary)
    return &Info{Name: name, Path: path}, nil
}

func detectLinuxManager(
    family string,
    probe func(string) (name, path string),
) (string, string) {
    switch family {
    case "debian":
        return probe("apt")
    case "rhel", "fedora", "amazon":
        if n, p := probe("dnf"); n != "" {
            return n, p
        }
        return probe("yum")
    case "suse":
        return probe("zypper")
    case "arch":
        return probe("pacman")
    case "alpine":
        return probe("apk")
    }
    return "", ""
}
```

**When to split into family files:** if a Linux collector's per-family
branches exceed ~50 lines each, split into sibling files for readability.
Every family file must have an explicit `//go:build linux` tag (filename
alone doesn't auto-tag for families; only for GOOS):

```
package_mgr/
  package_mgr.go
  linux.go              # dispatcher — switches on Family, calls collectDebian / collectRhel / ...
  linux_debian.go       # //go:build linux — debian-specific collector logic
  linux_rhel.go         # //go:build linux
  linux_suse.go         # //go:build linux
  darwin.go
  other.go
```

**Getting `platform.Family` inside a collector.** Our `Collect(ctx) (any,
error)` signature doesn't receive prior collector results. Two options:

1. **Re-detect locally** (simple — call gopsutil's `host.Info()` or parse
   `/etc/os-release`). gopsutil caches so the extra call is cheap. Default
   to this.
2. **Change the Collector interface** to accept prior results. Avoid unless
   we have a concrete perf reason — it's an API break.

**`linux.go`** (build-tagged `//go:build linux`):

```go
// (MIT header)
// Use a testable core that accepts the library function as a parameter
// so tests can stub the system call.
func collect(
    ctx context.Context,
) (any, error) {
    return collectWithInfo(ctx, cpu.InfoWithContext)
}

func collectWithInfo(
    ctx context.Context,
    fn func(context.Context) ([]cpu.InfoStat, error),
) (any, error) {
    // wrap library, return *Info
}
```

**`darwin.go`** — same shape, possibly same code if the library is
cross-platform. **Always keep `darwin.go` and `linux.go` as separate
files** even when code is byte-identical. Uniform layout across collectors
is more valuable than saving ~30 lines of duplication.

**`other.go`** (build-tagged `//go:build !linux && !darwin`):

```go
func collect(
    _ context.Context,
) (any, error) {
    return nil, nil
}
```

### Step 4: Register in `pkg/gohai/gohai.go`

Add to the `builtinCollectors()` slice:

```go
func builtinCollectors() []collector.Collector {
    return []collector.Collector{
        platform.New(),
        cpu.New(),  // <-- add here
    }
}
```

Import the sub-package at the top of `gohai.go`.

### Step 5: Tests — 100% coverage required

**`linux_export_test.go` / `darwin_export_test.go`** — expose private
symbols (usually `Collect` and `CollectWithInfo`) so public tests can
exercise them:

```go
//go:build linux

// (MIT header)
package cpu

var (
    Collect         = collect
    CollectWithInfo = collectWithInfo
)
```

**`<name>_public_test.go`** — package-level public tests
(`package cpu_test`):

- Test `New()` returns the right Name, Tier, Dependencies
- Test `Collect()` satisfies the `collector.Collector` interface
- Test the real `Collect()` runs without error on the current OS

**`linux_public_test.go` / `darwin_public_test.go`** — OS-specific public
tests that stub the library call via the exported `CollectWithInfo`:

- Table-driven scenarios covering happy path + error path
- Happy path returns the expected `*Info` with field values mapped correctly
- Error path (stub returns `error`) confirms error is propagated

**Coverage target: 100%.** Run:

```bash
go test ./... -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out | grep -v '100.0%'
```

Should return nothing (except `main`/`cmd` which are `.coverignore`d).

### Step 6: Update the collector doc

Edit `docs/collectors/<name>.md` (already a stub). Fill in:

- Status: Implemented ✅
- Description
- Collected Fields table
- Platform Support table (sources used on each OS)
- Example Output (JSON) for Linux and macOS
- SDK Usage snippet
- Enable/Disable flags
- Dependencies
- Backing library + license

Use `docs/collectors/platform.md` as the template.

### Step 7: Update README.md

Flip the "Implemented" column from 🚧 to ✅ (plus the library name if you
wrapped one, e.g., `✅ (gopsutil)`).

### Step 8: Verify

```bash
go build ./...                                                       # compiles
go test ./... -count=1                                               # tests pass
go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run   # 0 issues
go run . --collector.<name> --pretty                                 # CLI works
```

### Step 9: Commit

```
feat(<name>): add <name> collector

Wraps <library> to collect <what>. Supports Linux and macOS.
```

[Chef Ohai]: https://docs.chef.io/ohai/
[OSAPI]: https://github.com/osapi-io/osapi
[gopsutil]: https://github.com/shirou/gopsutil
[ghw]: https://github.com/jaypipes/ghw
[procfs]: https://github.com/prometheus/procfs
[go-sysinfo]: https://github.com/elastic/go-sysinfo
[node_exporter]: https://github.com/prometheus/node_exporter
[ohai-plugins]: https://github.com/chef/ohai/tree/main/lib/ohai/plugins
