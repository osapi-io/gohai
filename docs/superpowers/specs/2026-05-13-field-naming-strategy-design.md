# Field Naming Strategy Design

**Date:** 2026-05-13
**Status:** Draft
**Scope:** All ~800 JSON field names across gohai's 43 collectors

## Problem

gohai inherited ad-hoc field names from Ohai's Ruby plugins and early collector
implementations. A previous attempt (branch `feat/schema-corpus-tier1`) tried to
standardize names by building a 30-source corpus and running a 4-role subagent
pipeline (drafter → citation verifier → missed-match hunter → naming critic) to
annotate each field with an `x-rationale` block citing the corpus. It produced
rigorous annotations for 31 fields across 5 collectors before stalling:

- **Coverage gap:** 792 of 823 fields had no corpus match. Most corpus sources
  (CycloneDX, SPDX, PURL, OSCAL, Yang) are software-supply-chain or network
  schemas with no opinion on system inventory fields.
- **Process cost:** The 4-role pipeline was too expensive per field to scale to
  800+ fields.
- **Signal-to-noise:** 30 sources that mostly agree on `hostname` and `os.name`
  but are silent on `hugepages_total` doesn't help — the "voting" only worked
  for universal terms.

## Goal

Standardize every gohai JSON field name using industry-standard schemas where
they have coverage, and a documented convention for the long tail. Produce a
verifiable mapping table with citations and an OCSF gap report that feeds
upstream contributions.

## Design

### Three-Tier Naming Ladder

One precedence order. No voting. No corpus aggregation.

#### Tier 1: OCSF (Primary Authority)

[OCSF](https://schema.ocsf.io/) (Open Cybersecurity Schema Framework) is the
primary naming authority. When OCSF has a field for the concept, use its name.

OCSF objects that map to gohai collectors:

| OCSF Object          | gohai Collectors                         |
| -------------------- | ---------------------------------------- |
| `device`             | hostname, machine_id, shard              |
| `device_hw_info`     | cpu (partial), memory (partial), dmi     |
| `os`                 | platform, kernel (partial), os_release   |
| `network_interface`  | network                                  |
| `software_package`   | package_mgr, packages                    |

Estimated coverage: ~80 fields.

The existing redundant-prefix rule applies: OCSF `device.cpu_count` under
gohai's `cpu` collector becomes `count`, not `cpu_count`.

#### Tier 2: OpenTelemetry Resource Semantic Conventions (Secondary)

When OCSF is silent but [OTel semconv](https://opentelemetry.io/docs/specs/semconv/resource/)
has a semantic convention, use OTel's name (leaf extracted, prefix-stripped same
as tier 1).

OTel namespaces that fill OCSF's gaps:

| OTel Namespace       | gohai Collectors                                    |
| -------------------- | --------------------------------------------------- |
| `host.cpu.*`         | cpu (model, family, stepping, vendor, cache)         |
| `system.memory.*`    | memory (state dimensions: used, free, cached, buffers) |
| `system.filesystem.*`| filesystem (type, mountpoint, mode)                  |
| `system.paging.*`    | memory (swap)                                        |
| `hardware.*`         | gpu, disk, dmi (battery, power, sensors)             |
| `host.*`             | uptime, hostname                                     |

Estimated coverage: ~60 additional fields.

#### Tier 3: gohai Convention (Long Tail)

For the remaining ~660 fields, apply a documented convention:

1. **Backing library name as starting point.** If gopsutil calls it `SwapFree`,
   our JSON key is `swap_free`. If ghw calls it `VendorID`, ours is `vendor_id`.
2. **`/proc` and `/sys` mirrors.** Fields that directly correspond to a kernel
   counter use the kernel's name lowercased to snake_case: `hugepages_total`,
   `mem_available`, `nr_dirty`.
3. **Unit suffixes when ambiguous.** `_bytes`, `_seconds`, `_percent`, `_mhz`.
   Omit when the unit is obvious from context (`total` in a memory object is
   clearly bytes).
4. **No abbreviations except universals.** Universal: `ip`, `mac`, `pid`, `uid`,
   `gid`, `mtu`, `fqdn`, `uuid`, `cidr`, `arn`, `id`. Everything else spelled
   out.
5. **Cloud canonical overlay.** Every cloud collector emits a common set of ~10
   fields with identical names regardless of provider (see "Cloud Canonical
   Fields" below). Provider-specific fields beyond these keep the provider API's
   native names.

#### Disagreement Handling

When OCSF and OTel disagree on the same concept (rare — e.g., OCSF `ram_size`
vs OTel `system.memory` with state labels), tier 1 (OCSF) wins. Document the
OTel alternative in the mapping table's Notes column.

No multi-agent voting, no steelmanning, no critic roles. The tier precedence IS
the tiebreaker.

### Cloud Canonical Fields

Every cloud collector (ec2, gce, azure, digital_ocean, openstack, alibaba,
linode, oci, scaleway) emits these cross-provider fields with identical JSON
keys:

| Canonical Field | Description                                      |
| --------------- | ------------------------------------------------ |
| `instance_id`   | Provider-assigned unique instance identifier     |
| `region`        | Cloud region                                     |
| `zone`          | Availability zone within region                  |
| `instance_type` | Instance size/shape (e.g., `m5.xlarge`, `e2-medium`) |
| `hostname`      | Instance hostname from provider metadata         |
| `public_ips`    | Public IP addresses (array of strings)           |
| `private_ips`   | Private IP addresses (array of strings)          |
| `account_id`    | Cloud account/project identifier                 |
| `image_id`      | Machine image identifier (AMI, image name, etc.) |
| `tags`          | Provider tags/labels (map of string to string)   |

Provider-specific fields beyond this set retain the provider API's native names.
The mapping from each provider's raw metadata to these canonical fields is
documented in the cloud canonical overlay deliverable.

## Process

### Phase 1: Build the Mapping Table

A single markdown file (`schemas/field-mapping.md`) with one row per gohai
field. Every field in every collector's `Info` struct gets a row.

**How to build:**

1. Enumerate all fields from Go structs across `pkg/gohai/collectors/`
2. For each field, look up OCSF objects (device, device_hw_info, os,
   network_interface, software_package) at
   [schema.ocsf.io](https://schema.ocsf.io/) and the
   [ocsf-schema GitHub repo](https://github.com/ocsf/ocsf-schema)
3. If no OCSF match, check OTel semconv at the
   [semantic-conventions GitHub repo](https://github.com/open-telemetry/semantic-conventions)
4. If neither, apply tier-3 convention and mark as OCSF gap candidate

**Execution:** One agent pass per collector. No multi-role pipeline. The tier
precedence eliminates judgment calls for most fields.

### Phase 2: Apply Renames

For each collector where a field name changes:

1. Rename Go struct field + JSON tag
2. Update collector doc's Collected Fields table
3. Update tests
4. One PR per collector (or batch small ones)

Pre-1.0, so breaking changes are acceptable.

### Phase 3: Publish Gap List and Cloud Overlay

Extract OCSF gap candidates into `schemas/ocsf-gaps.md`. Extract cloud
canonical mappings into `schemas/cloud-canonical.md`. These become the upstream
contribution roadmap.

## Deliverables

### 1. Field Mapping Table (`schemas/field-mapping.md`)

Every row includes:

| Column         | Description                                            |
| -------------- | ------------------------------------------------------ |
| **Collector**  | Which gohai collector owns this field                   |
| **Go Field**   | Current Go struct field name                            |
| **Current JSON** | Current `json:"..."` tag value                       |
| **Tier**       | 1 (OCSF), 2 (OTel), or 3 (Convention)                  |
| **Chosen JSON** | The name we're adopting                                |
| **Changed?**   | `yes` / `no`                                            |
| **Source**      | Which schema object/attribute this comes from           |
| **Citation**   | Clickable link to the exact definition (see below)      |

**Citation requirements:**

- **Tier 1 (OCSF):** GitHub permalink to the object definition, e.g.,
  `https://github.com/ocsf/ocsf-schema/blob/main/objects/device_hw_info.json`
  or the rendered schema site
  `https://schema.ocsf.io/1.4.0/objects/device_hw_info`
- **Tier 2 (OTel):** GitHub permalink to the semconv YAML, e.g.,
  `https://github.com/open-telemetry/semantic-conventions/blob/main/model/host/registry.yaml`
- **Tier 3 (Convention):** No external citation. Note the backing library field
  name that inspired it (e.g., "gopsutil `SwapFree`", "ghw
  `Product.SerialNumber`", "kernel `/proc/meminfo` `HugePages_Total`")

**No fabricated links.** Every tier-1 and tier-2 citation must point to an
actual file or page. The agent building the table verifies each link resolves
before recording it.

### 2. OCSF Gap Report (`schemas/ocsf-gaps.md`)

Grouped by the OCSF object the field would most naturally extend. Each entry
includes:

- **What it is** — one-line description of the concept
- **Why OCSF doesn't have it** — the schema design reason (event-centric focus,
  too domain-specific, etc.)
- **Why it should exist** — the inventory/security justification for adding it
- **OTel/ECS precedent** — if another standard covers it, link to their
  definition as supporting evidence for the OCSF PR

This report is the input for future OCSF upstream PRs. Each entry contains
enough context to draft a proposal without re-researching.

### 3. Cloud Canonical Overlay (`schemas/cloud-canonical.md`)

The ~10 cross-provider fields with their per-provider mappings:

| Canonical  | EC2 Source             | GCE Source          | Azure Source         |
| ---------- | ---------------------- | ------------------- | -------------------- |
| `instance_id` | `instance-id`       | `id`                | `compute.vmId`       |
| `region`   | `placement.region`     | `zone` (trimmed)    | `compute.location`   |
| ...        | ...                    | ...                 | ...                  |

Plus DigitalOcean, OpenStack, Alibaba, Linode, OCI, Scaleway.

### 4. CLAUDE.md Update

Replace the current 5-level field naming precedence in CLAUDE.md with the
three-tier ladder:

1. **OCSF** — primary authority, cited per-field
2. **OTel semconv** — when OCSF is silent, cited per-field
3. **gohai convention** — documented rules, no external authority

Drop Ohai as a naming reference entirely (Ohai is for methodology, not naming).
Drop "industry standard" as a separate tier — ECS/osquery/Facter are useful
cross-references but not naming authorities.

## What We're NOT Doing

- **No `gohai.schema.json` initially.** The previous branch built a JSON Schema
  as the primary artifact. The mapping table is simpler and more actionable for
  driving renames. JSON Schema can come later from the mapping table.
- **No `x-rationale` annotations.** The per-field rationale pipeline was the
  bottleneck. The tier column IS the rationale.
- **No corpus fetching/grepping.** We look up OCSF and OTel docs directly. They
  are well-documented standards with browsable websites and GitHub repos.
- **No multi-agent verification.** The tier precedence eliminates voting. One
  agent pass per collector, verified by the human reviewer.
- **No Ohai-shaped JSON output.** We use Ohai for methodology (data sources,
  distro quirks, fallback chains). We never use Ohai for field naming.

## Relationship to Prior Work

The `feat/schema-corpus-tier1` branch produced useful artifacts that inform this
design:

- **DESIGN.md:** The top-level shape decision (Option A, flat by collector) and
  unit/casing/optionality conventions carry forward unchanged.
- **analysis.md:** The per-collector field tables are a useful starting point for
  the mapping table, though they need re-evaluation under the simplified tier
  system.
- **Corpus:** Not carried forward. The 30 corpus sources are replaced by direct
  lookups against OCSF and OTel.
- **Audit pipeline:** Not carried forward. The 4-role verifier process is
  replaced by single-pass mapping with human review.

## Open Questions

1. **OCSF version pinning:** Should we pin to a specific OCSF version (e.g.,
   1.4.0) or track latest? Recommendation: pin, update periodically.
2. **OTel semconv version pinning:** Same question. Recommendation: pin to the
   latest stable release.
3. **Rename execution order:** Should we rename all collectors in one big PR or
   one-per-collector? Recommendation: one per collector for reviewability, but
   batch small ones (e.g., init + root_group + shells in one PR).
