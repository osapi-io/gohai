# Kernel

> **Status:** Implemented ✅

## Description

The collector reports the kernel identity: the name, release, version, machine,
processor and high-level OS. On macOS, the collector also reports if the process
runs under Rosetta 2 translation. If it does, `machine` shows the real hardware
architecture (`arm64`), not the translated architecture (`x86_64`).

In 2026-04, the list of loaded modules moved to the related
[`kernel_modules`](kernel_modules.md) collector. The reason is that the module
list runs `kextstat -k -l` on macOS (about 280ms of wall time). On Linux, it
reads `/sys/module/<name>/version` for each module. Most consumers do not need
these. Because `kernel` has only the identity, it stays in the default set and
returns in less than a millisecond. A caller that needs the module list enables
`kernel_modules` separately, or with `--category=system`.

Consumers use this data to:

- Identify the exact kernel for CVE and patch correlation. The release is linked
  to the version of the source package.
- Tell kernel architectures apart in more detail than `runtime.GOARCH`, for
  example `aarch64`, `arm64` and `armv7l`.
- Find processes that Rosetta translates on Apple Silicon. Thus consumers that
  depend on the architecture can correct for it.

## Collected fields

| Field                | Type   | Description                                                         | Schema mapping                                                          |
| -------------------- | ------ | ------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `name`               | string | `uname -s` (`"Linux"`, `"Darwin"`).                                 | `os.name`.                                                              |
| `release`            | string | `uname -r`, the kernel release (`"5.15.0-47-generic"`).             | OCSF `os.kernel_release`. The leaf is removed as CONTRIBUTING.md tells. |
| `version`            | string | `uname -v`, the build string.                                       | No direct schema mapping.                                               |
| `machine`            | string | `uname -m`, the hardware architecture (`"x86_64"`, `"arm64"`).      | Nearest: `device.hw_info.cpu_bits`. OCSF keeps only the bits.           |
| `processor`          | string | Made like `uname -p`: the same as `machine` on Linux and Darwin.    | No direct schema mapping.                                               |
| `os`                 | string | Made like `uname -o`: `"GNU/Linux"` on Linux, `"Darwin"` on Darwin. | `os.type`.                                                              |
| `rosetta_translated` | bool   | macOS only. True if the process runs under Rosetta 2.               | No direct schema mapping.                                               |

The separate [`kernel_modules`](kernel_modules.md) collector reports the loaded
modules.

## Platform support

| Platform | Supported                                     |
| -------- | --------------------------------------------- |
| Linux    | ✅ (uname syscall)                            |
| macOS    | ✅ (uname syscall and `sysctl` Rosetta probe) |

## Example output

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

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.kernel      # enable (default)
gohai collect --no-collector.kernel   # disable
```

## Dependencies

None.

## Data sources

Ohai's `kernel.rb` runs `uname -s`, `uname -r`, `uname -v`, `uname -m` and
`uname -p` one at a time. gohai calls `unix.Uname()`, the Go syscall wrapper. It
returns all fields in one call. The data is the same, and there is no subprocess
overhead. Both give `name`, `release`, `version` and `machine`. gohai adds
`processor` and `os`, which are equivalent to `uname -p` and `uname -o`. On
macOS, gohai also adds `rosetta_translated`. This is a gohai extension, and Ohai
has no equivalent.

On Linux:

1. `unix.Uname()` gives `name`, `release`, `version` and `machine`.
2. `processor` is a copy of `machine`. `os` is always the string `"GNU/Linux"`.
   Ohai runs `uname -p` and `uname -o` to get the same values. gohai does not do
   the extra fork and exec, because the syscall already has all the data (Option
   A of issue #29).

On macOS:

1. `unix.Uname()` gives `name`, `release`, `version` and `machine`.
2. `processor` is a copy of `machine`. `os` is always the string `"Darwin"`.
3. The collector runs `sysctl -n hw.optional.x86_64` through the shared
   `internal/executor` runner. It does this only if uname reports
   `machine == "x86_64"`. If the output without whitespace is `"1"`, the process
   runs under Rosetta 2. Then the collector sets `machine` and `processor` to
   `arm64`, the real hardware, and sets `rosetta_translated = true`. On native
   Apple Silicon, the collector does not run this sysctl. Thus, in the usual
   case, the collector takes less than a millisecond.

The list of loaded modules (`/proc/modules`, `/sys/module/<n>/version`,
`kextstat`) is in [`kernel_modules`](kernel_modules.md).

## Backing library

- [`golang.org/x/sys/unix`](https://pkg.go.dev/golang.org/x/sys/unix). The
  `Uname` syscall for the top-level identity fields.
- [`internal/executor`](../../internal/executor). The shared command runner. The
  collector uses it to run `sysctl` for the Rosetta probe on macOS. Tests mock
  it with `go.uber.org/mock`.
