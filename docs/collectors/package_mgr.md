# Package manager

> **Status:** Implemented ✅

## Description

The collector finds the active package manager that the host uses to install
software. Ohai's `packages` plugin selects `dpkg-query`, `rpm -qa` or
`pacman -Qi` from the `platform_family`, but keeps that decision inside the
plugin. gohai reports the decision as an explicit fact.

Consumers use this data to:

- Select the correct install, query or update command without a second detection
  of the distribution.
- Run remediation tools that must run `apt-get install X` on one host and
  `dnf install X` on a different host.
- Find hosts in a managed fleet that do not have the expected manager, for
  example a "Debian-family" host without apt.

## Collected fields

| Field  | Type   | Description                                                                                                                                                                     | Schema mapping                                                                                          |
| ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `name` | string | The canonical manager name: `apt`, `apt-get`, `dnf`, `yum`, `zypper`, `pacman`, `apk`, `xbps-install`, `emerge`, `brew`, `port`. It is empty if the collector finds no manager. | No direct schema mapping. OCSF has no canonical field for the tool that manages the software inventory. |
| `path` | string | The absolute path to the manager binary, for example `/usr/bin/apt`. It is empty if the collector finds no manager.                                                             | No direct schema mapping.                                                                               |

## Platform support

| Platform                                 | Supported                                                              |
| ---------------------------------------- | ---------------------------------------------------------------------- |
| Linux (Debian family)                    | ✅ (`apt` first, then `apt-get`)                                       |
| Linux (RHEL family)                      | ✅ (`dnf` first, then `yum`)                                           |
| Linux (SUSE, Arch, Alpine, Void, Gentoo) | ✅ (the collector looks for zypper, pacman, apk, xbps-install, emerge) |
| macOS                                    | ✅ (`brew` first, then `port`)                                         |

The platform family decides which managers the collector looks for. The family
comes from the caller's sources. If the sources have no family,
`family.Detect()` reads `/etc/os-release`. It maps `ubuntu`, `debian` and
`raspbian` to `debian`. It maps `rhel`, `redhat`, `centos`, `fedora`, `rocky`,
`alma`, `amazon` and `oracle` to `rhel`.

## Example output

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

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.package_mgr      # enable (default)
gohai collect --no-collector.package_mgr   # disable
```

## Dependencies

None at the registry level. The collector uses `platform.Detect()` directly,
through `internal/platform`. It does not read the output of the `platform`
collector. Thus you can enable or disable the two collectors independently.

## Data sources

| Platform | What we read                                                                                                                             | Ohai plugin                                                                                                                     | Alignment                                                                                                                                                                                                                         |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `exec.LookPath` for the manager of the family: `apt` on debian, `dnf` or `yum` on rhel, `zypper`, `pacman` and others on other families. | [`packages.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/packages.rb) uses the same family selection internally. | **Equivalent selection, different output.** Ohai keeps the selection internal and reports the inventory of installed packages. gohai reports the selection as a typed fact. Consumers then make their own install or query calls. |
| macOS    | `exec.LookPath brew`, then `port`.                                                                                                       | Ohai has no detection of macOS package managers.                                                                                | **Extension.** Ohai does not look for macOS package managers. gohai finds brew and MacPorts.                                                                                                                                      |

**Known gaps:** Windows (Chocolatey, winget, Scoop) is out of scope. gohai does
not support Windows yet.

## Backing library

- The Go standard library (`os/exec`, `exec.LookPath`). There is no third-party
  dependency.
