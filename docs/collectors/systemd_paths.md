# Systemd paths

> **Status:** Implemented ✅

## Description

The collector reports the standard systemd directory paths. It runs
`systemd-path` and parses its `name: /path` output. It is the same as Ohai's
`linux/systemd_paths` plugin. Darwin returns `nil` because systemd is only on
Linux.

DefaultEnabled is `false`. The data is only useful on Linux hosts that systemd
manages.

## Collected fields

| Field   | Type              | Description                                                                              | Schema mapping    |
| ------- | ----------------- | ---------------------------------------------------------------------------------------- | ----------------- |
| `paths` | map[string]string | A map of systemd path names to absolute directory paths, as `systemd-path` reports them. | gohai convention. |

Some common keys in `paths` are below. Other keys can also occur.

| Key                          | Typical value                 |
| ---------------------------- | ----------------------------- |
| `systemd`                    | `/usr/lib/systemd`            |
| `systemd-search-system-unit` | `/etc/systemd/system.control` |
| `systemd-system-unit`        | `/etc/systemd/system`         |
| `user-configuration`         | `~/.config`                   |
| `user-runtime`               | `/run/user/<uid>`             |

The keys that the collector reports change with the installed systemd version.

## Platform support

| Platform | Supported                                |
| -------- | ---------------------------------------- |
| Linux    | ✅ (`systemd-path` through the executor) |
| macOS    | `nil` (systemd is only on Linux)         |

## Example output

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

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("systemd_paths"))
facts, _ := g.Collect(context.Background())
if sp := facts.SystemdPaths; sp != nil {
    fmt.Println(sp.Paths["systemd-system-unit"])
}
```

## Enable/disable

```bash
gohai collect --collector.systemd_paths    # enable (opt-in)
gohai collect --no-collector.systemd_paths # disable
```

## Dependencies

None.

## Data sources

On Linux, the collector runs `systemd-path` with no arguments through the shared
`internal/executor` runner. Then it parses the output:

1. The collector splits each line on the first `": "` separator. It skips lines
   that do not have this separator.
2. It skips lines that have an empty key.
3. It stores the key and the path in the `Paths` map with no change.
4. If `systemd-path` is missing or returns an error, the collector returns an
   empty `Paths` map and no error. This is the same as Ohai, which does not
   panic in this condition.

The collector does not support macOS, because `systemd-path` is a Linux systemd
utility. Ohai's plugin is only `collect_data(:linux)`.

## Backing library

- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `systemd-path` on Linux. Tests mock it with
  `go.uber.org/mock`.
