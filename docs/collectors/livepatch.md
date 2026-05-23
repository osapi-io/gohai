# Livepatch

> **Status:** Implemented ✅

## Description

Reports the status of kernel livepatch modules loaded on a Linux host. Kernel
livepatching allows security patches to be applied to a running kernel without a
reboot. Each entry under `/sys/kernel/livepatch/` represents one loaded patch
module.

On macOS the collector returns nil — kernel livepatching is a Linux-only feature
(requires `CONFIG_LIVEPATCH=y` in the kernel build).

Consumers use this to:

- Verify that a critical CVE patch has been applied and is in `enabled` state
  (not mid-transition) across a fleet.
- Detect hosts where a livepatch is transitioning (`transition: true`) — the
  kernel is still patching live tasks and the patch is not yet fully active.
- Enumerate all loaded livepatch modules for audit trails or change tracking.

## Signals

- `patches` — map of patch module names to their status. A **nil** value means
  the `/sys/kernel/livepatch` directory does not exist — the kernel was compiled
  without livepatch support or no livepatches have ever been loaded. An **empty
  map** means the directory exists but no patches are currently loaded.
- `patches[name].enabled` — `true` when the patch is active. A loaded-but-not-
  enabled patch is unusual and may indicate a failed or rolled-back application.
- `patches[name].transition` — `true` when the kernel is mid-transition, meaning
  it is still patching live tasks. The patch is not fully effective until
  `transition` becomes `false`.

## Collected Fields

| Field                      | Type               | Description                                                        | Schema mapping                                                  |
| -------------------------- | ------------------ | ------------------------------------------------------------------ | --------------------------------------------------------------- |
| `patches`                  | `map[string]Patch` | Map of patch module name → status. Nil if livepatch not present.   | No direct OCSF or OTel mapping. gohai convention: `patches`.    |
| `patches[name].enabled`    | `bool`             | `true` if the patch is active (sysfs `enabled == "1"`).            | No direct OCSF or OTel mapping. gohai convention: `enabled`.    |
| `patches[name].transition` | `bool`             | `true` if the patch is mid-transition (sysfs `transition == "1"`). | No direct OCSF or OTel mapping. gohai convention: `transition`. |

## Platform Support

| Platform | Supported                                                   |
| -------- | ----------------------------------------------------------- |
| Linux    | ✅ (requires `CONFIG_LIVEPATCH=y`)                          |
| macOS    | Returns nil — kernel livepatching is not available on macOS |

## Example Output

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

```json
{
  "livepatch": {
    "patches": null
  }
}
```

### Livepatch supported but no patches loaded

```json
{
  "livepatch": {
    "patches": {}
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

g, _ := gohai.New(gohai.WithCollectors("livepatch"))
facts, _ := g.Collect(context.Background())

if facts.Livepatch != nil && facts.Livepatch.Patches != nil {
    for name, p := range facts.Livepatch.Patches {
        fmt.Printf("%s: enabled=%v transition=%v\n", name, p.Enabled, p.Transition)
    }
}
```

## Enable/Disable

```bash
gohai --collector.livepatch       # enable (opt-in)
gohai --no-collector.livepatch    # disable
```

This collector is opt-in (`DefaultEnabled: false`) because livepatching is a
specialized Linux feature not present on most hosts.

## Dependencies

None.

## Data Sources

On Linux:

1. Attempt to read the directory listing of `/sys/kernel/livepatch/` via the
   injected `avfs.VFS`. If the directory does not exist (kernel compiled without
   `CONFIG_LIVEPATCH`, or kernel older than 4.0), return `{patches: nil}` — the
   nil value signals "no livepatch support" to consumers, distinct from an empty
   map which means "supported but none loaded".
2. For each directory entry under `/sys/kernel/livepatch/`:
   - Skip non-directory entries (sysfs may add regular files at the top level).
   - Read `<patch>/enabled` — `"1"` → `enabled: true`; `"0"` or absent →
     `false`.
   - Read `<patch>/transition` — `"1"` → `transition: true`; `"0"` or absent →
     `false`.
3. Return the populated map.

Ohai's `linux/livepatch.rb` uses the same sysfs approach — `dir_glob` over
`/sys/kernel/livepatch/*` and reads `enabled` and `transition` per entry.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) (`osfs` in production,
  `memfs` in tests) for `/sys/kernel/livepatch/` reads.
