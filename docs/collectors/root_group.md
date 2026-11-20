# Root group

> **Status:** Implemented ✅

## Description

The collector reports the name of the primary group of root. It does the lookup
in two steps, the same as Ohai. It finds the primary GID of the user `root`.
Then it finds the group name of that GID. Different operating systems use
different conventions:

- **Linux:** almost always `root`.
- **macOS / BSD:** `wheel`.
- **Custom / hardened hosts:** the value that the site policy sets.

Consumers use this data to:

- Set the file ownership and mode in provisioning code that works on all
  platforms. Ohai added this fact so that recipes can write
  `group: node['root_group']` and not a fixed `root`.
- Find non-standard hosts in compliance scans. A value that is not the expected
  `root` or `wheel` for the platform needs attention.
- Keep the Ohai contract for consumers that move their recipes to gohai.

## Collected fields

| Field  | Type   | Description                                                           | Schema mapping                                                                                                                  |
| ------ | ------ | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `name` | string | The name of the primary group of root, for example `root` or `wheel`. | No direct schema mapping. OCSF's `group` object is for access-control events, not host facts. gohai keeps it as a gohai scalar. |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |
| Other    | No        |

## Example output

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

## SDK usage

```go
import (
    "context"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("root_group"))
facts, _ := g.Collect(context.Background())

fmt.Println(facts.RootGroup.Name)
```

## Enable/disable

```bash
gohai collect --collector.root_group      # enable (default)
gohai collect --no-collector.root_group   # disable
```

## Dependencies

None.

## Data sources

| Platform | What we read                                           | Ohai plugin ([`root_group.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/root_group.rb)) | Alignment                                                                                                |
| -------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| Linux    | `/etc/passwd` gid of `root` → its name in `/etc/group` | `Etc.getgrgid(Etc.getpwnam("root").gid).name`, same two-hop pattern.                                   | **Identical.** The lookup in two steps gives the correct name, also if the primary GID of root is not 0. |
| macOS    | The same two files                                     | The same Ruby `Etc` call                                                                               | **Identical.** The usual result is "wheel".                                                              |

**Known gaps:** None. Windows support is out of scope. On Windows, Ohai does a
WMI SID-544 lookup for the localized name of the Administrators group. gohai
does not support Windows yet.

## Backing library

None. The collector reads `/etc/passwd` and `/etc/group` through its filesystem.
On macOS, the two files have `root` and `wheel`, which Directory Services also
has.
