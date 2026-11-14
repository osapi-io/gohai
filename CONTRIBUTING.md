# Contributing

We welcome contributions to gohai. Read this document before you submit a PR. It
has all that you need: the prerequisites, the setup, the conventions that the
code follows, and the pull request workflow.

The
[specifications repository](https://github.com/osapi-io/specs/blob/main/components/gohai/collectors.md)
gives the design of a collector. It tells you which backing library to wrap and
in which order. It tells you what an extension can read through. It tells you
why collector code has no build tag, and how to name a field. Three of these
rules are short, so [docs/methodology.md](docs/methodology.md) repeats them.
That page also tells you which library each collector wraps.

To validate the OCSF output against the upstream schema, refer to
[docs/ocsf-validation.md](docs/ocsf-validation.md).

## Before you start

- Read the [Code of Conduct](CODE_OF_CONDUCT.md). It applies to all interactions
  in this repo.

- **Design docs.** The design of this repository is in
  [osapi-io/specs](https://github.com/osapi-io/specs) under `components/gohai/`.
  Read it before you change the behaviour, and change it when you change the
  behaviour. A design doc in this repository becomes incorrect when the code
  changes, and no check finds the difference.

- **Check the current work.** Look for a PR that already exists. Look for issues
  that discuss the change that you want to make. Use what these discussions say
  in your work.

- **Backwards compatibility.** Make sure that your change does not break the
  current consumers of gohai. A change that is backwards compatible is more
  likely to merge. Look for a method that keeps this compatibility. If you
  cannot find one, open an issue first. Then we can discuss the API change
  before you spend time on a PR.

## Prerequisites

Install the tools with [mise](https://mise.jdx.dev):

```bash
mise install
```

- **[Go].** gohai is written in Go. We support the latest two major Go versions.
  Make sure that your version is one of them.
- **[just].** The task runner. It builds, tests, formats and runs other
  development tasks. Install it with `brew install just`.
- **[uv](https://docs.astral.sh/uv/).** The Python package runner. `just md-fmt`
  formats markdown with mdformat through `uvx`. It installs nothing into the
  repository.

### Claude Code

If you use [Claude Code] for development, install this plugin from the default
marketplace:

```
/plugin install commit-commands@claude-plugins-official
```

- **commit-commands.** Gives the `/commit` and `/commit-push-pr` slash commands.
  They follow the commit conventions of the project automatically.

**The design is in the design docs.** Write or change the page in
[osapi-io/specs](https://github.com/osapi-io/specs) before you build. If the
build shows that the page is wrong, correct the page. A design document in this
repository becomes incorrect when the code changes, and no check finds the
difference.

## Setup

Fetch the shared justfiles and install all dependencies:

```bash
just fetch
just deps
```

## Project structure

- **`main.go`.** The entry point at the repo root. It only calls
  `cmd.Execute()`.
- **`cmd/`.** The Cobra CLI subcommands.
  - `root.go`. The root command, the banner, the context setup and the
    `AddCommand` calls.
  - `collect.go`. `gohai collect`, the collector flags and the SDK calls. It
    sends the output to `internal/cli/`.
  - `validate.go`. `gohai validate`. It validates JSON against the embedded
    schema, from stdin or from `--file`.
  - `version.go`. `gohai version`. The build-time identity, through
    `caarlos0/go-version`.
- **`internal/cli/`.** The CLI output helpers. `pkg/gohai/` never imports them.
  - `theme.go`. The Maxheadroom palette (`#b4a7d6` lavender accent), `Banner()`,
    and the color helpers for each role (`Mute`, `Accent`, `OK`, `Err`, `Info`,
    `Success`, `Failure`).
  - `output.go`. `WriteOutput`, `WriteJSON`, `WriteFlat` and
    `WriteCollectorList` for the collect subcommand.
- **`pkg/gohai/`.** The public SDK.
  - `gohai.go`. The `Gohai` struct, `New()` and `Collect()`.
  - `facts.go`. The `Facts` struct, with typed collector fields and the JSON and
    Flat methods.
  - `options.go`. The functional options (`WithEnabled`, `WithDisabled`,
    `WithCollectors`).
  - `registry.go`. `PublicRegistry`. The CLI uses it to list the flags.
- **`pkg/gohai/collectors/<name>/`.** The public sub-package of each collector.
  Use the osapi pattern of one struct for each OS, with no build tags. The
  canonical reference is `pkg/gohai/collectors/shells/`.
  - `<name>.go`. The `Info` struct, the `Collector` interface, and the `base`
    struct. The `base` struct holds the shared `Name()`/`DefaultEnabled()`/
    `Dependencies()`. The file also has the `New()` factory, which selects the
    variant from `platform.Detect()`. It also has the helpers that all OS
    variants share, such as shared parsing and shared constants.
  - `linux.go`.
    `type Linux struct { base; FS avfs.VFS; Exec executor.Executor }`, with
    `NewLinux()` and the `(l *Linux) Collect(ctx)` method. Add a field only when
    the collector needs it. **No build tag.**
  - `darwin.go`. `type Darwin struct { base; FS; Exec }` with `NewDarwin()` and
    the `(d *Darwin) Collect(ctx)` method. **No build tag.**
  - `debian.go` / `rhel.go`. Add these only when a distribution is really
    different. They use the same pattern, and the `New()` switch selects them.
  - `<name>_public_test.go`. The **only** test file. At the top, it has the
    compile-time `collector.Collector` asserts. It has `TestNew` for the factory
    selection. It has one table-driven `TestCollect`. Each row has a
    `variant: "linux" | "darwin"` column and makes the correct struct for that
    OS. It can also have separate test methods for public helpers that are
    really pure, for example `TestHumanDuration` and `TestBytesToString`. There
    are no `linux_public_test.go` / `darwin_public_test.go` files.
- **`schemas/`.** The JSON Schema and the field-name artifacts.
  - `gen/`. The generator tool. `go run .` reflects `gohai.Facts` into JSON
    Schema through `invopop/jsonschema`. `just generate` runs its
    `//go:generate` directive.
  - `gohai.schema.json`. The generated schema (draft 2020-12). It is committed.
  - `schema.go`. The `//go:embed` of `gohai.schema.json` for the validate
    subcommand.
  - `field-mapping.md`. The tier mapping of each field (OCSF/OTel/ convention),
    with citations. It has 950 rows.
  - `ocsf-gaps.md`. 82 candidates for upstream OCSF PRs.
- **`internal/platform/`.** The OS and distribution detection. It wraps
  gopsutil. `Detect()` is a `var` that tests can replace. Thus collector tests
  can force each branch and do not import gopsutil. `hostInfoFn` is private.
  Only the tests of platform can get it, through `export_test.go`.
- **`internal/collector/`.** The collector interface and the registry code.
  - `collector.go`. The `Collector` interface.
  - `registry.go`. `Registry`. It registers collectors, resolves the
    dependencies and runs the collectors at the same time.
  - `mocks/`. The gomock generation (`go generate`) and the committed mock.
- **`internal/executor/`.** The abstraction for command execution.
  - `executor.go`. The `Executor` interface (`Execute(ctx, name, args...)`).
  - `mocks/`. The gomock generation (`go generate`) and the committed mock.

## Code style

[gofumpt] formats the Go code and [golangci-lint] lints it. CI enforces both.

```bash
just go-fmt-check   # Check formatting
just go-fmt         # Auto-fix formatting
just go-vet         # Run linter
```

`.golangci.yml` declares the linters that run. Read them there. Do not look for
a list here, because a copied list becomes incorrect when the configuration
changes. Formatting does not apply to generated files (`*.gen.go`, `*.pb.go`).

### Documentation

[mdformat] formats the Markdown files through `uvx`. CI enforces this style.

```bash
just md-fmt-check   # Check formatting
just md-fmt         # Auto-fix formatting
```

## Code standards

### Function signatures

A function with parameters uses the multi-line format. Put one parameter on each
line. Put the closing parenthesis and the return types on a line of their own:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
}
```

A function with no parameters stays on one line:

```go
func Name() string {
}
```

Then a new parameter shows as one added line in the diff, not as a new
signature.

### File naming

Give a file the name of what it holds. Do not use `helpers.go`, `utils.go`, or
names of that type. These names tell where the code is, not what it is. They
also collect all code that has no other location.

`types.go` holds only type declarations: structs, interfaces, constants and
aliases. A function goes in a file with the name of what the function does.

A test file has the name of the production file that it tests. If the tests
become too large to read, split the production file first. Then each test file
keeps a related production file. Do not split the tests away from the file that
they test.

### Go patterns

- Error wrapping: `fmt.Errorf("context: %w", err)`. Then the chain names each
  layer that the error went through, and `errors.Is` and `errors.As` can examine
  it.
- Return early. Do not put the successful path inside conditionals.
- Unused parameters: rename them to `_`.
- Import order: the standard library, then third party, then local. Put a blank
  line between the groups.

### Test doubles

Use `mockgen` to generate a double for an interface that this organization
defines, and commit it. Do not write a struct by hand for this interface.

Generated mocks are in a `mocks` package next to the code that they mock. A
`generate.go` file holds the directive that makes them:

```go
package mocks

//go:generate go tool go.uber.org/mock/mockgen -source=../types.go -destination=types.gen.go -package=mocks
```

Go finds the generator through the tool dependencies of the module. Thus each
checkout runs the version that `go.mod` records. The destination files end in
`.gen.go`, and you commit them. Do not use `gen/` for mocks. API code generation
uses that name.

If the interface is **unexported**, a sibling package cannot work. The mock must
import the package to name the types in the interface, and the tests of the
package must import the mock. Generate the mock into the package. Give it a
destination that only tests use. Then the mocking library is not in the
dependency graph of code that imports the package:

```go
// generate.go, in the package that declares the interface
package thispackage

//go:generate go tool go.uber.org/mock/mockgen -source=thing.go -destination=thing.gen_test.go -package=thispackage
```

In the two cases, the directives are in a `generate.go` that holds no code. The
name of the generated file has `.gen`, so that a reader knows not to edit it.

If many call sites set the same expectations, write a constructor that returns a
configured mock. Do not add a hand-written type. The generated mock is still
what satisfies the interface.

You write three types of double by hand, because generation gives no benefit:

- A double for a standard library interface: `net.Conn`, `fs.File`, `io.Writer`,
  `slog.Handler`. These do not change when our code changes.
- A double that has a real implementation of the behavior under test, for
  example signatures with a key pair that is really generated.
- A recorder for a dependency that a goroutine calls, if the test cannot join
  that goroutine. A generated mock asserts a call count at a time that the test
  cannot control. Write this reason where you define the recorder.

### File headers

Each `.go` file MUST start with the MIT license header. Refer to a Go file in
the repo for the exact format. In a file with a build tag, put `//go:build` on
line 1, then a blank line, then the header.

### Error wrapping at the module boundary

Never let raw gopsutil, ghw or procfs error types go out through the public API.
Wrap them. Then callers do not need these packages in their module graph to
handle an error.

## Testing

```bash
just test           # Run all tests (lint + unit + coverage)
just go-unit       # Run unit tests only
just go-unit-cov   # Generate coverage report
go test -run TestName -v ./internal/collector/platform/...  # Run a single test
```

The coverage gate is 100%. If the total coverage is less than this, `just test`
fails. Thus a change that adds untested code fails locally and in CI:

```bash
just go-unit-cov-check   # Report coverage and fail below the target
```

`.github/codecov.yml` and the shared `go` justfile module declare the target.
Change the two together.

### Test file conventions

- Public tests: `*_public_test.go` in the `_test` package of the package. They
  test the exported API. This is the default.
- Internal tests: `*_test.go` in the same package. They test what the exported
  API cannot get to.
- Suite names: `*_public_test.go` → `{Name}PublicTestSuite`, `*_test.go` →
  `{Name}TestSuite`.
- `testify/suite` with table-driven cases.
- One suite method for each function under test. Success, errors and edge cases
  are rows in one table, not separate methods.
- `export_test.go` gives external tests access to unexported symbols, with an
  alias or a setter. Do not use an alias to test again the behavior that the
  test of the caller already gets to. Use the pattern for a helper that has its
  own contract.

External tests in this repository are in a `_test` package next to the code that
they test. Tables have `validateFunc` callbacks. The setter form is
`SetXFn(fn) func()`. It returns a restore func, and the caller defers it.

These rules are specific to collectors:

- **One `TestCollect` for each collector.** All scenarios are rows in one table
  with a `variant` column: Linux and Darwin, the success paths and the error
  paths. Do not split into `TestCollectLinux` / `TestCollectDarwin`.
- Use separate test methods only for public helpers that are really pure and
  have their own contract (`TestHumanDuration`, `TestBytesToString`). Do not use
  them for bridge code that `Collect` already tests.
- **Replace at the boundary.** Each `TestCollect` row replaces the raw call to
  the upstream library (`hostInfoFn`, `partitionsFn`, ...). Thus the bridge
  mapping runs on each row.
- **No custom assertion messages.** Write `s.Equal(want, got)`, not
  `s.Equal(want, got, "expected equal")`.
- The target is 100% test coverage on all packages.

## Quick reference

```bash
just fetch / just deps / just test / just go-unit / just go-vet / just go-fmt
gohai collect --pretty             # run default collectors
gohai collect --no-defaults --collector.cpu  # specific collectors
gohai collect --pretty | gohai validate      # validate against schema
gohai version                      # build info
```

## VFS and executor abstractions

A collector that reads files or runs commands **MUST** use two shared
abstractions. Inject them as struct fields on the variant for each OS. This is
the same pattern as the Agent struct of osapi.

### `avfs.VFS`: filesystem

Use [`github.com/avfs/avfs`](https://github.com/avfs/avfs) directly, with no
custom wrapper. Production uses the real OS filesystem through
`osfs.NewWithNoIdm()`. Tests use `memfs.New()` with canned files at real
absolute paths, for example `/proc/meminfo` and `/etc/os-release`. Tests run the
real `ReadFile` / `Open` / `Stat` code path against content in memory. This is a
real integration test of how the collector uses the filesystem. It does not
replace a function with a stub.

**Struct shape for each OS:**

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

The reference implementation is `pkg/gohai/collectors/shells/`.

### `executor.Executor`: command execution

`internal/executor` gives a minimal interface with one method:
`Execute(ctx, name, args...) ([]byte, error)`. Its gomock mock is at
`internal/executor/mocks/`. The production implementation wraps
`exec.CommandContext` and returns stdout and stderr together. A collector that
runs commands (sysctl, sw_vers, lsb_release, loginctl, lscpu, kextstat and
others) holds the Executor as a struct field.

**Struct for each OS, with FS and Executor:**

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

To make the mocks again, run `go generate ./internal/executor/...`, then commit
them. The pinned tool is `go.uber.org/mock`, the maintained fork of the archived
`golang/mock`.

### Migration status

All new code and new collectors MUST use these abstractions. Some collectors
still use the legacy `ReadFileFn` / `RunCmdFn` struct-field pattern. They move
to the new pattern when methodology work changes them. The canonical references
are `pkg/gohai/collectors/shells/` (VFS only) and
`pkg/gohai/collectors/platform/` (VFS + Executor).

## Adding a new collector

[docs/adding-a-collector.md](docs/adding-a-collector.md) gives the steps. It has
the code examples, the file layout, the test setup and the commit template.

The **reference implementation** is `pkg/gohai/collectors/shells/`. Copy its
patterns exactly.

### Done-definition (every collector, every time)

Before you mark a collector as complete, each item below must be true:

01. **You analyzed Ohai's plugin and spec** for HOW it collects: the data
    sources, the special cases of each distribution and the fallback chains. Our
    collection logic is the same as theirs, so we get their years of bug fixes.
    Each difference has a documented reason.
02. **You checked the OCSF schema** ([schema.ocsf.io](https://schema.ocsf.io/))
    for canonical field names. If OCSF has no field, you checked the
    [OpenTelemetry Resource Semantic Conventions][otel-semconv]. You recorded
    the schema mappings in the **Schema mapping** column of the Collected fields
    table on the collector page. If a schema has a field that we can report but
    do not, add the field or write why not.
03. **The osapi pattern of one struct for each OS**. No build tags. The factory
    selects the variant from `platform.Detect()`. Each struct for an OS
    implements Collect.
04. **100% test coverage.**
    `go tool cover -func=/tmp/cov.out | grep -v '100.0%'` returns nothing for
    the files of the collector.
05. **One `<name>_public_test.go`, one `TestCollect`.** Linux and Darwin
    scenarios are in the same table, with a `variant` column. No separate
    `linux_public_test.go` / `darwin_public_test.go` files. No `TestReadX`
    methods that test bridge code that `TestCollect` already tests. The only
    permitted extra test methods are tests of pure public helpers, for example
    `TestHumanDuration`.
06. **The seams are at the boundary.** `export_test.go` replaces the call to the
    upstream library (`hostInfoFn`, `partitionsFn` and others). Thus the bridge
    mapping runs on each row. No `readXFn = readX` wrappers in the middle of the
    collector. No alias that tests a bridge that `TestCollect` already tests.
07. **`docs/collectors/<name>.md`** is a functional spec that is complete in
    itself. It has these sections: Description (what and why, in our voice),
    Collected fields with a **Schema mapping** column (the OCSF path first, the
    OpenTelemetry attribute if OCSF has no field), Platform support, Example
    output, SDK usage, Enable/disable, Dependencies, Data sources (the method,
    step by step, in OUR voice, not an Ohai parity table), and Backing library.
    **No "Known gaps vs. Ohai" section.** Methodology gaps are GitHub issues
    with the labels `methodology-gap` / `collector:<name>`.
08. **README.md**: the row of the collector shows `✅ (<backing>)`.
09. **Lint clean**: `just go-vet` returns 0 issues.
10. **The commit message** tells the "why". It names the Ohai and OCSF
    references that gave the implementation, the extensions that we added to the
    upstream library, and each intentional difference.
11. **Check the GitHub issues** for tracked methodology gaps:
    `gh issue list --label methodology-gap --label collector:<name>`. If the
    work closes a tracked issue, the "Doc after this fix lands" block of the
    issue IS the content to paste into Data sources. The PR description must
    include `Closes #N`.

For the full steps, refer to
[docs/adding-a-collector.md](docs/adding-a-collector.md). It has the code
examples, the test setup, the doc template and the commit template.

## Color palette (Max Headroom)

```
#b4a7d6  lavender  accent, banner
#00d4ff  cyan      info hints
#50fa7b  green     success
#ff6ec7  pink      errors
```

`internal/cli/theme.go` defines all palette values as named constants. Never
pass raw ANSI escape strings in command code. Use the theme roles (`Accent`,
`OK`, `Err`, `Info`, `Mute`). `install.sh` uses the same `#b4a7d6` hex value as
a truecolor escape. Thus the install banner and the CLI show the same color.

## CLI architecture

All CLI output style and format is in `internal/cli/`:

- **`theme.go`.** The maxheadroom palette, `Banner()`, and the color helpers for
  each role (`Mute`, `Accent`, `OK`, `Err`, `Info`, `Success`, `Failure`).
- **`output.go`.** `WriteOutput`, `WriteJSON`, `WriteFlat` and
  `WriteCollectorList` for the collect subcommand.

The `cmd/` files are thin. They parse flags, call the SDK, and send all output
to `internal/cli/`. Do not use raw `fmt.Fprintf` with color codes in `cmd/`.

## Before committing

Before you commit, run `just ready`. It makes sure that the generated code, the
package docs, the format and the lint are all current:

```bash
just ready
```

## Branching

Make all changes on feature branches. Create a branch from `main` with the name
`type/short-description`. The `type` is a [Conventional Commits] type:

- `feat/add-cpu-collector`
- `fix/memory-parsing-error`
- `docs/update-collector-reference`
- `refactor/simplify-registry`
- `chore/update-dependencies`

If you are on `main`, the `/commit` command of Claude Code creates a branch
automatically.

## Commit messages

Follow [Conventional Commits] with the 50/72 rule:

- **Subject line**: 50 characters maximum, imperative mood, capitalized, no
  period.
- **Body**: wrap at 72 characters. Put a blank line between the subject and the
  body.
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`
- Tell the "what" and the "why", not the "how".

Write commit messages that have meaning. Do not put too many commits on a PR.
Most PRs have one commit. A large PR can have a small number of commits. Use Git
squash and rebase.

## Submitting a PR

- **Describe your changes.** Tell what changed and why. A reviewer must not have
  to read the diff to know the reason.
- **Issue/PR links.** Link to related issues or PRs. Tell how your changes are
  different from this work or add to it.
- **Examples.** Add examples or screenshots that show the effect of your
  changes.
- **Draft PRs.** If your changes are not complete and you want to discuss them,
  open the PR as a draft. Add a comment to start the discussion. Use comments,
  not the PR description. Then you can change the description later, and the
  discussion stays.

## AI usage

This repo is written with AI help. The [AI Usage Policy](AI_POLICY.md) applies
to all contributions. Tell which tool you used. Make sure that you can explain
what your change does without AI tools.

## FAQ

> I want to contribute, where do I start?

We welcome all types of contributions, from a typo fix to a new collector. You
can also vote on issues, comment on issues, or help to answer questions.

> I'm stuck, where can I get help?

If you have questions, open a [Discussion] on GitHub.

[claude code]: https://claude.ai/code
[conventional commits]: https://www.conventionalcommits.org
[discussion]: https://github.com/osapi-io/gohai/discussions
[go]: https://go.dev
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[just]: https://just.systems
[mdformat]: https://pypi.org/project/mdformat/
