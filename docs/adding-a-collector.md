# Adding a new collector

The steps to build a new collector. For the rules and the principles (library
first, OCSF + OpenTelemetry names, no build tags and others), refer to
[CONTRIBUTING.md](../CONTRIBUTING.md).

**Reference implementation:** `pkg/gohai/collectors/shells/`. Copy its file
layout and its patterns exactly.

## Done-definition

Refer to
[CONTRIBUTING.md: Done-definition](../CONTRIBUTING.md#done-definition-every-collector-every-time).
Each item there must be true before you mark the collector as complete.

## Step 1: create the sub-package

Path: `pkg/gohai/collectors/<name>/`. The package is public, because consumers
such as OSAPI import the `Info` structs directly.

## Step 2: `<name>.go` (top-level factory + shared surface)

```go
// (MIT header)
// Package cpu collects CPU topology and feature facts.
package cpu

import (
    "context"

    "github.com/osapi-io/gohai/internal/collector"
    "github.com/osapi-io/gohai/pkg/gohai/family"
    "github.com/osapi-io/gohai/pkg/gohai/source"
)

// Info holds CPU information. Field names follow OCSF first, then
// OpenTelemetry when OCSF is silent.
type Info struct {
    CPUCount  int      `json:"cpu_count"`            // OCSF: device.cpu_count
    CPUCores  int      `json:"cpu_cores"`            // OCSF: device.cpu_cores
    VendorID  string   `json:"vendor_id,omitempty"`  // OTel: host.cpu.vendor.id
    ModelName string   `json:"model_name,omitempty"` // No direct schema
    Flags     []string `json:"flags,omitempty"`      // No direct schema
}

// Collector is the public interface every cpu variant satisfies.
type Collector interface {
    collector.Collector
}

// base holds the identity (Name/DefaultEnabled/Dependencies) common to
// every per-OS variant.
type base struct{}

func (base) Name() string           { return "cpu" }
func (base) DefaultEnabled() bool   { return true }
func (base) Dependencies() []string { return nil }

// New returns the cpu variant for the real host.
func New() Collector {
    return NewWith(source.Sources{})
}

// NewWith returns the cpu variant for the family of src, which reads
// through the sources of src. A zero field of src is the real host's.
func NewWith(src source.Sources) Collector {
    switch collector.Family(src) {
    case family.Darwin:
        return collector.Wire(NewDarwin(), src)
    case family.Debian: // only if debian diverges; else drop this case
        return collector.Wire(NewDebian(), src)
    case family.RHEL, family.Fedora, family.Amazon: // only if RPM hosts diverge
        return collector.Wire(NewRHEL(), src)
    default:
        return collector.Wire(NewLinux(), src)
    }
}

// Cross-OS helpers (shared parsers, shared constants) live here too.
```

## Step 3: per-OS struct implementations (no build tags)

Each file declares a struct for its OS. The struct embeds `base` and implements
`Collect`. The struct reads the host only through its source fields: `FS` for
files, `Exec` for commands, and `Now` for the clock. `collector.Wire` sets these
fields from the caller's sources. Thus a test, or osapi's fake host, gives the
files and the command output, and the collector reads them as it reads a real
host.

**`linux.go`:**

```go
// (MIT header) — NO //go:build tag
package cpu

import (
    "context"
    "fmt"

    "github.com/avfs/avfs"
    "github.com/avfs/avfs/vfs/osfs"

    "github.com/osapi-io/gohai/internal/collector"
    "github.com/osapi-io/gohai/internal/executor"
)

type Linux struct {
    base

    FS   avfs.VFS          // files: /proc, /sys, /etc
    Exec executor.Executor // commands: lscpu, ip, df
}

func NewLinux() *Linux {
    return &Linux{FS: osfs.NewWithNoIdm(), Exec: executor.New()}
}

func (l *Linux) Collect(
    ctx context.Context,
    _ collector.PriorResults,
) (any, error) {
    b, err := l.FS.ReadFile("/proc/cpuinfo")
    if err != nil {
        return nil, fmt.Errorf("read /proc/cpuinfo: %w", err)
    }

    return parseCPUInfo(b), nil
}
```

`darwin.go` has the same shape, with `type Darwin struct`, `NewDarwin()` and its
own `Collect`. Darwin reads through `sysctl`, `ioreg` and other commands, run
through `Exec`. It has no build tag.

Add `debian.go` / `rhel.go` **only** when the distribution is really different
from generic Linux. For example, `package_mgr` needs apt or dnf. In all other
cases, the generic `Linux` struct covers all hosts that are not darwin.

Read the source the way Ohai reads it. When a fact has a value today, keep the
arithmetic that gives that value.

## Step 4: register in `pkg/gohai/gohai.go`

```go
func builtinCollectors(src source.Sources) []collector.Collector {
    return []collector.Collector{
        platform.NewWith(src),
        cpu.NewWith(src), // <-- add here
    }
}
```

Add the related `Facts` field and `set()` switch case in `pkg/gohai/facts.go`.

**This step can fail and show no error.** `registerBuiltins` discards the error
from each `Register` call, and no test examines it. Thus a duplicate name
removes a collector, and nothing reports it. The
[registration limitation](https://github.com/osapi-io/specs/blob/main/components/gohai/README.md#known-limitations)
gives the details. Before you open the pull request, make sure that
`NewRegistry().Names()` contains your collector.

## Step 5: tests (100% coverage, one test file)

**One file only**: `<name>_public_test.go`. Do not split into
`linux_public_test.go` or `darwin_public_test.go` files.

The file must have these contents, in this order:

1. Compile-time interface asserts at the top:
   ```go
   var (
       _ collector.Collector = (*cpu.Linux)(nil)
       _ collector.Collector = (*cpu.Darwin)(nil)
   )
   ```
2. `TestNew`. It gives `NewWith` a `source.Sources` with a `Family`, or replaces
   `platform.Detect` (a `var` that tests can replace) for `New`. It asserts that
   the factory returns the correct concrete type for each OS. It also asserts
   `Name()` / `DefaultEnabled()` / `Dependencies()`. It is table-driven.
3. `TestCollect`. **One** table-driven method, with a
   `variant: "linux" | "darwin"` column. Each row gives its files and its
   command output. The loop makes the correct struct for each OS:
   ```go
   switch tt.variant {
   case "linux":
       c = &cpu.Linux{FS: sourcetest.FS(s.T(), tt.files), Exec: tt.runner.Mock(s.T())}
   case "darwin":
       c = &cpu.Darwin{Exec: tt.runner.Mock(s.T())}
   }
   ```
   `tt.runner` is a `sourcetest.Commands`: its `Out` and `Err` maps answer a
   command line, such as `"sysctl -n hw.logicalcpu"`. `Mock` makes the generated
   `source/mocks.MockRunner` that answers them. Never write a runner by hand.
4. Optional: separate test methods for public helper functions that are really
   pure and independent (`TestHumanDuration`, `TestBytesToString`,
   `TestNeighFamily`). These are permitted because the helper has its own
   contract. **Do not** make `TestReadX` methods that test bridge code that
   `TestCollect` already tests.

**The test seams are the sources.** A row gives files, command output and a
clock. Use a package `var` that a test replaces only for a read that is not a
source, such as a DNS lookup. Do NOT add wrappers in the middle, such as
`readCPUFn = readCPU`. That is test-only code in production code, and the
consistency rule forbids it.

No test file has a build tag. `go test ./...` runs all tests on each development
OS.

**Coverage check:**

```bash
go test ./... -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out | grep -v '100.0%'
```

The expected result is empty output. All files are at 100%.

## Step 6: write the collector doc

`docs/collectors/<name>.md` is a functional spec that is complete in itself. It
must have these sections, in this order:

- `# <Name>` and **Status:** Implemented ✅
- **Description**. What the fact is, why consumers want it, and typical values
  on Linux and on macOS. Use our voice.
- **Signals**. Only for complex collectors. Refer to `docs/collectors/fips.md`
  as an example.
- **Collected fields**. A Markdown table with the columns Field / Type /
  Description / **Schema mapping**. Each field gives its OCSF path first
  (`os.kernel_release`, `device.cpu_count`). If OCSF has no field, it gives the
  OpenTelemetry attribute (`host.cpu.vendor.id`, `system.load_average.1m`). If
  neither has a field, it says "No direct schema" with a reason of one line.
- **Platform support**. A table.
- **Example output**. Realistic JSON for Linux and macOS.
- **SDK usage**. A Go snippet that uses `gohai.New(...).Collect(ctx)`.
- **Enable/disable**. The CLI flags.
- **Dependencies**. Other collectors.
- **Data sources**. The method, step by step, in OUR voice. Use a section for
  each OS when the behavior is different. No "vs. Ohai" parity comparison. Name
  Ohai in the text only when a specific method choice needs attribution.
- **Backing library**. The Go library or libraries that we wrap.

**No "Known gaps vs. Ohai" section.** Methodology gaps are GitHub issues with
the labels `methodology-gap` / `collector:<name>`.

Use `docs/collectors/shells.md` as the canonical template.

## Step 7: update `docs/collectors/README.md`

In `docs/collectors/README.md`, change the "Implemented" column of the collector
from 🚧 to ✅, with the source in parentheses: `✅ (procfs + sysctl)`,
`✅ (stdlib)`.

## Step 8: verify

```bash
go build ./...
go test ./... -count=1
just go-vet
go run . collect --collector.<name> --pretty
```

## Step 9: commit

```
feat(<name>): add <name> collector

Reads <files / commands> through the collector's sources, as Ohai's
<Y> does. Field names follow OCSF / OpenTelemetry conventions where
applicable (<list>). Supports Linux and macOS; NewWith selects the
variant from the family of its sources.

Closes #<N>  (if work closes a methodology issue)
```

Follow [Conventional Commits](https://www.conventionalcommits.org/) with the
50/72 rule. For the full conventions, refer to
[CONTRIBUTING.md#commit-messages](../CONTRIBUTING.md#commit-messages).
