# Kernel

> **Status:** Implemented ✅

## Description

Reports kernel identity (name, release, version, machine, processor, high-level
OS). On macOS the collector also reports whether the process is running under
Rosetta 2 translation; when it is, `machine` reflects the true hardware
architecture (`arm64`) rather than the translated one (`x86_64`).

Loaded-module enumeration was split out into the sibling
[`kernel_modules`](kernel_modules.md) collector in 2026-04. Rationale:
enumerating modules shells out to `kextstat -k -l` on macOS (~280ms wall time)
and walks `/sys/module/<name>/version` on Linux — neither of which most
consumers need. Keeping `kernel` identity-only lets it stay in the default set
and return in under a millisecond; callers who need the module list opt into
`kernel_modules` separately (or via `--category=system`).

Consumers use this to:

- Identify the exact kernel for CVE / patch correlation (release ties to the
  source package version).
- Distinguish kernel architecture at a finer grain than `runtime.GOARCH` (e.g.
  `aarch64` vs `arm64` vs `armv7l`).
- Detect Rosetta-translated processes on Apple Silicon so architecture-sensitive
  consumers can compensate.

## Collected Fields

| Field                | Type   | Description                                                         | Schema mapping                                              |
| -------------------- | ------ | ------------------------------------------------------------------- | ----------------------------------------------------------- |
| `name`               | string | `uname -s` (`"Linux"`, `"Darwin"`).                                 | `os.name`.                                                  |
| `release`            | string | `uname -r` — kernel release (`"5.15.0-47-generic"`).                | OCSF `os.kernel_release` (leaf stripped per CLAUDE.md).     |
| `version`            | string | `uname -v` — build string.                                          | No direct schema mapping.                                   |
| `machine`            | string | `uname -m` — hardware arch (`"x86_64"`, `"arm64"`).                 | Nearest: `device.hw_info.cpu_bits` (OCSF only stores bits). |
| `processor`          | string | `uname -p` synthesis: same as `machine` on Linux and Darwin.        | No direct schema mapping.                                   |
| `os`                 | string | `uname -o` synthesis: `"GNU/Linux"` on Linux, `"Darwin"` on Darwin. | `os.type`.                                                  |
| `rosetta_translated` | bool   | macOS only. True when the process is running under Rosetta 2.       | No direct schema mapping.                                   |

Loaded modules are reported by the separate
[`kernel_modules`](kernel_modules.md) collector.

## Platform Support

| Platform | Supported                                   |
| -------- | ------------------------------------------- |
| Linux    | ✅ (uname syscall)                          |
| macOS    | ✅ (uname syscall + `sysctl` Rosetta probe) |

## Example Output

### Linux

```json
{
  "kernel": {
    "name": "Linux",
    "release": "5.15.0-47-generic",
    "version": "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
    "machine": "x86_64",
    "processor": "x86_64",
    "os": "GNU/Linux"
  }
}
```

### macOS (native arm64 Apple Silicon)

```json
{
  "kernel": {
    "name": "Darwin",
    "release": "23.4.0",
    "version": "Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
    "machine": "arm64",
    "processor": "arm64",
    "os": "Darwin"
  }
}
```

### macOS (Rosetta-translated)

```json
{
  "kernel": {
    "name": "Darwin",
    "release": "22.6.0",
    "machine": "arm64",
    "processor": "arm64",
    "os": "Darwin",
    "rosetta_translated": true
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("kernel"))
facts, _ := g.Collect(context.Background())

k := facts.Kernel
fmt.Printf("%s %s (%s)\n", k.Name, k.Release, k.Machine)

// Architecture-sensitive code path on Apple Silicon.
if k.RosettaTranslated {
    log.Println("running under Rosetta 2")
}

// Module list lives on its own collector — opt in when needed.
// gohai.New(gohai.WithCollectors("kernel", "kernel_modules"))
// if mods := facts.KernelModules; mods != nil { ... }
```

## Enable/Disable

```bash
gohai --collector.kernel      # enable (default)
gohai --no-collector.kernel   # disable
```

## Dependencies

None.

## Data Sources

On Linux:

1. `unix.Uname()` provides `name`, `release`, `version`, `machine`.
2. `processor` is synthesized as a copy of `machine`; `os` is the static string
   `"GNU/Linux"`. Ohai shells out to `uname -p` / `uname -o` to produce the same
   values — we skip the extra fork/exec because the syscall already has
   everything needed (Option A of issue #29).

On macOS:

1. `unix.Uname()` provides `name`, `release`, `version`, `machine`.
2. `processor` is synthesized as a copy of `machine`; `os` is the static string
   `"Darwin"`.
3. `sysctl -n hw.optional.x86_64` is run through the shared `internal/executor`
   runner — only when uname reports `machine == "x86_64"`. When the trimmed
   output is `"1"`, the process is running under Rosetta 2 — we overwrite
   `machine` / `processor` to `arm64` (the real hardware) and set
   `rosetta_translated = true`. On native Apple Silicon this sysctl is skipped
   entirely, keeping the collector under a millisecond in the common case.

Loaded-module enumeration (`/proc/modules`, `/sys/module/<n>/version`,
`kextstat`) lives in [`kernel_modules`](kernel_modules.md).

## Backing library

- [`golang.org/x/sys/unix`](https://pkg.go.dev/golang.org/x/sys/unix) — `Uname`
  syscall for the top-level identity fields.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `sysctl` for the Rosetta probe on macOS. Tests mock
  it with `go.uber.org/mock`.
