# Sessions

> **Status:** Implemented ✅

## Description

The collector reports the user sessions that are logged in now. This is the same
data that `who`, `w` and `loginctl list-sessions` show. On systemd hosts, the
collector uses `loginctl` first. `loginctl` shows graphical (GDM/KDE),
remote-desktop and `systemd-run` sessions that do not get to `utmp`. On Linux
hosts without systemd and on macOS, the collector uses `utmp` or `utmpx` through
gopsutil.

This collector came out of the original `users` collector. `users` is now the
same as Ohai's `passwd` plugin, which lists `/etc/passwd` and `/etc/group`. The
logged-in sessions are in this collector. Ohai has no direct equivalent. This
collector is a gohai extension. Consumers need it to find active interactive
sessions for audit and security tools.

## Collected fields

| Field       | Type        | Description                            | Schema mapping             |
| ----------- | ----------- | -------------------------------------- | -------------------------- |
| `logged_in` | `[]Session` | One entry for each active user session | OCSF `session` (per-entry) |

### Session

| Field        | Type     | Description                                                        |
| ------------ | -------- | ------------------------------------------------------------------ |
| `user`       | `string` | The login name.                                                    |
| `terminal`   | `string` | The tty, pts or console (utmp path only).                          |
| `host`       | `string` | The remote host that the session came from (utmp path only).       |
| `started`    | `uint64` | The session start time as a Unix timestamp (utmp path only).       |
| `session_id` | `string` | The systemd session ID (loginctl path only).                       |
| `uid`        | `string` | The numeric UID of the session owner (loginctl path only).         |
| `seat`       | `string` | The systemd seat identifier, usually `seat0` (loginctl path only). |

A field that only one path gives stays empty on the other path. To find which
source gave the data, a consumer checks for `session_id` (loginctl) or for
`terminal` and `started` (utmp).

## Platform support

| Platform | Supported                      |
| -------- | ------------------------------ |
| Linux    | ✅ (loginctl first, then utmp) |
| macOS    | ✅ (utmpx only, no systemd)    |

## Example output

### systemd host (loginctl path)

```json
{
  "sessions": {
    "logged_in": [
      {
        "user": "john",
        "session_id": "c1",
        "uid": "1000",
        "seat": "seat0"
      },
      { "user": "root", "session_id": "2", "uid": "0" }
    ]
  }
}
```

### Non-systemd or macOS (utmp path)

```json
{
  "sessions": {
    "logged_in": [
      {
        "user": "john",
        "terminal": "pts/0",
        "host": "10.0.0.1",
        "started": 1712908800
      }
    ]
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("sessions"))
facts, _ := g.Collect(context.Background())

for _, s := range facts.Sessions.LoggedIn {
    fmt.Printf("%s on %s\n", s.User, s.Terminal)
}
```

## Enable/disable

```bash
gohai --collector.sessions      # enable (opt-in)
gohai --no-collector.sessions   # disable (default)
```

## Dependencies

None.

## Data sources

On Linux:

1. If `loginctl` is on the PATH, the collector runs
   `loginctl --no-pager --no-legend --no-ask-password list-sessions` through the
   shared `internal/executor`. It splits each line that is not empty at the
   whitespace into `session_id uid user [seat]`. It adds the result to
   `logged_in[]`. Empty output means that there are no active sessions. This is
   a valid result, and the collector does not try utmp in that case.
2. If `loginctl` is missing or fails, the collector uses gopsutil's
   `host.UsersWithContext`. This occurs on a host without systemd or in a
   minimal container without systemd. `host.UsersWithContext` reads `utmp(5)`,
   which is the same source that `who` and `w` use. A gopsutil error goes back
   to the caller as a real failure.

On macOS:

1. gopsutil's `host.UsersWithContext` reads `utmpx` through the standard C
   library. This is the only source, because Darwin has no systemd. Errors go
   back to the caller.

Ohai has no equivalent plugin. gohai added this collector for consumers that
audit sessions across a fleet. Current Linux distributions document `loginctl`
as the standard way to find sessions, so the collector uses it first. The
collector keeps utmp for hosts from before systemd and for containers.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) to
  read `utmp` and `utmpx`.
- [`internal/executor`](../../internal/executor) to run `loginctl` on Linux.
