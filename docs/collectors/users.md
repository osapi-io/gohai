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
- Get facts like `root_group` or `current_user` without a second lookup.

## Collected fields

| Field          | Type                     | Description                                                                                                    | Schema mapping            |
| -------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------- | ------------------------- |
| `passwd`       | `map[string]PasswdEntry` | The accounts, with the username as the key. If a username occurs two times, the first entry stays.             | No direct schema mapping. |
| `group`        | `map[string]GroupEntry`  | The groups, with the group name as the key.                                                                    | No direct schema mapping. |
| `current_user` | `string`                 | The username of the effective UID: the name `id -un` gives, else the first name in sorted order with that UID. | OCSF `actor.user.name`    |

### PasswdEntry

| Field    | Type     | Description                                                                              |
| -------- | -------- | ---------------------------------------------------------------------------------------- |
| `uid`    | `int`    | The numeric user ID.                                                                     |
| `gid`    | `int`    | The primary numeric group ID.                                                            |
| `dir`    | `string` | The path of the home directory.                                                          |
| `shell`  | `string` | The path of the login shell.                                                             |
| `gecos`  | `string` | The comment or display-name field. It usually holds a full name and other data.          |
| `locked` | `bool`   | Whether the account cannot log in with a password. Unset when the caller cannot read it. |

### GroupEntry

| Field     | Type       | Description                                          |
| --------- | ---------- | ---------------------------------------------------- |
| `gid`     | `int`      | The numeric group ID.                                |
| `members` | `[]string` | The supplementary members, in the order of the file. |

## Platform support

| Platform | Supported                                  |
| -------- | ------------------------------------------ |
| Linux    | ✅                                         |
| macOS    | ✅ (the directory service, through `dscl`) |

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
gohai collect --collector.users      # enable (opt-in)
gohai collect --no-collector.users   # disable (default)
```

## Dependencies

None.

## Data sources

On Linux, the collector reads the POSIX flat files:

1. The collector reads `/etc/passwd` through the injected `avfs.VFS`. It parses
   each line into `passwd[<username>]`. Each line has seven fields with colons
   between them: `name:password:uid:gid:gecos:home:shell`. The collector keeps
   `uid`, `gid`, `dir`, `shell` and `gecos`. It does not keep the placeholder of
   the password hash. If a username occurs two times, the first entry stays.
   This is the same as the C-library loop of `Etc.passwd`. The collector skips
   comments (`#`) and lines that are not valid (fewer than 7 fields). A missing
   `/etc/passwd` gives an empty map. One that exists and does not read is an
   error, and the collector keeps the facts of the files that did read, with the
   status `partial`.
2. The collector reads `/etc/group` through the same VFS. Each line has four
   fields with colons between them: `name:password:gid:members`, with commas
   between the members. The collector keeps `gid` and the `members` slice that
   it splits on the commas. The skip rules are the same, and so are the rules
   for a missing file and a file that does not read.
3. The collector gets `current_user`. It runs `id -u` through its command runner
   for the effective UID. Then it finds the entry with that UID in the parsed
   `passwd` map. This is the same as Ohai's `Etc.getpwuid(Process.euid).name`.
   If `id` fails, if `/etc/passwd` is missing, or if no entry has the euid,
   `current_user` stays empty.

The method is the same as Ohai's `passwd` plugin. It reads the same files. It
keeps the first entry of a duplicate username. It gets the current user from the
effective UID. gohai parses `/etc/passwd` directly and does not call
`getpwent(3)`. Thus the collector has no cgo dependency.

On macOS, the local accounts are in the directory service (`dscl .`). The flat
files `/etc/passwd` and `/etc/group` have only the well-known system accounts
(`root`, `daemon` and others). They do not have the accounts that a person made.
Thus the collector reads the accounts with `dscl`. A user that is not root can
run these commands:

1. `dscl . -list /Users <attribute>` for `UniqueID`, `PrimaryGroupID`,
   `NFSHomeDirectory`, `UserShell` and `RealName`. Each line has the name of a
   record and its value. Each user with a numeric `UniqueID` gives an entry in
   `passwd`: `uid`, `gid`, `dir`, `shell` and `gecos` (the `RealName`). A user
   with two homes keeps the first.
2. `dscl . -list /Groups PrimaryGroupID` and
   `dscl . -list /Groups GroupMembership`. Each group with a numeric
   `PrimaryGroupID` gives an entry in `group`, with the members that
   `GroupMembership` names.
3. If the host has no `dscl`, or the directory service lists no users, the
   collector uses `/etc/passwd`. For the groups, it uses `/etc/group` in the
   same way. A `dscl` run that fails is an error. The collector keeps the facts
   of the flat files with that error, and its status is `partial`, so a caller
   does not take the flat files' system accounts for the directory service.
4. The collector gets `current_user` from `id -u`, as on Linux.

This agrees with Ohai. Ohai's `passwd` plugin uses Ruby's `Etc.passwd`, which
calls `getpwent(3)`. On macOS, `getpwent(3)` reads the directory service. Thus
Ohai also reports the accounts that a person made.

The collector also reads `/etc/shadow` when the caller can (root). An account
whose password field starts with `!` or `*` is `locked`, as `passwd -S` reports
`L`. On macOS, one `dscl . -list /Users AuthenticationAuthority` run gives each
account: `DisabledUser` is locked. When neither is readable, `locked` is unset.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for `/etc/passwd` and
  `/etc/group` reads.
- `source.Runner` to run `id -u`.
