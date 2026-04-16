# gohai Schema Design Principles

Decisions that shape `gohai.schema.json` before we draft a single field.
This doc needs review + sign-off before field-by-field analysis begins.

---

## 1. Top-level shape

**The question:** should gohai output be flat-by-collector (what we have
today) or nested-by-domain (what OCSF and Redfish do)?

### Option A — Flat by collector (current)

```json
{
  "platform": { "name": "Ubuntu", "version": "24.04" },
  "kernel": { "release": "6.8.0-45-generic", "rosetta_translated": false },
  "cpu": { "count": 16, "cores": 8, "model_name": "Intel Xeon ..." },
  "memory": { "total": 68719476736, "free": 42949672960 },
  "hostname": { "name": "web-01", "fqdn": "web-01.example.com" },
  "hardware": { "machine_model": "MacBookPro18,2", ... },
  "dmi": { "bios": { "vendor": "..." } },
  "filesystem": { "mounts": [...] },
  "network": { "interfaces": [...] },
  "cloud": { "provider": "aws" },
  "ec2": { "instance_id": "i-abc", ... }
}
```

**Pros:**
- Matches gohai's Go package structure 1:1. `facts.Platform`,
  `facts.CPU`, `facts.Hardware` map directly to their collectors.
- Easy for consumers to enable / disable specific collectors and know
  exactly which JSON key appears or disappears.
- No naming collisions — every collector owns its namespace.
- Matches how Ohai, Facter, Ansible, and osquery shape their output.

**Cons:**
- Duplication: `dmi.product.serial_number` vs `hardware.serial_number`
  are the same concept on different platforms. Consumers have to
  branch on OS.
- Semantic clustering is weak — CPU count, memory total, and BIOS
  version are all "hardware" but live in separate keys.
- Cloud provider metadata spreads across `cloud` + `ec2` + `gce` +
  `azure` etc.

### Option B — Nested by domain (OCSF / Redfish style)

```json
{
  "device": {
    "hostname": "web-01",
    "fqdn": "web-01.example.com",
    "uid": "abc-1234-...",
    "os": {
      "name": "Ubuntu",
      "version": "24.04",
      "kernel_release": "6.8.0-45-generic",
      "init": "systemd"
    },
    "hw_info": {
      "cpu": { "count": 16, "cores": 8, "model_name": "..." },
      "memory": { "total": 68719476736 },
      "bios": { "vendor": "...", "version": "...", "date": "..." },
      "chassis": { "vendor": "Dell", "serial_number": "..." },
      "product": { "name": "MacBookPro18,2", "uuid": "..." },
      "gpu": [...],
      "pci": [...],
      "scsi": [...]
    },
    "filesystem": { "mounts": [...] },
    "network": { "interfaces": [...], "default_interface": "eth0" },
    "virtualization": { "system": "docker", "role": "guest" }
  },
  "cloud": {
    "provider": "aws",
    "region": "us-east-1",
    "ec2": { "instance_id": "i-abc", ... }
  },
  "runtime": {
    "uptime_seconds": 86400,
    "load_avg": { "one": 0.4, "five": 0.3, "fifteen": 0.2 },
    "users": { ... },
    "sessions": [...],
    "processes": [...]
  }
}
```

**Pros:**
- Semantically clustered — "all OS facts", "all hardware facts" live
  together. Consumers can query `device.hw_info.cpu.cores` without
  caring which collector produced it.
- Aligns with OCSF's `device` object — mechanical mapping for SIEM
  consumers.
- Hardware abstraction unifies Linux (`dmi` + `hardware`) and macOS
  (`hardware` only) into one `hw_info` hierarchy.
- Cleaner as an external contract. Less coupled to gohai's internal
  collector package layout.

**Cons:**
- Breaking change on every current consumer (OSAPI + any importer).
- Enable/disable collectors no longer maps 1:1 to JSON keys — the
  user toggling `--collector.dmi` expects `facts.DMI` to
  appear/disappear, but under Option B its data lives at
  `device.hw_info.bios` / `.chassis` / `.product`.
- Per-collector docs and Schema mapping columns need restructuring.
- A few collectors have no obvious home: `shard`, `timezone`, `fips`,
  `shells`, `machine_id`, `lsb`, `init`, `os_release`.

### Hybrid — flat top-level with nested-by-domain sub-objects

Keep the collector-level top-level keys but nest common hardware and
OS concepts under shared objects:

```json
{
  "platform": { ... },
  "kernel": { ... },
  "hardware": {
    "cpu": {...},       // reference to cpu collector's output
    "memory": {...},    // reference to memory collector's output
    "dmi": { "bios": {...}, "chassis": {...} },
    "gpu": [...],
    "pci": [...]
  },
  ...
}
```

Only works if we're willing to put some collector output in two
places (`facts.CPU` AND `facts.Hardware.CPU`) or rename collectors so
the flat top-level collectors ARE the nested structure. Messy.

### Recommendation

**Option A.** Reasons:

1. **Consumer compatibility.** Every current importer (OSAPI being
   primary) and every doc / test / example references `facts.CPU`,
   `facts.Kernel`, etc. Option B is a total rewrite of that surface.
2. **Collector toggle semantics.** `--collector.dmi` disappearing
   `facts.DMI` is a contract consumers rely on. Option B breaks that
   cleanly — where does `facts.Device.HWInfo.BIOS` go when dmi is
   disabled?
3. **Aligned with peer tools.** Ohai, Facter, Ansible, osquery all
   use the flat-by-collector shape. Option B would make gohai's
   output oddly distinct among peers.
4. **Option B's win is theoretical.** The "SIEM consumer wants a
   clean `device.hw_info` path" user doesn't really exist today;
   it's hypothetical. The Go importer user definitely exists.
5. **Cross-collector consistency can be fixed under A.** The real
   duplication problems (hardware identity on Linux vs macOS; cloud
   provider metadata across providers) can be solved with naming
   conventions, not with restructure.

**Decision required before field-by-field begins.** Default to A
unless there's a strong argument for B.

---

## 2. Naming conventions

These apply regardless of Option A or B.

### 2.1 Casing

- **JSON keys:** `snake_case`. Always.
- **Go field names:** PascalCase with idiomatic initialisms (`CPUID`,
  `HTTPClient`). Go convention wins for field names when there's a
  conflict; JSON tag controls the wire.

### 2.2 Field naming precedence

For every field, the source of the name is chosen in this order:

1. **Clear industry standard** — when a term has a single
   unambiguous canonical form across the corpus (e.g. `hostname`,
   `serial_number`, `mac`, `pid`).
2. **OCSF canonical name** — when OCSF has a clean match.
3. **OpenTelemetry semantic convention** — when OCSF is silent.
4. **osquery / Redfish / ECS name** — when the above are silent but
   two or more of these agree.
5. **gohai's own snake_case choice** — last resort, Go-idiomatic.

Rationale: multi-source consensus is the signal of "industry
standard." Single-source names get lower priority because they're
vendor vocabularies. OCSF/OTel get elevated among single sources
because they're designed as standards (even if adoption is uneven).

### 2.3 Redundant-prefix rule (from CLAUDE.md)

Strip parent-object prefixes when they duplicate our key. OCSF
`device.cpu_count` nested under `cpu` becomes `count`, not
`cpu_count`. Rationale: the parent key already provides the
namespace.

Examples:

| OCSF path | Under gohai key | Stripped leaf |
| --------- | --------------- | ------------- |
| `device.cpu_count` | `cpu` | `count` |
| `device.cpu_cores` | `cpu` | `cores` |
| `device.memory_size` | `memory` | `size` |
| `os.kernel_release` | `kernel` | `release` |

### 2.4 Collision handling

When two upstream sources pick different names for the same concept,
document the decision:

- Prefer shorter, clearer names.
- Prefer names that match Go stdlib if relevant (e.g. `proc.name`
  rather than `proc.command` because Go's `os/exec` uses `Command`).
- Break ties toward OCSF.

Record the chosen name + rationale in the field-by-field analysis
table.

---

## 3. Unit conventions

### 3.1 Time

- **Durations:** integer **seconds** (`uint64`). Not milliseconds, not
  nanoseconds. Exception: if we ever need sub-second precision for a
  specific field, mark it with a `_ns` or `_ms` suffix.
- **Timestamps:** Unix epoch **seconds** as `int64`, named with
  `_time` suffix (`boot_time`, `collect_time`). Not ISO-8601 strings
  — consumers can format, strings lose type info.
- **Human-readable durations:** separate `_human` field (string),
  sibling to the `_seconds` field. Format is gohai's: `1d 2h 3m 4s`
  (compact, not pluralized).

### 3.2 Size

- **Bytes.** Always. `uint64`. No MB / GB / "16 GB" strings.
- Exception: verbatim passthrough from OS tools like
  `system_profiler` where the string contains the unit
  (`physical_memory: "16 GB"`). These are clearly-marked "raw OS
  strings" and live alongside a parsed `_bytes` sibling where possible.

### 3.3 Percent

- `0-100` integer. Not `0.0-1.0` float. Not `"85%"` string. Named
  with `_percent` suffix when the scale isn't obvious from the name.

### 3.4 Booleans

- Real JSON booleans. Not `"yes"`/`"no"`, not `"TRUE"`/`"FALSE"`,
  not `0`/`1`. OS tools that return string booleans get normalized
  at the collector layer.

### 3.5 Enums

- Lowercase `snake_case` strings. Document accepted values per field.
- Never integer codes.

---

## 4. Optionality and null

- **`omitempty` by default** on every non-required field. Absent
  means "we didn't collect it" — not "it's zero". Zero values are
  ambiguous for consumers.
- **Required fields** get no `omitempty` and are documented as
  always-present. Candidate list: per collector, a single identity
  field (hostname, instance_id, uid) typically.
- **`null` not allowed** anywhere. Absent = omitted.

---

## 5. Versioning

### 5.1 Schema versioning

- `$schema` — JSON Schema draft reference (`draft-2020-12`).
- `$id` — stable URL for this schema version.
- Semver applied to the schema itself (major bump on breaking
  changes to field names, types, or required-ness).
- Embed schema version in every output payload under `_schema` key:

```json
{
  "_schema": {
    "version": "1.0.0",
    "uri": "https://schema.gohai.dev/v1/gohai.schema.json"
  },
  "platform": { ... }
}
```

### 5.2 gohai version

- Embed the gohai binary / library version separately under the same
  `_meta` or `_schema` key so consumers can correlate collector bugs
  to gohai releases.

```json
{
  "_meta": {
    "schema_version": "1.0.0",
    "gohai_version": "0.5.0",
    "collected_at": 1712908800
  }
}
```

### 5.3 Compatibility

- Within a major version: additions only. New fields, new
  collectors. Never a rename, retype, or removal.
- Across major versions: breaking changes allowed, documented in
  CHANGELOG, migration guide required.

---

## 6. Extension mechanism

Consumers need a place to park gohai-unaware data without polluting
the schema or requiring a major version bump. Two standard
extension slots:

### 6.1 `_vendor` — per-namespace vendor extensions

```json
{
  "ec2": {
    "instance_id": "i-abc",
    "_vendor": {
      "osapi": {
        "region_tag": "prod-us-east-1",
        "datacenter": "iad-01"
      }
    }
  }
}
```

Consumers who enrich gohai output put their fields under
`_vendor.<namespace>`. The schema reserves `_vendor` as an `object`
with free-form `additionalProperties`; nothing inside is validated.

### 6.2 `_raw` — experimental / pre-schema fields

Fields that a collector starts capturing but aren't yet part of the
schema go under `_raw`:

```json
{
  "ec2": {
    "instance_id": "i-abc",
    "_raw": {
      "imds_latency_ms": 42,
      "experimental_new_field": "value"
    }
  }
}
```

Absorbed into the schema proper (promoted out of `_raw`) in the next
minor version if they stabilize. Removed entirely if they don't.

---

## 7. What's NOT in scope

- **Event wrappers** (class_uid / activity_id / severity_id /
  metadata.product). gohai is an inventory SDK, not an event emitter.
  Consumers who bridge to OCSF events wrap our output themselves.
- **Time-series / metric shape.** gohai collects point-in-time facts.
  Runtime metrics (CPU usage %, memory working set) that change
  continuously are out of scope — that's Prometheus territory.
- **Security findings.** No vulnerabilities, CVEs, misconfig
  detections. That's osquery / agent-side tooling.
- **Active remediation.** Read-only by design.

---

## 8. Open questions (require decision)

1. **Top-level shape:** Option A (flat) vs Option B (nested). See §1.
2. **Schema URL:** where does `gohai.schema.json` live? Options:
   - `https://schema.gohai.dev/v1/gohai.schema.json` (needs domain)
   - `https://osapi.io/schemas/gohai/v1.json` (OSAPI domain)
   - `https://github.com/osapi-io/gohai/blob/main/gohai.schema.json`
     (GitHub raw — simplest but least "official")
3. **schemastore.org submission:** do we submit for editor
   autocomplete? Probably yes but not blocker.
4. **OCSF upstream contribution:** once the schema is stable, do we
   propose it as an OCSF community extension for the
   Device Inventory gap? Aspirational — not in scope for v1.

---

## 9. Next phase

After these decisions:

1. **Field-by-field analysis** — a big table with one row per current
   gohai field, showing OCSF / OTel / osquery / ECS / Redfish / Ohai
   / Facter / Wazuh candidate names, consensus signal, chosen name,
   rationale. Lives at `schemas/analysis.md`.
2. **Draft `gohai.schema.json`** — hand-written from the analysis,
   following the conventions above.
3. **Conformance test** — Go reflection of `Facts` + every `Info`
   struct asserted against the hand-written schema. CI fails on
   drift.
4. **Go-type refactor** — rename Go fields + JSON tags to match the
   schema. Breaking change. Pre-1.0.
5. **Publish** — schemastore.org + stable URL + README + CHANGELOG.
