# Sessions

> **Status:** Implemented ✅

## Description

Reports currently logged-in user sessions — the same data `who` / `w` /
`loginctl list-sessions` show. On systemd hosts we prefer `loginctl`, which
surfaces graphical (GDM/KDE), remote-desktop, and `systemd-run` sessions that
never reach `utmp`. On non-systemd Linux hosts and macOS we fall back to `utmp`
/ `utmpx` via gopsutil.

This collector was split out of the original `users` collector — `users` now
matches Ohai's `passwd` plugin (enumerate `/etc/passwd` + `/etc/group`) and
logged-in sessions live here. Ohai does not have a direct equivalent; this is a
gohai extension driven by the consumer need to detect active interactive
sessions for audit and security tooling.

## Collected Fields

| Field       | Type        | Description                       | Schema mapping             |
| ----------- | ----------- | --------------------------------- | -------------------------- |
| `logged_in` | `[]Session` | One entry per active user session | OCSF `session` (per-entry) |

### Session

| Field        | Type     | Description                                                      |
| ------------ | -------- | ---------------------------------------------------------------- |
| `user`       | `string` | Login name.                                                      |
| `terminal`   | `string` | tty / pts / console (utmp path only).                            |
| `host`       | `string` | Remote host the session originated from (utmp path only).        |
| `started`    | `uint64` | Session start time as Unix timestamp (utmp path only).           |
| `session_id` | `string` | systemd session id (loginctl path only).                         |
| `uid`        | `string` | Numeric UID of the session owner (loginctl path only).           |
| `seat`       | `string` | systemd seat identifier, typically `seat0` (loginctl path only). |

Fields present only on one path stay empty on the other — consumers can detect
which source fed the data by checking for `session_id` (loginctl) vs.
`terminal`/`started` (utmp).

## Platform Support

| Platform | Supported                              |
| -------- | -------------------------------------- |
| Linux    | ✅ (loginctl preferred, utmp fallback) |
| macOS    | ✅ (utmpx only — no systemd)           |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("sessions"))
facts, _ := g.Collect(context.Background())

for _, s := range facts.Sessions.LoggedIn {
    fmt.Printf("%s on %s\n", s.User, s.Terminal)
}
```

## Enable/Disable

```bash
gohai --collector.sessions      # enable (opt-in)
gohai --no-collector.sessions   # disable (default)
```

## Dependencies

None.

## Data Sources

On Linux:

1. If `loginctl` is on PATH, run
   `loginctl --no-pager --no-legend --no-ask-password list-sessions` through the
   shared `internal/executor`. Each non-empty line is whitespace-split into
   `session_id uid user [seat]` and appended to `logged_in[]`. Empty output (no
   active sessions) is a valid result — we don't fall back in that case.
2. If `loginctl` is missing or errors (non-systemd host, minimal container
   without systemd), fall back to gopsutil's `host.UsersWithContext` which reads
   `utmp(5)` — the same source `who` / `w` consult. Gopsutil errors propagate as
   a real failure.

On macOS:

1. gopsutil's `host.UsersWithContext` reads `utmpx` via the standard C library.
   That's the only source — Darwin has no systemd. Errors propagate.

Ohai ships no equivalent plugin — this is a gohai-native surface added for
consumers doing fleet-level session auditing. The loginctl-first ordering
mirrors what modern Linux distros document as the canonical session-discovery
path; utmp is retained for pre-systemd and container compatibility.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) for
  `utmp` / `utmpx` reads.
- [`internal/executor`](../../internal/executor) for the `loginctl` call on
  Linux.
