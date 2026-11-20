# Sysconf

> **Status:** Implemented ✅

## Description

The collector reports four POSIX `sysconf(3)` values: the clock tick rate, the
page size, the configured CPU count and the online CPU count. The data is the
same as Ohai's `sysconf` plugin, and gohai gets it the same way: with `getconf`.
Linux and macOS both have the four names.

DefaultEnabled is `false`. Most consumers do not need this data.

## Signals

The collector reports four related signals:

- `clk_tck`. The clock ticks per second (SC_CLK_TCK). Use it to change
  `/proc/stat` jiffy counts to wall-clock seconds when you calculate the CPU
  time of a process. It is usually 100 on Linux and on macOS.
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

| Platform | Supported      |
| -------- | -------------- |
| Linux    | ✅ (`getconf`) |
| macOS    | ✅ (`getconf`) |

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

Ohai's `sysconf.rb` runs `getconf -a` and parses its output. gohai runs
`getconf` for each of the four names, through the command runner of its sources.
One name for each run gives the same command on Linux and macOS, and a fake host
answers each one:

1. `getconf CLK_TCK` → `clk_tck`
2. `getconf PAGESIZE` → `pagesize`
3. `getconf _NPROCESSORS_CONF` → `nprocessors_conf`
4. `getconf _NPROCESSORS_ONLN` → `nprocessors_onln`

If a run fails, or its output is not a number, the collector returns the error.

## Backing library

None. The collector runs `getconf` through its command runner.
