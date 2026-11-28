# Languages

> **Status:** Implemented ✅

## Description

The collector reports the programming language runtimes that are installed on
the host. For each supported language, the collector runs its standard
`--version` command. A `nil` field shows that the runtime is not on PATH. The
collector finds the same set of runtimes as Ohai's languages plugin. Ohai uses a
sub-plugin for each language. gohai puts them all in one collector.

## Collected fields

| Field    | Type      | Description                                    | Schema mapping                                                             |
| -------- | --------- | ---------------------------------------------- | -------------------------------------------------------------------------- |
| `go`     | `*string` | The Go toolchain version, for example `1.21.0` | No direct OCSF/OTel mapping. This is host inventory, not a process runtime |
| `python` | `*string` | The Python 3 version, from `python3 --version` | No direct OCSF/OTel mapping                                                |
| `ruby`   | `*string` | The Ruby version, from `ruby --version`        | No direct OCSF/OTel mapping                                                |
| `node`   | `*string` | The Node.js version, without the leading `v`   | No direct OCSF/OTel mapping                                                |
| `java`   | `*string` | The Java version, from `java --version`        | No direct OCSF/OTel mapping                                                |
| `perl`   | `*string` | The Perl version, from the `(vX.Y.Z)` form     | No direct OCSF/OTel mapping                                                |

If a field is `nil`, it is not in the JSON output. This shows that the runtime
is missing.

## Platform support

| Platform | Supported |
| -------- | --------- |
| Linux    | ✅        |
| macOS    | ✅        |

## Example output

```json
{
  "languages": {
    "go": "1.21.0",
    "python": "3.11.4",
    "ruby": "3.2.2",
    "node": "20.1.0",
    "java": "21.0.1",
    "perl": "5.36.0"
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("languages"))
facts, _ := g.Collect(ctx)
```

## Enable/disable

Default: **disabled** (opt-in). To find the runtimes, the collector runs six
commands. This is a large overhead.

```bash
gohai collect --collector.languages       # enable
gohai collect --no-collector.languages    # disable
```

## Dependencies

None.

## Data sources

Ohai's `languages.rb` sends the work to a sub-plugin for each language (`go.rb`,
`python.rb`, `ruby.rb`, `nodejs.rb`, `java.rb`, `perl.rb`). Each sub-plugin runs
the same `--version` command. gohai puts all six probes in one collector. The
version parse is the same as in Ohai's parser for each language. gohai handles
the same special cases:

- the version in parentheses of Perl, without its `v`
- the `v` at the start of the Node version

Linux and Darwin use the same probe sequence. The version flags are the same on
the two platforms:

1. **Go.** `go version`. The collector gets the `goX.Y.Z` token and removes the
   `go` prefix.
2. **Python.** `python3 --version`. The collector gets the second field, with
   whitespace as the separator (`Python 3.11.4` → `3.11.4`).
3. **Ruby.** `ruby --version`. The collector gets the second field, with
   whitespace as the separator (`ruby 3.2.2 (...)` → `3.2.2`).
4. **Node.** `node --version`. The collector removes the leading `v` from the
   output (`v20.1.0` → `20.1.0`).
5. **Java.** `java --version`, which writes to stdout. The collector gets the
   second field of the first line (`openjdk 21.0.1 2023-10-17` → `21.0.1`). Java
   8 does not have `--version` and exits non-zero for it. For a `java` that is
   on the PATH and fails `--version`, the collector runs `java -version`, which
   writes to stderr, and reads the version from its `version "..."` line.
6. **Perl.** `perl --version`. The collector gets the `(vX.Y.Z)` form in
   parentheses from the first line, and removes the `v` (`(v5.36.0)` →
   `5.36.0`).

If a runtime's command is not on the PATH, the collector sets its field to
`nil`, because missing runtimes are usual on minimal hosts. A command that is
there and fails, for example with a non-zero exit code or a permission error, is
an error, and so is a cancelled context. An absent source is an empty result,
and a failed read is an error.

## Backing library

`source.Runner` to run the commands.
