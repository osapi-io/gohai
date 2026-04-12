# Root Group

> **Status:** Implemented ✅

## Description

Reports the name of the primary group for UID 0 (the root user). Looked up via
`os/user.LookupGroupId("0")`, which consults NSS (typically `/etc/group` plus
any configured directory service).

The fact that matters is the _name_ — different operating systems use different
conventions:

- **Linux:** almost always `root`.
- **macOS:** `wheel`.
- **Unusual or hardened hosts:** could be anything. Some site policies rename or
  split the root-equivalent group.

Consumers use this to:

- Set file ownership and mode bits portably in provisioning code — Ohai
  originally introduced this fact so Chef recipes could write
  `group: node['root_group']` instead of hard-coding `root`.
- Detect non-standard hosts during compliance sweeps (anything other than the
  expected `root`/`wheel` for the platform is worth flagging).
- Mirror the Ohai contract for consumers migrating recipes to gohai.

## Collected Fields

| Field  | Type     | Description                                                 |
| ------ | -------- | ----------------------------------------------------------- |
| `name` | `string` | Name of the primary group for UID 0 (e.g. `root`, `wheel`). |

## Platform Support

| Platform | Source                       | Supported |
| -------- | ---------------------------- | --------- |
| Linux    | `os/user.LookupGroupId("0")` | ✅        |
| macOS    | `os/user.LookupGroupId("0")` | ✅        |
| Other    | —                            | `nil`     |

If GID 0 has no group entry (extremely unusual), `Collect` returns an error and
the field is left `nil` on `Facts`.

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

None — Tier 1 core collector with no upstream collector dependencies.

## Backing library

- Go stdlib (`os/user`) — no third-party dependency.
