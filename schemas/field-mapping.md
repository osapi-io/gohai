# Field Mapping Table

Three-tier naming ladder applied to every gohai JSON field.

**Tier legend:**

- **T1** — OCSF: name comes from [OCSF](https://schema.ocsf.io/) object
- **T2** — OTel: name comes from [OTel semconv](https://github.com/open-telemetry/semantic-conventions)
- **T3** — Convention: name follows gohai convention rules (backing library + snake_case + unit suffixes)

**OCSF version:** 1.8.0
**OTel semconv version:** v1.41.1

---

## System Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |
| platform | OS | `os` | T1 | `os` | no | OCSF `os.type` — runtime.GOOS value ("linux", "darwin") | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| platform | Name | `name` | T1 | `name` | no | OCSF `os.name` — distro/product name | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| platform | Version | `version` | T1 | `version` | no | OCSF `os.version` — OS version string | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| platform | VersionExtra | `version_extra` | T3 | `version_extra` | no | No OCSF/OTel equivalent — macOS RSR patch suffix | Convention — gopsutil supplement |
| platform | Family | `family` | T3 | `family` | no | No OCSF/OTel equivalent — distro family ("debian", "rhel") | Convention — gopsutil `PlatformFamily` |
| platform | Architecture | `architecture` | T2 | `architecture` | no | OTel `host.arch` — CPU architecture the host runs on | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| platform | Build | `build` | T1 | `build` | no | OCSF `os.build` — OS build identifier | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| hostname | Name | `name` | T1 | `name` | no | OCSF `device.hostname` — leaf stripped per redundant-prefix rule | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| hostname | MachineName | `machine_name` | T1 | `machine_name` | no | OCSF `device.name` — alternate device name assigned by admin/user | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| hostname | FQDN | `fqdn` | T3 | `fqdn` | no | No OCSF/OTel equivalent — fully qualified domain name from DNS | Convention — Ohai `hostname/fqdn` |
| hostname | Domain | `domain` | T1 | `domain` | no | OCSF `device.domain` — network domain the device resides in | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| kernel | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — uname -s sysname ("Linux", "Darwin") | Convention — POSIX uname `sysname` |
| kernel | Release | `release` | T1 | `release` | no | OCSF `os.kernel_release` — prefix `kernel_` stripped per redundant-prefix rule | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| kernel | Version | `version` | T3 | `version` | no | No OCSF/OTel equivalent — uname -v build/version string | Convention — POSIX uname `version` |
| kernel | Machine | `machine` | T3 | `machine` | no | No OCSF/OTel equivalent — uname -m hardware identifier ("x86_64", "arm64") | Convention — POSIX uname `machine` |
| kernel | Processor | `processor` | T3 | `processor` | no | No OCSF/OTel equivalent — uname -p processor type | Convention — POSIX uname `processor` |
| kernel | OS | `os` | T3 | `os` | no | No OCSF/OTel equivalent — uname -o operating system ("GNU/Linux") | Convention — POSIX uname `os` |
| kernel | RosettaTranslated | `rosetta_translated` | T3 | `rosetta_translated` | no | No OCSF/OTel equivalent — macOS Rosetta 2 translation state | Convention — no schema covers Rosetta |
| kernel_modules | Modules | `modules` | T3 | `modules` | no | No OCSF/OTel equivalent — map of loaded kernel modules | Convention — /proc/modules + kextstat |
| kernel_modules.module | Size | `size` | T3 | `size` | no | No OCSF/OTel equivalent — module size in bytes | Convention — /proc/modules field |
| kernel_modules.module | RefCount | `refcount` | T3 | `refcount` | no | No OCSF/OTel equivalent — module reference count | Convention — /proc/modules field |
| kernel_modules.module | Version | `version` | T3 | `version` | no | No OCSF/OTel equivalent — module version string | Convention — /sys/module/*/version |
| kernel_modules.module | Index | `index` | T3 | `index` | no | No OCSF/OTel equivalent — macOS kextstat load order index | Convention — kextstat field |
| uptime | Seconds | `seconds` | T3 | `seconds` | no | No OCSF/OTel equivalent — seconds since boot | Convention — gopsutil `Uptime` |
| uptime | BootTime | `boot_time` | T1 | `boot_time` | no | OCSF `device.boot_time` — unix timestamp of last boot | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| uptime | Human | `human` | T3 | `human` | no | No OCSF/OTel equivalent — human-readable uptime string | Convention — display field |
| uptime | IdleSeconds | `idle_seconds` | T3 | `idle_seconds` | no | No OCSF/OTel equivalent — aggregate CPU idle seconds | Convention — /proc/uptime field 2 |
| uptime | IdleHuman | `idle_human` | T3 | `idle_human` | no | No OCSF/OTel equivalent — human-readable idle time | Convention — display field |
| timezone | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — IANA timezone name | Convention — stdlib `time.Now().Location()` |
| timezone | Abbrev | `abbrev` | T3 | `abbrev` | no | No OCSF/OTel equivalent — timezone abbreviation ("PDT", "UTC") | Convention — stdlib `time.Now().Zone()` |
| timezone | Offset | `offset` | T3 | `offset` | no | No OCSF/OTel equivalent — UTC offset in seconds | Convention — stdlib `time.Now().Zone()` |
| os_release | ID | `id` | T3 | `id` | no | No OCSF/OTel equivalent — os-release(5) `ID` field | Convention — os-release(5) spec |
| os_release | IDLike | `id_like` | T3 | `id_like` | no | No OCSF/OTel equivalent — os-release(5) `ID_LIKE` field | Convention — os-release(5) spec |
| os_release | Name | `name` | T1 | `name` | no | OCSF `os.name` — os-release(5) `NAME` field | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| os_release | PrettyName | `pretty_name` | T2 | `pretty_name` | no | OTel `os.description` — human-readable OS description | [OTel os](https://github.com/open-telemetry/semantic-conventions/blob/main/model/os/registry.yaml) |
| os_release | Version | `version` | T1 | `version` | no | OCSF `os.version` — os-release(5) `VERSION` field | [OCSF os](https://schema.ocsf.io/1.8.0/objects/os) |
| os_release | VersionID | `version_id` | T3 | `version_id` | no | No OCSF/OTel equivalent — os-release(5) `VERSION_ID` field | Convention — os-release(5) spec |
| os_release | VersionCodename | `version_codename` | T3 | `version_codename` | no | No OCSF/OTel equivalent — os-release(5) `VERSION_CODENAME` | Convention — os-release(5) spec |
| os_release | BuildID | `build_id` | T2 | `build_id` | no | OTel `os.build_id` — unique build/compilation identifier | [OTel os](https://github.com/open-telemetry/semantic-conventions/blob/main/model/os/registry.yaml) |
| os_release | Variant | `variant` | T3 | `variant` | no | No OCSF/OTel equivalent — os-release(5) `VARIANT` field | Convention — os-release(5) spec |
| os_release | VariantID | `variant_id` | T3 | `variant_id` | no | No OCSF/OTel equivalent — os-release(5) `VARIANT_ID` field | Convention — os-release(5) spec |
| os_release | HomeURL | `home_url` | T3 | `home_url` | no | No OCSF/OTel equivalent — os-release(5) `HOME_URL` field | Convention — os-release(5) spec |
| os_release | SupportURL | `support_url` | T3 | `support_url` | no | No OCSF/OTel equivalent — os-release(5) `SUPPORT_URL` field | Convention — os-release(5) spec |
| os_release | BugReportURL | `bug_report_url` | T3 | `bug_report_url` | no | No OCSF/OTel equivalent — os-release(5) `BUG_REPORT_URL` field | Convention — os-release(5) spec |
| os_release | Extra | `extra` | T3 | `extra` | no | No OCSF/OTel equivalent — unparsed os-release(5) keys | Convention — os-release(5) catch-all |
| init | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — init system name (systemd, launchd) | Convention — /proc/1/comm |
| fips | Kernel | `kernel` | T3 | `kernel` | no | No OCSF/OTel equivalent — kernel-level FIPS state container | Convention — no schema covers FIPS |
| fips.kernel | Enabled | `enabled` | T3 | `enabled` | no | No OCSF/OTel equivalent — /proc/sys/crypto/fips_enabled flag | Convention — no schema covers FIPS |
| fips | Policy | `policy` | T3 | `policy` | no | No OCSF/OTel equivalent — crypto-policy state container | Convention — no schema covers FIPS |
| fips.policy | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — crypto-policies config name | Convention — /etc/crypto-policies/config |
| fips.policy | FIPSEffective | `fips_effective` | T3 | `fips_effective` | no | No OCSF/OTel equivalent — true when policy starts with "FIPS" | Convention — no schema covers FIPS |
| machine_id | ID | `id` | T1 | `id` | no | OCSF `device.uid` — unique device identifier; OTel `host.id` also matches | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| root_group | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — root user's primary group name | Convention — stdlib `os/user` |
| shells | Paths | `paths` | T3 | `paths` | no | No OCSF/OTel equivalent — valid login shell paths from /etc/shells | Convention — /etc/shells |
| shard | Seed | `seed` | T3 | `seed` | no | No OCSF/OTel equivalent — deterministic shard seed (MD5-based) | Convention — Ohai shard algorithm |

## Hardware Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |

## Network Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |

## Cloud Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |

## Other Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |
