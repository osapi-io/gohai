# Systemd Paths

> **Status:** Implemented ✅

## Description

Reports standard systemd directory paths by running `systemd-path` and parsing
its `name: /path` output. Matches Ohai's `linux/systemd_paths` plugin. Darwin
returns `nil` — systemd is Linux-only.

DefaultEnabled is `false` — relevant only on systemd-managed Linux hosts.

## Collected Fields

| Field   | Type              | Description                                                                          | Schema mapping    |
| ------- | ----------------- | ------------------------------------------------------------------------------------ | ----------------- |
| `paths` | map[string]string | Map of systemd path names to absolute directory paths as reported by `systemd-path`. | gohai convention. |

Common keys in `paths` include (but are not limited to):

| Key                          | Typical value                 |
| ---------------------------- | ----------------------------- |
| `systemd`                    | `/usr/lib/systemd`            |
| `systemd-search-system-unit` | `/etc/systemd/system.control` |
| `systemd-system-unit`        | `/etc/systemd/system`         |
| `user-configuration`         | `~/.config`                   |
| `user-runtime`               | `/run/user/<uid>`             |

The exact keys emitted depend on the systemd version installed.

## Platform Support

| Platform | Supported                        |
| -------- | -------------------------------- |
| Linux    | ✅ (`systemd-path` via executor) |
| macOS    | `nil` (systemd is Linux-only)    |

## Example Output

### Linux with systemd

```json
{
  "systemd_paths": {
    "paths": {
      "systemd": "/usr/lib/systemd",
      "systemd-search-system-unit": "/etc/systemd/system.control",
      "systemd-system-unit": "/etc/systemd/system",
      "user-configuration": "/home/user/.config",
      "user-runtime": "/run/user/1000"
    }
  }
}
```

### System without `systemd-path`

```json
{
  "systemd_paths": {
    "paths": {}
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("systemd_paths"))
facts, _ := g.Collect(context.Background())
if sp := facts.SystemdPaths; sp != nil {
    fmt.Println(sp.Paths["systemd-system-unit"])
}
```

## Enable/Disable

```bash
gohai --collector.systemd_paths    # enable (opt-in)
gohai --no-collector.systemd_paths # disable
```

## Dependencies

None.

## Data Sources

On Linux the collector runs `systemd-path` (no arguments) through the shared
`internal/executor` runner and parses its output:

1. Each line is split on the first `": "` separator. Lines without this
   separator are skipped.
2. Lines with an empty key are skipped.
3. Key and path are stored as-is in the `Paths` map.
4. When `systemd-path` is absent or returns an error, an empty `Paths` map is
   returned without error — matches Ohai's no-panic behaviour.

macOS is not covered — `systemd-path` is a Linux systemd utility. Ohai's plugin
is `collect_data(:linux)` only.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `systemd-path` on Linux. Tests mock it with
  `go.uber.org/mock`.
