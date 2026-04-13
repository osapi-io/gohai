# LSB Release

> **Status:** Implemented ✅

## Description

Parses `/etc/lsb-release` — the Linux Standard Base metadata file. Largely
superseded by `/etc/os-release` on modern distros, but Debian/Ubuntu derivatives
still populate it, and many tools (including older Ansible modules and Chef
Infra clients) still read it first.

Use `os_release` as the primary distro-identity source on modern hosts; fall
back to `lsb` on legacy systems or for Debian-tooling compatibility.

## Collected Fields

| Field         | Type   | Description                                        | Schema mapping |
| ------------- | ------ | -------------------------------------------------- | ---------------------------------------- |
| `id`          | string | DISTRIB_ID — distro short name (`"Ubuntu"`).       | `os.name`.                               |
| `release`     | string | DISTRIB_RELEASE — version (`"24.04"`).             | `os.version`.                            |
| `codename`    | string | DISTRIB_CODENAME (`"noble"`).                      | No OCSF equivalent.                      |
| `description` | string | DISTRIB_DESCRIPTION — formatted string for humans. | No OCSF equivalent (presentation-level). |

## Platform Support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | `nil`     |

## Example Output

### Ubuntu

```json
{
  "lsb": {
    "id": "Ubuntu",
    "release": "24.04",
    "codename": "noble",
    "description": "Ubuntu 24.04 LTS"
  }
}
```

### Minimal host without /etc/lsb-release

```json
{
  "lsb": {}
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("lsb"))
facts, _ := g.Collect(context.Background())
if l := facts.LSB; l != nil && l.ID == "Ubuntu" {
    fmt.Println(l.Description)
}
```

## Enable/Disable

```bash
gohai --collector.lsb      # enable (default)
gohai --no-collector.lsb   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read       | Ohai plugin                                                                                                                                                                              | Alignment                                                                                                                                                                                                                                                                                                                                                                                                      |
| -------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `/etc/lsb-release` | [`linux/lsb.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/lsb.rb) — shells out to `lsb_release -a` when `/usr/bin/lsb_release` is present, otherwise emits nothing. | **Same four fields (`id`/`release`/`codename`/`description`), different source.** Ohai invokes the `lsb_release` CLI (requires the `lsb-release` package). We read `/etc/lsb-release` directly — faster, no subprocess, works on Debian/Ubuntu derivatives even without the `lsb-release` package installed. Distros that only populate via the CLI (rare on modern Debian/Ubuntu) would not be covered by us. |
| macOS    | —                  | No Ohai handler                                                                                                                                                                          | Parity                                                                                                                                                                                                                                                                                                                                                                                                         |

**Known gaps vs. Ohai:** Ohai's `lsb_release -a` fallback can surface data on
hosts that have the CLI installed but no `/etc/lsb-release` file (uncommon on
modern Debian/Ubuntu; more common on older RHEL with the `redhat-lsb` package).
We do not shell out — adding an exec path for a legacy format isn't worth the
complexity.

## Backing library

- Go stdlib (`os`, `bufio`) — no third-party dependency.
