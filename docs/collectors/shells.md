# Shells

> **Status:** Implemented ✅

## Description

The collector reports the list of valid login shells on the host, from
`/etc/shells`. The system package manager keeps this file, and operators
sometimes edit it. `login(1)`, `chsh(1)`, FTP daemons and other tools read this
file to decide if a shell is "legitimate". If a user's `passwd` entry points to
a shell that is not in `/etc/shells`, these tools usually treat the user as
non-interactive.

Consumers use this data to:

- List the shells that you can give a user as a login shell, for fleet inventory
  and policy compliance. For example: "is `/bin/zsh` available on this host?"
- Start remediation tools that install a missing shell before `chsh`.
- Find drift across a fleet: hosts where an operator edited `/etc/shells` by
  hand or removed entries.

The collector removes comments (lines that start with `#`) and blank lines. It
removes the whitespace at the start and end of each entry. It ignores entries
that are not absolute, that is, entries that do not start with `/`. This is the
same as Ohai's strict `/`-prefix filter.

## Collected fields

| Field   | Type       | Description                                                  | Schema mapping                                                                                                                               |
| ------- | ---------- | ------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `paths` | `[]string` | The absolute paths of the valid login shells, in file order. | No direct schema mapping. OCSF has `user.shell` for each user, but no object for the shells of a host. gohai treats it as a gohai extension. |

## Platform support

| Platform | Supported  |
| -------- | ---------- |
| Linux    | ✅         |
| macOS    | ✅         |
| Other    | Empty list |

If `/etc/shells` is missing, the collector returns an empty list, not an error.
Distroless and scratch containers have no `/etc/shells`. This is the same as
Ohai's `file_exist?` check.

## Example output

### Typical Linux

```json
{
  "shells": {
    "paths": [
      "/bin/sh",
      "/bin/bash",
      "/usr/bin/bash",
      "/bin/dash",
      "/usr/bin/dash",
      "/usr/bin/zsh",
      "/bin/zsh"
    ]
  }
}
```

### Typical macOS

```json
{
  "shells": {
    "paths": [
      "/bin/bash",
      "/bin/csh",
      "/bin/dash",
      "/bin/ksh",
      "/bin/sh",
      "/bin/tcsh",
      "/bin/zsh"
    ]
  }
}
```

### Minimal container without /etc/shells

```json
{
  "shells": {
    "paths": []
  }
}
```

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("shells"))
facts, _ := g.Collect(context.Background())

for _, p := range facts.Shells.Paths {
    fmt.Println(p)
}
```

## Enable/disable

```bash
gohai collect --collector.shells      # enable (default)
gohai collect --no-collector.shells   # disable
```

## Dependencies

None.

## Data sources

On Linux and macOS, the steps are the same. `/etc/shells` follows the POSIX
convention, and Ohai's plugin has no special path for `:darwin`.

1. The collector reads `/etc/shells` through the injected `avfs.VFS`. If the
   file is missing, the collector returns `{paths: []}`, not an error.
   Distroless containers and scratch images have no `/etc/shells`. In the same
   case, Ohai does not report the `shells` attribute at all. The gohai typed
   struct is always there. The empty slice is a difference in Go style, not a
   difference in the collected data.
2. The collector reads the file line by line. It removes the whitespace at the
   start and end of each line. It skips blank lines and comment lines (`#`
   prefix). It skips each line that does not start with `/` after this trim. The
   `TrimSpace` step accepts a little more than Ohai's `line[0] == "/"` check on
   the raw line. For example, gohai accepts `  /bin/bash` as a valid entry. This
   is intentional, safer behavior.
3. The collector adds the remaining absolute paths to `paths[]` in file order.

If the read fails because the permission is denied, or for any other reason than
a missing file, the error goes back to the caller as a real failure. This is the
same as Ohai, which lets `file_open` raise the error.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) to read `/etc/shells`.
  Production uses `osfs`. Tests use `memfs`.
- The Go standard library `strings` to read the lines.
