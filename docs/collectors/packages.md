# Packages

> **Status:** Implemented ✅

## Description

The collector reports the list of installed packages on the host. On Linux, it
reads the native package database: dpkg on Debian and Ubuntu, rpm on RHEL and
Fedora. On macOS, it queries Homebrew. If the package manager is missing, the
collector returns an empty list. Containers and minimal base images often have
no package manager.

## Collected fields

| Field                        | Type     | Description                                                                                                                                 | Schema mapping                 |
| ---------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `packages`                   | `list`   | The list of installed package objects.                                                                                                      | No direct schema mapping.      |
| `packages[].name`            | `string` | The package name.                                                                                                                           | OCSF `package.name`            |
| `packages[].version`         | `string` | The installed version. For brew, the newest installed version.                                                                              | OCSF `package.version`         |
| `packages[].versions`        | `list`   | Each installed version, for brew, which keeps more than one. It is absent for dpkg and rpm.                                                 | No direct schema mapping       |
| `packages[].architecture`    | `string` | The CPU architecture. It is empty for brew packages.                                                                                        | OCSF `package.architecture`    |
| `packages[].package_manager` | `string` | The package manager: `dpkg`, `rpm` or `brew`.                                                                                               | OCSF `package.package_manager` |
| `packages[].kind`            | `string` | For brew, `formula` or `cask`. A formula and a cask can have the same name, and this field tells them apart. It is absent for dpkg and rpm. | No direct schema mapping       |
| `packages[].description`     | `string` | The one-line summary of the package.                                                                                                        | OCSF `package.description`     |
| `packages[].size`            | `int64`  | The installed size, in bytes.                                                                                                               | No direct schema mapping       |

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
   `dpkg-query -W -f='${Package}\t${Version}\t${Architecture}\t${db:Status-Status}\t${binary:Summary}\t${Installed-Size}\n'`.
2. It parses the tab-delimited output, one package on each line.
3. It skips lines where `db:Status-Status` is not `installed`. This is the same
   as the debian branch of Ohai's `packages.rb`, which also filters on the
   install status.
4. It skips malformed lines (fewer than 4 fields) and entries with an empty
   name.

**On Linux (the `rhel`, `fedora`, `amazon`, `suse` and `mandriva` families):**

1. The collector runs
   `rpm -qa --qf '%{NAME}\t%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\t%{ARCH}\t%{SUMMARY}\t%{SIZE}\n'`.
2. It parses the tab-delimited output, one package on each line.
3. It skips malformed lines and entries with an empty name.
4. The version string has the release component (`5.1-6.el9`), and the epoch
   first when the package has one (`1:3.2.4-3.fc42`), as rpm and dnf print it.

**On macOS:**

1. The collector finds brew at `/opt/homebrew/bin/brew` (Apple silicon) or
   `/usr/local/bin/brew` (Intel). If neither exists, it runs `brew` from the
   PATH. A launchd daemon, such as the osapi agent, has neither path on its
   PATH, so the collector does not depend on the PATH.
2. brew refuses to run as root. When `id -u` prints `0`, the collector reads the
   owner of the brew executable with `stat -L -f %u <brew>`. It then runs each
   brew command as `sudo -u '#<uid>' -H <brew> ...`, as Ohai runs brew as the
   brew owner. When the owner is root, or `stat` fails, it runs brew as it is. A
   user that is not root runs `<brew> ...`.
3. The collector runs two commands: `<brew> list --formula --versions` and
   `<brew> list --cask --versions`. The formulae come first in the list, then
   the casks. A single `brew list --versions` stops at the first cask that brew
   refuses to load, such as a cask from a tap that is not trusted, and exits
   with status 1. With two commands, a refused cask does not hide the formulae.
   When one command fails, the collector keeps the packages of the other and
   returns the error, so the status is `partial`. When both fail, there is no
   fact, and the status is `failed`. Each package from the first command has the
   kind `formula`, and each package from the second has the kind `cask`.
4. The output format is `<name> <ver1> [<ver2> ...]`. Homebrew can have many
   versions of a package installed at the same time, and does not promise their
   order. `versions` lists each of them, and `version` is the newest. The
   collector compares two versions as Homebrew does in outline. Each version
   splits into runs of digits and runs of letters. Digit runs compare as
   numbers, and a digit run is newer than a letter run. When one version is a
   prefix of the other, the longer one is newer, unless its next run is letters,
   which marks a pre-release such as `1.2.0rc1`.
5. If Homebrew is not on the host, the collector returns an empty list, not an
   error.

**Errors:** If the command is not on the host, the collector returns an empty
package list and no error. If the command is there and fails, for example with a
non-zero exit or a timeout, the collector returns the error. Ohai's
`packages.rb` continues after both, but then a failure looks the same as a host
with no package manager.

`description` and `size` come from the same run: `${binary:Summary}` and
`${Installed-Size}` (KiB) of `dpkg-query`, or `%{SUMMARY}` and `%{SIZE}` (bytes)
of `rpm`. The family of the collector's sources picks the database: `debian`
reads dpkg; `rhel`, `fedora`, `amazon`, `suse` and `mandriva` read rpm. A family
with neither database, such as `arch` or `alpine`, gives no fact, with the
status `unsupported`: this collector reads no package list there. A derivative
that names its parent in `ID_LIKE`, such as elementary or zorin, has its
parent's family.

## Backing library

`source.Runner` to run the commands.
