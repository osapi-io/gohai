# Platform

> **Status:** Implemented ✅

## Description

Reports OS identification — canonical name, version, family,
architecture, and (macOS) build / RSR patch suffix. The `platform`
collector is the foundation other code reads through
`internal/platform.Detect()` to choose per-OS / per-family code paths
— `package_mgr`, `shard`, and the per-OS struct factories in every
collector all rely on this data.

The collector prefers `/etc/os-release` (systemd standard) and falls
back through legacy release files (`redhat-release`,
`SuSE-release`, `debian_version`, etc.) so pre-systemd and appliance
distributions are still identified. On macOS, `sw_vers` supplements
gopsutil with the build identifier and the Rapid Security Response
patch suffix.

Consumers use this to:

- Branch on OS family (`debian` vs `rhel` vs `arch`) for installer /
  configuration logic.
- Surface fleet composition (`name` + `version` count distributions).
- Gate features on architecture (`amd64` vs `arm64`).
- Detect macOS hosts that have / haven't applied a Rapid Security
  Response patch.

## Collected Fields

| Field           | Type   | Description                                                                              | Schema mapping                            |
| --------------- | ------ | ---------------------------------------------------------------------------------------- | ----------------------------------------- |
| `os`            | string | `runtime.GOOS` — `"linux"`, `"darwin"`, `"windows"`.                                     | `os.type`.                                |
| `name`          | string | Canonical distro / product ID (`"ubuntu"`, `"redhat"`, `"darwin"`).                      | `os.name`.                                |
| `version`       | string | Distro version (`"24.04"`, `"7.9.2009"`, `"14.4.1"`).                                    | `os.version`.                             |
| `version_extra` | string | macOS RSR patch suffix (`"(a)"`). Empty when no RSR is applied.                          | No direct OCSF.                           |
| `family`        | string | Family grouping (`"debian"`, `"rhel"`, `"fedora"`, `"suse"`, `"arch"`).                  | No direct OCSF — input to packaging logic. |
| `architecture`  | string | Hardware arch (`"amd64"`, `"arm64"`).                                                    | `device.hw_info.cpu_bits` is the nearest. |
| `build`         | string | macOS build identifier from `sw_vers` `BuildVersion` (`"23E224"`).                       | `os.build`.                               |

`name` values pass through our `platformIDRemap` table (mirrors
Ohai's `OS_RELEASE_PLATFORM_REMAP`) so common variants canonicalize
to their parent distro: `rhel` → `redhat`, `sles` → `suse`, `ol` →
`oracle`, `archarm` → `arch`, `cumulus-linux` → `cumulus`, `sles_sap`
→ `suse`, etc. The raw `/etc/os-release` `ID=` is available
unremapped via the `os_release` collector.

`family` is filled from a maintained distro → family table covering
every Ohai-known derivative: RHEL family (alma, rocky, alibabalinux,
sangoma, clearos, parallels, virtuozzo, ibm_powerkvm, nexus_centos,
bigip, xenserver, xcp-ng, cloudlinux, scientific,
enterpriseenterprise, oracle, amazon), Debian family (cumulus, kali,
pop, linuxmint, raspbian), Arch (manjaro, antergos), Fedora-based
network OS (arista_eos), and WRLinux variants (nexus, ios_xr).

## Platform Support

| Platform | Supported                                                          |
| -------- | ------------------------------------------------------------------ |
| Linux    | ✅ (gopsutil `/etc/os-release` + redhat-release supplement + 10-file legacy fallback cascade) |
| macOS    | ✅ (gopsutil + `sw_vers` for `BuildVersion` + `ProductVersionExtra`) |

## Example Output

### Linux (Ubuntu 24.04 x86_64)

```json
{
  "platform": {
    "os": "linux",
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "architecture": "amd64"
  }
}
```

### Linux (CentOS 7.9 — minor version supplemented from /etc/redhat-release)

```json
{
  "platform": {
    "os": "linux",
    "name": "centos",
    "version": "7.9.2009",
    "family": "rhel",
    "architecture": "x86_64"
  }
}
```

### macOS (Sonoma arm64 with RSR)

```json
{
  "platform": {
    "os": "darwin",
    "name": "darwin",
    "version": "14.4.1",
    "version_extra": "(a)",
    "family": "Standalone Workstation",
    "architecture": "arm64",
    "build": "23E224"
  }
}
```

## SDK Usage

```go
g, _ := gohai.New(gohai.WithCollectors("platform"))
facts, _ := g.Collect(context.Background())

p := facts.Platform
switch p.Family {
case "debian":
    // apt
case "rhel":
    // dnf / yum
}
if p.OS == "darwin" && p.VersionExtra != "" {
    fmt.Printf("RSR-patched: %s\n", p.VersionExtra)
}
```

## Enable/Disable

```bash
gohai --collector.platform      # enable (default)
gohai --no-collector.platform   # disable
```

## Dependencies

None. `platform` is the foundation other collectors consult via
`internal/platform.Detect()`.

## Data Sources

On Linux the collector cascades through gopsutil + extensions:

1. **gopsutil `host.Info`** reads `/etc/os-release` — the modern
   systemd path. Produces `Platform` (raw distro ID),
   `PlatformVersion`, `PlatformFamily`.
2. **`platformIDRemap`** normalizes the raw ID to its canonical form
   (Ohai's `OS_RELEASE_PLATFORM_REMAP` table). Adds `archarm` → `arch`,
   `cumulus-linux` → `cumulus`, `sles_sap` → `suse` on top of the
   entries Ohai carries.
3. **Minor-version supplement** for RHEL family (`centos`, `rocky`,
   `almalinux`, `redhat`, `rhel`, `scientific`, `oracle`, `amazon`):
   when `VERSION_ID` is major-only (matches `^\d+$`), read
   `/etc/redhat-release` and extract the dotted version via
   `release\s+(\d[\d.]*)`. For `debian` with empty `VERSION_ID`
   (testing/unstable), read `/etc/debian_version` verbatim. Missing
   supplementary files leave the original version unchanged.
4. **Legacy `/etc/*-release` cascade** when gopsutil produces no
   name (pre-systemd / appliance distros). First file that yields a
   name wins:
   - `/etc/redhat-release` → family `rhel` (regex extracts name +
     dotted version).
   - `/etc/SuSE-release` → family `suse` (`VERSION` + `PATCHLEVEL`).
   - `/etc/f5-release` → family `rhel` (F5 BIG-IP).
   - `/etc/system-release` → Amazon / Alma / Fedora-style.
   - `/etc/debian_version` → family `debian`; file contents are the
     version.
   - `/etc/arch-release` → name `arch`, version empty (rolling).
   - `/etc/gentoo-release`, `/etc/slackware-version`,
     `/etc/enterprise-release`, `/etc/exherbo-release` — family
     inferred from filename.
5. **Family fallback** from the maintained distro → family table
   when gopsutil's `PlatformFamily` is empty (long-tail distros
   gopsutil doesn't recognize: alma, rocky, kali, raspbian,
   cloudlinux, etc.).
6. **Architecture** from `runtime.GOARCH`.

On macOS:

1. **gopsutil `host.Info`** provides `Name` / `Version` / `Family`
   plus `KernelVersion` (used as `Build` fallback).
2. **`sw_vers`** is run through the shared `internal/executor`
   runner. Each `Key: Value` line is parsed; we extract:
   - `BuildVersion` → `Build` (overrides gopsutil's KernelVersion as
     the canonical macOS build id).
   - `ProductVersionExtra` → `VersionExtra` (Apple Rapid Security
     Response suffix, e.g. `"(a)"`; absent on most macOS versions).
3. When `sw_vers` errors or its output is missing a key, the
   corresponding field is left at the gopsutil-derived value (Build)
   or empty (VersionExtra).

## Backing library

- [`github.com/shirou/gopsutil/v4/host`](https://github.com/shirou/gopsutil) —
  BSD-3. Primary source for `/etc/os-release` parse + macOS `host.Info`.
- [`github.com/avfs/avfs`](https://github.com/avfs/avfs) — virtual
  filesystem for the redhat-release / debian_version supplements and
  the 10-file legacy `/etc/*-release` fallback cascade.
- [`internal/executor`](../../internal/executor) — shared command-runner
  abstraction used to invoke `sw_vers` on macOS. Tests mock with
  `go.uber.org/mock`.
