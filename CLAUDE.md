# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

Go-based system information collector inspired by [Chef Ohai][] — collects
comprehensive system facts via a pluggable collector architecture with
node_exporter-style flag toggling. Usable as a standalone CLI or as a Go
library/SDK for integration with [OSAPI][].

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

- **`cmd/gohai/`** — CLI entrypoint
  - Cobra-based CLI with node_exporter-style `--collector.<name>` /
    `--no-collector.<name>` flags
- **`pkg/gohai/`** — Public SDK
  - `Gohai` struct, `New()`, `Collect()`, functional options
  - Typed result structs (`Facts`, per-collector info types)
  - Registry for collector enable/disable
- **`internal/collector/`** — Collector implementations (not exported)
  - `Collector` interface definition
  - One sub-package per collector (`platform/`, `cpu/`, `memory/`, etc.)
  - Each collector returns a strongly-typed struct

## Code Standards (MANDATORY)

### Function Signatures

ALL function signatures MUST use multi-line format:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
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

When adding a new collector (e.g., `gpu`), follow these steps in order.
Every collector must ship with tests, docs, and typed structs.

### Step 1: Collector Types

Create `internal/collector/gpu/gpu.go` with the typed result struct and
`Collector` interface implementation:

```go
package gpu

// Info holds GPU information collected from the system.
type Info struct {
    Devices []Device `json:"devices"`
}

// Device represents a single GPU.
type Device struct {
    Model  string `json:"model"`
    Driver string `json:"driver"`
    Memory string `json:"memory"`
}
```

### Step 2: Collector Implementation

Implement the `Collector` interface:

```go
// Collector collects GPU information.
type Collector struct{}

// Name returns the collector name.
func (c *Collector) Name() string {
    return "gpu"
}

// Collect gathers GPU facts from the system.
func (c *Collector) Collect(
    ctx context.Context,
) (*Info, error) {
    // Platform-specific implementation
}
```

### Step 3: Register the Collector

Add the collector to the registry in `internal/collector/registry.go`
with its default-enabled/disabled state and tier.

### Step 4: Add Facts Field

Add the typed field to `pkg/gohai/facts.go`:

```go
type Facts struct {
    // ...existing fields...
    GPU *gpu.Info `json:"gpu,omitempty"`
}
```

### Step 5: Tests

Two test files:

**`internal/collector/gpu/gpu_test.go`** — internal tests for collection
logic, parsing, edge cases.

**`internal/collector/gpu/gpu_public_test.go`** — public tests verifying
the collector interface contract.

Target 100% coverage on both files.

### Step 6: Collector Doc

Create `docs/collectors/gpu.md` with:

- Description
- Collected fields table
- Platform support (Linux, macOS)
- Example output (JSON)
- Enable/disable flags

### Step 7: Update README.md

Add the collector to the appropriate tier table in README.md.

### Step 8: Verify

```bash
go build ./...                    # compiles
go test ./... -count=1            # tests pass
go run ./cmd/gohai                # CLI works
```

[Chef Ohai]: https://docs.chef.io/ohai/
[OSAPI]: https://github.com/osapi-io/osapi
