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
| `java`   | `*string` | The Java version, from the quoted string       | No direct OCSF/OTel mapping                                                |
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
    "perl": "v5.36.0"
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
the same edge cases: the stderr output of Java, the version in parentheses of
Perl, and the leading `v` of Node.

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
5. **Java.** `java -version`. Java writes to stderr. The executor returns the
   combined output. The collector gets the version from the first quoted string
   (`"21.0.1"`).
6. **Perl.** `perl --version`. The collector gets the `(vX.Y.Z)` form in
   parentheses from the first line.

If a probe fails, the collector sets its field to `nil`. A probe fails if the
command is not found or gives a non-zero exit code. The collector never returns
an error, because missing runtimes are usual on minimal hosts.

## Backing library

`internal/executor.Executor` to run the commands.
