# Platform

> **Status:** Implemented ✅

## Description

Reports OS / distribution identification — the canonical "what is this host"
fact. Consumers key off `family` for OS-family branching (apt vs dnf vs brew)
and `os` for runtime (`linux` vs `darwin` vs `windows`) decisions without
re-detecting.

The `platform` collector is the foundation other code reads through
`internal/platform.Detect()` to choose per-OS / per-family code paths —
`package_mgr`, `shard`, and the per-OS struct factories in every collector all
rely on this data.

Consumers use this to:

- Branch on OS family (`debian` vs `rhel` vs `arch`) for installer /
  configuration logic.
- Surface fleet composition (`name` + `version` count distributions).
- Gate features on architecture (`amd64` vs `arm64`).

## Collected Fields

| Field           | Type   | Description                                               | Schema mapping                             |
| --------------- | ------ | --------------------------------------------------------- | ------------------------------------------ |
| `os`            | string | `runtime.GOOS` — `"linux"`, `"darwin"`, `"windows"`.      | `os.type`.                                 |
| `name`          | string | Distro / product ID (`"ubuntu"`, `"redhat"`, `"darwin"`). | `os.name`.                                 |
| `version`       | string | Distro version (`"24.04"`, `"14.4.1"`).                   | `os.version`.                              |
| `version_extra` | string | Extra version info (macOS RSR patch level, if present).   | No direct OCSF.                            |
| `family`        | string | Family (`"debian"`, `"rhel"`, `"mac_os_x"`).              | No direct OCSF — input to packaging logic. |
| `architecture`  | string | Hardware arch (`"amd64"`, `"arm64"`).                     | `device.hw_info.cpu_bits` is the nearest.  |
| `build`         | string | Kernel build string (macOS only — `"23F79"` etc.).        | No direct OCSF.                            |

`name` values pass through Ohai's `OS_RELEASE_PLATFORM_REMAP` table so that
common variants canonicalize to their parent distro (`rhel` → `redhat`, `sles` →
`suse`, `ol` → `oracle`). The raw `/etc/os-release` `ID=` is available
unremapped on the `os_release` collector.

## Platform Support

| Platform | Supported                                                          |
| -------- | ------------------------------------------------------------------ |
| Linux    | ✅ (parses `/etc/os-release`; family inferred from `ID`/`ID_LIKE`) |
| macOS    | ✅ (`sw_vers` for product version + `uname` for build)             |

## Example Output

### Linux (Ubuntu 24.04 x86_64)

```json
{
  "platform": {
    "os": "linux",
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "architecture": "amd64"
  }
}
```

### macOS (Sonoma arm64)

```json
{
  "platform": {
    "os": "darwin",
    "name": "darwin",
    "version": "14.4.1",
    "family": "mac_os_x",
    "architecture": "arm64",
    "build": "23E224"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("platform"))
facts, _ := g.Collect(context.Background())

p := facts.Platform
switch p.Family {
case "debian":
    // apt
case "rhel":
    // dnf / yum
case "mac_os_x":
    // brew
}
```

## Enable/Disable

```bash
gohai --collector.platform      # enable (default)
gohai --no-collector.platform   # disable
```

## Dependencies

None. `platform` is itself the foundation other collectors consult via
`internal/platform.Detect()`.

## Data Sources

| Platform | What we read                                                     | Ohai plugin                                                                                                                                        | Alignment                                                                                                                                                                                                                                          |
| -------- | ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `/etc/os-release` parse (gopsutil `host.Info` + our own parser). | [`linux/platform.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/platform.rb) — `/etc/os-release` + `/etc/*-release` fallbacks. | **Same primary source (`os-release`).** We share Ohai's [`OS_RELEASE_PLATFORM_REMAP`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/platform.rb) table so canonical names match. Legacy `/etc/*-release` fallbacks not yet ported. |
| macOS    | `sw_vers -productVersion` + `uname -r`.                          | [`darwin/platform.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/darwin/platform.rb) — `sw_vers` + `system_profiler`.                | **Equivalent on product version.** Ohai adds `system_profiler` hardware metadata, which belongs to the future `hardware` collector.                                                                                                                |

**Known gaps vs. Ohai:**

- No legacy `/etc/redhat-release` / `/etc/lsb-release` fallbacks for ancient
  distros without `os-release` (all supported modern distros have it).
- No `platform_family_group` (Ohai sometimes groups rhel+fedora+amazon as
  "fedora-like") — our `family` stays 1:1 with the canonical ID.

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3 — plus a local `/etc/os-release` parser for `ID_LIKE` / remap logic.
