# Collector Methodology

How gohai decides what a collector reads, which library it wraps, and what its
fields are called. This is reference material for writing or modifying a
collector; see [CONTRIBUTING](../CONTRIBUTING.md) for setup and workflow.

## Implementation Methodology

**We are a wrapper and aggregator, not a re-implementor.** Each collector's job
is to wrap a well-maintained backing source (Go library, provider SDK, or a thin
file/command parser) and reshape its output into our typed `Info` struct.

### Extend upstream, don't replace

**MANDATORY:** When a Go library (gopsutil, ghw, procfs, cloud SDKs) covers part
of what a collector needs, use it for that part. If the library doesn't cover
everything we want, add the extension logic **in our collector** on top of the
library's output. Do not replace the library wholesale just because it's missing
one piece, you lose years of accumulated bug fixes and cross-platform handling
that way.

Extension pattern:

```go
// Wrap the library for what it does well.
info, err := upstream.Get(ctx)
if err != nil { return nil, err }

// Layer our extension on top for the gap.
if shouldAddOurBit(info) {
    info.OurField = readOurSource()
}
```

Only replace the library entirely when its scope genuinely doesn't match what
the collector is supposed to report (e.g. a library mixes sources in a way that
produces an ambiguous value, or its output isn't reshapeable into the right
semantics). When you do replace, justify the decision in the collector's Data
Sources doc.

**Decision order for each collector:**

1. **[ghw]**. Canonical for physical hardware topology: CPU NUMA
   - arch-aware counts, memory DIMMs/page-sizes, block devices with
     UUID/label/unmounted, network drivers/speed, DMI (baseboard / BIOS /
     chassis / product), GPU, PCI. Use ghw first for anything about static
     hardware shape.
2. **[gopsutil]**. Canonical for dynamic runtime state: memory
   free/available/used, disk I/O counters, network I/O counters, process
   enumeration, sessions (utmp), virtualization detection, host info. Use
   gopsutil for anything that changes per collection.
3. **[go-sysinfo]**. Alternative for host / platform / kernel where gopsutil is
   weaker. Evaluate case-by-case; don't stack both for the same fact.
4. **[procfs]**. Raw Linux `/proc` and `/sys` parsing when none of the above
   cover a field. Preferred over rolling our own scanner.
5. **Official provider SDKs** (aws-sdk-go, google.golang.org/cloud,
   azure-sdk-for-go) for cloud collectors; plain `net/http` to IMDS endpoints
   when the SDK is too heavy.
6. **Our own extension**. Last resort. ONLY the fields the libraries above don't
   expose. Extensions read files via `avfs.VFS` and shell out via
   `executor.Executor` so tests never touch the real host.
7. **[Ohai's Ruby plugin][ohai-plugins] as methodology reference only**. NOT an
   import. We read Ohai to learn WHICH edge cases exist (fallback chains, distro
   quirks, retries). We then check whether ghw/gopsutil/stdlib already cover
   them. Only the residual gap becomes our extension code.

**We learn from, but don't directly import, [node_exporter]**, their collectors
are a gold reference for tricky Linux `/proc` and `/sys` parsing (Apache-2
licensed). Read, understand, rewrite in our style.

### Library-first principle

**Never roll your own parsing when a library covers it.** If gopsutil reads
`/proc/meminfo` already. We don't write a second `/proc/meminfo` parser, and we
surface the fields gopsutil already exposes on our typed `Info`. If ghw
enumerates block devices with UUID/label, we don't shell out to `lsblk`.

Before implementing or extending a collector, verify in this order:

1. Does our primary library for this collector expose the field? (Check the
   library's Go source, not docs. Docs may undersell.)
2. If no: does a secondary library (the next one down in the Decision order)
   expose it?
3. If still no: we need an extension. The extension uses `avfs.VFS` for file
   reads and `executor.Executor` for exec calls, never plain `os.ReadFile` /
   `exec.Command` in collector Collect methods.

### Per-collector library stack

Primary library for each collector. Changes require a PR updating this table
with rationale.

| Collector              | Primary               | Candidate migration / supplement           |
| ---------------------- | --------------------- | ------------------------------------------ |
| cpu                    | gopsutil              | ghw/cpu for NUMA/topology/arch-math        |
| memory                 | gopsutil              | ghw/memory for hugepages/page-sizes        |
| filesystem             | gopsutil              | ghw/block for UUID/label/unmounted         |
| disk                   | gopsutil              | ghw/block for device metadata              |
| network                | gopsutil              | ghw/net for driver/speed                   |
| hostname               | gopsutil + stdlib     | —                                          |
| platform               | gopsutil              | go-sysinfo alternative considered          |
| uptime                 | gopsutil              | —                                          |
| kernel                 | `x/sys/unix` + stdlib | —                                          |
| load                   | gopsutil              | —                                          |
| process                | gopsutil              | , (ghw doesn't do processes)               |
| users (sessions)       | gopsutil (utmp)       | supplement with loginctl via executor      |
| virtualization         | gopsutil              | go-sysinfo has some                        |
| fips                   | stdlib                | No library covers                          |
| machine_id             | gopsutil + stdlib     | stdlib fallback chain                      |
| shard                  | stdlib + machine_id   | —                                          |
| init                   | stdlib                | `/proc/1/comm`                             |
| os_release             | stdlib                | Our own parser                             |
| lsb                    | stdlib                | supplement with `lsb_release` via executor |
| shells                 | stdlib                | —                                          |
| timezone               | stdlib                | —                                          |
| root_group             | stdlib (`os/user`)    | —                                          |
| package_mgr            | stdlib exec           | executor-based                             |
| dmi                    | **ghw**               | baseboard + BIOS + chassis + product       |
| gpu                    | **ghw**               | —                                          |
| pci                    | **ghw**               | —                                          |
| block_device (planned) | **ghw**               | —                                          |

New collectors must justify library choice in their PR. Migrations (gopsutil →
ghw, etc.) need their own issue labeled `library-migration` + `collector:<name>`
with: current coverage, candidate coverage, migration plan.

### Cross-platform compilation: no build tags (osapi pattern)

**MANDATORY:** Collector code must compile on every target platform, with **no
`//go:build` tags anywhere**. This is the pattern OSAPI uses in
`internal/provider/`, study that code before writing a new collector. Result:
`go test ./...` on any dev machine compiles and runs every collector's tests,
coverage is visible cross-platform, and CI on linux runners still validates
actual linux runtime behavior.

Shape (see [adding-a-collector.md](adding-a-collector.md) for full code
examples):

```
pkg/gohai/collectors/<name>/
  <name>.go                # Info, Collector interface, base, New() factory
  linux.go / darwin.go     # type Linux / Darwin struct; implements Collector
  debian.go / rhel.go      # (only when distro diverges from generic linux)
  export_test.go           # SetXFn setters (upstream-library seams) for external tests
  <name>_public_test.go    # TestNew dispatch + single table-driven TestCollect
                           # keyed by a `variant` column that builds the right per-OS
                           # struct; compile-time interface asserts at the top.
```

**One TestCollect per collector, end of story.** No `linux_public_test.go` or
`darwin_public_test.go` files. Both OS variants are tested through the same
`TestCollect` method with a `variant: "linux" | "darwin"` column dispatching to
`&X.Linux{...}` or `&X.Darwin{...}`. Consolidation makes the whole test surface
of a collector visible at a glance; consistency across collectors is the
priority.

The factory dispatches on `platform.Detect()` (wraps gopsutil's `host.Info`;
returns `"darwin"` / `"debian"` / `"rhel"` / `""` for generic linux).

**Key rules:**

- Every struct must compile on every platform. Use cross-platform APIs only:
  stdlib, gopsutil, `golang.org/x/sys/unix` (per-OS layouts but compiles
  everywhere), ghw, cloud SDKs. No raw `syscall.Utsname` etc.
- Missing OS-specific paths (e.g. `/proc/modules` on darwin) return empty
  gracefully, never error.
- Add a `Debian` variant (or `RHEL`, `SUSE`, etc.) **only** when that distro
  family genuinely diverges. Otherwise generic `Linux` covers all non-darwin.
- Dependency-inject file readers, command runners, and gopsutil calls via struct
  fields, lets tests exercise every branch without touching the real host.
- **NEVER leak third-party types through public `Fn` fields.** Per-OS struct
  `Fn` fields are forbidden, no public field on a `Linux` / `Darwin` / etc.
  variant may have a function type whose signature mentions gopsutil / ghw /
  procfs types.
- **Test seams swap at the upstream library boundary, not in the middle.** The
  upstream call lives in a private package var
  (`var hostInfoFn = host.InfoWithContext`); tests swap it via a `Set<X>Fn`
  setter declared in `export_test.go`. **Do not** add intermediate wrappers like
  `readBaseFn = readBase` that let tests bypass a bridge function, that forces a
  second test method (`TestReadBase`) to cover the bridge, which is exactly what
  we're consolidating away from. One seam, one `TestCollect`. Collect calls
  `readBase(ctx)` directly; tests swap `hostInfoFn` and the bridge mapping runs
  on every row. See `pkg/gohai/collectors/uptime/` and
  `pkg/gohai/collectors/load/` for canonical examples.
- File reads and command execution go through `FS avfs.VFS` and
  `Exec executor.Executor` struct fields on the per-OS variant. These are _our_
  abstractions (not third-party types), so they're fine to expose publicly. See
  the "VFS + Executor Abstractions" section for the pattern.

The Collector interface and `Info` struct shape are the contract, whatever
backing strategy a collector uses, its output must match the typed struct and
consumer expectations.

### Field naming

**Three-tier naming ladder.** Every JSON field name comes from one of three
tiers, applied in strict order of precedence:

1. **[OCSF][]** (Open Cybersecurity Schema Framework). Primary authority. When
   OCSF has a field for the concept, use its name. Browse
   [schema.ocsf.io][ocsf-schema] objects: `device`, `device_hw_info`, `os`,
   `network_interface`, `package`, `process`, `cloud`. (~108 gohai fields are
   tier 1.)
2. **[OpenTelemetry Resource Semantic Conventions][otel-semconv]**. When OCSF is
   silent. Covers CPU microarchitecture (`host.cpu.*`), memory states
   (`system.memory.*`), filesystem attributes (`system.filesystem.*`), hardware
   detail (`hardware.*`), and process attributes (`process.*`). (~74 gohai
   fields are tier 2.)
3. **gohai convention**. For the long tail where no standard has an opinion
   (~768 fields):
   - Start from the backing library's field name (gopsutil/ghw), converted to
     `snake_case`.
   - `/proc` and `/sys` mirrors use the kernel's name in `snake_case`.
   - Unit suffixes (`_bytes`, `_seconds`, `_percent`, `_mhz`) when the unit is
     ambiguous.
   - No abbreviations except universals: `ip`, `mac`, `pid`, `uid`, `gid`,
     `mtu`, `fqdn`, `uuid`, `cidr`, `arn`, `id`.

The complete per-field mapping with verifiable citations lives in
[`schemas/field-mapping.md`](../schemas/field-mapping.md). Fields where OCSF is
silent are tracked in [`schemas/ocsf-gaps.md`](../schemas/ocsf-gaps.md) as
upstream contribution candidates.

**Not a naming reference:** Ohai (methodology only, not naming), node_exporter
(methodology only), OCP (hardware design spec), CIS/SCAP/XCCDF (compliance
policies). ECS, osquery, and Facter are useful cross-references but not naming
authorities.

**Both Go field names AND JSON tags derive from the chosen schema (OCSF primary,
OpenTelemetry when OCSF is silent).**

**Redundant-prefix rule:** JSON keys use the schema's leaf name with any
parent-object prefix stripped when that prefix duplicates our collector name.
Our output nests by collector (`{"cpu": {...}, "memory": {...}}`), so restating
the prefix inside the nested object is noise. Examples:

| OCSF path            | Our collector | Redundant prefix?                                            | Our JSON key |
| -------------------- | ------------- | ------------------------------------------------------------ | ------------ |
| `device.cpu_count`   | `cpu`         | `cpu_` → strip                                               | `count`      |
| `device.cpu_cores`   | `cpu`         | `cpu_` → strip                                               | `cores`      |
| `device.memory_size` | `memory`      | `memory_` → strip                                            | `size`       |
| `os.kernel_release`  | `kernel`      | `kernel_` → strip                                            | `release`    |
| `device.hostname`    | `hostname`    | `hostname` == collector → `name`                             | `name`       |
| `process.cmd_line`   | `process`     | no match → keep                                              | `cmd_line`   |
| `host.cpu.vendor.id` | `cpu`         | OTel leaf is `id`, parent `vendor` isn't our collector, keep | `vendor_id`  |

The full schema path (OCSF first, OTel if OCSF silent) is cited in every
collector doc's **Schema mapping** column so consumers bridging to OCSF can
write a mechanical transform.

The Go field is the PascalCase rendering of the final JSON key
(`` Count int `json:"count"`  ``, `` Name string `json:"name"`  ``). When Go
idiom on initialisms conflicts (OCSF `cpu_id` → Go `CPUID`, not `CpuId`), Go
convention wins the field name but the JSON tag still follows the rule above.
Don't invent internal names that have no schema-mapping claim.

**Do not mirror Ohai's JSON shape.** Ohai is for **data-source** reference (what
file/command to read, which distro edge cases, which fallback), not field names
or struct layout.

### MANDATORY: Cross-reference Ohai's data sources before implementing

Before writing code for a new collector (or modifying an existing one), **read
Ohai's corresponding plugin and spec**, but the goal is to match their
**collection approach** (what file/command/library they read, what edge cases
they handle, how they detect per-distro differences), **not** their JSON output
shape. Ruby Mash ↔ Go struct translation isn't worthwhile to pin byte-for-byte;
Go-native JSON shape is fine.

What matters: Ohai has years of accumulated bug fixes and distro-specific
quirks. Use them. If they read `/proc/X` plus fall back to `cmd Y` on SUSE, we
should too. If they have special handling for Amazon Linux vs RHEL, we need to
think about it too.

Fetch both files with `gh api`:

```bash
gh api repos/chef/ohai/contents/lib/ohai/plugins/<name>.rb --jq .content | base64 -d
gh api repos/chef/ohai/contents/spec/unit/plugins/<name>_spec.rb --jq .content | base64 -d
```

Filenames occasionally differ, many plugins live under OS subdirs (`linux/`,
`darwin/`, `windows/`). Browse `repos/chef/ohai/contents/lib/ohai/plugins` if
the direct path 404s.

Every collector doc **must** carry a standard **"Data Sources"** section.
Complex collectors that emit multiple derived facts answering different
questions also get a **"Signals"** section.

**Data Sources** (required on every doc):

The Data Sources section is a self-contained spec of HOW the collector collects
data, written in **our voice**. Numbered step-by-step, per-OS sections when
behavior differs. Describe the actual sequence of reads, fallbacks, distro
branches, and error handling. Do NOT frame it as a parity comparison with Ohai.
Example shape:

```md
## Data Sources

On Linux the collector cascades through multiple signals:

1. **Fast path:** if `systemd-detect-virt` is on PATH, call it.
2. **Container-runtime presence:** `which(docker)` / `which(podman)`.
3. **Xen:** `/proc/xen` and `/proc/xen/capabilities`.
4. ...
```

Ohai is mentioned inline only when a specific methodology choice needs
attribution ("we mirror Ohai's legacy `/etc/*-release` fallback chain"). The
section is a spec of OUR behavior, not a diff against Ohai.

**`Known gaps vs. Ohai` is NOT a permanent section.** Methodology gaps live on
GitHub as issues labeled `methodology-gap` and `collector:<name>`. Each issue
carries a "Doc after this fix lands" block with the exact prose the fix PR
pastes into the Data Sources section. When all open methodology issues for a
collector close, the doc has zero Ohai residue. See the "Methodology Work"
section below for the full workflow.

**Signals** (required on complex collectors like `fips` where multiple fields
answer different consumer questions; omit for simple collectors like `shells` or
`root_group` where the fact is a single value).

Use a prose list immediately after the Description section:

```md
The collector reports N related signals:

- `<field>`. What it means, what source it comes from, what question it answers
  for the consumer.
- `<field>`. Same, including when this signal and the one above can disagree
  and what that disagreement tells you.
```

Signals are about **meaning**, not structure. Use them whenever a consumer can
reasonably ask "which of these fields should I look at for X?", the Signals
section answers that before they have to read the field table.

This keeps docs consistent and makes it obvious at a glance whether we're
leveraging Ohai's hard-won knowledge or flying solo. If Ohai has coverage we
lack, either add it in the same PR or open a tracked issue, don't silently drop
it.

[Reference PR adding this rule: chef/ohai#1754]

## Methodology Work

Methodology gaps between gohai and Ohai live on GitHub as issues labeled
`methodology-gap` and `collector:<name>`. See
`gh issue list --label methodology-gap`. Each issue carries:

- Full Ohai methodology breakdown, source-cited with file + line ranges.
- Our current implementation and what it misses.
- Risk / severity / which hosts fail.
- Proposed fix, concrete code plan.
- Acceptance criteria.
- **"Doc after this fix lands"**. The exact prose (Description + Collected
  Fields table + Data Sources) the fix PR pastes into the collector's
  `docs/collectors/<name>.md`.

**Workflow when working a methodology issue:**

1. `gh issue view <N>`. Read end to end, especially the "Doc after this fix
   lands" block.
2. Implement the code change per "Proposed fix", use the VFS / Executor
   abstractions if Phase 1 has landed, otherwise the `export_test.go` + private
   var + `Set<X>Fn` pattern.
3. Paste the issue's "Doc after this fix lands" block into the collector doc,
   replacing Description / Collected Fields / Data Sources as specified.
4. PR description must include `Closes #N`.
5. CI green, 100% coverage, `just go-vet` clean.

When every open methodology issue closes, every collector doc reads as a
self-contained spec and the SDK has zero unresolved methodology divergences from
Ohai.

[ghw]: https://github.com/jaypipes/ghw
[go-sysinfo]: https://github.com/elastic/go-sysinfo
[gopsutil]: https://github.com/shirou/gopsutil
[node_exporter]: https://github.com/prometheus/node_exporter
[ohai-plugins]: https://github.com/chef/ohai/tree/main/lib/ohai/plugins
[otel-semconv]: https://opentelemetry.io/docs/specs/semconv/resource/
[procfs]: https://github.com/prometheus/procfs
