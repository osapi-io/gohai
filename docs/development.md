# Development

This guide covers the tools, setup, and conventions needed to work on gohai.

## Prerequisites

Install tools using [mise][]:

```bash
mise install
```

- **[Go][]** — gohai is written in Go. We always support the latest two major Go
  versions, so make sure your version is recent enough.
- **[just][]** — Task runner used for building, testing, formatting, and other
  development workflows. Install with `brew install just`.

### Claude Code

If you use [Claude Code][] for development, install these plugins from the default
marketplace:

```
/plugin install commit-commands@claude-plugins-official
/plugin install superpowers@claude-plugins-official
```

- **commit-commands** — provides `/commit` and `/commit-push-pr` slash commands
  that follow the project's commit conventions automatically.
- **superpowers** — provides structured workflows for planning, TDD, debugging,
  code review, and git worktree isolation.

## Project Structure

```
cmd/gohai/                 # CLI entrypoint (Cobra)
pkg/gohai/                 # Public SDK (Gohai struct, Facts, options)
  gohai.go                 # New(), Collect(), functional options
  facts.go                 # Typed Facts struct with all collector results
  options.go               # WithCollectors, WithProfile, etc.
  registry.go              # Collector registry (enable/disable)
internal/collector/        # Collector implementations (one sub-package each)
  collector.go             # Collector interface definition
  registry.go              # Internal registry mechanics
  platform/                # OS/platform detection
  cpu/                     # CPU information
  memory/                  # Memory information
  ...                      # One package per collector
docs/
  collectors/              # Per-collector reference (one doc per collector)
    README.md              # Master index with tier tables
    platform.md            # Individual collector reference
    ...
```

### Adding a new collector

See the [Adding a New Collector](../CLAUDE.md#adding-a-new-collector) guide in
CLAUDE.md for the full checklist: types, implementation, registration, tests,
docs, and README update.

### Implementation methodology

gohai is an **SDK first** — a library consumed by [OSAPI][osapi] and other Go
services. The CLI is a thin wrapper. Design choices should optimize for the
importer.

Each collector **wraps** a well-maintained backing source rather than
reimplementing OS parsing. Decision order:

1. Prefer an existing Go library ([gopsutil][], [ghw][], [procfs][],
   [go-sysinfo][]).
2. Prefer an official provider SDK for cloud collectors, or `net/http` to IMDS
   endpoints.
3. Composite approach combining multiple sources.
4. Roll our own thin parser for simple data (one file, one command).
5. Fall back to porting [Ohai's Ruby plugin][ohai-plugins] when no Go library
   covers the domain.

Use [node_exporter][] as a reference for tricky Linux `/proc`/`/sys` parsing —
read, learn, rewrite in our style (don't import their code directly).

Whatever backing strategy you pick, shape the output into the collector's typed
`Info` struct so the JSON output stays Ohai-compatible.

## Setup

Fetch shared justfiles and install all dependencies:

```bash
just fetch
just deps
```

## Code style

Go code should be formatted by [`gofumpt`][gofumpt] and linted using
[`golangci-lint`][golangci-lint]. This style is enforced by CI.

```bash
just go::fmt-check   # Check formatting
just go::fmt         # Auto-fix formatting
just go::vet         # Run linter
```

### Documentation

Markdown files are formatted with [Prettier][prettier] via Bun. This style is
enforced by CI.

```bash
just docs::fmt-check   # Check formatting
just docs::fmt         # Auto-fix formatting
```

## Testing

```bash
just test           # Run all tests (lint + unit + coverage)
just go::unit       # Run unit tests only
just go::unit-cov   # Generate coverage report
go test -run TestName -v ./internal/collector/platform/...  # Run a single test
```

### Test file conventions

- Public tests: `*_public_test.go` in test package (`package platform_test`) for
  exported functions.
- `export_test.go` in the same package exposes unexported symbols to external
  `_test.go` files via typed aliases (`var ReadX = readX`) and setter functions
  (`SetXFn(fn) func()` returning a restore func the caller defers). Never put
  production-only code in `export_test.go`.
- Use `testify/suite` with table-driven patterns.
- Table-driven structure with `validateFunc` callbacks.
- **One suite method per function under test.** All scenarios for a function
  (success, error codes, edge cases) belong as rows in a single table — never
  split into separate `TestFoo`, `TestFooError`, `TestFooNilResponse` methods.
- **No custom assertion messages** — `s.Equal(want, got)`, not
  `s.Equal(want, got, "expected equal")`. Matches osapi's style.
- Target 100% coverage on all packages.
- **Don't coverage-chase trivial bridges.** If a wrapper is a single gopsutil
  call, make the call itself swappable via private var + `Set<X>Fn` setter in
  `export_test.go`; the error-branch test then lives naturally as a table row
  exercising both success and error paths. See `pkg/gohai/collectors/load/` for
  the canonical shape.

### Upcoming: VFS + Executor (Phase 1, WIP)

The per-field `Fn` injection pattern is being replaced by shared
`vfs.Filesystem` and `executor.Executor` abstractions (avfs + gomock-backed)
threaded through `Collect` the same way `context.Context` is. Until Phase 1
lands, use `export_test.go` + private var + setter for gopsutil / syscall /
file-read seams. Do NOT add new `Fn` struct fields for new seams. See the
"Upcoming: VFS + Executor Abstractions" section in CLAUDE.md.

## Before committing

Run `just ready` before committing to ensure generated code, package docs,
formatting, and lint are all up to date:

```bash
just ready   # generate, docs::fmt, go::fmt, go::vet
```

## Branching

All changes should be developed on feature branches. Create a branch from `main`
using the naming convention `type/short-description`, where `type` matches the
[Conventional Commits][] type:

- `feat/add-cpu-collector`
- `fix/memory-parsing-error`
- `docs/update-collector-reference`
- `refactor/simplify-registry`
- `chore/update-dependencies`

When using Claude Code's `/commit` command, a branch will be created
automatically if you are on `main`.

## Commit messages

Follow [Conventional Commits][] with the 50/72 rule:

- **Subject line**: max 50 characters, imperative mood, capitalized, no period
- **Body**: wrap at 72 characters, separated from subject by a blank line
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
- Summarize the "what" and "why", not the "how"

Try to write meaningful commit messages and avoid having too many commits on a
PR. Most PRs should likely have a single commit (although for bigger PRs it may
be reasonable to split it in a few). Git squash and rebase is your friend!

[mise]: https://mise.jdx.dev
[Go]: https://go.dev
[just]: https://just.systems
[Claude Code]: https://claude.ai/code
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[Conventional Commits]: https://www.conventionalcommits.org
[prettier]: https://prettier.io
[osapi]: https://github.com/osapi-io/osapi
[gopsutil]: https://github.com/shirou/gopsutil
[ghw]: https://github.com/jaypipes/ghw
[procfs]: https://github.com/prometheus/procfs
[go-sysinfo]: https://github.com/elastic/go-sysinfo
[node_exporter]: https://github.com/prometheus/node_exporter
[ohai-plugins]: https://github.com/chef/ohai/tree/main/lib/ohai/plugins
