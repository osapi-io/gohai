# Collector methodology

Which library each collector wraps, what a collector's own page has to contain,
and where methodology gaps are tracked.

**The rules are not here.** How a backing library is chosen, what an extension
may do and what it may read through, why collector code carries no build tag,
and how a field is named are stated once in the
[specifications repository](https://github.com/osapi-io/specs/blob/main/components/gohai/collectors.md).
Read that before adding or modifying a collector; this page tells you what the
existing ones already do.

Three rules are short enough to carry here, because somebody about to break one
is standing in this repository:

- Wrap a maintained library for what it covers and layer an extension on top.
  Never replace a library wholesale for one missing field.
- An extension reads files through `avfs.VFS` and runs commands through
  `executor.Executor`, never `os.ReadFile` or `exec.Command` in a `Collect`
  method.
- Collector code compiles on every platform. No `//go:build` tag anywhere.

See [CONTRIBUTING](../CONTRIBUTING.md) for setup and workflow, and
[adding-a-collector.md](adding-a-collector.md) for the nine steps.

## Which library each collector wraps

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

## What a collector's own page must contain

Two sections, and this is the convention for writing them rather than a fact
about any one collector.

**Data Sources** describes the cascade a collector actually walks, in order:

1. **Fast path:** if `systemd-detect-virt` is on PATH, call it.
2. **Container-runtime presence:** `which(docker)` / `which(podman)`.
3. **Xen:** `/proc/xen` and `/proc/xen/capabilities`.
4. ...

````

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

- `<field>` — what it means, what source it comes from, what question it answers
  for the consumer.
- `<field>` — same, including when this signal and the one above can disagree
  and what that disagreement tells you.
````

Signals are about **meaning**, not structure. Use them whenever a consumer can
reasonably ask "which of these fields should I look at for X?", the Signals
section answers that before they have to read the field table.

This keeps docs consistent and makes it obvious at a glance whether we're using
Ohai's hard-won knowledge or flying solo. If Ohai has coverage we lack, either
add it in the same PR or open a tracked issue, don't silently drop it.

[Reference PR adding this rule: chef/ohai#1754]

## Methodology work

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
