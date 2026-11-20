set allow-duplicate-variables

# Optional modules: mod? allows `just fetch` to work before .just/remote/ exists.
import? '.just/remote/go.just'
import? '.just/remote/md.just'
import? '.just/remote/just.just'

# No documentation site, so md formats every markdown file in the repository.
md_site_dir := ""

# --- Fetch ---

# List the recipes
default:
    @just --list

# Fetch shared justfiles from osapi-justfiles
#
# It fetches the commit that main is at, not refs/heads/main: the raw URL of a
# branch is cached for five minutes, so a run just after a merge got the old
# modules and said nothing (osapi-io/osapi-justfiles#108). git ls-remote asks
# the server and has no cache, and the URL of a commit never changes. The
# commit goes to .just/remote/VERSION, so a build records what it used.
fetch:
    #!/usr/bin/env bash
    set -euo pipefail
    sha=$(git ls-remote https://github.com/osapi-io/osapi-justfiles refs/heads/main | cut -f1)
    base="https://raw.githubusercontent.com/osapi-io/osapi-justfiles/$sha"
    mkdir -p .just/remote
    for module in go md just; do
        curl -sSfL "$base/$module/$module.just" -o ".just/remote/$module.just"
    done
    echo "$sha" > .just/remote/VERSION
    echo "osapi-justfiles $sha"

# The checks of `just test`, which CI also reads (`just go-ci-checks`): a check
# added here runs on a laptop and in CI, with no change to the workflow.
go_slow_checks := "go-fmt-check go-vet go-generate-check"

# Fail when what go generate writes is not what is committed
#
# The mocks and schemas/gohai.schema.json are generated. The schema went stale
# on main without a failure (osapi-io/gohai#290): a collector's type changed,
# and nobody ran go generate. Each generated file is compared, before and after
# regenerating, so uncommitted work elsewhere does not trip it.
go-generate-check:
    #!/usr/bin/env bash
    set -euo pipefail
    generated() {
        find . \( -path ./.worktrees -o -path ./.git \) -prune -o \
            -type f \( -name '*.gen.go' -o -path ./schemas/gohai.schema.json \) -print | sort | xargs shasum
    }
    before=$(generated)
    go generate ./...
    if [ "$before" != "$(generated)" ]; then
        diff <(echo "$before") <(generated) || true
        echo "generated code is stale: run go generate ./... and commit the result" >&2
        exit 1
    fi

# --- Top-level orchestration ---

# Install all dependencies
deps:
    just go-deps

# Run all tests
test:
    just go-test

# Generate code
generate:
    just go-generate

# Format, lint, and generate before committing
ready:
    just generate
    just md-fmt
    just go-fmt
    just go-vet
    just just-fmt
