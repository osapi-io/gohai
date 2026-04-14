# Kernel Modules

> **Status:** Implemented ✅

## Description

Enumerates loaded kernel modules (Linux) and legacy kernel extensions (macOS).
Module entries carry size, reference count, and version so downstream tooling
can correlate loaded modules against CVE feeds or policy baselines.

Split from the [`kernel`](kernel.md) collector in 2026-04. Rationale: module
enumeration is expensive — `kextstat -k -l` takes ~280ms on macOS, and reading
`/sys/module/<name>/version` for every loaded module walks a fair chunk of the
`/sys` hierarchy on Linux. Most consumers only want kernel identity (name,
release, machine) and shouldn't pay that cost by default. This collector is
therefore opt-in; enable it when you actually need the module list.

Consumers use this to:

- Audit loaded modules/kexts for security compliance — is `cramfs` loaded? is an
  unsigned third-party module inserted? which EDR kext is active?
- Correlate loaded modules against CVE feeds (module version ↔ known CVE).
- Verify EDR / anti-malware agents are loaded (their kexts or modules should
  always be present).

## Collected Fields

| Field     | Type            | Description                    | Schema mapping            |
| --------- | --------------- | ------------------------------ | ------------------------- |
| `modules` | map[name]Module | Loaded kernel modules / kexts. | No direct schema mapping. |

### `Module` entry

| Field      | Type   | Description                                                                                     |
| ---------- | ------ | ----------------------------------------------------------------------------------------------- |
| `size`     | uint64 | Memory footprint in bytes (Linux: `/proc/modules`; macOS: `kextstat` size column as hex).       |
| `refcount` | int    | Number of holders / dependent modules.                                                          |
| `version`  | string | Linux: `/sys/module/<name>/version` when present. macOS: parenthesized version from `kextstat`. |

On macOS, `modules` enumerates **legacy kernel extensions only** — System
Extensions introduced in macOS 11+ live under `/Library/SystemExtensions/` and
require `systemextensionsctl`, which is not yet queried.

## Platform Support

| Platform | Supported                                        |
| -------- | ------------------------------------------------ |
| Linux    | ✅ (`/proc/modules` + `/sys/module/<n>/version`) |
| macOS    | ✅ (`kextstat -k -l`)                            |

## Example Output

### Linux

```json
{
  "kernel_modules": {
    "modules": {
      "nf_tables": { "size": 217088, "refcount": 25, "version": "1.2.3" },
      "ipv6": { "size": 557056, "refcount": 24 }
    }
  }
}
```

### macOS

```json
{
  "kernel_modules": {
    "modules": {
      "com.apple.iokit.IOPCIFamily": {
        "size": 2216,
        "refcount": 0,
        "version": "2.9"
      }
    }
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("kernel", "kernel_modules"))
facts, _ := g.Collect(context.Background())

mods := facts.KernelModules
if mods == nil { return }

// Security check: is an unsigned module loaded?
if _, ok := mods.Modules["suspicious_mod"]; ok {
    log.Println("unexpected kernel module present")
}
```

## Enable/Disable

```bash
gohai --collector.kernel_modules      # enable (opt-in)
gohai --no-collector.kernel_modules   # disable (default)
gohai --category=system                # enabling the system category also pulls this in
```

## Dependencies

None.

## Data Sources

On Linux:

1. `/proc/modules` is read through the injected `avfs.VFS` and parsed
   line-by-line for per-module `name`, `size`, `refcount`. A missing file yields
   an empty Info with no error (systems without `procfs` mounted).
2. For each module, `/sys/module/<name>/version` is read (when present) and
   trimmed into `modules[].version`. A missing file leaves the field empty —
   many built-in modules do not expose a version (matches Ohai's silent-on-miss
   behaviour).

On macOS:

1. `kextstat -k -l` is run through the shared `internal/executor` runner. Each
   row's fixed columns (`Index Refs Address Size Wired Name (Version)`) are
   parsed into a `Module` with `name`, `version` (parens stripped), `size`
   (hex), and `refcount`. When `kextstat` is absent or returns non-zero the
   modules map is left empty — there is no `/proc/modules` equivalent on macOS.
2. System Extensions (macOS 11+) are out of scope; a future enhancement may
   consult `systemextensionsctl list`.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) — virtual filesystem
  used to read `/proc/modules` and `/sys/module/<name>/version` on Linux. Tests
  inject `memfs` with canned content.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `kextstat` on macOS. Tests mock it with
  `go.uber.org/mock`.
