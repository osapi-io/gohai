# Package Manager

> **Status:** Implemented ✅

## Description

Identifies the active package manager the host uses for software installation.
Surfaces the dispatch decision Ohai bakes inside its `packages` plugin (where it
picks between `dpkg-query` / `rpm -qa` / `pacman -Qi` based on
`platform_family`) as an explicit fact.

Consumers use this to:

- Pick the right install/query/update command without re-detecting the distro
  themselves.
- Drive remediation tooling that needs to run `apt-get install X` on one host
  and `dnf install X` on another.
- Detect hosts in a managed fleet that don't have the expected manager installed
  (e.g. a "Debian-family" host missing apt).

## Collected Fields

| Field  | Type   | Description                                                                                                                                               | OCSF mapping                                                                                         |
| ------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `name` | string | Canonical manager name: `apt`, `apt-get`, `dnf`, `yum`, `zypper`, `pacman`, `apk`, `xbps-install`, `emerge`, `brew`, `port`. Empty when no manager found. | No OCSF equivalent — managed-software inventory has no canonical OCSF "which tool manages it" field. |
| `path` | string | Absolute path to the manager binary (e.g. `/usr/bin/apt`). Empty when no manager found.                                                                   | No OCSF equivalent.                                                                                  |

## Platform Support

| Platform                                 | Supported                                         |
| ---------------------------------------- | ------------------------------------------------- |
| Linux (Debian family)                    | ✅ (`apt` preferred, `apt-get` fallback)          |
| Linux (RHEL family)                      | ✅ (`dnf` preferred, `yum` fallback)              |
| Linux (SUSE, Arch, Alpine, Void, Gentoo) | ✅ (zypper/pacman/apk/xbps-install/emerge probed) |
| macOS                                    | ✅ (`brew` preferred, `port` fallback)            |

Dispatch is driven by `internal/platform.Detect()`, which wraps gopsutil's host
info and maps `ubuntu`/`debian`/`raspbian` → `debian`,
`rhel`/`redhat`/`centos`/`fedora`/`rocky`/`alma`/`amazon`/ `oracle` → `rhel`.

## Example Output

### Ubuntu

```json
{
  "package_mgr": {
    "name": "apt",
    "path": "/usr/bin/apt"
  }
}
```

### Rocky Linux 9

```json
{
  "package_mgr": {
    "name": "dnf",
    "path": "/usr/bin/dnf"
  }
}
```

### macOS with Homebrew

```json
{
  "package_mgr": {
    "name": "brew",
    "path": "/opt/homebrew/bin/brew"
  }
}
```

### Host with no manager installed

```json
{
  "package_mgr": {}
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("package_mgr"))
facts, _ := g.Collect(context.Background())

switch facts.PackageMgr.Name {
case "apt":
    // exec apt-get install -y X
case "dnf", "yum":
    // exec dnf install -y X
case "brew":
    // exec brew install X
}
```

## Enable/Disable

```bash
gohai --collector.package_mgr      # enable (default)
gohai --no-collector.package_mgr   # disable
```

## Dependencies

None at the registry level. Conceptually driven by `platform.Detect()` but uses
it directly (via `internal/platform`) rather than consuming the `platform`
collector's output — so the two collectors can be enabled/disabled
independently.

## Data Sources

| Platform | What we read                                                                                                              | Ohai plugin                                                                                                                    | Alignment                                                                                                                                                                                              |
| -------- | ------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Linux    | `exec.LookPath` for the family-specific manager (`apt` on debian, `dnf`/`yum` on rhel, `zypper`/`pacman`/etc. on others). | [`packages.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/packages.rb) uses the same family dispatch internally. | **Equivalent dispatch; different output**: Ohai hides the dispatch and emits installed-package inventory. We surface the dispatch as a typed fact so consumers can make their own install/query calls. |
| macOS    | `exec.LookPath brew`, fallback `port`.                                                                                    | Ohai has no macOS package-manager detection.                                                                                   | **Extension**: Ohai doesn't target macOS package managers; we cover brew and MacPorts.                                                                                                                 |

**Known gaps:** Windows (Chocolatey / winget / Scoop) is out of scope — gohai
doesn't target Windows yet.

## Backing library

- Go stdlib (`os/exec` — `exec.LookPath`) — no third-party dependency.
