# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read [CONTRIBUTING.md](CONTRIBUTING.md) first. It covers the prerequisites, the
setup, the package structure, the code standards, the tests, and how to add a
collector. All of it applies to agents in the same way as to people. The
collector methodology is reference material in
[docs/methodology.md](docs/methodology.md). This file has only what is specific
to agents.

## Running tools

Invoke tools through `mise`, not from your path:

```bash
mise exec -- just test
```

`mise` is active in a person's shell and supplies the versions that `.mise.toml`
declares. An agent's shell has no activation. Thus a bare `just` runs the
version that is installed globally, which is usually an older version.

The symptom is a check that fails here and passes in continuous integration, on
a file that nobody edited. When this occurs, find which version ran before you
accept the failure as real.

## Where the rules come from

[CONTRIBUTING.md](CONTRIBUTING.md#before-you-start) names the specification.
When a convention here and the specification disagree, the specification wins.
Say that they disagree. Do not follow the code.

## Before writing a collector

Read [docs/methodology.md](docs/methodology.md) before you write or change a
collector. These parts of it are mandatory:

- The decision order for backing libraries.
- The three-tier ladder for field names.
- The rule to compare with Ohai's data sources.

A reviewer returns a collector that does not follow them.

## Commit trailer

When you commit through Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

## Task tracking

**The design is in the design docs.** Write or change the page in
[osapi-io/specs](https://github.com/osapi-io/specs) before you build. If the
build shows that the page is wrong, correct the page. A design document in this
repository becomes incorrect when the code changes, and no check finds the
difference.
