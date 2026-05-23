# OCSF Gap Report

Fields where gohai collects data that OCSF does not currently model. Each entry
is a candidate for an upstream OCSF PR to
[ocsf/ocsf-schema](https://github.com/ocsf/ocsf-schema).

**Generated from:** `schemas/field-mapping.md` (T3 fields)
**OCSF version:** 1.8.0
**Date:** 2026-05-14

---

## Summary

- Total gohai fields: 803
- OCSF-covered (T1): 97
- OTel-covered (T2): 73
- No schema coverage (T3): 633
- OCSF PR candidates: 62 (from T3)
- Excluded from PR candidates: 571 (provider-specific cloud metadata,
  macOS-only system_profiler passthrough, display/human-readable fields,
  internal identifiers, deeply nested provider IMDS passthrough, and
  filesystem-specific formats like btrfs/ZFS)

### Exclusion breakdown

| Reason                                       | Count |
| -------------------------------------------- | ----- |
| Provider-specific cloud IMDS passthrough      | ~370  |
| macOS system_profiler passthrough (hardware)  | ~45   |
| Deeply nested sub-object fields (ethtool, tunnel, XDP, btrfs, ZFS) | ~55 |
| Display/human-readable formatting fields      | 2     |
| Internal identifiers (shard.seed)             | 1     |
| Per-CPU array element fields (mirrors top-level) | 4  |
| os-release URL/metadata fields                | 4     |
| Legacy/niche fields (32-bit highmem, NFS_Unstable, bounce) | ~10 |
| PCI/SCSI device detail (hardware bus enumeration) | ~25 |
| Remaining misc (already narrow-scope)         | ~55   |

---

## `device` -- Host Identity Gaps

### fqdn (`hostname.fqdn`)

- **What:** Fully qualified domain name resolved via DNS (hostname + domain
  joined with a dot)
- **Why OCSF lacks it:** OCSF has `device.hostname` and `device.domain` as
  separate fields. FQDN is derivable from them, but many tools treat the FQDN
  as the canonical host identity (Chef, Puppet, Ansible all key on FQDN).
- **Why it should exist:** Asset inventory and configuration management systems
  identify hosts by FQDN. Having it as a first-class field avoids every consumer
  doing the same `hostname + "." + domain` concatenation with edge-case handling
  for empty domain.
- **OTel precedent:** None directly. OTel `host.name` "may contain what the
  hostname command returns, or the fully qualified hostname" but does not break
  it out as a dedicated attribute.
- **gohai type:** `string` / `"fqdn"`

### timezone_name (`timezone.name`)

- **What:** IANA timezone database name (e.g., "America/New_York",
  "Europe/London")
- **Why OCSF lacks it:** OCSF models time as timestamps with explicit UTC
  offsets in events. Device timezone is a static inventory fact, not an event
  attribute.
- **Why it should exist:** Timezone is essential for security correlation (was
  this login at 3 AM local time?), compliance auditing (are all servers set to
  UTC?), and fleet management. Every major CMDB tracks it.
- **OTel precedent:** None.
- **gohai type:** `string` / `"name"` (under `timezone` collector)

### timezone_offset (`timezone.offset`)

- **What:** Current UTC offset in seconds (e.g., -18000 for EST, 0 for UTC)
- **Why OCSF lacks it:** Same as above -- event-centric schema.
- **Why it should exist:** Enables programmatic timezone math without parsing
  IANA names. Useful for dashboards that group hosts by UTC offset band.
- **OTel precedent:** None.
- **gohai type:** `int` / `"offset"` (under `timezone` collector)

### init_system (`init.name`)

- **What:** Name of the init system (systemd, launchd, upstart, sysvinit)
- **Why OCSF lacks it:** OCSF is event-centric; init system is a static host
  property. The concept doesn't appear in security event schemas.
- **Why it should exist:** Critical for configuration management (which service
  manager to target), vulnerability scanning (systemd-specific CVEs), and
  compliance (CIS benchmarks differ by init system). Present on every Linux and
  macOS host.
- **OTel precedent:** None.
- **gohai type:** `string` / `"name"` (under `init` collector)

### uptime_seconds (`uptime.seconds`)

- **What:** Seconds since last boot (monotonic clock)
- **Why OCSF lacks it:** OCSF has `device.boot_time` as a timestamp. Uptime is
  derivable from `now - boot_time`.
- **Why it should exist:** Pre-computed uptime is a standard fleet metric.
  Consumers shouldn't need to know the collection timestamp to compute it. Node
  exporter, Prometheus, and Datadog all emit uptime as a direct gauge.
- **OTel precedent:** OTel has `system.uptime` as a metric
  (`metric.system.uptime` in seconds), confirming the concept is worth modeling
  directly.
- **gohai type:** `uint64` / `"seconds"` (under `uptime` collector)

### idle_seconds (`uptime.idle_seconds`)

- **What:** Aggregate CPU idle time in seconds across all cores since boot
- **Why OCSF lacks it:** CPU idle is a performance metric, not a security event
  attribute. OCSF doesn't model host-level performance counters.
- **Why it should exist:** Idle time divided by (uptime x cpu_count) gives a
  single-number fleet utilization metric. Useful for capacity planning and
  right-sizing.
- **OTel precedent:** OTel `system.cpu.time` with `state=idle` covers CPU idle
  as a cumulative counter.
- **gohai type:** `uint64` / `"idle_seconds"` (under `uptime` collector)

---

## `device_hw_info` -- Hardware Detail Gaps

### cpu_sockets (`cpu.sockets`)

- **What:** Number of physical CPU sockets/packages in the system
- **Why OCSF lacks it:** OCSF has `cpu_count` (logical) and `cpu_cores`
  (physical cores) but not socket count. Multi-socket awareness is a server
  topology concept OCSF hasn't reached.
- **Why it should exist:** Socket count determines NUMA topology, licensing costs
  (many enterprise licenses are per-socket), and performance characteristics.
  Every hardware inventory tool reports it.
- **OTel precedent:** None directly. OTel has `system.cpu.physical.count` and
  `system.cpu.logical.count` but no socket count.
- **gohai type:** `int` / `"sockets"` (under `cpu` collector)

### cpu_flags (`cpu.flags`)

- **What:** CPU feature flags / instruction set extensions (e.g., "avx2",
  "aes", "ssse3")
- **Why OCSF lacks it:** Feature flags are a hardware inventory detail, not a
  security event attribute. The flag list varies by architecture (x86, ARM,
  s390x).
- **Why it should exist:** Critical for security (does the CPU support AES-NI
  for FIPS compliance?), vulnerability assessment (is the CPU affected by
  speculative execution flaws?), and workload placement (does the host support
  AVX-512 for ML workloads?). Present on both Linux and macOS.
- **OTel precedent:** None.
- **gohai type:** `[]string` / `"flags"` (under `cpu` collector)

### cpu_vulnerabilities (`cpu.vulnerabilities`)

- **What:** Map of CPU hardware vulnerability mitigations and their status from
  `/sys/devices/system/cpu/vulnerabilities/` (e.g., "spectre_v2: Mitigation:
  Retpolines")
- **Why OCSF lacks it:** OCSF models vulnerabilities as events
  (`vulnerability_finding`), not as static hardware properties.
- **Why it should exist:** CPU microarchitectural vulnerabilities (Spectre,
  Meltdown, MDS, MMIO stale data) are permanent hardware properties. Knowing
  mitigation status per-host is essential for fleet security posture. This is
  a cross-platform concept (Linux exposes via sysfs, similar data available on
  other OS).
- **OTel precedent:** None.
- **gohai type:** `map[string]string` / `"vulnerabilities"` (under `cpu`
  collector)

### cpu_bogomips (`cpu.bogomips`)

- **What:** Linux kernel BogoMIPS calibration value
- **Why OCSF lacks it:** A Linux-specific performance approximation metric.
- **Why it should exist:** While imprecise, BogoMIPS is universally present in
  Linux `/proc/cpuinfo` and lscpu output. It serves as a quick relative
  performance indicator for fleet comparison and is used in some capacity
  planning heuristics. Cross-distro (all Linux).
- **OTel precedent:** None.
- **gohai type:** `float64` / `"bogomips"` (under `cpu` collector)

### cpu_byte_order (`cpu.byte_order`)

- **What:** CPU byte order ("Little Endian" or "Big Endian")
- **Why OCSF lacks it:** Byte order is an architectural detail not typically
  relevant to security events.
- **Why it should exist:** Critical for binary compatibility assessment, data
  format validation, and cross-architecture fleet management. Present on both
  Linux (lscpu) and derivable on macOS.
- **OTel precedent:** None.
- **gohai type:** `string` / `"byte_order"` (under `cpu` collector)

### cpu_address_sizes (`cpu.address_sizes`)

- **What:** Physical and virtual address width (e.g., "48 bits physical, 57 bits
  virtual")
- **Why OCSF lacks it:** An architectural detail below OCSF's abstraction level.
- **Why it should exist:** Address space size determines maximum addressable
  memory (relevant for memory-intensive workloads) and has security implications
  (wider virtual address space improves ASLR entropy). Cross-architecture.
- **OTel precedent:** None.
- **gohai type:** `string` / `"address_sizes"` (under `cpu` collector)

### cpu_numa_nodes_count (`cpu.numa_nodes_count`)

- **What:** Number of NUMA (Non-Uniform Memory Access) nodes
- **Why OCSF lacks it:** NUMA topology is a server hardware concept below
  OCSF's abstraction level.
- **Why it should exist:** NUMA topology affects performance of database
  workloads, VMs, and containers. Knowing NUMA node count is essential for
  capacity planning and workload scheduling. Present on all multi-socket Linux
  systems and derivable on macOS.
- **OTel precedent:** None.
- **gohai type:** `int` / `"numa_nodes_count"` (under `cpu` collector)

### cpu_caches (L1d, L1i, L3) (`cpu.caches.l1d`, `cpu.caches.l1i`, `cpu.caches.l3`)

- **What:** Per-level CPU cache sizes (L1 data, L1 instruction, L3 unified).
  OTel covers L2 only.
- **Why OCSF lacks it:** OCSF doesn't model CPU cache at all. OTel only has
  `host.cpu.cache.l2.size`.
- **Why it should exist:** Cache hierarchy is fundamental to performance
  characterization. L1 sizes determine microarchitecture generation; L3 size
  affects shared-workload performance. Both are hardware inventory staples.
  Cross-platform.
- **OTel precedent:** `host.cpu.cache.l2.size` exists. Natural extension to add
  L1d, L1i, L3, and L4.
- **gohai type:** `string` / `"l1d"`, `"l1i"`, `"l3"` (under `cpu.caches`)

### cpu_mhz_max / cpu_mhz_min (`cpu.mhz_max`, `cpu.mhz_min`)

- **What:** Maximum and minimum CPU frequency bounds
- **Why OCSF lacks it:** OCSF has `cpu_speed` (current) but not frequency
  bounds. Frequency scaling is a dynamic hardware property.
- **Why it should exist:** Frequency bounds define the performance envelope.
  Max frequency is used for fleet sizing; min frequency indicates power-saving
  behavior. Both are needed for performance anomaly detection.
- **OTel precedent:** None directly. OTel `system.cpu.frequency` metric exists
  but only measures current frequency.
- **gohai type:** `string` / `"mhz_max"`, `"mhz_min"` (under `cpu` collector)

### cpu_cpus_online / cpu_cpus_offline (`cpu.cpus_online`, `cpu.cpus_offline`)

- **What:** Count of online vs offline logical CPUs
- **Why OCSF lacks it:** OCSF has `cpu_count` (total logical) but not the
  online/offline split.
- **Why it should exist:** CPUs can be hot-plugged or offlined (common in VMs
  and containers). The online count is the actual available compute. A mismatch
  between total and online indicates misconfiguration or active scaling.
- **OTel precedent:** OTel `system.cpu.logical.count` is total; no online/offline
  split.
- **gohai type:** `int` / `"cpus_online"`, `"cpus_offline"` (under `cpu`
  collector)

### memory_active / memory_inactive (`memory.active`, `memory.inactive`)

- **What:** Active and inactive LRU page counts (bytes)
- **Why OCSF lacks it:** OCSF doesn't model memory subsystem counters. OTel has
  states (used, free, cached, buffers) but not active/inactive.
- **Why it should exist:** Active/inactive split is the primary indicator of
  memory pressure on Linux. High inactive with low active means the system has
  headroom; both high means pressure. Every monitoring system (Prometheus
  node_exporter, Datadog, etc.) tracks these. Also present on macOS (via
  vm_stat).
- **OTel precedent:** OTel `system.memory.usage` has `state` enum but the enum
  does not include `active`/`inactive`.
- **gohai type:** `uint64` / `"active"`, `"inactive"` (under `memory` collector)

### memory_dirty (`memory.dirty`)

- **What:** Memory pages waiting to be written back to disk (bytes)
- **Why OCSF lacks it:** A performance/reliability metric, not a security event.
- **Why it should exist:** High dirty page count indicates I/O bottleneck risk
  and potential data loss on sudden power failure. Critical for database and
  storage system monitoring. Linux-specific but universally available via
  `/proc/meminfo`.
- **OTel precedent:** None.
- **gohai type:** `uint64` / `"dirty"` (under `memory` collector)

### memory_mapped (`memory.mapped`)

- **What:** Memory-mapped file pages (bytes)
- **Why OCSF lacks it:** A kernel memory accounting detail.
- **Why it should exist:** Mapped memory indicates file-backed memory usage
  (shared libraries, mmap'd files). Important for understanding application
  memory behavior and detecting anomalous memory-mapped file usage (potential
  security indicator).
- **OTel precedent:** None.
- **gohai type:** `uint64` / `"mapped"` (under `memory` collector)

### memory_commit_limit / memory_committed_as (`memory.commit_limit`, `memory.committed_as`)

- **What:** Kernel overcommit limit and total committed address space (bytes)
- **Why OCSF lacks it:** Kernel memory management internals.
- **Why it should exist:** CommittedAS approaching CommitLimit means the system
  is near the OOM threshold even if `free` memory looks healthy. Critical for
  overcommit-aware environments (databases, JVMs). Linux-specific but standard
  across all distributions.
- **OTel precedent:** None.
- **gohai type:** `uint64` / `"commit_limit"`, `"committed_as"` (under `memory`
  collector)

### memory_swap_cached (`memory.swap.cached`)

- **What:** Swap pages that also remain in the page cache (bytes)
- **Why OCSF lacks it:** OTel covers swap total/used/free but not swap cache.
- **Why it should exist:** Swap cache size indicates how efficiently the kernel
  is managing swapped-back pages. High swap cache means pages were swapped out
  then read back but kept in swap for quick re-eviction. Important for
  performance debugging.
- **OTel precedent:** OTel `system.paging.usage` covers total/used/free but not
  cached. Natural extension.
- **gohai type:** `uint64` / `"cached"` (under `memory.swap`)

### inode counts (`filesystem.mounts[].inodes_total`, `inodes_used`, `inodes_free`, `inodes_used_percent`)

- **What:** Inode allocation and usage per mounted filesystem
- **Why OCSF lacks it:** OCSF doesn't model filesystem metrics at all. OTel has
  filesystem space usage but not inode usage.
- **Why it should exist:** Inode exhaustion is a common production incident --
  a filesystem can have plenty of free space but be unable to create new files
  because all inodes are consumed. Every monitoring system tracks inodes. Both
  Linux and macOS support inode reporting.
- **OTel precedent:** None. OTel `system.filesystem.usage` covers space only.
  Natural extension to add inode variant.
- **gohai type:** `uint64` / `"inodes_total"`, `"inodes_used"`, `"inodes_free"`;
  `float64` / `"inodes_used_percent"`

### filesystem_uuid / filesystem_label (`filesystem.mounts[].uuid`, `filesystem.mounts[].label`)

- **What:** Filesystem UUID and human-assigned label
- **Why OCSF lacks it:** OCSF doesn't model filesystem identity, only network
  interface identity.
- **Why it should exist:** UUID is the stable filesystem identifier that persists
  across device re-enumeration. Critical for asset inventory, mount validation,
  and forensic correlation. Label provides human-readable identification. Both
  are standard `lsblk` / `blkid` output on Linux.
- **OTel precedent:** None.
- **gohai type:** `string` / `"uuid"`, `"label"` (under `filesystem.mounts[]`)

### dmi_baseboard (`dmi.baseboard.*`)

- **What:** Motherboard/baseboard identity: vendor, product, version, serial
  number, asset tag (DMI/SMBIOS Type 2)
- **Why OCSF lacks it:** OCSF has `device_hw_info.bios_*` (Type 0) and chassis
  (Type 3) but skipped baseboard (Type 2). Likely because baseboard data is less
  commonly consumed in security events.
- **Why it should exist:** Baseboard serial number is a hardware fingerprint
  used for licensing, asset tracking, and warranty management. The
  vendor+product combination identifies the motherboard model for firmware
  vulnerability tracking. Standard across all x86 systems (SMBIOS spec).
- **OTel precedent:** None.
- **gohai type:** `struct` with `string` fields: `"vendor"`, `"product"`,
  `"version"`, `"serial_number"`, `"asset_tag"` (under `dmi.baseboard`)

### dmi_product (family, version, sku) (`dmi.product.family`, `dmi.product.version`, `dmi.product.sku`)

- **What:** System product family, version, and SKU (DMI/SMBIOS Type 1 fields
  not in OCSF). OCSF already covers product vendor, name, serial, and UUID.
- **Why OCSF lacks it:** OCSF has the core identity fields
  (`device.model`, `device_hw_info.serial_number`, `device_hw_info.uuid`) but
  not the full SMBIOS Type 1 record.
- **Why it should exist:** Product family groups related models (e.g., "ThinkPad
  T14s" family spans multiple versions). SKU is used in procurement and warranty
  systems. Both are standard SMBIOS fields available on every x86 system.
- **OTel precedent:** None.
- **gohai type:** `string` / `"family"`, `"version"`, `"sku"` (under
  `dmi.product`)

### chassis_type (`dmi.chassis.type`, `dmi.chassis.type_description`)

- **What:** SMBIOS chassis type code and its human-readable description (e.g.,
  1/"Other", 3/"Desktop", 9/"Laptop", 23/"Rack Mount Chassis")
- **Why OCSF lacks it:** OCSF has `device_hw_info.chassis` as a string but
  doesn't break out the SMBIOS type code.
- **Why it should exist:** Chassis type is the primary way to classify hardware
  form factor. Security policies often differ by form factor (laptops require
  disk encryption; rack servers don't). The SMBIOS type code is a well-defined
  integer enum present on every x86 system.
- **OTel precedent:** `hw.enclosure.type` in OTel hardware semconv covers the
  concept but with string values ("Computer", "Storage", "Switch") rather than
  SMBIOS codes.
- **gohai type:** `string` / `"type"`, `"type_description"` (under
  `dmi.chassis`)

### chassis_asset_tag (`dmi.chassis.asset_tag`)

- **What:** Chassis asset tag from SMBIOS Type 3
- **Why OCSF lacks it:** Asset tags are procurement metadata, not security event
  attributes.
- **Why it should exist:** Asset tags are the link between logical inventory
  (what the OS reports) and physical inventory (what procurement/CMDB tracks).
  Present on every x86 system. Used for compliance auditing and hardware
  lifecycle management.
- **OTel precedent:** None.
- **gohai type:** `string` / `"asset_tag"` (under `dmi.chassis`)

---

## `os` -- Operating System Gaps

### os_family (`platform.family`)

- **What:** OS distribution family (e.g., "debian", "rhel", "suse", "arch")
- **Why OCSF lacks it:** OCSF has `os.name` (specific distro) and `os.type`
  (linux/darwin/windows) but no intermediate grouping. Family is a
  gopsutil/Ohai concept.
- **Why it should exist:** Family determines package management, service
  management, and security patch channels. "Is this host in the Debian family?"
  is a fundamental fleet query. Cross-distro concept present on all Linux
  variants.
- **OTel precedent:** None.
- **gohai type:** `string` / `"family"` (under `platform` collector)

### kernel_name (`kernel.name`)

- **What:** Kernel sysname from `uname -s` ("Linux", "Darwin", "FreeBSD")
- **Why OCSF lacks it:** OCSF has `os.type` which overlaps but uses different
  values (lowercase). Kernel sysname is the raw POSIX value.
- **Why it should exist:** The raw uname sysname is the standard way systems
  identify themselves. Used by configuration management tools, compatibility
  checks, and forensic analysis. Present on every POSIX system.
- **OTel precedent:** None directly. OTel `os.type` is an enum with different
  values.
- **gohai type:** `string` / `"name"` (under `kernel` collector)

### kernel_version (`kernel.version`)

- **What:** Full kernel version string from `uname -v` (e.g., "#1 SMP PREEMPT_DYNAMIC
  Wed Mar 12 14:15:24 UTC 2025")
- **Why OCSF lacks it:** OCSF has `os.kernel_release` (uname -r) but not
  uname -v. The version string contains build metadata (compiler, config,
  timestamp).
- **Why it should exist:** The version string reveals kernel build configuration,
  which is security-relevant (is PREEMPT_DYNAMIC enabled? what compiler was
  used?). Used in forensics and compliance. Present on every POSIX system.
- **OTel precedent:** None.
- **gohai type:** `string` / `"version"` (under `kernel` collector)

### kernel_machine (`kernel.machine`)

- **What:** Hardware identifier from `uname -m` (e.g., "x86_64", "aarch64",
  "arm64")
- **Why OCSF lacks it:** OCSF doesn't have a raw uname machine field. The
  concept overlaps with `host.arch` in OTel but uses different string values.
- **Why it should exist:** The raw uname machine value is used in package
  repository selection, binary compatibility checks, and cross-compilation
  targeting. Present on every POSIX system. Different from architecture (uname -m
  returns "x86_64" while OTel `host.arch` uses "amd64").
- **OTel precedent:** `host.arch` covers the concept but with a normalized enum,
  not the raw uname value.
- **gohai type:** `string` / `"machine"` (under `kernel` collector)

### os_release_id (`os_release.id`)

- **What:** Machine-readable OS identifier from os-release(5) `ID` field (e.g.,
  "ubuntu", "fedora", "alpine", "amzn")
- **Why OCSF lacks it:** OCSF has `os.name` (human-readable) but not the
  machine-parseable ID. The os-release(5) spec explicitly separates these.
- **Why it should exist:** The `ID` field is the canonical machine-parseable
  distro identifier used for conditional logic in scripts, Ansible playbooks,
  and package management. Present on every modern Linux distro (systemd
  requirement).
- **OTel precedent:** None.
- **gohai type:** `string` / `"id"` (under `os_release` collector)

### os_release_id_like (`os_release.id_like`)

- **What:** Space-separated list of related OS identifiers from os-release(5)
  `ID_LIKE` field (e.g., "debian" for Ubuntu, "rhel fedora" for Amazon Linux)
- **Why OCSF lacks it:** No equivalent concept in OCSF.
- **Why it should exist:** `ID_LIKE` enables "is this a Debian-family distro?"
  queries without maintaining a manual mapping. Essential for fleet management
  tools that need to group distros by family. Standard os-release(5) field.
- **OTel precedent:** None.
- **gohai type:** `[]string` / `"id_like"` (under `os_release` collector)

### os_release_version_id (`os_release.version_id`)

- **What:** Machine-parseable version number from os-release(5) `VERSION_ID`
  field (e.g., "22.04", "9", "3.18")
- **Why OCSF lacks it:** OCSF `os.version` exists but maps to the human-readable
  `VERSION` field. `VERSION_ID` is the machine-parseable companion.
- **Why it should exist:** `VERSION_ID` is what package managers and version
  comparison logic use. It's the only reliable way to do programmatic version
  comparison ("is this Ubuntu >= 22.04?"). Standard os-release(5) field.
- **OTel precedent:** `os.version` in OTel covers the concept but doesn't
  distinguish human-readable vs machine-parseable.
- **gohai type:** `string` / `"version_id"` (under `os_release` collector)

### os_release_version_codename (`os_release.version_codename`)

- **What:** Release codename from os-release(5) `VERSION_CODENAME` field (e.g.,
  "jammy", "bookworm")
- **Why OCSF lacks it:** Codenames are a distro convention, not a universal OS
  concept.
- **Why it should exist:** Codenames map to specific package repository
  channels. "Is this host on bookworm or trixie?" determines available security
  patches. Standard os-release(5) field on Debian-family and some others.
- **OTel precedent:** None.
- **gohai type:** `string` / `"version_codename"` (under `os_release` collector)

### os_release_variant_id (`os_release.variant_id`)

- **What:** Machine-parseable OS variant from os-release(5) `VARIANT_ID` field
  (e.g., "server", "workstation", "container", "iot")
- **Why OCSF lacks it:** OCSF doesn't differentiate OS variants.
- **Why it should exist:** Variant determines the installed package set, default
  services, and applicable security policies. "Is this a server or workstation
  install?" is a common compliance question. Standard os-release(5) field.
- **OTel precedent:** None.
- **gohai type:** `string` / `"variant_id"` (under `os_release` collector)

### fips_kernel_enabled (`fips.kernel.enabled`)

- **What:** Boolean indicating whether the kernel's FIPS 140 mode is active
  (`/proc/sys/crypto/fips_enabled`)
- **Why OCSF lacks it:** FIPS mode is a US government compliance requirement.
  OCSF's security posture modeling doesn't extend to cryptographic policy.
- **Why it should exist:** FIPS 140 compliance is a hard requirement for
  FedRAMP, FISMA, and DoD environments. Knowing per-host FIPS status is the
  first question in any government compliance audit. Linux-specific but present
  on all enterprise distros (RHEL, Ubuntu Pro, SUSE).
- **OTel precedent:** None.
- **gohai type:** `bool` / `"enabled"` (under `fips.kernel`)

### fips_crypto_policy (`fips.policy.name`, `fips.policy.fips_effective`)

- **What:** System-wide crypto policy name and whether it's FIPS-effective (e.g.,
  policy="FIPS", fips_effective=true)
- **Why OCSF lacks it:** Same as above -- crypto policy is out of scope for
  event-centric OCSF.
- **Why it should exist:** Crypto policy determines which algorithms/protocols
  are allowed system-wide. A host can have kernel FIPS enabled but a non-FIPS
  crypto policy (or vice versa). Both signals are needed for compliance.
  RHEL-family-specific but covers a large enterprise fleet segment.
- **OTel precedent:** None.
- **gohai type:** `string` / `"name"` and `bool` / `"fips_effective"` (under
  `fips.policy`)

### lsb_id / lsb_release / lsb_codename (`lsb.id`, `lsb.release`, `lsb.codename`)

- **What:** Linux Standard Base distributor ID, release number, and codename
  from `lsb_release` or `/etc/lsb-release`
- **Why OCSF lacks it:** LSB is a legacy Linux standard largely superseded by
  os-release. OCSF chose the modern path.
- **Why it should exist:** Many older enterprise systems still rely on LSB for
  distro identification. Some distros (especially older Ubuntu LTS and Debian)
  populate LSB fields with data not in os-release. Backward compatibility for
  existing tooling.
- **OTel precedent:** None.
- **gohai type:** `string` / `"id"`, `"release"`, `"codename"` (under `lsb`
  collector)

---

## `network_interface` -- Network Gaps

### interface_mtu (`network.interfaces[].mtu`)

- **What:** Maximum Transmission Unit for the network interface
- **Why OCSF lacks it:** OCSF models interface identity (name, MAC, IP, type)
  but not configuration parameters.
- **Why it should exist:** MTU misconfiguration is a common cause of network
  issues (jumbo frames, tunnel overhead, PMTUD failures). Security tools need
  MTU to assess fragmentation attack exposure. Present on every OS and interface
  type.
- **OTel precedent:** None.
- **gohai type:** `int` / `"mtu"` (under `network.interfaces[]`)

### interface_driver (`network.interfaces[].driver`)

- **What:** Kernel driver name bound to the interface (e.g., "e1000e",
  "virtio_net", "mlx5_core")
- **Why OCSF lacks it:** Driver is an implementation detail below OCSF's
  abstraction.
- **Why it should exist:** Driver name identifies the hardware/virtual adapter
  type, which is critical for firmware vulnerability tracking, performance
  tuning, and virtual vs physical NIC detection. Linux-specific but also
  available on macOS via IOKit.
- **OTel precedent:** `hw.driver_version` exists in OTel hardware semconv but
  not the driver name itself.
- **gohai type:** `string` / `"driver"` (under `network.interfaces[]`)

### interface_duplex (`network.interfaces[].duplex`)

- **What:** Link duplex mode (full, half, unknown)
- **Why OCSF lacks it:** A physical layer detail below OCSF's abstraction.
- **Why it should exist:** Half-duplex on a Gigabit link indicates
  misconfiguration (auto-negotiation failure). Important for network
  troubleshooting and capacity planning. Available on Linux and macOS for
  physical interfaces.
- **OTel precedent:** None.
- **gohai type:** `string` / `"duplex"` (under `network.interfaces[]`)

### interface_flags (`network.interfaces[].flags`)

- **What:** Interface flag set (e.g., UP, BROADCAST, MULTICAST, PROMISC,
  NOARP)
- **Why OCSF lacks it:** OCSF doesn't model interface operational flags.
- **Why it should exist:** The PROMISC flag indicates promiscuous mode
  (potential packet capture / sniffer). NOARP indicates a point-to-point link.
  Interface flags are a standard POSIX concept present on every OS.
- **OTel precedent:** None.
- **gohai type:** `[]string` / `"flags"` (under `network.interfaces[]`)

### address_family (`network.interfaces[].addresses[].family`)

- **What:** Address family identifier ("inet" for IPv4, "inet6" for IPv6)
- **Why OCSF lacks it:** OCSF has `network_interface.ip` but doesn't
  distinguish the family -- it's implicit from the address format.
- **Why it should exist:** Explicit family tagging simplifies filtering and
  querying without parsing IP address strings. Standard POSIX concept.
- **OTel precedent:** None.
- **gohai type:** `string` / `"family"` (under
  `network.interfaces[].addresses[]`)

### address_scope (`network.interfaces[].addresses[].scope`)

- **What:** Address scope (Global, Link, Host)
- **Why OCSF lacks it:** Address scope is a routing detail not modeled in OCSF.
- **Why it should exist:** Scope determines reachability -- link-local addresses
  are not routable beyond the broadcast domain; host-scope addresses are
  loopback-only. Important for network topology mapping and security zone
  analysis.
- **OTel precedent:** None.
- **gohai type:** `string` / `"scope"` (under
  `network.interfaces[].addresses[]`)

### default_interface / default_gateway (`network.default_interface`, `network.default_gateway`)

- **What:** IPv4 default route egress interface name and gateway address
- **Why OCSF lacks it:** OCSF models per-event source/destination but not the
  host's routing configuration.
- **Why it should exist:** The default gateway and egress interface define a
  host's primary network path. Essential for network topology mapping,
  segmentation verification, and incident response ("which network segment is
  this host on?"). Present on every networked system (Linux, macOS, Windows).
- **OTel precedent:** None.
- **gohai type:** `string` / `"default_interface"`, `"default_gateway"` (under
  `network` collector)

### routes (destination, gateway, interface, scope, proto, metric) (`network.routes[].*`)

- **What:** Kernel routing table entries with destination CIDR, gateway,
  interface, scope, protocol origin, and metric
- **Why OCSF lacks it:** OCSF is event-centric. Routing tables are host
  configuration state, not events.
- **Why it should exist:** Routing tables define reachability and are essential
  for network forensics, segmentation verification, and lateral movement
  analysis. A compromised host's routing table reveals what other networks it
  can reach. Cross-platform concept (Linux netlink, macOS `route`).
- **OTel precedent:** None.
- **gohai type:** `[]struct` with fields `"destination"`, `"gateway"`,
  `"interface"`, `"family"`, `"scope"`, `"proto"`, `"metric"`

### neighbours (address, mac, interface, state) (`network.neighbours[].*`)

- **What:** ARP/NDP neighbour cache entries with IP address, MAC address,
  interface, and NUD state
- **Why OCSF lacks it:** OCSF doesn't model host network state tables.
- **Why it should exist:** The neighbour cache reveals which other hosts are on
  the same broadcast domain -- critical for lateral movement detection and
  network mapping. NUD state (STALE, REACHABLE, FAILED) indicates recent
  communication patterns. Cross-platform concept.
- **OTel precedent:** None.
- **gohai type:** `[]struct` with fields `"address"`, `"family"`, `"mac"`,
  `"interface"`, `"state"`

---

## `cloud` -- Cross-Provider Canonical Gaps

These fields appear across multiple cloud providers under different names, and
represent concepts that OCSF's `cloud` object could standardize.

### instance_type (`ec2.instance_type`, `gce.machine_type`, `azure.vm_size`, `oci.shape`, `openstack.instance_type`, `alibaba.instance_type`, `scaleway.commercial_type`)

- **What:** Cloud provider instance size/shape (e.g., "m5.xlarge", "n2-standard-4",
  "Standard_D4s_v3")
- **Why OCSF lacks it:** OCSF has `cloud_resource.uid` (instance ID) and region/zone
  but not instance type. It's provider-specific in naming but universal in concept.
- **Why it should exist:** Instance type determines CPU/memory/network capacity,
  pricing tier, and available features. It's the most-queried cloud metadata
  field after instance ID and region. Every cloud provider exposes it. OTel has
  `host.type` which maps to this concept.
- **OTel precedent:** `host.type` -- "Type of host. For Cloud, this must be the
  machine type." [OTel host
  semconv](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml)
- **gohai type:** `string` / provider-specific key (varies)

### instance_lifecycle (`ec2.instance_life_cycle`, `gce.preemptible`, `azure.priority`, `alibaba.spot_termination_time`)

- **What:** Whether the instance is on-demand, spot/preemptible, or reserved
- **Why OCSF lacks it:** Lifecycle/scheduling type is a billing/operations
  concept not modeled in OCSF's security-event focus.
- **Why it should exist:** Spot/preemptible instances can be terminated with
  short notice. Security incident response needs to know if a host might
  disappear. Cost optimization tools need lifecycle type for fleet analysis.
  Present across all major providers.
- **OTel precedent:** None.
- **gohai type:** `string` (varies by provider)

### tags (`ec2` via labels, `gce.tags`, `azure.tags_list`, `oci.freeform_tags`, `digital_ocean.tags`, `alibaba.tags`, `scaleway.tags`)

- **What:** User-defined key/value tags or labels attached to the instance
- **Why OCSF lacks it:** Tags are user-defined metadata, not part of the cloud
  resource model in OCSF.
- **Why it should exist:** Tags are the primary mechanism for organizing cloud
  resources. Security policies are often tag-based ("all instances tagged
  env:production must have encryption enabled"). Asset inventory without tags is
  incomplete. Universal across all cloud providers.
- **OTel precedent:** None as a structured object, though OTel allows arbitrary
  resource attributes.
- **gohai type:** varies (`[]string`, `map[string]string`, structured array)

---

## New Object Candidates -- Concepts Not in OCSF

These represent entirely new OCSF object types that would be needed to model
gohai's output. They are larger proposals than field additions to existing
objects.

### `kernel_module` -- Loaded Kernel Modules

### kernel_modules (`kernel_modules.modules`)

- **What:** Map of loaded kernel modules with name, size, reference count, and
  version. On Linux: `/proc/modules` + `/sys/module/*/version`. On macOS:
  kextstat output.
- **Why OCSF lacks it:** OCSF models processes but not kernel modules. Modules
  are a host inventory concept, not an event.
- **Why it should exist:** Loaded kernel modules are a primary indicator of host
  capability and attack surface. Rootkits often load as kernel modules. Security
  tools need to enumerate loaded modules for integrity verification. CIS
  benchmarks check for specific modules (e.g., "ensure cramfs module is not
  loaded"). Cross-platform (Linux modules, macOS kexts).
- **OTel precedent:** None.
- **gohai type:** `map[string]struct` with fields `"size"` (int64),
  `"refcount"` (int), `"version"` (string)

### `virtualization` -- Hypervisor / Container Detection

### virtualization_system (`virtualization.system`)

- **What:** Primary detected virtualization system name (e.g., "kvm", "vmware",
  "docker", "lxc", "hyper-v")
- **Why OCSF lacks it:** OCSF has `device.hypervisor` (mapped to our T1 field)
  but the `virtualization` collector also reports role and multi-layer detection
  which OCSF doesn't model.
- **Why it should exist:** Multi-layer virtualization is common (VM inside a
  container). Knowing all detected layers (not just the primary) is essential
  for security context -- a container on a VM has different isolation properties
  than bare metal.
- **OTel precedent:** None for multi-layer detection.
- **gohai type:** `string` / `"system"` (T1, already mapped); `string` /
  `"role"`; `map[string]string` / `"systems"` (T3 gaps)

### virtualization_role (`virtualization.role`)

- **What:** Whether the host is a virtualization "host" or "guest"
- **Why OCSF lacks it:** OCSF doesn't distinguish host vs guest role.
- **Why it should exist:** Role determines security responsibilities -- a
  hypervisor host has different patching requirements and attack surface than a
  guest. Essential for compliance scoping.
- **OTel precedent:** None.
- **gohai type:** `string` / `"role"` (under `virtualization` collector)

### `session` -- Login Sessions

### sessions_terminal / sessions_host (`sessions.session.terminal`, `sessions.session.host`)

- **What:** TTY/pts terminal name and remote host for active login sessions
- **Why OCSF lacks it:** OCSF has `session.uid` and `session.user` but not the
  terminal/host detail from utmp. OCSF models sessions in the context of
  authentication events, not as inventory.
- **Why it should exist:** Active session enumeration is fundamental to
  security monitoring -- "who is logged in from where?" is the first question
  in incident response. Terminal name indicates local vs remote access. Remote
  host enables lateral movement tracking. POSIX utmp is cross-platform (Linux
  and macOS).
- **OTel precedent:** None.
- **gohai type:** `string` / `"terminal"`, `"host"` (under `sessions.session`)

### session_seat (`sessions.session.seat`)

- **What:** Systemd seat assignment for the session
- **Why OCSF lacks it:** Systemd seats are a Linux-specific concept.
- **Why it should exist:** Seat assignment distinguishes physical console access
  from remote access in systemd environments. Security-relevant for physical
  access policies.
- **OTel precedent:** None.
- **gohai type:** `string` / `"seat"` (under `sessions.session`)

### `user` -- User Account Inventory

### users_passwd (`users.passwd.*`)

- **What:** User account inventory from /etc/passwd: UID, GID, home directory,
  login shell, GECOS comment
- **Why OCSF lacks it:** OCSF `user` object has `uid` and `name` for
  event-context user references but not full account inventory.
- **Why it should exist:** User account enumeration is a security baseline --
  CIS benchmarks check for accounts with UID 0 (root equivalents), accounts
  without passwords, accounts with shells set to /bin/false, etc. Essential for
  identity governance and access management.
- **OTel precedent:** None.
- **gohai type:** per-user struct with `"uid"` (int), `"gid"` (int), `"dir"`
  (string), `"shell"` (string), `"gecos"` (string)

### users_group (`users.group.*`)

- **What:** Group account inventory from /etc/group: GID and member list
- **Why OCSF lacks it:** OCSF `group` object exists but for event-context
  references, not inventory.
- **Why it should exist:** Group membership determines file permissions and
  sudo/privilege escalation paths. Auditing group membership is a core security
  practice. Cross-platform (/etc/group on Linux and macOS).
- **OTel precedent:** None.
- **gohai type:** per-group struct with `"gid"` (int), `"members"` ([]string)

### `process_list` -- Process Inventory

### process_count (`process.count`)

- **What:** Total number of running processes at collection time
- **Why OCSF lacks it:** OCSF models individual process events, not process
  census.
- **Why it should exist:** Process count is a basic system health metric. A
  sudden spike can indicate a fork bomb or runaway service. A baseline process
  count is useful for anomaly detection.
- **OTel precedent:** OTel `system.process.count` metric exists with `status`
  dimension.
- **gohai type:** `int` / `"count"` (under `process` collector)

### `load_average` -- System Load

### load_averages (`load.one`, `load.five`, `load.fifteen`)

- **What:** 1-minute, 5-minute, and 15-minute load averages from getloadavg(3)
- **Why OCSF lacks it:** Load averages are a performance metric, not a security
  event.
- **Why it should exist:** Load averages are the most widely used system health
  indicator on Unix systems. They appear in every monitoring dashboard,
  node_exporter, and capacity planning tool. A load average exceeding CPU count
  indicates saturation. Cross-platform (Linux and macOS).
- **OTel precedent:** None as a dedicated semconv, but universally collected by
  OTel collectors in practice.
- **gohai type:** `float64` / `"one"`, `"five"`, `"fifteen"` (under `load`
  collector)

### `package_manager` -- Software Management

### package_manager_path (`package_mgr.path`)

- **What:** Absolute filesystem path to the active package manager binary
- **Why OCSF lacks it:** OCSF `package` object models installed packages, not
  the package manager itself.
- **Why it should exist:** The package manager binary path confirms which package
  management system is active and can be verified against expected paths for
  integrity checking.
- **OTel precedent:** None.
- **gohai type:** `string` / `"path"` (under `package_mgr` collector)

### `shells` -- Available Login Shells

### shells_paths (`shells.paths`)

- **What:** List of valid login shell paths from /etc/shells
- **Why OCSF lacks it:** Available shells are a host configuration detail not
  modeled in OCSF.
- **Why it should exist:** The list of valid shells determines which programs
  can be set as login shells. Security audits check for unexpected entries (a
  penetration testing tool's shell in /etc/shells). CIS benchmarks verify this
  file. Cross-platform (Linux and macOS).
- **OTel precedent:** None.
- **gohai type:** `[]string` / `"paths"` (under `shells` collector)

### `root_group` -- Root Group Name

### root_group_name (`root_group.name`)

- **What:** Name of the root user's primary group ("root" on Linux, "wheel" on
  macOS/BSD)
- **Why OCSF lacks it:** A very narrow host fact.
- **Why it should exist:** Configuration management tools need to know the
  correct root group name for file ownership. This differs between OS families
  and is a common source of cross-platform bugs.
- **OTel precedent:** None.
- **gohai type:** `string` / `"name"` (under `root_group` collector)

## New Objects (from new collectors)

### SELinux Security Posture (`selinux.status`, `selinux.current_mode`)

- **What:** SELinux enabled/disabled status and runtime enforcement mode
  (enforcing/permissive/disabled).
- **Why OCSF lacks it:** OCSF models security events but does not model host
  security configuration state.
- **Why it should exist:** SELinux is a major Linux Mandatory Access Control
  framework. Knowing whether a host is in enforcing, permissive, or disabled
  mode is fundamental to security posture assessment. This affects vulnerability
  management, compliance (CIS benchmarks), and incident response.
- **OTel precedent:** None. OTel does not model host security posture.
- **gohai types:** `string` / `"status"`, `string` / `"current_mode"`,
  `string` / `"config_mode"`, `string` / `"policy_version"`,
  `string` / `"loaded_policy_name"` (under `selinux` collector)

### SSH Host Key (`ssh.keys[]`)

- **What:** SSH host key algorithm, fingerprint (SHA-256 and MD5), and key
  length for each key type (RSA, ECDSA, Ed25519).
- **Why OCSF lacks it:** OCSF has a `tls` object for TLS certificate
  fingerprints in network events, but no equivalent for SSH host keys.
- **Why it should exist:** SSH host keys are a critical part of host identity
  in infrastructure. Key rotation, algorithm deprecation (RSA < 2048, DSA),
  and fingerprint verification are security-relevant operations. An
  `ssh_host_key` object would complement the existing `tls` object.
- **OTel precedent:** None.
- **gohai types:** `string` / `"type"`, `string` / `"fingerprint_sha256"`,
  `string` / `"fingerprint_md5"`, `int` / `"key_length"` (under `ssh`
  collector)

### Container Inventory (`docker.containers[]`)

- **What:** Docker container inventory with container ID, name, image, state,
  and status.
- **Why OCSF lacks it:** OCSF has a `container` profile extension but it
  describes the container *context* of a security event, not host-level
  container inventory.
- **Why it should exist:** Container inventory is increasingly important for
  asset management and vulnerability scanning. Knowing which containers are
  running on a host is a security posture signal.
- **OTel precedent:** OTel has `container.id`, `container.name`,
  `container.image.name`, `container.image.tag` — but scoped to per-container
  resource identity, not host-level inventory.
- **gohai types:** `string` / `"id"`, `string` / `"name"`,
  `string` / `"image"`, `string` / `"state"`, `string` / `"status"` (under
  `docker` collector)
