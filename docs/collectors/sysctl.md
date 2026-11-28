# Sysctl

> **Status:** Implemented ✅

## Description

The collector reports the full kernel parameter table. It runs `sysctl -a`. It
returns a flat `map[string]string` of all the sysctl key/value pairs. The data
is the same as Ohai's `linux/sysctl` plugin. The collector supports Linux and
Darwin. Both platforms have `sysctl`, and the output formats are compatible.

DefaultEnabled is `false`. The full sysctl table is large, with hundreds to
thousands of entries. Most consumers need only some keys.

## Collected fields

| Field    | Type              | Description                                      | Schema mapping    |
| -------- | ----------------- | ------------------------------------------------ | ----------------- |
| `params` | map[string]string | All the sysctl key/value pairs from `sysctl -a`. | gohai convention. |

## Platform support

| Platform | Supported                         |
| -------- | --------------------------------- |
| Linux    | ✅ (`sysctl -a` through executor) |
| macOS    | ✅ (`sysctl -a` through executor) |

## Example output

### Linux

```json
{
  "sysctl": {
    "params": {
      "kernel.hostname": "myhost",
      "kernel.ostype": "Linux",
      "net.ipv4.ip_forward": "0",
      "vm.swappiness": "60"
    }
  }
}
```

### macOS

```json
{
  "sysctl": {
    "params": {
      "kern.ostype": "Darwin",
      "kern.osrelease": "23.5.0",
      "hw.ncpu": "10"
    }
  }
}
```

### System without `sysctl`

```json
{
  "sysctl": {
    "params": {}
  }
}
```

## SDK usage

```go
g, _ := gohai.New(gohai.WithCollectors("sysctl"))
facts, _ := g.Collect(context.Background())
if s := facts.Sysctl; s != nil {
    fmt.Println(s.Params["vm.swappiness"])
}
```

## Enable/disable

```bash
gohai collect --collector.sysctl    # enable (opt-in)
gohai collect --no-collector.sysctl # disable
```

## Dependencies

None.

## Data sources

Ohai's `linux/sysctl.rb` runs `sysctl -a` and parses the key=value output. gohai
does the same on Linux and macOS. It runs `sysctl -a` through the shared
Executor. The only difference is the separator. Linux prints `key = value` and
macOS prints `key: value`, and each line splits at the first separator of its
own host. Thus a Linux value that contains `: `, such as `dev.cdrom.info`, and a
macOS value that contains `=`, such as `vm.swapusage`, keep their text.

A host with no `sysctl` gives an empty table. When `sysctl -a` exits with a
status that is not zero but prints part of the table, as it does for a user who
cannot read some keys, the collector keeps that part. Any other failure is an
error: a run that prints nothing, and a run that a timeout, a cancel or the
output limit stops, since its output is cut short.

On Linux and macOS, the collector runs `sysctl -a` through the shared
`source.Runner`:

1. The collector splits each output line at the first separator of its host:
   `" = "` on Linux and `": "` on macOS. It skips lines that do not have that
   separator.
2. The collector removes the whitespace around the key and the value. It skips
   lines with an empty key.
3. The collector keeps each key and its value in the `Params` map.
4. A line longer than 1 MiB is an error, not a short table.

Ohai's `linux/sysctl.rb` fills the mash only when `exitstatus == 0`. gohai keeps
the partial table of a run that exits with a status, because a host where gohai
is not root gives the keys that it can read.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `sysctl -a`. Tests mock it with `go.uber.org/mock`.
