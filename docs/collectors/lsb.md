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

| Field         | Type   | Description                                        | OCSF mapping                             |
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

| Platform | What we read       | Ohai plugin                                                                                                   | Alignment  |
| -------- | ------------------ | ------------------------------------------------------------------------------------------------------------- | ---------- |
| Linux    | `/etc/lsb-release` | [`linux/lsb.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/linux/lsb.rb) — parses the same file | Equivalent |
| macOS    | —                  | No Ohai handler                                                                                               | Parity     |

**Known gaps:** None. Ohai also runs `lsb_release -a` as a fallback when the
file is absent; we don't (the file exists on every host that has the `lsb`
package installed, and adding an exec path for a legacy format isn't worth the
complexity).

## Backing library

- Go stdlib (`os`, `bufio`) — no third-party dependency.
