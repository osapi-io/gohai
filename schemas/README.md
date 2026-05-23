# Schemas

This directory holds the field-naming strategy, JSON Schema, and gap analysis
artifacts for gohai's ~803 JSON fields.

## What's here

| File                  | Purpose                                                      |
| --------------------- | ------------------------------------------------------------ |
| `gohai.schema.json`   | Generated JSON Schema (draft 2020-12) for `gohai.Facts`      |
| `schema.go`           | `//go:embed` of the schema for the `gohai validate` command  |
| `field-mapping.md`    | Per-field tier mapping (803 rows): OCSF / OTel / convention  |
| `ocsf-gaps.md`        | 73 OCSF upstream PR candidates — fields OCSF should cover    |
| `cloud-canonical.md`  | 10 cross-provider canonical fields x 9 cloud providers       |
| `gen/`                | Generator tool that reflects `gohai.Facts` into JSON Schema  |
| `references/`         | OCSF JSON + OTel YAML reference files used during mapping    |
| `corpus/`             | External schema corpora (OSSEM) for cross-reference          |

## Three-tier naming ladder

Every JSON field name comes from one of three tiers, applied in strict order:

1. **OCSF** (~108 fields) — primary authority. Browse
   [schema.ocsf.io](https://schema.ocsf.io/).
2. **OpenTelemetry Resource Semantic Conventions** (~74 fields) — when OCSF is
   silent. See
   [OTel semconv](https://opentelemetry.io/docs/specs/semconv/resource/).
3. **gohai convention** (~768 fields) — backing library names in `snake_case`
   with unit suffixes when ambiguous.

The complete per-field mapping with verifiable citations lives in
[`field-mapping.md`](field-mapping.md). Fields where OCSF is silent are tracked
in [`ocsf-gaps.md`](ocsf-gaps.md) as upstream contribution candidates.

## JSON Schema

`gohai.schema.json` is generated from `gohai.Facts` via
[`invopop/jsonschema`](https://github.com/invopop/jsonschema). It contains 118
`$defs` covering all 39 implemented collector `Info` types and their nested
structs.

Regenerate:

```bash
just generate          # or: go generate ./schemas/gen/...
```

Validate gohai output against the schema:

```bash
gohai collect --pretty | gohai validate
gohai validate --file facts.json
```

The schema is embedded into the `gohai` binary via `schema.go` so the validate
command works without the source tree.

## OCSF gap analysis

[`ocsf-gaps.md`](ocsf-gaps.md) lists 76 fields that gohai emits but OCSF
doesn't yet cover. Each entry includes:

- What the field is
- Why OCSF lacks it
- Why it should exist
- OpenTelemetry precedent (if any)
- The gohai Go type

These are candidates for upstream PRs to the
[OCSF schema repo](https://github.com/ocsf/ocsf-schema).

## Cloud canonical overlay

[`cloud-canonical.md`](cloud-canonical.md) maps 10 cross-provider fields
(`instance_id`, `region`, `zone`, `account_id`, `image_id`, `instance_type`,
`provider`, `availability_zone`, `hostname`, `network`) across all 9 cloud
providers, noting per-provider derivation quirks.

## Workflow

When adding or renaming fields:

1. Check `field-mapping.md` for the current tier assignment and citation.
2. If the field is new, assign a tier (OCSF first, then OTel, then convention).
3. Run `just generate` to regenerate `gohai.schema.json`.
4. If the field exposes something OCSF should cover, add it to `ocsf-gaps.md`.
