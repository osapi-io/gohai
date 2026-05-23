# RPM

> **Status:** Implemented ✅

## Description

Reports RPM macro definitions from `rpm --showrc`. Macros are the RPM build
system's configuration primitives — they control build paths (`%_topdir`,
`%_builddir`), compiler flags (`%optflags`), platform identifiers (`%_arch`),
and many other aspects of how RPM packages are built and installed.

On macOS the collector returns nil — RPM is not natively present on macOS.

Consumers use this to:

- Audit the RPM macro environment on a build host before triggering a package
  build (verify `%_topdir`, `%optflags`, `%_arch` are set correctly).
- Detect macro overrides in `~/.rpmmacros` or `/etc/rpm/macros.d/` that could
  affect package builds.
- Inventory RPM version and platform compatibility metadata for fleet-wide build
  system configuration management.

## Collected Fields

| Field    | Type                | Description                                                             | Schema mapping                                                                              |
| -------- | ------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `macros` | `map[string]string` | RPM macro name → definition. Macro names as reported by `rpm --showrc`. | No direct OCSF or OTel mapping. gohai convention: `macros` (mirrors Ohai's `rpm[:macros]`). |

## Platform Support

| Platform | Supported                                   |
| -------- | ------------------------------------------- |
| Linux    | ✅ (requires `rpm` to be installed)         |
| macOS    | Returns nil — RPM is not available on macOS |

## Example Output

### Typical RHEL/Fedora host

```json
{
  "rpm": {
    "macros": {
      "%_topdir": "/root/rpmbuild",
      "%_builddir": "%{_topdir}/BUILD",
      "%_rpmdir": "%{_topdir}/RPMS",
      "%_sourcedir": "%{_topdir}/SOURCES",
      "%_specdir": "%{_topdir}/SPECS",
      "%_srcrpmdir": "%{_topdir}/SRPMS",
      "%__cc": "gcc\n  -m64 -mtune=generic",
      "%optflags": "-O2 -flto=auto -ffat-lto-objects -fexceptions -g -grecord-gcc-switches -pipe -Wall -Werror=format-security"
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

## SDK Usage

```go
import (
    "context"
    "fmt"

    "github.com/osapi-io/gohai/pkg/gohai"
)

g, _ := gohai.New(gohai.WithCollectors("rpm"))
facts, _ := g.Collect(context.Background())

if facts.RPM != nil {
    fmt.Println("RPM top directory:", facts.RPM.Macros["%_topdir"])
}
```

## Enable/Disable

```bash
gohai --collector.rpm       # enable (opt-in)
gohai --no-collector.rpm    # disable
```

This collector is opt-in (`DefaultEnabled: false`) because it forks a subprocess
and is only relevant on RPM-based Linux distributions.

## Dependencies

None.

## Data Sources

On Linux:

1. Run `rpm --showrc` via the injected `executor.Executor`. If the command fails
   (rpm not installed, permission denied), return `{macros: {}}` with no error —
   this matches Ohai's "skip if rpm not found" behaviour and avoids erroring on
   non-RPM systems (Debian, Alpine, etc.).
2. Locate the two `===...===` marker lines (5 or more `=` signs) in the output.
   The macro definitions live between these two markers.
3. Parse each line in the macro section:
   - Lines starting with `-` introduce a new macro definition. Split the line on
     spaces into at most 3 parts: the `-` prefix, the macro name, and the
     remainder as the initial value. A `-` line with no name field is skipped.
   - Continuation lines (no `-` prefix) append to the current macro's value with
     a newline separator.
4. Store the last accumulated macro when the marker section ends.

Note: Ohai's `rpm.rb` also parses `ARCHITECTURE AND OS`, `RPMRC VALUES`,
`Features supported`, and `Macro path` sections from `rpm --showrc`. gohai
collects only the macros section — the other sections are lower-demand fields.
Add them via a methodology issue if needed.

## Backing library

- `internal/executor` for `rpm --showrc` execution (gomock-backed in tests).
