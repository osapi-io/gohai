# OCSF Schema + Ohai Sources

> **Status:** Implemented ✅

## Overview

gohai separates two decisions most fact collectors conflate:

- **What** each fact is called and shaped like → follows the
  [OCSF](https://schema.ocsf.io/) (Open Cybersecurity Schema Framework)
  vocabulary. OCSF is the industry schema backed by AWS and Splunk for asset /
  observability / security data, so gohai's output feeds SIEMs, data lakes, and
  inventory tools without translation.
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

1. **Collected Fields table** with an **OCSF mapping column** — documents the
   OCSF path (`os.kernel_release`, `device.hostname`, etc.) each field maps to,
   or explicit "No OCSF equivalent" with reason.
2. **Data Sources table** — two rows (Linux, macOS) showing what gohai reads,
   what the corresponding Ohai plugin reads, and whether the two are equivalent
   / we extend / we diverge and why.
3. **Known gaps** callout — any Ohai coverage we don't yet mirror, with a note
   of what it would take to add.

This structure is enforced for every collector so drift against either source is
easy to spot.

## Field-naming precedence (applied repo-wide)

When picking a JSON tag for a new field:

1. **OCSF** ([schema.ocsf.io/dictionary](https://schema.ocsf.io/dictionary)) —
   if OCSF names it, use that.
2. **Industry standard** (node_exporter / systemd / Prometheus exporters) — when
   OCSF is silent.
3. **Ohai's name** — only when the above two are silent and Ohai has a clear
   meaningful name.
4. **Our own** — last resort, Go-idiomatic snake_case.

## Related

- [CLAUDE.md — Implementation Methodology](../../CLAUDE.md#implementation-methodology)
  has the full field-naming rule and cross-reference requirements.
- [schema.ocsf.io/objects](https://schema.ocsf.io/objects) — the canonical OCSF
  object browser.
