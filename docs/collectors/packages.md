# Packages

> **Status:** Implemented ✅

## Description

The collector reports the list of installed packages on the host. On Linux, it
reads the native package database: dpkg on Debian and Ubuntu, rpm on RHEL and
Fedora. On macOS, it queries Homebrew. If the package manager is missing, the
collector returns an empty list. Containers and minimal base images often have
no package manager.

## Collected fields

| Field                        | Type     | Description                                          | Schema mapping                 |
| ---------------------------- | -------- | ---------------------------------------------------- | ------------------------------ |
| `packages`                   | `list`   | The list of installed package objects.               | No direct schema mapping.      |
| `packages[].name`            | `string` | The package name.                                    | OCSF `package.name`            |
| `packages[].version`         | `string` | The installed version.                               | OCSF `package.version`         |
| `packages[].architecture`    | `string` | The CPU architecture. It is empty for brew packages. | OCSF `package.architecture`    |
| `packages[].package_manager` | `string` | The package manager: `dpkg`, `rpm` or `brew`.        | OCSF `package.package_manager` |
| `packages[].description`     | `string` | The one-line summary of the package.                 | OCSF `package.description`     |
| `packages[].size`            | `int64`  | The installed size, in bytes.                        | No direct schema mapping       |

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |

## Example output

```json
{
  "packages": {
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
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("packages"))
facts, _ := g.Collect(ctx)
```

## Enable/disable

Default: **disabled** (opt-in). A full package inventory is slow on hosts with
thousands of packages.

```bash
gohai collect --collector.packages        # enable
gohai collect --no-collector.packages     # disable
```

## Dependencies

None.

## Data sources

The collector selects its method from the platform that `platform.Detect()`
finds:

**On Linux (Debian/Ubuntu family):**

1. The collector runs
   `dpkg-query -W -f='${Package}\t${Version}\t${Architecture}\t${db:Status-Status}\n'`.
2. It parses the tab-delimited output, one package on each line.
3. It skips lines where `db:Status-Status` is not `installed`. This is the same
   as the debian branch of Ohai's `packages.rb`, which also filters on the
   install status.
4. It skips malformed lines (fewer than 4 fields) and entries with an empty
   name.

**On Linux (RHEL/Fedora/SUSE and all non-debian distros):**

1. The collector runs
   `rpm -qa --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n'`.
2. It parses the tab-delimited output, one package on each line.
3. It skips malformed lines and entries with an empty name.
4. The version string has the release component (`5.1-6.el9`), as rpm reports
   it.

**On macOS:**

1. The collector runs `brew list --versions`.
2. The output format is `<name> <ver1> [<ver2> ...]`. Homebrew can have many
   versions of a package installed at the same time. The collector records the
   last version token, which is the most recently installed version.
3. If Homebrew is not on PATH, the collector returns an empty list, not an
   error.

**Errors:** If a command fails (command not found, exit error), the collector
returns an empty package list and no error. This is the same as Ohai's
`packages.rb`, which also continues after errors.

`description` and `size` come from the same run: `${binary:Summary}` and
`${Installed-Size}` (KiB) of `dpkg-query`, or `%{SUMMARY}` and `%{SIZE}` (bytes)
of `rpm`. The family of the collector's sources picks the database: `debian`
reads dpkg, and the other Linux families read rpm.

## Backing library

`internal/executor.Executor` to run the commands.
