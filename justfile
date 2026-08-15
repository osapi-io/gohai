set allow-duplicate-variables

# Optional modules: mod? allows `just fetch` to work before .just/remote/ exists.
import? '.just/remote/go.just'
import? '.just/remote/md.just'
mod? just '.just/remote/just.mod.just'

# No documentation site, so md formats every markdown file in the repository.
md_site_dir := ""

# --- Fetch ---

# Fetch shared justfiles from osapi-justfiles
fetch:
    mkdir -p .just/remote
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/go/go.just -o .just/remote/go.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/md/md.just -o .just/remote/md.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/just.mod.just -o .just/remote/just.mod.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/just.just -o .just/remote/just.just

# --- Top-level orchestration ---

# Install all dependencies
deps:
    just go-deps
    just go-mod

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
    just just::fmt
