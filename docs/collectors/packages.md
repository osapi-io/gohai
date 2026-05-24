# packages

## Description

Reports the list of installed packages on the host. On Linux the collector reads
from the native package database (dpkg on Debian/Ubuntu, rpm on RHEL/Fedora). On
macOS it queries Homebrew. Returns an empty list when the package manager is
absent (containers, minimal base images).

## Collected Fields

| Field                | Type   | Schema mapping              | Notes                                     |
| -------------------- | ------ | --------------------------- | ----------------------------------------- |
| `packages`           | list   | —                           | List of installed package objects         |
| `packages[].name`    | string | OCSF `package.name`         | Package name                              |
| `packages[].version` | string | OCSF `package.version`      | Installed version                         |
| `packages[].architecture`    | string | OCSF `package.architecture`    | CPU architecture; empty for brew packages |
| `packages[].package_manager` | string | gohai convention               | Package manager: `dpkg`, `rpm`, or `brew` |

## Platform Support

| Platform     | Supported | Backing source           |
| ------------ | --------- | ------------------------ |
| Linux/Debian | Yes       | `dpkg-query -W -f='...'` |
| Linux/RHEL   | Yes       | `rpm -qa --qf '...'`     |
| macOS        | Yes       | `brew list --versions`   |

## Example Output

```json
{
  "packages": [
    {
      "name": "bash",
      "version": "5.1-6",
      "architecture": "amd64",
      "package_manager": "dpkg"
    },
    {
      "name": "libc6",
      "version": "2.35-0ubuntu3.6",
      "architecture": "amd64",
      "package_manager": "dpkg"
    }
  ]
}
```

## SDK Usage

```go
g := gohai.New(gohai.WithEnabled("packages"))
facts, _ := g.Collect(ctx)
```

## Enable/Disable

Default: **disabled** (opt-in). Full package inventory is expensive on hosts
with thousands of packages.

```bash
gohai --collector.packages        # enable
gohai --no-collector.packages     # disable
```

## Dependencies

None.

## Data Sources

The collector dispatches on the platform detected by `platform.Detect()`:

**On Linux (Debian/Ubuntu family):**

1. Runs
   `dpkg-query -W -f='${Package}\t${Version}\t${Architecture}\t${db:Status-Status}\n'`.
2. Parses tab-delimited output, one package per line.
3. Skips lines where `db:Status-Status` is not `installed` — mirrors Ohai's
   debian branch in `packages.rb` which also filters on install status.
4. Skips malformed lines (fewer than 4 fields) and empty-name entries.

**On Linux (RHEL/Fedora/SUSE and all non-debian distros):**

1. Runs `rpm -qa --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n'`.
2. Parses tab-delimited output, one package per line.
3. Skips malformed lines and empty-name entries.
4. Version string includes the release component (`5.1-6.el9`) as rpm reports
   it.

**On macOS:**

1. Runs `brew list --versions`.
2. Output format: `<name> <ver1> [<ver2> ...]` — multiple versions can be
   installed simultaneously. The collector records the last version token (most
   recently installed).
3. Returns an empty list when Homebrew is not on PATH — not an error.

**Error handling:** Any exec failure (command not found, exit error) returns an
empty package list rather than propagating an error. Matches Ohai's
`packages.rb` resilient stance.

## Backing Library

`internal/executor.Executor` for command execution.
