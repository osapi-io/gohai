# OCSF + OpenTelemetry Schema + Ohai Sources

> **Status:** Implemented ✅

## Overview

gohai separates two decisions most fact collectors conflate:

- **What** each fact is called and shaped like → follows the
  [OCSF](https://schema.ocsf.io/) (Open Cybersecurity Schema Framework)
  vocabulary as the primary schema, with
  [OpenTelemetry Resource Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/resource/)
  as the secondary when OCSF is silent. OCSF is the industry schema backed by
  AWS and Splunk for asset / observability / security data. OpenTelemetry is
  the widely-adopted standard for observability telemetry and covers areas OCSF
  doesn't (per-CPU vendor/family/model, system load averages, process runtime
  metadata, host uptime). Aligning with both means gohai's output feeds SIEMs,
  data lakes, inventory tools, and OTel-compatible backends without
  translation.
- **How** each fact is collected (which file to read, which distro quirks to
  handle, which command to fall back to) → mirrors
  [Chef Ohai](https://github.com/chef/ohai/tree/main/lib/ohai/plugins)'s
  plugins. Ohai has years of accumulated bug fixes and per-distro knowledge;
  re-using their data sources means we inherit the knowledge without
  re-discovering the edge cases.

**We do NOT emit byte-for-byte Ohai-compatible JSON.** Ruby Mash ↔ Go struct
translation isn't worth pinning; every collector doc is explicit about which
Ohai plugin we mirror and any naming/shape differences (see each collector's
**Data Sources** section).

## How it shows up per collector

Every `docs/collectors/<name>.md` has:

1. **Collected Fields table** with a **Schema mapping column** — documents the
   canonical schema path each field maps to. OCSF first (e.g.
   `device.cpu_count`, `os.kernel_release`), OpenTelemetry second (e.g.
   `host.cpu.vendor.id`, `system.load_average.1m`), or explicit "No direct
   schema equivalent" with a one-line reason.
2. **Data Sources section** — step-by-step description of HOW the collector
   gathers data, in gohai's voice. Not a parity comparison with Ohai.
3. No "Known gaps vs. Ohai" section — methodology gaps live as GitHub issues
   labeled `methodology-gap` + `collector:<name>`.

This structure is enforced for every collector so drift against either source
is easy to spot.

## Field-naming precedence (applied repo-wide)

When picking a Go field name + JSON tag for a new field:

1. **OCSF** ([schema.ocsf.io/dictionary](https://schema.ocsf.io/dictionary)) —
   if OCSF names it, use that.
2. **OpenTelemetry** ([opentelemetry.io/docs/specs/semconv/resource/](https://opentelemetry.io/docs/specs/semconv/resource/))
   — when OCSF is silent.
3. **Industry standard** (node_exporter / systemd / Prometheus exporters) —
   when OCSF and OpenTelemetry are silent.
4. **Ohai's name** — only when the above are silent and Ohai has a clear
   meaningful name.
5. **Our own** — last resort, Go-idiomatic snake_case.

Both Go field names and JSON tags derive from the chosen schema name. The
JSON tag is the schema's leaf name verbatim (`json:"cpu_count"`); the Go field
is the PascalCase rendering (`CPUCount int`). When Go idiom on initialisms
conflicts (OCSF `cpu_id` → Go `CPUID`, not `CpuId`), Go convention wins the
field name but the JSON tag still matches the schema.

## Not reference schemas for our naming

- **Open Compute Project (OCP)** is a hardware design spec, not a data/naming
  schema. Ignore for field naming.
- **CIS / SCAP / XCCDF** describe compliance policies, not field schemas.
  Ignore for naming.
- **CloudEvents** is an event envelope spec. Ignore for naming.

## Related

- [CLAUDE.md — Implementation Methodology](../../CLAUDE.md#implementation-methodology)
  has the full field-naming rule and cross-reference requirements.
- [schema.ocsf.io/objects](https://schema.ocsf.io/objects) — the canonical OCSF
  object browser.
- [opentelemetry.io Resource attributes](https://opentelemetry.io/docs/specs/semconv/resource/)
  — OpenTelemetry semantic conventions for host / os / process / network
  resources.
