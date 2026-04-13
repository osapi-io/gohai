# Load

> **Status:** Implemented ✅

## Description

Reports the 1/5/15-minute system load averages — the same numbers `uptime(1)`,
`w(1)`, and `top(1)` show. Load average is the exponentially-weighted moving
average of the run queue length (runnable + uninterruptible tasks) over each
window.

Rule of thumb: a 1-minute load of N on a host with M cores means N tasks were
ready to run against M execution units. Sustained `one > cores` indicates CPU
saturation; sustained `fifteen > cores` indicates a systemic backlog rather than
a short burst.

Consumers use this to:

- Feed heartbeat / health-probe payloads with a cheap saturation signal (one
  `getloadavg(3)` call, no heavy process scan).
- Correlate CPU pressure with incident start times (the 15-min window survives
  short spikes).
- Trigger load shedding or autoscaling when `one` crosses a per-fleet threshold.

Consumers that need per-core saturation should divide by `cpu.total` from the
`cpu` collector.

## Collected Fields

| Field     | Type    | Description             | Schema mapping  |
| --------- | ------- | ----------------------- | --------------- |
| `one`     | float64 | 1-minute load average.  | No direct OCSF. |
| `five`    | float64 | 5-minute load average.  | No direct OCSF. |
| `fifteen` | float64 | 15-minute load average. | No direct OCSF. |

Field names are a gohai choice: OCSF has no load-average field and Ohai ships no
loadavg plugin, so `one`/`five`/`fifteen` are picked to match the conceptual
triple returned by `getloadavg(3)`.

## Platform Support

| Platform | Supported                             |
| -------- | ------------------------------------- |
| Linux    | ✅ (`/proc/loadavg` via gopsutil)     |
| macOS    | ✅ (`sysctl vm.loadavg` via gopsutil) |

## Example Output

```json
{
  "load": {
    "one": 0.25,
    "five": 0.48,
    "fifteen": 0.62
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("load", "cpu"))
facts, _ := g.Collect(context.Background())

l := facts.Load
perCore := l.One / float64(facts.CPU.Total)
if perCore > 1.0 {
    log.Printf("CPU saturated: %.2f per core", perCore)
}
```

## Enable/Disable

```bash
gohai --collector.load      # enable (default)
gohai --no-collector.load   # disable
```

## Dependencies

None. (Consumers that want per-core saturation correlate with `cpu.total`
themselves — we don't declare a runtime dependency because `Dependencies()`
would auto-include `cpu` even when the user disabled it. See
[dependencies.md](../features/dependencies.md#why-built-ins-dont-declare-dependencies-today).)

## Data Sources

| Platform | What we read                                                | Ohai plugin                                                                    | Alignment                                                                                                                                                                 |
| -------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `load.AvgWithContext` (reads `/proc/loadavg`).     | **No Ohai equivalent.** Ohai does not ship a loadavg plugin in current `main`. | **gohai extension.** The `one`/`five`/`fifteen` field names are a Go-idiomatic choice aligned with `getloadavg(3)`'s conceptual triple; they are not inherited from Ohai. |
| macOS    | gopsutil `load.AvgWithContext` (reads `sysctl vm.loadavg`). | **No Ohai equivalent.**                                                        | Same as Linux — gohai extension.                                                                                                                                          |

**Known gaps vs. Ohai:** N/A — Ohai has no loadavg plugin, so there is no
coverage to mirror. Trailing fields of `/proc/loadavg` (`runnable_tasks`,
`last_pid`) aren't exposed by `getloadavg(3)`; deferred until a concrete
consumer asks.

## Backing library

- [`github.com/shirou/gopsutil/v4/load`](https://github.com/shirou/gopsutil) —
  BSD-3.
