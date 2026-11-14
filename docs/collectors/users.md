# Users

> **Status:** Implemented ✅

## Description

The collector reports the user and group accounts from `/etc/passwd` and
`/etc/group`. It also reports the effective current user. The method is the same
as Ohai's `passwd` plugin. The [sessions](sessions.md) collector reports the
logged-in sessions, that is, who is signed in now.

Consumers use this to:

- Audit the accounts and the group membership of a fleet.
- Make sure that a specific user, uid or shell is on the host.
- Get facts like `root_group` or `current_user` without a second `os/user`
  lookup.

## Collected fields

| Field          | Type                     | Description                                                                                        | Schema mapping            |
| -------------- | ------------------------ | -------------------------------------------------------------------------------------------------- | ------------------------- |
| `passwd`       | `map[string]PasswdEntry` | The accounts, with the username as the key. If a username occurs two times, the first entry stays. | No direct schema mapping. |
| `group`        | `map[string]GroupEntry`  | The groups, with the group name as the key.                                                        | No direct schema mapping. |
| `current_user` | `string`                 | The username of the effective UID at the time of collection.                                       | OCSF `actor.user.name`    |

### PasswdEntry

| Field   | Type     | Description                                                                     |
| ------- | -------- | ------------------------------------------------------------------------------- |
| `uid`   | `int`    | The numeric user ID.                                                            |
| `gid`   | `int`    | The primary numeric group ID.                                                   |
| `dir`   | `string` | The path of the home directory.                                                 |
| `shell` | `string` | The path of the login shell.                                                    |
| `gecos` | `string` | The comment or display-name field. It usually holds a full name and other data. |

### GroupEntry

| Field     | Type       | Description                                          |
| --------- | ---------- | ---------------------------------------------------- |
| `gid`     | `int`      | The numeric group ID.                                |
| `members` | `[]string` | The supplementary members, in the order of the file. |

## Platform support

| Platform | Supported                                              |
| -------- | ------------------------------------------------------ |
| Linux    | ✅                                                     |
| macOS    | ✅ (macOS has POSIX flat files and also OpenDirectory) |

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("users"))
facts, _ := g.Collect(context.Background())

if entry, ok := facts.Users.Passwd["deploy"]; ok {
    fmt.Printf("deploy user uid=%d shell=%s\n", entry.UID, entry.Shell)
}
```

## Enable/disable

```bash
gohai --collector.users      # enable (opt-in)
gohai --no-collector.users   # disable (default)
```

## Dependencies

None.

## Data sources

On Linux and macOS, the method is the same, because the two platforms have POSIX
flat files:

1. The collector reads `/etc/passwd` through the injected `avfs.VFS`. It parses
   each line into `passwd[<username>]`. Each line has seven fields with colons
   between them: `name:password:uid:gid:gecos:home:shell`. The collector keeps
   `uid`, `gid`, `dir`, `shell` and `gecos`. It does not keep the placeholder of
   the password hash. If a username occurs two times, the first entry stays.
   This is the same as the C-library loop of `Etc.passwd`. The collector skips
   comments (`#`) and lines that are not valid (fewer than 7 fields).
2. The collector reads `/etc/group` through the same VFS. Each line has four
   fields with colons between them: `name:password:gid:members`, with commas
   between the members. The collector keeps `gid` and the `members` slice that
   it splits on the commas. The skip rules are the same.
3. The collector gets `current_user`. It calls `os.Geteuid()` for the effective
   UID. Then it finds the entry with that UID in the parsed `passwd` map. This
   is the same as Ohai's `Etc.getpwuid(Process.euid).name`. If `/etc/passwd` is
   missing, or if no entry has the euid, `current_user` stays empty.

The method is the same as Ohai's `passwd` plugin. It reads the same files. It
keeps the first entry of a duplicate username. It gets the current user from the
effective UID. gohai parses `/etc/passwd` directly and does not call
`getpwent(3)`. Thus the collector has no cgo dependency.

On macOS, the local accounts are in Directory Services (`dscl .`). The flat
files `/etc/passwd` and `/etc/group` have the well-known system users (`root`,
`daemon` and others). They do not have the accounts that a person made. Those
accounts are in OpenDirectory, and only `dscl` can list them. Ohai's `passwd`
plugin has the same limit, because Ruby's `Etc` reads the POSIX flat files on
macOS. A future change can run `dscl . -list /Users` to get all accounts.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for `/etc/passwd` and
  `/etc/group` reads.
- The Go standard library (`os.Geteuid`, `bufio`, `strings`, `strconv`) to parse
  the files.
