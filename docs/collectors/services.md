# Services

> **Status:** Implemented ✅

## Description

The collector reports the states of the systemd services on Linux hosts. It runs
`systemctl list-units --type=service --all --no-pager --plain` and parses the
output. macOS uses launchd, which has a very different service model. On Darwin,
the collector returns `nil` and no error.

Ohai has no direct equivalent of this collector. Ohai has no plugin that lists
services. This collector is a gohai addition.

## Collected fields

| Field                | Type     | Description                                            | Schema mapping              |
| -------------------- | -------- | ------------------------------------------------------ | --------------------------- |
| `services`           | `list`   | The list of service objects                            | No mapping                  |
| `services[].name`    | `string` | The unit name without the `.service` suffix            | No direct OCSF/OTel mapping |
| `services[].state`   | `string` | The systemd SUB state, for example `running` or `dead` | No direct OCSF/OTel mapping |
| `services[].enabled` | `bool`   | `true` when the ACTIVE state is `active`               | No direct OCSF/OTel mapping |
| `services[].type`    | `string` | Kept for future use. It is empty now                   | No direct OCSF/OTel mapping |

## Platform support

| Platform | Supported                 |
| -------- | ------------------------- |
| Linux    | ✅                        |
| macOS    | nil (launchd is not read) |

## Example output

```json
{
  "services": {
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
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("services"))
facts, _ := g.Collect(ctx)
```

## Enable/disable

Default: **disabled** (opt-in). The full list of services can be large, and the
collector needs systemd.

```bash
gohai --collector.services        # enable
gohai --no-collector.services     # disable
```

## Dependencies

None.

## Data sources

Ohai has no equivalent of this collector. Ohai has no plugin that lists systemd
services. This collector is a gohai addition.

On Linux:

1. The collector runs
   `systemctl list-units --type=service --all --no-pager --plain`.
2. It parses the columns, which whitespace separates:
   `UNIT LOAD ACTIVE SUB DESCRIPTION`.
3. It skips the header line (the line with `UNIT`), blank lines, and legend or
   summary lines that do not contain `.service`.
4. It skips lines with fewer than 4 fields.
5. The ACTIVE state `active` → `enabled: true`. Any other ACTIVE value →
   `false`.
6. The collector keeps the SUB state in `state`, for example `running`, `dead`,
   `failed` or `exited`.
7. When systemctl is missing, the collector returns an empty list, not an error.
   Many containers do not have systemd.

On macOS, the collector returns `nil`.

## Backing library

`internal/executor.Executor` to run the command.
