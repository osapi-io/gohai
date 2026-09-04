# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read [CONTRIBUTING.md](CONTRIBUTING.md) first. It covers prerequisites, setup,
package structure, code standards, testing, and how to add a collector. All of
it applies to agents exactly as they apply to people. The collector methodology
is reference material in [docs/methodology.md](docs/methodology.md). This file
carries only what is specific to agents.

## Running tools

Invoke tools through `mise`, not from your path:

```bash
mise exec -- just test
```

`mise` is active in a person's shell and supplies the versions `.mise.toml`
declares. An agent's shell has no activation, so a bare `just` resolves to
whatever is installed globally, usually an older version.

The symptom is a check that fails here and passes in continuous integration, on
a file nobody edited. When that happens, establish which version ran before
treating the failure as real.

## Where the rules come from

[CONTRIBUTING.md](CONTRIBUTING.md#before-you-start) names the specification.
When a convention here and the specification disagree, the specification wins.
Say so rather than following the code.

## Before writing a collector

Read [docs/methodology.md](docs/methodology.md) before implementing or modifying
one. The decision order for backing libraries, the three-tier field naming
ladder, and the requirement to cross-reference Ohai's data sources are not
optional, and a collector written without them will be sent back.

## Commit trailer

When committing via Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Task tracking

**Do not use superpowers.** Spec Kit governs specification, planning, and
implementation, and the design record for a change lives in
[osapi-io/specs](https://github.com/osapi-io/specs). A second workflow over that
ground gives two answers to which artifact is authoritative, and the answer that
loses is the one nobody reads. Nothing superpowers produces is committed.
