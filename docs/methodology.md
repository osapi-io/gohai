# Collector methodology

This page tells which library each collector wraps, what the page of a collector
must contain, and where we track methodology gaps.

**The rules are not here.** The
[specifications repository](https://github.com/osapi-io/specs/blob/main/components/gohai/collectors.md)
gives these rules one time: how to select a backing library, what an extension
can do and what it can read through, why collector code has no build tag, and
how to name a field. Read it before you add or change a collector. This page
tells you what the current collectors do.

Three rules are short, so this page repeats them. A person who is about to break
one of them works in this repository:

- Wrap a maintained library for what it covers, and add an extension on top.
  Never replace all of a library because one field is missing.
- An extension reads files through `avfs.VFS` and runs commands through
  `executor.Executor`. Never use `os.ReadFile` or `exec.Command` in a `Collect`
  method.
- Collector code compiles on all platforms. No file has a `//go:build` tag.

For the setup and the workflow, refer to [CONTRIBUTING](../CONTRIBUTING.md). For
the nine steps, refer to [adding-a-collector.md](adding-a-collector.md).

## Which library each collector wraps

The primary library for each collector. To change it, open a PR that updates
this table and gives the reason.

| Collector        | Primary               | Candidate migration / supplement         |
| ---------------- | --------------------- | ---------------------------------------- |
| cpu              | gopsutil              | ghw/cpu for NUMA/topology/arch-math      |
| memory           | gopsutil              | ghw/memory for hugepages/page-sizes      |
| filesystem       | gopsutil              | ghw/block for UUID/label/unmounted       |
| disk             | gopsutil              | ghw/block for device metadata            |
| network          | gopsutil              | ghw/net for driver/speed                 |
| hostname         | gopsutil + stdlib     | None                                     |
| platform         | gopsutil              | We examined go-sysinfo as an alternative |
| uptime           | gopsutil              | None                                     |
| kernel           | `x/sys/unix` + stdlib | None                                     |
| load             | gopsutil              | None                                     |
| process          | gopsutil              | None. ghw has no process data            |
| users (sessions) | gopsutil (utmp)       | Add loginctl through executor            |
| virtualization   | gopsutil              | go-sysinfo has some of the data          |
| fips             | stdlib                | No library covers it                     |
| machine_id       | gopsutil + stdlib     | stdlib fallback chain                    |
| shard            | stdlib + machine_id   | None                                     |
| init             | stdlib                | `/proc/1/comm`                           |
| os_release       | stdlib                | Our own parser                           |
| lsb              | stdlib                | Add `lsb_release` through executor       |
| shells           | stdlib                | None                                     |
| timezone         | stdlib                | None                                     |
| root_group       | stdlib (`os/user`)    | None                                     |
| package_mgr      | stdlib exec           | Uses executor                            |
| dmi              | **ghw**               | baseboard + BIOS + chassis + product     |
| gpu              | **ghw**               | None                                     |
| pci              | **ghw**               | None                                     |
| block_device     | stdlib + **avfs**     | Reads sysfs through the injected VFS     |

The PR of a new collector must give the reason for its library. A migration
(gopsutil → ghw and others) needs its own issue with the labels
`library-migration` + `collector:<name>`. The issue gives the current coverage,
the coverage of the candidate, and the migration plan.

## What a collector's own page must contain

This part gives the convention for two sections. It is not a fact about one
collector.

**Data Sources** gives the sequence of checks that a collector does, in order:

1. **Fast path:** if `systemd-detect-virt` is on PATH, call it.
2. **Container runtime:** `which(docker)` / `which(podman)`.
3. **Xen:** `/proc/xen` and `/proc/xen/capabilities`.
4. ...

````

Name Ohai in the text only when a specific method choice needs attribution
("we use the same legacy `/etc/*-release` fallback chain as Ohai"). The section
is a spec of OUR behavior, not a diff against Ohai.

**`Known gaps vs. Ohai` is NOT a permanent section.** Methodology gaps are
GitHub issues with the labels `methodology-gap` and `collector:<name>`. Each
issue has a "Doc after this fix lands" block. This block has the exact text
that the fix PR pastes into the Data Sources section. When all open methodology
issues of a collector close, the page has no Ohai comparisons that remain. For
the full workflow, refer to the "Methodology work" section below.

**Signals**. A complex collector such as `fips` must have this section, because
its fields answer different questions of consumers. A simple collector such as
`shells` or `root_group` has one value, and does not have this section.

Write a list in prose directly after the Description section:

```md
The collector reports N related signals:

- `<field>` — what it means, what source it comes from, what question it answers
  for the consumer.
- `<field>` — same, including when this signal and the one above can disagree
  and what that disagreement tells you.
````

Signals tell the **meaning** of fields, not the structure. Use them when a
consumer can ask "which of these fields do I look at for X?". The Signals
section answers that question before the consumer reads the field table.

This keeps the pages consistent. It also shows quickly if a collector uses the
knowledge of Ohai or not. If Ohai has coverage that we do not have, add it in
the same PR or open a tracked issue. Do not remove it without a record.

[Reference PR adding this rule: chef/ohai#1754]

## Methodology work

Methodology gaps between gohai and Ohai are GitHub issues with the labels
`methodology-gap` and `collector:<name>`. To list them, run
`gh issue list --label methodology-gap`. Each issue has:

- The full Ohai methodology, with the source file and line ranges.
- Our current implementation and what it does not do.
- The risk, the severity, and which hosts fail.
- The proposed fix, as a concrete code plan.
- The acceptance criteria.
- **"Doc after this fix lands"**. The exact text (Description + Collected Fields
  table + Data Sources) that the fix PR pastes into the
  `docs/collectors/<name>.md` page of the collector.

**Workflow for a methodology issue:**

1. Run `gh issue view <N>`. Read all of it, and specially the "Doc after this
   fix lands" block.
2. Make the code change as the "Proposed fix" tells. If Phase 1 is complete, use
   the VFS / Executor abstractions. If not, use the `export_test.go` + private
   var + `Set<X>Fn` pattern.
3. Paste the "Doc after this fix lands" block of the issue into the collector
   page. Replace Description / Collected Fields / Data Sources as the block
   tells.
4. Put `Closes #N` in the PR description.
5. Make sure that CI passes, the coverage is 100%, and `just go-vet` is clean.

When all open methodology issues close, each collector page is a spec that is
complete in itself. Then the SDK has no open methodology differences from Ohai.
