# Livepatch

> **Status:** Implemented ✅

## Description

The collector reports the status of the kernel livepatch modules that are loaded
on a Linux host. With kernel livepatch, you can apply security patches to a
running kernel without a reboot. Each entry under `/sys/kernel/livepatch/` is
one loaded patch module.

On macOS, the collector returns `nil`. Kernel livepatch is a Linux feature only.
It needs `CONFIG_LIVEPATCH=y` in the kernel build.

Consumers use this data to:

- Make sure that a critical CVE patch is applied on all hosts of a fleet. The
  patch must be in the `enabled` state and not in a transition.
- Find hosts where a livepatch is in a transition (`transition: true`). On these
  hosts, the kernel still patches live tasks, and the patch is not fully active.
- List all loaded livepatch modules for audit records or to track changes.

## Signals

- `patches`. A map of patch module names to their status. An **empty map** shows
  that the directory exists, but no patches are loaded now. When the
  `/sys/kernel/livepatch` directory does not exist, the kernel has no livepatch
  support, and the collector gives no `livepatch` fact.
- `patches[name].enabled`. `true` when the patch is active. A patch that is
  loaded but not enabled is unusual. It can show that the patch failed or that
  someone rolled it back.
- `patches[name].transition`. `true` when the kernel is in a transition. Then
  the kernel still patches live tasks. The patch is not fully active until
  `transition` becomes `false`.

## Collected fields

| Field                      | Type               | Description                                                                                                                   | Schema mapping                                                  |
| -------------------------- | ------------------ | ----------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `patches`                  | `map[string]Patch` | A map of patch module name to status. Empty when no patch is loaded. With no livepatch support, there is no `livepatch` fact. | No direct OCSF or OTel mapping. gohai convention: `patches`.    |
| `patches[name].enabled`    | `bool`             | `true` if the patch is active (sysfs `enabled == "1"`).                                                                       | No direct OCSF or OTel mapping. gohai convention: `enabled`.    |
| `patches[name].transition` | `bool`             | `true` if the patch is in a transition (sysfs `transition == "1"`).                                                           | No direct OCSF or OTel mapping. gohai convention: `transition`. |

## Platform support

| Platform | Supported                                                  |
| -------- | ---------------------------------------------------------- |
| Linux    | ✅ (requires `CONFIG_LIVEPATCH=y`)                         |
| macOS    | Returns `nil`. Kernel livepatch is not available on macOS. |

## Example output

### Host with one active patch

```json
{
  "livepatch": {
    "patches": {
      "lp_cve_2023_0001": {
        "enabled": true,
        "transition": false
      }
    }
  }
}
```

### Kernel without livepatch support

The facts have no `livepatch` key.

### Livepatch supported but no patches loaded

```json
{
  "livepatch": {
    "patches": {}
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

g, _ := gohai.New(gohai.WithCollectors("livepatch"))
facts, _ := g.Collect(context.Background())

if facts.Livepatch != nil {
    for name, p := range facts.Livepatch.Patches {
        fmt.Printf("%s: enabled=%v transition=%v\n", name, p.Enabled, p.Transition)
    }
}
```

## Enable/disable

```bash
gohai collect --collector.livepatch       # enable (opt-in)
gohai collect --no-collector.livepatch    # disable
```

This collector is opt-in (`DefaultEnabled: false`). Livepatch is a special Linux
feature, and most hosts do not have it.

## Dependencies

None.

## Data sources

Ohai's `linux/livepatch.rb` uses the same sysfs method. It uses `dir_glob` on
`/sys/kernel/livepatch/*` and reads the `enabled` and `transition` files of each
patch module. gohai uses the same method, with VFS reads.

On Linux:

1. The collector reads the directory list of `/sys/kernel/livepatch/` through
   the injected `avfs.VFS`. The directory does not exist if the kernel build has
   no `CONFIG_LIVEPATCH`, or if the kernel is older than 4.0. Then the collector
   gives no fact, so consumers see that there is no livepatch support. An empty
   map is different. It means that the kernel has support, but no patch is
   loaded.
2. For each directory entry under `/sys/kernel/livepatch/`, the collector does
   these steps:
   - It skips entries that are not directories. Sysfs can add regular files at
     the top level.
   - It reads `<patch>/enabled`. `"1"` gives `enabled: true`. `"0"` or a missing
     file gives `false`.
   - It reads `<patch>/transition`. `"1"` gives `transition: true`. `"0"` or a
     missing file gives `false`.
3. The collector returns the map. A directory or a file that is there and can't
   be read is an error.

Ohai's `linux/livepatch.rb` uses the same sysfs method. It uses `dir_glob` on
`/sys/kernel/livepatch/*` and reads `enabled` and `transition` for each entry.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) for the
  `/sys/kernel/livepatch/` reads. Production uses `osfs`. Tests use `memfs`.
