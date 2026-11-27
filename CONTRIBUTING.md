# Contributing

Thank you for your contribution to gohai.

## Before you start

- Read the [Code of Conduct](CODE_OF_CONDUCT.md). It applies to all
  communication in this repository.
- **Design pages.** The design pages of this repository are in
  [osapi-io/specs](https://github.com/osapi-io/specs), under
  `components/gohai/`. Read them before you change the behavior. When you change
  the behavior, change the design page in the same work. A design page in this
  repository becomes incorrect when the code changes. No check finds the
  difference.
- **Keep the descriptions in sync.** `gohai.Collectors()` describes each
  collector, and `gohai collector list|get|spec get|validate` prints it. osapi
  mounts the same command as `osapi agent collector`
  (`cmd.NewCollectorCommand(path)`). Thus a change to a collector, to its
  description or to these commands changes what osapi prints too. Change the
  data, never a copy. Then tag a release, so osapi can take it. The rule is in
  [Keep the descriptions in sync](https://github.com/osapi-io/specs/blob/main/CONTRIBUTING.md#keep-the-descriptions-in-sync).
- **Examine the work that exists.** Look for a pull request or an issue about
  the change. If you find one, use what it says in your work.
- **Backward compatibility.** Make sure that your change does not stop a program
  that uses gohai. A change that keeps compatibility is more likely to merge. If
  your change cannot keep it, open an issue first. Then the maintainers can
  discuss the API change before you do the work.

## Prerequisites

Install the tools with [mise]:

```bash
mise install
```

- **[Go].** gohai is written in Go. The project supports the two latest major
  versions of Go. Use one of them.
- **[uv].** A Python package runner. `just md-fmt` uses it to run [mdformat]. It
  installs nothing into the repository.
- **[just].** The task runner. Use it to build, test and format. You can also
  install it with `brew install just`.

### Coding agents

This repository works with any coding agent that reads `AGENTS.md`. `CLAUDE.md`
only points to `AGENTS.md`. Prepare your agent as its documentation tells you.
This repository does not need a plugin or an extension.

## Setup

Download the shared justfiles and install the dependencies:

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
    `WithCollectors`, `WithSources`).
  - `source/`. `Sources`: the filesystem, the command runner, the clock, the
    platform family and the name resolver that a collector reads. A zero field
    is the real host. `Host()` gives the real host. `Runner` is the interface of
    the command runner, and `Resolver` of the name lookups.
  - `source/mocks/`. `MockRunner` and `MockResolver`, the generated doubles of
    `source.Runner` and `source.Resolver`, for gohai's tests and for a caller
    that gives gohai its sources.
  - `registry.go`. `PublicRegistry`. The CLI uses it to list the flags.
- **`pkg/gohai/collectors/<name>/`.** The public sub-package of each collector.
  Use the osapi pattern of one struct for each OS, with no build tags. The
  canonical reference is `pkg/gohai/collectors/shells/`.
  - `<name>.go`. The `Info` struct, the `Collector` interface, and the `base`
    struct. The `base` struct holds the shared `Name()`/`DefaultEnabled()`/
    `Dependencies()`. The file also has `NewWith(src)`, which selects the
    variant from the family of `src` and gives the variant the sources of `src`
    through `collector.Wire`. `New()` is `NewWith(source.Sources{})`. The file
    also has the helpers that all OS variants share, such as shared parsing and
    shared constants.
  - `linux.go`. `type Linux struct { base; FS avfs.VFS; Exec source.Runner }`,
    with `NewLinux()` and the `(l *Linux) Collect(ctx)` method. A collector that
    reads the clock also has `Now func() time.Time`. Add a field only when the
    collector needs it. Each read of the host goes through these fields. **No
    build tag.**
  - `darwin.go`. `type Darwin struct { base; FS; Exec }` with `NewDarwin()` and
    the `(d *Darwin) Collect(ctx)` method. **No build tag.**
  - `debian.go` / `rhel.go`. Add these only when a distribution is really
    different. They use the same pattern, and the `NewWith` switch selects them.
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
    `just go-generate-check` fails when it, or a mock, differs from what
    `go generate` writes.
  - `schema.go`. The `//go:embed` of `gohai.schema.json` for the validate
    subcommand.
  - `field-mapping.md`. The tier mapping of each field (OCSF/OTel/ convention),
    with citations. It has 970 rows.
  - `ocsf-gaps.md`. The candidates for upstream OCSF PRs, from the audit that
    the page gives the date of.
- **`internal/platform/`.** `Detect()`, a `var` that tests can replace, which is
  `family.Detect` on a real host. `NewWith` uses it only when `src` has no
  family and no filesystem. With a filesystem, the family is that of its
  `/etc/os-release`.
- **`internal/collector/`.** The collector interface and the registry code.
  - `collector.go`. The `Collector` interface.
  - `sources.go`. `IsDarwin(src)` and `Family(src)` select a variant. `Wire`
    sets the `FS`, `Exec`, `Now` and `Resolver` fields of a variant from the
    sources, and panics on a field with one of those names and another type.
  - `registry.go`. `Registry`. It registers collectors, resolves the
    dependencies and runs the collectors at the same time.
  - `mocks/`. The gomock generation (`go generate`) and the committed mock.
- **`internal/sourcetest/`.** The sources of a collector test. `FS` gives a
  memory filesystem that holds the given files. `Commands` holds the command
  output of a table row, and its `Mock` is the generated `MockRunner` that
  answers it.

## Code style

[gofumpt] formats the Go code. [golangci-lint] examines the Go code for
problems. CI makes sure that both pass.

```bash
just go-fmt-check   # Check formatting
just go-fmt         # Auto-fix formatting
just go-vet         # Run linter
```

`.golangci.yml` lists the linters that run. Read the list there. A copy of the
list in a page becomes incorrect when the configuration changes. Formatting does
not apply to generated files (`*.gen.go`, `*.pb.go`).

Do not write `//nolint:revive`. It hides a problem and does not fix it. Fix the
name or the code. When a name must be different in an API, give it an explicit
name, such as a struct tag.

### Documentation

[mdformat] formats the markdown files, through `uvx`. CI makes sure that the
format is correct.

```bash
just md-fmt-check   # Check formatting
just md-fmt         # Auto-fix formatting
```

Write each page in Simplified Technical English, as
[the osapi-io constitution](https://github.com/osapi-io/specs/blob/main/CONSTITUTION.md)
requires.

## Code standards

### Function signatures

If a function has parameters, write its signature on more than one line. Put
each parameter on its own line. Put the last `)` and the return types on their
own line:

```go
func FunctionName(
    param1 type1,
    param2 type2,
) (returnType, error) {
}
```

Write a function that has no parameters on one line:

```go
func Name() string {
}
```

Then a new parameter shows as one added line in a diff, not as a changed
signature.

### File names

Give a file the name of what it holds. Do not use `helpers.go`, `utils.go` or
names of that type. They tell where code is, not what it is. They also collect
code that has no other place.

`types.go` holds only type declarations: structs, interfaces, constants and
aliases. Put a function in a file that has the name of what the function does.

A test file has the name of the production file that it tests. If a test file
becomes too large to read, split the production file first. Then each test file
keeps its production file.

### Go patterns

- Wrap errors with `fmt.Errorf("context: %w", err)`. Then the chain names each
  layer. `errors.Is` and `errors.As` can examine the chain.
- Return early. Do not put the successful path inside conditions.
- Give an unused parameter the name `_`.
- Put imports in three groups, with a blank line between them: the standard
  library, third-party packages, then local packages.
- A package holds one kind of thing, and its name tells what. Do not make a
  package named `util`, `common`, `helpers` or `shared`.
- The name of a package is the last part of its path. An exported name does not
  repeat it inside a longer name: `atomicfile.Write`, not
  `atomicfile.WriteFileAtomic`. The main type can have the name of the package,
  as `time.Time` does.
- A caller does not need an alias. If two packages that a file imports have the
  same name, rename one of them. Mock packages are the exception. Each is named
  `mocks`, so a test that uses two of them gives one an alias. A clash with a
  package outside the module, or between two names that are both correct, also
  takes an alias.
- A directory can group packages that belong together. Do not put many packages
  under one package.
  [A Go package is one thing](https://github.com/osapi-io/specs/blob/main/CONSTITUTION.md#a-go-package-is-one-thing)
  tells why.

### Test doubles

For an interface that this organization defines, generate the double with
`mockgen` and commit it. Do not write a struct by hand for it.

A generated mock is in a `mocks` package next to the code that it mocks. A
`generate.go` file in that package holds the directive:

```go
package mocks

//go:generate go tool go.uber.org/mock/mockgen -source=../types.go -destination=types.gen.go -package=mocks
```

The tool dependencies of the module supply the generator. Thus each checkout
runs the version that `go.mod` records. The names of the generated files end in
`.gen.go`. Commit these files. Do not put mocks in `gen/`. That name is for API
code generation.

If the interface is **unexported**, a sibling package cannot hold the mock. The
mock must import the package to name the types of the interface. The tests of
the package must import the mock. Thus generate the mock into the package. Give
the mock a destination that only tests use. Then the mock library is not a
dependency of code that imports the package:

```go
// generate.go, in the package that declares the interface
package thispackage

//go:generate go tool go.uber.org/mock/mockgen -source=thing.go -destination=thing.gen_test.go -package=thispackage
```

In both cases, the directives are in a `generate.go` that holds no code. The
name of the generated file has `.gen`, so a reader knows not to edit it.

If many tests set the same expectations, write a constructor that returns a
configured mock. Do not write a type by hand for it. The generated mock is still
the type that satisfies the interface.

Write these three doubles by hand, because a generated one gives no advantage:

- A double for a standard library interface: `net.Conn`, `fs.File`, `io.Writer`,
  `slog.Handler`. These interfaces do not change when our code changes.
- A double that has a real implementation of the behavior under test, for
  example signatures from a key pair that the test generates.
- A recorder for a dependency that a goroutine calls, if the test cannot wait
  for that goroutine. A generated mock counts the calls at a time that the test
  cannot control. Write that reason where you define the recorder.

### File headers

Start each `.go` file with the MIT license header. Copy the format from a Go
file in the repository. In a file with a build tag, put `//go:build` on line 1,
then a blank line, then the header.

### Error wrapping at the module boundary

Never let raw error types of a library go out through the public API. Wrap them.
Then callers do not need these packages in their module graph to handle an
error.

## Tests

```bash
just test           # Run all tests (lint + unit + coverage)
just go-unit       # Run unit tests only
just go-unit-cov   # Generate coverage report
go test -run TestName -v ./internal/platform/...  # Run a single test
```

The coverage target is 100%. `just test` fails when the total coverage is less
than the target. Thus a change that adds code without tests fails on your
machine and in CI:

```bash
just go-unit-cov-check   # Report coverage and fail below the target
```

`.github/codecov.yml` and the shared `go` justfile module both declare the
target. Change the two together.

`.coverignore` leaves out only code that no test is for: the generated mocks and
`schemas/gen`, the examples, and the root `main.go`. `main.go` gives the
process's arguments, streams and signals to `cmd.Execute`, which the tests of
`cmd` run. The `main.go` line is anchored to the root, so that a file named
`main.go` in a package is counted.

### Test file conventions

- Public tests: `*_public_test.go`, in the `_test` package of the package. They
  test the exported API. Write this type of test first.
- Internal tests: `*_test.go`, in the same package. Use them only for what the
  exported API cannot reach.
- Suite names: `*_public_test.go` has `{Name}PublicTestSuite`, and `*_test.go`
  has `{Name}TestSuite`.
- Use `testify/suite` with table-driven cases.
- Write one suite method for each function under test. Success, errors and edge
  cases are rows in one table, not different methods.
- `export_test.go` gives external tests access to unexported symbols, through an
  alias or a setter. Do not use an alias to test again what the test of the
  caller already reaches. Use it for a helper that has its own contract.

External tests in this repository are in a `_test` package next to the code that
they test. Tables have `validateFunc` callbacks. The setter form is
`SetXFn(fn) func()`. It returns a restore func, and the caller defers it.

These rules are specific to collectors:

- **One `TestCollect` for each collector.** All scenarios are rows in one table
  with a `variant` column. The rows include Linux and Darwin, and the success
  paths and the error paths. Do not split into `TestCollectLinux` /
  `TestCollectDarwin`.
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

### `source.Runner`: command execution

`pkg/gohai/source` gives a minimal interface with one method:
`Execute(ctx, name, args...) ([]byte, error)`. Its gomock mock is `MockRunner`,
at `pkg/gohai/source/mocks/`. The production implementation,
`source.Commands()`, wraps `exec.CommandContext`. It returns stdout, and puts
stderr in the error. A collector that runs commands (sysctl, sw_vers,
lsb_release, loginctl, lscpu, kextstat and others) holds the Runner as a struct
field.

**Struct for each OS, with FS and Runner:**

```go
type Darwin struct {
    base
    FS   avfs.VFS
    Exec source.Runner
}

func NewDarwin() *Darwin {
    return &Darwin{
        FS:   osfs.NewWithNoIdm(),
        Exec: source.Commands(),
    }
}
```

**Test shape (gomock):**

```go
ctrl := gomock.NewController(t)
mockExec := sourcemocks.NewMockRunner(ctrl)
mockExec.EXPECT().
    Execute(gomock.Any(), "sw_vers", "-productVersionExtra").
    Return([]byte("(a)\n"), nil)

c := &platform.Darwin{FS: memfs.New(), Exec: mockExec}
```

To make the mocks again, run `go generate ./pkg/gohai/source/...`, then commit
them. The pinned tool is `go.uber.org/mock`, the maintained fork of the archived
`golang/mock`.

### Migration status

All collectors use these abstractions. No Go file has the old `ReadFileFn` or
`RunCmdFn` struct fields. All new code and new collectors use them too. The
references are `pkg/gohai/collectors/shells/` (VFS only) and
`pkg/gohai/collectors/platform/` (VFS and Executor).

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

03. **The osapi pattern of one struct for each OS**. No build tags. `NewWith`
    selects the variant from the family of its sources. Each struct for an OS
    implements Collect, and reads the host only through its `FS`, `Exec` and
    `Now` fields.

04. **100% test coverage.**
    `go tool cover -func=/tmp/cov.out | grep -v '100.0%'` returns nothing for
    the files of the collector.

05. **One `<name>_public_test.go`, one `TestCollect`.** Linux and Darwin
    scenarios are in the same table, with a `variant` column. No separate
    `linux_public_test.go` / `darwin_public_test.go` files. No `TestReadX`
    methods that test bridge code that `TestCollect` already tests. The only
    permitted extra test methods are tests of pure public helpers, for example
    `TestHumanDuration`.

06. **The seams are the sources.** A test row gives the files, the command
    output, the clock and the name lookups, through `internal/sourcetest` and
    the generated mocks of `pkg/gohai/source/mocks`. Thus the parse runs on each
    row. No package `var` that a test replaces for a read, and no
    `readXFn = readX` wrappers in the middle of the collector.

07. **`docs/collectors/<name>.md`** is a functional spec that is complete in
    itself. It has these sections:

    - Description: what and why, in our voice.
    - Collected fields, with a **Schema mapping** column. Give the OCSF path
      first. If OCSF has no field, give the OpenTelemetry attribute.
    - Platform support.
    - Example output.
    - SDK usage.
    - Enable/disable.
    - Dependencies.
    - Data sources: the method, step by step, in OUR voice. Not an Ohai parity
      table.
    - Backing library.

    **No "Known gaps vs. Ohai" section.** Methodology gaps are GitHub issues
    with the labels `methodology-gap` / `collector:<name>`.

08. **docs/collectors/README.md**: the row of the collector shows
    `✅ (<backing>)`.

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

## Before you commit

Run `just ready` before you commit. It makes sure that generated code, package
documentation, format and lint are current:

```bash
just ready
```

## Branches

Make all changes on a branch. Make the branch from `main`. Give it the name
`type/short-description`, where `type` is the [Conventional Commits] type:

- `feat/add-retry-logic`
- `fix/null-pointer-crash`
- `docs/update-api-reference`
- `refactor/simplify-handler`
- `chore/update-dependencies`

## Commit messages

Use [Conventional Commits] with the 50/72 rule:

- **Subject line.** 50 characters or fewer, in the imperative mood, with no
  period at the end.
- **Body.** Wrap it at 72 characters. Put a blank line between the subject and
  the body.
- **Format.** `type(scope): description`.
- **Types.** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
  `chore`.
- Tell what changed and why, not how.

Write commit messages that help the reader. Keep the number of commits in a pull
request small. Most pull requests have one commit. A large pull request can have
a few. Use squash and rebase to combine commits.

## Submit a pull request

- **Describe your changes.** Tell what changed and why. Then a reviewer does not
  have to read the diff to find the reason.
- **Links to issues and pull requests.** Link to related issues and pull
  requests. Tell how your change is different from that work, or how it adds to
  it.
- **Examples.** Add examples or screenshots that show the effect of your change.
- **Draft pull requests.** If your change is not complete but you want to
  discuss it, open the pull request as a draft. Start the discussion in a
  comment, not in the description. Then you can change the description later and
  keep the discussion.

## AI usage

We write this repository with the help of AI tools. The
[AI Usage Policy](AI_POLICY.md) applies to all contributions. Tell which tool
you used. Make sure that you can explain what your change does without the help
of an AI tool.

## FAQ

> I want to contribute. Where do I start?

All contributions are welcome, from a typo fix to a new feature. You can also
vote or comment on issues, or help answer questions.

> I need help. Where do I ask?

Open a [Discussion] on GitHub.

[conventional commits]: https://www.conventionalcommits.org
[discussion]: https://github.com/osapi-io/gohai/discussions
[go]: https://go.dev
[gofumpt]: https://github.com/mvdan/gofumpt
[golangci-lint]: https://golangci-lint.run
[just]: https://just.systems
[mdformat]: https://pypi.org/project/mdformat/
[mise]: https://mise.jdx.dev
[otel-semconv]: https://opentelemetry.io/docs/specs/semconv/resource/
[uv]: https://docs.astral.sh/uv/
