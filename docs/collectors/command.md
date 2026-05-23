# Command

> **Status:** Implemented ✅

## Description

Provides Ohai `command/ps` parity by running `ps -ef` and capturing its raw
output lines. Ohai's `command.rb` plugin stores the ps command string; the
companion `ps.rb` plugin defines that string as `"ps -ef"` on Linux and macOS.
gohai runs the command and stores the actual output so consumers get the process
listing without needing to shell out themselves.

Both Linux and macOS support `ps -ef` with identical POSIX output format.

DefaultEnabled is `false` — the full process table can be large and most
consumers either use the `process` collector (which provides structured data) or
don't need it at all.

## Collected Fields

| Field | Type     | Description                                       | Schema mapping    |
| ----- | -------- | ------------------------------------------------- | ----------------- |
| `ps`  | []string | Raw lines from `ps -ef` including the header row. | gohai convention. |

## Platform Support

| Platform | Supported                  |
| -------- | -------------------------- |
| Linux    | ✅ (`ps -ef` via executor) |
| macOS    | ✅ (`ps -ef` via executor) |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("command"))
facts, _ := g.Collect(context.Background())
if c := facts.Command; c != nil {
    for _, line := range c.PS {
        fmt.Println(line)
    }
}
```

## Enable/Disable

```bash
gohai --collector.command    # enable (opt-in)
gohai --no-collector.command # disable
```

## Dependencies

None.

## Data Sources

On both Linux and macOS the collector runs `ps -ef` through the shared
`internal/executor` runner:

1. Each output line has trailing whitespace trimmed.
2. Blank lines are skipped.
3. The header row and all process rows are stored as raw strings in `PS`.
4. When `ps` is absent or returns an error, an empty `PS` slice is returned
   without error — matches Ohai's no-panic posture.

Ohai's `ps.rb` uses `"ps -ef"` on Linux, macOS, AIX, and Solaris, and
`"ps -axww"` on BSDs. We use `ps -ef` on both Linux and Darwin — macOS `ps`
fully supports the POSIX `-ef` flag set.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `ps -ef`. Tests mock it with `go.uber.org/mock`.
