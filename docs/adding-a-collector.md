# Adding a New Collector

Step-by-step walkthrough for building a new collector. For the rules and
principles (library-first, OCSF + OpenTelemetry naming, no build tags, etc.) see
[CLAUDE.md](../CLAUDE.md).

**Reference implementation:** `pkg/gohai/collectors/shells/`. Copy its file
layout and patterns exactly.

## Done-definition

See
[CLAUDE.md — Done-definition](../CLAUDE.md#done-definition-every-collector-every-time).
Every item there must be true before marking the collector complete.

## Step 1 — Create the sub-package

Path: `pkg/gohai/collectors/<name>/` (public — consumers like OSAPI import
`Info` structs directly).

## Step 2 — `<name>.go` (top-level factory + shared surface)

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

## Step 3 — Per-OS struct implementations (no build tags)

Each file declares a struct for that OS, embeds `base`, and implements
`Collect`. Injectable fields make tests independent of the host OS.

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

`darwin.go` — same shape with `type Darwin struct`, `NewDarwin()`, and its own
`Collect`. No build tag. Runtime gracefully handles Linux-only paths.

`debian.go` / `rhel.go` — add **only** when the distro genuinely diverges from
generic Linux (e.g. `package_mgr` needs apt vs dnf). Otherwise generic `Linux`
covers all non-darwin.

The private gopsutil wrapper lives in `<name>.go`:

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

`export_test.go` exposes it to external tests:

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

## Step 4 — Register in `pkg/gohai/gohai.go`

```go
func builtinCollectors() []collector.Collector {
    return []collector.Collector{
        platform.New(),
        cpu.New(), // <-- add here
    }
}
```

Add matching `Facts` field + `set()` switch case in `pkg/gohai/facts.go`.

## Step 5 — Tests (100% coverage, one test file)

**One file only**: `<name>_public_test.go`. No `linux_public_test.go` or
`darwin_public_test.go` split files.

Required contents, in order:

1. Compile-time interface asserts at the top:
   ```go
   var (
       _ collector.Collector = (*cpu.Linux)(nil)
       _ collector.Collector = (*cpu.Darwin)(nil)
   )
   ```
2. `TestNew` — stubs `platform.Detect` (a swappable `var`), asserts the factory
   returns the right concrete type per OS plus `Name()` / `DefaultEnabled()` /
   `Dependencies()`. Table-driven.
3. `TestCollect` — **one** method, table-driven, with a
   `variant: "linux" | "darwin"` column. Each row sets up its seams, the loop
   constructs the right per-OS struct:
   ```go
   switch tt.variant {
   case "linux":
       c = &cpu.Linux{FS: tt.fs, Exec: tt.exec}
   case "darwin":
       c = &cpu.Darwin{Exec: tt.exec}
   }
   ```
4. Optional: separate test methods for genuinely pure, independent public helper
   functions (`TestHumanDuration`, `TestBytesToString`, `TestNeighFamily`).
   These are legitimate because the helper has its own contract — **do not**
   create `TestReadX` methods that shadow bridge code `TestCollect` already
   exercises.

**Test seams swap at the upstream library boundary.** `export_test.go` exposes
`Set<X>Fn` setters for the raw gopsutil / ghw / netlink calls (`SetHostInfoFn`,
`SetPartitionsFn`, `SetInterfacesFn`, etc.). Do NOT add intermediate wrappers
like `readCPUFn = readCPU` — that's test-only scaffolding in production code,
and the consistency rule forbids it.

No build tags on any test file. `go test ./...` on any dev OS runs every test.

**Coverage check:**

```bash
go test ./... -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out | grep -v '100.0%'
```

Expected: empty output (everything at 100%).

## Step 6 — Write the collector doc

`docs/collectors/<name>.md` is a self-contained functional spec. Required
sections in order:

- `# <Name>` and **Status:** Implemented ✅
- **Description** — what the fact is, why consumers want it, typical values on
  Linux vs macOS. Our voice.
- **Signals** (only for complex collectors — see `docs/collectors/fips.md` for
  reference).
- **Collected Fields** — markdown table with Field / Type / Description /
  **Schema mapping** columns. Every field cites its OCSF path first
  (`os.kernel_release`, `device.cpu_count`), OpenTelemetry attribute second
  (`host.cpu.vendor.id`, `system.load_average.1m`) when OCSF is silent, or
  explicit "No direct schema" with a one-line reason.
- **Platform Support** — table.
- **Example Output** — realistic JSON for Linux and macOS.
- **SDK Usage** — Go snippet using `gohai.New(...).Collect(ctx)`.
- **Enable/Disable** — CLI flags.
- **Dependencies** — other collectors.
- **Data Sources** — step-by-step methodology in OUR voice. Per-OS sections when
  behavior differs. No "vs. Ohai" parity comparison. Ohai mentioned inline only
  when a specific methodology choice needs attribution.
- **Backing library** — the Go library(s) we wrap.

**No "Known gaps vs. Ohai" section** — methodology gaps live as GitHub issues
labeled `methodology-gap` / `collector:<name>`.

Use `docs/collectors/shells.md` as the canonical template.

## Step 7 — Update `README.md`

Flip the "Implemented" column from 🚧 to ✅ (with the library in parentheses:
`✅ (gopsutil)`, `✅ (stdlib)`).

Also update `docs/collectors/README.md` if the collector status changes.

## Step 8 — Verify

```bash
go build ./...
go test ./... -count=1
just go::vet
go run . --collector.<name> --pretty
```

## Step 9 — Commit

```
feat(<name>): add <name> collector

Wraps <library> for the core collection, extending with <X> that the
library doesn't expose (mirrors Ohai's <Y> behavior). Field names
follow OCSF / OpenTelemetry conventions where applicable (<list>). Supports Linux and
macOS with platform.Detect() dispatch.

Closes #<N>  (if work closes a methodology issue)
```

Follow [Conventional Commits](https://www.conventionalcommits.org/) with the
50/72 rule. See
[docs/development.md#commit-messages](development.md#commit-messages) for the
full conventions.
