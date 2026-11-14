# OS release

> **Status:** Implemented ✅

## Description

The collector parses `/etc/os-release`. This is the standard freedesktop.org
file in which each modern Linux distribution describes itself. It is the
authoritative, machine-readable source for the distribution identity, version,
codename and upstream compatibility.

Consumers use this to:

Consumers use this data to:

- Find the exact Linux distribution and version of a host, for example
  `"ubuntu 24.04 (noble)"`. This is more reliable than parsing `/etc/issue` or
  `uname -v`.
- Select a compatible parent for a distribution that comes from a parent:
  `ID_LIKE=debian` for Ubuntu, `ID_LIKE=rhel` for Amazon Linux.
- Run distribution logic of the Chef, Ansible or Puppet type without an external
  tool.

macOS has no equivalent file. On macOS, the collector returns `nil`.

## Collected fields

| Field              | Type              | Description                                                                                   | Schema mapping                                                              |
| ------------------ | ----------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `id`               | string            | The short identifier: `ubuntu`, `rhel`, `debian`, `alpine`.                                   | `os.name`. OCSF uses `name` for the distribution identifier.                |
| `id_like`          | []string          | The parent or upstream distributions that this host is compatible with.                       | No direct schema mapping. The values are compatible with `os.name` aliases. |
| `name`             | string            | The display name, for example `"Ubuntu"`.                                                     | No direct schema mapping. `os.name` is already `id`.                        |
| `pretty_name`      | string            | The name for people to read, for example `"Ubuntu 24.04 LTS"`.                                | No direct schema mapping. The value is only for display.                    |
| `version`          | string            | The full version string, with the codename.                                                   | No OCSF. `os.version` holds the version number.                             |
| `version_id`       | string            | Only the version number, for example `"24.04"`.                                               | OCSF `os.version`.                                                          |
| `version_codename` | string            | Only the codename, for example `"noble"`.                                                     | No direct schema mapping.                                                   |
| `build_id`         | string            | The build identifier, for rolling distributions.                                              | OCSF `os.build`. This is the nearest match.                                 |
| `variant`          | string            | The distribution variant: edge, workstation, server, cloud.                                   | No direct schema mapping.                                                   |
| `variant_id`       | string            | The short identifier of the variant.                                                          | No direct schema mapping.                                                   |
| `home_url`         | string            | The home page of the distribution.                                                            | No OCSF.                                                                    |
| `support_url`      | string            | The support URL of the distribution.                                                          | No OCSF.                                                                    |
| `bug_report_url`   | string            | The upstream URL for bug reports.                                                             | No OCSF.                                                                    |
| `extra`            | map[string]string | All keys that the collector does not parse into a typed field, for example `UBUNTU_CODENAME`. | No OCSF.                                                                    |

## Platform support

| Platform | Supported                  |
| -------- | -------------------------- |
| Linux    | ✅                         |
| macOS    | `nil` (no equivalent file) |

## Example output

### Ubuntu 24.04

```json
{
  "os_release": {
    "id": "ubuntu",
    "id_like": ["debian"],
    "name": "Ubuntu",
    "pretty_name": "Ubuntu 24.04 LTS",
    "version": "24.04 LTS (Noble Numbat)",
    "version_id": "24.04",
    "version_codename": "noble",
    "home_url": "https://www.ubuntu.com/",
    "support_url": "https://help.ubuntu.com/",
    "bug_report_url": "https://bugs.launchpad.net/ubuntu/",
    "extra": { "UBUNTU_CODENAME": "noble" }
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("os_release"))
facts, _ := g.Collect(context.Background())
if r := facts.OSRelease; r != nil && r.ID == "ubuntu" {
    fmt.Println("Running Ubuntu", r.VersionID)
}
```

## Enable/disable

```bash
gohai --collector.os_release      # enable (default)
gohai --no-collector.os_release   # disable
```

## Dependencies

None.

## Data sources

| Platform | What we read      | Ohai plugin                                                                                                                                  | Alignment                                                                                                                                      |
| -------- | ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `/etc/os-release` | [`linux/os_release.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/os_release.rb) parses the same file into `os_release`. | **Equivalent.** The file is the same, and both read all keys. gohai keeps the known keys as typed fields and puts the unknown keys in `extra`. |
| macOS    | None              | Ohai has no handler.                                                                                                                         | Parity. gohai returns `nil`.                                                                                                                   |

**Known gaps:** None. Ohai also reports some old aliases for backward
compatibility, for example `lsb.id`. In gohai, the `lsb` collector reports those
aliases, not this collector.

## Backing library

- The Go standard library (`os`, `bufio`). There is no third-party dependency.
