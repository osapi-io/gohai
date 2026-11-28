# Init

> **Status:** Implemented ✅

## Description

The collector finds the init system (PID 1) of the host. Consumers use this data
to decide how to restart services, reload the configuration, or set the order of
dependencies:

- `systemd`. The `systemctl` commands, and unit files under
  `/etc/systemd/system/`.
- `upstart`. `initctl`, and `/etc/init/*.conf`.
- `init`. sysvinit, with classic `/etc/init.d/*` scripts and runlevels, or
  upstart on an old Ubuntu, or busybox. Each runs PID 1 as `init`, so the comm
  does not tell which one it is.
- `openrc`. Gentoo and Alpine. `rc-service`, `/etc/init.d/` and the OpenRC
  configuration.
- `runit`. Void Linux and Artix. The `sv` command.
- `launchd`. macOS. `launchctl`, and `*.plist` files under
  `/Library/LaunchDaemons/`.

## Collected fields

| Field  | Type   | Description                                                                                                                   | Schema mapping                                                                                                  |
| ------ | ------ | ----------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `name` | string | The canonical name of the init system. The list above gives the known values. The collector does not change an unknown value. | No direct schema mapping. The nearest concept is `process.name` of PID 1. No field like `os.pid_1_name` exists. |

## Platform support

| Platform | Supported                 |
| -------- | ------------------------- |
| Linux    | ✅ (reads `/proc/1/comm`) |
| macOS    | ✅ (always `launchd`)     |

On Linux, if the collector cannot read `/proc/1/comm`, `name` is empty. This
occurs in restricted containers. Consumers must read an empty value as unknown.
An empty value does not identify an init system.

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
gohai collect --collector.init      # enable (default)
gohai collect --no-collector.init   # disable
```

## Dependencies

None.

## Data sources

On Linux:

1. The collector reads `/proc/1/comm` through the injected `avfs.VFS` and
   removes the whitespace. Then `openrc-init` becomes `openrc`. Other values do
   not change, `init` included: sysvinit, upstart and busybox all run PID 1 as
   `init`. Ohai's `init_package.rb` reads the same file and does not change its
   value.

On macOS, the value is always `launchd`, because Darwin has no other init
system. Ohai's `init_package.rb` is for Linux only and returns nothing on macOS.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for the file reads,
  through the injected `avfs.VFS`, and the Go standard library.
