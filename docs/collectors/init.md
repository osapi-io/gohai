# Init

> **Status:** Implemented ✅

## Description

Detects which init system (PID 1) the host is running under. Matters when
consumers decide how to restart services, reload configuration, or declare
dependency ordering:

- `systemd`. `systemctl` family, unit files under `/etc/systemd/system/`
- `upstart`. `initctl`, `/etc/init/*.conf`
- `sysvinit`. Classic `/etc/init.d/*` scripts + runlevels
- `openrc`. Gentoo / Alpine; `rc-service`, `/etc/init.d/` + OpenRC conf
- `runit`. Void Linux, Artix; `sv` command
- `launchd`. MacOS; `launchctl`, `*.plist` under `/Library/LaunchDaemons/`

## Collected fields

| Field  | Type   | Description                                                                               | Schema mapping                                                                                                        |
| ------ | ------ | ----------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `name` | string | Canonical init system name. Known values listed above; unknown values pass through as-is. | No direct schema mapping, `process.name` of PID 1 is the closest concept (`os.pid_1_name`-style field doesn't exist). |

## Platform support

| Platform | Supported                 |
| -------- | ------------------------- |
| Linux    | ✅ (reads `/proc/1/comm`) |
| macOS    | ✅ (hard-coded `launchd`) |

On Linux, if `/proc/1/comm` is unreadable (restricted containers), the `name` is
empty. Consumers should treat empty as unknown, not as an assertion of any
particular init system.

## Example output

### systemd host

```json
{ "init": { "name": "systemd" } }
```

### macOS

```json
{ "init": { "name": "launchd" } }
```

### Alpine Linux

```json
{ "init": { "name": "openrc" } }
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("init"))
facts, _ := g.Collect(context.Background())
switch facts.Init.Name {
case "systemd":
    // use systemctl
case "launchd":
    // use launchctl
}
```

## Enable/disable

```bash
gohai --collector.init      # enable (default)
gohai --no-collector.init   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. Read `/proc/1/comm` via the injected `avfs.VFS` and trim whitespace. Apply a
   small normalization table: `init` → `sysvinit`, `openrc-init` → `openrc`.
   Other values pass through unchanged. Mirrors Ohai's `init_package.rb`, which
   reads the same file; we add the canonicalization step that Ohai skips.

On macOS we hard-code `launchd`, it's the only init system Darwin ships. Ohai's
`init_package.rb` is Linux-only and returns nothing here.

## Backing library

- Go stdlib (`os`), no third-party dependency.
