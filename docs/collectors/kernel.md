# Kernel

> **Status:** Implemented ✅

## Description

Reports kernel identity (name, release, version, machine, processor, high-level
OS) and the set of currently loaded modules. Module entries carry size,
reference count, and version so downstream tooling can correlate loaded modules
against CVE feeds or policy baselines. On macOS the collector also reports
whether the process is running under Rosetta 2 translation; when it is,
`machine` reflects the true hardware architecture (`arm64`) rather than the
translated one (`x86_64`).

Consumers use this to:

- Identify the exact kernel for CVE / patch correlation (release ties to the
  source package version).
- Audit loaded modules/kexts (security compliance — is `cramfs` loaded? is an
  unsigned third-party module inserted? which EDR kext is active?).
- Distinguish kernel architecture at a finer grain than `runtime.GOARCH` (e.g.
  `aarch64` vs `arm64` vs `armv7l`).
- Detect Rosetta-translated processes on Apple Silicon so architecture-sensitive
  consumers can compensate.

## Collected Fields

| Field                | Type            | Description                                                         | Schema mapping                                              |
| -------------------- | --------------- | ------------------------------------------------------------------- | ----------------------------------------------------------- |
| `name`               | string          | `uname -s` (`"Linux"`, `"Darwin"`).                                 | `os.name`.                                                  |
| `release`            | string          | `uname -r` — kernel release (`"5.15.0-47-generic"`).                | OCSF `os.kernel_release` (leaf stripped per CLAUDE.md).     |
| `version`            | string          | `uname -v` — build string.                                          | No direct OCSF.                                             |
| `machine`            | string          | `uname -m` — hardware arch (`"x86_64"`, `"arm64"`).                 | Nearest: `device.hw_info.cpu_bits` (OCSF only stores bits). |
| `processor`          | string          | `uname -p` synthesis: same as `machine` on Linux and Darwin.        | No direct OCSF.                                             |
| `os`                 | string          | `uname -o` synthesis: `"GNU/Linux"` on Linux, `"Darwin"` on Darwin. | `os.type`.                                                  |
| `rosetta_translated` | bool            | macOS only. True when the process is running under Rosetta 2.       | No direct OCSF.                                             |
| `modules`            | map[name]Module | Loaded kernel modules / kexts.                                      | No OCSF equivalent.                                         |

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

| Platform | Supported                                                        |
| -------- | ---------------------------------------------------------------- |
| Linux    | ✅ (uname syscall + `/proc/modules` + `/sys/module/<n>/version`) |
| macOS    | ✅ (uname syscall + `sysctl` Rosetta probe + `kextstat -k -l`)   |

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
    "os": "GNU/Linux",
    "modules": {
      "nf_tables": { "size": 217088, "refcount": 25, "version": "1.2.3" },
      "ipv6": { "size": 557056, "refcount": 24 }
    }
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
    "os": "Darwin",
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

// Security check: is an unsigned module loaded?
if _, ok := k.Modules["suspicious_mod"]; ok {
    log.Println("unexpected kernel module present")
}

// Architecture-sensitive code path on Apple Silicon.
if k.RosettaTranslated {
    log.Println("running under Rosetta 2")
}
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
3. `/proc/modules` is read through the injected `avfs.VFS` and parsed
   line-by-line for per-module `name`, `size`, `refcount`.
4. For each module, `/sys/module/<name>/version` is read (when present) and
   trimmed into `modules[].version`. A missing file leaves the field empty —
   many built-in modules do not expose a version (matches Ohai's silent-on-miss
   behaviour).

On macOS:

1. `unix.Uname()` provides `name`, `release`, `version`, `machine`.
2. `processor` is synthesized as a copy of `machine`; `os` is the static string
   `"Darwin"`.
3. `sysctl -n hw.optional.x86_64` is run through the shared `internal/executor`
   runner. When the trimmed output is `"1"` AND uname reports
   `machine == "x86_64"`, the process is running under Rosetta 2 — we overwrite
   `machine` / `processor` to `arm64` (the real hardware) and set
   `rosetta_translated = true`.
4. `kextstat -k -l` is run through the same runner. Each row's fixed columns
   (`Index Refs Address Size Wired Name (Version)`) are parsed into a `Module`
   with `name`, `version` (parens stripped), `size` (hex), and `refcount`. When
   `kextstat` is absent or non-zero the modules list is left empty — there is no
   `/proc/modules` equivalent on macOS.
5. System Extensions (macOS 11+) are out of scope; a separate enhancement may
   consult `systemextensionsctl list` in the future.

## Backing library

- [`golang.org/x/sys/unix`](https://pkg.go.dev/golang.org/x/sys/unix) — `Uname`
  syscall for the top-level identity fields.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) — virtual filesystem
  used to read `/proc/modules` and `/sys/module/<name>/version` on Linux. Tests
  inject `memfs` with canned content.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `sysctl` and `kextstat` on macOS. Tests mock it
  with `go.uber.org/mock`.
