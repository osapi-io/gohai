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
    "github.com/osapi-io/gohai/internal/platform"
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

## Step 3: per-OS struct implementations (no build tags)

Each file declares a struct for its OS. The struct embeds `base` and implements
`Collect`. Injectable fields make the tests independent of the host OS.

**`linux.go`:**

```go
// (MIT header) — NO //go:build tag
package cpu

import (
    "context"

    "github.com/shirou/gopsutil/v4/cpu"
)

type Linux struct {
    base

    // Fn fields are typed in OUR *Info / []OurType, never gopsutil types.
    // The gopsutil call lives in a private package var swapped via
    // export_test.go — see pkg/gohai/collectors/uptime/ for the pattern.
    ReadFn func(context.Context) (*Info, error)
}

func NewLinux() *Linux {
    return &Linux{ReadFn: readCPU}
}

func (l *Linux) Collect(ctx context.Context) (any, error) {
    return l.ReadFn(ctx)
}
```

`darwin.go` has the same shape, with `type Darwin struct`, `NewDarwin()` and its
own `Collect`. It has no build tag. At runtime, the code handles the paths that
exist only on Linux without failure.

Add `debian.go` / `rhel.go` **only** when the distribution is really different
from generic Linux. For example, `package_mgr` needs apt or dnf. In all other
cases, the generic `Linux` struct covers all hosts that are not darwin.

The private gopsutil wrapper is in `<name>.go`:

```go
// Private injection seam — not exposed on any public Fn field.
var infoFn = cpu.InfoWithContext

func readCPU(ctx context.Context) (*Info, error) {
    stats, err := infoFn(ctx)
    if err != nil {
        return nil, fmt.Errorf("cpu.Info: %w", err)
    }
    // Map gopsutil stats → our Info.
    // Layer extensions for fields the library doesn't expose.
    return &Info{ /* mapped */ }, nil
}
```

`export_test.go` gives external tests access to it:

```go
package cpu

import (
    "context"

    "github.com/shirou/gopsutil/v4/cpu"
)

var ReadCPU = readCPU

func SetInfoFn(fn func(context.Context) ([]cpu.InfoStat, error)) (restore func()) {
    orig := infoFn
    infoFn = fn
    return func() { infoFn = orig }
}
```

## Step 4: register in `pkg/gohai/gohai.go`

```go
func builtinCollectors() []collector.Collector {
    return []collector.Collector{
        platform.New(),
        cpu.New(), // <-- add here
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
2. `TestNew`. It replaces `platform.Detect` (a `var` that tests can replace)
   with a stub. It asserts that the factory returns the correct concrete type
   for each OS. It also asserts `Name()` / `DefaultEnabled()` /
   `Dependencies()`. It is table-driven.
3. `TestCollect`. **One** table-driven method, with a
   `variant: "linux" | "darwin"` column. Each row sets its seams. The loop makes
   the correct struct for each OS:
   ```go
   switch tt.variant {
   case "linux":
       c = &cpu.Linux{FS: tt.fs, Exec: tt.exec}
   case "darwin":
       c = &cpu.Darwin{Exec: tt.exec}
   }
   ```
4. Optional: separate test methods for public helper functions that are really
   pure and independent (`TestHumanDuration`, `TestBytesToString`,
   `TestNeighFamily`). These are permitted because the helper has its own
   contract. **Do not** make `TestReadX` methods that test bridge code that
   `TestCollect` already tests.

**Test seams replace calls at the boundary of the upstream library.**
`export_test.go` gives `Set<X>Fn` setters for the raw gopsutil / ghw / netlink
calls (`SetHostInfoFn`, `SetPartitionsFn`, `SetInterfacesFn` and others). Do NOT
add wrappers in the middle, such as `readCPUFn = readCPU`. That is test-only
code in production code, and the consistency rule forbids it.

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

## Step 7: update `README.md`

Change the "Implemented" column from 🚧 to ✅, with the library in parentheses:
`✅ (gopsutil)`, `✅ (stdlib)`.

If the collector status changes, also update `docs/collectors/README.md`.

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

Wraps <library> for the core collection, extending with <X> that the
library doesn't expose (mirrors Ohai's <Y> behavior). Field names
follow OCSF / OpenTelemetry conventions where applicable (<list>). Supports Linux and
macOS with platform.Detect() dispatch.

Closes #<N>  (if work closes a methodology issue)
```

Follow [Conventional Commits](https://www.conventionalcommits.org/) with the
50/72 rule. For the full conventions, refer to
[CONTRIBUTING.md#commit-messages](../CONTRIBUTING.md#commit-messages).
