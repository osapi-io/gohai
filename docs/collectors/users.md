# Users

> **Status:** Implemented ✅

## Description

Enumerates user and group accounts from `/etc/passwd` and `/etc/group`, plus the
effective current user. Matches Ohai's `passwd` plugin methodology. Logged-in
session data (who is currently signed in) lives in the separate
[sessions](sessions.md) collector.

Consumers use this to:

- Audit accounts and group membership across a fleet.
- Check that a specific user / uid / shell exists on the host.
- Resolve `root_group` or `current_user`-like facts without a second `os/user`
  lookup.

## Collected Fields

| Field          | Type                     | Description                                                      | Schema mapping            |
| -------------- | ------------------------ | ---------------------------------------------------------------- | ------------------------- |
| `passwd`       | `map[string]PasswdEntry` | Accounts keyed by username. First occurrence wins on duplicates. | No direct schema mapping. |
| `group`        | `map[string]GroupEntry`  | Groups keyed by group name.                                      | No direct schema mapping. |
| `current_user` | `string`                 | Username corresponding to the effective UID at collection time.  | OCSF `actor.user.name`    |

### PasswdEntry

| Field   | Type     | Description                                                 |
| ------- | -------- | ----------------------------------------------------------- |
| `uid`   | `int`    | Numeric user ID.                                            |
| `gid`   | `int`    | Primary numeric group ID.                                   |
| `dir`   | `string` | Home directory path.                                        |
| `shell` | `string` | Login shell path.                                           |
| `gecos` | `string` | Comment / display-name field (commonly a full name + info). |

### GroupEntry

| Field     | Type       | Description                                    |
| --------- | ---------- | ---------------------------------------------- |
| `gid`     | `int`      | Numeric group ID.                              |
| `members` | `[]string` | Supplementary members, in file-declared order. |

## Platform Support

| Platform | Supported                                                 |
| -------- | --------------------------------------------------------- |
| Linux    | ✅                                                        |
| macOS    | ✅ (macOS ships POSIX flat files alongside OpenDirectory) |

## Example Output

```json
{
  "users": {
    "passwd": {
      "root": {
        "uid": 0,
        "gid": 0,
        "dir": "/root",
        "shell": "/bin/bash",
        "gecos": "root"
      },
      "john": {
        "uid": 1000,
        "gid": 1000,
        "dir": "/home/john",
        "shell": "/bin/zsh",
        "gecos": "John Doe"
      }
    },
    "group": {
      "wheel": { "gid": 10, "members": ["john", "root"] },
      "users": { "gid": 100, "members": ["john"] }
    },
    "current_user": "john"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("users"))
facts, _ := g.Collect(context.Background())

if entry, ok := facts.Users.Passwd["deploy"]; ok {
    fmt.Printf("deploy user uid=%d shell=%s\n", entry.UID, entry.Shell)
}
```

## Enable/Disable

```bash
gohai --collector.users      # enable (opt-in)
gohai --no-collector.users   # disable (default)
```

## Dependencies

None.

## Data Sources

On Linux and macOS (identical — both platforms ship POSIX flat files):

1. Read `/etc/passwd` through the injected `avfs.VFS` and parse line-by-line
   into `passwd[<username>]`. Format is seven colon-separated fields:
   `name:password:uid:gid:gecos:home:shell`. We keep `uid`, `gid`, `dir`,
   `shell`, and `gecos`; the password hash placeholder is dropped. Duplicate
   usernames keep the first occurrence — matches `Etc.passwd`'s C-library
   iteration. Comments (`#`) and malformed lines (< 7 fields) are skipped.
2. Read `/etc/group` through the same VFS. Format is four colon-separated
   fields: `name:password:gid:members(comma-separated)`. We keep `gid` and the
   comma-split `members` slice. Same skip rules.
3. Resolve `current_user` by calling `os.Geteuid()` (the effective UID) and
   looking up the matching entry in the parsed `passwd` map. Matches Ohai's
   `Etc.getpwuid(Process.euid).name`. Missing `/etc/passwd` or an euid with no
   matching entry leaves `current_user` empty.

Mirrors Ohai's `passwd` plugin — same files, same duplicate-drops-second rule,
same effective-UID lookup for current user. We parse `/etc/passwd` directly
rather than calling `getpwent(3)` so the collector has no cgo dependency.

On macOS, local accounts live in Directory Services (`dscl .`). The flat files
`/etc/passwd` and `/etc/group` carry the well-known system users (`root`,
`daemon`, etc.) but omit user-created accounts — those live in OpenDirectory and
require `dscl` to enumerate. Ohai's `passwd` plugin has the same limitation
(Ruby's `Etc` wraps POSIX flat files on macOS); a future enhancement could shell
out to `dscl . -list /Users` for a richer view.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for `/etc/passwd` and
  `/etc/group` reads.
- Go stdlib (`os.Geteuid`, `bufio`, `strings`, `strconv`) for parsing.
