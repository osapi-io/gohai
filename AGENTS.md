# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read @CONTRIBUTING.md first. It covers prerequisites, setup, package structure,
code standards, testing, and how to add a collector — all of which apply to
agents exactly as they apply to people. The collector methodology is reference
material in @docs/methodology.md. This file carries only what is specific to
agents.

## Running tools

Invoke tools through `mise`, not from your path:

```bash
mise exec -- just test
```

`mise` is active in a person's shell and supplies the versions `.mise.toml`
declares. An agent's shell has no activation, so a bare `just` resolves to
whatever is installed globally — usually an older version.

The symptom is a check that fails here and passes in continuous integration, on
a file nobody edited. When that happens, establish which version ran before
treating the failure as real.

## Where the rules come from

@CONTRIBUTING.md names the specification under "Before you start". When a
convention here and the specification disagree, the specification wins — say so
rather than following the code.

## Before writing a collector

Read `docs/methodology.md` before implementing or modifying one. The decision
order for backing libraries, the three-tier field naming ladder, and the
requirement to cross-reference Ohai's data sources are not optional, and a
collector written without them will be sent back.

## Commit trailer

When committing via Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Task tracking

Implementation planning and execution uses the superpowers plugin workflows
(`writing-plans` and `executing-plans`). Plans live in `docs/superpowers/`,
created when the first plan is written.
