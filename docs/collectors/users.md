# Users

> **Status:** Implemented ⚠️ (logged-in sessions only; `/etc/passwd` enumeration
> planned)

## Description

Reports currently logged-in user sessions. On systemd hosts we prefer
`loginctl list-sessions` — the same data `loginctl` itself shows — which
surfaces graphical (GDM/KDE), remote-desktop, and `systemd-run` sessions that
never reach `utmp`. On non-systemd hosts and macOS we fall back to `utmp` /
`utmpx` via gopsutil, which matches `who` / `w` output.

**Scope note:** despite the name, this collector covers _logged-in sessions
only_ — it does NOT enumerate `/etc/passwd`. Ohai splits this into two plugins
(`passwd.rb` for account enumeration, `linux/sessions.rb` for loginctl). gohai
has a planned `passwd` collector for the enumeration gap and a planned
`sessions` collector that may take over the logged-in half (see
[README](README.md#-users--sessions)).

Consumers use this to:

- Detect unexpected interactive sessions.
- Audit origin IPs for SSH sessions.
- Correlate session start times with other events.

## Collected Fields

| Field                    | Type   | Description                                                                | Schema mapping                               |
| ------------------------ | ------ | -------------------------------------------------------------------------- | -------------------------------------------- |
| `logged_in[].user`       | string | Username.                                                                  | `user.name`.                                 |
| `logged_in[].terminal`   | string | Terminal (`pts/0`, `ttys001`). Empty for remote / graphical.               | No direct schema mapping.                    |
| `logged_in[].host`       | string | Origin host for remote logins (IP or hostname).                            | `src_endpoint.hostname` / `src_endpoint.ip`. |
| `logged_in[].started`    | uint64 | Session start time (unix seconds). Populated from utmp path only.          | No direct schema mapping.                    |
| `logged_in[].session_id` | string | systemd session id (`c1`, `2`). Empty on utmp fallback.                    | `process.session_uid` (nearest).             |
| `logged_in[].uid`        | string | Numeric UID as reported by loginctl. Empty on utmp fallback.               | `user.uid`.                                  |
| `logged_in[].seat`       | string | systemd seat (`seat0`, empty for remote sessions). Empty on utmp fallback. | No direct schema mapping.                    |

## Platform Support

| Platform | Supported                                                    |
| -------- | ------------------------------------------------------------ |
| Linux    | ✅ (`loginctl list-sessions` via executor; utmp fallback)    |
| macOS    | ✅ (`/var/run/utmpx` via gopsutil — no `loginctl` on Darwin) |

## Example Output

### Linux with systemd (loginctl)

```json
{
  "users": {
    "logged_in": [
      { "user": "alice", "session_id": "c1", "uid": "1000", "seat": "seat0" },
      { "user": "bob", "session_id": "2", "uid": "1001" }
    ]
  }
}
```

### Linux without systemd / macOS (utmp fallback)

```json
{
  "users": {
    "logged_in": [
      {
        "user": "alice",
        "terminal": "pts/0",
        "host": "10.0.0.42",
        "started": 1712068800
      }
    ]
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("users"))
facts, _ := g.Collect(context.Background())

for _, s := range facts.Users.LoggedIn {
    if s.Host != "" {
        fmt.Printf("%s logged in from %s on %s\n", s.User, s.Host, s.Terminal)
    }
}
```

## Enable/Disable

```bash
gohai --collector.users      # opt-in (off by default — niche)
gohai --no-collector.users   # explicitly disable (e.g. when stripping defaults)
```

## Dependencies

None.

## Data Sources

On Linux the collector probes for `loginctl` on PATH:

1. **When present (systemd hosts):** run
   `loginctl --no-pager --no-legend --no-ask-password list-sessions` through the
   shared `internal/executor` runner and parse each line's
   `session, uid, user, seat` whitespace-split columns into a `Session`.
   `terminal`, `host`, and `started` are left empty unless we later extend with
   `loginctl show-session` enrichment.
2. **When absent or errors (non-systemd hosts, minimized containers):** fall
   back to gopsutil's `host.Users`, which reads `/var/run/utmp`. `session_id`,
   `uid`, and `seat` are empty in this mode.

On macOS we read `/var/run/utmpx` via gopsutil — `loginctl` does not exist on
Darwin.

The `loginctl` extension is the methodology Ohai uses
([`linux/sessions.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/sessions.rb))
and matters because graphical (GDM/KDE), remote-desktop, and `systemd-run`
sessions never write utmp — gopsutil's utmp-only path silently misses them.

Ohai's
[`passwd.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/passwd.rb)
(which enumerates `/etc/passwd` + `/etc/group`) is the scope of the planned
gohai `passwd` collector — it will not be merged into this one.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `loginctl` on Linux. Tests mock it with
  `go.uber.org/mock`.
- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3. Fallback (Linux non-systemd) and primary source on macOS for utmp /
  utmpx.
