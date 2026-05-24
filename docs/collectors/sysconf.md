# Sysconf

> **Status:** Implemented ✅

## Description

Reports four POSIX `sysconf(3)` values: clock tick rate, page size, and
configured and online CPU counts. Matches Ohai's `sysconf` plugin but uses
`github.com/tklauser/go-sysconf` (already in the module graph as a transitive
dependency) instead of shelling out to `getconf -a` — faster, no child process,
identical semantics. Both Linux and macOS expose the same four `SC_*` constants.

DefaultEnabled is `false` — niche fact most consumers don't need.

## Signals

The collector reports four related signals:

- `clk_tck` — clock ticks per second (SC_CLK_TCK). Used to convert `/proc/stat`
  jiffy counts to wall-clock seconds when computing per-process CPU time.
  Typically 100 on Linux, 128 on macOS.
- `pagesize` — memory page size in bytes (SC_PAGESIZE). Needed when converting
  page-count fields from `/proc/meminfo` or `vm_stat` into bytes.
- `nprocessors_conf` — CPUs configured in the kernel (SC_NPROCESSORS_CONF).
  Includes offline CPUs; higher than `nprocessors_onln` on hosts with CPU
  hotplug.
- `nprocessors_onln` — CPUs currently online (SC_NPROCESSORS_ONLN). The value
  used for parallelism decisions.

## Collected Fields

| Field              | Type  | Description                                                         | Schema mapping                                  |
| ------------------ | ----- | ------------------------------------------------------------------- | ----------------------------------------------- |
| `clk_tck`          | int64 | Clock ticks per second (SC_CLK_TCK).                                | `host.cpu.count` family — OTel; no exact match. |
| `pagesize`         | int64 | Memory page size in bytes (SC_PAGESIZE).                            | `system.memory.page.size` — OTel semconv.       |
| `nprocessors_conf` | int64 | Total configured CPU count including offline (SC_NPROCESSORS_CONF). | `host.cpu.count` (OTel) — total configured.     |
| `nprocessors_onln` | int64 | Currently online CPU count (SC_NPROCESSORS_ONLN).                   | `host.cpu.count` (OTel) — online subset.        |

## Platform Support

| Platform | Supported                              |
| -------- | -------------------------------------- |
| Linux    | ✅ (via `go-sysconf` SC\_\* constants) |
| macOS    | ✅ (same SC\_\* constants supported)   |

## Example Output

### Linux (x86-64, 4-core)

```json
{
  "sysconf": {
    "clk_tck": 100,
    "pagesize": 4096,
    "nprocessors_conf": 4,
    "nprocessors_onln": 4
  }
}
```

### macOS (Apple Silicon, 10-core)

```json
{
  "sysconf": {
    "clk_tck": 100,
    "pagesize": 16384,
    "nprocessors_conf": 10,
    "nprocessors_onln": 10
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("sysconf"))
facts, _ := g.Collect(context.Background())
if s := facts.Sysconf; s != nil {
    fmt.Printf("page size: %d, online CPUs: %d\n", s.Pagesize, s.NprocessorsOnln)
}
```

## Enable/Disable

```bash
gohai --collector.sysconf    # enable (opt-in)
gohai --no-collector.sysconf # disable
```

## Dependencies

None.

## Data Sources

Ohai's `sysconf.rb` shells out to `getconf -a` and parses the key=value output.
gohai uses `github.com/tklauser/go-sysconf` instead — a pure Go wrapper around
the POSIX `sysconf(3)` syscall. This avoids the subprocess overhead and gives
type-safe `int64` values directly. The four constants collected (CLK_TCK,
PAGESIZE, NPROCESSORS_CONF, NPROCESSORS_ONLN) match Ohai's output.

On both Linux and macOS the collector calls `Sysconf(SC_*)` from
`github.com/tklauser/go-sysconf` directly — no subprocess:

1. `SC_CLK_TCK` → `clk_tck`
2. `SC_PAGESIZE` → `pagesize`
3. `SC_NPROCESSORS_CONF` → `nprocessors_conf`
4. `SC_NPROCESSORS_ONLN` → `nprocessors_onln`

Any `Sysconf` error is propagated as a wrapped error.

**Why `go-sysconf` instead of `getconf -a`?** Ohai's `sysconf.rb` shells out to
`getconf -a` and parses its output. We use the Go library because it is already
present in the module graph (brought in transitively by gopsutil), avoids
spawning a subprocess, and provides type-safe constants — no string parsing
needed. The library wraps the same POSIX `sysconf(3)` syscall under the hood, so
semantics are identical.

## Backing library

- [`github.com/tklauser/go-sysconf`](https://github.com/tklauser/go-sysconf) —
  pure-Go POSIX `sysconf(3)` wrapper. Already in the module graph as a
  transitive dep of gopsutil.
