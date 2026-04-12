# gohai Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the gohai foundation (collector interface, registry, public SDK, CLI) and ship 3 representative collectors (platform, virtualization, docker) to validate the architecture end-to-end before implementing the remaining 62 collectors.

**Architecture:** Pluggable collector system. Each collector lives in its own sub-package under `internal/collector/`, implements a common interface, registers itself in an internal registry, and returns strongly-typed Go structs. Public SDK in `pkg/gohai/` exposes `New()`, `Collect()`, and a `Facts` struct with typed fields. CLI in `cmd/gohai/` uses Cobra with node_exporter-style `--collector.<name>` / `--no-collector.<name>` flags dynamically registered from the registry.

**Tech Stack:** Go 1.25.7, Cobra (CLI), testify/suite (tests), build tags for platform-specific code.

**Spec:** [docs/superpowers/specs/2026-04-11-gohai-design.md](../specs/2026-04-11-gohai-design.md)

## File Structure

```
internal/collector/
  collector.go                          # Collector interface, Tier enum
  collector_test.go                     # Tier tests
  registry.go                           # Internal registry (singleton)
  registry_test.go                      # Internal tests for registry
  registry_public_test.go               # Public API tests for registry
  platform/
    platform.go                         # Info struct, Collector type, Name/Tier/Deps
    platform_linux.go                   # //go:build linux — collect impl
    platform_darwin.go                  # //go:build darwin — collect impl
    platform_other.go                   # //go:build !linux && !darwin — stub
    platform_test.go                    # internal tests for parsing
    platform_public_test.go             # public interface tests
  virtualization/
    virtualization.go
    virtualization_linux.go
    virtualization_darwin.go
    virtualization_other.go
    virtualization_test.go
    virtualization_public_test.go
  docker/
    docker.go
    docker_linux.go
    docker_darwin.go
    docker_other.go
    docker_test.go
    docker_public_test.go
pkg/gohai/
  gohai.go                              # Gohai struct, New(), Collect()
  gohai_public_test.go
  facts.go                              # Facts struct, JSON, PrettyJSON, Flat
  facts_test.go                         # Internal tests for Flat key building
  facts_public_test.go                  # Public tests for JSON output
  options.go                            # WithCollectors, WithEnabled, WithDisabled
  options_public_test.go
cmd/gohai/
  main.go                               # Cobra root + run
  flags.go                              # Dynamic flag registration from registry
  output.go                             # JSON / pretty / flat / list-collectors
  main_test.go
```

---

## Task 1: Initialize Go module and dependencies

**Files:**
- Modify: `go.mod`
- Create: `go.sum` (via `go mod tidy`)

- [ ] **Step 1: Verify current go.mod**

Run: `cat go.mod`
Expected: shows `module github.com/osapi-io/gohai`, `go 1.25.7`, requires `cobra` and `testify`.

- [ ] **Step 2: Run go mod tidy to resolve deps**

Run: `cd /Users/john/git/osapi-io/gohai && go mod tidy`
Expected: `go.sum` created, no errors.

- [ ] **Step 3: Verify build works on empty project**

Run: `go build ./...`
Expected: PASS (nothing to build yet).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: resolve go module dependencies"
```

---

## Task 2: Collector interface and Tier type

**Files:**
- Create: `internal/collector/collector.go`
- Create: `internal/collector/collector_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/collector/collector_test.go`:

```go
package collector

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CollectorTestSuite struct {
	suite.Suite
}

func TestCollectorTestSuite(t *testing.T) {
	suite.Run(t, new(CollectorTestSuite))
}

func (s *CollectorTestSuite) TestTierString() {
	tests := []struct {
		name string
		tier Tier
		want string
	}{
		{"core", TierCore, "core"},
		{"extended", TierExtended, "extended"},
		{"opt-in", TierOptIn, "opt-in"},
		{"unknown", Tier(99), "unknown"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.tier.String())
		})
	}
}

func (s *CollectorTestSuite) TestTierEnabledByDefault() {
	tests := []struct {
		name string
		tier Tier
		want bool
	}{
		{"core enabled", TierCore, true},
		{"extended enabled", TierExtended, true},
		{"opt-in disabled", TierOptIn, false},
		{"unknown disabled", Tier(99), false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, tt.tier.EnabledByDefault())
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/collector/... -run TestCollectorTestSuite -v`
Expected: FAIL — `Tier`, `TierCore`, etc. undefined.

- [ ] **Step 3: Implement collector.go**

Create `internal/collector/collector.go`:

```go
// Package collector defines the Collector interface and tier classifications
// for gohai's pluggable fact collection system.
package collector

import "context"

// Tier classifies a collector for default enable/disable behavior.
type Tier int

const (
	// TierCore collectors are foundational and enabled by default.
	TierCore Tier = iota + 1
	// TierExtended collectors are enabled by default but cover broader topics.
	TierExtended
	// TierOptIn collectors are disabled by default and require explicit opt-in.
	TierOptIn
)

// String returns the human-readable name of the tier.
func (t Tier) String() string {
	switch t {
	case TierCore:
		return "core"
	case TierExtended:
		return "extended"
	case TierOptIn:
		return "opt-in"
	default:
		return "unknown"
	}
}

// EnabledByDefault reports whether collectors of this tier run by default.
func (t Tier) EnabledByDefault() bool {
	return t == TierCore || t == TierExtended
}

// Collector is the interface every fact collector must implement.
type Collector interface {
	// Name returns the unique identifier (e.g., "platform").
	Name() string

	// Tier returns the tier classification.
	Tier() Tier

	// Dependencies returns names of collectors that must run before this one.
	Dependencies() []string

	// Collect gathers facts and returns a typed struct (or nil if not supported).
	Collect(
		ctx context.Context,
	) (any, error)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/collector/... -run TestCollectorTestSuite -v`
Expected: PASS.

- [ ] **Step 5: Verify 100% coverage**

Run: `go test ./internal/collector/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | grep -v 'total:'`
Expected: every function shows `100.0%`.

- [ ] **Step 6: Commit**

```bash
git add internal/collector/collector.go internal/collector/collector_test.go
git commit -m "feat(collector): add Collector interface and Tier type"
```

---

## Task 3: Registry — registration and lookup

**Files:**
- Create: `internal/collector/registry.go`
- Create: `internal/collector/registry_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/collector/registry_test.go`:

```go
package collector

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
	reg *Registry
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (s *RegistryTestSuite) SetupTest() {
	s.reg = NewRegistry()
}

type fakeCollector struct {
	name string
	tier Tier
	deps []string
}

func (f *fakeCollector) Name() string         { return f.name }
func (f *fakeCollector) Tier() Tier           { return f.tier }
func (f *fakeCollector) Dependencies() []string { return f.deps }
func (f *fakeCollector) Collect(_ context.Context) (any, error) {
	return f.name + "-result", nil
}

func (s *RegistryTestSuite) TestRegister() {
	tests := []struct {
		name      string
		collector Collector
		wantErr   bool
	}{
		{
			name:      "registers a new collector",
			collector: &fakeCollector{name: "alpha", tier: TierCore},
			wantErr:   false,
		},
		{
			name:      "rejects empty name",
			collector: &fakeCollector{name: "", tier: TierCore},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			reg := NewRegistry()
			err := reg.Register(tt.collector)
			if tt.wantErr {
				s.Error(err)
				return
			}
			s.NoError(err)
			got, ok := reg.Get(tt.collector.Name())
			s.True(ok)
			s.Equal(tt.collector, got)
		})
	}
}

func (s *RegistryTestSuite) TestRegisterDuplicate() {
	c := &fakeCollector{name: "dup", tier: TierCore}
	s.Require().NoError(s.reg.Register(c))
	err := s.reg.Register(c)
	s.Error(err)
}

func (s *RegistryTestSuite) TestNames() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", tier: TierCore}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", tier: TierCore}))
	names := s.reg.Names()
	sort.Strings(names)
	s.Equal([]string{"a", "b"}, names)
}

func (s *RegistryTestSuite) TestGetMissing() {
	_, ok := s.reg.Get("missing")
	s.False(ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/collector/ -run TestRegistryTestSuite -v`
Expected: FAIL — `Registry`, `NewRegistry`, etc. undefined.

- [ ] **Step 3: Implement registry.go**

Create `internal/collector/registry.go`:

```go
package collector

import (
	"errors"
	"fmt"
	"sync"
)

// Registry holds the set of registered collectors.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register adds a collector to the registry. Returns an error if the name is
// empty or already registered.
func (r *Registry) Register(
	c Collector,
) error {
	if c.Name() == "" {
		return errors.New("collector name must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collectors[c.Name()]; exists {
		return fmt.Errorf("collector %q already registered", c.Name())
	}
	r.collectors[c.Name()] = c
	return nil
}

// Get returns the collector with the given name, if registered.
func (r *Registry) Get(
	name string,
) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

// Names returns the names of all registered collectors.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.collectors))
	for n := range r.collectors {
		names = append(names, n)
	}
	return names
}
```

- [ ] **Step 4: Run test and verify pass**

Run: `go test ./internal/collector/ -run TestRegistryTestSuite -v`
Expected: PASS.

- [ ] **Step 5: Verify 100% coverage**

Run: `go test ./internal/collector/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: every Registry function shows `100.0%`.

- [ ] **Step 6: Commit**

```bash
git add internal/collector/registry.go internal/collector/registry_test.go
git commit -m "feat(collector): add Registry for collector registration"
```

---

## Task 4: Registry — enable/disable selection logic

**Files:**
- Modify: `internal/collector/registry.go`
- Modify: `internal/collector/registry_test.go`

- [ ] **Step 1: Add failing test**

Append to `internal/collector/registry_test.go`:

```go
func (s *RegistryTestSuite) TestSelected() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "core1", tier: TierCore}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "core2", tier: TierCore}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "ext", tier: TierExtended}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "opt", tier: TierOptIn}))

	tests := []struct {
		name    string
		enable  []string
		disable []string
		want    []string
	}{
		{
			name: "defaults: core+extended on, opt-in off",
			want: []string{"core1", "core2", "ext"},
		},
		{
			name:    "disable a default-on collector",
			disable: []string{"core1"},
			want:    []string{"core2", "ext"},
		},
		{
			name:   "enable an opt-in collector",
			enable: []string{"opt"},
			want:   []string{"core1", "core2", "ext", "opt"},
		},
		{
			name:    "disable wins over enable for same name",
			enable:  []string{"opt"},
			disable: []string{"opt"},
			want:    []string{"core1", "core2", "ext"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.reg.Selected(tt.enable, tt.disable)
			s.Require().NoError(err)
			names := make([]string, 0, len(got))
			for _, c := range got {
				names = append(names, c.Name())
			}
			sort.Strings(names)
			s.Equal(tt.want, names)
		})
	}
}

func (s *RegistryTestSuite) TestSelectedUnknown() {
	_, err := s.reg.Selected([]string{"missing"}, nil)
	s.Error(err)

	_, err = s.reg.Selected(nil, []string{"missing"})
	s.Error(err)
}
```

- [ ] **Step 2: Run test, verify failure**

Run: `go test ./internal/collector/ -run TestRegistryTestSuite/TestSelected -v`
Expected: FAIL — `Selected` undefined.

- [ ] **Step 3: Implement Selected**

Append to `internal/collector/registry.go`:

```go
// Selected returns the collectors that should run given the user's enable and
// disable lists. Defaults: TierCore and TierExtended are on, TierOptIn is off.
// Disable wins over enable for the same name. Unknown names return an error.
func (r *Registry) Selected(
	enable []string,
	disable []string,
) ([]Collector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, n := range enable {
		if _, ok := r.collectors[n]; !ok {
			return nil, fmt.Errorf("unknown collector %q in enable list", n)
		}
	}
	for _, n := range disable {
		if _, ok := r.collectors[n]; !ok {
			return nil, fmt.Errorf("unknown collector %q in disable list", n)
		}
	}

	enableSet := make(map[string]bool, len(enable))
	for _, n := range enable {
		enableSet[n] = true
	}
	disableSet := make(map[string]bool, len(disable))
	for _, n := range disable {
		disableSet[n] = true
	}

	out := make([]Collector, 0, len(r.collectors))
	for name, c := range r.collectors {
		if disableSet[name] {
			continue
		}
		if c.Tier().EnabledByDefault() || enableSet[name] {
			out = append(out, c)
		}
	}
	return out, nil
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/collector/ -v`
Expected: PASS.

- [ ] **Step 5: Verify 100% coverage**

Run: `go test ./internal/collector/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: 100% on all functions.

- [ ] **Step 6: Commit**

```bash
git add internal/collector/registry.go internal/collector/registry_test.go
git commit -m "feat(collector): add Selected for enable/disable selection"
```

---

## Task 5: Registry — dependency resolution and concurrent execution

**Files:**
- Modify: `internal/collector/registry.go`
- Modify: `internal/collector/registry_test.go`

- [ ] **Step 1: Add failing test**

Append to `internal/collector/registry_test.go`:

```go
func (s *RegistryTestSuite) TestRunOrdersByDependency() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", tier: TierCore}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", tier: TierCore, deps: []string{"a"}}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "c", tier: TierCore, deps: []string{"b"}}))

	results, err := s.reg.Run(context.Background(), []string{"a", "b", "c"}, nil)
	s.Require().NoError(err)
	s.Equal("a-result", results["a"])
	s.Equal("b-result", results["b"])
	s.Equal("c-result", results["c"])
}

func (s *RegistryTestSuite) TestRunAutoIncludesDependencies() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", tier: TierOptIn}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", tier: TierOptIn, deps: []string{"a"}}))

	results, err := s.reg.Run(context.Background(), []string{"b"}, nil)
	s.Require().NoError(err)
	s.Contains(results, "a")
	s.Contains(results, "b")
}

func (s *RegistryTestSuite) TestRunDetectsCycle() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", tier: TierCore, deps: []string{"b"}}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "b", tier: TierCore, deps: []string{"a"}}))

	_, err := s.reg.Run(context.Background(), []string{"a", "b"}, nil)
	s.Error(err)
	s.Contains(err.Error(), "cycle")
}

func (s *RegistryTestSuite) TestRunMissingDependency() {
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "a", tier: TierCore, deps: []string{"missing"}}))
	_, err := s.reg.Run(context.Background(), []string{"a"}, nil)
	s.Error(err)
}

type errCollector struct{ name string }

func (e *errCollector) Name() string                            { return e.name }
func (e *errCollector) Tier() Tier                              { return TierCore }
func (e *errCollector) Dependencies() []string                  { return nil }
func (e *errCollector) Collect(_ context.Context) (any, error)  { return nil, errors.New("boom") }

func (s *RegistryTestSuite) TestRunCollectsErrors() {
	s.Require().NoError(s.reg.Register(&errCollector{name: "bad"}))
	s.Require().NoError(s.reg.Register(&fakeCollector{name: "good", tier: TierCore}))

	results, err := s.reg.Run(context.Background(), []string{"bad", "good"}, nil)
	s.Require().NoError(err)
	s.Equal("good-result", results["good"])
	s.NotContains(results, "bad") // failed collectors omitted
}
```

Add `"errors"` to imports.

- [ ] **Step 2: Run, verify failure**

Run: `go test ./internal/collector/ -v`
Expected: FAIL — `Run` undefined.

- [ ] **Step 3: Implement Run with topological sort and concurrency**

Append to `internal/collector/registry.go`:

```go
// Run executes the named collectors in dependency order. Dependencies are
// auto-included even if not in the names list. Collectors at the same level
// (no inter-dependencies) run concurrently. Returns a map of collector name
// to result. Failed collectors are omitted from the result map.
func (r *Registry) Run(
	ctx context.Context,
	names []string,
	onError func(name string, err error),
) (map[string]any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wanted, err := r.expandWithDeps(names)
	if err != nil {
		return nil, err
	}

	levels, err := r.topoLevels(wanted)
	if err != nil {
		return nil, err
	}

	results := make(map[string]any, len(wanted))
	var resultsMu sync.Mutex

	for _, level := range levels {
		var wg sync.WaitGroup
		for _, name := range level {
			c := r.collectors[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				out, err := c.Collect(ctx)
				if err != nil {
					if onError != nil {
						onError(c.Name(), err)
					}
					return
				}
				resultsMu.Lock()
				results[c.Name()] = out
				resultsMu.Unlock()
			}()
		}
		wg.Wait()
	}
	return results, nil
}

func (r *Registry) expandWithDeps(
	names []string,
) (map[string]bool, error) {
	wanted := make(map[string]bool)
	var visit func(string) error
	visit = func(n string) error {
		if wanted[n] {
			return nil
		}
		c, ok := r.collectors[n]
		if !ok {
			return fmt.Errorf("unknown collector %q", n)
		}
		wanted[n] = true
		for _, dep := range c.Dependencies() {
			if err := visit(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range names {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return wanted, nil
}

func (r *Registry) topoLevels(
	wanted map[string]bool,
) ([][]string, error) {
	indeg := make(map[string]int, len(wanted))
	for n := range wanted {
		indeg[n] = 0
	}
	for n := range wanted {
		for _, dep := range r.collectors[n].Dependencies() {
			if !wanted[dep] {
				continue
			}
			indeg[n]++
		}
	}

	var levels [][]string
	remaining := len(indeg)
	for remaining > 0 {
		var level []string
		for n, d := range indeg {
			if d == 0 {
				level = append(level, n)
			}
		}
		if len(level) == 0 {
			return nil, fmt.Errorf("dependency cycle detected among collectors")
		}
		sort.Strings(level)
		levels = append(levels, level)
		for _, n := range level {
			delete(indeg, n)
			remaining--
			for m := range indeg {
				for _, dep := range r.collectors[m].Dependencies() {
					if dep == n {
						indeg[m]--
					}
				}
			}
		}
	}
	return levels, nil
}
```

Add `"sort"` to imports.

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/collector/ -v`
Expected: PASS.

- [ ] **Step 5: Verify 100% coverage**

Run: `go test ./internal/collector/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: 100% on all functions.

- [ ] **Step 6: Commit**

```bash
git add internal/collector/registry.go internal/collector/registry_test.go
git commit -m "feat(collector): add dependency-aware concurrent execution"
```

---

## Task 6: Platform collector — Info struct and base type

**Files:**
- Create: `internal/collector/platform/platform.go`
- Create: `internal/collector/platform/platform_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/collector/platform/platform_test.go`:

```go
package platform

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type PlatformTestSuite struct {
	suite.Suite
}

func TestPlatformTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformTestSuite))
}

func (s *PlatformTestSuite) TestNew() {
	c := New()
	s.Equal("platform", c.Name())
	s.Equal(collector.TierCore, c.Tier())
	s.Empty(c.Dependencies())
}
```

- [ ] **Step 2: Run test, verify failure**

Run: `go test ./internal/collector/platform/ -v`
Expected: FAIL — package empty.

- [ ] **Step 3: Implement platform.go**

Create `internal/collector/platform/platform.go`:

```go
// Package platform collects operating system platform identification facts.
package platform

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds platform identification data.
type Info struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Family       string `json:"family"`
	Architecture string `json:"architecture"`
	Build        string `json:"build,omitempty"`
}

// Collector implements the collector.Collector interface for platform facts.
type Collector struct{}

// New returns a new platform Collector.
func New() *Collector {
	return &Collector{}
}

// Name returns "platform".
func (c *Collector) Name() string { return "platform" }

// Tier returns TierCore.
func (c *Collector) Tier() collector.Tier { return collector.TierCore }

// Dependencies returns no dependencies.
func (c *Collector) Dependencies() []string { return nil }

// Collect gathers platform facts. Implementation lives in platform_<os>.go.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
```

- [ ] **Step 4: Run, verify failure (collect undefined)**

Run: `go build ./internal/collector/platform/...`
Expected: FAIL — `collect` undefined for current OS.

- [ ] **Step 5: Add stub for unsupported platforms**

Create `internal/collector/platform/platform_other.go`:

```go
//go:build !linux && !darwin

package platform

import "context"

func collect(_ context.Context) (any, error) {
	return nil, nil
}
```

- [ ] **Step 6: Verify build still fails on linux/darwin (need impls)**

Run: `go build ./internal/collector/platform/...`
Expected: depends on host OS — if linux or darwin, will FAIL until next task.

If failing, proceed; if passing (other OS), continue anyway.

- [ ] **Step 7: Commit (will compile on other-OS only; commit anyway, fix in next tasks)**

```bash
git add internal/collector/platform/
git commit -m "feat(platform): add platform Collector type and Info struct"
```

---

## Task 7: Platform collector — Linux implementation

**Files:**
- Create: `internal/collector/platform/platform_linux.go`
- Create: `internal/collector/platform/platform_linux_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/collector/platform/platform_linux_test.go`:

```go
//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PlatformLinuxTestSuite struct {
	suite.Suite
	tmpDir string
}

func TestPlatformLinuxTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformLinuxTestSuite))
}

func (s *PlatformLinuxTestSuite) SetupTest() {
	s.tmpDir = s.T().TempDir()
}

func (s *PlatformLinuxTestSuite) writeOSRelease(content string) string {
	path := filepath.Join(s.tmpDir, "os-release")
	s.Require().NoError(os.WriteFile(path, []byte(content), 0o644))
	return path
}

func (s *PlatformLinuxTestSuite) TestParseOSRelease() {
	tests := []struct {
		name    string
		content string
		want    Info
	}{
		{
			name: "ubuntu 24.04",
			content: `NAME="Ubuntu"
ID=ubuntu
ID_LIKE=debian
VERSION_ID="24.04"
VERSION="24.04.1 LTS (Noble Numbat)"
`,
			want: Info{Name: "ubuntu", Version: "24.04", Family: "debian"},
		},
		{
			name: "rhel 9",
			content: `ID="rhel"
ID_LIKE="fedora"
VERSION_ID="9.3"
`,
			want: Info{Name: "rhel", Version: "9.3", Family: "rhel"},
		},
		{
			name: "alpine no ID_LIKE",
			content: `ID=alpine
VERSION_ID=3.19.0
`,
			want: Info{Name: "alpine", Version: "3.19.0", Family: "alpine"},
		},
		{
			name:    "empty file",
			content: "",
			want:    Info{},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			path := s.writeOSRelease(tt.content)
			got, err := parseOSRelease(path)
			s.Require().NoError(err)
			tt.want.Architecture = got.Architecture // arch is from runtime
			s.Equal(tt.want, *got)
		})
	}
}

func (s *PlatformLinuxTestSuite) TestParseOSReleaseMissingFile() {
	_, err := parseOSRelease(filepath.Join(s.tmpDir, "nope"))
	s.Error(err)
}

func (s *PlatformLinuxTestSuite) TestRemapID() {
	tests := []struct {
		id   string
		want string
	}{
		{"amzn", "amazon"},
		{"ol", "oracle"},
		{"rhel", "rhel"},
		{"sles", "suse"},
		{"opensuse-leap", "suse"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		s.Run(tt.id, func() {
			s.Equal(tt.want, remapID(tt.id))
		})
	}
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./internal/collector/platform/ -v`
Expected: FAIL — `parseOSRelease`, `remapID` undefined.

- [ ] **Step 3: Implement platform_linux.go**

Create `internal/collector/platform/platform_linux.go`:

```go
//go:build linux

package platform

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
)

const osReleasePath = "/etc/os-release"

func collect(_ context.Context) (any, error) {
	return parseOSRelease(osReleasePath)
}

func parseOSRelease(
	path string,
) (*Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	info := &Info{Architecture: runtime.GOARCH}
	var idLike string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, val, ok := splitKV(scanner.Text())
		if !ok {
			continue
		}
		switch key {
		case "ID":
			info.Name = remapID(val)
		case "VERSION_ID":
			info.Version = val
		case "ID_LIKE":
			idLike = val
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	info.Family = deriveFamily(info.Name, idLike)
	return info, nil
}

func splitKV(
	line string,
) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key := line[:idx]
	val := strings.Trim(line[idx+1:], `"`)
	return key, val, true
}

func remapID(
	id string,
) string {
	switch id {
	case "amzn":
		return "amazon"
	case "ol":
		return "oracle"
	case "sles", "opensuse-leap", "opensuse":
		return "suse"
	default:
		return id
	}
}

func deriveFamily(
	name, idLike string,
) string {
	if idLike != "" {
		first := strings.Fields(idLike)[0]
		return remapID(first)
	}
	return name
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/collector/platform/ -v`
Expected: PASS on linux. On non-linux, the linux test file is excluded by build tag — it should still pass.

- [ ] **Step 5: Verify coverage on linux**

Run: `go test ./internal/collector/platform/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: parseOSRelease, splitKV, remapID, deriveFamily all 100%.

- [ ] **Step 6: Commit**

```bash
git add internal/collector/platform/platform_linux.go internal/collector/platform/platform_linux_test.go
git commit -m "feat(platform): add Linux /etc/os-release parser"
```

---

## Task 8: Platform collector — macOS implementation

**Files:**
- Create: `internal/collector/platform/platform_darwin.go`
- Create: `internal/collector/platform/platform_darwin_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/collector/platform/platform_darwin_test.go`:

```go
//go:build darwin

package platform

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PlatformDarwinTestSuite struct {
	suite.Suite
}

func TestPlatformDarwinTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformDarwinTestSuite))
}

func (s *PlatformDarwinTestSuite) TestParseSwVers() {
	tests := []struct {
		name string
		out  string
		want Info
	}{
		{
			name: "macOS 14",
			out:  "ProductName:\tmacOS\nProductVersion:\t14.4.1\nBuildVersion:\t23E224\n",
			want: Info{Name: "mac_os_x", Version: "14.4.1", Family: "mac_os_x", Build: "23E224"},
		},
		{
			name: "missing build",
			out:  "ProductName:\tmacOS\nProductVersion:\t13.0\n",
			want: Info{Name: "mac_os_x", Version: "13.0", Family: "mac_os_x"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := parseSwVers(tt.out)
			tt.want.Architecture = got.Architecture
			s.Equal(tt.want, *got)
		})
	}
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./internal/collector/platform/ -v` (skip if not on darwin — file is build-tagged out)
Expected on darwin: FAIL — `parseSwVers` undefined.

- [ ] **Step 3: Implement platform_darwin.go**

Create `internal/collector/platform/platform_darwin.go`:

```go
//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func collect(
	ctx context.Context,
) (any, error) {
	out, err := exec.CommandContext(ctx, "sw_vers").Output()
	if err != nil {
		return nil, fmt.Errorf("run sw_vers: %w", err)
	}
	return parseSwVers(string(out)), nil
}

func parseSwVers(
	out string,
) *Info {
	info := &Info{Name: "mac_os_x", Family: "mac_os_x", Architecture: runtime.GOARCH}
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch key {
		case "ProductVersion":
			info.Version = val
		case "BuildVersion":
			info.Build = val
		}
	}
	return info
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./internal/collector/platform/ -v`
Expected: PASS on darwin. On linux, darwin test is excluded.

- [ ] **Step 5: Verify TestNew passes on all platforms**

Run: `go test ./internal/collector/platform/ -run PlatformTestSuite -v`
Expected: PASS on linux, darwin, and other.

- [ ] **Step 6: Commit**

```bash
git add internal/collector/platform/platform_darwin.go internal/collector/platform/platform_darwin_test.go
git commit -m "feat(platform): add macOS sw_vers parser"
```

---

## Task 9: Platform collector — public test for interface contract

**Files:**
- Create: `internal/collector/platform/platform_public_test.go`

- [ ] **Step 1: Write public test**

Create `internal/collector/platform/platform_public_test.go`:

```go
package platform_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/collector/platform"
)

type PlatformPublicTestSuite struct {
	suite.Suite
}

func TestPlatformPublicTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformPublicTestSuite))
}

func (s *PlatformPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = platform.New()
}

func (s *PlatformPublicTestSuite) TestCollectReturnsInfoOrNil() {
	c := platform.New()
	got, err := c.Collect(context.Background())
	s.Require().NoError(err)
	if got == nil {
		return // unsupported OS
	}
	info, ok := got.(*platform.Info)
	s.Require().True(ok, "expected *platform.Info, got %T", got)
	s.NotEmpty(info.Architecture)
}
```

- [ ] **Step 2: Run, verify pass**

Run: `go test ./internal/collector/platform/ -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/collector/platform/platform_public_test.go
git commit -m "test(platform): add public interface contract test"
```

---

## Task 10: Virtualization collector — types and Linux implementation

**Files:**
- Create: `internal/collector/virtualization/virtualization.go`
- Create: `internal/collector/virtualization/virtualization_linux.go`
- Create: `internal/collector/virtualization/virtualization_darwin.go`
- Create: `internal/collector/virtualization/virtualization_other.go`
- Create: `internal/collector/virtualization/virtualization_test.go`
- Create: `internal/collector/virtualization/virtualization_linux_test.go`
- Create: `internal/collector/virtualization/virtualization_public_test.go`

- [ ] **Step 1: Write base test**

Create `internal/collector/virtualization/virtualization_test.go`:

```go
package virtualization

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type VirtualizationTestSuite struct {
	suite.Suite
}

func TestVirtualizationTestSuite(t *testing.T) {
	suite.Run(t, new(VirtualizationTestSuite))
}

func (s *VirtualizationTestSuite) TestNew() {
	c := New()
	s.Equal("virtualization", c.Name())
	s.Equal(collector.TierExtended, c.Tier())
	s.Equal([]string{"dmi", "cpu"}, c.Dependencies())
}
```

- [ ] **Step 2: Implement base virtualization.go**

Create `internal/collector/virtualization/virtualization.go`:

```go
// Package virtualization detects hypervisor and container runtime presence.
package virtualization

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds detected virtualization data.
type Info struct {
	System  string            `json:"system,omitempty"`
	Role    string            `json:"role,omitempty"`
	Systems map[string]string `json:"systems,omitempty"`
}

// Collector implements collector.Collector for virtualization facts.
type Collector struct{}

// New returns a new virtualization Collector.
func New() *Collector { return &Collector{} }

// Name returns "virtualization".
func (c *Collector) Name() string { return "virtualization" }

// Tier returns TierExtended.
func (c *Collector) Tier() collector.Tier { return collector.TierExtended }

// Dependencies returns dmi and cpu.
func (c *Collector) Dependencies() []string { return []string{"dmi", "cpu"} }

// Collect detects virtualization. Implementation in virtualization_<os>.go.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
```

- [ ] **Step 3: Add unsupported-OS stub**

Create `internal/collector/virtualization/virtualization_other.go`:

```go
//go:build !linux && !darwin

package virtualization

import "context"

func collect(_ context.Context) (any, error) {
	return nil, nil
}
```

- [ ] **Step 4: Add Linux test**

Create `internal/collector/virtualization/virtualization_linux_test.go`:

```go
//go:build linux

package virtualization

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type VirtLinuxTestSuite struct {
	suite.Suite
}

func TestVirtLinuxTestSuite(t *testing.T) {
	suite.Run(t, new(VirtLinuxTestSuite))
}

func (s *VirtLinuxTestSuite) TestDetectFromCgroup() {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"docker", "12:devices:/docker/abc123\n", "docker"},
		{"lxc", "1:name=systemd:/lxc/container\n", "lxc"},
		{"podman", "0::/machine.slice/libpod-abc.scope\n", "podman"},
		{"none", "12:devices:/user.slice\n", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, detectFromCgroup(tt.content))
		})
	}
}

func (s *VirtLinuxTestSuite) TestDetectFromDockerEnv() {
	s.True(detectFromFile("/dev/null"))   // exists
	s.False(detectFromFile("/no/such/path"))
}
```

- [ ] **Step 5: Implement Linux collector**

Create `internal/collector/virtualization/virtualization_linux.go`:

```go
//go:build linux

package virtualization

import (
	"context"
	"os"
	"strings"
)

func collect(_ context.Context) (any, error) {
	info := &Info{Systems: make(map[string]string)}

	if data, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		if sys := detectFromCgroup(string(data)); sys != "" {
			info.System = sys
			info.Role = "guest"
			info.Systems[sys] = "guest"
		}
	}

	if detectFromFile("/.dockerenv") && info.System == "" {
		info.System = "docker"
		info.Role = "guest"
		info.Systems["docker"] = "guest"
	}
	return info, nil
}

func detectFromCgroup(
	content string,
) string {
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.Contains(line, "/docker/"):
			return "docker"
		case strings.Contains(line, "/lxc/"):
			return "lxc"
		case strings.Contains(line, "libpod-"):
			return "podman"
		}
	}
	return ""
}

func detectFromFile(
	path string,
) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

- [ ] **Step 6: Add Darwin stub**

Create `internal/collector/virtualization/virtualization_darwin.go`:

```go
//go:build darwin

package virtualization

import "context"

func collect(_ context.Context) (any, error) {
	// macOS virtualization detection is complex (sysctl kern.hv_vmm_present,
	// ioreg, sysctl machdep.cpu.features). For Phase 1, return empty Info.
	return &Info{Systems: make(map[string]string)}, nil
}
```

- [ ] **Step 7: Add public test**

Create `internal/collector/virtualization/virtualization_public_test.go`:

```go
package virtualization_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/collector/virtualization"
)

type VirtPublicTestSuite struct {
	suite.Suite
}

func TestVirtPublicTestSuite(t *testing.T) {
	suite.Run(t, new(VirtPublicTestSuite))
}

func (s *VirtPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = virtualization.New()
}

func (s *VirtPublicTestSuite) TestCollectReturnsInfo() {
	c := virtualization.New()
	got, err := c.Collect(context.Background())
	s.Require().NoError(err)
	if got == nil {
		return
	}
	_, ok := got.(*virtualization.Info)
	s.True(ok)
}
```

- [ ] **Step 8: Run all tests**

Run: `go test ./internal/collector/virtualization/ -v`
Expected: PASS.

- [ ] **Step 9: Verify coverage**

Run: `go test ./internal/collector/virtualization/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: 100% on parser functions; collect() may have uncovered lines depending on host (acceptable for OS-conditional code, documented in spec).

- [ ] **Step 10: Commit**

```bash
git add internal/collector/virtualization/
git commit -m "feat(virtualization): add hypervisor/container detection"
```

---

## Task 11: Docker collector

**Files:**
- Create: `internal/collector/docker/docker.go`
- Create: `internal/collector/docker/docker_linux.go`
- Create: `internal/collector/docker/docker_darwin.go`
- Create: `internal/collector/docker/docker_other.go`
- Create: `internal/collector/docker/docker_test.go`
- Create: `internal/collector/docker/docker_unix_test.go`
- Create: `internal/collector/docker/docker_public_test.go`

- [ ] **Step 1: Write base test**

Create `internal/collector/docker/docker_test.go`:

```go
package docker

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
)

type DockerTestSuite struct {
	suite.Suite
}

func TestDockerTestSuite(t *testing.T) {
	suite.Run(t, new(DockerTestSuite))
}

func (s *DockerTestSuite) TestNew() {
	c := New()
	s.Equal("docker", c.Name())
	s.Equal(collector.TierOptIn, c.Tier())
	s.Equal([]string{"virtualization"}, c.Dependencies())
}
```

- [ ] **Step 2: Implement base docker.go**

Create `internal/collector/docker/docker.go`:

```go
// Package docker collects Docker daemon and container facts.
package docker

import (
	"context"

	"github.com/osapi-io/gohai/internal/collector"
)

// Info holds Docker daemon facts.
type Info struct {
	Version    string `json:"version,omitempty"`
	Containers int    `json:"containers,omitempty"`
	Images     int    `json:"images,omitempty"`
	ServerOS   string `json:"server_os,omitempty"`
	Available  bool   `json:"available"`
}

// Collector implements collector.Collector for Docker facts.
type Collector struct{}

// New returns a new Docker Collector.
func New() *Collector { return &Collector{} }

// Name returns "docker".
func (c *Collector) Name() string { return "docker" }

// Tier returns TierOptIn.
func (c *Collector) Tier() collector.Tier { return collector.TierOptIn }

// Dependencies returns virtualization (Docker depends on vt detection).
func (c *Collector) Dependencies() []string { return []string{"virtualization"} }

// Collect runs `docker info` and parses the result.
func (c *Collector) Collect(
	ctx context.Context,
) (any, error) {
	return collect(ctx)
}
```

- [ ] **Step 3: Add unix test (linux + darwin)**

Create `internal/collector/docker/docker_unix_test.go`:

```go
//go:build linux || darwin

package docker

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DockerUnixTestSuite struct {
	suite.Suite
}

func TestDockerUnixTestSuite(t *testing.T) {
	suite.Run(t, new(DockerUnixTestSuite))
}

func (s *DockerUnixTestSuite) TestParseDockerInfo() {
	tests := []struct {
		name string
		out  string
		want Info
	}{
		{
			name: "typical output",
			out: `Server Version: 25.0.3
Containers: 5
Images: 12
Operating System: Ubuntu 24.04
`,
			want: Info{Version: "25.0.3", Containers: 5, Images: 12, ServerOS: "Ubuntu 24.04", Available: true},
		},
		{
			name: "minimal output",
			out:  `Server Version: 24.0.0`,
			want: Info{Version: "24.0.0", Available: true},
		},
		{
			name: "empty",
			out:  "",
			want: Info{Available: true},
		},
		{
			name: "non-numeric containers",
			out:  "Containers: abc\nServer Version: 1.0",
			want: Info{Version: "1.0", Available: true},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.want, *parseDockerInfo(tt.out))
		})
	}
}
```

- [ ] **Step 4: Implement Linux/Darwin shared code**

Create `internal/collector/docker/docker_linux.go`:

```go
//go:build linux

package docker

import "context"

func collect(ctx context.Context) (any, error) {
	return collectUnix(ctx)
}
```

Create `internal/collector/docker/docker_darwin.go`:

```go
//go:build darwin

package docker

import "context"

func collect(ctx context.Context) (any, error) {
	return collectUnix(ctx)
}
```

Create `internal/collector/docker/docker_other.go`:

```go
//go:build !linux && !darwin

package docker

import "context"

func collect(_ context.Context) (any, error) {
	return nil, nil
}
```

Create `internal/collector/docker/docker_unix.go`:

```go
//go:build linux || darwin

package docker

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

func collectUnix(ctx context.Context) (any, error) {
	out, err := exec.CommandContext(ctx, "docker", "info").Output()
	if err != nil {
		// Docker not installed or daemon not running — return nil (omit fact)
		return nil, nil
	}
	return parseDockerInfo(string(out)), nil
}

func parseDockerInfo(
	out string,
) *Info {
	info := &Info{Available: true}
	for _, line := range strings.Split(out, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch key {
		case "Server Version":
			info.Version = val
		case "Containers":
			if n, err := strconv.Atoi(val); err == nil {
				info.Containers = n
			}
		case "Images":
			if n, err := strconv.Atoi(val); err == nil {
				info.Images = n
			}
		case "Operating System":
			info.ServerOS = val
		}
	}
	return info
}
```

- [ ] **Step 5: Add public test**

Create `internal/collector/docker/docker_public_test.go`:

```go
package docker_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/collector/docker"
)

type DockerPublicTestSuite struct {
	suite.Suite
}

func TestDockerPublicTestSuite(t *testing.T) {
	suite.Run(t, new(DockerPublicTestSuite))
}

func (s *DockerPublicTestSuite) TestImplementsCollectorInterface() {
	var _ collector.Collector = docker.New()
}

func (s *DockerPublicTestSuite) TestCollectNeverErrors() {
	c := docker.New()
	_, err := c.Collect(context.Background())
	s.NoError(err) // missing docker should not error, just omit
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/collector/docker/ -v`
Expected: PASS.

- [ ] **Step 7: Verify coverage**

Run: `go test ./internal/collector/docker/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: parser at 100%; `collectUnix` may have unhit branches if docker is/isn't installed (acceptable).

- [ ] **Step 8: Commit**

```bash
git add internal/collector/docker/
git commit -m "feat(docker): add Docker daemon info collector"
```

---

## Task 12: Public Facts struct and JSON output

**Files:**
- Create: `pkg/gohai/facts.go`
- Create: `pkg/gohai/facts_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/gohai/facts_test.go`:

```go
package gohai

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type FactsTestSuite struct {
	suite.Suite
}

func TestFactsTestSuite(t *testing.T) {
	suite.Run(t, new(FactsTestSuite))
}

func (s *FactsTestSuite) TestFlatBuildKeys() {
	tests := []struct {
		name string
		in   map[string]any
		want map[string]any
	}{
		{
			name: "scalars",
			in:   map[string]any{"a": 1, "b": "x"},
			want: map[string]any{"a": 1, "b": "x"},
		},
		{
			name: "nested map",
			in:   map[string]any{"cpu": map[string]any{"total": 8, "model": "intel"}},
			want: map[string]any{"cpu.total": 8, "cpu.model": "intel"},
		},
		{
			name: "deep nest",
			in:   map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			want: map[string]any{"a.b.c": 1},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := flattenMap("", tt.in)
			s.Equal(tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./pkg/gohai/ -v`
Expected: FAIL — package empty.

- [ ] **Step 3: Implement facts.go**

Create `pkg/gohai/facts.go`:

```go
// Package gohai is the public SDK for collecting system facts.
package gohai

import (
	"encoding/json"
	"fmt"
	"time"
)

// Facts holds the result of a collection run. The Data map is keyed by
// collector name and contains each collector's typed result struct.
type Facts struct {
	Data            map[string]any `json:"-"`
	CollectTime     time.Time      `json:"collect_time"`
	CollectDuration time.Duration  `json:"collect_duration_ns"`
}

// JSON returns the compact JSON representation of the facts.
func (f *Facts) JSON() ([]byte, error) {
	return json.Marshal(f.toMap())
}

// PrettyJSON returns the indented JSON representation of the facts.
func (f *Facts) PrettyJSON() ([]byte, error) {
	return json.MarshalIndent(f.toMap(), "", "  ")
}

// Flat returns a flat dot-separated key map of all facts.
func (f *Facts) Flat() map[string]any {
	asMap, ok := normalize(f.toMap()).(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return flattenMap("", asMap)
}

func (f *Facts) toMap() map[string]any {
	out := make(map[string]any, len(f.Data)+2)
	for k, v := range f.Data {
		out[k] = v
	}
	out["collect_time"] = f.CollectTime
	out["collect_duration_ns"] = int64(f.CollectDuration)
	return out
}

func flattenMap(
	prefix string,
	in map[string]any,
) map[string]any {
	out := make(map[string]any)
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			for sk, sv := range flattenMap(key, sub) {
				out[sk] = sv
			}
			continue
		}
		out[key] = v
	}
	return out
}

// normalize round-trips through JSON to convert typed structs into
// generic map[string]any so flattenMap can walk them uniformly.
func normalize(
	v any,
) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}

// Get returns the value at a dot-separated key path, or nil if absent.
func (f *Facts) Get(
	path string,
) any {
	return f.Flat()[path]
}

// String returns a printable summary.
func (f *Facts) String() string {
	return fmt.Sprintf("Facts{%d collectors, took %s}", len(f.Data), f.CollectDuration)
}
```

- [ ] **Step 4: Run, verify pass**

Run: `go test ./pkg/gohai/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gohai/facts.go pkg/gohai/facts_test.go
git commit -m "feat(gohai): add Facts struct with JSON and Flat output"
```

---

## Task 13: Public Facts API tests (JSON, Flat, Get)

**Files:**
- Create: `pkg/gohai/facts_public_test.go`

- [ ] **Step 1: Write public tests**

Create `pkg/gohai/facts_public_test.go`:

```go
package gohai_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type FactsPublicTestSuite struct {
	suite.Suite
	facts *gohai.Facts
}

func TestFactsPublicTestSuite(t *testing.T) {
	suite.Run(t, new(FactsPublicTestSuite))
}

func (s *FactsPublicTestSuite) SetupTest() {
	s.facts = &gohai.Facts{
		Data: map[string]any{
			"platform": map[string]any{"name": "ubuntu", "version": "24.04"},
			"cpu":      map[string]any{"total": 8},
		},
		CollectTime:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
		CollectDuration: 100 * time.Millisecond,
	}
}

func (s *FactsPublicTestSuite) TestJSON() {
	tests := []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{"compact", s.facts.JSON},
		{"pretty", s.facts.PrettyJSON},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			b, err := tt.fn()
			s.Require().NoError(err)
			var got map[string]any
			s.Require().NoError(json.Unmarshal(b, &got))
			s.Contains(got, "platform")
			s.Contains(got, "cpu")
			s.Contains(got, "collect_time")
		})
	}
}

func (s *FactsPublicTestSuite) TestFlat() {
	flat := s.facts.Flat()
	s.Equal("ubuntu", flat["platform.name"])
	s.Equal("24.04", flat["platform.version"])
	s.EqualValues(8, flat["cpu.total"])
}

func (s *FactsPublicTestSuite) TestGet() {
	tests := []struct {
		path string
		want any
	}{
		{"platform.name", "ubuntu"},
		{"cpu.total", float64(8)},
		{"missing", nil},
	}
	for _, tt := range tests {
		s.Run(tt.path, func() {
			s.Equal(tt.want, s.facts.Get(tt.path))
		})
	}
}

func (s *FactsPublicTestSuite) TestString() {
	s.Contains(s.facts.String(), "Facts{")
}
```

- [ ] **Step 2: Run, verify pass**

Run: `go test ./pkg/gohai/ -v`
Expected: PASS.

- [ ] **Step 3: Verify coverage**

Run: `go test ./pkg/gohai/ -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out`
Expected: 100%.

- [ ] **Step 4: Commit**

```bash
git add pkg/gohai/facts_public_test.go
git commit -m "test(gohai): add public Facts API tests"
```

---

## Task 14: Public Gohai SDK — New, Collect, options

**Files:**
- Create: `pkg/gohai/gohai.go`
- Create: `pkg/gohai/options.go`
- Create: `pkg/gohai/gohai_public_test.go`

- [ ] **Step 1: Write the failing public test**

Create `pkg/gohai/gohai_public_test.go`:

```go
package gohai_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/osapi-io/gohai/pkg/gohai"
)

type GohaiPublicTestSuite struct {
	suite.Suite
}

func TestGohaiPublicTestSuite(t *testing.T) {
	suite.Run(t, new(GohaiPublicTestSuite))
}

func (s *GohaiPublicTestSuite) TestNewDefaults() {
	g, err := gohai.New()
	s.Require().NoError(err)
	s.NotNil(g)
}

func (s *GohaiPublicTestSuite) TestCollectAllRegistered() {
	g, err := gohai.New()
	s.Require().NoError(err)
	facts, err := g.Collect(context.Background())
	s.Require().NoError(err)
	s.NotNil(facts)
	// platform is TierCore so should always be present (or nil on unsupported OS)
	s.Contains(facts.Data, "platform")
}

func (s *GohaiPublicTestSuite) TestWithEnabledOnly() {
	g, err := gohai.New(gohai.WithEnabled("docker"))
	s.Require().NoError(err)
	facts, err := g.Collect(context.Background())
	s.Require().NoError(err)
	// docker is opt-in; with WithEnabled it runs (and pulls in virtualization dep)
	s.Contains(facts.Data, "virtualization")
}

func (s *GohaiPublicTestSuite) TestWithDisabled() {
	g, err := gohai.New(gohai.WithDisabled("platform"))
	s.Require().NoError(err)
	facts, err := g.Collect(context.Background())
	s.Require().NoError(err)
	s.NotContains(facts.Data, "platform")
}

func (s *GohaiPublicTestSuite) TestWithCollectorsOnly() {
	g, err := gohai.New(gohai.WithCollectors("platform"))
	s.Require().NoError(err)
	facts, err := g.Collect(context.Background())
	s.Require().NoError(err)
	// only platform should run
	delete(facts.Data, "platform")
	s.Empty(facts.Data)
}

func (s *GohaiPublicTestSuite) TestUnknownCollector() {
	_, err := gohai.New(gohai.WithEnabled("nope"))
	s.Error(err)
}
```

- [ ] **Step 2: Run, verify failure**

Run: `go test ./pkg/gohai/ -v`
Expected: FAIL — `New`, `WithEnabled`, etc. undefined.

- [ ] **Step 3: Implement options.go**

Create `pkg/gohai/options.go`:

```go
package gohai

// Option configures a Gohai instance.
type Option func(*config)

type config struct {
	enabled  []string
	disabled []string
	only     []string // if set, ONLY these collectors run
}

// WithEnabled explicitly enables one or more opt-in collectors.
func WithEnabled(
	names ...string,
) Option {
	return func(c *config) { c.enabled = append(c.enabled, names...) }
}

// WithDisabled explicitly disables one or more collectors.
func WithDisabled(
	names ...string,
) Option {
	return func(c *config) { c.disabled = append(c.disabled, names...) }
}

// WithCollectors restricts collection to ONLY the named collectors and their
// dependencies. Overrides the default tier-based selection.
func WithCollectors(
	names ...string,
) Option {
	return func(c *config) { c.only = append(c.only, names...) }
}
```

- [ ] **Step 4: Implement gohai.go**

Create `pkg/gohai/gohai.go`:

```go
package gohai

import (
	"context"
	"fmt"
	"time"

	"github.com/osapi-io/gohai/internal/collector"
	"github.com/osapi-io/gohai/internal/collector/docker"
	"github.com/osapi-io/gohai/internal/collector/platform"
	"github.com/osapi-io/gohai/internal/collector/virtualization"
)

// Gohai is the SDK entry point.
type Gohai struct {
	registry *collector.Registry
	cfg      config
	selected []collector.Collector
}

// New constructs a Gohai instance with the given options.
func New(
	opts ...Option,
) (*Gohai, error) {
	g := &Gohai{registry: collector.NewRegistry()}
	if err := registerBuiltins(g.registry); err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(&g.cfg)
	}

	var sel []collector.Collector
	var err error
	if len(g.cfg.only) > 0 {
		sel, err = collectorsByName(g.registry, g.cfg.only)
	} else {
		sel, err = g.registry.Selected(g.cfg.enabled, g.cfg.disabled)
	}
	if err != nil {
		return nil, fmt.Errorf("select collectors: %w", err)
	}
	g.selected = sel
	return g, nil
}

// Collect runs all selected collectors and returns Facts.
func (g *Gohai) Collect(
	ctx context.Context,
) (*Facts, error) {
	names := make([]string, 0, len(g.selected))
	for _, c := range g.selected {
		names = append(names, c.Name())
	}
	start := time.Now()
	results, err := g.registry.Run(ctx, names, nil)
	if err != nil {
		return nil, fmt.Errorf("run collectors: %w", err)
	}
	return &Facts{
		Data:            results,
		CollectTime:     start,
		CollectDuration: time.Since(start),
	}, nil
}

func collectorsByName(
	reg *collector.Registry,
	names []string,
) ([]collector.Collector, error) {
	out := make([]collector.Collector, 0, len(names))
	for _, n := range names {
		c, ok := reg.Get(n)
		if !ok {
			return nil, fmt.Errorf("unknown collector %q", n)
		}
		out = append(out, c)
	}
	return out, nil
}

func registerBuiltins(
	reg *collector.Registry,
) error {
	for _, c := range []collector.Collector{
		platform.New(),
		virtualization.New(),
		docker.New(),
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}
```

Note: virtualization depends on dmi+cpu, which aren't implemented in Phase 1. Update virtualization to use `nil` deps for Phase 1, and add a TODO. Edit `internal/collector/virtualization/virtualization.go`:

Change:
```go
func (c *Collector) Dependencies() []string { return []string{"dmi", "cpu"} }
```
to:
```go
// Dependencies — dmi and cpu are not implemented in Phase 1; will be added later.
func (c *Collector) Dependencies() []string { return nil }
```

And update the test in `internal/collector/virtualization/virtualization_test.go`:
```go
s.Empty(c.Dependencies())
```

- [ ] **Step 5: Run all tests**

Run: `go test ./... -v`
Expected: PASS.

- [ ] **Step 6: Verify coverage**

Run: `go test ./... -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | grep -v 100`
Expected: only OS-conditional `collect()` functions show < 100% (acceptable, see spec).

- [ ] **Step 7: Commit**

```bash
git add pkg/gohai/ internal/collector/virtualization/
git commit -m "feat(gohai): add public SDK with New, Collect, and options"
```

---

## Task 15: CLI — Cobra root command and dynamic flags

**Files:**
- Create: `cmd/gohai/main.go`
- Create: `cmd/gohai/flags.go`
- Create: `cmd/gohai/output.go`
- Create: `cmd/gohai/main_test.go`

- [ ] **Step 1: Write CLI test**

Create `cmd/gohai/main_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type CLITestSuite struct {
	suite.Suite
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestRunDefaultJSON() {
	var buf bytes.Buffer
	rc := run([]string{}, &buf)
	s.Equal(0, rc)
	var got map[string]any
	s.Require().NoError(json.Unmarshal(buf.Bytes(), &got))
	s.Contains(got, "collect_time")
}

func (s *CLITestSuite) TestRunPretty() {
	var buf bytes.Buffer
	rc := run([]string{"--pretty"}, &buf)
	s.Equal(0, rc)
	s.Contains(buf.String(), "  ") // indented
}

func (s *CLITestSuite) TestRunFlat() {
	var buf bytes.Buffer
	rc := run([]string{"--flat"}, &buf)
	s.Equal(0, rc)
	s.Contains(buf.String(), "=")
}

func (s *CLITestSuite) TestRunListCollectors() {
	var buf bytes.Buffer
	rc := run([]string{"--list-collectors"}, &buf)
	s.Equal(0, rc)
	s.Contains(buf.String(), "platform")
	s.Contains(buf.String(), "docker")
}

func (s *CLITestSuite) TestRunUnknownFlag() {
	var buf bytes.Buffer
	rc := run([]string{"--bogus"}, &buf)
	s.NotEqual(0, rc)
}

func (s *CLITestSuite) TestRunCollectorEnable() {
	var buf bytes.Buffer
	rc := run([]string{"--collector.docker", "--pretty"}, &buf)
	s.Equal(0, rc)
}

func (s *CLITestSuite) TestRunCollectorDisable() {
	var buf bytes.Buffer
	rc := run([]string{"--no-collector.platform"}, &buf)
	s.Equal(0, rc)
	var got map[string]any
	s.Require().NoError(json.Unmarshal(buf.Bytes(), &got))
	s.NotContains(got, "platform")
}
```

- [ ] **Step 2: Implement main.go**

Create `cmd/gohai/main.go`:

```go
// Command gohai collects system facts and prints them as JSON.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(
	args []string,
	out io.Writer,
) int {
	var (
		pretty         bool
		flat           bool
		listCollectors bool
	)
	enabled, disabled := newCollectorFlagSets()

	cmd := &cobra.Command{
		Use:           "gohai",
		Short:         "Collect system facts",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if listCollectors {
				return printCollectorList(out)
			}

			opts := []gohai.Option{}
			if names := enabled.values(); len(names) > 0 {
				opts = append(opts, gohai.WithEnabled(names...))
			}
			if names := disabled.values(); len(names) > 0 {
				opts = append(opts, gohai.WithDisabled(names...))
			}

			g, err := gohai.New(opts...)
			if err != nil {
				return err
			}
			facts, err := g.Collect(context.Background())
			if err != nil {
				return err
			}
			return writeOutput(out, facts, pretty, flat)
		},
	}

	cmd.Flags().BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	cmd.Flags().BoolVar(&flat, "flat", false, "output flat key=value pairs")
	cmd.Flags().BoolVar(&listCollectors, "list-collectors", false, "list available collectors and exit")
	registerCollectorFlags(cmd, enabled, disabled)

	cmd.SetArgs(args)
	cmd.SetOut(out)
	cmd.SetErr(out)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 3: Implement flags.go**

Create `cmd/gohai/flags.go`:

```go
package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/osapi-io/gohai/pkg/gohai"
)

// collectorFlagSet captures which collectors the user toggled on/off.
type collectorFlagSet struct{ set map[string]bool }

func newCollectorFlagSets() (*collectorFlagSet, *collectorFlagSet) {
	return &collectorFlagSet{set: map[string]bool{}}, &collectorFlagSet{set: map[string]bool{}}
}

func (c *collectorFlagSet) values() []string {
	out := make([]string, 0, len(c.set))
	for n := range c.set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// registerCollectorFlags adds --collector.<name> / --no-collector.<name>
// flags for every registered collector.
func registerCollectorFlags(
	cmd *cobra.Command,
	enabled, disabled *collectorFlagSet,
) {
	for _, name := range listAllCollectorNames() {
		n := name
		cmd.Flags().Bool("collector."+n, false, fmt.Sprintf("enable %s collector", n))
		cmd.Flags().Bool("no-collector."+n, false, fmt.Sprintf("disable %s collector", n))
		_ = cmd.PreRunE
		cobra.OnInitialize(func() {})
	}
	cmd.PreRunE = func(c *cobra.Command, _ []string) error {
		for _, name := range listAllCollectorNames() {
			if v, _ := c.Flags().GetBool("collector." + name); v {
				enabled.set[name] = true
			}
			if v, _ := c.Flags().GetBool("no-collector." + name); v {
				disabled.set[name] = true
			}
		}
		return nil
	}
}

// listAllCollectorNames builds a fresh Gohai (default options) just to
// enumerate registered collector names.
func listAllCollectorNames() []string {
	g, err := gohai.NewRegistry()
	if err != nil {
		return nil
	}
	names := g.Names()
	sort.Strings(names)
	return names
}

func printCollectorList(
	out interface {
		Write([]byte) (int, error)
	},
) error {
	for _, n := range listAllCollectorNames() {
		fmt.Fprintln(out, n)
	}
	return nil
}
```

This requires a public `gohai.NewRegistry()` helper. Add to `pkg/gohai/registry.go`:

```go
package gohai

import (
	"github.com/osapi-io/gohai/internal/collector"
)

// PublicRegistry exposes registered collector names and metadata for CLI use.
type PublicRegistry struct{ reg *collector.Registry }

// NewRegistry returns a PublicRegistry pre-populated with built-in collectors.
func NewRegistry() (*PublicRegistry, error) {
	r := collector.NewRegistry()
	if err := registerBuiltins(r); err != nil {
		return nil, err
	}
	return &PublicRegistry{reg: r}, nil
}

// Names returns all registered collector names.
func (p *PublicRegistry) Names() []string { return p.reg.Names() }
```

- [ ] **Step 4: Implement output.go**

Create `cmd/gohai/output.go`:

```go
package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/osapi-io/gohai/pkg/gohai"
)

func writeOutput(
	out io.Writer,
	facts *gohai.Facts,
	pretty, flat bool,
) error {
	if flat {
		flatMap := facts.Flat()
		keys := make([]string, 0, len(flatMap))
		for k := range flatMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(out, "%s=%v\n", k, flatMap[k])
		}
		return nil
	}

	var (
		b   []byte
		err error
	)
	if pretty {
		b, err = facts.PrettyJSON()
	} else {
		b, err = facts.JSON()
	}
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	_, err = out.Write(append(b, '\n'))
	return err
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./... -v`
Expected: PASS.

- [ ] **Step 6: Run the binary manually**

Run: `go run ./cmd/gohai --pretty`
Expected: pretty-printed JSON with platform/virtualization data.

Run: `go run ./cmd/gohai --list-collectors`
Expected: prints `docker`, `platform`, `virtualization`.

Run: `go run ./cmd/gohai --flat`
Expected: prints flat key=value lines.

- [ ] **Step 7: Verify coverage**

Run: `go test ./... -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | grep -v 100`
Expected: only OS-conditional code shows < 100%.

- [ ] **Step 8: Commit**

```bash
git add cmd/gohai/ pkg/gohai/registry.go
git commit -m "feat(cli): add Cobra CLI with dynamic collector flags"
```

---

## Task 16: Update goreleaser to build CLI

**Files:**
- Modify: `.goreleaser.yaml`

- [ ] **Step 1: Verify goreleaser config**

Read: `.goreleaser.yaml`
Confirm `builds:` references `main: ./cmd/gohai`.

It already does (set up in skeleton). No changes needed if `main` is set.

- [ ] **Step 2: Run goreleaser snapshot to verify build**

Run: `goreleaser build --snapshot --clean --single-target` (skip if goreleaser not installed)
Expected: Builds binary in `dist/`.

If goreleaser not installed, skip and rely on `go build ./cmd/gohai`.

- [ ] **Step 3: Verify go build alone**

Run: `go build -o /tmp/gohai ./cmd/gohai && /tmp/gohai --list-collectors`
Expected: lists collectors.

- [ ] **Step 4: No commit needed if no file changes; otherwise:**

```bash
git add .goreleaser.yaml
git commit -m "build: enable goreleaser CLI builds"
```

---

## Task 17: End-to-end smoke test and documentation update

**Files:**
- Modify: `docs/collectors/platform.md`
- Modify: `docs/collectors/virtualization.md`
- Modify: `docs/collectors/docker.md`

- [ ] **Step 1: Update platform.md with real implementation details**

Replace `docs/collectors/platform.md` with:

```markdown
# Platform

> **Status:** Implemented (Phase 1)

## Description

Collects operating system platform identification facts: name, version,
family, and architecture.

## Collected Fields

| Field          | Type   | Description                                       |
| -------------- | ------ | ------------------------------------------------- |
| `name`         | string | OS identifier (e.g., `ubuntu`, `rhel`, `mac_os_x`) |
| `version`      | string | OS version (e.g., `24.04`, `14.4.1`)              |
| `family`       | string | OS family (e.g., `debian`, `rhel`, `mac_os_x`)    |
| `architecture` | string | CPU architecture (e.g., `amd64`, `arm64`)         |
| `build`        | string | Build version (macOS only)                        |

## Platform Support

| Platform | Source                                   | Supported |
| -------- | ---------------------------------------- | --------- |
| Linux    | `/etc/os-release`                        | ✅        |
| macOS    | `sw_vers`                                | ✅        |

## Example Output

\`\`\`json
{
  "platform": {
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "architecture": "amd64"
  }
}
\`\`\`

## Enable/Disable

\`\`\`bash
gohai --collector.platform      # enable (default)
gohai --no-collector.platform   # disable
\`\`\`
```

- [ ] **Step 2: Update virtualization.md and docker.md similarly**

For `docs/collectors/virtualization.md` — note Phase 1 is Linux cgroup detection only.

For `docs/collectors/docker.md` — note opt-in, requires `docker` in PATH.

(Use the same template as platform.md, adapted to the collector's fields.)

- [ ] **Step 3: Run final test pass**

Run: `go test ./... -count=1 -coverprofile=/tmp/cov.out && go tool cover -func=/tmp/cov.out | tail -1`
Expected: total coverage close to 100% (OS-conditional code may pull it slightly lower; aim for ≥ 95%).

- [ ] **Step 4: Run linter**

Run: `go vet ./...`
Expected: no issues.

If golangci-lint installed: `golangci-lint run ./...`
Expected: no issues.

- [ ] **Step 5: Commit**

```bash
git add docs/collectors/platform.md docs/collectors/virtualization.md docs/collectors/docker.md
git commit -m "docs: document Phase 1 collectors"
```

- [ ] **Step 6: Final smoke test**

Run: `go run ./cmd/gohai --pretty`
Expected: JSON output with platform fact populated.

Run: `go run ./cmd/gohai --collector.docker --pretty`
Expected: JSON includes virtualization (auto-included via dependency? No — virtualization is TierExtended so already on by default; docker is opt-in and now enabled).

---

## Done — Phase 1 Complete

After all tasks: foundation is built and validated end-to-end with 3 representative collectors. Each subsequent collector follows the same pattern (Tasks 6–9 for platform are the template). Phase 2+ plans should be written one per category (system, hardware, network, cloud, virtualization, security, software, users, linux-specific) using this plan as the reference.
