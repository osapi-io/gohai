# Schemas

This directory holds the rules for field names, the JSON Schema, and the gap
analysis for the 950 JSON fields of gohai.

## What's here

| File                | Purpose                                                                                                                                      |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `field-mapping.md`  | **Source of truth.** The tier mapping of each field (964 rows). `Changed? yes/no` shows which fields need a new name to agree with OCSF/OTel |
| `ocsf-gaps.md`      | Fields that OCSF does not cover, but that it must cover. These are candidates for upstream PRs to the OCSF schema repo                       |
| `gohai.schema.json` | The generated JSON Schema (draft 2020-12) for `gohai.Facts`. It shows the current Go tags. Generate it again after a rename                  |
| `schema.go`         | The `//go:embed` of the schema for the `gohai validate` command                                                                              |
| `gen/`              | The generator tool. It reflects `gohai.Facts` into JSON Schema                                                                               |
| `ocsf-extension/`   | The gohai vendor extension (uid 1337) for the OCSF schema compiler                                                                           |
| `references/`       | The OCSF JSON + OTel YAML reference files for audits                                                                                         |

## Three-tier naming ladder

Each JSON field name comes from one of three tiers. Use the tiers in this order:

1. **OCSF** (108 fields). The primary authority. Refer to
   [schema.ocsf.io](https://schema.ocsf.io/).
2. **OpenTelemetry Resource Semantic Conventions** (92 fields). Use this tier
   when OCSF has no field. Refer to
   [OTel semconv](https://opentelemetry.io/docs/specs/semconv/resource/).
3. **gohai convention** (764 fields). The names of the backing library in
   `snake_case`. Add a unit suffix when the unit is not clear.

[`field-mapping.md`](field-mapping.md) has the full mapping of each field, with
citations that you can verify. [`ocsf-gaps.md`](ocsf-gaps.md) records the fields
that OCSF has no name for, as candidates for upstream contributions.

## JSON schema

[`invopop/jsonschema`](https://github.com/invopop/jsonschema) generates
`gohai.schema.json` from `gohai.Facts`. It contains 157 `$defs`. They cover the
`Info` types of all 63 collectors and their nested structs.

To generate the schema again:

```bash
just generate          # or: go generate ./schemas/gen/...
```

To validate gohai output against the schema:

```bash
gohai collect --pretty | gohai validate
gohai validate --file facts.json
```

`schema.go` embeds the schema into the `gohai` binary. Thus the validate command
works without the source tree.

## OCSF gap analysis

### Why gohai cares about OCSF

[OCSF](https://ocsf.io/) (Open Cybersecurity Schema Framework) is a
vendor-neutral schema for security events and asset inventory. AWS, Splunk and
more than 150 organizations support it. It defines a shared taxonomy. Thus
SIEMs, data lakes, vulnerability scanners and inventory tools use the same
names.

The gohai output is **asset inventory**. It tells what a host IS. Security
workflows use that data directly:

- **Vulnerability management.** "which hosts have this CPU flag / kernel version
  / SELinux mode?"
- **Compliance.** "do all hosts meet CIS benchmarks for X?"
- **Incident response.** "what was this host's hardware, software, and network
  state when the alert fired?"
- **Asset inventory.** "what's running where, what hardware, what containers,
  what services?"

Ask this question for each gohai field: **can an analyst use this field in a
normalized schema?** An analyst is a security analyst, a compliance auditor or
an incident responder. If yes, the field is an OCSF gap candidate.

### Gap candidates

[`ocsf-gaps.md`](ocsf-gaps.md) lists the fields that gohai reports but OCSF does
not cover yet. Each entry gives:

- What the field is.
- Why OCSF does not have it.
- Why it must exist.
- The OpenTelemetry precedent, if there is one.
- The gohai Go type.

These are candidates for upstream PRs to the
[OCSF schema repo](https://github.com/ocsf/ocsf-schema).

## Workflow

### Per-field audit (the core loop)

Do these steps for each JSON field in each collector:

1. **Check OCSF first.** Open the reference schemas in `references/ocsf-*.json`
   and look at [schema.ocsf.io](https://schema.ocsf.io/). If OCSF has a field
   for this concept:
   - The field is **T1**.
   - The JSON tag is the OCSF field name. First apply the redundant-prefix rule:
     remove the prefix of the parent object when it is the same as the collector
     name.
   - If our current JSON tag is different, set `Changed? yes` in
     `field-mapping.md`. Put the correct name in `Chosen JSON`.
2. **Then check OTel**, only if OCSF has no field. Open the semconv YAMLs in
   `references/otel-*.yaml`. If OTel has an attribute for this concept:
   - The field is **T2**.
   - The JSON tag is the OTel attribute name. This is the last segment of the
     dotted path, after you remove the redundant prefix.
   - The rename rule above also applies.
3. **Convention (T3).** Neither OCSF nor OTel covers the field. Use the field
   name of the backing library in `snake_case`. Add a unit suffix when the unit
   is not clear. No rename is necessary, so record the field as T3.
4. **OCSF gap candidate?** If OCSF must cover the concept of the field but does
   not, add it to `ocsf-gaps.md`.

### After the audit

5. **Update `field-mapping.md`** with the correct tiers, `Changed?` flags and
   `Chosen JSON` values.
6. **Rename the Go code**. For each `Changed? yes` row, change the struct field
   name and the `json:"..."` tag to the `Chosen JSON` value.
7. **Run `just generate`** to generate `gohai.schema.json` again with the new
   tags.
8. **Run the tests**. Run `go test ./...` to make sure that nothing broke.

### Key rule

**The JSON tag must be the same as the schema name.** For example,
`field-mapping.md` maps a field to OTel `cloud.resource_id`, but the JSON tag is
`instance_id`. That is a bug, and you must rename the tag to `resource_id`. The
mapping document is not a translation table. It declares the name that the field
HAS, verified against the schema source.
