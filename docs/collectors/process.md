# Process

> **Status:** Implemented ✅

## Description

The collector lists the running processes with their PID, parent PID, name,
owner, command line, state and start time. The data is a snapshot of
`/proc/<pid>/*` (Linux) or `kinfo_proc` (macOS) at one point in time.

A read error on one field does not remove the entry. The field gets its zero
value. These errors occur, for example, if access is denied to a process of a
different user, for zombies, and for short processes that stop during the scan.
Thus the snapshot gives as much data as possible, but it is not fully uniform.

Consumers use this data to:

- Find unexpected processes, for security. For example, is `cryptominer`
  running?
- Map each process to its owner, for accountability.
- Make the parent and child tree again from `pid` and `parent_pid`.
- List the processes that listen on services, by name.

## Collected fields

| Field                       | Type    | Description                                                                             | Schema mapping                |
| --------------------------- | ------- | --------------------------------------------------------------------------------------- | ----------------------------- |
| `count`                     | int     | The total number of processes.                                                          | No direct schema mapping.     |
| `processes[].pid`           | int32   | The process ID.                                                                         | `process.pid`.                |
| `processes[].parent_pid`    | int32   | The parent process ID.                                                                  | `process.parent_process.pid`. |
| `processes[].name`          | string  | The executable name, without the path or arguments.                                     | `process.name`.               |
| `processes[].owner`         | string  | The username of the owner.                                                              | `process.user.name`.          |
| `processes[].cmd_line`      | string  | The full command line: argv, with spaces between the items.                             | `process.cmd_line`.           |
| `processes[].state`         | string  | The state as a word: `running`, `sleep`, `zombie`, `stop`, …                            | No direct schema mapping.     |
| `processes[].creation_time` | int64   | The process start time, in Unix seconds.                                                | `process.created_time`.       |
| `processes[].cpu_percent`   | float64 | CPU time used since start, as a share of the time since start. 100 is one full CPU.     | No direct schema mapping.     |
| `processes[].mem_percent`   | float64 | RSS as a share of physical memory.                                                      | No direct schema mapping.     |
| `processes[].rss`           | int64   | The resident set size, in bytes.                                                        | OTel `process.memory.usage`.  |
| `processes[].cpu_seconds`   | float64 | User and system CPU time used, in seconds. Two samples give the CPU share between them. | No direct schema mapping.     |

The field name `cmd_line` comes from OCSF (`process.cmd_line`), not from Ohai's
`command`. If OCSF and Ohai both name a field, the OCSF name is used.

## Platform support

| Platform | Supported                                |
| -------- | ---------------------------------------- |
| Linux    | ✅ (`/proc/<pid>/{stat,status,cmdline}`) |
| macOS    | ✅ (`ps -axo`)                           |

## Example output

```json
{
  "process": {
    "count": 3,
    "processes": [
      {
        "pid": 1,
        "parent_pid": 0,
        "name": "systemd",
        "owner": "root",
        "cmd_line": "/sbin/init",
        "state": "S",
        "creation_time": 1712064000
      },
      {
        "pid": 1234,
        "parent_pid": 1,
        "name": "sshd",
        "owner": "root",
        "cmd_line": "/usr/sbin/sshd -D",
        "state": "S",
        "creation_time": 1712064010
      }
    ]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("process"))
facts, _ := g.Collect(context.Background())

for _, p := range facts.Process.Processes {
    if p.Name == "nginx" {
        fmt.Printf("nginx pid=%d owner=%s\n", p.PID, p.Owner)
    }
}
```

## Enable/disable

```bash
gohai collect --collector.process      # opt-in (off by default — process enumeration scales with process count)
gohai collect --no-collector.process   # explicitly disable (e.g. when stripping defaults)
```

## Dependencies

None.

## Data sources

| Platform | What we read                                                                                   | Ohai plugin                                                                                                                                                          | Alignment                                                                                                                                                                                                                                                                                                                      |
| -------- | ---------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Linux    | `/proc/<pid>/stat`, `/proc/<pid>/status`, `/proc/<pid>/cmdline` and `/proc/meminfo`.           | [`ps.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/ps.rb) sets `command[:ps] = "ps -ef"`. The `command` plugin runs it and reports the raw `ps` text. | **Different shape, same source data.** Ohai reports the raw `ps -ef` output as one text block through its companion `command` collector (refer to `docs/collectors/command.md`). gohai parses `/proc` into typed records, one for each process. Ohai's `ps -ef` columns include `tty` and `stime`. gohai does not report them. |
| macOS    | `ps -ax -o pid=,ppid=,user=,state=,pcpu=,pmem=,rss=,time=,lstart=`, then `ucomm=` and `args=`. | The same top-level [`ps.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/ps.rb), with `ps -ef` in the `:darwin` branch.                                  | **Same source data, different shape.** This is the same as the Linux row.                                                                                                                                                                                                                                                      |

On Linux, `cpu_percent` is `utime` plus `stime` of `/proc/<pid>/stat`, in
seconds, divided by the seconds since the process started, times 100. `rss` is
`VmRSS` of `/proc/<pid>/status`, and `mem_percent` is `rss` divided by
`MemTotal` of `/proc/meminfo`. On macOS, `ps` gives the three: its `%cpu` is a
decaying average, not the average since start. `cpu_seconds` is `utime` plus
`stime` on Linux, and the `time` column of `ps` on macOS. The difference between
two samples, over the seconds between them, is the CPU share in that time.

`Read(ctx, pid)` reads one process without the others: its `/proc/<pid>`
directory on Linux, and `ps -p <pid>` on macOS. It is an error when there is no
such process. A caller that watches one process, such as its own, uses it on
each sample, not `Collect`.

**Known gaps vs. Ohai:** gohai does not report the `tty` and `stime` columns of
`ps`. It also does not report threads, the file descriptor count or the
environment of each process. These are planned extensions, if consumers need
them. For parity with Ohai, the gohai `command` collector reports the raw
`ps -ef` text block separately.

## Backing library

None. The collector reads `/proc` and `/etc/passwd` through its filesystem, and
runs `ps` through its command runner.
