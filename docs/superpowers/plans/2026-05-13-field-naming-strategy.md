# Field Naming Strategy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Standardize all ~1146 JSON field names across 39 collectors using a
three-tier naming ladder (OCSF → OTel → convention), producing a mapping table
with verifiable citations, an OCSF gap report, and a cloud canonical overlay.

**Architecture:** Phase 1 builds the mapping table by looking up each field
against OCSF objects and OTel semconv, then applying convention rules for the
long tail. Phase 2 applies renames to Go code. Phase 3 extracts the gap list and
cloud overlay as standalone documents. All work happens on a feature branch off
`docs/field-naming-strategy`.

**Tech Stack:** Markdown tables, GitHub API (`gh api`) for OCSF/OTel schema
lookups, `grep` for Go struct enumeration, Go toolchain for renames.

**Spec:** `docs/superpowers/specs/2026-05-13-field-naming-strategy-design.md`

---

## Phase 1: Build the Mapping Table

Phase 1 is the bulk of the work. Each task maps one collector category's fields
against OCSF and OTel, producing rows for `schemas/field-mapping.md`.

### How to look up fields

**OCSF lookups:** The OCSF schema is published at
[schema.ocsf.io](https://schema.ocsf.io/) and on GitHub at
[ocsf/ocsf-schema](https://github.com/ocsf/ocsf-schema). The relevant objects
for gohai are:

| OCSF Object | GitHub path | gohai domain |
| --- | --- | --- |
| `device` | `objects/device.json` | hostname, machine_id, shard |
| `device_hw_info` | `objects/device_hw_info.json` | cpu, memory, dmi |
| `os` | `objects/os.json` | platform, kernel, os_release |
| `network_interface` | `objects/network_interface.json` | network |
| `package` | `objects/package.json` | package_mgr |
| `process` | `objects/process.json` | process |
| `cloud` | `objects/cloud.json` | cloud collectors |

Fetch an object definition:
```bash
gh api repos/ocsf/ocsf-schema/contents/objects/device.json --jq .content | base64 -d | python3 -m json.tool
```

For citation links, use the format:
`https://schema.ocsf.io/1.4.0/objects/device` (rendered) or
`https://github.com/ocsf/ocsf-schema/blob/main/objects/device.json` (source).

**OTel lookups:** OpenTelemetry semantic conventions are at
[open-telemetry/semantic-conventions](https://github.com/open-telemetry/semantic-conventions).
The relevant model directories:

| OTel namespace | GitHub path | gohai domain |
| --- | --- | --- |
| `host.*` | `model/host/` | hostname, uptime, cpu (model/vendor) |
| `system.cpu.*` | `model/system/` | cpu (usage metrics) |
| `system.memory.*` | `model/system/` | memory |
| `system.filesystem.*` | `model/system/` | filesystem |
| `system.paging.*` | `model/system/` | memory (swap) |
| `system.network.*` | `model/system/` | network |
| `os.*` | `model/os/` | platform, kernel |
| `process.*` | `model/process/` | process |
| `cloud.*` | `model/cloud/` | cloud collectors |
| `hardware.*` | `model/hardware/` | gpu, dmi, disk |

Fetch a registry file:
```bash
gh api repos/open-telemetry/semantic-conventions/contents/model/host/registry.yaml --jq .content | base64 -d
```

For citation links, use:
`https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml`

**Tier 3 convention rules** (apply when OCSF and OTel are both silent):

1. Use the backing library's field name (gopsutil/ghw) as starting point,
   converted to `snake_case`.
2. For `/proc` or `/sys` mirrors, use the kernel's name in `snake_case`.
3. Add unit suffixes (`_bytes`, `_seconds`, `_percent`, `_mhz`) when the unit is
   ambiguous from context.
4. No abbreviations except universals: `ip`, `mac`, `pid`, `uid`, `gid`, `mtu`,
   `fqdn`, `uuid`, `cidr`, `arn`, `id`.
5. Note the inspiration source in the Citation column (e.g., "gopsutil
   `SwapFree`", "kernel `/proc/meminfo` `HugePages_Total`").

**Redundant-prefix rule:** Strip parent-object prefixes when they duplicate the
collector key. OCSF `device.cpu_count` under gohai's `cpu` collector → `count`.

### Table format

Every row in `schemas/field-mapping.md` follows this format:

```markdown
| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
```

### Task 1: Scaffold the mapping table and fetch OCSF/OTel references

**Files:**
- Create: `schemas/field-mapping.md`
- Create: `schemas/references/ocsf-device.json` (cached OCSF object)
- Create: `schemas/references/ocsf-device-hw-info.json`
- Create: `schemas/references/ocsf-os.json`
- Create: `schemas/references/ocsf-network-interface.json`
- Create: `schemas/references/ocsf-package.json`
- Create: `schemas/references/ocsf-process.json`
- Create: `schemas/references/ocsf-cloud.json`
- Create: `schemas/references/otel-host-registry.yaml`
- Create: `schemas/references/otel-system-registry.yaml`
- Create: `schemas/references/otel-os-registry.yaml`
- Create: `schemas/references/otel-process-registry.yaml`
- Create: `schemas/references/otel-cloud-registry.yaml`
- Create: `schemas/references/otel-hardware-registry.yaml`

- [ ] **Step 1: Create the schemas/references directory and fetch OCSF objects**

```bash
mkdir -p schemas/references

# Fetch OCSF object definitions
for obj in device device_hw_info os network_interface package process cloud; do
  gh api repos/ocsf/ocsf-schema/contents/objects/${obj}.json \
    --jq .content | base64 -d > schemas/references/ocsf-${obj//_/-}.json
done
```

Verify each file is valid JSON:
```bash
for f in schemas/references/ocsf-*.json; do
  python3 -m json.tool "$f" > /dev/null && echo "OK: $f" || echo "FAIL: $f"
done
```

- [ ] **Step 2: Fetch OTel semconv registry files**

```bash
# Host attributes
gh api repos/open-telemetry/semantic-conventions/contents/model/host \
  --jq '.[].name' | while read f; do
  gh api repos/open-telemetry/semantic-conventions/contents/model/host/$f \
    --jq .content | base64 -d > schemas/references/otel-host-${f}
done

# OS attributes
gh api repos/open-telemetry/semantic-conventions/contents/model/os \
  --jq '.[].name' | while read f; do
  gh api repos/open-telemetry/semantic-conventions/contents/model/os/$f \
    --jq .content | base64 -d > schemas/references/otel-os-${f}
done

# Cloud attributes
gh api repos/open-telemetry/semantic-conventions/contents/model/cloud \
  --jq '.[].name' | while read f; do
  gh api repos/open-telemetry/semantic-conventions/contents/model/cloud/$f \
    --jq .content | base64 -d > schemas/references/otel-cloud-${f}
done

# System metrics (memory, filesystem, paging, network, cpu)
gh api repos/open-telemetry/semantic-conventions/contents/model/system \
  --jq '.[].name' | while read f; do
  gh api repos/open-telemetry/semantic-conventions/contents/model/system/$f \
    --jq .content | base64 -d > schemas/references/otel-system-${f}
done

# Process attributes
gh api repos/open-telemetry/semantic-conventions/contents/model/process \
  --jq '.[].name' | while read f; do
  gh api repos/open-telemetry/semantic-conventions/contents/model/process/$f \
    --jq .content | base64 -d > schemas/references/otel-process-${f}
done

# Hardware attributes (if exists)
gh api repos/open-telemetry/semantic-conventions/contents/model/hardware \
  --jq '.[].name' 2>/dev/null | while read f; do
  gh api repos/open-telemetry/semantic-conventions/contents/model/hardware/$f \
    --jq .content | base64 -d > schemas/references/otel-hardware-${f}
done
```

- [ ] **Step 3: Create the mapping table scaffold**

Create `schemas/field-mapping.md` with the header, tier legend, and empty
sections for each collector category:

```markdown
# Field Mapping Table

Three-tier naming ladder applied to every gohai JSON field.

**Tier legend:**
- **T1** — OCSF: name comes from [OCSF](https://schema.ocsf.io/) object
- **T2** — OTel: name comes from [OTel semconv](https://github.com/open-telemetry/semantic-conventions)
- **T3** — Convention: name follows gohai convention rules (backing library + snake_case + unit suffixes)

**OCSF version:** 1.4.0
**OTel semconv version:** (fill after fetching — check `model/README.md` or latest release tag)

---

## System Collectors

| Collector | Go Field | Current JSON | Tier | Chosen JSON | Changed? | Source | Citation |
| --------- | -------- | ------------ | ---- | ----------- | -------- | ------ | -------- |

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
```

- [ ] **Step 4: Commit scaffold**

```bash
git add schemas/references/ schemas/field-mapping.md
git commit -m "docs(schema): scaffold mapping table + fetch OCSF/OTel references"
```

---

### Task 2: Map system collectors (57 fields)

**Files:**
- Modify: `schemas/field-mapping.md`
- Read: `pkg/gohai/collectors/platform/platform.go`
- Read: `pkg/gohai/collectors/hostname/hostname.go`
- Read: `pkg/gohai/collectors/kernel/kernel.go`
- Read: `pkg/gohai/collectors/kernel_modules/kernel_modules.go`
- Read: `pkg/gohai/collectors/uptime/uptime.go`
- Read: `pkg/gohai/collectors/timezone/timezone.go`
- Read: `pkg/gohai/collectors/os_release/os_release.go`
- Read: `pkg/gohai/collectors/init/init.go`
- Read: `pkg/gohai/collectors/fips/fips.go`
- Read: `pkg/gohai/collectors/machine_id/machine_id.go`
- Read: `pkg/gohai/collectors/root_group/root_group.go`
- Read: `pkg/gohai/collectors/shells/shells.go`
- Read: `pkg/gohai/collectors/shard/shard.go`
- Read: `schemas/references/ocsf-device.json`
- Read: `schemas/references/ocsf-os.json`
- Read: `schemas/references/otel-host-*.yaml`
- Read: `schemas/references/otel-os-*.yaml`

Collectors in scope: platform (7), hostname (4), kernel (7), kernel_modules (5),
uptime (5), timezone (3), os_release (14), init (1), fips (5), machine_id (1),
root_group (1), shells (1), shard (3). **Total: 57 fields.**

- [ ] **Step 1: Extract all JSON fields from system collectors**

```bash
for c in platform hostname kernel kernel_modules uptime timezone os_release \
         init fips machine_id root_group shells shard; do
  echo "=== $c ==="
  grep 'json:"' pkg/gohai/collectors/$c/*.go | grep -v '_test.go' | \
    grep -v 'export_test' | \
    sed 's/.*\t\([A-Za-z]*\).*json:"\([^"]*\)".*/  \1 → \2/'
done
```

- [ ] **Step 2: Look up each field against OCSF references**

For each field, check the relevant OCSF object JSON file in
`schemas/references/`. The mapping logic:

- `platform.*` → check `ocsf-os.json` for `name`, `version`, `type`,
  `build`, `kernel_release`, `edition`, `sp_name`, `sp_ver`, `cpe_name`,
  `country`, `lang`, `cpu_bits`
- `hostname.*` → check `ocsf-device.json` for `hostname`, `domain`, `uid`,
  `name`
- `kernel.*` → check `ocsf-os.json` for `kernel_release`
- `uptime.*` → check `ocsf-device.json` for `boot_time`; no OCSF uptime field
- `machine_id.*` → check `ocsf-device.json` for `uid`
- All others → likely no OCSF match (kernel_modules, timezone, os_release
  details, init, fips, root_group, shells, shard)

- [ ] **Step 3: Look up remaining fields against OTel references**

For fields with no OCSF match, check OTel:

- `kernel.*` → `otel-os-*.yaml` for `os.type`, `os.name`, `os.version`,
  `os.build_id`, `os.description`
- `uptime.*` → `otel-host-*.yaml` for `host.uptime` (if it exists)
- `os_release.*` → `otel-os-*.yaml`

- [ ] **Step 4: Apply tier-3 convention to remaining fields**

For fields with no OCSF or OTel match, apply convention rules. Most system
collector fields are straightforward — they're already well-named. Document the
backing library or kernel source in the Citation column.

- [ ] **Step 5: Fill in the System Collectors section of field-mapping.md**

Write all 57 rows into the System Collectors table. Each row must have all 8
columns filled. Example rows:

```markdown
| platform | Name | `name` | T1 | `name` | no | OCSF `os.name` | [os.json](https://schema.ocsf.io/1.4.0/objects/os) |
| platform | Version | `version` | T1 | `version` | no | OCSF `os.version` | [os.json](https://schema.ocsf.io/1.4.0/objects/os) |
| hostname | Name | `name` | T1 | `name` | no | OCSF `device.hostname` (stripped) | [device.json](https://schema.ocsf.io/1.4.0/objects/device) |
| kernel | Release | `release` | T1 | `release` | no | OCSF `os.kernel_release` (stripped) | [os.json](https://schema.ocsf.io/1.4.0/objects/os) |
| fips | Enabled | `enabled` | T3 | `enabled` | no | Convention | gohai standard — no schema covers FIPS |
```

- [ ] **Step 6: Commit**

```bash
git add schemas/field-mapping.md
git commit -m "docs(schema): map system collectors — 57 fields (T1/T2/T3)"
```

---

### Task 3: Map hardware collectors (280 fields)

**Files:**
- Modify: `schemas/field-mapping.md`
- Read: `pkg/gohai/collectors/cpu/cpu.go`
- Read: `pkg/gohai/collectors/memory/memory.go`
- Read: `pkg/gohai/collectors/disk/disk.go`
- Read: `pkg/gohai/collectors/filesystem/filesystem.go`
- Read: `pkg/gohai/collectors/dmi/dmi.go`
- Read: `pkg/gohai/collectors/gpu/gpu.go`
- Read: `pkg/gohai/collectors/pci/pci.go`
- Read: `pkg/gohai/collectors/scsi/scsi.go`
- Read: `pkg/gohai/collectors/hardware/hardware.go`
- Read: `schemas/references/ocsf-device-hw-info.json`
- Read: `schemas/references/otel-host-*.yaml`
- Read: `schemas/references/otel-system-*.yaml`
- Read: `schemas/references/otel-hardware-*.yaml`

Collectors in scope: cpu (50), memory (60), disk (9), filesystem (46), dmi (25),
gpu (14), pci (15), scsi (7), hardware (74). **Total: 280 fields.**

This is the largest and most interesting batch — OCSF covers `cpu_count`,
`cpu_cores`, `cpu_speed`, `ram_size`, `serial_number`, `bios_*`, `gpu_*`. OTel
covers `host.cpu.model.name`, `host.cpu.family`, `host.cpu.stepping`,
`system.memory.state`, `system.filesystem.type/mountpoint`. The remaining ~200
fields are tier 3.

- [ ] **Step 1: Extract all JSON fields from hardware collectors**

```bash
for c in cpu memory disk filesystem dmi gpu pci scsi hardware; do
  echo "=== $c ==="
  grep 'json:"' pkg/gohai/collectors/$c/*.go | grep -v '_test.go' | \
    grep -v 'export_test' | \
    sed 's/.*\t\([A-Za-z]*\).*json:"\([^"]*\)".*/  \1 → \2/'
done
```

- [ ] **Step 2: Map cpu fields (50)**

OCSF `device_hw_info` covers: `cpu_count` → `count`, `cpu_cores` → `cores`,
`cpu_speed` → check if `mhz` should become `speed`, `cpu_type`,
`cpu_architecture`.

OTel `host.cpu.*` covers: `model.name` → `model_name`, `vendor.id` →
`vendor_id`, `family`, `model.id` → `model`, `stepping`, `cache.l2.size`.

Remaining ~35 fields (bogomips, byte_order, flags, op_modes, numa_*,
vulnerabilities, caches, hypervisor_vendor, virtualization_type, etc.) are tier
3.

- [ ] **Step 3: Map memory fields (60)**

OCSF `device_hw_info` covers: `ram_size` → check if `total` should stay or
become `size`.

OTel `system.memory.*` state dimensions: `used`, `free`, `shared`, `buffers`,
`cached` — these are metric state labels, not field names, so they're evidence
for our naming but not direct name sources. Check `system.paging.*` for swap.

Remaining ~50 fields (active_anon, inactive_file, slab, page_tables, vmalloc_*,
hugepages.*, direct_map.*, etc.) are tier 3 — direct `/proc/meminfo` mirrors.

- [ ] **Step 4: Map filesystem fields (46)**

OTel `system.filesystem.*` covers: `type` (→ `fstype` or `type`), `mountpoint`,
`mode`, `state`.

Remaining fields (device, opts, total/free/used capacity, inodes, etc.) are
mostly well-named already — check gopsutil field names.

- [ ] **Step 5: Map disk, dmi, gpu, pci, scsi, hardware fields (124)**

- `dmi` (25): OCSF `device_hw_info` covers BIOS fields (`bios_date`,
  `bios_manufacturer`, `bios_ver`), `serial_number`, `uuid`, `chassis`,
  `vendor_name`. Check each field.
- `gpu` (14): OCSF `device_hw_info` has `gpu_count`, `gpu_info_list`. OTel
  `hardware.*` may have GPU-related attributes.
- `pci` (15), `scsi` (7): Likely all tier 3 — no schema covers PCI/SCSI device
  enumeration.
- `disk` (9): OTel `system.disk.*` may cover I/O counters.
- `hardware` (74): macOS-specific (system_profiler output) — almost all tier 3.

- [ ] **Step 6: Fill in the Hardware Collectors section**

Write all 280 rows. For the large `/proc/meminfo` block in memory, note that
these are kernel counter mirrors and cite the kernel source.

- [ ] **Step 7: Commit**

```bash
git add schemas/field-mapping.md
git commit -m "docs(schema): map hardware collectors — 280 fields (T1/T2/T3)"
```

---

### Task 4: Map network collector (77 fields)

**Files:**
- Modify: `schemas/field-mapping.md`
- Read: `pkg/gohai/collectors/network/network.go`
- Read: `schemas/references/ocsf-network-interface.json`
- Read: `schemas/references/otel-system-*.yaml` (network metrics)

**Total: 77 fields.**

OCSF `network_interface` has strong coverage here: `name`, `ip`, `mac`,
`hostname`, `subnet_prefix`, `type`, `uid`, `namespace`. OTel `system.network.*`
covers I/O counter dimensions.

- [ ] **Step 1: Extract all JSON fields**

```bash
echo "=== network ==="
grep 'json:"' pkg/gohai/collectors/network/*.go | grep -v '_test.go' | \
  grep -v 'export_test' | \
  sed 's/.*\t\([A-Za-z]*\).*json:"\([^"]*\)".*/  \1 → \2/'
```

- [ ] **Step 2: Map against OCSF network_interface and OTel system.network**

Check OCSF coverage for interface-level fields (name, mac, ip, type). Check OTel
for I/O counter naming (bytes_sent, bytes_recv, packets_sent, packets_recv,
errors, dropped).

- [ ] **Step 3: Apply convention to remaining fields**

Route/neighbor/DNS fields are likely tier 3. Use gopsutil field names as the
baseline.

- [ ] **Step 4: Fill in Network Collectors section and commit**

```bash
git add schemas/field-mapping.md
git commit -m "docs(schema): map network collector — 77 fields (T1/T2/T3)"
```

---

### Task 5: Map cloud collectors (672 fields)

**Files:**
- Modify: `schemas/field-mapping.md`
- Read: `pkg/gohai/collectors/ec2/ec2.go`
- Read: `pkg/gohai/collectors/gce/gce.go`
- Read: `pkg/gohai/collectors/azure/azure.go`
- Read: `pkg/gohai/collectors/digital_ocean/digital_ocean.go`
- Read: `pkg/gohai/collectors/openstack/openstack.go`
- Read: `pkg/gohai/collectors/alibaba/alibaba.go`
- Read: `pkg/gohai/collectors/linode/linode.go`
- Read: `pkg/gohai/collectors/oci/oci.go`
- Read: `pkg/gohai/collectors/scaleway/scaleway.go`
- Read: `schemas/references/ocsf-cloud.json`
- Read: `schemas/references/otel-cloud-*.yaml`

Collectors: ec2 (68), gce (99), azure (152), digital_ocean (50), openstack (42),
alibaba (49), linode (2), oci (138), scaleway (72). **Total: 672 fields.**

This is the largest batch by field count, but the approach is consistent:

1. The ~10 **canonical cross-provider fields** (`instance_id`, `region`, `zone`,
   `instance_type`, `hostname`, `public_ips`, `private_ips`, `account_id`,
   `image_id`, `tags`) get tier 1 from OCSF `cloud` object or tier 2 from OTel
   `cloud.*` semconv. Check which fields each provider already uses vs. needs
   renaming to the canonical name.
2. All remaining provider-specific fields are tier 3, keeping the provider API's
   native naming.

- [ ] **Step 1: Extract all JSON fields from cloud collectors**

```bash
for c in ec2 gce azure digital_ocean openstack alibaba linode oci scaleway; do
  echo "=== $c ==="
  grep 'json:"' pkg/gohai/collectors/$c/*.go | grep -v '_test.go' | \
    grep -v 'export_test' | \
    sed 's/.*\t\([A-Za-z]*\).*json:"\([^"]*\)".*/  \1 → \2/'
done
```

- [ ] **Step 2: Check OCSF cloud object and OTel cloud semconv**

OCSF `cloud.json` likely has: `provider`, `region`, `zone`, `account.uid`.
OTel `cloud.*` has: `cloud.provider`, `cloud.account.id`, `cloud.region`,
`cloud.availability_zone`, `cloud.platform`.

Map the canonical cross-provider fields first.

- [ ] **Step 3: For each provider, identify the canonical field mapping**

For each cloud collector, find which existing fields map to the 10 canonical
names:

**EC2:**
- `instance_id` → already `instance_id` ✓
- `region` → already `region` ✓
- `availability_zone` → canonical is `zone` — check if rename needed
- `instance_type` → already `instance_type` ✓
- `hostname` → already `hostname` ✓
- `local_ipv4` → canonical is `private_ips` (as array) — needs discussion
- `public_ipv4` → canonical is `public_ips` (as array) — needs discussion
- `account_id` → already `account_id` ✓
- `ami_id` → canonical is `image_id` — needs discussion
- `tags` → check if exists

**GCE, Azure, DigitalOcean, etc.:** Same exercise. Document each mapping.

- [ ] **Step 4: Mark all provider-specific fields as tier 3**

Provider-specific fields (e.g., EC2 `ami_launch_index`,
`spot_instance_action`, `services_domain`; Azure `vm_size`,
`os_profile`, `security_profile`) keep their current names. Citation is
"provider API native name".

- [ ] **Step 5: Fill in Cloud Collectors section and commit**

```bash
git add schemas/field-mapping.md
git commit -m "docs(schema): map cloud collectors — 672 fields (T1/T2/T3)"
```

---

### Task 6: Map remaining collectors (40 fields)

**Files:**
- Modify: `schemas/field-mapping.md`
- Read: `pkg/gohai/collectors/virtualization/virtualization.go`
- Read: `pkg/gohai/collectors/users/users.go`
- Read: `pkg/gohai/collectors/sessions/sessions.go`
- Read: `pkg/gohai/collectors/process/process.go`
- Read: `pkg/gohai/collectors/load/load.go`
- Read: `pkg/gohai/collectors/lsb/lsb.go`
- Read: `pkg/gohai/collectors/package_mgr/package_mgr.go`
- Read: `schemas/references/ocsf-process.json`

Collectors: virtualization (4), users (10), sessions (8), process (9), load (3),
lsb (4), package_mgr (2). **Total: 40 fields.**

- [ ] **Step 1: Extract all JSON fields**

```bash
for c in virtualization users sessions process load lsb package_mgr; do
  echo "=== $c ==="
  grep 'json:"' pkg/gohai/collectors/$c/*.go | grep -v '_test.go' | \
    grep -v 'export_test' | \
    sed 's/.*\t\([A-Za-z]*\).*json:"\([^"]*\)".*/  \1 → \2/'
done
```

- [ ] **Step 2: Look up against OCSF and OTel**

- `process.*` → OCSF `process.json` has `pid`, `name`, `cmd_line`, `uid`,
  `user`. OTel `process.*` has similar.
- `load.*` → OTel has no standard for load averages. Tier 3.
- `virtualization.*` → no direct OCSF/OTel object. Tier 3.
- `users.*`, `sessions.*` → OCSF has `user` object but focused on identity, not
  system user enumeration. Likely tier 3.
- `lsb.*` → no schema covers LSB. Tier 3.
- `package_mgr.*` → OCSF `package.json` may have `package_manager`. Check.

- [ ] **Step 3: Fill in Other Collectors section and commit**

```bash
git add schemas/field-mapping.md
git commit -m "docs(schema): map remaining collectors — 40 fields (T1/T2/T3)"
```

---

### Task 7: Review and validate the complete mapping table

**Files:**
- Modify: `schemas/field-mapping.md`

- [ ] **Step 1: Count and verify completeness**

```bash
# Count rows in the mapping table (excluding headers)
grep -c '^|' schemas/field-mapping.md

# Count total JSON fields in Go code
grep -r 'json:"' pkg/gohai/collectors/ --include="*.go" | \
  grep -v '_test.go' | grep -v 'export_test' | wc -l
```

The two numbers should be close (the table count may differ slightly because
some fields are on nested sub-structs that share a collector).

- [ ] **Step 2: Verify all tier-1 citations resolve**

For every row with Tier = T1, confirm the OCSF link points to a real page:

```bash
grep '| T1 |' schemas/field-mapping.md | \
  grep -o 'https://[^)]*' | sort -u | while read url; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
  echo "$status $url"
done
```

All should return 200. Fix any broken links.

- [ ] **Step 3: Verify all tier-2 citations resolve**

Same for OTel links:

```bash
grep '| T2 |' schemas/field-mapping.md | \
  grep -o 'https://[^)]*' | sort -u | while read url; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
  echo "$status $url"
done
```

- [ ] **Step 4: Count tier distribution**

```bash
echo "Tier 1 (OCSF):"; grep -c '| T1 |' schemas/field-mapping.md
echo "Tier 2 (OTel):"; grep -c '| T2 |' schemas/field-mapping.md
echo "Tier 3 (Convention):"; grep -c '| T3 |' schemas/field-mapping.md
echo "Changed:"; grep -c '| yes |' schemas/field-mapping.md
```

- [ ] **Step 5: Commit validated table**

```bash
git add schemas/field-mapping.md
git commit -m "docs(schema): validate mapping table — all citations verified"
```

---

## Phase 2: Apply Renames

Phase 2 only runs for fields where `Changed? = yes` in the mapping table.
Each task is one collector (or a batch of small ones).

**Important:** Phase 2 tasks are generated from Phase 1 output. The tasks below
are templates — the actual list depends on which fields need renaming.

### Task 8: Identify all renames from the mapping table

**Files:**
- Read: `schemas/field-mapping.md`
- Create: `schemas/rename-plan.md`

- [ ] **Step 1: Extract all changed fields**

```bash
grep '| yes |' schemas/field-mapping.md | \
  awk -F'|' '{print $2 "|" $3 "|" $4 "|" $6}' | \
  sed 's/^ *//;s/ *$//' > /tmp/renames.txt
cat /tmp/renames.txt
```

- [ ] **Step 2: Group by collector and create rename plan**

Create `schemas/rename-plan.md` listing each collector that needs changes, with
the specific field renames. Format:

```markdown
# Rename Plan

Generated from field-mapping.md. Each section is one PR.

## memory (3 renames)
- `size` → `total` (Go: `Size` → `Total`)
- ...

## ec2 (2 renames)
- `availability_zone` → `zone` (Go: `AvailabilityZone` → `Zone`)
- `ami_id` → `image_id` (Go: `AMIID` → `ImageID`)
```

- [ ] **Step 3: Commit rename plan**

```bash
git add schemas/rename-plan.md
git commit -m "docs(schema): generate rename plan from mapping table"
```

---

### Task 9: Apply renames (per-collector, template)

**This task repeats for each collector in the rename plan.** The steps are
identical; only the file paths and field names change.

**Files (per collector):**
- Modify: `pkg/gohai/collectors/<name>/<name>.go` — struct field + JSON tag
- Modify: `pkg/gohai/collectors/<name>/<name>_public_test.go` — test references
- Modify: `docs/collectors/<name>.md` — Collected Fields table

- [ ] **Step 1: Rename the Go struct field and JSON tag**

For each field in the rename plan for this collector, edit the struct definition.
Example for memory `size` → `total`:

In `pkg/gohai/collectors/memory/memory.go`, change:
```go
Size uint64 `json:"size"`
```
to:
```go
Total uint64 `json:"total"`
```

- [ ] **Step 2: Update all references in the collector package**

Search for uses of the old field name in the collector's `.go` files:

```bash
grep -rn 'OldFieldName' pkg/gohai/collectors/<name>/
```

Update each reference.

- [ ] **Step 3: Update the collector doc**

In `docs/collectors/<name>.md`, update the Collected Fields table to use the new
JSON key and note the schema mapping.

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/gohai/collectors/<name>/... -v
```

All tests must pass.

- [ ] **Step 5: Run vet**

```bash
just go::vet
```

Must be clean.

- [ ] **Step 6: Commit**

```bash
git add pkg/gohai/collectors/<name>/ docs/collectors/<name>.md
git commit -m "refactor(<name>): rename fields per naming strategy"
```

---

## Phase 3: Publish Deliverables

### Task 10: Generate the OCSF gap report

**Files:**
- Read: `schemas/field-mapping.md`
- Create: `schemas/ocsf-gaps.md`

- [ ] **Step 1: Extract all tier-3 fields that are OCSF PR candidates**

Not every tier-3 field is an OCSF candidate. Filter for fields that represent
concepts OCSF *should* cover (device inventory, hardware detail, OS
configuration) vs. fields that are too domain-specific (provider-native cloud
metadata, macOS system_profiler output).

Criteria for OCSF candidacy:
- The field represents a concept that appears in 2+ systems (Linux + macOS, or
  multiple providers)
- The field is relevant to security or asset inventory use cases
- The field isn't already covered by a more specific OCSF object

- [ ] **Step 2: Write the gap report**

Create `schemas/ocsf-gaps.md` grouped by the OCSF object the field would extend:

```markdown
# OCSF Gap Report

Fields where gohai collects data that OCSF does not currently model. Each entry
is a candidate for an upstream OCSF PR.

**Generated from:** `schemas/field-mapping.md` (tier 3 fields)
**OCSF version:** 1.4.0
**Date:** 2026-05-13

---

## device_hw_info — CPU Detail

### cpu.stepping
- **What:** Silicon revision number within a CPU model family
- **Why OCSF lacks it:** `device_hw_info` focuses on aggregate counts
  (cpu_count, cpu_cores, cpu_speed), not microarchitecture detail
- **Why it matters:** Vulnerability correlation — Spectre/Meltdown mitigations
  vary by CPU stepping. Asset inventory tools need this for patch targeting.
- **OTel precedent:** [`host.cpu.stepping`](https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml)
- **gohai field:** `cpu.stepping` (int32)

### cpu.model_name
...
```

Each entry includes:
- **What** — one-line description
- **Why OCSF lacks it** — schema design reason
- **Why it matters** — inventory/security justification
- **OTel/ECS precedent** — link to another standard's definition (if exists)
- **gohai field** — our field name and type

- [ ] **Step 3: Commit**

```bash
git add schemas/ocsf-gaps.md
git commit -m "docs(schema): OCSF gap report — upstream PR candidates"
```

---

### Task 11: Generate the cloud canonical overlay

**Files:**
- Read: `schemas/field-mapping.md`
- Read: `pkg/gohai/collectors/ec2/ec2.go`
- Read: `pkg/gohai/collectors/gce/gce.go`
- Read: `pkg/gohai/collectors/azure/azure.go`
- Read: `pkg/gohai/collectors/digital_ocean/digital_ocean.go`
- Read: `pkg/gohai/collectors/openstack/openstack.go`
- Read: `pkg/gohai/collectors/alibaba/alibaba.go`
- Read: `pkg/gohai/collectors/linode/linode.go`
- Read: `pkg/gohai/collectors/oci/oci.go`
- Read: `pkg/gohai/collectors/scaleway/scaleway.go`
- Create: `schemas/cloud-canonical.md`

- [ ] **Step 1: Build the cross-provider mapping table**

For each of the 10 canonical fields, document how each provider's raw metadata
maps to the canonical name:

```markdown
# Cloud Canonical Overlay

Cross-provider field standardization. Every cloud collector emits these fields
with identical JSON keys regardless of provider.

## Canonical Fields

| Canonical | Type | Description |
| --------- | ---- | ----------- |
| `instance_id` | string | Provider-assigned unique instance identifier |
| `region` | string | Cloud region |
| `zone` | string | Availability zone within region |
| `instance_type` | string | Instance size/shape |
| `hostname` | string | Instance hostname from provider metadata |
| `public_ips` | []string | Public IP addresses |
| `private_ips` | []string | Private IP addresses |
| `account_id` | string | Cloud account/project identifier |
| `image_id` | string | Machine image identifier |
| `tags` | map[string]string | Provider tags/labels |

## Per-Provider Mapping

| Canonical | EC2 | GCE | Azure | DO | OpenStack | Alibaba | Linode | OCI | Scaleway |
| --------- | --- | --- | ----- | -- | --------- | ------- | ------ | --- | -------- |
| `instance_id` | `instance-id` | `id` | `compute.vmId` | `droplet_id` | `uuid` | `instance-id` | N/A | `id` | `id` |
| ... | ... | ... | ... | ... | ... | ... | ... | ... | ... |
```

Fill in by reading each cloud collector's Go code to find how it sources each
canonical concept from the provider's metadata endpoint.

- [ ] **Step 2: Document fields that require code changes**

For each provider where the current Go field name doesn't match the canonical
name, note what needs to change. These feed back into Task 9 (apply renames).

- [ ] **Step 3: Commit**

```bash
git add schemas/cloud-canonical.md
git commit -m "docs(schema): cloud canonical overlay — 10 cross-provider fields"
```

---

### Task 12: Update CLAUDE.md field naming section

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Replace the field naming precedence**

In `CLAUDE.md`, find the "Field naming" subsection under "Implementation
Methodology". Replace the current 5-level precedence list:

```
1. OCSF
2. OpenTelemetry semantic conventions
3. Industry standard
4. Ohai's name
5. Our own name
```

With the three-tier ladder:

```markdown
### Field naming

**Three-tier naming ladder.** Every JSON field name comes from one of three
tiers, applied in strict order:

1. **[OCSF][]** (Open Cybersecurity Schema Framework) — primary authority.
   When OCSF has a field for the concept, use its name. Browse
   [schema.ocsf.io][ocsf-schema] objects: `device`, `device_hw_info`, `os`,
   `network_interface`, `package`, `process`, `cloud`.
2. **[OpenTelemetry Resource Semantic Conventions][otel-semconv]** — when
   OCSF is silent. Covers CPU microarchitecture (`host.cpu.*`), memory
   states (`system.memory.*`), filesystem attributes
   (`system.filesystem.*`), and hardware detail (`hardware.*`).
3. **gohai convention** — for the long tail where no standard has an
   opinion:
   - Start from the backing library's field name (gopsutil/ghw),
     converted to `snake_case`.
   - `/proc` and `/sys` mirrors use the kernel's name in `snake_case`.
   - Unit suffixes (`_bytes`, `_seconds`, `_percent`, `_mhz`) when the
     unit is ambiguous.
   - No abbreviations except universals: `ip`, `mac`, `pid`, `uid`,
     `gid`, `mtu`, `fqdn`, `uuid`, `cidr`, `arn`, `id`.

The complete per-field mapping with citations lives in
`schemas/field-mapping.md`. Fields where OCSF is silent are tracked in
`schemas/ocsf-gaps.md` as upstream contribution candidates.

**Not a naming reference:** Ohai (methodology only, not naming),
node_exporter (methodology only), OCP (hardware design spec), CIS/SCAP/XCCDF
(compliance policies). ECS, osquery, and Facter are useful cross-references
but not naming authorities — the three-tier ladder is the precedence.
```

- [ ] **Step 2: Remove the "Not a reference for our schema" paragraph**

The current CLAUDE.md has a separate paragraph about OCP/CIS/SCAP not being
references. This is now covered in the updated section above. Remove the
duplicate.

- [ ] **Step 3: Update the Ohai reference to be methodology-only**

Search CLAUDE.md for any mention of Ohai as a naming reference and ensure it's
clear that Ohai is methodology-only. The "MANDATORY: Cross-reference Ohai's data
sources" section should remain unchanged — it's about collection approach, not
naming.

- [ ] **Step 4: Run docs formatting**

```bash
just docs::fmt
```

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md field naming to three-tier ladder"
```

---

### Task 13: Final review and summary commit

**Files:**
- Read: `schemas/field-mapping.md`
- Read: `schemas/ocsf-gaps.md`
- Read: `schemas/cloud-canonical.md`
- Read: `CLAUDE.md`

- [ ] **Step 1: Generate summary statistics**

```bash
echo "=== Mapping Table ==="
echo "Total fields mapped:"; grep -c '^| [a-z]' schemas/field-mapping.md
echo "Tier 1 (OCSF):"; grep -c '| T1 |' schemas/field-mapping.md
echo "Tier 2 (OTel):"; grep -c '| T2 |' schemas/field-mapping.md
echo "Tier 3 (Convention):"; grep -c '| T3 |' schemas/field-mapping.md
echo "Fields renamed:"; grep -c '| yes |' schemas/field-mapping.md

echo ""
echo "=== OCSF Gap Report ==="
echo "Gap entries:"; grep -c '^### ' schemas/ocsf-gaps.md

echo ""
echo "=== Cloud Canonical ==="
echo "Providers covered:"; head -1 schemas/cloud-canonical.md | tr '|' '\n' | wc -l
```

- [ ] **Step 2: Verify all files are committed**

```bash
git status
git log --oneline docs/field-naming-strategy..HEAD
```

- [ ] **Step 3: Run full test suite**

```bash
just test
```

Must pass (Phase 1-3 are docs-only unless Phase 2 renames were applied).

- [ ] **Step 4: Clean up reference files (optional)**

The `schemas/references/` directory contains cached OCSF/OTel files used during
mapping. Decide whether to keep them (useful for future re-verification) or
remove them (they're fetchable on demand). If keeping, add a README explaining
their purpose.

- [ ] **Step 5: Final commit if any cleanup**

```bash
git add -A
git commit -m "docs(schema): field naming strategy — complete mapping + gap report"
```
