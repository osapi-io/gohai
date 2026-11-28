# LSB release

> **Status:** Implemented ✅

## Description

The collector reports the Linux Standard Base identification fields: `id`,
`release`, `codename` and `description`. Together, these fields are a stable
identity that is the same on all distributions. Old compliance tools and old
packaging still read LSB. Thus gohai reports it, although `os_release` replaces
it on most modern distributions.

The `lsb_release` CLI is the only source. This is the same as the current
linux/lsb plugin of Ohai. If the CLI is missing, `Info` stays empty. The
collector does not parse `/etc/lsb-release`. Data sources tells why.

## Collected fields

| Field         | Type   | Description                                                  | Schema mapping                                 |
| ------------- | ------ | ------------------------------------------------------------ | ---------------------------------------------- |
| `id`          | string | The distributor ID (`Ubuntu`, `RedHatEnterprise`, `CentOS`). | `os.name` (closest).                           |
| `release`     | string | The release version (`22.04`, `9.3`).                        | `os.version`.                                  |
| `codename`    | string | The release codename (`jammy`, `Plow`).                      | No direct schema mapping.                      |
| `description` | string | A description that people can read (`Ubuntu 22.04.3 LTS`).   | No direct schema mapping (presentation-level). |

## Platform support

| Platform | Supported                              |
| -------- | -------------------------------------- |
| Linux    | ✅ (`lsb_release -a` through executor) |
| macOS    | `nil` (Darwin has no LSB)              |

## Example output

### Ubuntu with `lsb-release` package

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

### Minimal host without `lsb_release` CLI

```json
{
  "lsb": {}
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("lsb"))
facts, _ := g.Collect(context.Background())
if l := facts.LSB; l != nil && l.ID == "Ubuntu" {
    fmt.Println(l.Description)
}
```

## Enable/disable

```bash
gohai collect --collector.lsb      # enable (default)
gohai collect --no-collector.lsb   # disable
```

## Dependencies

None.

## Data sources

On Linux, the collector runs `lsb_release -a` through the shared
`source.Runner`. It parses the four labelled lines of the output
(`Distributor ID`, `Release`, `Codename`, `Description`) into the related `Info`
fields. If the CLI is missing, `Info` stays empty, and the collector returns no
error. A run of the CLI that fails is an error.

**Why is there no fallback to the `/etc/lsb-release` file?** The previous
implementation of gohai parsed the file directly. gohai removed that path to
agree with Ohai. Ohai removed its file fallback in
[chef/ohai#1562](https://github.com/chef/ohai/pull/1562) (2021). The same
reasons apply to gohai:

- On modern Debian and Ubuntu, the `lsb-release` package installs both the file
  and the CLI. If one is there, the other is there too. Thus the fallback is not
  necessary.
- Ohai added the file fallback for old Debian \<7, where only the file was
  installed. gohai does not support these hosts.
- Some minimal container images remove the CLI but keep the file. This is an
  edge case. On these images, install the `lsb-release` package. Or read
  `os_release` instead, which gohai reports in its own collector.

The collector does not support macOS, because Darwin has no LSB.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `lsb_release -a` on Linux. Tests mock it with
  `go.uber.org/mock`.
