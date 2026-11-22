# Services

> **Status:** Implemented ✅

## Description

The collector reports the states of the systemd services on Linux hosts. It runs
`systemctl list-units --type=service --all --no-pager --plain` for the state and
the description of each unit, and `systemctl list-unit-files` for the state of
each unit file. `list-units --all` lists only the units that systemd has loaded,
and a unit that is stopped and disabled is usually not loaded. Thus each unit of
`list-unit-files` that `list-units` did not give is in the list too, with the
state `dead` and no description. A template, such as `getty@`, is not a unit
that runs, and an alias is a second name of a unit, so the list has neither.
macOS uses launchd, which has a very different service model. On Darwin, the
collector returns `nil` and no error.

Ohai has no direct equivalent of this collector. Ohai has no plugin that lists
services. This collector is a gohai addition.

## Collected fields

| Field                        | Type     | Description                                                                                                                                  | Schema mapping              |
| ---------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| `services`                   | `list`   | The list of service objects                                                                                                                  | No mapping                  |
| `services[].name`            | `string` | The unit name without the `.service` suffix                                                                                                  | No direct OCSF/OTel mapping |
| `services[].state`           | `string` | The systemd SUB state, for example `running` or `dead`                                                                                       | No direct OCSF/OTel mapping |
| `services[].enabled`         | `bool`   | `true` when the unit file is `enabled` or `enabled-runtime`: the unit starts at boot                                                         | No direct OCSF/OTel mapping |
| `services[].unit_file_state` | `string` | The state of the unit file: `enabled`, `enabled-runtime`, `disabled`, `static`, `masked` and the others. Empty when `list-unit-files` failed | No direct OCSF/OTel mapping |
| `services[].description`     | `string` | The unit's `Description=`, for example `OpenSSH server daemon`. Empty for a unit that systemd has not loaded                                 | No direct OCSF/OTel mapping |
| `services[].type`            | `string` | Kept for future use. It is empty now                                                                                                         | No direct OCSF/OTel mapping |

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
        "enabled": true,
        "unit_file_state": "enabled",
        "description": "OpenSSH server daemon"
      },
      {
        "name": "NetworkManager",
        "state": "dead",
        "enabled": false,
        "unit_file_state": "disabled",
        "description": "Network Manager"
      },
      {
        "name": "console-getty",
        "state": "dead",
        "enabled": false,
        "unit_file_state": "disabled"
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
gohai collect --collector.services        # enable
gohai collect --no-collector.services     # disable
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
5. The collector keeps the SUB state in `state`, for example `running`, `dead`,
   `failed` or `exited`, and the rest of the line in `description`.
6. It runs `systemctl list-unit-files --type=service --no-legend --no-pager`. A
   unit whose state is `enabled` or `enabled-runtime` is `enabled: true`. A unit
   that runs now, but is disabled, is `enabled: false`. When this command fails,
   no unit is enabled.
7. When systemctl is missing, the collector returns an empty list, not an error.
   Many containers do not have systemd.

On macOS, the collector returns `nil`.

## Backing library

`internal/executor.Executor` to run the command.
