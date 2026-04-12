# Concurrent Collection

> **Status:** Implemented ✅

## Overview

When `gohai.Collect(ctx)` runs, every selected collector runs in its own
goroutine. The registry groups collectors by dependency level and dispatches
each level concurrently; levels run serially so dependent collectors see their
inputs finished before they start.

**In practice today**, no built-in collector declares a dependency (every
`Dependencies()` returns `nil`), so the whole selected set lands in a single
level and runs fully in parallel.

## How it works

### The algorithm

`internal/collector.Registry.Run(ctx, names, onError)` does:

1. **Expand** — walks each name's `Dependencies()` recursively and auto-includes
   anything missing. If `packages` depended on `platform` (it doesn't),
   requesting `packages` alone would also pull in `platform`.
2. **Topological levels** — partitions the expanded set into levels where every
   node in level _N_ has all its dependencies resolved in levels _0 … N-1_.
   Cycles fail with `"dependency cycle detected among collectors"`.
3. **Per-level fan-out** — each level runs one `sync.WaitGroup` and spawns a
   goroutine per collector. Results land in a mutex-protected map. When the
   level finishes, the next starts.
4. **Error handling** — a collector returning an error skips its result and
   calls the optional `onError` callback; siblings keep running. One failure
   doesn't cancel the others.

### Context cancellation

Every collector receives the same `context.Context` you pass to `Collect`. The
CLI binds it to SIGINT/SIGTERM (see `cmd/root.go`) so `Ctrl-C` propagates to
every in-flight collector. SDK consumers pass their own context and cancel the
same way.

## What the user controls

- **Which collectors run** — `gohai.WithEnabled(...)`,
  `gohai.WithDisabled(...)`, `gohai.WithCollectors(...)`. See
  [Pluggable Collectors](collectors.md).
- **Parallelism** — not currently a tunable. Every selected collector starts
  concurrently. No per-collector timeout or rate limit. If one slow collector
  blocks the overall `Collect` call, cancel the context.
- **Dependencies** — not user-declared. Dependencies are per-collector metadata
  on the `Collector` interface; users never author them.

## When dependency ordering matters

The infrastructure exists for future collectors that genuinely consume another
collector's output. Example: if a `package_inventory` collector needed
`platform` to finish first so it could pick `apt` vs. `dnf`, it would declare:

```go
func (base) Dependencies() []string { return []string{"platform"} }
```

The registry would then run `platform` to completion before starting
`package_inventory`. Today, `package_mgr` does its own dispatch via
`internal/platform.Detect()` at construction time, so no declared dependency is
needed — which is why every shipped collector's `Dependencies()` returns `nil`.

## Where this lives in the code

- `internal/collector/registry.go` — `Registry.Run`, `expandWithDeps`,
  `topoLevels`.
- `internal/collector/collector.go` — the `Collector` interface, including
  `Dependencies() []string`.
- `pkg/gohai/gohai.go` — the `Gohai` facade that constructs a Registry, selects
  collectors, and calls `Run`.

## Related features

- [Pluggable Collectors](collectors.md) — enable/disable selection.
- [Collector Dependencies](dependencies.md) — how the graph is wired.
- [SDK Integration](sdk.md) — calling `Collect(ctx)` from consumer code.
