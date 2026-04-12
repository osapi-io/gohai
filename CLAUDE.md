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
struct.

### Extend upstream, don't replace

**MANDATORY:** When a Go library (gopsutil, ghw, procfs, cloud SDKs) covers
part of what a collector needs, use it for that part. If the library doesn't
cover everything we want, add the extension logic **in our collector** on
top of the library's output. Do not replace the library wholesale just
because it's missing one piece — you lose years of accumulated bug fixes
and cross-platform handling that way.

Extension pattern:

```go
// Wrap the library for what it does well.
info, err := upstream.Get(ctx)
if err != nil { return nil, err }

// Layer our extension on top for the gap.
if shouldAddOurBit(info) {
    info.OurField = readOurSource()
}
```

Only replace the library entirely when its scope genuinely doesn't match
what the collector is supposed to report (e.g. a library mixes sources in
a way that produces an ambiguous value, or its output isn't reshapeable
into the right semantics). When you do replace, justify the decision in
the collector's Data Sources doc.

**Decision order for each collector:**

1. **Prefer a well-maintained Go library** — saves time, handles edge cases.
   Standard picks: [gopsutil][] (CPU/memory/disk/network/host/process),
   [ghw][] (hardware detail), [procfs][] (raw `/proc`, `/sys`),
   [go-sysinfo][] (alternative host info). **Extend if needed, don't
   replace.**
2. **Prefer official provider SDKs** for cloud collectors (aws-sdk-go,
   google.golang.org/cloud, azure-sdk-for-go) or plain `net/http` to IMDS
   endpoints for smaller binaries.
3. **Composite approach** when needed — combine multiple sources (library
   + our own direct reads for fields the library doesn't expose).
4. **Roll our own thin parser** when the data is simple (single file or
   command) and no library covers the domain.
5. **Fall back to porting [Ohai's Ruby plugin][ohai-plugins]** when no Go
   library covers the domain. Ohai has solved the edge cases for every major
   OS/distro; our job is translate, not re-discover.

**We learn from, but don't directly import, [node_exporter][]** — their
collectors are a gold reference for tricky Linux `/proc` and `/sys` parsing
(Apache-2 licensed). Read, understand, rewrite in our style.

### Cross-platform compilation — no build tags (osapi pattern)

**MANDATORY:** Collector code must compile on every target platform,
with **no `//go:build` tags anywhere**. This is the pattern OSAPI uses
in `internal/provider/` — study that code before writing a new
collector. Result: `go test ./...` on any dev machine compiles and
runs every collector's tests, coverage is visible cross-platform, and
CI on linux runners still validates actual linux runtime behavior.

The shape of a collector:

```
pkg/gohai/collectors/<name>/
  <name>.go         # Info struct, Collector interface, New() factory
  linux.go          # type Linux struct {...}; implements Collector
  darwin.go         # type Darwin struct {...}; implements Collector
  debian.go         # (only when debian diverges from generic linux)
  <name>_public_test.go
  linux_public_test.go
  darwin_public_test.go
```

**Per-OS struct pattern:**

```go
// linux.go — NO build tag
package machineid

type Linux struct {
    ReadFileFn func(string) ([]byte, error) // injectable for tests
}

func NewLinux() *Linux {
    return &Linux{ReadFileFn: os.ReadFile}
}

func (l *Linux) Collect(ctx context.Context) (any, error) {
    // Linux-specific logic. Reads /proc paths, etc.
    // On a non-linux host these reads fail gracefully (ErrNotExist)
    // but the code still compiles — which is what matters.
}
```

```go
// darwin.go — NO build tag
package machineid

type Darwin struct {
    HostInfoFn func(context.Context) (*host.InfoStat, error)
}

func NewDarwin() *Darwin {
    return &Darwin{HostInfoFn: host.InfoWithContext}
}

func (d *Darwin) Collect(ctx context.Context) (any, error) {
    // Darwin-specific logic.
}
```

**Single-entry factory** in `<name>.go` dispatches on
`platform.Detect()`:

```go
func New() collector.Collector {
    switch platform.Detect() {
    case "debian":
        return NewDebian()
    case "darwin":
        return NewDarwin()
    default:
        return NewLinux()
    }
}
```

`platform.Detect()` (add a new `internal/platform` package mirroring
OSAPI's `pkg/sdk/platform`) wraps gopsutil's `host.Info` and returns
`"debian"` / `"darwin"` / empty-string (generic linux).

**Key rules:**

- Every struct must compile on every platform. Use cross-platform APIs
  only: stdlib, gopsutil, `golang.org/x/sys/unix` (per-OS layouts but
  compiles everywhere), ghw, cloud SDKs. No raw `syscall.Utsname` etc.
- Missing OS-specific paths (e.g. `/proc/modules` on darwin) return
  empty gracefully — never error.
- Add a `Debian` variant (or `RHEL`, `SUSE`, etc.) **only** when that
  distro family genuinely diverges. Otherwise generic `Linux` covers
  all non-darwin.
- Dependency-inject file readers, command runners, and gopsutil calls
  via struct fields — lets tests exercise every branch without
  touching the real host.
- **NEVER leak third-party types through public `Fn` fields.** Per-OS
  struct `Fn` fields MUST be typed in our `*Info` / `[]OurType`, never
  in gopsutil / ghw / procfs types. The upstream call lives in a
  private package var (`var hostInfoFn = host.InfoWithContext`); tests
  swap it via a `Set<X>Fn` setter declared in `export_test.go`. See
  `pkg/gohai/collectors/uptime/` for the canonical example. Without
  this rule, importing a collector sub-package transitively pulls
  gopsutil into consumers' module graphs — an SDK leak.
- **Upcoming (Phase 1):** the per-field `Fn` pattern will be replaced
  by a shared VFS + Executor abstraction threaded through `Collect`.
  Until that lands, follow the private-var + `export_test.go` pattern
  above. Do NOT expand the old `Fn` field pattern for new code — if
  you need a new seam, check the Phase 1 status in the "Upcoming:
  VFS + Executor Abstractions" section below.

The Collector interface and `Info` struct shape are the contract — whatever
backing strategy a collector uses, its output must match the typed struct
and consumer expectations.

### Field naming

**OCSF is our data schema.** When adding or renaming a collector field,
**always check [schema.ocsf.io](https://schema.ocsf.io/) first** — OCSF
(Open Cybersecurity Schema Framework, backed by AWS and Splunk) has
canonical names for ~99% of what we collect. Using OCSF names means
gohai output feeds SIEMs, data lakes, and inventory tools without
translation, so the lookup is mandatory, not aspirational.

**Both Go field names AND JSON tags derive from OCSF.** The JSON tag is
the OCSF snake_case path verbatim (`json:"kernel_release"`); the Go
field is the PascalCase rendering of the same path (`KernelRelease
string`). When Go idiom on initialisms conflicts (OCSF `cpu_id` → Go
`CPUID`, not `CpuId`), the Go convention wins the field name but the
JSON tag still matches OCSF. Don't invent internal names that diverge
from OCSF.

Collector JSON field names use `snake_case`. Precedence:

1. **OCSF** — if OCSF has a name for the field, use it. Examples:
   `process.cmd_line` (not `cmdline`), `network_interface.mac` (not
   `hardware_addr`), `os.kernel_release` (not `kernel_version`),
   `device.hostname`, `os.name`, `file.path`. Browse OCSF's
   [objects](https://schema.ocsf.io/objects), [data types][], and
   [dictionary][] to find the right field. When OCSF has a field we
   don't emit yet but easily could (e.g. `device.hw_info.serial_number`,
   `os.build`), consider adding it — they've thought about what a
   consumer wants.
2. **Industry standard** — when OCSF is silent, use whatever
   node_exporter / systemd / Prometheus exporters standardized on.
   Example: filesystem `mountpoint` / `fstype` follow node_exporter.
3. **Ohai's name** — only when OCSF and industry standards are silent
   AND Ohai has a clear, meaningful name.
4. **Our own name** — last resort. Go-idiomatic snake_case.

**Do not mirror Ohai's JSON shape.** Ohai is for **data-source**
reference (what file/command to read, which distro edge cases, which
fallback) — not field names or struct layout. Ruby Mash ↔ Go struct
translation isn't worth pinning byte-for-byte.

Record OCSF alignment in the collector doc's **Data Sources** section:
call out which OCSF object each field maps to, or note "no OCSF
equivalent" with a one-line reason.

[data types]: https://schema.ocsf.io/data_types
[dictionary]: https://schema.ocsf.io/dictionary

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

The Data Sources section is a self-contained spec of HOW the collector
collects data, written in **our voice**. Numbered step-by-step,
per-OS sections when behavior differs. Describe the actual sequence of
reads, fallbacks, distro branches, and error handling. Do NOT frame
it as a parity comparison with Ohai. Example shape:

```md
## Data Sources

On Linux the collector cascades through multiple signals:

1. **Fast path:** if `systemd-detect-virt` is on PATH, call it.
2. **Container-runtime presence:** `which(docker)` / `which(podman)`.
3. **Xen:** `/proc/xen` and `/proc/xen/capabilities`.
4. ...
```

Ohai is mentioned inline only when a specific methodology choice needs
attribution ("we mirror Ohai's legacy `/etc/*-release` fallback chain").
The section is a spec of OUR behavior, not a diff against Ohai.

**`Known gaps vs. Ohai` is NOT a permanent section.** Methodology gaps
live on GitHub as issues labeled `methodology-gap` and `collector:<name>`.
Each issue carries a "Doc after this fix lands" block with the exact
prose the fix PR pastes into the Data Sources section. When all open
methodology issues for a collector close, the doc has zero Ohai residue.
See the "Methodology Work" section below for the full workflow.

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
- **`pkg/gohai/collectors/<name>/`** — Public per-collector sub-packages.
  Use the osapi-style per-OS struct pattern (no build tags). See
  `pkg/gohai/collectors/shells/` for the canonical reference.
  - `<name>.go` — `Info` struct, `Collector` interface, `base` struct
    (holds shared `Name()`/`Tier()`/`Dependencies()`), `New()` factory
    that dispatches on `platform.Detect()`, and any cross-OS helpers
    (shared parsing, shared constants).
  - `linux.go` — `type Linux struct { base; <injectable fns> }` with
    `NewLinux()` and `(l *Linux) Collect(ctx)` method. **No build tag.**
  - `darwin.go` — `type Darwin struct { base; <injectable fns> }` with
    `NewDarwin()` and `(d *Darwin) Collect(ctx)` method. **No build tag.**
  - `debian.go` / `rhel.go` (only when distro genuinely diverges) — same
    pattern, added to the `New()` dispatch switch.
  - `<name>_public_test.go` — tests New()/base methods + `TestNewDispatch`
    that swaps `platform.Detect` to exercise every dispatch branch
    regardless of host OS. **Never import gopsutil in these tests.**
  - `linux_public_test.go` — tests the `Linux` struct's `Collect` with
    injected stubs. **No build tag.**
  - `darwin_public_test.go` — tests the `Darwin` struct's `Collect`. **No build tag.**
- **`internal/platform/`** — OS/distro detection wrapping gopsutil.
  `Detect()` is a swappable `var` so collector tests can force any
  branch without importing gopsutil. `hostInfoFn` is private, exposed
  only to platform's own tests via `export_test.go`.
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
- Internal tests: `*_test.go` in same package (`package gohai`) for
  private functions — avoid when the external package can reach what
  it needs via an `export_test.go` alias
- `export_test.go` in the same package exposes unexported symbols to
  external `_test.go` files via typed aliases (`var ReadX = readX`)
  and setter functions (`SetXFn(fn) func()` returning a restore func
  the caller defers). Never put production-only code in
  `export_test.go`; the `_test.go` suffix makes it test-only
- Suite naming: `*_public_test.go` → `{Name}PublicTestSuite`,
  `*_test.go` → `{Name}TestSuite`
- Use `testify/suite` with table-driven patterns
- One suite method per function under test — all scenarios (success, errors,
  edge cases) as rows in one table
- **No custom assertion messages** — `s.Equal(want, got)`, not
  `s.Equal(want, got, "expected equal")`. Matches osapi's test style
- Target 100% test coverage on all packages
- **Don't coverage-chase trivial bridges.** If the wrapper is a single
  gopsutil call, make the upstream call itself swappable via private
  package var + `Set<X>Fn` setter in `export_test.go`; the error-branch
  test then lives naturally as a table row exercising both success and
  error paths. See `pkg/gohai/collectors/load/` for the canonical shape

### Go Patterns

- Error wrapping: `fmt.Errorf("context: %w", err)`. Wrap upstream
  library errors with context — **never expose raw gopsutil / ghw /
  procfs error types through our API**. Callers must never need those
  packages in their module graph to handle errors
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

**Reference implementation:** `pkg/gohai/collectors/shells/`. Copy its
file layout and patterns exactly.

### Done-definition (every collector, every time)

Before marking a collector complete, every item below must be true:

1. **Analyzed Ohai's plugin + spec** for HOW it collects (data sources,
   distro edge cases, fallback chains). Our collection logic mirrors
   theirs — we inherit their years of bug fixes. Deviations are
   documented and justified.
2. **Checked OCSF schema** ([schema.ocsf.io](https://schema.ocsf.io/))
   for canonical field names. OCSF mappings recorded in the collector
   doc's Collected Fields table. When OCSF has a field we could emit
   but don't, either add it or note why.
3. **osapi per-OS struct pattern** — no build tags, factory dispatch
   on `platform.Detect()`, per-OS structs each implementing Collect.
4. **100% test coverage.** `go tool cover -func=/tmp/cov.out | grep -v '100.0%'`
   returns nothing for the collector's files.
5. **Collector tests do NOT import gopsutil** — stub `platform.Detect`
   directly. Compile-time enforcement.
6. **`docs/collectors/<name>.md`** is a self-contained functional
   spec: Description (what + why in our voice), Collected Fields with
   OCSF mapping column, Platform Support, Example Output, SDK Usage,
   Enable/Disable, Dependencies, Data Sources (step-by-step
   methodology in OUR voice — not a Ohai parity table), Backing
   library. **No "Known gaps vs. Ohai" section** — methodology gaps
   live as GitHub issues (labeled `methodology-gap` /
   `collector:<name>`).
7. **README.md** row flipped to `✅ (<backing>)`.
8. **Lint clean**, `just go::vet` returns 0 issues.
9. **Commit message** explains the "why" — what Ohai/OCSF
   cross-references drove the implementation, what extensions over the
   upstream library we added, any deliberate deviations.
10. **Check GitHub issues** for tracked methodology gaps:
    `gh issue list --label methodology-gap --label collector:<name>`.
    If the work closes a tracked issue, the issue's "Doc after this
    fix lands" block IS the doc content to paste into Data Sources.
    The PR description must include `Closes #N`.

### Step 1: Create the sub-package

Path: `pkg/gohai/collectors/<name>/` (public — consumers like OSAPI
import `Info` structs directly).

### Step 2: `<name>.go` — top-level factory + shared surface

```go
// (MIT header)
// Package cpu collects CPU topology and feature facts.
package cpu

import (
    "context"

    "github.com/osapi-io/gohai/internal/collector"
    "github.com/osapi-io/gohai/internal/platform"
)

// Info holds CPU information. Field names follow OCSF where applicable.
type Info struct {
    Total     int      `json:"total"`                // OCSF: device.cpu_count
    Cores     int      `json:"cores"`                // OCSF: device.cpu_cores
    ModelName string   `json:"model_name,omitempty"` // no OCSF
    Flags     []string `json:"flags,omitempty"`      // no OCSF
}

// Collector is the public interface every cpu variant satisfies.
type Collector interface {
    collector.Collector
}

// base holds the identity (Name/Tier/Dependencies) common to every
// per-OS variant. Embedded in Linux, Darwin, Debian, etc.
type base struct{}

func (base) Name() string                   { return "cpu" }
func (base) Tier() collector.Tier           { return collector.TierCore }
func (base) Dependencies() []string         { return nil }

// New returns the cpu variant appropriate to the host OS.
func New() Collector {
    switch platform.Detect() {
    case "darwin":
        return NewDarwin()
    case "debian":
        return NewDebian() // only if debian diverges; else drop this case
    case "rhel":
        return NewRHEL()   // only if rhel diverges; else drop this case
    default:
        return NewLinux()
    }
}

// Cross-OS helpers (shared parsers, shared constants) live here too.
```

### Step 3: Per-OS struct implementations (no build tags)

Each file declares a struct for that OS, embeds `base`, and implements
`Collect`. Injectable fields (function types) make tests independent of
the host OS.

**`linux.go`:**

```go
// (MIT header) — NO //go:build tag

package cpu

import (
    "context"

    "github.com/shirou/gopsutil/v4/cpu"
)

// Linux collects CPU facts on generic Linux hosts.
type Linux struct {
    base

    // InfoFn is gopsutil's cpu.InfoWithContext. Injectable for tests.
    InfoFn func(context.Context) ([]cpu.InfoStat, error)
}

// NewLinux returns a Linux variant wired to gopsutil.
func NewLinux() *Linux {
    return &Linux{InfoFn: cpu.InfoWithContext}
}

// Collect wraps gopsutil and extends with any fields the library lacks.
// See "Implementation Methodology — Extend upstream, don't replace".
func (l *Linux) Collect(ctx context.Context) (any, error) {
    stats, err := l.InfoFn(ctx)
    if err != nil { return nil, err }
    info := &Info{ /* mapped from stats */ }

    // Extension: add fields gopsutil doesn't expose (e.g., vulnerabilities
    // map from /sys/devices/system/cpu/vulnerabilities/).
    info.Vulnerabilities = readVulnerabilities()

    return info, nil
}
```

**`darwin.go`** — same pattern, `type Darwin struct`, `NewDarwin()`, its
own `Collect`. No build tag. Runtime gracefully handles Linux-only paths
(they just don't exist on darwin).

**`debian.go` / `rhel.go`** — add **only** when the distro genuinely
diverges from generic Linux (e.g., `package_mgr` needs apt vs dnf).
Otherwise the generic `Linux` variant covers all non-darwin.

### Step 4: Register in `pkg/gohai/gohai.go`

Add to the `builtinCollectors()` slice:

```go
func builtinCollectors() []collector.Collector {
    return []collector.Collector{
        platform.New(),
        cpu.New(), // <-- add here
    }
}
```

### Step 5: Tests — 100% coverage, no gopsutil leaks

**`<name>_public_test.go`** — package-level tests:

- `TestNew` → calls `New()`, asserts `Name()`/`Tier()`/`Dependencies()`
  on the returned interface.
- `TestImplementsCollectorInterface` → compile-time check that every
  per-OS struct satisfies `collector.Collector`.
- `TestNewDispatch` → stubs `platform.Detect` (a swappable `var`) to
  force every branch (darwin/debian/rhel/linux-generic) and asserts the
  factory returns the right concrete type. **No gopsutil import.**

**`linux_public_test.go`** — tests the `Linux` struct directly with
injected stubs:

```go
c := &cpu.Linux{InfoFn: fakeInfoFn}
got, err := c.Collect(context.Background())
```

Table-driven scenarios covering happy path, error path, edge cases.

**`darwin_public_test.go`** — same for `Darwin` struct.

No build tags on any test file. `go test ./...` on any dev OS runs
every test.

**Coverage check:**

```bash
go test ./... -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out | grep -v '100.0%'
```

### Step 6: Update the collector doc

`docs/collectors/<name>.md` must contain, in order:

- `# <Name>` and **Status:** Implemented ✅
- **Description** — what the fact is, why consumers want it, typical
  values on Linux vs macOS.
- **Signals** (only for complex collectors where multiple fields
  answer different questions — see `docs/collectors/fips.md` for
  reference).
- **Collected Fields** — markdown table with columns: Field, Type,
  Description, **OCSF mapping**. Every field gets an OCSF row
  (`os.kernel_release`, `network_interface.mac`, etc.) or explicit
  "No OCSF equivalent" with one-line reason.
- **Platform Support** — table.
- **Example Output** — realistic JSON for Linux and macOS.
- **SDK Usage** — Go snippet using `gohai.New(...).Collect(ctx)`.
- **Enable/Disable** — CLI flags.
- **Dependencies** — other collectors.
- **Data Sources** — table: Platform | What we read | Ohai plugin +
  what it reads | Alignment. Plus a "Known gaps" line.
- **Backing library** — the Go library (or "stdlib") we wrap.

Use `docs/collectors/shells.md` as the canonical template.

### Step 7: Update README.md

Flip the "Implemented" column from 🚧 to ✅ (with the library in
parentheses: `✅ (gopsutil)`, `✅ (stdlib)`).

### Step 8: Verify

```bash
go build ./...
go test ./... -count=1
just go::vet
go run . --collector.<name> --pretty
```

### Step 9: Commit

```
feat(<name>): add <name> collector

Wraps <library> for the core collection, extending with <X> that the
library doesn't expose (mirrors Ohai's <Y> behavior). Field names
follow OCSF conventions where applicable (<list>). Supports Linux and
macOS with platform.Detect() dispatch.
```

## Methodology Work

Methodology gaps between gohai and Ohai live on GitHub as issues
labeled `methodology-gap` and `collector:<name>`. See
`gh issue list --label methodology-gap`. Each issue carries:

- Full Ohai methodology breakdown, source-cited with file + line ranges.
- Our current implementation and what it misses.
- Risk / severity / which hosts fail.
- Proposed fix — concrete code plan.
- Acceptance criteria.
- **"Doc after this fix lands"** — the exact prose (Description +
  Collected Fields table + Data Sources) the fix PR pastes into the
  collector's `docs/collectors/<name>.md`.

**Workflow when working a methodology issue:**

1. `gh issue view <N>` — read end to end, especially the "Doc after
   this fix lands" block.
2. Implement the code change per "Proposed fix" — use the VFS /
   Executor abstractions if Phase 1 has landed, otherwise the
   `export_test.go` + private var + `Set<X>Fn` pattern.
3. Paste the issue's "Doc after this fix lands" block into the
   collector doc, replacing Description / Collected Fields / Data
   Sources as specified.
4. PR description must include `Closes #N`.
5. CI green, 100% coverage, `just go::vet` clean.

When every open methodology issue closes, every collector doc reads
as a self-contained spec and the SDK has zero unresolved methodology
divergences from Ohai.

## Upcoming: VFS + Executor Abstractions (Phase 1, WIP)

The current per-field `Fn` pattern (`ReadFileFn`, `RunCmdFn`,
`HostInfoFn`, ...) is being replaced by two shared abstractions
threaded through `Collect` the same way `context.Context` is threaded:

- **`vfs.Filesystem`** — virtual filesystem (avfs-backed) for all
  file reads. Production: real OS filesystem. Tests: in-memory FS
  with canned files at their real absolute paths (`/proc/meminfo`,
  `/etc/os-release`). Lets tests exercise real `os.ReadFile` code
  paths against controlled file content.
- **`executor.Executor`** — interface for shell-out calls.
  Production: wraps `exec.CommandContext`. Tests: gomock-generated
  mock asserting command name, flags, and stdin; returns canned
  stdout. Mirrors osapi's executor pattern.

Target signature once adopted:

```go
Collect(ctx context.Context, fs vfs.Filesystem, exec executor.Executor) (any, error)
```

**Until Phase 1 lands:** use the `export_test.go` + private var +
`Set<X>Fn` setter pattern for gopsutil / syscall / file-read seams.
Do NOT add new per-field `Fn` struct fields; if you need a new seam,
check the Phase 1 status before expanding the old pattern.

Reference: osapi's executor / filesystem / mockgen setup at
[github.com/osapi-io/osapi](https://github.com/osapi-io/osapi) —
gohai adopts the same shapes so patterns transfer.

[Chef Ohai]: https://docs.chef.io/ohai/
[OSAPI]: https://github.com/osapi-io/osapi
[gopsutil]: https://github.com/shirou/gopsutil
[ghw]: https://github.com/jaypipes/ghw
[procfs]: https://github.com/prometheus/procfs
[go-sysinfo]: https://github.com/elastic/go-sysinfo
[node_exporter]: https://github.com/prometheus/node_exporter
[ohai-plugins]: https://github.com/chef/ohai/tree/main/lib/ohai/plugins
