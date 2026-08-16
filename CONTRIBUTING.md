# Contributing

Contributions to gohai are very welcome, but we ask that you read this document
before submitting a PR. It covers everything you need: prerequisites, setup, the
conventions code follows, and the pull request workflow.

The collector methodology — which library a collector wraps, what its fields are
called, and how data sources are chosen — is reference material in
[docs/methodology.md](docs/methodology.md).

## Before you start

- Read the [Code of Conduct](CODE_OF_CONDUCT.md). It applies to every
  interaction in this repo.

- **Check existing work** — Is there an existing PR? Are there issues discussing
  the feature/change you want to make? Please make sure you consider/address
  these discussions in your work.

- **Backwards compatibility** — Will your change break existing consumers of
  gohai? It is much more likely that your change will be merged if it is
  backwards compatible. Is there an approach you can take that maintains this
  compatibility? If not, consider opening an issue first so that API changes can
  be discussed before you invest your time into a PR.

## Prerequisites

Install tools using [mise](https://mise.jdx.dev):

```bash
mise install
```

- **[Go]** — gohai is written in Go. We always support the latest two major Go
  versions, so make sure your version is recent enough.
- **[just]** — Task runner used for building, testing, formatting, and other
  development workflows. Install with `brew install just`.
- **[uv](https://docs.astral.sh/uv/)** — Python package runner. `just md-fmt`
  formats markdown with mdformat through `uvx`; nothing is installed into the
  repository.

### Claude Code

If you use [Claude Code] for development, install these plugins from the default
marketplace:

```
/plugin install commit-commands@claude-plugins-official
/plugin install superpowers@claude-plugins-official
```

- **commit-commands** — provides `/commit` and `/commit-push-pr` slash commands
  that follow the project's commit conventions automatically.
- **superpowers** — provides structured workflows for planning, TDD, debugging,
  code review, and git worktree isolation.

## Setup

Fetch shared justfiles and install all dependencies:

```bash
just fetch
just deps
```

## Project structure

- **`main.go`** — repo-root entry point; just calls `cmd.Execute()`
- **`cmd/`** — Cobra CLI subcommands
  - `root.go` — root command, banner, context setup, `AddCommand` wiring
  - `collect.go` — `gohai collect` — collector flags, SDK wiring, delegates
    output to `internal/cli/`
  - `validate.go` — `gohai validate` — JSON Schema validation against embedded
    schema (stdin or `--file`)
  - `version.go` — `gohai version` — build-time identity via
    `caarlos0/go-version`
- **`internal/cli/`** — CLI output helpers (never imported by `pkg/gohai/`)
  - `theme.go` — maxheadroom palette (`#b4a7d6` lavender accent), `Banner()`,
    role-based color helpers (`Mute`, `Accent`, `OK`, `Err`, `Info`, `Success`,
    `Failure`)
  - `output.go` — `WriteOutput`, `WriteJSON`, `WriteFlat`, `WriteCollectorList`
    — facts formatting for the collect subcommand
- **`pkg/gohai/`** — Public SDK
  - `gohai.go` — `Gohai` struct, `New()`, `Collect()`
  - `facts.go` — `Facts` struct with typed collector fields and JSON/Flat
    methods
  - `options.go` — functional options (`WithEnabled`, `WithDisabled`,
    `WithCollectors`)
  - `registry.go` — `PublicRegistry` used by CLI for flag enumeration
- **`pkg/gohai/collectors/<name>/`** — Public per-collector sub-packages. Use
  the osapi-style per-OS struct pattern (no build tags). See
  `pkg/gohai/collectors/shells/` for the canonical reference.
  - `<name>.go` — `Info` struct, `Collector` interface, `base` struct (holds
    shared `Name()`/`DefaultEnabled()`/`Dependencies()`), `New()` factory that
    dispatches on `platform.Detect()`, and any cross-OS helpers (shared parsing,
    shared constants).
  - `linux.go` —
    `type Linux struct { base; FS avfs.VFS; Exec executor.Executor }` (fields
    only when the collector needs them) with `NewLinux()` and
    `(l *Linux) Collect(ctx)` method. **No build tag.**
  - `darwin.go` — `type Darwin struct { base; FS; Exec }` with `NewDarwin()` and
    `(d *Darwin) Collect(ctx)` method. **No build tag.**
  - `debian.go` / `rhel.go` (only when distro genuinely diverges) — same
    pattern, added to the `New()` dispatch switch.
  - `<name>_public_test.go` — the **only** test file. Contains compile-time
    `collector.Collector` asserts at the top, `TestNew` for the factory
    dispatch, a single table-driven `TestCollect` whose rows carry a
    `variant: "linux" | "darwin"` column and construct the right per-OS struct,
    and optionally separate test methods for genuinely-pure public helpers (e.g.
    `TestHumanDuration`, `TestBytesToString`). No `linux_public_test.go` /
    `darwin_public_test.go` files.
- **`schemas/`** — JSON Schema and field-naming artifacts
  - `gen/` — generator tool (`go run .` reflects `gohai.Facts` into JSON Schema
    via `invopop/jsonschema`); `//go:generate` directive picked up by
    `just generate`
  - `gohai.schema.json` — generated schema (draft 2020-12), committed
  - `schema.go` — `//go:embed` of `gohai.schema.json` for the validate
    subcommand
  - `field-mapping.md` — 950-row per-field tier mapping (OCSF/OTel/ convention)
    with citations
  - `ocsf-gaps.md` — 82 OCSF upstream PR candidates
- **`internal/platform/`** — OS/distro detection wrapping gopsutil. `Detect()`
  is a swappable `var` so collector tests can force any branch without importing
  gopsutil. `hostInfoFn` is private, exposed only to platform's own tests via
  `export_test.go`.
- **`internal/collector/`** — Collector interface + registry plumbing
  - `collector.go` — `Collector` interface
  - `registry.go` — `Registry` (register, resolve deps, run concurrently)
- **`internal/executor/`** — command execution abstraction
  - `executor.go` — `Executor` interface (`Execute(ctx, name, args...)`)
  - `gen/` — gomock mock generation (`go generate`) and the committed mock

## Code style

Go code is formatted by \[`gofumpt`\][gofumpt] and linted using
\[`golangci-lint`\][golangci-lint]. This style is enforced by CI.

```bash
just go-fmt-check   # Check formatting
just go-fmt         # Auto-fix formatting
just go-vet         # Run linter
```

The linters that run are declared in `.golangci.yml`. Read them there rather
than looking for a list here — a copied list goes stale the first time the
configuration changes.

### Documentation

Markdown files are formatted with
[mdformat](https://pypi.org/project/mdformat/), run through `uvx`. This style is
enforced by CI.

```bash
just md-fmt-check   # Check formatting
just md-fmt         # Auto-fix formatting
```

## Code standards

### Function signatures

Functions with parameters use multi-line format — one parameter per line, with
the closing parenthesis and the return types on a line of their own:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
}
```

Functions taking no parameters stay on one line:

```go
func Name() string {
}
```

Adding a parameter then shows as one added line rather than a rewritten
signature.

### File naming

Name a file for what it holds. Avoid `helpers.go`, `utils.go`, and names of that
kind: they describe where code was put rather than what it is, and they
accumulate whatever has no other home.

`types.go` holds only type declarations — structs, interfaces, constants, and
aliases. A function belongs in a file named for what it does.

A test file is named for the production file it tests. Where tests grow too
large to read, split the production file first so each test file keeps a
counterpart, rather than splitting tests away from the file they cover.

### Go patterns

- Error wrapping: `fmt.Errorf("context: %w", err)`, so the chain names each
  layer it passed through and stays inspectable with `errors.Is` and
  `errors.As`.
- Early returns rather than nesting the successful path inside conditionals.
- Unused parameters: rename to `_`.
- Import order: standard library, third party, then local, separated by blank
  lines.

### File headers

Every `.go` file MUST start with the MIT license header — see any existing Go
file in the repo for the exact format. Build-tagged files put `//go:build` on
line 1, blank line, then the header.

### Error wrapping at the module boundary

Never expose raw gopsutil, ghw, or procfs error types through the public API.
Wrap them, so callers do not need those packages in their module graph to handle
an error.

## Testing

```bash
just test           # Run all tests (lint + unit + coverage)
just go-unit       # Run unit tests only
just go-unit-cov   # Generate coverage report
go test -run TestName -v ./internal/collector/platform/...  # Run a single test
```

Coverage is gated at 100%. `just test` fails if total coverage drops below it,
so a change that adds untested code fails locally and in CI:

```bash
just go-unit-cov-check   # Report coverage and fail below the target
```

The target is declared in `.github/codecov.yml` and in the shared `go` justfile
module — change both together.

### Test file conventions

- Public tests: `*_public_test.go` in the package's `_test` package, exercising
  the exported surface. This is the default.
- Internal tests: `*_test.go` in the same package, for what the exported surface
  cannot reach.
- Suite naming: `*_public_test.go` → `{Name}PublicTestSuite`, `*_test.go` →
  `{Name}TestSuite`.
- `testify/suite` with table-driven cases.
- One suite method per function under test — success, errors, and edge cases are
  rows in one table, not separate methods.
- `export_test.go` exposes unexported symbols to external tests, by alias or by
  setter. Do not use an alias to re-cover behavior the caller's own test already
  reaches; a helper with its own contract is what the pattern is for.
- Mocks are generated with `go.uber.org/mock` and committed, never hand-written.
  A double that carries a real implementation — signing with a real key, serving
  real HTTP — is not a mock and does not need generating.

External tests in this repository live in `package gohai_test` or
`package collector_test`, and the setter form is `SetXFn(fn) func()`, returning
a restore func the caller defers.

Collector-specific rules on top of that:

- **One `TestCollect` per collector.** All scenarios — both Linux and Darwin,
  success and error paths — live as rows in one table keyed by a `variant`
  column. No `TestCollectLinux` / `TestCollectDarwin` splits.
- Separate test methods are reserved for genuinely pure public helpers with
  their own contract (`TestHumanDuration`, `TestBytesToString`) — not for
  bridges `Collect` already exercises.
- **Swap at the boundary.** `TestCollect` rows swap the raw upstream library
  call (`hostInfoFn`, `partitionsFn`, ...) so the bridge mapping runs on every
  row.
- **No custom assertion messages** — `s.Equal(want, got)`, not
  `s.Equal(want, got, "expected equal")`.
- Target 100% test coverage on all packages.

## Quick reference

```bash
just fetch / just deps / just test / just go-unit / just go-vet / just go-fmt
gohai collect --pretty             # run default collectors
gohai collect --no-defaults --collector.cpu  # specific collectors
gohai collect --pretty | gohai validate      # validate against schema
gohai version                      # build info
```

## VFS and executor abstractions

Collectors that read files or shell out **MUST** use two shared abstractions,
injected as struct fields on the per-OS variant (same pattern as osapi's Agent
struct).

### `avfs.VFS` — filesystem

[`github.com/avfs/avfs`](https://github.com/avfs/avfs) used directly — no custom
wrapper. Production wires the real OS FS via `osfs.NewWithNoIdm()`; tests wire
`memfs.New()` with canned files at real absolute paths (`/proc/meminfo`,
`/etc/os-release`, etc.). Tests exercise the real `ReadFile` / `Open` / `Stat`
code path against memory-backed content — a genuine integration test of the
collector's FS interaction, not a function-stub swap.

**Per-OS struct shape:**

```go
type Linux struct {
    base
    FS avfs.VFS
}

func NewLinux() *Linux {
    return &Linux{FS: osfs.NewWithNoIdm()}
}

func (l *Linux) Collect(ctx context.Context) (any, error) {
    b, err := l.FS.ReadFile("/etc/shells")
    // ...
}
```

**Test shape:**

```go
f := memfs.New()
_ = f.MkdirAll("/etc", 0o755)          // memfs requires the directory
_ = f.WriteFile("/etc/shells", canned, 0o644)
c := &shells.Linux{FS: f}
got, err := c.Collect(ctx)
```

Reference implementation: `pkg/gohai/collectors/shells/`.

### `executor.Executor` — command execution

`internal/executor` provides a minimal interface (single method:
`Execute(ctx, name, args...) ([]byte, error)`) with a gomock mock at
`internal/executor/gen/`. Production impl wraps `exec.CommandContext` and
returns combined stdout+stderr. Collectors that shell out (sysctl, sw_vers,
lsb_release, loginctl, lscpu, kextstat, etc.) hold the Executor as a struct
field.

**Per-OS struct with both FS and Executor:**

```go
type Darwin struct {
    base
    FS   avfs.VFS
    Exec executor.Executor
}

func NewDarwin() *Darwin {
    return &Darwin{
        FS:   osfs.NewWithNoIdm(),
        Exec: executor.New(),
    }
}
```

**Test shape (gomock):**

```go
ctrl := gomock.NewController(t)
mockExec := mocks.NewMockExecutor(ctrl)
mockExec.EXPECT().
    Execute(gomock.Any(), "sw_vers", "-productVersionExtra").
    Return([]byte("(a)\n"), nil)

c := &platform.Darwin{FS: memfs.New(), Exec: mockExec}
```

Mocks are regenerated via `go generate ./internal/executor/...` and committed.
Pinned tool: `go.uber.org/mock`, the maintained fork of the archived
`golang/mock`.

### Migration status

All new code and new collectors MUST use these abstractions. Existing collectors
still on the legacy `ReadFileFn` / `RunCmdFn` struct-field pattern migrate as
methodology work touches them. Canonical reference:
`pkg/gohai/collectors/shells/` (VFS only), `pkg/gohai/collectors/platform/` (VFS
\+ Executor).

## Adding a new collector

Step-by-step walkthrough lives in
[docs/adding-a-collector.md](docs/adding-a-collector.md) — code examples, file
layout, test setup, and the commit template.

The **reference implementation** is `pkg/gohai/collectors/shells/`. Copy its
patterns exactly.

### Done-definition (every collector, every time)

Before marking a collector complete, every item below must be true:

01. **Analyzed Ohai's plugin + spec** for HOW it collects (data sources, distro
    edge cases, fallback chains). Our collection logic mirrors theirs — we
    inherit their years of bug fixes. Deviations are documented and justified.
02. **Checked OCSF schema** ([schema.ocsf.io](https://schema.ocsf.io/)) and,
    when OCSF is silent, \[OpenTelemetry Resource Semantic
    Conventions\][otel-semconv] for canonical field names. Schema mappings
    recorded in the collector doc's Collected Fields table under the **Schema
    mapping** column. When a schema has a field we could emit but don't, either
    add it or note why.
03. **osapi per-OS struct pattern** — no build tags, factory dispatch on
    `platform.Detect()`, per-OS structs each implementing Collect.
04. **100% test coverage.**
    `go tool cover -func=/tmp/cov.out | grep -v '100.0%'` returns nothing for
    the collector's files.
05. **One `<name>_public_test.go`, one `TestCollect`.** Linux and Darwin
    scenarios share the same table, keyed by a `variant` column. No
    `linux_public_test.go` / `darwin_public_test.go` split files. No `TestReadX`
    methods shadowing bridge code `TestCollect` already exercises. Pure-helper
    public-function tests (e.g. `TestHumanDuration`) are the only legitimate
    extra test methods.
06. **Seams sit at the boundary.** `export_test.go` swaps the upstream library
    call (`hostInfoFn`, `partitionsFn`, etc.), so the bridge mapping runs on
    every row. No `readXFn = readX` wrappers partway through the collector, and
    no alias used to test a bridge `TestCollect` already exercises.
07. **`docs/collectors/<name>.md`** is a self-contained functional spec:
    Description (what + why in our voice), Collected Fields with **Schema
    mapping** column (OCSF path first, OpenTelemetry attribute when OCSF is
    silent), Platform Support, Example Output, SDK Usage, Enable/Disable,
    Dependencies, Data Sources (step-by-step methodology in OUR voice — not a
    Ohai parity table), Backing library. **No "Known gaps vs. Ohai" section** —
    methodology gaps live as GitHub issues (labeled `methodology-gap` /
    `collector:<name>`).
08. **README.md** row flipped to `✅ (<backing>)`.
09. **Lint clean**, `just go-vet` returns 0 issues.
10. **Commit message** explains the "why" — what Ohai/OCSF cross-references
    drove the implementation, what extensions over the upstream library we
    added, any deliberate deviations.
11. **Check GitHub issues** for tracked methodology gaps:
    `gh issue list --label methodology-gap --label collector:<name>`. If the
    work closes a tracked issue, the issue's "Doc after this fix lands" block IS
    the doc content to paste into Data Sources. The PR description must include
    `Closes #N`.

See [docs/adding-a-collector.md](docs/adding-a-collector.md) for the full
step-by-step walkthrough (code examples, test setup, doc template, commit
template).

## Color palette (Max Headroom)

```
#b4a7d6  lavender  accent, banner
#00d4ff  cyan      info hints
#50fa7b  green     success
#ff6ec7  pink      errors
```

All palette values are defined as named constants in `internal/cli/theme.go`.
Never pass raw ANSI escape strings in command code — reference the theme roles
(`Accent`, `OK`, `Err`, `Info`, `Mute`) instead. `install.sh` uses the same
`#b4a7d6` hex value as a truecolor escape so the install banner and the running
CLI paint with the exact same hue.

## CLI architecture

All CLI output styling and formatting lives in `internal/cli/`:

- **`theme.go`** — maxheadroom palette, `Banner()`, role-based color helpers
  (`Mute`, `Accent`, `OK`, `Err`, `Info`, `Success`, `Failure`)
- **`output.go`** — `WriteOutput`, `WriteJSON`, `WriteFlat`,
  `WriteCollectorList` — facts formatting for the collect subcommand

`cmd/` files are thin wiring: they parse flags, call the SDK, and delegate all
output to `internal/cli/`. No raw `fmt.Fprintf` with color codes in `cmd/`.

## Before committing

Run `just ready` before committing to ensure generated code, package docs,
formatting, and lint are all up to date:

```bash
just ready   # generate, md-fmt, go-fmt, go-vet
```

## Branching

All changes should be developed on feature branches. Create a branch from `main`
using the naming convention `type/short-description`, where `type` matches the
[Conventional Commits] type:

- `feat/add-cpu-collector`
- `fix/memory-parsing-error`
- `docs/update-collector-reference`
- `refactor/simplify-registry`
- `chore/update-dependencies`

When using Claude Code's `/commit` command, a branch will be created
automatically if you are on `main`.

## Commit messages

Follow [Conventional Commits] with the 50/72 rule:

- **Subject line**: max 50 characters, imperative mood, capitalized, no period
- **Body**: wrap at 72 characters, separated from subject by a blank line
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
- Summarize the "what" and "why", not the "how"

Try to write meaningful commit messages and avoid having too many commits on a
PR. Most PRs should likely have a single commit (although for bigger PRs it may
be reasonable to split it in a few). Git squash and rebase is your friend!

## Submitting a PR

- **Describe your changes** — Ensure that you provide a comprehensive
  description of your changes.
- **Issue/PR links** — Link any previous work such as related issues or PRs.
  Please describe how your changes differ to/extend this work.
- **Examples** — Add any examples or screenshots that you think are useful to
  demonstrate the effect of your changes.
- **Draft PRs** — If your changes are incomplete, but you would like to discuss
  them, open the PR as a draft and add a comment to start a discussion. Using
  comments rather than the PR description allows the description to be updated
  later while preserving any discussions.

## AI usage

All contributions are subject to the [AI Usage Policy](AI_POLICY.md) — disclose
the tool you used, and make sure you can explain what your change does without
the aid of AI tools.

## FAQ

> I want to contribute, where do I start?

All kinds of contributions are welcome, whether it's a typo fix or a shiny new
collector. You can also contribute by upvoting/commenting on issues or helping
to answer questions.

> I'm stuck, where can I get help?

If you have questions, feel free to open a [Discussion] on GitHub.

[claude code]: https://claude.ai/code
[conventional commits]: https://www.conventionalcommits.org
[discussion]: https://github.com/osapi-io/gohai/discussions
[go]: https://go.dev
[just]: https://just.systems
