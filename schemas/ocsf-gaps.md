# OCSF Gap Report

Fields where gohai collects data that OCSF does not currently model. Each entry
is a candidate for an upstream OCSF PR to
[ocsf/ocsf-schema](https://github.com/ocsf/ocsf-schema).

**Generated from:** full codebase audit of all 1303 JSON fields across 62
collectors
**OCSF version:** 1.8.0
**Date:** 2026-05-23

---

## Summary

| Metric                           | Count    |
| -------------------------------- | -------- |
| Total gohai fields (from source) | 1303     |
| OCSF-covered (T1)                | ~97      |
| OTel-covered (T2)                | ~73      |
| No schema coverage (T3)          | ~1133    |
| **OCSF gap candidates**          | **~169** |
| Excluded (non-security)          | ~964     |

### Gap candidates by OCSF object

| OCSF Object                          | Candidates |
| ------------------------------------ | ---------- |
| `device` (host identity/state)       | 11         |
| `device_hw_info` (hardware)          | 27         |
| `os` (operating system)              | 16         |
| `network_interface` (network config) | 12         |
| `cloud` (cross-provider)             | 14         |
| `process` (extend existing)          | 1          |
| `package` (extend existing)          | 1          |
| `user` / `group` (extend existing)   | 7          |
| `session` (extend existing)          | 3          |
| New: `kernel_module`                 | 4          |
| New: `routing_table`                 | 7          |
| New: `neighbour_cache`               | 5          |
| New: `security_posture`              | 10         |
| New: `load_average`                  | 3          |
| New: `container_inventory`           | 5          |
| New: `sysctl_params`                 | 1          |
| New: `memory_info`                   | 6          |
| New: `filesystem`                    | 6          |
| New: `ipc_limits`                    | 9          |
| Other (shells, root_group, block)    | 15         |

### Top 20 OCSF upstream PR candidates (ranked)

1. `device_hw_info.cpu_vulnerabilities` — kernel-reported Spectre/Meltdown/MDS mitigations
2. New `kernel_module` object — loaded kernel modules (rootkit detection, CIS benchmarks)
3. New `security_posture` object — FIPS mode, SELinux status/mode/policy
4. `device_hw_info.cpu_flags` — security-relevant CPU feature flags (aes, sev, sgx, nx)
5. `network_interface.flags` — interface flags including PROMISC detection
6. `cloud.security_groups` — cloud firewall group membership
7. `cloud.iam_role` / `cloud.service_accounts` — IAM bindings
8. New `ssh_host_key` object — host key type, fingerprint, length
9. New `routing_entry` object — destination, gateway, interface, metric
10. New `neighbor_entry` object — ARP/NDP table entries
11. `cloud.vpc_id` / `cloud.subnet_id` — network segmentation identifiers
12. `cloud.encryption_at_host` — data-at-rest encryption status
13. `device_hw_info.cpu_vendor_id` — OTel `host.cpu.vendor.id` promotion
14. `device_hw_info.cpu_family` — OTel `host.cpu.family` promotion
15. `device_hw_info.cpu_model_id` — OTel `host.cpu.model.id` promotion
16. `device_hw_info.cpu_stepping` — OTel `host.cpu.stepping` promotion
17. `os.distribution_id` — machine-parseable distro identifier
18. `os.distribution_family` — parent distro lineage (id_like)
19. `session.terminal` — terminal device for login sessions
20. `session.remote_host` — remote origin of login sessions

### OTel precedent candidates

These have the highest chance of OCSF acceptance because OTel has already
established the concept:

- `instance_type` — OTel `host.type`
- `cpu_vendor_id` / `cpu_family` / `cpu_model_id` / `cpu_stepping` — OTel `host.cpu.*`
- `cpu_caches` L1/L3 — OTel has L2 via `host.cpu.cache.l2.size`
- `uptime_seconds` — OTel `system.uptime`
- `process_count` — OTel `system.process.count`
- `interface_speed` — OTel `hw.network.bandwidth.limit`
- `boot_rom_version` — OTel `hw.firmware_version`
- `chassis_type` — OTel `hw.enclosure.type`
- `pci.driver` — OTel `hw.driver_version`

### Excluded from candidates (~1006 fields)

| Reason                                                | Count |
| ----------------------------------------------------- | ----- |
| Provider-specific cloud IMDS passthrough              | ~370  |
| macOS system_profiler passthrough                     | ~45   |
| Deeply nested sub-objects (ethtool, tunnel, XDP, etc) | ~55   |
| PCI/SCSI/GPU bus enumeration detail                   | ~40   |
| Display/human-readable fields                         | ~5    |
| Internal identifiers (shard, timings)                 | ~5    |
| Per-CPU array mirrors of top-level fields             | ~4    |
| os-release URL/metadata fields                        | ~4    |
| Legacy/niche kernel memory fields                     | ~10   |
| Already covered by T1/T2                              | ~182  |
| Remaining provider/tool-specific passthrough          | ~286  |

---

## `device` — Host Identity & State

### fqdn (`hostname.fqdn`)

- **What:** Fully qualified domain name
- **Security relevance:** Primary host identity in configuration management;
  asset correlation key in SIEMs; DNS-based threat detection
- **OCSF object:** `device`
- **OTel precedent:** `host.name` may contain FQDN but doesn't break it out
- **gohai type:** `string` / `"fqdn"`

### timezone_name (`timezone.name`)

- **What:** IANA timezone name (e.g., "America/New_York")
- **Security relevance:** Anomaly detection ("login at 3 AM local time?");
  compliance auditing (all servers UTC?); fleet segmentation
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `string` / `"name"`

### timezone_offset (`timezone.offset`)

- **What:** Current UTC offset in seconds
- **Security relevance:** Programmatic timezone math for event correlation
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `int` / `"offset"`

### init_system (`init.name`)

- **What:** Init system name (systemd, launchd, upstart, sysvinit)
- **Security relevance:** Determines service management capabilities; CIS
  benchmark applicability differs by init system
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `string` / `"name"`

### uptime_seconds (`uptime.seconds`)

- **What:** Seconds since last boot
- **Security relevance:** Short uptimes may indicate compromise/reboot; long
  uptimes may indicate missing kernel patches
- **OCSF object:** `device`
- **OTel precedent:** `system.uptime` metric
- **gohai type:** `uint64` / `"seconds"`

### idle_seconds (`uptime.idle_seconds`)

- **What:** Aggregate CPU idle time since boot
- **Security relevance:** Idle ratio reveals crypto-mining or unauthorized CPU
  consumption
- **OCSF object:** `device`
- **OTel precedent:** `system.cpu.time` with `state=idle`
- **gohai type:** `uint64` / `"idle_seconds"`

### hypervisor_host (`virtualization.hypervisor_host`)

- **What:** Hostname of the hypervisor (Hyper-V KVP)
- **Security relevance:** Maps guest to hypervisor for blast-radius analysis
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `string` / `"hypervisor_host"`

### virtualization_role (`virtualization.role`)

- **What:** "host" or "guest" participation role
- **Security relevance:** Role determines patching responsibilities and attack
  surface
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `string` / `"role"`

### virtualization_systems (`virtualization.systems`)

- **What:** All detected virtualization layers (name→role map)
- **Security relevance:** Multi-layer detection (container inside VM); each
  layer has different isolation properties
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `map[string]string` / `"systems"`

### machine_model (`hardware.machine_model`)

- **What:** macOS machine model identifier
- **Security relevance:** Hardware model determines firmware vulnerability
  exposure and applicable security updates
- **OCSF object:** `device`
- **OTel precedent:** None
- **gohai type:** `string` / `"machine_model"`

### machine_id (`machine_id.id`)

- **What:** Stable machine identifier from `/etc/machine-id`
- **Security relevance:** Unique host identity that persists across reboots;
  asset tracking key
- **OCSF object:** `device`
- **OTel precedent:** `host.id` (related but not identical)
- **gohai type:** `string` / `"id"`

---

## `device_hw_info` — Hardware Detail

### cpu_sockets (`cpu.sockets`)

- **What:** Physical CPU socket count
- **Security relevance:** Licensing compliance; speculative execution
  vulnerability profiles differ by multi-socket
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `int` / `"sockets"`

### cpu_flags (`cpu.flags`)

- **What:** CPU feature flags (avx2, aes, ssse3, etc.)
- **Security relevance:** AES-NI for FIPS compliance; speculative execution
  mitigation flags (ibrs, stibp, ssbd)
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `[]string` / `"flags"`

### cpu_vulnerabilities (`cpu.vulnerabilities`)

- **What:** CPU vulnerability mitigations from sysfs
- **Security relevance:** Directly reports Spectre, Meltdown, MDS mitigation
  status; essential for fleet security posture
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `map[string]string` / `"vulnerabilities"`

### cpu_bogomips (`cpu.bogomips`)

- **What:** Linux BogoMIPS calibration value
- **Security relevance:** Performance baseline for anomaly detection
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `float64` / `"bogomips"`

### cpu_byte_order (`cpu.byte_order`)

- **What:** CPU byte order (Little/Big Endian)
- **Security relevance:** Binary compatibility; data format validation
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"byte_order"`

### cpu_address_sizes (`cpu.address_sizes`)

- **What:** Physical and virtual address width
- **Security relevance:** Virtual address width determines ASLR entropy
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"address_sizes"`

### cpu_numa_nodes_count (`cpu.numa_nodes_count`)

- **What:** NUMA node count
- **Security relevance:** NUMA topology affects performance; capacity planning
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `int` / `"numa_nodes_count"`

### cpu_caches L1/L3 (`cpu.caches.l1d`, `l1i`, `l3`)

- **What:** Per-level CPU cache sizes
- **Security relevance:** Cache hierarchy determines microarchitecture
  generation; L1 sizes relevant to cache side-channel attacks
- **OCSF object:** `device_hw_info`
- **OTel precedent:** `host.cpu.cache.l2.size` — natural extension to L1/L3
- **gohai type:** `string` / `"l1d"`, `"l1i"`, `"l3"`

### cpu_mhz_max / cpu_mhz_min (`cpu.mhz_max`, `cpu.mhz_min`)

- **What:** Maximum and minimum CPU frequency bounds
- **Security relevance:** Crypto-mining detection; performance envelope
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"mhz_max"`, `"mhz_min"`

### cpu_online / cpu_offline (`cpu.online`, `cpu.offline`)

- **What:** Online vs offline logical CPU count
- **Security relevance:** Mismatch indicates misconfiguration or scaling
- **OCSF object:** `device_hw_info`
- **OTel precedent:** OTel has `system.cpu.logical.count` total only
- **gohai type:** `int` / `"online"`, `"offline"`

### cpu_virtualization (`cpu.virtualization`)

- **What:** CPU virtualization capability (VT-x, AMD-V)
- **Security relevance:** Hardware-assisted virtualization availability for
  container isolation
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"virtualization"`

### dmi_baseboard (`dmi.baseboard.*`)

- **What:** Motherboard vendor, product, version, serial, asset_tag
- **Security relevance:** Hardware fingerprinting; firmware CVE tracking;
  asset inventory
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** struct with 5 string fields

### dmi_product_family/version/sku (`dmi.product.*`)

- **What:** System product family, version, SKU from SMBIOS
- **Security relevance:** Product family for fleet-wide firmware updates; SKU
  for procurement correlation
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"family"`, `"version"`, `"sku"`

### chassis_type (`dmi.chassis.type`, `type_description`)

- **What:** SMBIOS chassis type code and description
- **Security relevance:** Classifies form factor; laptops require disk
  encryption, servers different physical access controls
- **OCSF object:** `device_hw_info`
- **OTel precedent:** `hw.enclosure.type`
- **gohai type:** `string` / `"type"`, `"type_description"`

### chassis_asset_tag (`dmi.chassis.asset_tag`)

- **What:** Chassis asset tag from SMBIOS
- **Security relevance:** Links logical to physical inventory; compliance
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"asset_tag"`

### boot_rom_version (`hardware.boot_rom_version`)

- **What:** macOS Boot ROM version
- **Security relevance:** Firmware vulnerability exposure
- **OCSF object:** `device_hw_info`
- **OTel precedent:** `hw.firmware_version`
- **gohai type:** `string` / `"boot_rom_version"`

### smc_version (`hardware.smc_version_system`)

- **What:** SMC firmware version
- **Security relevance:** SMC firmware vulnerabilities compromise hardware
  root of trust
- **OCSF object:** `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"smc_version_system"`

### block_devices (`block_device.devices[].*`)

- **What:** Block device inventory: name, size, removable, rotational, model,
  vendor
- **Security relevance:** Removable media detection for DLP; device
  vendor/model for firmware CVE tracking
- **OCSF object:** `device_hw_info` (new sub-object)
- **OTel precedent:** `hw.physical_disk.type`, `hw.model`, `hw.vendor`
- **gohai type:** struct with 13 fields

---

## `os` — Operating System

### os_family (`platform.family`)

- **What:** Distribution family (debian, rhel, suse, arch)
- **Security relevance:** Determines patch channels and applicable CIS
  benchmarks
- **OCSF object:** `os`
- **OTel precedent:** None
- **gohai type:** `string` / `"family"`

### kernel_name (`kernel.name`)

- **What:** uname -s sysname
- **Security relevance:** Standard POSIX system identification
- **OCSF object:** `os`
- **OTel precedent:** `os.type` (normalized enum, not raw)
- **gohai type:** `string` / `"name"`

### kernel_version (`kernel.version`)

- **What:** Full kernel version string from uname -v
- **Security relevance:** Build configuration, compiler, timestamp; forensics
- **OCSF object:** `os`
- **OTel precedent:** None
- **gohai type:** `string` / `"version"`

### kernel_machine (`kernel.machine`)

- **What:** uname -m (x86_64, aarch64)
- **Security relevance:** Package repository selection; binary compatibility
- **OCSF object:** `os`
- **OTel precedent:** `host.arch` (normalized, not raw)
- **gohai type:** `string` / `"machine"`

### rosetta_translated (`kernel.rosetta_translated`)

- **What:** macOS Rosetta 2 translation state
- **Security relevance:** Indicates x86_64 emulation on ARM64
- **OCSF object:** `os`
- **OTel precedent:** None
- **gohai type:** `bool` / `"rosetta_translated"`

### os_release fields (`os_release.id`, `id_like`, `version_id`, `version_codename`, `variant_id`)

- **What:** Machine-parseable OS identity from os-release(5)
- **Security relevance:** `id` + `version_id` drive programmatic version
  comparison for patch management; `id_like` enables family queries;
  `variant_id` distinguishes server/workstation/container
- **OCSF object:** `os`
- **OTel precedent:** `os.version` (doesn't distinguish human vs machine)
- **gohai type:** `string` / various keys

### lsb fields (`lsb.id`, `release`, `codename`)

- **What:** Linux Standard Base distributor identity
- **Security relevance:** Legacy distro identification still used by
  enterprise systems
- **OCSF object:** `os`
- **OTel precedent:** None
- **gohai type:** `string` / various keys

---

## `network_interface` — Network Configuration

### mtu (`network.interfaces[].mtu`)

- **What:** Maximum Transmission Unit
- **Security relevance:** MTU misconfiguration causes fragmentation;
  fragmentation attacks
- **OCSF object:** `network_interface`
- **OTel precedent:** None
- **gohai type:** `int` / `"mtu"`

### driver (`network.interfaces[].driver`)

- **What:** Kernel driver name (e1000e, virtio_net, mlx5_core)
- **Security relevance:** Driver identifies hardware for firmware CVE tracking;
  detects virtual vs physical NICs
- **OCSF object:** `network_interface`
- **OTel precedent:** None
- **gohai type:** `string` / `"driver"`

### duplex (`network.interfaces[].duplex`)

- **What:** Link duplex mode
- **Security relevance:** Half-duplex on Gigabit indicates negotiation failure
- **OCSF object:** `network_interface`
- **OTel precedent:** None
- **gohai type:** `string` / `"duplex"`

### flags (`network.interfaces[].flags`)

- **What:** Interface flags (UP, BROADCAST, PROMISC, etc.)
- **Security relevance:** PROMISC flag indicates packet capture; interface
  state affects reachability
- **OCSF object:** `network_interface`
- **OTel precedent:** None
- **gohai type:** `[]string` / `"flags"`

### address scope/netmask/broadcast (`network.interfaces[].addresses[].*`)

- **What:** Address scope, netmask, broadcast per address
- **Security relevance:** Scope determines reachability; netmask determines
  subnet size and lateral movement blast radius
- **OCSF object:** `network_interface`
- **OTel precedent:** None
- **gohai type:** `string` fields

### default_interface / default_gateway (`network.*`)

- **What:** Default route egress interface and gateway (v4 + v6)
- **Security relevance:** Primary network path; segmentation verification
- **OCSF object:** `network_interface` or new `network_config`
- **OTel precedent:** None
- **gohai type:** `string` / 4 fields

### vlan_id (`network.interfaces[].vlan.id`)

- **What:** 802.1Q VLAN tag
- **Security relevance:** VLAN assignment determines segmentation; VLAN
  hopping is a known attack vector
- **OCSF object:** `network_interface`
- **OTel precedent:** None
- **gohai type:** `int` / `"id"`

---

## New: `routing_table`

### routes (`network.routes[].*`)

- **What:** Kernel routing table: destination, gateway, interface, scope,
  proto, metric, family
- **Security relevance:** Routes define reachability; essential for network
  forensics and lateral movement analysis
- **OCSF object:** New `routing_table`
- **OTel precedent:** None
- **gohai type:** struct with 7 fields per entry

---

## New: `neighbour_cache`

### neighbours (`network.neighbours[].*`)

- **What:** ARP/NDP cache: address, mac, interface, state, family
- **Security relevance:** Reveals broadcast domain members; NUD state indicates
  recent communication patterns
- **OCSF object:** New `neighbour_entry`
- **OTel precedent:** None
- **gohai type:** struct with 5 fields per entry

---

## New: `security_posture`

### fips_kernel_enabled (`fips.kernel.enabled`)

- **What:** Kernel FIPS 140 mode active
- **Security relevance:** Hard requirement for FedRAMP, FISMA, DoD
- **OCSF object:** New `security_posture`
- **OTel precedent:** None
- **gohai type:** `bool` / `"enabled"`

### fips_crypto_policy (`fips.policy.*`)

- **What:** System crypto policy name and FIPS effectiveness
- **Security relevance:** Determines allowed algorithms system-wide
- **OCSF object:** New `security_posture`
- **OTel precedent:** None
- **gohai type:** `string` + `bool`

### selinux (`selinux.*`)

- **What:** SELinux status, mode, policy version, policy name
- **Security relevance:** MAC framework state; CIS benchmarks require
  enforcing mode
- **OCSF object:** New `security_posture`
- **OTel precedent:** None
- **gohai type:** 6 string fields

### ssh_host_keys (`ssh.keys[].*`)

- **What:** SSH host key algorithm, fingerprint, key length
- **Security relevance:** Key rotation, algorithm deprecation, fingerprint
  verification
- **OCSF object:** New `ssh_host_key` (complement existing `tls`)
- **OTel precedent:** None
- **gohai type:** 4 fields per key

---

## New: `kernel_module`

### kernel_modules (`kernel_modules.modules.*`)

- **What:** Loaded kernel modules: name, size, refcount, version
- **Security relevance:** Rootkits load as kernel modules; CIS benchmarks
  check for specific modules (cramfs, udf, USB storage)
- **OCSF object:** New `kernel_module`
- **OTel precedent:** None
- **gohai type:** map of structs

---

## `cloud` — Cross-Provider

### instance_type (cross-provider)

- **What:** Instance size/shape (m5.xlarge, n2-standard-4, etc.)
- **Security relevance:** Determines capacity; most-queried cloud metadata
  after instance ID and region
- **OCSF object:** `cloud`
- **OTel precedent:** `host.type`
- **gohai type:** `string` (provider-specific key)

### instance_lifecycle (cross-provider)

- **What:** On-demand, spot/preemptible, or reserved
- **Security relevance:** Spot instances can disappear — IR evidence may be
  lost
- **OCSF object:** `cloud`
- **OTel precedent:** None
- **gohai type:** `string`

### tags (cross-provider)

- **What:** User-defined key/value tags
- **Security relevance:** Primary mechanism for security policy application
- **OCSF object:** `cloud`
- **OTel precedent:** None
- **gohai type:** varies by provider

### security_groups (`ec2`, `openstack`)

- **What:** Attached security group names/IDs
- **Security relevance:** Network access control; blast-radius analysis
- **OCSF object:** `cloud`
- **OTel precedent:** None
- **gohai type:** `[]string`

### iam_instance_profile (`ec2.iam_info.*`)

- **What:** IAM instance profile ARN
- **Security relevance:** Determines AWS API permissions; over-privileged
  profiles are common findings
- **OCSF object:** `cloud`
- **OTel precedent:** None
- **gohai type:** `string`

### vpc_id / subnet_id (cross-provider)

- **What:** Virtual network and subnet identifiers
- **Security relevance:** Network isolation boundary; public vs private subnet
- **OCSF object:** `cloud`
- **OTel precedent:** None
- **gohai type:** `string`

### azure_security_profile (`azure.security_profile.*`)

- **What:** Secure Boot, Virtual TPM, host encryption
- **Security relevance:** Directly measures VM security posture
- **OCSF object:** `cloud` or new `cloud_security_profile`
- **OTel precedent:** None
- **gohai type:** 3 bool fields

### service_accounts (`gce.service_accounts[].*`)

- **What:** GCE service account email and OAuth scopes
- **Security relevance:** Determines GCP API permissions
- **OCSF object:** `cloud`
- **OTel precedent:** None
- **gohai type:** struct with email + scopes

---

## Extend `user` / `group`

### users_passwd (`users.passwd.*`)

- **What:** User inventory: UID, GID, home, shell, GECOS
- **Security relevance:** UID 0 accounts, accounts without passwords,
  non-login shells; privilege escalation paths
- **OCSF object:** Extend `user`
- **OTel precedent:** None
- **gohai type:** per-user struct

### users_group (`users.group.*`)

- **What:** Group inventory: GID and members
- **Security relevance:** Group membership determines sudo/privilege paths
- **OCSF object:** Extend `group`
- **OTel precedent:** None
- **gohai type:** per-group struct

---

## Extend `session`

### session fields (`sessions.session.terminal`, `host`, `seat`)

- **What:** Terminal name, remote host, systemd seat
- **Security relevance:** "Who is logged in from where?" — first IR question
- **OCSF object:** Extend `session`
- **OTel precedent:** None
- **gohai type:** 3 string fields

---

## Extend `process`

### process_count (`process.count`)

- **What:** Total running process count
- **Security relevance:** Process spike indicates fork bomb or runaway service
- **OCSF object:** `process`
- **OTel precedent:** `system.process.count`
- **gohai type:** `int` / `"count"`

---

## `package` — Extend

### package_manager_path (`package_mgr.path`)

- **What:** Absolute path to package manager binary
- **Security relevance:** Binary path integrity verification
- **OCSF object:** `package`
- **OTel precedent:** None
- **gohai type:** `string` / `"path"`

---

## New: `container_inventory`

### docker (`docker.*`)

- **What:** Container inventory: ID, name, image, state, status
- **Security relevance:** Workload surface; container image identity for CVE
  tracking
- **OCSF object:** Extend `container` profile
- **OTel precedent:** `container.id`, `container.name`, `container.image.name`
  (per-container, not inventory)
- **gohai type:** 5 fields per container

---

## New: `sysctl_params`

### sysctl (`sysctl.params`)

- **What:** Kernel parameter table from `sysctl -a`
- **Security relevance:** Controls ASLR, IP forwarding, ptrace scope, yama
  LSM; CIS benchmarks check dozens of sysctl values
- **OCSF object:** New `kernel_parameter` or extend `os`
- **OTel precedent:** None
- **gohai type:** `map[string]string`

---

## New: `memory_info`

### memory subsystem (`memory.active`, `inactive`, `dirty`, `mapped`, `commit_limit`, `committed_as`, `swap.cached`)

- **What:** Detailed memory state from /proc/meminfo
- **Security relevance:** Resource exhaustion detection; crypto-mining
  indicators; OOM threshold monitoring
- **OCSF object:** New `memory_info` or extend `device_hw_info`
- **OTel precedent:** `system.memory.usage` (partial), `system.paging.usage`
  (partial)
- **gohai type:** multiple `uint64` fields

---

## New: `filesystem`

### inodes (`filesystem.mounts[].inodes_*`)

- **What:** Inode allocation and usage per filesystem
- **Security relevance:** Inode exhaustion is a DoS vector
- **OCSF object:** New `filesystem`
- **OTel precedent:** None
- **gohai type:** `uint64` + `float64`

### uuid / label (`filesystem.mounts[].uuid`, `label`)

- **What:** Filesystem UUID and label
- **Security relevance:** Stable forensic identifier; mount validation
- **OCSF object:** New `filesystem`
- **OTel precedent:** None
- **gohai type:** `string`

---

## New: `ipc_limits`

### ipc (`ipc.sem.*`, `msg.*`, `shm.*`)

- **What:** Semaphore, message queue, shared memory kernel limits
- **Security relevance:** IPC limits determine shared memory sizes; DoS
  vectors; database security
- **OCSF object:** New `ipc_limits` or extend `os`
- **OTel precedent:** None
- **gohai type:** multiple int fields

---

## Other

### load_averages (`load.one`, `five`, `fifteen`)

- **What:** 1/5/15-minute load averages
- **Security relevance:** Crypto-mining and DoS detection; universal health
  metric
- **OCSF object:** Extend `device` or new `load_average`
- **OTel precedent:** None dedicated
- **gohai type:** `float64` / 3 fields

### shells_paths (`shells.paths`)

- **What:** Valid login shells from /etc/shells
- **Security relevance:** Unexpected entries indicate compromise; CIS
  benchmarks verify this file
- **OCSF object:** Extend `device` or new `shell_config`
- **OTel precedent:** None
- **gohai type:** `[]string` / `"paths"`

### root_group_name (`root_group.name`)

- **What:** Root user's primary group name
- **Security relevance:** Differs between OS families; compliance-relevant for
  file permission audits
- **OCSF object:** Extend `os`
- **OTel precedent:** None
- **gohai type:** `string` / `"name"`

### pci_iommu_group (`pci.devices[].iommu_group`)

- **What:** IOMMU group assignment for PCI devices
- **Security relevance:** IOMMU grouping determines DMA isolation boundaries;
  devices in the same group can DMA attack each other
- **OCSF object:** Extend `device_hw_info`
- **OTel precedent:** None
- **gohai type:** `string` / `"iommu_group"`

### pci_driver (`pci.devices[].driver`)

- **What:** Kernel driver bound to PCI device
- **Security relevance:** Driver vulnerability targeting; identifies which
  kernel module manages the hardware
- **OCSF object:** Extend `device_hw_info`
- **OTel precedent:** `hw.driver_version`
- **gohai type:** `string` / `"driver"`

### cloud_public_keys (cross-provider)

- **What:** SSH public keys authorized on the instance
- **Security relevance:** Authorized access enumeration; unauthorized key
  injection detection
- **OCSF object:** `cloud` or new `authorized_key`
- **OTel precedent:** None
- **gohai type:** `[]string` or `map` / `"public_keys"`

### cloud_user_data (cross-provider)

- **What:** Instance user data / bootstrap scripts
- **Security relevance:** User data often contains secrets, credentials, and
  bootstrap scripts; sensitive but critical for security assessment
- **OCSF object:** `cloud` (with sensitivity warning)
- **OTel precedent:** None
- **gohai type:** `string` / `"user_data"`
