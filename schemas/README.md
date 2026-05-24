# Schemas

This directory holds the field-naming strategy, JSON Schema, and gap analysis
artifacts for gohai's ~950 JSON fields.

## What's here

| File                | Purpose                                                               |
| ------------------- | --------------------------------------------------------------------- |
| `field-mapping.md`  | **Source of truth.** Per-field tier mapping (~950 rows) with `Changed? yes/no` showing which fields need renaming to match OCSF/OTel |
| `ocsf-gaps.md`      | Fields where OCSF *should* have coverage but doesn't — upstream PR candidates for the OCSF schema repo |
| `gohai.schema.json` | Generated JSON Schema (draft 2020-12) for `gohai.Facts` — reflects current Go tags, regenerated after renames |
| `schema.go`         | `//go:embed` of the schema for the `gohai validate` command           |
| `gen/`              | Generator tool that reflects `gohai.Facts` into JSON Schema           |
| `references/`       | OCSF JSON + OTel YAML reference files used during audits              |
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
[`invopop/jsonschema`](https://github.com/invopop/jsonschema). It contains 157
`$defs` covering all 62 collector `Info` types and their nested structs.

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

## Workflow

### Per-field audit (the core loop)

For every JSON field in every collector, apply this sequence:

1. **Check OCSF first.** Open the reference schemas in `references/ocsf-*.json`
   and browse [schema.ocsf.io](https://schema.ocsf.io/). Does OCSF have a field
   for this concept? If yes:
   - The field is **T1**.
   - The JSON tag **MUST** use the OCSF field name (after applying the
     redundant-prefix rule — strip the parent-object prefix when it duplicates
     the collector name).
   - If our current JSON tag doesn't match, set `Changed? yes` in
     `field-mapping.md` and put the correct name in `Chosen JSON`.
2. **Check OTel next** (only if OCSF is silent). Open the semconv YAMLs in
   `references/otel-*.yaml`. Does OTel have an attribute for this concept?
   If yes:
   - The field is **T2**.
   - The JSON tag **MUST** use the OTel attribute name (last segment of the
     dotted path, after redundant-prefix stripping).
   - Same rename rule as above.
3. **Convention (T3).** Neither OCSF nor OTel covers it. Use the backing
   library's field name in `snake_case` with unit suffixes when ambiguous.
   No rename needed — document as T3.
4. **OCSF gap candidate?** If the field represents a concept OCSF *should*
   cover but doesn't, add it to `ocsf-gaps.md`.

### After the audit

5. **Update `field-mapping.md`** with correct tiers, `Changed?` flags, and
   `Chosen JSON` values.
6. **Rename Go code** — change struct field names and `json:"..."` tags to
   match `Chosen JSON` for every `Changed? yes` row.
7. **Run `just generate`** to regenerate `gohai.schema.json` with the new tags.
8. **Run tests** — `go test ./...` to verify nothing broke.

### Key rule

**The JSON tag must match the schema name.** If `field-mapping.md` says a field
maps to OTel `cloud.resource_id` but the JSON tag is `instance_id`, that's a
bug — the tag must be renamed to `resource_id`. The mapping document is not a
translation table; it's a declaration of what the field IS named, verified
against the schema source.
