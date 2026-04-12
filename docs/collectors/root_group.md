# Root Group

> **Status:** Implemented ✅

## Description

Reports the name of root's primary group. Two-hop lookup (matches Ohai): resolve
user `root` → its primary GID → group name. Different operating systems use
different conventions:

- **Linux:** almost always `root`
- **macOS / BSD:** `wheel`
- **Custom / hardened hosts:** whatever site policy sets

Consumers use this to:

- Set file ownership/mode portably in provisioning code (Ohai introduced this
  fact so recipes could write `group: node['root_group']` instead of hard-coding
  `root`).
- Detect non-standard hosts in compliance sweeps (anything other than the
  expected `root`/`wheel` for the platform is worth flagging).
- Mirror the Ohai contract for consumers migrating recipes to gohai.

## Collected Fields

| Field  | Type   | Description                                       | OCSF mapping                                                                                                                            |
| ------ | ------ | ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `name` | string | Primary group name for root (`root`, `wheel`, …). | No direct OCSF equivalent. OCSF's `group` object targets access-control events, not host-level facts. Treated as a gohai-native scalar. |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |
| Other    | —         |

## Example Output

### Linux

```json
{
  "root_group": {
    "name": "root"
  }
}
```

### macOS

```json
{
  "root_group": {
    "name": "wheel"
  }
}
```

## SDK Usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("root_group"))
facts, _ := g.Collect(context.Background())

fmt.Println(facts.RootGroup.Name)
```

## Enable/Disable

```bash
gohai --collector.root_group      # enable (default)
gohai --no-collector.root_group   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                         | Ohai plugin ([`root_group.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/root_group.rb)) | Alignment                                                                             |
| -------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- |
| Linux    | `os/user.Lookup("root")` → `LookupGroupId(user.Gid)` | `Etc.getgrgid(Etc.getpwnam("root").gid).name` — same two-hop pattern.                                  | **Identical.** Two-hop lookup ensures correctness even if root's primary GID isn't 0. |
| macOS    | Same Go stdlib calls                                 | Same Ruby `Etc` call                                                                                   | **Identical.** Typical result "wheel".                                                |

**Known gaps:** None. Windows support (Ohai has a WMI SID-544 lookup for
localized Administrators group name) is out of scope — gohai doesn't target
Windows yet.

## Backing library

- Go stdlib (`os/user`) — no third-party dependency.
