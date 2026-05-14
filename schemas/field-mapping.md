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
| cpu | Count | `count` | T1 | `count` | no | OCSF `device_hw_info.cpu_count` — prefix `cpu_` stripped per redundant-prefix rule | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| cpu | Sockets | `sockets` | T3 | `sockets` | no | No OCSF/OTel equivalent — physical CPU socket/package count | Convention — gopsutil `InfoStat.PhysicalID` cardinality |
| cpu | Cores | `cores` | T1 | `cores` | no | OCSF `device_hw_info.cpu_cores` — prefix `cpu_` stripped per redundant-prefix rule | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| cpu | ModelName | `model_name` | T2 | `model_name` | no | OTel `host.cpu.model.name` — processor model designation | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu | VendorID | `vendor_id` | T2 | `vendor_id` | no | OTel `host.cpu.vendor.id` — processor manufacturer identifier | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu | Family | `family` | T2 | `family` | no | OTel `host.cpu.family` — CPU family or generation | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu | Model | `model` | T2 | `model` | no | OTel `host.cpu.model.id` — model identifier within family | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu | Stepping | `stepping` | T2 | `stepping` | no | OTel `host.cpu.stepping` — core revision/stepping | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu | Mhz | `mhz` | T1 | `mhz` | no | OCSF `device_hw_info.cpu_speed` — current frequency in MHz | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| cpu | CacheSize | `cache_size` | T2 | `cache_size` | no | OTel `host.cpu.cache.l2.size` — aggregate cache from /proc/cpuinfo (KB) | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu | Flags | `flags` | T3 | `flags` | no | No OCSF/OTel equivalent — CPU feature flags | Convention — gopsutil `InfoStat.Flags` |
| cpu | Caches | `caches` | T3 | `caches` | no | No OCSF/OTel equivalent — per-level cache sizes container | Convention — lscpu output |
| cpu | NumaNodes | `numa_nodes` | T3 | `numa_nodes` | no | No OCSF/OTel equivalent — NUMA node → CPU index mapping | Convention — lscpu `NUMA node<N> CPU(s)` |
| cpu | NumaNodesCount | `numa_nodes_count` | T3 | `numa_nodes_count` | no | No OCSF/OTel equivalent — NUMA node count | Convention — lscpu `NUMA node(s)` |
| cpu | Vulnerabilities | `vulnerabilities` | T3 | `vulnerabilities` | no | No OCSF/OTel equivalent — mitigation → status map | Convention — /sys/devices/system/cpu/vulnerabilities |
| cpu | CPUsOnline | `cpus_online` | T3 | `cpus_online` | no | No OCSF/OTel equivalent — online logical CPU count | Convention — lscpu `On-line CPU(s) list` |
| cpu | CPUsOffline | `cpus_offline` | T3 | `cpus_offline` | no | No OCSF/OTel equivalent — offline logical CPU count | Convention — lscpu `Off-line CPU(s) list` |
| cpu | BIOSVendorID | `bios_vendor_id` | T3 | `bios_vendor_id` | no | No OCSF/OTel equivalent — BIOS-reported CPU vendor from lscpu | Convention — lscpu `BIOS Vendor ID` |
| cpu | BIOSModelName | `bios_model_name` | T3 | `bios_model_name` | no | No OCSF/OTel equivalent — BIOS-reported CPU model from lscpu | Convention — lscpu `BIOS Model name` |
| cpu | MachineType | `machine_type` | T3 | `machine_type` | no | No OCSF/OTel equivalent — s390x mainframe machine type | Convention — lscpu `Machine type` |
| cpu | MhzMax | `mhz_max` | T3 | `mhz_max` | no | No OCSF/OTel equivalent — maximum CPU frequency string | Convention — lscpu `CPU max MHz` |
| cpu | MhzMin | `mhz_min` | T3 | `mhz_min` | no | No OCSF/OTel equivalent — minimum CPU frequency string | Convention — lscpu `CPU min MHz` |
| cpu | MhzDynamic | `mhz_dynamic` | T3 | `mhz_dynamic` | no | No OCSF/OTel equivalent — dynamic CPU frequency (s390x) | Convention — lscpu `CPU dynamic MHz` |
| cpu | Bogomips | `bogomips` | T3 | `bogomips` | no | No OCSF/OTel equivalent — BogoMIPS calibration value | Convention — lscpu `BogoMIPS` |
| cpu | CPUOpmodes | `cpu_opmodes` | T3 | `cpu_opmodes` | no | No OCSF/OTel equivalent — supported CPU operation modes | Convention — lscpu `CPU op-mode(s)` |
| cpu | ByteOrder | `byte_order` | T3 | `byte_order` | no | No OCSF/OTel equivalent — CPU byte order | Convention — lscpu `Byte Order` |
| cpu | AddressSizes | `address_sizes` | T3 | `address_sizes` | no | No OCSF/OTel equivalent — physical/virtual address sizes | Convention — lscpu `Address sizes` |
| cpu | Virtualization | `virtualization` | T3 | `virtualization` | no | No OCSF/OTel equivalent — CPU virtualization capability | Convention — lscpu `Virtualization` |
| cpu | VirtualizationType | `virtualization_type` | T3 | `virtualization_type` | no | No OCSF/OTel equivalent — virtualization type string | Convention — lscpu `Virtualization type` |
| cpu | HypervisorVendor | `hypervisor_vendor` | T3 | `hypervisor_vendor` | no | No OCSF/OTel equivalent — hypervisor vendor name from lscpu | Convention — lscpu `Hypervisor vendor` |
| cpu | DispatchingMode | `dispatching_mode` | T3 | `dispatching_mode` | no | No OCSF/OTel equivalent — s390x dispatching mode | Convention — lscpu `Dispatching mode` |
| cpu | CPUs | `cpus` | T3 | `cpus` | no | No OCSF/OTel equivalent — per-logical-CPU breakdown array | Convention — Ohai `cpu["N"]` entries |
| cpu.caches | L1d | `l1d` | T3 | `l1d` | no | No OCSF/OTel equivalent — L1 data cache size string | Convention — lscpu `L1d cache` |
| cpu.caches | L1i | `l1i` | T3 | `l1i` | no | No OCSF/OTel equivalent — L1 instruction cache size string | Convention — lscpu `L1i cache` |
| cpu.caches | L2 | `l2` | T2 | `l2` | no | OTel `host.cpu.cache.l2.size` — L2 cache size string | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.caches | L2d | `l2d` | T3 | `l2d` | no | No OCSF/OTel equivalent — L2 data cache (split-L2 archs) | Convention — lscpu `L2d cache` |
| cpu.caches | L2i | `l2i` | T3 | `l2i` | no | No OCSF/OTel equivalent — L2 instruction cache (split-L2 archs) | Convention — lscpu `L2i cache` |
| cpu.caches | L3 | `l3` | T3 | `l3` | no | No OCSF/OTel equivalent — L3 cache size string | Convention — lscpu `L3 cache` |
| cpu.caches | L4 | `l4` | T3 | `l4` | no | No OCSF/OTel equivalent — L4 cache size string (rare) | Convention — lscpu `L4 cache` |
| cpu.cpus[] | VendorID | `vendor_id` | T2 | `vendor_id` | no | OTel `host.cpu.vendor.id` — per-CPU vendor identifier | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.cpus[] | Family | `family` | T2 | `family` | no | OTel `host.cpu.family` — per-CPU family | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.cpus[] | Model | `model` | T2 | `model` | no | OTel `host.cpu.model.id` — per-CPU model identifier | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.cpus[] | ModelName | `model_name` | T2 | `model_name` | no | OTel `host.cpu.model.name` — per-CPU model designation | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.cpus[] | Stepping | `stepping` | T2 | `stepping` | no | OTel `host.cpu.stepping` — per-CPU stepping | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.cpus[] | PhysicalID | `physical_id` | T3 | `physical_id` | no | No OCSF/OTel equivalent — socket index from /proc/cpuinfo | Convention — gopsutil `InfoStat.PhysicalID` |
| cpu.cpus[] | CoreID | `core_id` | T3 | `core_id` | no | No OCSF/OTel equivalent — physical core index within socket | Convention — gopsutil `InfoStat.CoreID` |
| cpu.cpus[] | Cores | `cores` | T3 | `cores` | no | No OCSF/OTel equivalent — cores on this socket | Convention — gopsutil `InfoStat.Cores` |
| cpu.cpus[] | Mhz | `mhz` | T1 | `mhz` | no | OCSF `device_hw_info.cpu_speed` — per-CPU frequency | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| cpu.cpus[] | CacheSize | `cache_size` | T2 | `cache_size` | no | OTel `host.cpu.cache.l2.size` — per-CPU cache size (KB) | [OTel host](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml) |
| cpu.cpus[] | Flags | `flags` | T3 | `flags` | no | No OCSF/OTel equivalent — per-CPU feature flags | Convention — gopsutil `InfoStat.Flags` |
| memory | Total | `total` | T1 | `total` | no | OCSF `device_hw_info.ram_size` — prefix `memory_` stripped; total physical RAM | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| memory | Available | `available` | T2 | `available` | no | OTel `system.memory.linux.available` — estimate of memory available for new apps | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory | Used | `used` | T2 | `used` | no | OTel `system.memory.usage` with `state=used` — memory in active use | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory | UsedPercent | `used_percent` | T2 | `used_percent` | no | OTel `system.memory.utilization` — percentage of memory in use | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory | Free | `free` | T2 | `free` | no | OTel `system.memory.usage` with `state=free` — free memory | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory | Active | `active` | T3 | `active` | no | No OCSF/OTel equivalent — active LRU pages | Convention — /proc/meminfo `Active` |
| memory | Inactive | `inactive` | T3 | `inactive` | no | No OCSF/OTel equivalent — inactive LRU pages | Convention — /proc/meminfo `Inactive` |
| memory | ActiveAnon | `active_anon` | T3 | `active_anon` | no | No OCSF/OTel equivalent — active anonymous pages | Convention — /proc/meminfo `Active(anon)` |
| memory | InactiveAnon | `inactive_anon` | T3 | `inactive_anon` | no | No OCSF/OTel equivalent — inactive anonymous pages | Convention — /proc/meminfo `Inactive(anon)` |
| memory | ActiveFile | `active_file` | T3 | `active_file` | no | No OCSF/OTel equivalent — active file-backed pages | Convention — /proc/meminfo `Active(file)` |
| memory | InactiveFile | `inactive_file` | T3 | `inactive_file` | no | No OCSF/OTel equivalent — inactive file-backed pages | Convention — /proc/meminfo `Inactive(file)` |
| memory | Unevictable | `unevictable` | T3 | `unevictable` | no | No OCSF/OTel equivalent — unevictable pages (mlock, ramfs) | Convention — /proc/meminfo `Unevictable` |
| memory | Wired | `wired` | T3 | `wired` | no | No OCSF/OTel equivalent — macOS wired memory | Convention — gopsutil `VirtualMemoryStat.Wired` |
| memory | Speculative | `speculative` | T3 | `speculative` | no | No OCSF/OTel equivalent — macOS speculative pages | Convention — darwin vm_stat |
| memory | Compressed | `compressed` | T3 | `compressed` | no | No OCSF/OTel equivalent — macOS compressed memory | Convention — darwin vm_stat |
| memory | Buffers | `buffers` | T2 | `buffers` | no | OTel `system.memory.usage` with `state=buffers` — buffer cache | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory | Cached | `cached` | T2 | `cached` | no | OTel `system.memory.usage` with `state=cached` — page cache | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory | Dirty | `dirty` | T3 | `dirty` | no | No OCSF/OTel equivalent — dirty pages waiting writeback | Convention — /proc/meminfo `Dirty` |
| memory | WriteBack | `writeback` | T3 | `writeback` | no | No OCSF/OTel equivalent — pages being written back | Convention — /proc/meminfo `Writeback` |
| memory | WriteBackTmp | `writeback_tmp` | T3 | `writeback_tmp` | no | No OCSF/OTel equivalent — FUSE temporary writeback | Convention — /proc/meminfo `WritebackTmp` |
| memory | Shared | `shared` | T2 | `shared` | no | OTel `system.memory.linux.shared` — shared memory (tmpfs) | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory | Mapped | `mapped` | T3 | `mapped` | no | No OCSF/OTel equivalent — memory-mapped file pages | Convention — /proc/meminfo `Mapped` |
| memory | Slab | `slab` | T2 | `slab` | no | OTel `system.memory.linux.slab.usage` — total slab allocator memory | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory | SReclaimable | `s_reclaimable` | T2 | `s_reclaimable` | no | OTel `system.memory.linux.slab.usage` with `state=reclaimable` — reclaimable slab | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory | SUnreclaim | `s_unreclaim` | T2 | `s_unreclaim` | no | OTel `system.memory.linux.slab.usage` with `state=unreclaimable` — unreclaimable slab | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory | KReclaimable | `k_reclaimable` | T3 | `k_reclaimable` | no | No OCSF/OTel equivalent — kernel reclaimable memory (slab + other) | Convention — /proc/meminfo `KReclaimable` |
| memory | PageTables | `page_tables` | T3 | `page_tables` | no | No OCSF/OTel equivalent — page table memory usage | Convention — /proc/meminfo `PageTables` |
| memory | KernelStack | `kernel_stack` | T3 | `kernel_stack` | no | No OCSF/OTel equivalent — kernel stack allocation | Convention — /proc/meminfo `KernelStack` |
| memory | PerCPU | `percpu` | T3 | `percpu` | no | No OCSF/OTel equivalent — per-CPU allocations | Convention — /proc/meminfo `Percpu` |
| memory | HighTotal | `high_total` | T3 | `high_total` | no | No OCSF/OTel equivalent — high memory total (32-bit) | Convention — /proc/meminfo `HighTotal` |
| memory | HighFree | `high_free` | T3 | `high_free` | no | No OCSF/OTel equivalent — high memory free (32-bit) | Convention — /proc/meminfo `HighFree` |
| memory | LowTotal | `low_total` | T3 | `low_total` | no | No OCSF/OTel equivalent — low memory total (32-bit) | Convention — /proc/meminfo `LowTotal` |
| memory | LowFree | `low_free` | T3 | `low_free` | no | No OCSF/OTel equivalent — low memory free (32-bit) | Convention — /proc/meminfo `LowFree` |
| memory | NFSUnstable | `nfs_unstable` | T3 | `nfs_unstable` | no | No OCSF/OTel equivalent — NFS unstable pages | Convention — /proc/meminfo `NFS_Unstable` |
| memory | Bounce | `bounce` | T3 | `bounce` | no | No OCSF/OTel equivalent — bounce buffer memory | Convention — /proc/meminfo `Bounce` |
| memory | AnonPages | `anon_pages` | T3 | `anon_pages` | no | No OCSF/OTel equivalent — anonymous page-backed memory | Convention — /proc/meminfo `AnonPages` |
| memory | Shmem | `shmem` | T3 | `shmem` | no | No OCSF/OTel equivalent — shmem + tmpfs usage | Convention — /proc/meminfo `Shmem` |
| memory | DirectMap | `direct_map` | T3 | `direct_map` | no | No OCSF/OTel equivalent — direct-map page-table granularity container | Convention — /proc/meminfo DirectMap fields |
| memory | CommitLimit | `commit_limit` | T3 | `commit_limit` | no | No OCSF/OTel equivalent — memory overcommit limit | Convention — /proc/meminfo `CommitLimit` |
| memory | CommittedAS | `committed_as` | T3 | `committed_as` | no | No OCSF/OTel equivalent — committed address space | Convention — /proc/meminfo `Committed_AS` |
| memory | VmallocTotal | `vmalloc_total` | T3 | `vmalloc_total` | no | No OCSF/OTel equivalent — vmalloc arena total | Convention — /proc/meminfo `VmallocTotal` |
| memory | VmallocUsed | `vmalloc_used` | T3 | `vmalloc_used` | no | No OCSF/OTel equivalent — vmalloc arena used | Convention — /proc/meminfo `VmallocUsed` |
| memory | VmallocChunk | `vmalloc_chunk` | T3 | `vmalloc_chunk` | no | No OCSF/OTel equivalent — largest contiguous vmalloc block | Convention — /proc/meminfo `VmallocChunk` |
| memory | HugePages | `hugepages` | T3 | `hugepages` | no | No OCSF/OTel equivalent — hugepages configuration container | Convention — /proc/meminfo HugePages fields |
| memory | Swap | `swap` | T3 | `swap` | no | No OCSF/OTel equivalent — swap usage container | Convention — gopsutil `SwapMemoryStat` |
| memory.hugepages | Total | `total` | T2 | `total` | no | OTel `system.memory.linux.hugepages.limit` — total hugepages count | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.hugepages | Free | `free` | T2 | `free` | no | OTel `system.memory.linux.hugepages.usage` with `state=free` — free hugepages | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.hugepages | Reserved | `reserved` | T2 | `reserved` | no | OTel `system.memory.linux.hugepages.reserved` — reserved hugepages | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.hugepages | Surplus | `surplus` | T2 | `surplus` | no | OTel `system.memory.linux.hugepages.surplus` — surplus hugepages | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.hugepages | Size | `size` | T2 | `size` | no | OTel `system.memory.linux.hugepages.page_size` — hugepage size in bytes | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.hugepages | AnonHugePages | `anon_hugepages` | T3 | `anon_hugepages` | no | No OCSF/OTel equivalent — transparent hugepage anonymous memory | Convention — /proc/meminfo `AnonHugePages` |
| memory.hugepages | Hugetlb | `hugetlb` | T3 | `hugetlb` | no | No OCSF/OTel equivalent — total hugepage TLB memory | Convention — /proc/meminfo `Hugetlb` |
| memory.direct_map | Map4k | `map_4k` | T3 | `map_4k` | no | No OCSF/OTel equivalent — memory mapped with 4k pages | Convention — /proc/meminfo `DirectMap4k` |
| memory.direct_map | Map2M | `map_2m` | T3 | `map_2m` | no | No OCSF/OTel equivalent — memory mapped with 2M pages | Convention — /proc/meminfo `DirectMap2M` |
| memory.direct_map | Map1G | `map_1g` | T3 | `map_1g` | no | No OCSF/OTel equivalent — memory mapped with 1G pages | Convention — /proc/meminfo `DirectMap1G` |
| memory.swap | Total | `total` | T2 | `total` | no | OTel `system.paging.usage` — total swap capacity | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.swap | Used | `used` | T2 | `used` | no | OTel `system.paging.usage` with `state=used` — swap in use | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory.swap | Free | `free` | T2 | `free` | no | OTel `system.paging.usage` with `state=free` — swap free | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| memory.swap | UsedPercent | `used_percent` | T2 | `used_percent` | no | OTel `system.paging.utilization` — swap utilization fraction | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| memory.swap | Cached | `cached` | T3 | `cached` | no | No OCSF/OTel equivalent — swap pages also in page cache | Convention — /proc/meminfo `SwapCached` |
| disk | Devices | `devices` | T3 | `devices` | no | No OCSF/OTel equivalent — block device I/O counters array | Convention — gopsutil `disk.IOCounters` |
| disk.devices[] | Name | `name` | T2 | `name` | no | OTel `system.device` — device identifier | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| disk.devices[] | ReadCount | `read_count` | T2 | `read_count` | no | OTel `system.disk.operations` with `direction=read` — read operation count | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| disk.devices[] | WriteCount | `write_count` | T2 | `write_count` | no | OTel `system.disk.operations` with `direction=write` — write operation count | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| disk.devices[] | ReadBytes | `read_bytes` | T2 | `read_bytes` | no | OTel `system.disk.io` with `direction=read` — bytes read | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| disk.devices[] | WriteBytes | `write_bytes` | T2 | `write_bytes` | no | OTel `system.disk.io` with `direction=write` — bytes written | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| disk.devices[] | ReadTime | `read_time` | T2 | `read_time` | no | OTel `system.disk.operation_time` with `direction=read` — cumulative read time | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| disk.devices[] | WriteTime | `write_time` | T2 | `write_time` | no | OTel `system.disk.operation_time` with `direction=write` — cumulative write time | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| disk.devices[] | IoTime | `io_time` | T2 | `io_time` | no | OTel `system.disk.io_time` — time disk spent activated | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| filesystem | Mounts | `mounts` | T3 | `mounts` | no | No OCSF/OTel equivalent — mounted filesystem array | Convention — gopsutil `disk.Partitions` |
| filesystem | Unmounted | `unmounted` | T3 | `unmounted` | no | No OCSF/OTel equivalent — unmounted filesystem array | Convention — lsblk output |
| filesystem | ZFSDatasets | `zfs_datasets` | T3 | `zfs_datasets` | no | No OCSF/OTel equivalent — ZFS dataset array | Convention — `zfs get all` output |
| filesystem.mounts[] | Device | `device` | T2 | `device` | no | OTel `system.device` — device identifier for filesystem | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| filesystem.mounts[] | Mountpoint | `mountpoint` | T2 | `mountpoint` | no | OTel `system.filesystem.mountpoint` — filesystem mount path | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| filesystem.mounts[] | Fstype | `fstype` | T2 | `fstype` | no | OTel `system.filesystem.type` — filesystem type | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| filesystem.mounts[] | Opts | `opts` | T2 | `opts` | no | OTel `system.filesystem.mode` — mount options (rw, ro, etc.) | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| filesystem.mounts[] | Total | `total` | T2 | `total` | no | OTel `system.filesystem.limit` — total filesystem capacity | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| filesystem.mounts[] | Used | `used` | T2 | `used` | no | OTel `system.filesystem.usage` with `state=used` — used space | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| filesystem.mounts[] | Free | `free` | T2 | `free` | no | OTel `system.filesystem.usage` with `state=free` — free space | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| filesystem.mounts[] | UsedPercent | `used_percent` | T2 | `used_percent` | no | OTel `system.filesystem.utilization` — usage fraction | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| filesystem.mounts[] | InodesTotal | `inodes_total` | T3 | `inodes_total` | no | No OCSF/OTel equivalent — total inode count | Convention — gopsutil `UsageStat.InodesTotal` |
| filesystem.mounts[] | InodesUsed | `inodes_used` | T3 | `inodes_used` | no | No OCSF/OTel equivalent — used inode count | Convention — gopsutil `UsageStat.InodesUsed` |
| filesystem.mounts[] | InodesFree | `inodes_free` | T3 | `inodes_free` | no | No OCSF/OTel equivalent — free inode count | Convention — gopsutil `UsageStat.InodesFree` |
| filesystem.mounts[] | InodesUsedPercent | `inodes_used_percent` | T3 | `inodes_used_percent` | no | No OCSF/OTel equivalent — inode usage percentage | Convention — gopsutil `UsageStat.InodesUsedPercent` |
| filesystem.mounts[] | UUID | `uuid` | T3 | `uuid` | no | No OCSF/OTel equivalent — filesystem UUID from lsblk | Convention — lsblk `UUID` |
| filesystem.mounts[] | Label | `label` | T3 | `label` | no | No OCSF/OTel equivalent — filesystem label from lsblk | Convention — lsblk `LABEL` |
| filesystem.mounts[] | PartUUID | `part_uuid` | T3 | `part_uuid` | no | No OCSF/OTel equivalent — GPT partition UUID from lsblk | Convention — lsblk `PARTUUID` |
| filesystem.mounts[] | PartLabel | `part_label` | T3 | `part_label` | no | No OCSF/OTel equivalent — GPT partition label from lsblk | Convention — lsblk `PARTLABEL` |
| filesystem.mounts[] | Btrfs | `btrfs` | T3 | `btrfs` | no | No OCSF/OTel equivalent — btrfs-specific data container | Convention — /sys/fs/btrfs sysfs |
| filesystem.mounts[].btrfs | RAID | `raid` | T3 | `raid` | no | No OCSF/OTel equivalent — btrfs RAID profile name | Convention — /sys/fs/btrfs/<UUID>/allocation |
| filesystem.mounts[].btrfs | Allocation | `allocation` | T3 | `allocation` | no | No OCSF/OTel equivalent — per-block-group-type allocation map | Convention — /sys/fs/btrfs/<UUID>/allocation |
| filesystem.mounts[].btrfs.allocation[] | TotalBytes | `total_bytes` | T3 | `total_bytes` | no | No OCSF/OTel equivalent — btrfs block-group total bytes | Convention — /sys/fs/btrfs total_bytes |
| filesystem.mounts[].btrfs.allocation[] | BytesUsed | `bytes_used` | T3 | `bytes_used` | no | No OCSF/OTel equivalent — btrfs block-group bytes used | Convention — /sys/fs/btrfs bytes_used |
| filesystem.unmounted[] | Device | `device` | T2 | `device` | no | OTel `system.device` — device identifier | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| filesystem.unmounted[] | Fstype | `fstype` | T2 | `fstype` | no | OTel `system.filesystem.type` — filesystem type | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/registry.yaml) |
| filesystem.unmounted[] | UUID | `uuid` | T3 | `uuid` | no | No OCSF/OTel equivalent — filesystem UUID from lsblk | Convention — lsblk `UUID` |
| filesystem.unmounted[] | Label | `label` | T3 | `label` | no | No OCSF/OTel equivalent — filesystem label from lsblk | Convention — lsblk `LABEL` |
| filesystem.unmounted[] | PartUUID | `part_uuid` | T3 | `part_uuid` | no | No OCSF/OTel equivalent — GPT partition UUID from lsblk | Convention — lsblk `PARTUUID` |
| filesystem.unmounted[] | PartLabel | `part_label` | T3 | `part_label` | no | No OCSF/OTel equivalent — GPT partition label from lsblk | Convention — lsblk `PARTLABEL` |
| filesystem.zfs_datasets[] | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — ZFS dataset path | Convention — `zfs get` output |
| filesystem.zfs_datasets[] | Mountpoint | `mountpoint` | T3 | `mountpoint` | no | No OCSF/OTel equivalent — ZFS dataset mount path | Convention — `zfs get` mountpoint property |
| filesystem.zfs_datasets[] | IsPool | `is_pool` | T3 | `is_pool` | no | No OCSF/OTel equivalent — true when dataset is zpool root | Convention — Ohai `zfs_zpool` |
| filesystem.zfs_datasets[] | Parents | `parents` | T3 | `parents` | no | No OCSF/OTel equivalent — ancestor dataset paths | Convention — Ohai `zfs_parents` |
| filesystem.zfs_datasets[] | Properties | `properties` | T3 | `properties` | no | No OCSF/OTel equivalent — full property map from `zfs get` | Convention — `zfs get all` output |
| filesystem.zfs_datasets[].properties[] | Value | `value` | T3 | `value` | no | No OCSF/OTel equivalent — ZFS property value | Convention — `zfs get` column |
| filesystem.zfs_datasets[].properties[] | Source | `source` | T3 | `source` | no | No OCSF/OTel equivalent — ZFS property source annotation | Convention — `zfs get` column |
| dmi | BIOS | `bios` | T1 | `bios` | no | OCSF `device_hw_info.bios_*` — firmware identity container | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi | Baseboard | `baseboard` | T3 | `baseboard` | no | No OCSF/OTel equivalent — motherboard identity container | Convention — ghw `BaseboardInfo` |
| dmi | Chassis | `chassis` | T1 | `chassis` | no | OCSF `device_hw_info.chassis` — enclosure identity container | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi | Product | `product` | T3 | `product` | no | No OCSF/OTel equivalent — system identity container (DMI type 1) | Convention — ghw `ProductInfo` |
| dmi.bios | Vendor | `vendor` | T1 | `vendor` | no | OCSF `device_hw_info.bios_manufacturer` — firmware vendor | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.bios | Version | `version` | T1 | `version` | no | OCSF `device_hw_info.bios_ver` — firmware version | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.bios | Date | `date` | T1 | `date` | no | OCSF `device_hw_info.bios_date` — firmware release date | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.baseboard | Vendor | `vendor` | T3 | `vendor` | no | No OCSF/OTel equivalent — baseboard manufacturer | Convention — ghw `BaseboardInfo.Vendor` |
| dmi.baseboard | Product | `product` | T3 | `product` | no | No OCSF/OTel equivalent — baseboard product name | Convention — ghw `BaseboardInfo.Product` |
| dmi.baseboard | Version | `version` | T3 | `version` | no | No OCSF/OTel equivalent — baseboard version | Convention — ghw `BaseboardInfo.Version` |
| dmi.baseboard | SerialNumber | `serial_number` | T3 | `serial_number` | no | No OCSF/OTel equivalent — baseboard serial number | Convention — ghw `BaseboardInfo.SerialNumber` |
| dmi.baseboard | AssetTag | `asset_tag` | T3 | `asset_tag` | no | No OCSF/OTel equivalent — baseboard asset tag | Convention — ghw `BaseboardInfo.AssetTag` |
| dmi.chassis | Vendor | `vendor` | T1 | `vendor` | no | OCSF `device.vendor_name` — chassis manufacturer | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| dmi.chassis | Type | `type` | T3 | `type` | no | No OCSF/OTel equivalent — chassis type code | Convention — ghw `ChassisInfo.Type` |
| dmi.chassis | TypeDescription | `type_description` | T3 | `type_description` | no | No OCSF/OTel equivalent — human-readable chassis type | Convention — ghw `ChassisInfo.TypeDescription` |
| dmi.chassis | Version | `version` | T3 | `version` | no | No OCSF/OTel equivalent — chassis version | Convention — ghw `ChassisInfo.Version` |
| dmi.chassis | SerialNumber | `serial_number` | T1 | `serial_number` | no | OCSF `device_hw_info.serial_number` — chassis serial number | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.chassis | AssetTag | `asset_tag` | T3 | `asset_tag` | no | No OCSF/OTel equivalent — chassis asset tag | Convention — ghw `ChassisInfo.AssetTag` |
| dmi.product | Vendor | `vendor` | T1 | `vendor` | no | OCSF `device_hw_info.vendor_name` — system manufacturer | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.product | Name | `name` | T1 | `name` | no | OCSF `device.model` — system/product name | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| dmi.product | Family | `family` | T3 | `family` | no | No OCSF/OTel equivalent — product family | Convention — ghw `ProductInfo.Family` |
| dmi.product | Version | `version` | T3 | `version` | no | No OCSF/OTel equivalent — product version | Convention — ghw `ProductInfo.Version` |
| dmi.product | SerialNumber | `serial_number` | T1 | `serial_number` | no | OCSF `device_hw_info.serial_number` — product serial | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.product | UUID | `uuid` | T1 | `uuid` | no | OCSF `device_hw_info.uuid` — SMBIOS system UUID | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| dmi.product | SKU | `sku` | T3 | `sku` | no | No OCSF/OTel equivalent — product SKU | Convention — ghw `ProductInfo.SKU` |
| gpu | Cards | `cards` | T3 | `cards` | no | No OCSF/OTel equivalent — GPU device array | Convention — ghw `gpu.GraphicsCards` |
| gpu.cards[] | Vendor | `vendor` | T2 | `vendor` | no | OTel `hw.vendor` — GPU vendor name | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/registry.yaml) |
| gpu.cards[] | Model | `model` | T2 | `model` | no | OTel `hw.model` — GPU model name | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/registry.yaml) |
| gpu.cards[] | Address | `address` | T2 | `address` | no | OTel `hw.id` — unique hardware component identifier (PCI address) | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/registry.yaml) |
| gpu.cards[] | VendorID | `vendor_id` | T3 | `vendor_id` | no | No OCSF/OTel equivalent — PCI vendor hex code | Convention — ghw `PCIAddress.Vendor` |
| gpu.cards[] | DeviceID | `device_id` | T3 | `device_id` | no | No OCSF/OTel equivalent — PCI device hex code | Convention — ghw `PCIAddress.Product` |
| gpu.cards[] | Cores | `cores` | T3 | `cores` | no | No OCSF/OTel equivalent — Apple GPU core count (darwin only) | Convention — system_profiler `sppci_cores` |
| gpu.cards[] | Bus | `bus` | T3 | `bus` | no | No OCSF/OTel equivalent — bus type (darwin: builtin/pcie) | Convention — system_profiler `sppci_bus` |
| pci | Devices | `devices` | T3 | `devices` | no | No OCSF/OTel equivalent — PCI device map keyed by address | Convention — ghw `pci.ListDevices` |
| pci.devices[] | VendorID | `vendor_id` | T3 | `vendor_id` | no | No OCSF/OTel equivalent — PCI vendor hex code | Convention — ghw `PCIDevice.Vendor.ID` |
| pci.devices[] | VendorName | `vendor_name` | T3 | `vendor_name` | no | No OCSF/OTel equivalent — PCI vendor human name | Convention — ghw `PCIDevice.Vendor.Name` |
| pci.devices[] | DeviceID | `device_id` | T3 | `device_id` | no | No OCSF/OTel equivalent — PCI device hex code | Convention — ghw `PCIDevice.Product.ID` |
| pci.devices[] | DeviceName | `device_name` | T3 | `device_name` | no | No OCSF/OTel equivalent — PCI device human name | Convention — ghw `PCIDevice.Product.Name` |
| pci.devices[] | ClassID | `class_id` | T3 | `class_id` | no | No OCSF/OTel equivalent — PCI class hex code | Convention — ghw `PCIDevice.Class.ID` |
| pci.devices[] | ClassName | `class_name` | T3 | `class_name` | no | No OCSF/OTel equivalent — PCI class human name | Convention — ghw `PCIDevice.Class.Name` |
| pci.devices[] | SubclassID | `subclass_id` | T3 | `subclass_id` | no | No OCSF/OTel equivalent — PCI subclass hex code | Convention — ghw `PCIDevice.Subclass.ID` |
| pci.devices[] | SubclassName | `subclass_name` | T3 | `subclass_name` | no | No OCSF/OTel equivalent — PCI subclass human name | Convention — ghw `PCIDevice.Subclass.Name` |
| pci.devices[] | SubsystemID | `sdevice_id` | T3 | `sdevice_id` | no | No OCSF/OTel equivalent — PCI subsystem device hex code | Convention — Ohai lspci `sdevice_id` |
| pci.devices[] | SubsystemName | `sdevice_name` | T3 | `sdevice_name` | no | No OCSF/OTel equivalent — PCI subsystem device human name | Convention — Ohai lspci `sdevice_name` |
| pci.devices[] | Revision | `revision` | T3 | `revision` | no | No OCSF/OTel equivalent — PCI revision ID | Convention — ghw `PCIDevice.Revision` |
| pci.devices[] | Driver | `driver` | T3 | `driver` | no | No OCSF/OTel equivalent — bound kernel driver name | Convention — ghw `PCIDevice.Driver` |
| pci.devices[] | IOMMUGroup | `iommu_group` | T3 | `iommu_group` | no | No OCSF/OTel equivalent — IOMMU group assignment | Convention — /sys/bus/pci/devices/*/iommu_group |
| pci.devices[] | ParentAddress | `parent_address` | T3 | `parent_address` | no | No OCSF/OTel equivalent — parent PCI bridge address | Convention — sysfs PCI hierarchy |
| scsi | Devices | `devices` | T3 | `devices` | no | No OCSF/OTel equivalent — SCSI device map keyed by address | Convention — lsscsi output |
| scsi.devices[] | SCSIAddr | `scsi_addr` | T3 | `scsi_addr` | no | No OCSF/OTel equivalent — SCSI H:C:T:L address | Convention — lsscsi address field |
| scsi.devices[] | Type | `type` | T3 | `type` | no | No OCSF/OTel equivalent — SCSI device type (disk, cd, etc.) | Convention — lsscsi type field |
| scsi.devices[] | Transport | `transport` | T3 | `transport` | no | No OCSF/OTel equivalent — SCSI transport protocol | Convention — lsscsi transport field |
| scsi.devices[] | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — vendor + model name | Convention — lsscsi vendor/model fields |
| scsi.devices[] | Revision | `revision` | T3 | `revision` | no | No OCSF/OTel equivalent — firmware revision | Convention — lsscsi revision field |
| scsi.devices[] | Device | `device` | T3 | `device` | no | No OCSF/OTel equivalent — device node path (/dev/sdX) | Convention — lsscsi device field |
| hardware | MachineModel | `machine_model` | T3 | `machine_model` | no | No OCSF/OTel equivalent — macOS machine model identifier | Convention — system_profiler `machine_model` |
| hardware | MachineName | `machine_name` | T3 | `machine_name` | no | No OCSF/OTel equivalent — macOS machine marketing name | Convention — system_profiler `machine_name` |
| hardware | SerialNumber | `serial_number` | T1 | `serial_number` | no | OCSF `device_hw_info.serial_number` — hardware serial number | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| hardware | PlatformUUID | `platform_uuid` | T1 | `platform_uuid` | no | OCSF `device_hw_info.uuid` — IOPlatformUUID (macOS hardware UUID) | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| hardware | ProvisioningUDID | `provisioning_udid` | T3 | `provisioning_udid` | no | No OCSF/OTel equivalent — macOS provisioning UDID | Convention — system_profiler `provisioning_udid` |
| hardware | CPUType | `cpu_type` | T1 | `cpu_type` | no | OCSF `device_hw_info.cpu_type` — CPU type label (Intel Macs) | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| hardware | ChipType | `chip_type` | T3 | `chip_type` | no | No OCSF/OTel equivalent — Apple Silicon chip name | Convention — system_profiler `chip_type` |
| hardware | CurrentProcessorSpeed | `current_processor_speed` | T3 | `current_processor_speed` | no | No OCSF/OTel equivalent — CPU speed string (Intel Macs) | Convention — system_profiler `current_processor_speed` |
| hardware | NumberProcessors | `number_processors` | T3 | `number_processors` | no | No OCSF/OTel equivalent — processor core count string | Convention — system_profiler `number_processors` |
| hardware | Packages | `packages` | T3 | `packages` | no | No OCSF/OTel equivalent — physical CPU package count | Convention — system_profiler `packages` |
| hardware | L2CacheCore | `l2_cache_core` | T3 | `l2_cache_core` | no | No OCSF/OTel equivalent — per-core L2 cache size string | Convention — system_profiler `l2_cache_core` |
| hardware | L3Cache | `l3_cache` | T3 | `l3_cache` | no | No OCSF/OTel equivalent — L3 cache size string | Convention — system_profiler `l3_cache` |
| hardware | PhysicalMemory | `physical_memory` | T1 | `physical_memory` | no | OCSF `device_hw_info.ram_size` — total physical memory string | [OCSF device_hw_info](https://schema.ocsf.io/1.8.0/objects/device_hw_info) |
| hardware | BootROMVersion | `boot_rom_version` | T3 | `boot_rom_version` | no | No OCSF/OTel equivalent — macOS Boot ROM version | Convention — system_profiler `boot_rom_version` |
| hardware | OSLoaderVersion | `os_loader_version` | T3 | `os_loader_version` | no | No OCSF/OTel equivalent — macOS OS loader version | Convention — system_profiler `os_loader_version` |
| hardware | SMCVersionSystem | `smc_version_system` | T3 | `smc_version_system` | no | No OCSF/OTel equivalent — SMC firmware version | Convention — system_profiler `SMC_version_system` |
| hardware | Storage | `storage` | T3 | `storage` | no | No OCSF/OTel equivalent — attached storage volume array | Convention — system_profiler `SPStorageDataType` |
| hardware | Battery | `battery` | T3 | `battery` | no | No OCSF/OTel equivalent — battery data container | Convention — system_profiler `SPPowerDataType` |
| hardware | Charger | `charger` | T3 | `charger` | no | No OCSF/OTel equivalent — AC charger data container | Convention — system_profiler `SPPowerDataType` |
| hardware.storage[] | Name | `name` | T3 | `name` | no | No OCSF/OTel equivalent — volume display name | Convention — system_profiler `_name` |
| hardware.storage[] | BSDName | `bsd_name` | T3 | `bsd_name` | no | No OCSF/OTel equivalent — BSD device node (disk0s1) | Convention — system_profiler `bsd_name` |
| hardware.storage[] | Capacity | `capacity` | T3 | `capacity` | no | No OCSF/OTel equivalent — volume capacity in bytes | Convention — system_profiler `size_in_bytes` |
| hardware.storage[] | FileSystem | `file_system` | T3 | `file_system` | no | No OCSF/OTel equivalent — filesystem type string | Convention — system_profiler `file_system` |
| hardware.storage[] | MountPoint | `mount_point` | T3 | `mount_point` | no | No OCSF/OTel equivalent — volume mount path | Convention — system_profiler `mount_point` |
| hardware.storage[] | FreeSpace | `free_space` | T3 | `free_space` | no | No OCSF/OTel equivalent — free space in bytes | Convention — system_profiler `free_space_in_bytes` |
| hardware.storage[] | Writable | `writable` | T3 | `writable` | no | No OCSF/OTel equivalent — volume writable flag | Convention — system_profiler `writable` |
| hardware.storage[] | DriveType | `drive_type` | T3 | `drive_type` | no | No OCSF/OTel equivalent — physical drive type (SSD, HDD) | Convention — system_profiler `physical_drive_mediatype` |
| hardware.storage[] | SmartStatus | `smart_status` | T3 | `smart_status` | no | No OCSF/OTel equivalent — S.M.A.R.T. status string | Convention — system_profiler `smart_status` |
| hardware.storage[] | Partitions | `partitions` | T3 | `partitions` | no | No OCSF/OTel equivalent — partition count | Convention — system_profiler `partition_map_type` |
| hardware.battery | CurrentCapacity | `current_capacity` | T3 | `current_capacity` | no | No OCSF/OTel equivalent — current charge level | Convention — system_profiler `sppower_battery_charge_info` |
| hardware.battery | MaxCapacity | `max_capacity` | T3 | `max_capacity` | no | No OCSF/OTel equivalent — maximum charge capacity | Convention — system_profiler `sppower_battery_charge_info` |
| hardware.battery | FullyCharged | `fully_charged` | T3 | `fully_charged` | no | No OCSF/OTel equivalent — battery fully charged flag | Convention — system_profiler `sppower_battery_charge_info` |
| hardware.battery | IsCharging | `is_charging` | T2 | `is_charging` | no | OTel `hw.battery.state` — battery charging state | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/registry.yaml) |
| hardware.battery | ChargeCycleCount | `charge_cycle_count` | T3 | `charge_cycle_count` | no | No OCSF/OTel equivalent — battery charge cycle count | Convention — system_profiler `sppower_battery_health_info` |
| hardware.battery | Health | `health` | T3 | `health` | no | No OCSF/OTel equivalent — battery health status string | Convention — system_profiler `sppower_battery_health_info` |
| hardware.battery | Serial | `serial` | T2 | `serial` | no | OTel `hw.serial_number` — battery serial number | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/registry.yaml) |
| hardware.battery | Remaining | `remaining` | T2 | `remaining` | no | OTel `hw.battery.charge` — remaining charge percentage | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/battery-metrics.yaml) |
| hardware.battery | Amperage | `amperage` | T3 | `amperage` | no | No OCSF/OTel equivalent — battery current in mA | Convention — system_profiler `sppower_battery_charge_info` |
| hardware.battery | Voltage | `voltage` | T3 | `voltage` | no | No OCSF/OTel equivalent — battery voltage in mV | Convention — system_profiler `sppower_battery_charge_info` |
| hardware.charger | ID | `id` | T3 | `id` | no | No OCSF/OTel equivalent — charger identifier | Convention — system_profiler `sppower_ac_charger_ID` |
| hardware.charger | Family | `family` | T3 | `family` | no | No OCSF/OTel equivalent — charger family code | Convention — system_profiler `sppower_ac_charger_family` |
| hardware.charger | Revision | `revision` | T3 | `revision` | no | No OCSF/OTel equivalent — charger firmware revision | Convention — system_profiler `sppower_ac_charger_revision` |
| hardware.charger | SerialNumber | `serial_number` | T3 | `serial_number` | no | No OCSF/OTel equivalent — charger serial number | Convention — system_profiler `sppower_ac_charger_serial_number` |
| hardware.charger | Watts | `watts` | T3 | `watts` | no | No OCSF/OTel equivalent — charger wattage | Convention — system_profiler `sppower_ac_charger_watts` |
| hardware.charger | Connected | `connected` | T3 | `connected` | no | No OCSF/OTel equivalent — charger connected flag | Convention — system_profiler `sppower_ac_charger_connected` |

## Network Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |
| network | Interfaces | `interfaces` | T3 | `interfaces` | no | No OCSF/OTel equivalent — per-interface array container | Convention — gopsutil `net.Interfaces` |
| network | Routes | `routes` | T3 | `routes` | no | No OCSF/OTel equivalent — kernel routing table array | Convention — netlink route dump |
| network | Neighbours | `neighbours` | T3 | `neighbours` | no | No OCSF/OTel equivalent — ARP/NDP neighbour cache array | Convention — netlink neigh dump |
| network | DefaultInterface | `default_interface` | T3 | `default_interface` | no | No OCSF/OTel equivalent — IPv4 default route egress interface | Convention — Ohai `network/default_interface` |
| network | DefaultGateway | `default_gateway` | T3 | `default_gateway` | no | No OCSF/OTel equivalent — IPv4 default gateway address | Convention — Ohai `network/default_gateway` |
| network | DefaultInet6Interface | `default_inet6_interface` | T3 | `default_inet6_interface` | no | No OCSF/OTel equivalent — IPv6 default route egress interface | Convention — Ohai `network/default_inet6_interface` |
| network | DefaultInet6Gateway | `default_inet6_gateway` | T3 | `default_inet6_gateway` | no | No OCSF/OTel equivalent — IPv6 default gateway address | Convention — Ohai `network/default_inet6_gateway` |
| network.interfaces[] | Name | `name` | T1 | `name` | no | OCSF `network_interface.name` — interface name | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.interfaces[] | Number | `number` | T1 | `number` | no | OCSF `network_interface.uid` — unique interface index | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.interfaces[] | State | `state` | T2 | `state` | no | OTel `hw.network.up` — admin state ("up" / "down") | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/network-metrics.yaml) |
| network.interfaces[] | MTU | `mtu` | T3 | `mtu` | no | No OCSF/OTel equivalent — maximum transmission unit | Convention — gopsutil `InterfaceStat.MTU` |
| network.interfaces[] | HardwareAddr | `hardware_addr` | T1 | `hardware_addr` | no | OCSF `network_interface.mac` — MAC address | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.interfaces[] | Encapsulation | `encapsulation` | T1 | `encapsulation` | no | OCSF `network_interface.type` — link layer type (Ethernet, Loopback, etc.) | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.interfaces[] | Driver | `driver` | T3 | `driver` | no | No OCSF/OTel equivalent — sysfs driver name (e1000e, virtio_net) | Convention — /sys/class/net/*/device/driver |
| network.interfaces[] | Speed | `speed` | T2 | `speed` | no | OTel `hw.network.bandwidth.limit` — link speed string | [OTel hardware](https://github.com/open-telemetry/semantic-conventions/blob/main/model/hardware/network-metrics.yaml) |
| network.interfaces[] | Duplex | `duplex` | T3 | `duplex` | no | No OCSF/OTel equivalent — link duplex (half / full / unknown) | Convention — ghw `NIC.Duplex` |
| network.interfaces[] | Flags | `flags` | T3 | `flags` | no | No OCSF/OTel equivalent — interface flag set | Convention — gopsutil `InterfaceStat.Flags` |
| network.interfaces[] | Addresses | `addresses` | T3 | `addresses` | no | No OCSF/OTel equivalent — per-address array container | Convention — gopsutil `InterfaceStat.Addrs` |
| network.interfaces[] | Routes | `routes` | T3 | `routes` | no | No OCSF/OTel equivalent — per-interface route array | Convention — netlink route dump |
| network.interfaces[] | Counters | `counters` | T3 | `counters` | no | No OCSF/OTel equivalent — I/O counter container | Convention — gopsutil `IOCountersStat` |
| network.interfaces[] | Ethtool | `ethtool` | T3 | `ethtool` | no | No OCSF/OTel equivalent — ethtool data container (Linux only) | Convention — Ohai `network/interfaces/*/ethtool` |
| network.interfaces[] | VLAN | `vlan` | T3 | `vlan` | no | No OCSF/OTel equivalent — VLAN sub-interface data container | Convention — Ohai `network/interfaces/*/vlan` |
| network.interfaces[] | TunnelInfo | `tunnel_info` | T3 | `tunnel_info` | no | No OCSF/OTel equivalent — IP tunnel metadata container | Convention — Ohai `network/interfaces/*/tunnel_info` |
| network.interfaces[] | XDP | `xdp` | T3 | `xdp` | no | No OCSF/OTel equivalent — XDP program info container | Convention — Ohai `network/interfaces/*/xdp` |
| network.interfaces[].addresses[] | Addr | `addr` | T1 | `addr` | no | OCSF `network_interface.ip` — IP address | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.interfaces[].addresses[] | Family | `family` | T3 | `family` | no | No OCSF/OTel equivalent — address family (inet / inet6) | Convention — Ohai address `family` |
| network.interfaces[].addresses[] | Prefixlen | `prefixlen` | T1 | `prefixlen` | no | OCSF `network_interface.subnet_prefix` — CIDR prefix length | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.interfaces[].addresses[] | Netmask | `netmask` | T3 | `netmask` | no | No OCSF/OTel equivalent — IPv4 dotted-decimal netmask | Convention — Ohai address `netmask` |
| network.interfaces[].addresses[] | Broadcast | `broadcast` | T3 | `broadcast` | no | No OCSF/OTel equivalent — IPv4 broadcast address | Convention — Ohai address `broadcast` |
| network.interfaces[].addresses[] | Scope | `scope` | T3 | `scope` | no | No OCSF/OTel equivalent — address scope (Global / Link / Host) | Convention — Ohai address `scope` |
| network.interfaces[].counters | BytesSent | `bytes_sent` | T2 | `bytes_sent` | no | OTel `system.network.io` with `direction=transmit` — bytes transmitted | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | BytesRecv | `bytes_recv` | T2 | `bytes_recv` | no | OTel `system.network.io` with `direction=receive` — bytes received | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | PacketsSent | `packets_sent` | T2 | `packets_sent` | no | OTel `system.network.packet.count` with `direction=transmit` — packets transmitted | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | PacketsRecv | `packets_recv` | T2 | `packets_recv` | no | OTel `system.network.packet.count` with `direction=receive` — packets received | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | Errin | `errin` | T2 | `errin` | no | OTel `system.network.errors` with `direction=receive` — receive errors | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | Errout | `errout` | T2 | `errout` | no | OTel `system.network.errors` with `direction=transmit` — transmit errors | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | Dropin | `dropin` | T2 | `dropin` | no | OTel `system.network.packet.dropped` with `direction=receive` — receive drops | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].counters | Dropout | `dropout` | T2 | `dropout` | no | OTel `system.network.packet.dropped` with `direction=transmit` — transmit drops | [OTel system](https://github.com/open-telemetry/semantic-conventions/blob/main/model/system/metrics.yaml) |
| network.interfaces[].ethtool | DriverInfo | `driver_info` | T3 | `driver_info` | no | No OCSF/OTel equivalent — `ethtool -i` driver info map | Convention — Ohai `ethtool/driver_info` |
| network.interfaces[].ethtool | RingParams | `ring_params` | T3 | `ring_params` | no | No OCSF/OTel equivalent — `ethtool -g` ring parameters map | Convention — Ohai `ethtool/ring_params` |
| network.interfaces[].ethtool | ChannelParams | `channel_params` | T3 | `channel_params` | no | No OCSF/OTel equivalent — `ethtool -l` channel parameters map | Convention — Ohai `ethtool/channel_params` |
| network.interfaces[].ethtool | CoalesceParams | `coalesce_params` | T3 | `coalesce_params` | no | No OCSF/OTel equivalent — `ethtool -c` coalesce parameters map | Convention — Ohai `ethtool/coalesce_params` |
| network.interfaces[].ethtool | OffloadParams | `offload_params` | T3 | `offload_params` | no | No OCSF/OTel equivalent — `ethtool -k` offload parameters map | Convention — Ohai `ethtool/offload_params` |
| network.interfaces[].ethtool | PauseParams | `pause_params` | T3 | `pause_params` | no | No OCSF/OTel equivalent — `ethtool -a` pause parameters map | Convention — Ohai `ethtool/pause_params` |
| network.interfaces[].vlan | ID | `id` | T3 | `id` | no | No OCSF/OTel equivalent — 802.1Q VLAN tag | Convention — Ohai `vlan/id` |
| network.interfaces[].vlan | Protocol | `protocol` | T3 | `protocol` | no | No OCSF/OTel equivalent — VLAN protocol (802.1Q / 802.1ad) | Convention — Ohai `vlan/protocol` |
| network.interfaces[].vlan | Flags | `flags` | T3 | `flags` | no | No OCSF/OTel equivalent — VLAN flag set | Convention — Ohai `vlan/flags` |
| network.interfaces[].tunnel_info | Proto | `proto` | T3 | `proto` | no | No OCSF/OTel equivalent — tunnel protocol (any / ipip6 / ip6ip6) | Convention — Ohai `tunnel_info/proto` |
| network.interfaces[].tunnel_info | External | `external` | T3 | `external` | no | No OCSF/OTel equivalent — tunnel external flag | Convention — Ohai `tunnel_info/external` |
| network.interfaces[].tunnel_info | Remote | `remote` | T3 | `remote` | no | No OCSF/OTel equivalent — tunnel remote endpoint address | Convention — Ohai `tunnel_info/remote` |
| network.interfaces[].tunnel_info | Local | `local` | T3 | `local` | no | No OCSF/OTel equivalent — tunnel local endpoint address | Convention — Ohai `tunnel_info/local` |
| network.interfaces[].tunnel_info | EncapLimit | `encaplimit` | T3 | `encaplimit` | no | No OCSF/OTel equivalent — tunnel encapsulation limit | Convention — Ohai `tunnel_info/encaplimit` |
| network.interfaces[].tunnel_info | HopLimit | `hoplimit` | T3 | `hoplimit` | no | No OCSF/OTel equivalent — tunnel hop limit | Convention — Ohai `tunnel_info/hoplimit` |
| network.interfaces[].tunnel_info | TClass | `tclass` | T3 | `tclass` | no | No OCSF/OTel equivalent — tunnel traffic class | Convention — Ohai `tunnel_info/tclass` |
| network.interfaces[].tunnel_info | Flowlabel | `flowlabel` | T3 | `flowlabel` | no | No OCSF/OTel equivalent — tunnel IPv6 flow label | Convention — Ohai `tunnel_info/flowlabel` |
| network.interfaces[].tunnel_info | AddrGenMode | `addrgenmode` | T3 | `addrgenmode` | no | No OCSF/OTel equivalent — address generation mode | Convention — Ohai `tunnel_info/addrgenmode` |
| network.interfaces[].tunnel_info | NumTxQueues | `numtxqueues` | T3 | `numtxqueues` | no | No OCSF/OTel equivalent — transmit queue count | Convention — Ohai `tunnel_info/numtxqueues` |
| network.interfaces[].tunnel_info | NumRxQueues | `numrxqueues` | T3 | `numrxqueues` | no | No OCSF/OTel equivalent — receive queue count | Convention — Ohai `tunnel_info/numrxqueues` |
| network.interfaces[].tunnel_info | GsoMaxSize | `gso_max_size` | T3 | `gso_max_size` | no | No OCSF/OTel equivalent — GSO maximum segment size | Convention — Ohai `tunnel_info/gso_max_size` |
| network.interfaces[].tunnel_info | GsoMaxSegs | `gso_max_segs` | T3 | `gso_max_segs` | no | No OCSF/OTel equivalent — GSO maximum segment count | Convention — Ohai `tunnel_info/gso_max_segs` |
| network.interfaces[].xdp | Attached | `attached` | T3 | `attached` | no | No OCSF/OTel equivalent — attached XDP program array | Convention — Ohai `xdp/attached` |
| network.interfaces[].xdp.attached[] | Mode | `mode` | T3 | `mode` | no | No OCSF/OTel equivalent — XDP mode (xdpdrv / xdpgeneric / xdpoffload) | Convention — Ohai `xdp/mode` |
| network.interfaces[].xdp.attached[] | ID | `id` | T3 | `id` | no | No OCSF/OTel equivalent — eBPF program ID | Convention — Ohai `xdp/id` |
| network.interfaces[].xdp.attached[] | Tag | `tag` | T3 | `tag` | no | No OCSF/OTel equivalent — eBPF program tag | Convention — Ohai `xdp/tag` |
| network.routes[] | Destination | `destination` | T3 | `destination` | no | No OCSF/OTel equivalent — route destination CIDR | Convention — Ohai `network/routes/destination` |
| network.routes[] | Family | `family` | T3 | `family` | no | No OCSF/OTel equivalent — address family (inet / inet6) | Convention — Ohai `network/routes/family` |
| network.routes[] | Gateway | `gateway` | T3 | `gateway` | no | No OCSF/OTel equivalent — route next-hop gateway | Convention — Ohai `network/routes/gateway` |
| network.routes[] | Interface | `interface` | T3 | `interface` | no | No OCSF/OTel equivalent — route egress interface | Convention — Ohai `network/routes/interface` |
| network.routes[] | Source | `source` | T3 | `source` | no | No OCSF/OTel equivalent — route source address | Convention — Ohai `network/routes/source` |
| network.routes[] | Scope | `scope` | T3 | `scope` | no | No OCSF/OTel equivalent — route scope (link / global / host) | Convention — Ohai `network/routes/scope` |
| network.routes[] | Proto | `proto` | T3 | `proto` | no | No OCSF/OTel equivalent — route protocol origin (kernel / boot / static) | Convention — Ohai `network/routes/proto` |
| network.routes[] | Metric | `metric` | T3 | `metric` | no | No OCSF/OTel equivalent — route metric / priority | Convention — Ohai `network/routes/metric` |
| network.neighbours[] | Address | `address` | T3 | `address` | no | No OCSF/OTel equivalent — neighbour IPv4/IPv6 address | Convention — `ip neigh` address field |
| network.neighbours[] | Family | `family` | T3 | `family` | no | No OCSF/OTel equivalent — address family (inet / inet6) | Convention — netlink `AF_INET` / `AF_INET6` |
| network.neighbours[] | MAC | `mac` | T1 | `mac` | no | OCSF `network_interface.mac` — neighbour hardware address | [OCSF network_interface](https://schema.ocsf.io/1.8.0/objects/network_interface) |
| network.neighbours[] | Interface | `interface` | T3 | `interface` | no | No OCSF/OTel equivalent — neighbour egress interface | Convention — `ip neigh` dev field |
| network.neighbours[] | State | `state` | T3 | `state` | no | No OCSF/OTel equivalent — NUD state (REACHABLE / STALE / etc.) | Convention — `ip neigh` state field |

## Cloud Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |
| ec2 | InstanceID | `instance_id` | T1 | `instance_id` | no | OCSF `cloud_resource.uid` / OTel `cloud.resource_id` — EC2 instance identity | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| ec2 | InstanceType | `instance_type` | T3 | `instance_type` | no | No OCSF/OTel equivalent — EC2 instance size/shape | Convention — EC2 IMDS `instance-type` |
| ec2 | InstanceLifecycle | `instance_life_cycle` | T3 | `instance_life_cycle` | no | No OCSF/OTel equivalent — on-demand vs spot lifecycle | Convention — EC2 IMDS `instance-life-cycle` |
| ec2 | AMIID | `ami_id` | T3 | `ami_id` | no | No OCSF/OTel equivalent — AMI image identifier | Convention — EC2 IMDS `ami-id` |
| ec2 | AMILaunchIndex | `ami_launch_index` | T3 | `ami_launch_index` | no | No OCSF/OTel equivalent — launch index within batch | Convention — EC2 IMDS `ami-launch-index` |
| ec2 | AMIManifestPath | `ami_manifest_path` | T3 | `ami_manifest_path` | no | No OCSF/OTel equivalent — S3 manifest path for instance-store AMIs | Convention — EC2 IMDS `ami-manifest-path` |
| ec2 | Hostname | `hostname` | T1 | `hostname` | no | OCSF `device.hostname` — instance IMDS hostname | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| ec2 | LocalHostname | `local_hostname` | T3 | `local_hostname` | no | No OCSF/OTel equivalent — private DNS hostname | Convention — EC2 IMDS `local-hostname` |
| ec2 | PublicHostname | `public_hostname` | T3 | `public_hostname` | no | No OCSF/OTel equivalent — public DNS hostname | Convention — EC2 IMDS `public-hostname` |
| ec2 | LocalIPv4 | `local_ipv4` | T3 | `local_ipv4` | no | No OCSF/OTel equivalent — primary private IPv4 address | Convention — EC2 IMDS `local-ipv4` |
| ec2 | LocalIPv4s | `local_ipv4s` | T3 | `local_ipv4s` | no | No OCSF/OTel equivalent — all private IPv4 addresses | Convention — EC2 IMDS `local-ipv4s` |
| ec2 | PublicIPv4 | `public_ipv4` | T3 | `public_ipv4` | no | No OCSF/OTel equivalent — primary public IPv4 address | Convention — EC2 IMDS `public-ipv4` |
| ec2 | MAC | `mac` | T3 | `mac` | no | No OCSF/OTel equivalent — primary ENI MAC address | Convention — EC2 IMDS `mac` |
| ec2 | SecurityGroups | `security_groups` | T3 | `security_groups` | no | No OCSF/OTel equivalent — attached security group names | Convention — EC2 IMDS `security-groups` |
| ec2 | NetworkInterfaces | `network_interfaces` | T3 | `network_interfaces` | no | No OCSF/OTel equivalent — per-ENI metadata map keyed by MAC | Convention — EC2 IMDS `network/interfaces/macs/` |
| ec2 | Region | `region` | T1 | `region` | no | OCSF `cloud.region` / OTel `cloud.region` — AWS region | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| ec2 | AvailabilityZone | `availability_zone` | T1 | `availability_zone` | no | OCSF `cloud.zone` / OTel `cloud.availability_zone` — AWS AZ | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| ec2 | AccountID | `account_id` | T1 | `account_id` | no | OCSF `cloud.account.uid` / OTel `cloud.account.id` — AWS account | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| ec2 | AvailabilityZoneID | `availability_zone_id` | T3 | `availability_zone_id` | no | No OCSF/OTel equivalent — AZ ID (consistent across accounts) | Convention — EC2 IMDS `placement/availability-zone-id` |
| ec2 | GroupName | `group_name` | T3 | `group_name` | no | No OCSF/OTel equivalent — placement group name | Convention — EC2 IMDS `placement/group-name` |
| ec2 | HostID | `host_id` | T3 | `host_id` | no | No OCSF/OTel equivalent — dedicated host ID | Convention — EC2 IMDS `placement/host-id` |
| ec2 | PartitionNumber | `partition_number` | T3 | `partition_number` | no | No OCSF/OTel equivalent — partition placement number | Convention — EC2 IMDS `placement/partition-number` |
| ec2 | KernelID | `kernel_id` | T3 | `kernel_id` | no | No OCSF/OTel equivalent — paravirt kernel ID (legacy) | Convention — EC2 IMDS `kernel-id` |
| ec2 | RamdiskID | `ramdisk_id` | T3 | `ramdisk_id` | no | No OCSF/OTel equivalent — paravirt ramdisk ID (legacy) | Convention — EC2 IMDS `ramdisk-id` |
| ec2 | InstanceAction | `instance_action` | T3 | `instance_action` | no | No OCSF/OTel equivalent — pending instance action | Convention — EC2 IMDS `instance-action` |
| ec2 | SpotInstanceAction | `spot_instance_action` | T3 | `spot_instance_action` | no | No OCSF/OTel equivalent — spot interruption action signal | Convention — EC2 IMDS `spot/instance-action` |
| ec2 | SpotTerminationTime | `spot_termination_time` | T3 | `spot_termination_time` | no | No OCSF/OTel equivalent — spot termination timestamp | Convention — EC2 IMDS `spot/termination-time` |
| ec2 | ServicesDomain | `services_domain` | T3 | `services_domain` | no | No OCSF/OTel equivalent — AWS services DNS domain | Convention — EC2 IMDS `services/domain` |
| ec2 | ServicesPartition | `services_partition` | T1 | `services_partition` | no | OCSF `cloud.cloud_partition` — AWS partition (aws, aws-cn, aws-us-gov) | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) |
| ec2 | ProductCodes | `product_codes` | T3 | `product_codes` | no | No OCSF/OTel equivalent — marketplace product codes | Convention — EC2 IMDS `product-codes` |
| ec2 | PublicKeys | `public_keys` | T3 | `public_keys` | no | No OCSF/OTel equivalent — SSH public keys attached at launch | Convention — EC2 IMDS `public-keys/` |
| ec2 | BlockDeviceMapping | `block_device_mapping` | T3 | `block_device_mapping` | no | No OCSF/OTel equivalent — AMI virtual disk → device path map | Convention — EC2 IMDS `block-device-mapping/` |
| ec2 | ReservationID | `reservation_id` | T3 | `reservation_id` | no | No OCSF/OTel equivalent — EC2 reservation identifier | Convention — EC2 IMDS `reservation-id` |
| ec2 | Profile | `profile` | T3 | `profile` | no | No OCSF/OTel equivalent — instance launch profile | Convention — EC2 IMDS `profile` |
| ec2 | IAMInfo | `iam_info` | T3 | `iam_info` | no | No OCSF/OTel equivalent — IAM instance profile container | Convention — EC2 IMDS `iam/info` |
| ec2 | APIVersion | `api_version` | T3 | `api_version` | no | No OCSF/OTel equivalent — negotiated IMDS API version | Convention — EC2 IMDS version negotiation |
| ec2 | UserData | `user_data` | T3 | `user_data` | no | No OCSF/OTel equivalent — instance user-data (base64 if binary) | Convention — EC2 IMDS `user-data` |
| ec2.network_interfaces[] | DeviceNumber | `device_number` | T3 | `device_number` | no | No OCSF/OTel equivalent — ENI device index | Convention — EC2 IMDS `device-number` |
| ec2.network_interfaces[] | InterfaceID | `interface_id` | T3 | `interface_id` | no | No OCSF/OTel equivalent — ENI interface ID (eni-xxx) | Convention — EC2 IMDS `interface-id` |
| ec2.network_interfaces[] | LocalHostname | `local_hostname` | T3 | `local_hostname` | no | No OCSF/OTel equivalent — per-ENI private DNS hostname | Convention — EC2 IMDS `local-hostname` |
| ec2.network_interfaces[] | LocalIPv4s | `local_ipv4s` | T3 | `local_ipv4s` | no | No OCSF/OTel equivalent — per-ENI private IPv4 addresses | Convention — EC2 IMDS `local-ipv4s` |
| ec2.network_interfaces[] | MAC | `mac` | T3 | `mac` | no | No OCSF/OTel equivalent — ENI MAC address | Convention — EC2 IMDS ENI mac |
| ec2.network_interfaces[] | NetworkCardIndex | `network_card_index` | T3 | `network_card_index` | no | No OCSF/OTel equivalent — network card slot index | Convention — EC2 IMDS `network-card-index` |
| ec2.network_interfaces[] | OwnerID | `owner_id` | T3 | `owner_id` | no | No OCSF/OTel equivalent — ENI owner account ID | Convention — EC2 IMDS `owner-id` |
| ec2.network_interfaces[] | PublicHostname | `public_hostname` | T3 | `public_hostname` | no | No OCSF/OTel equivalent — per-ENI public DNS hostname | Convention — EC2 IMDS `public-hostname` |
| ec2.network_interfaces[] | PublicIPv4s | `public_ipv4s` | T3 | `public_ipv4s` | no | No OCSF/OTel equivalent — per-ENI public IPv4 addresses | Convention — EC2 IMDS `public-ipv4s` |
| ec2.network_interfaces[] | SecurityGroupIDs | `security_group_ids` | T3 | `security_group_ids` | no | No OCSF/OTel equivalent — per-ENI security group IDs | Convention — EC2 IMDS `security-group-ids` |
| ec2.network_interfaces[] | SecurityGroups | `security_groups` | T3 | `security_groups` | no | No OCSF/OTel equivalent — per-ENI security group names | Convention — EC2 IMDS `security-groups` |
| ec2.network_interfaces[] | SubnetID | `subnet_id` | T3 | `subnet_id` | no | No OCSF/OTel equivalent — per-ENI VPC subnet ID | Convention — EC2 IMDS `subnet-id` |
| ec2.network_interfaces[] | SubnetIPv4CIDRBlock | `subnet_ipv4_cidr_block` | T3 | `subnet_ipv4_cidr_block` | no | No OCSF/OTel equivalent — per-ENI subnet IPv4 CIDR | Convention — EC2 IMDS `subnet-ipv4-cidr-block` |
| ec2.network_interfaces[] | SubnetIPv6CIDRBlocks | `subnet_ipv6_cidr_blocks` | T3 | `subnet_ipv6_cidr_blocks` | no | No OCSF/OTel equivalent — per-ENI subnet IPv6 CIDRs | Convention — EC2 IMDS `subnet-ipv6-cidr-blocks` |
| ec2.network_interfaces[] | VPCID | `vpc_id` | T3 | `vpc_id` | no | No OCSF/OTel equivalent — per-ENI VPC ID | Convention — EC2 IMDS `vpc-id` |
| ec2.network_interfaces[] | VPCIPv4CIDRBlock | `vpc_ipv4_cidr_block` | T3 | `vpc_ipv4_cidr_block` | no | No OCSF/OTel equivalent — per-ENI VPC IPv4 CIDR | Convention — EC2 IMDS `vpc-ipv4-cidr-block` |
| ec2.network_interfaces[] | VPCIPv4CIDRBlocks | `vpc_ipv4_cidr_blocks` | T3 | `vpc_ipv4_cidr_blocks` | no | No OCSF/OTel equivalent — per-ENI VPC IPv4 CIDRs | Convention — EC2 IMDS `vpc-ipv4-cidr-blocks` |
| ec2.network_interfaces[] | VPCIPv6CIDRBlocks | `vpc_ipv6_cidr_blocks` | T3 | `vpc_ipv6_cidr_blocks` | no | No OCSF/OTel equivalent — per-ENI VPC IPv6 CIDRs | Convention — EC2 IMDS `vpc-ipv6-cidr-blocks` |
| ec2.network_interfaces[] | IPv6s | `ipv6s` | T3 | `ipv6s` | no | No OCSF/OTel equivalent — per-ENI IPv6 addresses | Convention — EC2 IMDS `ipv6s` |
| ec2.iam_info | Code | `code` | T3 | `code` | no | No OCSF/OTel equivalent — IAM info response code | Convention — EC2 IMDS `iam/info` JSON |
| ec2.iam_info | LastUpdated | `last_updated` | T3 | `last_updated` | no | No OCSF/OTel equivalent — IAM info last-updated timestamp | Convention — EC2 IMDS `iam/info` JSON |
| ec2.iam_info | InstanceProfileArn | `instance_profile_arn` | T3 | `instance_profile_arn` | no | No OCSF/OTel equivalent — instance profile ARN | Convention — EC2 IMDS `iam/info` JSON |
| ec2.iam_info | InstanceProfileID | `instance_profile_id` | T3 | `instance_profile_id` | no | No OCSF/OTel equivalent — instance profile ID | Convention — EC2 IMDS `iam/info` JSON |
| gce | InstanceID | `instance_id` | T1 | `instance_id` | no | OCSF `cloud_resource.uid` / OTel `cloud.resource_id` — GCE instance identity | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| gce | Name | `name` | T1 | `name` | no | OCSF `device.hostname` — instance name (GCE names are unique per project+zone) | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| gce | Hostname | `hostname` | T1 | `hostname` | no | OCSF `device.hostname` — custom hostname if set | [OCSF device](https://schema.ocsf.io/1.8.0/objects/device) |
| gce | CPUPlatform | `cpu_platform` | T3 | `cpu_platform` | no | No OCSF/OTel equivalent — underlying CPU platform (Intel Haswell, etc.) | Convention — GCE metadata `cpuPlatform` |
| gce | MachineType | `machine_type` | T3 | `machine_type` | no | No OCSF/OTel equivalent — GCE machine type (short form) | Convention — GCE metadata `machineType` |
| gce | Image | `image` | T3 | `image` | no | No OCSF/OTel equivalent — source image (short form) | Convention — GCE metadata `image` |
| gce | Description | `description` | T3 | `description` | no | No OCSF/OTel equivalent — instance description text | Convention — GCE metadata `description` |
| gce | Tags | `tags` | T3 | `tags` | no | No OCSF/OTel equivalent — instance network tags | Convention — GCE metadata `tags` |
| gce | Preemptible | `preemptible` | T3 | `preemptible` | no | No OCSF/OTel equivalent — preemptible/spot scheduling flag | Convention — GCE metadata `scheduling.preemptible` |
| gce | AutomaticRestart | `automatic_restart` | T3 | `automatic_restart` | no | No OCSF/OTel equivalent — automatic restart on failure | Convention — GCE metadata `scheduling.automaticRestart` |
| gce | OnHostMaintenance | `on_host_maintenance` | T3 | `on_host_maintenance` | no | No OCSF/OTel equivalent — maintenance event behavior (MIGRATE/TERMINATE) | Convention — GCE metadata `scheduling.onHostMaintenance` |
| gce | MaintenanceEvent | `maintenance_event` | T3 | `maintenance_event` | no | No OCSF/OTel equivalent — current maintenance event signal | Convention — GCE metadata `maintenanceEvent` |
| gce | Zone | `zone` | T1 | `zone` | no | OCSF `cloud.zone` / OTel `cloud.availability_zone` — GCE zone (short form) | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| gce | Region | `region` | T1 | `region` | no | OCSF `cloud.region` / OTel `cloud.region` — GCE region (derived from zone) | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| gce | ProjectID | `project_id` | T1 | `project_id` | no | OCSF `cloud.account.uid` / OTel `cloud.account.id` — GCP project identifier | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) / [OTel cloud](https://github.com/open-telemetry/semantic-conventions/blob/main/model/cloud/registry.yaml) |
| gce | NumericProjectID | `numeric_project_id` | T1 | `numeric_project_id` | no | OCSF `cloud.project_uid` — GCP numeric project ID | [OCSF cloud](https://schema.ocsf.io/1.8.0/objects/cloud) |
| gce | ProjectAttributes | `project_attributes` | T3 | `project_attributes` | no | No OCSF/OTel equivalent — project-level metadata key/value pairs | Convention — GCE metadata `project.attributes` |
| gce | Licenses | `licenses` | T3 | `licenses` | no | No OCSF/OTel equivalent — GCP license IDs attached to the VM | Convention — GCE metadata `licenses` |
| gce | Attributes | `attributes` | T3 | `attributes` | no | No OCSF/OTel equivalent — instance-level metadata key/value pairs | Convention — GCE metadata `instance.attributes` |
| gce | NetworkInterfaces | `network_interfaces` | T3 | `network_interfaces` | no | No OCSF/OTel equivalent — attached VNIC array | Convention — GCE metadata `networkInterfaces` |
| gce | Disks | `disks` | T3 | `disks` | no | No OCSF/OTel equivalent — attached disk array | Convention — GCE metadata `disks` |
| gce | ServiceAccounts | `service_accounts` | T3 | `service_accounts` | no | No OCSF/OTel equivalent — attached service account array | Convention — GCE metadata `serviceAccounts` |
| gce | VirtualClockDriftToken | `virtual_clock_drift_token` | T3 | `virtual_clock_drift_token` | no | No OCSF/OTel equivalent — virtualClock drift detection token | Convention — GCE metadata `virtualClock.driftToken` |
| gce | RemainingCPUTime | `remaining_cpu_time` | T3 | `remaining_cpu_time` | no | No OCSF/OTel equivalent — remaining CPU time before spot preemption | Convention — GCE metadata `remainingCpuTime` |
| gce | PartnerAttributes | `partner_attributes` | T3 | `partner_attributes` | no | No OCSF/OTel equivalent — partner image metadata key/value pairs | Convention — GCE metadata `partnerAttributes` |
| gce.network_interfaces[] | IP | `ip` | T3 | `ip` | no | No OCSF/OTel equivalent — VNIC private IPv4 address | Convention — GCE metadata `networkInterfaces[].ip` |
| gce.network_interfaces[] | MAC | `mac` | T3 | `mac` | no | No OCSF/OTel equivalent — VNIC MAC address | Convention — GCE metadata `networkInterfaces[].mac` |
| gce.network_interfaces[] | Network | `network` | T3 | `network` | no | No OCSF/OTel equivalent — VPC network name (short form) | Convention — GCE metadata `networkInterfaces[].network` |
| gce.network_interfaces[] | Subnetmask | `subnetmask` | T3 | `subnetmask` | no | No OCSF/OTel equivalent — subnet mask | Convention — GCE metadata `networkInterfaces[].subnetmask` |
| gce.network_interfaces[] | Gateway | `gateway` | T3 | `gateway` | no | No OCSF/OTel equivalent — default gateway address | Convention — GCE metadata `networkInterfaces[].gateway` |
| gce.network_interfaces[] | DNSServers | `dns_servers` | T3 | `dns_servers` | no | No OCSF/OTel equivalent — VNIC DNS server list | Convention — GCE metadata `networkInterfaces[].dnsServers` |
| gce.network_interfaces[] | IPAliases | `ip_aliases` | T3 | `ip_aliases` | no | No OCSF/OTel equivalent — alias IP ranges | Convention — GCE metadata `networkInterfaces[].ipAliases` |
| gce.network_interfaces[] | ForwardedIPs | `forwarded_ips` | T3 | `forwarded_ips` | no | No OCSF/OTel equivalent — forwarded IP addresses | Convention — GCE metadata `networkInterfaces[].forwardedIps` |
| gce.network_interfaces[] | TargetInstanceIPs | `target_instance_ips` | T3 | `target_instance_ips` | no | No OCSF/OTel equivalent — target instance IP addresses | Convention — GCE metadata `networkInterfaces[].targetInstanceIps` |
| gce.network_interfaces[] | MTU | `mtu` | T3 | `mtu` | no | No OCSF/OTel equivalent — VNIC MTU | Convention — GCE metadata `networkInterfaces[].mtu` |
| gce.network_interfaces[] | AccessConfigs | `access_configs` | T3 | `access_configs` | no | No OCSF/OTel equivalent — external access config array | Convention — GCE metadata `networkInterfaces[].accessConfigs` |
| gce.network_interfaces[].access_configs[] | ExternalIP | `external_ip` | T3 | `external_ip` | no | No OCSF/OTel equivalent — external/public IP address | Convention — GCE metadata `accessConfigs[].externalIp` |
| gce.network_interfaces[].access_configs[] | Type | `type` | T3 | `type` | no | No OCSF/OTel equivalent — access config type (ONE_TO_ONE_NAT) | Convention — GCE metadata `accessConfigs[].type` |
| gce.disks[] | DeviceName | `device_name` | T3 | `device_name` | no | No OCSF/OTel equivalent — disk device name | Convention — GCE metadata `disks[].deviceName` |
| gce.disks[] | Type | `type` | T3 | `type` | no | No OCSF/OTel equivalent — disk type (PERSISTENT / SCRATCH) | Convention — GCE metadata `disks[].type` |
| gce.disks[] | Mode | `mode` | T3 | `mode` | no | No OCSF/OTel equivalent — disk access mode (READ_WRITE / READ_ONLY) | Convention — GCE metadata `disks[].mode` |
| gce.disks[] | Index | `index` | T3 | `index` | no | No OCSF/OTel equivalent — disk attachment index | Convention — GCE metadata `disks[].index` |
| gce.disks[] | Interface | `interface` | T3 | `interface` | no | No OCSF/OTel equivalent — bus interface (SCSI / NVME) | Convention — GCE metadata `disks[].interface` |
| gce.disks[] | Encrypted | `encrypted` | T3 | `encrypted` | no | No OCSF/OTel equivalent — customer-managed encryption flag | Convention — GCE metadata `disks[].encrypted` |
| gce.service_accounts[] | Key | `key` | T3 | `key` | no | No OCSF/OTel equivalent — service account map key (default/email) | Convention — GCE metadata `serviceAccounts` map key |
| gce.service_accounts[] | Email | `email` | T3 | `email` | no | No OCSF/OTel equivalent — service account email | Convention — GCE metadata `serviceAccounts[].email` |
| gce.service_accounts[] | Aliases | `aliases` | T3 | `aliases` | no | No OCSF/OTel equivalent — service account aliases | Convention — GCE metadata `serviceAccounts[].aliases` |
| gce.service_accounts[] | Scopes | `scopes` | T3 | `scopes` | no | No OCSF/OTel equivalent — OAuth scopes granted | Convention — GCE metadata `serviceAccounts[].scopes` |
| linode | PublicIP | `public_ip` | T3 | `public_ip` | no | No OCSF/OTel equivalent — eth0 public IPv4 address | Convention — Linode host interface detection |
| linode | PrivateIP | `private_ip` | T3 | `private_ip` | no | No OCSF/OTel equivalent — eth0:1 private IPv4 address | Convention — Linode host interface detection |

## Other Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |
