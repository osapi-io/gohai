# languages

## Description

Reports installed programming language runtimes on the host. Each supported
language is probed by running its standard `--version` command. A nil field
means the runtime was not found on PATH. The set of detected runtimes mirrors
Ohai's languages plugin (which delegates to per-language sub-plugins; gohai
consolidates them into a single collector).

## Collected Fields

| Field    | Type          | Schema mapping                 | Notes                                 |
| -------- | ------------- | ------------------------------ | ------------------------------------- |
| `go`     | string / null | OTel `process.runtime.version` | Go toolchain version, e.g. `1.21.0`   |
| `python` | string / null | OTel `process.runtime.version` | `python3 --version` output            |
| `ruby`   | string / null | OTel `process.runtime.version` | `ruby --version` output               |
| `node`   | string / null | OTel `process.runtime.version` | `node --version` output, `v` stripped |
| `java`   | string / null | OTel `process.runtime.version` | Version extracted from quoted string  |
| `perl`   | string / null | OTel `process.runtime.version` | `(vX.Y.Z)` form extracted             |

Fields are omitted from JSON output when nil (runtime absent).

## Platform Support

| Platform | Supported | Backing source                                             |
| -------- | --------- | ---------------------------------------------------------- |
| Linux    | Yes       | `go version`, `python3 --version`, `ruby --version`, etc.  |
| macOS    | Yes       | Same commands — version flags are cross-platform identical |

## Example Output

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

## SDK Usage

```go
g := gohai.New(gohai.WithEnabled("languages"))
facts, _ := g.Collect(ctx)
```

## Enable/Disable

Default: **disabled** (opt-in). Runtime detection requires shelling out six
times, which is significant overhead.

```bash
gohai --collector.languages       # enable
gohai --no-collector.languages    # disable
```

## Dependencies

None.

## Data Sources

Both Linux and Darwin use identical probe sequences — version flags are the same
across platforms:

1. **Go:** `go version` — extracts the `goX.Y.Z` token and strips the `go`
   prefix.
2. **Python:** `python3 --version` — extracts the second whitespace-delimited
   field (`Python 3.11.4` → `3.11.4`).
3. **Ruby:** `ruby --version` — extracts the second whitespace-delimited field
   (`ruby 3.2.2 (...)` → `3.2.2`).
4. **Node:** `node --version` — strips the leading `v` from the output
   (`v20.1.0` → `20.1.0`).
5. **Java:** `java -version` — Java writes to stderr; the executor returns
   combined output. Extracts the version from the first quoted string
   (`"21.0.1"`).
6. **Perl:** `perl --version` — extracts the `(vX.Y.Z)` parenthesised form from
   the first line.

Any probe that fails (command not found, non-zero exit) sets the corresponding
field to nil. The collector never returns an error — missing runtimes are the
normal case on minimal hosts.

## Backing Library

`internal/executor.Executor` for command execution.
