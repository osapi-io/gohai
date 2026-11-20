# Collector methodology

This page tells what each collector reads, what the page of a collector must
contain, and where we track methodology gaps.

**The rules are not here.** The
[specifications repository](https://github.com/osapi-io/specs/blob/main/components/gohai/collectors.md)
gives these rules one time:

- how to select a backing library
- what an extension can do, and what it can read through
- why collector code has no build tag
- how to name a field

Read it before you add or change a collector. This page tells you what the
current collectors do.

Three rules are short, so this page repeats them. A person who is about to break
one of them works in this repository:

- A collector reads the host only through its sources: files through `avfs.VFS`,
  commands through `executor.Executor`, and the clock through `Now`. Never use
  `os.ReadFile`, `exec.Command` or a library that reads the host itself in a
  `Collect` method. Read the files and the commands that Ohai reads.
- Collector code compiles on all platforms. No file has a `//go:build` tag.

For the setup and the workflow, refer to [CONTRIBUTING](../CONTRIBUTING.md). For
the nine steps, refer to [adding-a-collector.md](adding-a-collector.md).

## What each collector reads

The primary source of each collector, through its sources. To change it, open a
PR that updates this table and gives the reason.

| Collector        | Primary source                                                    |
| ---------------- | ----------------------------------------------------------------- |
| cpu              | Linux: `/proc/cpuinfo`, sysfs, `lscpu`. macOS: `sysctl`, `ioreg`  |
| memory           | Linux: `/proc/meminfo`. macOS: `sysctl`, `vm_stat`                |
| filesystem       | Linux: `/proc/mounts`, `df`, `lsblk`. macOS: `mount`, `df`        |
| disk             | Linux: `/proc/diskstats`. macOS: `ioreg`                          |
| network          | Linux: sysfs, `/proc/net/dev`, `ip`. macOS: `ifconfig`, `netstat` |
| hostname         | `hostname`. Linux fallback: `/proc/sys/kernel/hostname`           |
| platform         | Linux: `/etc/os-release`, release files. macOS: `sw_vers`         |
| uptime           | Linux: `/proc/uptime`, `/proc/stat`. macOS: `sysctl`              |
| kernel           | `x/sys/unix` + stdlib                                             |
| load             | Linux: `/proc/loadavg`. macOS: `sysctl`                           |
| process          | Linux: `/proc/<pid>`, `/etc/passwd`. macOS: `ps`                  |
| users (sessions) | Linux: `loginctl`, `/var/run/utmp`. macOS: `/var/run/utmpx`       |
| fips             | stdlib                                                            |
| machine_id       | Linux: `/etc/machine-id`, dbus, DMI. macOS: `ioreg`               |
| shard            | stdlib + machine_id                                               |
| init             | `/proc/1/comm`                                                    |
| os_release       | Our own parser                                                    |
| lsb              | stdlib                                                            |
| shells           | stdlib                                                            |
| timezone         | `/etc/localtime` and the clock                                    |
| root_group       | stdlib (`os/user`)                                                |
| package_mgr      | executor                                                          |
| dmi              | **ghw**                                                           |
| gpu              | **ghw**                                                           |
| pci              | **ghw**                                                           |
| block_device     | sysfs, through the injected VFS                                   |

The PR of a new collector must give the reason for its source. A migration (for
example from ghw to sysfs) needs its own issue with the labels
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

Name Ohai in the text only when a specific method choice needs attribution ("we
use the same legacy `/etc/*-release` fallback chain as Ohai"). The section is a
spec of OUR behavior, not a diff against Ohai.

**`Known gaps vs. Ohai` is NOT a permanent section.** Methodology gaps are
GitHub issues with the labels `methodology-gap` and `collector:<name>`. Each
issue has a "Doc after this fix lands" block. This block has the exact text that
the fix PR pastes into the Data Sources section. When all open methodology
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
```

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
2. Make the code change as the "Proposed fix" tells. Use the VFS and Executor
   abstractions.
3. Paste the "Doc after this fix lands" block of the issue into the collector
   page. Replace Description / Collected Fields / Data Sources as the block
   tells.
4. Put `Closes #N` in the PR description.
5. Make sure that CI passes, the coverage is 100%, and `just go-vet` is clean.

When all open methodology issues close, each collector page is a spec that is
complete in itself. Then the SDK has no open methodology differences from Ohai.
