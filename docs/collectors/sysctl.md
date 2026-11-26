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
Executor. The only difference is the separator. gohai tries `: ` before `=`, so
that it parses macOS values that contain `=` correctly, for example
`vm.swapusage`.

On Linux and macOS, the collector runs `sysctl -a` through the shared
`source.Runner`:

1. The collector looks for a separator in each output line. It tries `": "`
   first. This is the native macOS format, and it is also used when values
   contain `" = "`. Then it tries `" = "`, the standard Linux format. It skips
   lines that have neither separator.
2. The collector removes the whitespace around the key. It skips lines with an
   empty key.
3. The collector keeps the key and the value unchanged in the `Params` map.
4. When `sysctl` is missing or returns an error, the collector returns an empty
   `Params` map and no error. This is the same as Ohai, which does not panic.
   Ohai's `linux/sysctl.rb` fills the mash only when `exitstatus == 0`.

**Why the collector tries `": "` first:** macOS values such as `vm.swapusage`
contain `" = "` in the value, for example
`vm.swapusage: total = 1024.00M used = ...`. When the collector tries `": "`
first, it splits at the colon after the key name, and the full value stays
complete. On Linux, sysctl keys never contain `": "`. Thus the `": "` check
finds no match, and the collector correctly uses `" = "`.

## Backing library

- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `sysctl -a`. Tests mock it with `go.uber.org/mock`.
