# services

## Description

Reports systemd service states on Linux hosts. Runs
`systemctl list-units --type=service --all --no-pager --plain` and parses the
output. macOS uses launchd, which has a substantially different service model;
Darwin returns nil gracefully.

There is no direct Ohai equivalent for this collector — Ohai does not ship a
service-list plugin. This is a gohai-native addition.

## Collected Fields

| Field                | Type    | Schema mapping              | Notes                                      |
| -------------------- | ------- | --------------------------- | ------------------------------------------ |
| `services`           | list    | —                           | List of service objects                    |
| `services[].name`    | string  | OCSF `process.name`         | Unit name with `.service` suffix stripped  |
| `services[].state`   | string  | gohai convention: `state`   | systemd SUB state: `running`, `dead`, etc. |
| `services[].enabled` | boolean | gohai convention: `enabled` | `true` when ACTIVE state is `active`       |
| `services[].type`    | string  | gohai convention: `type`    | Reserved for future use; currently empty   |

## Platform Support

| Platform | Supported | Backing source                                                 |
| -------- | --------- | -------------------------------------------------------------- |
| Linux    | Yes       | `systemctl list-units --type=service --all --no-pager --plain` |
| macOS    | No        | Returns nil (launchd model differs substantially)              |

## Example Output

```json
{
  "services": [
    {
      "name": "ssh",
      "state": "running",
      "enabled": true
    },
    {
      "name": "NetworkManager",
      "state": "dead",
      "enabled": false
    }
  ]
}
```

## SDK Usage

```go
g := gohai.New(gohai.WithEnabled("services"))
facts, _ := g.Collect(ctx)
```

## Enable/Disable

Default: **disabled** (opt-in). Full service inventory can be large and requires
systemd to be present.

```bash
gohai --collector.services        # enable
gohai --no-collector.services     # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. Runs `systemctl list-units --type=service --all --no-pager --plain`.
2. Parses whitespace-delimited columns: `UNIT LOAD ACTIVE SUB DESCRIPTION`.
3. Skips the header line (contains `UNIT`), blank lines, and legend/summary
   lines that do not contain `.service`.
4. Skips lines with fewer than 4 fields.
5. ACTIVE state `active` → `enabled: true`; any other ACTIVE value → `false`.
6. SUB state (e.g. `running`, `dead`, `failed`, `exited`) is stored in `state`.
7. Returns an empty list (not an error) when systemctl is absent — containers
   without systemd are common.

On macOS: returns nil.

## Backing Library

`internal/executor.Executor` for command execution.
