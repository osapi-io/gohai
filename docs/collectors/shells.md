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
trailing whitespace on each entry is trimmed. Non-absolute entries (anything
that doesn't start with `/`) are ignored — matches Ohai's strict `/`-prefix
filter.

## Collected Fields

| Field   | Type       | Description                                          | OCSF mapping                                                                                                                       |
| ------- | ---------- | ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `paths` | `[]string` | Absolute paths to valid login shells, in file order. | No direct OCSF equivalent — OCSF has `user.shell` per-user but no host-level shell inventory object. Treated as a gohai extension. |

## Platform Support

| Platform | Supported  |
| -------- | ---------- |
| Linux    | ✅         |
| macOS    | ✅         |
| Other    | Empty list |

Missing `/etc/shells` (distroless/scratch containers) soft-misses to an empty
list rather than erroring — matches Ohai's `file_exist?` gate.

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

### Minimal container without /etc/shells

```json
{
  "shells": {
    "paths": []
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

None.

## Data Sources

| Platform | What we read  | Ohai plugin ([`shells.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/shells.rb)) | Alignment                                                                         |
| -------- | ------------- | ---------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Linux    | `/etc/shells` | `/etc/shells` via `file_open` + line iteration; `line[0] == "/"` filter                        | **Identical.** Same file, same `/`-prefix filter, same soft-miss on missing file. |
| macOS    | `/etc/shells` | Same (no `:darwin`-specific code path in Ohai — same plugin runs).                             | **Identical.**                                                                    |

**Methodology notes vs. Ohai:**

- We `TrimSpace` each line before the `/`-prefix check, so paths with leading
  whitespace (e.g. `  /bin/bash`) are accepted. Ohai tests `line[0] == "/"` on
  the raw line and would skip a space-prefixed entry. Permissive by design —
  indented entries in `/etc/shells` aren't a real case but trimming is safer
  behavior.
- On a missing `/etc/shells` we soft-miss to `{paths: []}`. Ohai omits the
  `shells` attribute entirely. Our typed struct is always present — this is a
  Go-idiom divergence, not a collection divergence.

No tracked methodology gaps. Both divergences above are intentional.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for the `/etc/shells` read.
- Go stdlib `bufio` for line scanning.
