# Users

> **Status:** Implemented ⚠️ (logged-in sessions only; `/etc/passwd` enumeration
> planned)

## Description

Reports currently logged-in user sessions — the same data `who` / `utmp` / the
`w` command surface. Each session carries the user, terminal, origin host (for
remote logins), and session start time.

**Scope note:** despite the name, this collector currently covers _logged-in
sessions only_ — it does NOT enumerate `/etc/passwd`. Ohai splits this into two
plugins (`passwd.rb` for account enumeration, `linux/sessions.rb` for logged-in
sessions). gohai has a planned `passwd` collector for the enumeration gap and a
planned `sessions` collector that will take over the logged-in half (see
[README](README.md#-users--sessions)).

Consumers use this to:

- Detect unexpected interactive sessions (security — who's logged in right
  now?).
- Audit origin IPs for SSH sessions (remote-access pattern).
- Correlate session start times with other events.

## Collected Fields

| Field                  | Type   | Description                                      | Schema mapping                               |
| ---------------------- | ------ | ------------------------------------------------ | -------------------------------------------- |
| `logged_in[].user`     | string | Username.                                        | `user.name`.                                 |
| `logged_in[].terminal` | string | Terminal (`pts/0`, `ttys001`). Empty for remote. | No direct OCSF.                              |
| `logged_in[].host`     | string | Origin host for remote logins (IP or hostname).  | `src_endpoint.hostname` / `src_endpoint.ip`. |
| `logged_in[].started`  | uint64 | Session start time (unix seconds).               | No direct OCSF.                              |

## Platform Support

| Platform | Supported                          |
| -------- | ---------------------------------- |
| Linux    | ✅ (`/var/run/utmp` via gopsutil)  |
| macOS    | ✅ (`/var/run/utmpx` via gopsutil) |

## Example Output

```json
{
  "users": {
    "logged_in": [
      {
        "user": "alice",
        "terminal": "pts/0",
        "host": "10.0.0.42",
        "started": 1712068800
      },
      {
        "user": "bob",
        "terminal": "tty1",
        "started": 1712032000
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
gohai --collector.users      # enable (default)
gohai --no-collector.users   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                    | Ohai plugin                                                                                                                                            | Alignment                                                                                                                                                                                                                                                                       |
| -------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | gopsutil `host.Users` (reads `/var/run/utmp`).  | [`linux/sessions.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/sessions.rb) — `loginctl list-sessions` + `loginctl show-session`. | **Different source, overlapping scope.** Ohai uses `loginctl` (systemd) which reports more detail per session (seat, VT, type, service). gopsutil reads `utmp` — simpler, cross-platform, no systemd dependency. `loginctl` enrichment is planned for the `sessions` collector. |
| macOS    | gopsutil `host.Users` (reads `/var/run/utmpx`). | Ohai has no dedicated macOS session plugin — only `passwd.rb` for account enumeration.                                                                 | **gohai extension on macOS.**                                                                                                                                                                                                                                                   |

Ohai's
[`passwd.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/passwd.rb)
(which enumerates `/etc/passwd` + `/etc/group`) is the scope of the planned
gohai `passwd` collector — it will not be merged into this one.

**Known gaps vs. Ohai:**

- No `/etc/passwd` / `/etc/group` enumeration (planned `passwd` collector).
- No systemd-session detail: seat, vtnr, type, service, scope, etc. (planned
  `sessions` collector).

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3.
