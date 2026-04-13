# Kernel

> **Status:** Implemented ✅

## Description

Reports kernel identification (POSIX `uname` fields) plus, on Linux, the map of
currently-loaded kernel modules.

Field semantics follow POSIX `uname(1)` — the same vocabulary Ohai uses
internally and OCSF's `os.kernel_release` names. On Linux this is what
`uname -s/-r/-v/-m` print; the `modules` map mirrors `lsmod`.

Consumers use this to:

- Identify exact kernel for CVE / patch correlation (release ties to the source
  package version).
- Audit loaded modules (security compliance — is `cramfs` loaded? is `kvm`
  present? is an unsigned third-party module inserted?).
- Distinguish kernel architecture at a finer grain than `runtime.GOARCH` (e.g.
  `aarch64` vs `arm64` vs `armv7l`).

## Collected Fields

| Field     | Type            | Description                                                      | Schema mapping                                              |
| --------- | --------------- | ---------------------------------------------------------------- | ----------------------------------------------------------- |
| `name`    | string          | `uname -s` (`"Linux"`, `"Darwin"`).                              | `os.name`.                                                  |
| `release` | string          | `uname -r` — kernel release (`"5.15.0-47-generic"`).             | `os.kernel_release`.                                        |
| `version` | string          | `uname -v` — build string (`"#51-Ubuntu SMP Thu Aug 11 ..."`).   | No direct OCSF.                                             |
| `machine` | string          | `uname -m` — hardware name (`"x86_64"`, `"aarch64"`, `"arm64"`). | Nearest: `device.hw_info.cpu_bits` (OCSF only stores bits). |
| `modules` | map[name]Module | Loaded kernel modules (Linux only).                              | No OCSF equivalent.                                         |

### `Module` entry

| Field      | Type   | Description                                                                   |
| ---------- | ------ | ----------------------------------------------------------------------------- |
| `size`     | uint64 | Memory footprint in bytes.                                                    |
| `refcount` | int    | Number of holders / dependent modules.                                        |
| `version`  | string | Module version (from `/sys/module/<m>/version` — not populated yet; planned). |

## Platform Support

| Platform | Supported                                                           |
| -------- | ------------------------------------------------------------------- |
| Linux    | ✅ (uname + `/proc/modules` parse)                                  |
| macOS    | ✅ (uname only — no modules map; Apple deprecated kexts in Big Sur) |

## Example Output

### Linux

```json
{
  "kernel": {
    "name": "Linux",
    "release": "5.15.0-47-generic",
    "version": "#51-Ubuntu SMP Thu Aug 11 07:51:15 UTC 2022",
    "machine": "x86_64",
    "modules": {
      "nf_tables": { "size": 217088, "refcount": 25 },
      "ipv6": { "size": 557056, "refcount": 24 }
    }
  }
}
```

### macOS

```json
{
  "kernel": {
    "name": "Darwin",
    "release": "23.4.0",
    "version": "Darwin Kernel Version 23.4.0: Wed Feb 21 21:44:31 PST 2024",
    "machine": "arm64"
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
```

## Enable/Disable

```bash
gohai --collector.kernel      # enable (default)
gohai --no-collector.kernel   # disable
```

## Dependencies

None.

## Data Sources

| Platform | What we read                                                   | Ohai plugin                                                                                                          | Alignment                                                                                                                                                                                       |
| -------- | -------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Linux    | `golang.org/x/sys/unix.Uname` syscall + `/proc/modules` parse. | [`kernel.rb`](https://github.com/chef/ohai/blob/main/lib/ohai/plugins/kernel.rb) — `uname -s/-r/-v/-m/-p` + `lsmod`. | **Same source of truth.** Ohai shells out to `uname`/`lsmod`; we call the syscall and parse `/proc/modules` directly (same data without a subprocess). Module-version sub-field is a follow-up. |
| macOS    | `golang.org/x/sys/unix.Uname` syscall.                         | Same `kernel.rb` plus `kextstat` for modules.                                                                        | **Same uname data.** We skip the module map on macOS — Apple deprecated kexts in Big Sur and replaced with system extensions that don't expose a comparable enumeration.                        |

**Known gaps vs. Ohai:**

- `modules.<name>.version` — Ohai reads `/sys/module/<m>/version` per module; we
  don't yet. Planned.
- `processor` (`uname -p`) and `os` (`uname -o` → `GNU/Linux`) — Ohai surfaces
  these; we don't. Low-value redundancy with `machine` + `name` respectively;
  deferred.

## Backing library

- Go stdlib +
  [`golang.org/x/sys/unix`](https://pkg.go.dev/golang.org/x/sys/unix) — no
  third-party runtime dependency beyond the sys package.
