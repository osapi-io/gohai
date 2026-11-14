# Sysconf

> **Status:** Implemented ✅

## Description

The collector reports four POSIX `sysconf(3)` values: the clock tick rate, the
page size, the configured CPU count and the online CPU count. The data is the
same as Ohai's `sysconf` plugin. But gohai uses `github.com/tklauser/go-sysconf`
and does not run `getconf -a`. The library is already in the module graph as a
transitive dependency. It is faster, it starts no child process, and the values
have the same meaning. Linux and macOS both have the same four `SC_*` constants.

DefaultEnabled is `false`. Most consumers do not need this data.

## Signals

The collector reports four related signals:

- `clk_tck`. The clock ticks per second (SC_CLK_TCK). Use it to change
  `/proc/stat` jiffy counts to wall-clock seconds when you calculate the CPU
  time of a process. It is usually 100 on Linux and 128 on macOS.
- `pagesize`. The memory page size in bytes (SC_PAGESIZE). You need it to change
  page-count fields from `/proc/meminfo` or `vm_stat` into bytes.
- `nprocessors_conf`. The CPUs configured in the kernel (SC_NPROCESSORS_CONF).
  It includes offline CPUs. On hosts with CPU hotplug, it is higher than
  `nprocessors_onln`.
- `nprocessors_onln`. The CPUs that are online now (SC_NPROCESSORS_ONLN). Use
  this value to decide how much work to do in parallel.

## Collected fields

| Field              | Type  | Description                                                              | Schema mapping                                 |
| ------------------ | ----- | ------------------------------------------------------------------------ | ---------------------------------------------- |
| `clk_tck`          | int64 | The clock ticks per second (SC_CLK_TCK).                                 | `host.cpu.count` family, OTel. No exact match. |
| `pagesize`         | int64 | The memory page size in bytes (SC_PAGESIZE).                             | `system.memory.page.size`, OTel semconv.       |
| `nprocessors_conf` | int64 | The total configured CPU count, with offline CPUs (SC_NPROCESSORS_CONF). | `host.cpu.count` (OTel), total configured.     |
| `nprocessors_onln` | int64 | The count of CPUs that are online now (SC_NPROCESSORS_ONLN).             | `host.cpu.count` (OTel), online subset.        |

## Platform support

| Platform | Supported                                      |
| -------- | ---------------------------------------------- |
| Linux    | ✅ (through the `go-sysconf` SC\_\* constants) |
| macOS    | ✅ (the same SC\_\* constants)                 |

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("sysconf"))
facts, _ := g.Collect(context.Background())
if s := facts.Sysconf; s != nil {
    fmt.Printf("page size: %d, online CPUs: %d\n", s.Pagesize, s.NprocessorsOnln)
}
```

## Enable/disable

```bash
gohai collect --collector.sysconf    # enable (opt-in)
gohai collect --no-collector.sysconf # disable
```

## Dependencies

None.

## Data sources

Ohai's `sysconf.rb` runs `getconf -a` in a shell and parses the key=value
output. gohai uses `github.com/tklauser/go-sysconf`, a pure Go wrapper around
the POSIX `sysconf(3)` syscall. Thus gohai starts no subprocess, and it gets
type-safe `int64` values directly. The four constants that gohai collects
(CLK_TCK, PAGESIZE, NPROCESSORS_CONF, NPROCESSORS_ONLN) are the same as Ohai's
output.

On Linux and macOS, the collector calls `Sysconf(SC_*)` from
`github.com/tklauser/go-sysconf` directly. It starts no subprocess:

1. `SC_CLK_TCK` → `clk_tck`
2. `SC_PAGESIZE` → `pagesize`
3. `SC_NPROCESSORS_CONF` → `nprocessors_conf`
4. `SC_NPROCESSORS_ONLN` → `nprocessors_onln`

If `Sysconf` returns an error, the collector wraps it and returns it.

**Why `go-sysconf` and not `getconf -a`?** Ohai's `sysconf.rb` runs `getconf -a`
in a shell and parses its output. gohai uses the Go library for three reasons:

- The library is already in the module graph. gopsutil brings it in as a
  transitive dependency.
- It starts no subprocess.
- It gives type-safe constants, so no string parse is necessary.

The library wraps the same POSIX `sysconf(3)` syscall, so the values have the
same meaning.

## Backing library

- [`github.com/tklauser/go-sysconf`](https://github.com/tklauser/go-sysconf). A
  pure-Go wrapper of POSIX `sysconf(3)`. It is already in the module graph as a
  transitive dependency of gopsutil.
