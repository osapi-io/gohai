# Adding a New Collector

Step-by-step walkthrough for building a new collector. For the rules and
principles (library-first, OCSF naming, no build tags, etc.) see
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

// Info holds CPU information. Field names follow OCSF where applicable.
type Info struct {
    Total     int      `json:"total"`                // OCSF: device.cpu_count
    Cores     int      `json:"cores"`                // OCSF: device.cpu_cores
    ModelName string   `json:"model_name,omitempty"` // no OCSF
    Flags     []string `json:"flags,omitempty"`      // no OCSF
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

## Step 5 — Tests (100% coverage, no gopsutil leaks)

**`<name>_public_test.go`** — package-level tests:

- `TestNew` — calls `New()`, asserts `Name()` / `DefaultEnabled()` /
  `Dependencies()` on the returned interface. Stubs `platform.Detect` (a
  swappable `var`) to force every dispatch branch. **No gopsutil import.**
- `var _ collector.Collector = (*cpu.Linux)(nil)` at package level for
  compile-time interface assertion.

**`linux_public_test.go`** — tests the `Linux` struct directly with injected
stubs:

```go
c := &cpu.Linux{ReadFn: func(ctx context.Context) (*cpu.Info, error) {
    return &cpu.Info{Total: 8}, nil
}}
got, err := c.Collect(context.Background())
```

Table-driven scenarios covering happy path, error path, edge cases.

Also add a `TestReadCPU` in `linux_public_test.go` that uses the `SetInfoFn`
setter to exercise both the success and gopsutil-error branches of the private
wrapper.

**`darwin_public_test.go`** — same for `Darwin` struct.

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
- **Collected Fields** — markdown table with Field / Type / Description / **OCSF
  mapping** columns. Every field gets an OCSF row (`os.kernel_release`,
  `network_interface.mac`, etc.) or explicit "No direct OCSF" with a one-line
  reason.
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
follow OCSF conventions where applicable (<list>). Supports Linux and
macOS with platform.Detect() dispatch.

Closes #<N>  (if work closes a methodology issue)
```

Follow [Conventional Commits](https://www.conventionalcommits.org/) with the
50/72 rule. See
[docs/development.md#commit-messages](development.md#commit-messages) for the
full conventions.
