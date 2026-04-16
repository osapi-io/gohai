# Schemas

This directory hosts the **gohai schema** — the canonical contract
describing gohai's output shape — and the **corpus** of prior-art
schemas used to inform field-by-field naming decisions.

## Layout

```
schemas/
├── README.md              ← this file
├── corpus/                ← committed prior-art snapshot
│   ├── ocsf/              ← Open Cybersecurity Schema Framework
│   ├── otel/              ← OpenTelemetry semantic conventions
│   ├── osquery/           ← osquery table specs
│   ├── ecs/               ← Elastic Common Schema
│   ├── redfish/           ← DMTF Redfish (hardware / BMC)
│   ├── k8s/               ← Kubernetes NodeStatus / NodeInfo
│   ├── ohai/              ← Chef Ohai plugins + specs
│   └── facter/            ← Puppet Facter fact schema
└── gohai.schema.json      ← (TBD) the canonical output contract
```

## The corpus

`corpus/` contains upstream schema sources committed at a point in time.
Each subdirectory carries a `PROVENANCE` file recording the source URL,
fetched commit SHA, and fetch timestamp.

### Why committed?

- **Reproducibility.** Anyone cloning the repo gets the exact corpus
  the schema was designed against. Upstream repos change; our
  analysis is anchored.
- **Auditability.** Every naming decision cites a corpus source; the
  source is right there to verify.
- **Forkability.** Downstream consumers can reuse the corpus without
  re-fetching every repo.

### Refreshing

Re-run `scripts/corpus-fetch.sh` to update every source to its current
upstream tip. Review the diff — upstream schema churn may change
naming decisions; the diff is where that review happens.

### License + provenance

Each source subdirectory preserves its upstream `LICENSE` file and
carries a `PROVENANCE` file documenting origin. All corpus material is
redistributed under its upstream license. See each subdirectory for
specifics; all tier-1 sources currently in the corpus are permissively
licensed (Apache-2, BSD-3, MIT).

## The schema (TBD)

`gohai.schema.json` will be the canonical output contract. It's
hand-written (the schema IS the spec, Go types conform to it, not the
other way around) and informed by the corpus. Status: in design.

Field-by-field analysis, naming rationale, and the final schema are
tracked in follow-up PRs.

### Scope

The schema covers every shipping gohai collector — the full `Facts`
object including nested `Info` structs. Planned-but-unimplemented
collectors (🚧 in [`docs/collectors/README.md`](../docs/collectors/README.md))
are reserved but not specified until they ship.

### Versioning

Semver. Breaking changes to field names, types, or shape require a
major version bump. The schema carries `$schema` and `$id` for
self-identification.

## Sources in the corpus

### Tier 1 — direct scope match

| Source | Role |
| ------ | ---- |
| **OCSF** (`corpus/ocsf/`) | Open Cybersecurity Schema Framework. AWS/Splunk-backed, LF-hosted. Objects (`device`, `os`, `network_interface`, `cloud`, etc.) + events. |
| **OpenTelemetry** (`corpus/otel/`) | Resource semantic conventions. Host, OS, CPU, cloud, process attributes. |
| **osquery** (`corpus/osquery/`) | Table specs for host-inventory SQL queries. 200+ tables across Linux + macOS. |
| **Elastic Common Schema** (`corpus/ecs/`) | Field naming for observability + security. Host, user, process, network, cloud namespaces. |
| **DMTF Redfish** (`corpus/redfish/`) | REST schema for hardware / BMC. Chassis, ComputerSystem, Memory, Processor, etc. |
| **Kubernetes** (`corpus/k8s/`) | `NodeStatus` / `NodeInfo` / `NodeSystemInfo` from `core/v1/types.go`. Production inventory shape for millions of nodes. |
| **Chef Ohai** (`corpus/ohai/`) | Primary methodology reference for gohai. Plugin output shapes. |
| **Puppet Facter** (`corpus/facter/`) | Puppet's fact collector; peer to Ohai. |

### Tier 2+ (to be added)

Planned additions to expand coverage for cloud-provider IMDS shapes,
SIEM/security vocabularies (ASIM, UDM, Splunk CIM, OSSEM), software
identifiers (CPE, SWID, SPDX, CycloneDX, PURL), DMTF CIM, and assorted
vendor configs.
