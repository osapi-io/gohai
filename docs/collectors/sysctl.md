# Sysctl

> **Status:** Implemented ✅

## Description

Collects the complete kernel parameter table by running `sysctl -a`. Returns a
flat `map[string]string` of all sysctl key/value pairs. Matches Ohai's
`linux/sysctl` plugin. Both Linux and Darwin are supported — both platforms ship
`sysctl` with compatible output formats.

DefaultEnabled is `false` — the full sysctl table is large (hundreds to
thousands of entries) and most consumers only need specific keys.

## Collected Fields

| Field    | Type              | Description                                         | Schema mapping    |
| -------- | ----------------- | --------------------------------------------------- | ----------------- |
| `params` | map[string]string | All sysctl key/value pairs returned by `sysctl -a`. | gohai convention. |

## Platform Support

| Platform | Supported                     |
| -------- | ----------------------------- |
| Linux    | ✅ (`sysctl -a` via executor) |
| macOS    | ✅ (`sysctl -a` via executor) |

## Example Output

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

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("sysctl"))
facts, _ := g.Collect(context.Background())
if s := facts.Sysctl; s != nil {
    fmt.Println(s.Params["vm.swappiness"])
}
```

## Enable/Disable

```bash
gohai --collector.sysctl    # enable (opt-in)
gohai --no-collector.sysctl # disable
```

## Dependencies

None.

## Data Sources

Ohai's `linux/sysctl.rb` runs `sysctl -a` and parses key=value output. gohai
follows the same approach on both Linux and macOS — running `sysctl -a` through
the shared Executor. The only difference is separator handling: gohai tries `: `
before `=` so macOS values containing `=` (e.g. `vm.swapusage`) parse correctly.

On both Linux and macOS the collector runs `sysctl -a` through the shared
`internal/executor` runner:

1. Each output line is parsed for a separator — `": "` is tried first (macOS
   native format; also used when values contain `" = "`), then `" = "` (Linux
   standard format). Lines with neither separator are skipped.
2. The key portion is trimmed of whitespace. Lines with an empty key are
   skipped.
3. Key and value are stored as-is in the `Params` map.
4. When `sysctl` is absent or returns an error, an empty `Params` map is
   returned without error — matches Ohai's no-panic stance. Ohai's
   `linux/sysctl.rb` only populates the mash when `exitstatus == 0`.

**Separator order rationale:** macOS values like `vm.swapusage` embed `" = "`
within the value portion (e.g., `vm.swapusage: total = 1024.00M used = ...`).
Trying `": "` first ensures the correct split on the colon after the key name,
leaving the full value intact. On Linux, sysctl keys never contain `": "`, so
the `": "` check returns no match and `" = "` is used correctly.

## Backing library

- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `sysctl -a`. Tests mock it with `go.uber.org/mock`.
