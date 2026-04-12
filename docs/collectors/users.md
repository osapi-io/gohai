# Users

> **Status:** Implemented ✅ (logged-in sessions only)

## Description

Lists currently logged-in user sessions (e.g., `who`/`w` output). Wraps
[gopsutil's `host.Users`](https://pkg.go.dev/github.com/shirou/gopsutil/v4/host).

Note: this collector does NOT enumerate all accounts in `/etc/passwd`. For
passwd/group enumeration a future enhancement would parse those files directly.
The README description mentions "passwd/group data, current user" which is
broader — this collector currently only covers the logged-in side.

## Collected Fields

Top-level: `logged_in []Session`.

Per-session:

| Field      | Type   | Description                               |
| ---------- | ------ | ----------------------------------------- |
| `user`     | string | User name                                 |
| `terminal` | string | Terminal (e.g., `tty1`, `pts/0`)          |
| `host`     | string | Origin host (e.g., `192.168.1.5` for SSH) |
| `started`  | uint64 | Unix timestamp of session start           |

## Platform Support

| Platform | Source                              | Supported |
| -------- | ----------------------------------- | --------- |
| Linux    | `gopsutil/v4/host.UsersWithContext` | ✅        |
| macOS    | `gopsutil/v4/host.UsersWithContext` | ✅        |
| Other    | Returns `nil`                       | —         |

## Example Output

```json
{
  "users": {
    "logged_in": [
      {
        "user": "john",
        "terminal": "pts/0",
        "host": "192.168.1.5",
        "started": 1712908800
      },
      {
        "user": "root",
        "terminal": "tty1",
        "started": 1712900000
      }
    ]
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("users"))
facts, _ := g.Collect(context.Background())

for _, s := range facts.Users.LoggedIn {
    fmt.Printf("%s on %s from %s\n", s.User, s.Terminal, s.Host)
}
```

## Enable/Disable

```bash
gohai --collector.users      # enable (default)
gohai --no-collector.users   # disable
```

## Dependencies

None.

## Backing library

[`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
BSD-3.
