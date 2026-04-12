# Package Mgr

> **Status:** Planned 🚧

## Description

Identifies the active system package manager (e.g., `apt`, `dnf`, `yum`,
`zypper`, `pacman`, `apk`, `brew`). Consumers like [OSAPI][osapi] use this to
decide how to install/remove/query software on the host.

Separate from the [`packages`](packages.md) collector, which enumerates
**installed** packages. `package_mgr` only reports **which manager to use**.

## Collected Fields

| Field     | Type   | Description                                                            |
| --------- | ------ | ---------------------------------------------------------------------- |
| `name`    | string | Canonical name: `apt`, `dnf`, `yum`, `zypper`, `pacman`, `apk`, `brew` |
| `path`    | string | Absolute path to the binary (e.g., `/usr/bin/dnf`)                     |
| `version` | string | Reported by `<mgr> --version` (optional)                               |

## Detection strategy

Primary manager per platform family, with file-existence check to disambiguate
when multiple coexist (e.g., dnf vs yum on RHEL):

| Family     | Candidates (priority order) |
| ---------- | --------------------------- |
| `debian`   | `apt`                       |
| `rhel`     | `dnf`, `yum`                |
| `fedora`   | `dnf`                       |
| `amazon`   | `dnf`, `yum`                |
| `suse`     | `zypper`                    |
| `arch`     | `pacman`                    |
| `alpine`   | `apk`                       |
| `mac_os_x` | `brew` (if installed)       |

Implementation walks the candidate list, stat()s each binary in common locations
(`/usr/bin`, `/usr/local/bin`, `/opt/homebrew/bin`), and returns the first hit.

## Platform Support

| Platform | Source                         | Supported |
| -------- | ------------------------------ | --------- |
| Linux    | `platform.Family` + PATH probe | Planned   |
| macOS    | `brew` binary probe            | Planned   |

## Example Output

```json
{
  "package_mgr": {
    "name": "apt",
    "path": "/usr/bin/apt",
    "version": "2.7.14"
  }
}
```

## Enable/Disable

```bash
gohai --collector.package_mgr     # enable (default)
gohai --no-collector.package_mgr  # disable
```

## Dependencies

- `platform` — uses `platform.Family` to pick candidate managers.

## Rationale

Added to fill the `FactsRegistration.PackageMgr` gap from [OSAPI][osapi]. Does
not exist as a standalone plugin in Chef Ohai — Ohai consumers derive this from
`platform_family` at runtime. Shipping it as a dedicated gohai collector makes
the result a single typed field consumers can read directly.

[osapi]: https://github.com/osapi-io/osapi
