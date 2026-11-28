# RPM

> **Status:** Implemented ✅

## Description

The collector reports the RPM macro definitions from `rpm --showrc`. Macros are
the basic configuration units of the RPM build system. They control the build
paths (`%_topdir`, `%_builddir`), the compiler flags (`%optflags`), the platform
identifiers (`%_arch`), and many other parts of how RPM builds and installs
packages.

On macOS, the collector gives no fact, with the status `unsupported`. macOS does
not have RPM.

Consumers use this data to:

- Audit the RPM macro environment on a build host before a package build. Make
  sure that `%_topdir`, `%optflags` and `%_arch` have the correct values.
- Find macro overrides in `~/.rpmmacros` or `/etc/rpm/macros.d/` that can change
  package builds.
- Compare the macro definitions across the fleet, to manage the build system
  configuration of each host.

## Collected fields

| Field    | Type                | Description                                                                                                    | Schema mapping                                                                                 |
| -------- | ------------------- | -------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `macros` | `map[string]string` | A map from the RPM macro name to its definition. The macro names are the same as in the `rpm --showrc` output. | No direct OCSF or OTel mapping. gohai convention: `macros`, the same as Ohai's `rpm[:macros]`. |

## Platform support

| Platform | Supported                                                        |
| -------- | ---------------------------------------------------------------- |
| Linux    | ✅ (`rpm` must be installed)                                     |
| macOS    | No fact, with the status `unsupported`. macOS does not have RPM. |

## Example output

### Typical RHEL/Fedora host

```json
{
  "rpm": {
    "macros": {
      "_builddir": "%{_topdir}/BUILD",
      "_rpmdir": "%{_topdir}/RPMS",
      "_sourcedir": "%{_topdir}/SOURCES",
      "_target": "aarch64-linux",
      "__spec_install_pre": "%{___build_pre}\n    %{__rm} -rf \"%{buildroot}\"\n%{nil}",
      "optflags": "-O2 -flto=auto -ffat-lto-objects -fexceptions -g -grecord-gcc-switches -pipe -Wall -Werror=format-security"
    }
  }
}
```

### Host without RPM

```json
{
  "rpm": {
    "macros": {}
  }
}
```

## SDK usage

```go
import (
    "context"
    "fmt"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("rpm"))
facts, _ := g.Collect(context.Background())

if facts.RPM != nil {
    fmt.Println("RPM top directory:", facts.RPM.Macros["_topdir"])
}
```

## Enable/disable

```bash
gohai collect --collector.rpm       # enable (opt-in)
gohai collect --no-collector.rpm    # disable
```

This collector is opt-in (`DefaultEnabled: false`). It starts a subprocess, and
it gives data only on Linux distributions that use RPM.

## Dependencies

None.

## Data sources

On Linux:

1. The collector runs `rpm --showrc` through the injected `source.Runner`. If
   rpm is not installed, the collector returns `{macros: {}}` and no error. This
   is the same as Ohai, which skips the plugin if it does not find rpm. Thus
   there is no error on systems without RPM, for example Debian and Alpine. If
   rpm is installed and `rpm --showrc` fails, the collector returns an error.

2. The collector finds the two `===...===` marker lines (5 or more `=` signs) in
   the output. The macro definitions are between these two markers.

3. The collector parses each line in the macro section:

   - A line such as `-13: _builddir<TAB>%{_topdir}/BUILD` starts a new macro
     definition: the level (negative for built-in and file macros, 0 or more for
     macros defined at run time), a colon and a space, the name, a tab, and the
     body. A macro with no tab has an empty body. The name is stored as `rpm`
     prints it, without a leading `%`.
   - Any other line adds to the body of the current macro, after a newline. A
     line before the first macro is dropped.

   This format was checked against `rpm --showrc` on Fedora and openSUSE.

4. When the marker section ends, the collector stores the last macro.

Note: Ohai's `rpm.rb` also parses the `ARCHITECTURE AND OS`, `RPMRC VALUES`,
`Features supported` and `Macro path` sections of `rpm --showrc`. gohai reads
only the macros section, because there is less demand for the other sections. If
you need them, open a methodology issue.

## Backing library

- `source.Runner` to run `rpm --showrc`. Tests use a gomock mock.
