# Process

> **Status:** Implemented ✅

## Description

Enumerates running processes with PID, parent PID, name, owner, command line,
state, and start time. A point-in-time snapshot of `/proc/<pid>/*` (Linux) or
`kinfo_proc` (macOS).

Per-field read errors (access denied for a process owned by another user,
zombies, short-lived processes that vanish mid-scan) leave the affected field
zero-valued but keep the entry — the snapshot is best-effort, not strictly
uniform.

Consumers use this to:

- Detect unexpected processes (security — is `cryptominer` running?).
- Map process → owner for accountability.
- Reconstruct the parent/child tree via `pid`/`ppid`.
- Enumerate things listening on services by name.

## Collected Fields

| Field                    | Type   | Description                                  | OCSF mapping                  |
| ------------------------ | ------ | -------------------------------------------- | ----------------------------- |
| `count`                  | int    | Total process count.                         | No direct OCSF.               |
| `processes[].pid`        | int32  | Process ID.                                  | `process.pid`.                |
| `processes[].ppid`       | int32  | Parent process ID.                           | `process.parent_process.pid`. |
| `processes[].name`       | string | Executable name (no path / args).            | `process.name`.               |
| `processes[].username`   | string | Owner username.                              | `process.user.name`.          |
| `processes[].cmd_line`   | string | Full command line (space-joined argv).       | `process.cmd_line`.           |
| `processes[].state`      | string | POSIX state letter: `R`/`S`/`D`/`Z`/`T`/`I`. | No direct OCSF.               |
| `processes[].start_time` | uint64 | Process start time (unix seconds).           | `process.created_time`.       |

Field name `cmd_line` follows OCSF (`process.cmd_line`) rather than Ohai's
`command` — OCSF precedes Ohai when both name a field.

## Platform Support

| Platform | Supported                                             |
| -------- | ----------------------------------------------------- |
| Linux    | ✅ (`/proc/<pid>/{stat,status,cmdline}` via gopsutil) |
| macOS    | ✅ (`kinfo_proc` via gopsutil)                        |

## Example Output

```json
{
  "process": {
    "count": 3,
    "processes": [
      {
        "pid": 1,
        "ppid": 0,
        "name": "systemd",
        "username": "root",
        "cmd_line": "/sbin/init",
        "state": "S",
        "start_time": 1712064000
      },
      {
        "pid": 1234,
        "ppid": 1,
        "name": "sshd",
        "username": "root",
        "cmd_line": "/usr/sbin/sshd -D",
        "state": "S",
        "start_time": 1712064010
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("process"))
facts, _ := g.Collect(context.Background())

for _, p := range facts.Process.Processes {
    if p.Name == "nginx" {
        fmt.Printf("nginx pid=%d owner=%s\n", p.PID, p.Username)
    }
}
```

## Enable/Disable

```bash
gohai --collector.process      # enable (default)
gohai --no-collector.process   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                                                    | Ohai plugin                                                                                                    | Alignment                                                                                                                                                                                                    |
| -------- | ------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Linux    | gopsutil `process.Processes` (reads `/proc/<pid>/stat`, `/proc/<pid>/cmdline`). | [`linux/ps.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/ps.rb) — shells out to `ps -ef`. | **Equivalent data, different path.** We read `/proc` directly (no subprocess). Ohai's `ps` captures more columns (%cpu, %mem, tty, stime); we extend via process.CPU/MemoryPercent in a follow-up if needed. |
| macOS    | gopsutil `process.Processes` (kinfo_proc syscall).                              | [`darwin/ps.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/darwin/ps.rb) — `ps -ef`.             | **Same source data.**                                                                                                                                                                                        |

**Known gaps vs. Ohai:** `%cpu`, `%mem`, `tty`, `stime` columns from `ps`. Also
no threads, no file-descriptor count, no per-process env. Planned as extensions
if consumers need them.

## Backing library

- [`github.com/shirou/gopsutil/v4/process`](https://github.com/shirou/gopsutil)
  — BSD-3.
