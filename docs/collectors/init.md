# Init

> **Status:** Implemented ✅

## Description

Detects which init system (PID 1) the host is running under. Matters when
consumers decide how to restart services, reload configuration, or declare
dependency ordering:

- `systemd` — `systemctl` family, unit files under `/etc/systemd/system/`
- `upstart` — `initctl`, `/etc/init/*.conf`
- `sysvinit` — classic `/etc/init.d/*` scripts + runlevels
- `openrc` — Gentoo / Alpine; `rc-service`, `/etc/init.d/` + OpenRC conf
- `runit` — Void Linux, Artix; `sv` command
- `launchd` — macOS; `launchctl`, `*.plist` under `/Library/LaunchDaemons/`

## Collected Fields

| Field  | Type   | Description                                                                               | Schema mapping                                                                                               |
| ------ | ------ | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `name` | string | Canonical init system name. Known values listed above; unknown values pass through as-is. | No direct OCSF — `process.name` of PID 1 is the closest concept (`os.pid_1_name`-style field doesn't exist). |

## Platform Support

| Platform | Supported                 |
| -------- | ------------------------- |
| Linux    | ✅ (reads `/proc/1/comm`) |
| macOS    | ✅ (hard-coded `launchd`) |

On Linux, if `/proc/1/comm` is unreadable (restricted containers), the `name` is
empty — consumers should treat empty as unknown, not as an assertion of any
particular init system.

## Example Output

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

## SDK Usage

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

## Enable/Disable

```bash
gohai --collector.init      # enable (default)
gohai --no-collector.init   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read   | Ohai plugin                                                                                                         | Alignment                                                                                                                                                                                      |
| -------- | -------------- | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `/proc/1/comm` | [`init_package.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/init_package.rb) — reads the same file. | **Equivalent**. We add light normalization: `init` → `sysvinit`, `openrc-init` → `openrc`. Ohai returns the raw value; we return the canonical form and pass unknown values through unchanged. |
| macOS    | —              | No Ohai plugin (`init_package` is linux-only).                                                                      | **Richer than Ohai**: macOS always uses launchd, so we hard-code it rather than returning nothing.                                                                                             |

**Known gaps:** Windows (service control manager) is out of scope — gohai
doesn't target Windows yet.

## Backing library

- Go stdlib (`os`) — no third-party dependency.
