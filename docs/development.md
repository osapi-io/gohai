# Development

This guide covers the tools, setup, and conventions needed to work on gohai.

## Prerequisites

Install tools using [mise][]:

```bash
mise install
```

- **[Go][]** — gohai is written in Go. We always support the latest two major
  Go versions, so make sure your version is recent enough.
- **[just][]** — Task runner used for building, testing, formatting, and other
  development workflows. Install with `brew install just`.

### Claude Code

If you use [Claude Code][] for development, install these plugins from the
default marketplace:

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

- Public tests: `*_public_test.go` in test package
  (`package platform_test`) for exported functions.
- Use `testify/suite` with table-driven patterns.
- Table-driven structure with `validateFunc` callbacks.
- **One suite method per function under test.** All scenarios for a function
  (success, error codes, edge cases) belong as rows in a single table — never
  split into separate `TestFoo`, `TestFooError`, `TestFooNilResponse` methods.
- Target 100% coverage on all packages.

## Before committing

Run `just ready` before committing to ensure generated code, package docs,
formatting, and lint are all up to date:

```bash
just ready   # generate, docs::fmt, go::fmt, go::vet
```

## Branching

All changes should be developed on feature branches. Create a branch from
`main` using the naming convention `type/short-description`, where `type`
matches the [Conventional Commits][] type:

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
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
  `chore`
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
