# Load

> **Status:** Implemented ✅

## Description

The collector reports the system load averages for 1, 5 and 15 minutes. These
are the same numbers that `uptime(1)`, `w(1)` and `top(1)` show. The load
average is the exponentially weighted moving average of the run queue length in
each time window. The run queue holds the tasks that can run and the tasks that
cannot be interrupted.

A general rule: a 1-minute load of N on a host with M cores means that N tasks
were ready to run on M execution units. If `one > cores` for a long time, the
CPU is saturated. If `fifteen > cores` for a long time, there is a backlog in
the system, not a short burst.

Consumers use this data to:

- Add a low-cost saturation signal to heartbeat and health-probe payloads. The
  collector makes one `getloadavg(3)` call and does not scan the processes.
- Compare CPU pressure with the start times of incidents. The 15-minute window
  keeps its value through short spikes.
- Start load shedding or autoscaling when `one` goes above a threshold for the
  fleet.

If a consumer needs the saturation of each core, it divides by `cpu.total` from
the `cpu` collector.

## Collected fields

| Field     | Type    | Description                 | Schema mapping            |
| --------- | ------- | --------------------------- | ------------------------- |
| `one`     | float64 | The 1-minute load average.  | No direct schema mapping. |
| `five`    | float64 | The 5-minute load average.  | No direct schema mapping. |
| `fifteen` | float64 | The 15-minute load average. | No direct schema mapping. |

gohai selected the field names. OCSF has no load-average field, and Ohai has no
loadavg plugin. The names `one`, `five` and `fifteen` agree with the three
values that `getloadavg(3)` returns.

## Platform support

| Platform | Supported                                 |
| -------- | ----------------------------------------- |
| Linux    | ✅ (`/proc/loadavg` through gopsutil)     |
| macOS    | ✅ (`sysctl vm.loadavg` through gopsutil) |

## Example output

```json
{
  "load": {
    "one": 0.25,
    "five": 0.48,
    "fifteen": 0.62
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("load", "cpu"))
facts, _ := g.Collect(context.Background())

l := facts.Load
perCore := l.One / float64(facts.CPU.Total)
if perCore > 1.0 {
    log.Printf("CPU saturated: %.2f per core", perCore)
}
```

## Enable/disable

```bash
gohai collect --collector.load      # enable (default)
gohai collect --no-collector.load   # disable
```

## Dependencies

None. A consumer that needs the saturation of each core compares the load with
`cpu.total` itself. The collector does not declare a runtime dependency. A
declared dependency makes `Dependencies()` add `cpu` automatically, also when
the user disabled it.

## Data sources

On Linux:

1. gopsutil's `load.AvgWithContext` reads `/proc/loadavg` and returns the three
   usual floats. The collector reports them as `one`, `five` and `fifteen`.

On macOS:

1. gopsutil's `load.AvgWithContext` calls `sysctl vm.loadavg` and returns the
   same three floats. The field mapping is the same as on Linux.

Ohai has no loadavg plugin in its current `main`. Thus this collector has no
equivalent in Ohai. The field names are the Go form of the three values of
`getloadavg(3)`. The collector does not report the last fields of
`/proc/loadavg` (`runnable_tasks`, `last_pid`), because `getloadavg(3)` does not
give them. gohai will add them when a consumer needs them.

## Backing library

- [`github.com/shirou/gopsutil/v4/load`](https://github.com/shirou/gopsutil),
  BSD-3.
