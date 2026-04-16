# gohai Schema Field Analysis

Field-by-field naming decisions for `gohai.schema.json`. For each field:
the chosen name, type, description, and provenance (which corpus source
informed the name, or "gohai standard" when we're defining it).

**Conventions applied throughout:**

- `snake_case` JSON keys.
- Redundant-prefix stripping: `cpu.cpu_count` → `cpu.count`.
- Universal abbreviations kept: `ip`, `mac`, `pid`, `uid`, `gid`, `mtu`,
  `fqdn`, `uuid`, `cidr`, `arn`.
- Sizes in bytes (`uint64`). Durations in seconds (`uint64`).
  Percentages as `0–100` integer. Booleans are real JSON booleans.
- Descriptions are consumer-facing: what the field means, not how it's
  collected.

---

## hostname

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `name` | string | Short hostname (unqualified). | OCSF `device.hostname`, ECS `host.hostname`, OTel `host.name`, Facter `networking.hostname` — 4 sources. Redundant-prefix stripped. |
| `fqdn` | string | Fully qualified domain name. | ECS `host.domain` references FQDN. Facter `networking.fqdn`. Universal term. |
| `domain` | string | DNS domain, derived by stripping hostname from FQDN. | OCSF `device.domain`, ECS, Facter. |
| `machine_name` | string | Raw `hostname` command output (macOS: ComputerName-derived). | osquery `system_info.computer_name`. Keep `machine_name` — it's what Go's `os.Hostname()` returns. |

**Changes from current:** none. All names are already canonical.

---

## platform

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `name` | string | OS distribution name (e.g. `Ubuntu`, `macOS`). | OCSF `os.name`, ECS `os.name`, OTel `os.name`, Facter `os.name` — 4 sources. |
| `version` | string | OS version string (e.g. `24.04`, `15.2`). | OCSF `os.version`, ECS, OTel — 3 sources. |
| `family` | string | OS family (`debian`, `rhel`, `darwin`). | ECS `os.family`, Facter `os.family` — 2 sources. |
| `architecture` | string | CPU architecture the OS was built for (`x86_64`, `arm64`). | ECS `host.architecture`, OTel `host.arch`, k8s `architecture` — 3 sources. |
| `build` | string | OS build identifier. | OCSF `os.build`, OTel `os.build_id`. Keep `build` (shorter). |
| `version_extra` | string | Supplementary version info (macOS `(a)` suffix from `sw_vers -productVersionExtra`). | gohai standard — no corpus equivalent. |

**Changes from current:** none.

---

## kernel

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `name` | string | Kernel name from `uname -s` (`Linux`, `Darwin`). | Facter `kernel`. Universal. |
| `release` | string | Kernel release string from `uname -r`. | OCSF `os.kernel_release` → stripped to `release`. ECS `host.os.kernel`. k8s `kernelVersion`. Facter `kernelrelease`. 4 sources. |
| `version` | string | Kernel version string from `uname -v`. | Facter `kernelversion`. |
| `machine` | string | Hardware identifier from `uname -m` (`x86_64`, `arm64`). | gohai standard. |
| `os` | string | Operating system name from `uname -o` (`GNU/Linux`, `Darwin`). | gohai standard. |
| `processor` | string | Processor type from `uname -p` (typically same as machine). | Facter `processors.isa`. |
| `rosetta_translated` | bool | True when macOS process runs under Rosetta 2 translation. | gohai standard — macOS-specific. |

**Changes from current:** none.

---

## cpu

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `count` | int | Logical CPU count (threads visible to the scheduler). | OCSF `device.cpu_count` → stripped. Facter `processors.count`. osquery `cpu_logical_cores`. 3 sources. |
| `cores` | int | Physical cores across all sockets. | OCSF `device.cpu_cores` → stripped. Facter. 2 sources. |
| `sockets` | int | Physical CPU packages. | osquery `cpu_sockets`. Facter `physicalcount`. |
| `model_name` | string | Human-readable CPU name (e.g. `Intel(R) Xeon(R) Gold 6130`). | OTel `host.cpu.model.name` leaf. osquery `cpu_brand`. |
| `vendor_id` | string | CPU vendor string (`GenuineIntel`, `AuthenticAMD`). | OTel `host.cpu.vendor.id` leaf. |
| `family` | string | CPU family number. | OTel `host.cpu.family`. |
| `model` | string | CPU model number within family. | OTel `host.cpu.model.id` leaf. |
| `stepping` | int | CPU stepping (silicon revision). | OTel `host.cpu.stepping`. |
| `mhz` | float64 | Clock speed in MHz. | OCSF `cpu_speed`. gohai keeps `mhz` — clearer unit. |
| `cache_size` | int | Aggregate cache size in KB (from first logical CPU). | OTel `host.cpu.cache.l2.size` — partial. gohai standard for aggregate. |
| `flags` | []string | CPU feature flags (e.g. `sse4_2`, `avx2`, `aes`). | gohai standard — no schema covers CPU flags as a list. |
| `bogomips` | float64 | Linux BogoMIPS benchmark value. | gohai standard — Linux-specific. |
| `byte_order` | string | Endianness (`Little Endian`, `Big Endian`). | gohai standard. |
| `op_modes` | []string | Supported CPU operation modes (`32-bit`, `64-bit`). | gohai standard — from lscpu. |
| `vulnerabilities` | map[string]string | CPU vulnerability mitigation status from `/sys/devices/system/cpu/vulnerabilities/`. | gohai standard — security inventory. |
| `caches` | object | Per-level cache sizes: `l1d`, `l1i`, `l2`, `l3`, `l4` (string with unit from lscpu). | gohai standard — OTel has `host.cpu.cache.l2.size` but not the full hierarchy. |
| `numa_nodes` | map | NUMA node → CPU index mapping. | gohai standard. |
| `numa_nodes_count` | int | Number of NUMA nodes. | gohai standard. |
| `hypervisor_vendor` | string | Hypervisor vendor from lscpu when running as guest (`KVM`, `VMware`). | gohai standard — feeds virtualization collector. |
| `virtualization_type` | string | Virtualization type from lscpu (`full`, `para`). | gohai standard. |
| `virtualization` | string | Hardware virtualization capability (`VT-x`, `AMD-V`). | gohai standard. |
| `dispatching_mode` | string | s390x CPU dispatching mode. | gohai standard — arch-specific. |
| `address_sizes` | []string | Physical/virtual address sizes. | gohai standard. |
| `cpus_online` | int | Online CPU count from lscpu. | gohai standard. |
| `cpus_offline` | int | Offline CPU count from lscpu. | gohai standard. |
| `cpus` | []CPU | Per-logical-CPU breakdown with vendor_id, family, model, model_name, stepping, physical_id, core_id, cores, mhz, cache_size, flags. | gohai standard — mirrors Ohai's `cpu["0"]` per-core entries. |

**Changes from current:** none needed. The 2 CANONICAL fields (`count`, `cores`) already match. The remaining 20+ fields are gohai-standard because no corpus covers them.

---

## memory

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `total` | uint64 | Total physical RAM in bytes. | OCSF `ram_size`. OTel `system.memory` metric. Facter `memory.system.total`. Rename from `size` to `total` — more intuitive. gohai standard. |
| `free` | uint64 | Free memory in bytes (not reclaimable). | OTel metric dimension. Facter. |
| `available` | uint64 | Available memory in bytes (free + reclaimable). | OTel. Facter. |
| `used` | uint64 | Used memory in bytes. | OTel. Facter. |
| `used_percent` | int | Memory utilization as 0–100 integer. | Facter `capacity`. gohai standard name. |
| `buffers` | uint64 | Buffer cache in bytes. | OTel metric dimension. |
| `cached` | uint64 | Page cache in bytes. | OTel metric dimension. |
| `shared` | uint64 | Shared memory (tmpfs) in bytes. | OTel metric dimension. |
| `active` | uint64 | Active memory in bytes. | gohai standard. |
| `inactive` | uint64 | Inactive memory in bytes. | gohai standard. |
| `wired` | uint64 | Wired (non-purgeable) memory in bytes (macOS). | gohai standard — macOS-specific. |
| `swap` | object | Swap space: `total`, `free`, `used`, `used_percent`, `cached` (all uint64 / int). | Facter `memory.swap`. |
| `hugepages` | object | Hugepage state: `total`, `free`, `reserved`, `surplus`, `size` (all uint64). | gohai standard — Linux-specific. |
| (25+ deep Linux /proc/meminfo fields) | uint64 | Individual /proc/meminfo counters. | gohai standard — no corpus covers these granularly. |

**Changes from current:** rename `size` → `total` (clearer, matches Facter and OTel).

---

## dmi

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `bios.vendor` | string | BIOS firmware vendor. | OCSF `bios_manufacturer`. Facter `dmi.bios.vendor`. 2 sources. |
| `bios.version` | string | BIOS firmware version. | OCSF `bios_ver`. Facter. 2 sources. |
| `bios.date` | string | BIOS release date. | OCSF `bios_date`. Facter `release_date`. |
| `baseboard.vendor` | string | Motherboard vendor. | Facter `dmi.board.manufacturer`. |
| `baseboard.product` | string | Motherboard product name. | Facter `dmi.board.product`. |
| `baseboard.version` | string | Motherboard version. | Facter. |
| `baseboard.serial_number` | string | Motherboard serial. | OCSF, OTel, Facter, osquery — 4 sources. |
| `baseboard.asset_tag` | string | Motherboard asset tag. | Facter. |
| `chassis.vendor` | string | Chassis manufacturer. | Facter. |
| `chassis.type` | string | Chassis type code. | Facter. |
| `chassis.type_description` | string | Chassis type human name. | gohai standard. |
| `chassis.version` | string | Chassis version. | Facter. |
| `chassis.serial_number` | string | Chassis serial. | OCSF, Facter — 2 sources. |
| `chassis.asset_tag` | string | Chassis asset tag. | Facter. |
| `product.vendor` | string | System manufacturer. | Facter `dmi.manufacturer`. osquery `hardware_vendor`. |
| `product.name` | string | System product name. | Facter. osquery `hardware_model`. |
| `product.family` | string | System product family. | gohai standard. |
| `product.version` | string | System product version. | Facter. |
| `product.serial_number` | string | System serial number. | OCSF `serial_number`, OTel `hw.serial_number`, Facter, osquery — 4 sources. |
| `product.uuid` | string | System UUID (SMBIOS type 1). | OCSF `uuid`, Facter, osquery — 3 sources. |
| `product.sku` | string | System SKU identifier. | gohai standard. |

**Changes from current:** none. Already matches Ohai+Facter consensus.

---

## network

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `default_interface` | string | Name of the interface carrying the default route. | Facter `networking.primary`. |
| `default_gateway` | string | IPv4 default gateway address. | gohai standard. |
| `interfaces[].name` | string | Interface name (`eth0`, `en0`). | OCSF, OTel, osquery — 3 sources. |
| `interfaces[].mac` | string | MAC address. | OCSF `network_interface.mac`, OTel `host.mac`, ECS `host.mac`, osquery `mac` — **4 sources. RENAME from `hardware_addr`.** |
| `interfaces[].mtu` | int | Maximum transmission unit. | osquery, Facter. |
| `interfaces[].flags` | []string | Interface flags (`up`, `broadcast`, `multicast`). | gohai standard. |
| `interfaces[].number` | int | Kernel interface index. | gohai standard — Ohai `iface[:number]`. |
| `interfaces[].state` | string | Admin state (`up` / `down`). | gohai standard — Ohai `iface[:state]`. |
| `interfaces[].encapsulation` | string | Link-layer type (`Ethernet`, `Loopback`). | gohai standard — Ohai. |
| `interfaces[].speed` | string | Link speed. | osquery `link_speed`. |
| `interfaces[].duplex` | string | Link duplex (`full`, `half`). | gohai standard. |
| `interfaces[].driver` | string | Kernel driver name. | osquery `interface_details`. |
| `interfaces[].addresses[].ip` | string | IP address. | OCSF `network_interface.ip`. **RENAME from `addr`.** |
| `interfaces[].addresses[].family` | string | Address family (`inet`, `inet6`). | gohai standard. |
| `interfaces[].addresses[].prefixlen` | int | CIDR prefix length. | gohai standard. |
| `interfaces[].addresses[].netmask` | string | IPv4 netmask. | Facter. |
| `interfaces[].addresses[].broadcast` | string | IPv4 broadcast address. | osquery. |
| `interfaces[].addresses[].scope` | string | Address scope (`Global`, `Link`, `Host`). | gohai standard. |
| `interfaces[].counters.*` | uint64 | I/O counters: `bytes_sent`, `bytes_recv`, `packets_sent`, `packets_recv`, `errin`, `errout`, `dropin`, `dropout`. | ECS partial (`host.network.egress/ingress`). gohai standard for full set. |
| `interfaces[].routes[].*` | object | Per-interface routes: `destination`, `gateway`, `family`, `source`, `scope`, `proto`, `metric`. | gohai standard. |
| `interfaces[].neighbours[].*` | object | ARP/NDP neighbours: `address`, `family`, `mac`, `interface`, `state`. | gohai standard. |
| `interfaces[].ethtool.*` | object | Ethtool sub-records: `driver_info`, `ring_params`, `channel_params`, `coalesce_params`, `offload_params`, `pause_params`. | gohai standard — Linux ethtool. |

**Changes from current:**
- **`hardware_addr` → `mac`** (4-source consensus, strongest rename)
- **`addr` → `ip`** (OCSF `network_interface.ip`)

---

## process

| Field | Type | Description | Provenance |
| ----- | ---- | ----------- | ---------- |
| `pid` | int | Process ID. | OCSF, OTel, ECS, osquery — universal. |
| `name` | string | Process name. | OCSF, OTel, ECS, osquery — universal. |
| `command_line` | string | Full command line. | OTel `process.command_line`, ECS `process.command_line` — 2 sources. **RENAME from `cmd_line`.** |
| `state` | string | Process state. | OTel, osquery. |
| `username` | string | Owner username. | gohai standard. |
| `ppid` | int | Parent process ID. | OTel `process.parent_pid`. Keep `ppid` — universally understood abbreviation. |
| `start_time` | int64 | Process start time (Unix epoch seconds). | osquery `start_time`. |
| `count` | int | Total process count. | gohai standard. |

**Changes from current:**
- **`cmd_line` → `command_line`** (OTel + ECS consensus)

---

## Summary of renames

Only **3 field renames** are evidence-based from the corpus:

| Collector | Current | New | Sources |
| --------- | ------- | --- | ------- |
| network | `hardware_addr` | `mac` | OCSF, OTel, ECS, osquery (4) |
| network | `addr` | `ip` | OCSF (1, but clearer) |
| process | `cmd_line` | `command_line` | OTel, ECS (2) |

Plus **1 clarity rename** not from corpus:

| Collector | Current | New | Rationale |
| --------- | ------- | --- | --------- |
| memory | `size` | `total` | Facter + OTel use `total`. `size` is ambiguous (installed vs available?). |

**All other fields retain their current names.** The 69% UNIQUE fields
are now the gohai standard — the canonical names for concepts no prior
schema covered.
