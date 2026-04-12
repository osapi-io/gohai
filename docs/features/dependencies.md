# Collector Dependencies

> **Status:** Implemented ✅ (infrastructure); unused by built-ins today.

## Overview

Every collector implements `Dependencies() []string` on the
`collector.Collector` interface. The registry uses the return value to
auto-include transitive dependencies and to order concurrent collection levels.

Today every built-in collector returns `nil`. The feature is load-bearing
infrastructure for future collectors that genuinely consume another collector's
typed output.

## The contract

```go
type Collector interface {
    Name() string
    DefaultEnabled() bool
    Dependencies() []string
    Collect(ctx context.Context) (any, error)
}
```

`Dependencies()` returns the list of other collector **names** (e.g.
`"platform"`, `"machine_id"`) that must run before this collector's `Collect` is
called.

## What the registry does with them

When `gohai.Collect(ctx)` runs the selected set:

1. **Auto-include transitive deps** — requesting one collector pulls in
   everything it depends on, even if the caller didn't ask for it.
2. **Topological ordering** — the dependency graph is partitioned into
   concurrency levels. A collector with no deps is in level 0; one depending
   only on level-0 nodes lands in level 1; etc. Levels run serially; collectors
   within a level run concurrently.
3. **Cycle detection** — mutual dependencies fail the run with
   `"dependency cycle detected among collectors"`.
4. **Missing-dependency errors** — naming a dependency that isn't registered
   fails the run early with `"unknown collector <name>"`.

See [Concurrent Collection](concurrency.md) for the full algorithm.

## Why built-ins don't declare dependencies today

Collectors that conceptually depend on host-identity data (`shard` on
machine_id + hostname; `package_mgr` on platform family) resolve those inputs
directly via `internal/platform.Detect()` or their own file reads, rather than
declaring a runtime data dependency on another collector's output.

This avoids a common gotcha: if `package_mgr` declared `platform` as a
dependency, then running `gohai --no-collector.platform` would silently
re-include `platform` via the dependency chain, defeating the user's disable.

## When to declare a dependency (guideline)

Add `"othername"` to `Dependencies()` only when your collector literally needs
the other collector's typed output (an `*otherpkg.Info`) during its own
`Collect` call. Since the current `Collect(ctx) (any, error)` signature doesn't
pass prior results, the collector that genuinely needs another's output has to
either:

- re-detect the same data itself (what `shard` and `package_mgr` do today), or
- wait for a future Collector interface revision that passes prior Facts into
  `Collect`.

Until the latter lands, declaring a dependency is **only** useful as a
scheduling hint (run me after X) — which no built-in currently needs.

## Where this lives in the code

- `internal/collector/collector.go` — the `Collector` interface.
- `internal/collector/registry.go` — `expandWithDeps`, `topoLevels`, cycle
  detection.

## Related features

- [Concurrent Collection](concurrency.md) — how dependency levels map to
  goroutine scheduling.
- [Pluggable Collectors](collectors.md) — the enable/disable controls that feed
  into dependency resolution.
