# OS Release

> **Status:** Implemented ✅

## Description

Parses `/etc/os-release` — the standardized freedesktop.org manifest every
modern Linux distribution ships to describe itself. This is the authoritative,
machine-readable source for distro identity, version, codename, and
upstream-compatibility hints.

Consumers use this to:

- Identify exactly what Linux distribution + version a host is running (e.g.
  `"ubuntu 24.04 (noble)"`), more reliably than parsing `/etc/issue` or
  `uname -v`.
- Pick a compatibility ancestor for distros that derive from a parent
  (`ID_LIKE=debian` for Ubuntu; `ID_LIKE=rhel` for Amazon Linux).
- Drive Chef/Ansible/Puppet-style distro logic without invoking an external
  tool.

macOS has no equivalent file; the collector returns `nil` there.

## Collected Fields

| Field              | Type              | Description                                              | Schema mapping |
| ------------------ | ----------------- | -------------------------------------------------------- | ------------------------------------------------------------------- |
| `id`               | string            | Short identifier (`ubuntu`, `rhel`, `debian`, `alpine`). | `os.name` (OCSF uses `name` for the distro identifier).             |
| `id_like`          | []string          | Parent/upstream distros this host is compatible with.    | No direct OCSF field; downstream-compatible with `os.name` aliases. |
| `name`             | string            | Display name (`"Ubuntu"`).                               | No direct OCSF — `os.name` is already `id`.                         |
| `pretty_name`      | string            | Human-readable (`"Ubuntu 24.04 LTS"`).                   | No OCSF equivalent (presentation-level).                            |
| `version`          | string            | Full version string including codename.                  | No OCSF — `os.version` is used for the version number.              |
| `version_id`       | string            | Version number only (`"24.04"`).                         | OCSF `os.version`.                                                  |
| `version_codename` | string            | Codename only (`"noble"`).                               | No OCSF equivalent.                                                 |
| `build_id`         | string            | Build identifier (rolling distros).                      | OCSF `os.build` (closest match).                                    |
| `variant`          | string            | Distro variant (edge, workstation, server, cloud).       | No OCSF equivalent.                                                 |
| `variant_id`       | string            | Variant identifier (short form).                         | No OCSF equivalent.                                                 |
| `home_url`         | string            | Distro homepage.                                         | No OCSF.                                                            |
| `support_url`      | string            | Distro support URL.                                      | No OCSF.                                                            |
| `bug_report_url`   | string            | Where to file bugs upstream.                             | No OCSF.                                                            |
| `extra`            | map[string]string | Any keys not explicitly parsed (e.g. `UBUNTU_CODENAME`). | No OCSF.                                                            |

## Platform Support

| Platform | Supported                  |
| -------- | -------------------------- |
| Linux    | ✅                         |
| macOS    | `nil` (no equivalent file) |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("os_release"))
facts, _ := g.Collect(context.Background())
if r := facts.OSRelease; r != nil && r.ID == "ubuntu" {
    fmt.Println("Running Ubuntu", r.VersionID)
}
```

## Enable/Disable

```bash
gohai --collector.os_release      # enable (default)
gohai --no-collector.os_release   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read      | Ohai plugin                                                                                                                                    | Alignment                                                                                                                      |
| -------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Linux    | `/etc/os-release` | [`linux/os_release.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/os_release.rb) — parses the same file into `os_release`. | **Equivalent**: same file, same every-key extraction. We keep known keys as typed fields and bucket unknown keys into `extra`. |
| macOS    | —                 | No Ohai handler.                                                                                                                               | Parity — we return nil.                                                                                                        |

**Known gaps:** None. Ohai additionally emits some legacy aliases (e.g. `lsb.id`
for backward compat); those belong to our `lsb` collector, not this one.

## Backing library

- Go stdlib (`os`, `bufio`) — no third-party dependency.
