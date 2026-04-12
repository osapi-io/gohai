# Shells

> **Status:** Implemented ✅

## Description

Reports the list of valid login shells installed on the host, as listed in
`/etc/shells`. This file is maintained by the system package manager (and
sometimes edited by operators) and is the authoritative source consulted by
`login(1)`, `chsh(1)`, FTP daemons, and other tools that need to decide whether
a given shell is "legitimate" — a user whose `passwd` entry points to a shell
not in `/etc/shells` is typically treated as non-interactive.

Consumers use this to:

- Enumerate which shells can be assigned as a login shell (fleet inventory,
  policy compliance — e.g. "is `/bin/zsh` available on this host?").
- Drive remediation tooling that installs a missing shell before `chsh`.
- Spot drift across a fleet (hosts where `/etc/shells` has been hand-edited or
  pruned).

Comments (lines beginning with `#`) and blank lines are stripped; leading and
trailing whitespace on each entry is trimmed.

## Collected Fields

| Field   | Type       | Description                                          |
| ------- | ---------- | ---------------------------------------------------- |
| `paths` | `[]string` | Absolute paths to valid login shells, in file order. |

## Platform Support

| Platform | Source        | Supported |
| -------- | ------------- | --------- |
| Linux    | `/etc/shells` | ✅        |
| macOS    | `/etc/shells` | ✅        |
| Other    | —             | `nil`     |

If `/etc/shells` is missing (uncommon, but possible on minimal containers),
`Collect` returns an error and the field is left `nil` on `Facts`.

## Example Output

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

## SDK Usage

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

## Enable/Disable

```bash
gohai --collector.shells      # enable (default)
gohai --no-collector.shells   # disable
```

## Dependencies

None — Tier 1 core collector with no upstream collector dependencies.

## Backing library

- Go stdlib (`os`, `bufio`) — no third-party dependency.
