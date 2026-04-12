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
and Ohai-compatible JSON shape consumers expect.

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
  - `linux.go` / `darwin.go` / `other.go` — build-tagged OS implementations
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

**`darwin.go`** — same shape, possibly same call if the library is
cross-platform. Factor out a shared `unix.go` if linux and darwin do the
same thing.

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
