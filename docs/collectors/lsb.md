# LSB Release

> **Status:** Implemented ✅

## Description

Reports Linux Standard Base identification fields — `id`, `release`, `codename`,
`description` — used as a stable cross-distro identity tuple. Legacy compliance
tooling and older packaging still consume LSB, so we surface it even though
`os_release` has largely supplanted it on modern distros.

The `lsb_release` CLI is the sole source — matches current Ohai's linux/lsb
plugin. When the CLI is absent, `Info` stays empty rather than parsing
`/etc/lsb-release`. See Data Sources for why.

## Collected Fields

| Field         | Type   | Description                                              | Schema mapping                                 |
| ------------- | ------ | -------------------------------------------------------- | ---------------------------------------------- |
| `id`          | string | Distributor ID (`Ubuntu`, `RedHatEnterprise`, `CentOS`). | `os.name` (closest).                           |
| `release`     | string | Release version (`22.04`, `9.3`).                        | `os.version`.                                  |
| `codename`    | string | Release codename (`jammy`, `Plow`).                      | No direct schema mapping.                      |
| `description` | string | Human-readable description (`Ubuntu 22.04.3 LTS`).       | No direct schema mapping (presentation-level). |

## Platform Support

| Platform | Supported                          |
| -------- | ---------------------------------- |
| Linux    | ✅ (`lsb_release -a` via executor) |
| macOS    | `nil` (no LSB concept on Darwin)   |

## Example Output

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

On Linux we run `lsb_release -a` through the shared `internal/executor` runner
and parse the four labelled lines it emits (`Distributor ID`, `Release`,
`Codename`, `Description`) into the matching `Info` fields. When the CLI is
absent or errors, `Info` stays empty — not an error. Matches Ohai's no-panic
behaviour.

**Why no `/etc/lsb-release` file fallback?** Our prior implementation parsed the
file directly. We removed that path to align with Ohai, which explicitly dropped
the file fallback in [chef/ohai#1562](https://github.com/chef/ohai/pull/1562)
(2021). The reasoning carries over:

- On modern Debian/Ubuntu the `lsb-release` package ships both the file and the
  CLI; when one is present the other is too, so the fallback is redundant.
- The file fallback was originally added for legacy Debian <7 where only the
  file was installed. That class of host is out of scope.
- Minimized container images that strip the CLI but keep the file are an edge
  case; the correct fix there is to install the `lsb-release` package (or
  consume `os_release` instead, which we already surface in its own collector).

macOS is not covered — no LSB concept on Darwin.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `lsb_release -a` on Linux. Tests mock it with
  `go.uber.org/mock`.
