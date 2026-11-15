# Command

> **Status:** Implemented ✅

## Description

The collector gives the same data as Ohai's `command/ps`. It runs `ps -ef` and
keeps the raw output lines. Ohai's `command.rb` plugin stores the ps command
string. The related `ps.rb` plugin sets that string to `"ps -ef"` on Linux and
macOS. gohai runs the command and stores the output. Thus consumers get the
process list and do not have to run a shell command.

Linux and macOS both support `ps -ef`, with the same POSIX output format.

DefaultEnabled is `false`. The full process table can be large. Most consumers
use the `process` collector, which gives structured data, or do not need the
table.

## Collected fields

| Field | Type     | Description                                       | Schema mapping    |
| ----- | -------- | ------------------------------------------------- | ----------------- |
| `ps`  | []string | The raw lines from `ps -ef`, with the header row. | gohai convention. |

## Platform support

| Platform | Supported                      |
| -------- | ------------------------------ |
| Linux    | ✅ (`ps -ef` through executor) |
| macOS    | ✅ (`ps -ef` through executor) |

## Example output

```json
{
  "command": {
    "ps": [
      "UID        PID  PPID  C STIME TTY          TIME CMD",
      "root         1     0  0 10:00 ?        00:00:01 /sbin/init",
      "root         2     0  0 10:00 ?        00:00:00 [kthreadd]",
      "user      1234     1  0 10:01 pts/0    00:00:00 bash"
    ]
  }
}
```

### System without `ps`

```json
{
  "command": {
    "ps": []
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("command"))
facts, _ := g.Collect(context.Background())
if c := facts.Command; c != nil {
    for _, line := range c.PS {
        fmt.Println(line)
    }
}
```

## Enable/disable

```bash
gohai collect --collector.command    # enable (opt-in)
gohai collect --no-collector.command # disable
```

## Dependencies

None.

## Data sources

On Linux and on macOS, the collector runs `ps -ef` through the shared
`internal/executor` runner:

1. The collector removes the whitespace at the end of each output line.
2. The collector skips blank lines.
3. The collector stores the header row and all process rows as raw strings in
   `PS`.
4. If `ps` is missing or returns an error, the collector returns an empty `PS`
   slice and no error. This is the same as Ohai, which does not panic.

Ohai's `ps.rb` uses `"ps -ef"` on Linux, macOS, AIX and Solaris, and
`"ps -axww"` on BSDs. gohai uses `ps -ef` on Linux and on Darwin. macOS `ps`
supports all of the POSIX `-ef` flags.

## Backing library

- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `ps -ef`. Tests mock it with `go.uber.org/mock`.
