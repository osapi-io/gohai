# Platform

> **Status:** Implemented ✅

## Description

The collector reports the OS identification, canonical name, version, family and
architecture. On macOS, it also reports the build and the RSR patch suffix.
`pkg/gohai/family` names the platform and the family, and other code reads the
host's family through `family.Detect()`: collectors through
`internal/platform.Detect()`, to select the code path for each OS and each
family, and osapi for its behaviors. `package_mgr`, `shard` and the per-OS
struct factories in all collectors use this data.

The collector reads `/etc/os-release` (the systemd standard) first, or
`/usr/lib/os-release` when `/etc/os-release` does not exist, as os-release(5)
says. If that gives no result, it tries old release files in sequence, for
example `redhat-release`, `SuSE-release` and `debian_version`. Thus it also
identifies distributions from before systemd and appliance distributions. On
macOS, `sw_vers` gives the version, the build identifier and the Rapid Security
Response patch suffix.

Consumers use this data to:

- Select installer or configuration logic from the OS family (`debian`, `rhel`
  or `arch`).
- Show the composition of the fleet (counts of each `name` and `version`).
- Enable features only on an architecture (`amd64` or `arm64`).
- Find macOS hosts that have, or do not have, a Rapid Security Response patch.

## Collected fields

| Field              | Type   | Description                                                                                                                                                                                                                                                                  | Schema mapping                                              |
| ------------------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `os`               | string | `runtime.GOOS`: `"linux"`, `"darwin"`, `"windows"`.                                                                                                                                                                                                                          | `os.type`.                                                  |
| `name`             | string | The canonical distribution or product ID, for example `"ubuntu"`, `"redhat"`, `"darwin"`.                                                                                                                                                                                    | `os.name`.                                                  |
| `version`          | string | The distribution version, for example `"24.04"`, `"7.9.2009"`, `"14.4.1"`.                                                                                                                                                                                                   | `os.version`.                                               |
| `version_extra`    | string | The macOS RSR patch suffix, for example `"(a)"`. It is empty if no RSR is applied.                                                                                                                                                                                           | No direct schema mapping.                                   |
| `family`           | string | The family, as Ohai names it: `"debian"`, `"rhel"`, `"fedora"`, `"amazon"`, `"suse"`; `"darwin"` on macOS.                                                                                                                                                                   | No direct schema mapping. Packaging logic uses it as input. |
| `cpu_architecture` | string | The host's architecture, by its Go name, for example `"amd64"`, `"arm64"`: from `uname -m`, and on macOS from `sysctl -n hw.optional.arm64` first, so a process under Rosetta still reports `arm64`. It is the binary's own architecture only when the host cannot be asked. | `device.hw_info.cpu_bits` is the nearest match.             |
| `build`            | string | The macOS build identifier, from `sw_vers` `BuildVersion`, for example `"23E224"`.                                                                                                                                                                                           | `os.build`.                                                 |

The collector maps each `name` value through `family.Platform`, which is Ohai's
`platform_id_remap`. It changes common variants to the canonical name of their
parent distribution: `rhel` → `redhat`, `amzn` → `amazon`, `sles` → `suse`, `ol`
→ `oracle`, `archarm` → `arch`, `cumulus-linux` → `cumulus`, and others. The
`os_release` collector gives the raw `/etc/os-release` `ID=` value without this
mapping.

The value of `family` comes from `family.Of`, which is Ohai's
`platform_family_from_platform`: the first family whose match is part of the
name. The families and their members:

- `debian`: ubuntu, debian, linuxmint, raspbian, cumulus, kali, pop.
- `rhel`: only rebuilds that track the versions of RHEL: centos, redhat, oracle,
  almalinux, rocky, scientific, and others. Ohai keeps this family strict, so
  another RPM distribution is `fedora`.
- `amazon`: Amazon Linux, a family of its own.
- `fedora`: fedora, arista_eos.
- `suse`, `arch` (with manjaro), `gentoo`, `wrlinux` (nexus, ios_xr), `alpine`,
  `exherbo`, `clearlinux`, `mandriva`, `slackware`.

A name that no family matches has no family. When an old release file gave the
name, the file's own family applies, for example `rhel` for
`/etc/system-release`.

One difference from Ohai is on purpose: on macOS, the name and the family are
`darwin`, the Go and kernel name, not Ohai's `mac_os_x`.

## Platform support

| Platform | Supported                                                                                     |
| -------- | --------------------------------------------------------------------------------------------- |
| Linux    | ✅ (`/etc/os-release`, the redhat-release supplement, and a sequence of 10 old release files) |
| macOS    | ✅ (`sw_vers` for `ProductVersion`, `BuildVersion` and `ProductVersionExtra`)                 |

## Example output

### Linux (Ubuntu 24.04 x86_64)

```json
{
  "platform": {
    "os": "linux",
    "name": "ubuntu",
    "version": "24.04",
    "family": "debian",
    "cpu_architecture": "amd64"
  }
}
```

### Linux (CentOS 7.9: minor version supplemented from /etc/redhat-release)

```json
{
  "platform": {
    "os": "linux",
    "name": "centos",
    "version": "7.9.2009",
    "family": "rhel",
    "cpu_architecture": "amd64"
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
    "cpu_architecture": "arm64",
    "build": "23E224"
  }
}
```

## SDK usage

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

## Enable/disable

```bash
gohai collect --collector.platform      # enable (default)
gohai collect --no-collector.platform   # disable
```

## Dependencies

None. Other collectors read the `platform` data through
`internal/platform.Detect()`.

## Data sources

On Linux, the collector reads these files through its filesystem, in this
sequence:

1. **`/etc/os-release`**, the modern systemd path. `ID` gives the raw
   distribution ID, and `VERSION_ID` gives the version.
2. **`family.Platform`** changes the raw ID to its canonical form, with Ohai's
   `OS_RELEASE_PLATFORM_REMAP` table. gohai adds `archarm` → `arch`,
   `cumulus-linux` → `cumulus` and `sles_sap` → `suse` to the entries that Ohai
   has.
3. **Minor-version supplement.** This step is for the RHEL family (`centos`,
   `rocky`, `almalinux`, `redhat`, `rhel`, `scientific`, `oracle`, `amazon`). If
   `VERSION_ID` has only the major version (it matches `^\d+$`), the collector
   reads `/etc/redhat-release`. It gets the dotted version with
   `release\s+(\d[\d.]*)`. For `debian` with an empty `VERSION_ID` (testing or
   unstable), the collector reads `/etc/debian_version` as it is. If a
   supplementary file is missing, the original version does not change.
4. **Old `/etc/*-release` files.** If `/etc/os-release` gives no name, the
   collector reads these files in sequence. This occurs on distributions from
   before systemd and on appliance distributions. The first file that gives a
   name decides the result:
   - `/etc/redhat-release` → family `rhel`. A regex gets the name and the dotted
     version.
   - `/etc/SuSE-release` → family `suse` (`VERSION` + `PATCHLEVEL`).
   - `/etc/f5-release` → family `rhel` (F5 BIG-IP).
   - `/etc/system-release` → Amazon, Alma or a Fedora-type distribution.
   - `/etc/debian_version` → family `debian`. The file contents are the version.
   - `/etc/arch-release` → name `arch`, with an empty version (rolling).
   - `/etc/gentoo-release`, `/etc/slackware-version`, `/etc/enterprise-release`,
     `/etc/exherbo-release`. The collector gets the family from the file name.
5. **Family.** `family.Of` gets the family of the name from the maintained
   table, for example `rhel` for alma and rocky, and `debian` for kali and
   raspbian.
6. **CPU architecture.** The value comes from `runtime.GOARCH`.

On macOS:

1. `os`, `name` and `family` are `darwin`.
2. **`sw_vers`.** The collector runs `sw_vers` through its command runner. It
   parses each `Key: Value` line and gets:
   - `ProductVersion` → `Version`.
   - `BuildVersion` → `Build`, the canonical macOS build ID.
   - `ProductVersionExtra` → `VersionExtra`. This is the Apple Rapid Security
     Response suffix, for example `"(a)"`. Most macOS versions do not have it.
3. If `sw_vers` fails, or its output does not have a key, the field stays empty.

## Backing library

- [`github.com/avfs/avfs`](https://github.com/avfs/avfs). The virtual filesystem
  for `/etc/os-release`, the redhat-release and debian_version supplements, and
  the sequence of 10 old `/etc/*-release` files.
- [`source.Runner`](../../pkg/gohai/source). The shared command runner. The
  collector uses it to run `sw_vers` on macOS. Tests mock it with
  `go.uber.org/mock`.
