# JSON Schema Generator Design

**Date:** 2026-05-14
**Status:** Draft
**Scope:** Single Go tool that generates `schemas/gohai.schema.json` from
`gohai.Facts`

## Problem

gohai has 39 collectors producing ~803 JSON fields. The field names are now
standardized via a three-tier naming ladder (OCSF > OTel > convention) and
documented in `schemas/field-mapping.md`. But there's no machine-readable
schema describing the output shape. A JSON Schema would serve as:

- A **reference document** for new collector development — check the schema to
  see what fields and naming patterns exist
- A **contract** for consumers parsing gohai output
- A future input for editor autocomplete (schemastore.org) and OpenAPI `$ref`
  if OSAPI wraps gohai in a REST endpoint

## Design

### Generator Tool

A single Go binary at `schemas/gen/main.go` that uses
[`invopop/jsonschema`](https://github.com/invopop/jsonschema) to reflect the
`gohai.Facts` struct into JSON Schema (draft 2020-12).

The `Facts` struct already aggregates every collector's `Info` type via typed
pointer fields:

```go
type Facts struct {
    Platform  *platform.Info  `json:"platform,omitempty"`
    Hostname  *hostname.Info  `json:"hostname,omitempty"`
    CPU       *cpu.Info       `json:"cpu,omitempty"`
    // ... 39 collectors total
}
```

`invopop/jsonschema` reflects the entire struct tree in one call: reads
`json:"..."` tags, respects `omitempty` (field not required), and generates
`$defs` for every nested struct type.

### File Layout

```
schemas/
  gen/
    main.go           # generator tool — reflects Facts → JSON Schema
    generate.go       # //go:generate directive + package doc
  gohai.schema.json   # generated output, committed to repo
  field-mapping.md    # (existing) per-field mapping table
  ocsf-gaps.md        # (existing) OCSF contribution candidates
  cloud-canonical.md  # (existing) cross-provider field mapping
```

### Generator Implementation

```go
func main() {
    r := &jsonschema.Reflector{}
    schema := r.Reflect(&gohai.Facts{})

    schema.ID = jsonschema.ID("https://gohai.dev/schemas/gohai.schema.json")
    schema.Title = "gohai Facts"
    schema.Description = "System facts collected by gohai (https://github.com/osapi-io/gohai)"

    data, _ := json.MarshalIndent(schema, "", "  ")
    os.WriteFile(outPath, data, 0o644)
}
```

The tool accepts a `-out` flag for the output path (default:
`../gohai.schema.json` relative to the tool, i.e., `schemas/gohai.schema.json`).

### `//go:generate` Directive

```go
// schemas/gen/generate.go
package gen

//go:generate go run . -out ../gohai.schema.json
```

### Integration

- **`go.mod`:** Add `github.com/invopop/jsonschema` as a dependency (it's a
  library imported by the generator, not a tool binary).
- **`just generate`:** Already runs `go generate ./...` which will pick up the
  new directive at `schemas/gen/`.
- **`just ready`:** Already calls `just generate`, so the schema regenerates
  before every commit.
- **Tool registration:** The generator is `go run .` (local package), not an
  external tool binary, so no `tool` directive needed in `go.mod`.

### What the Schema Contains

- **Top-level `properties`:** One per `Facts` field (platform, hostname, cpu,
  memory, etc.) plus `collect_time`, `collect_duration_ns`, `_timings`.
- **`$defs`:** One per collector `Info` type and their nested sub-structs
  (e.g., `CPUInfo`, `MemoryInfo`, `Swap`, `Hugepages`, `DMIBIOS`, etc.).
- **Types:** Derived from Go types — `string`, `integer`, `number`, `boolean`,
  `array`, `object`. Maps become `object` with `additionalProperties`.
- **Required vs optional:** Fields without `omitempty` are required; fields with
  `omitempty` are optional (not in `required` array).

### What It Does NOT Do

- **No custom annotations.** No tier/citation metadata in the schema — that
  stays in `schemas/field-mapping.md`.
- **No CI enforcement.** No test that validates Go structs against the schema.
  Can be added later.
- **No schema versioning.** The `$id` URL is stable; the schema content changes
  when collectors change. Versioning can be added later if consumers need it.
- **No schemastore.org submission.** Future step once the schema stabilizes.

## Open Questions

None — this is intentionally minimal.
