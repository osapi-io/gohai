#!/usr/bin/env bash
# Copyright (c) 2026 John Dewey
#
# Pulls the schema corpus used to inform gohai.schema.json field
# naming and shape decisions. Committed to schemas/corpus/ so the
# corpus is versioned alongside the schema it informed.
#
# Re-run to refresh any source to its current tip. Review the diff
# before committing.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORPUS="$ROOT/schemas/corpus"

fetch() {
    local name="$1" url="$2"
    shift 2
    local tmp
    tmp="$(mktemp -d)"
    echo "==> $name"
    git clone --depth 1 --quiet "$url" "$tmp"

    local sha
    sha="$(git -C "$tmp" rev-parse HEAD)"
    rm -rf "${CORPUS:?}/$name"
    mkdir -p "$CORPUS/$name"

    for path in "$@"; do
        if [[ -e "$tmp/$path" ]]; then
            mkdir -p "$CORPUS/$name/$(dirname "$path")"
            cp -R "$tmp/$path" "$CORPUS/$name/$path"
        else
            echo "WARN: $name missing $path"
        fi
    done

    # Preserve license + provenance.
    for f in LICENSE LICENSE.md LICENSE.txt COPYING NOTICE README.md; do
        [[ -f "$tmp/$f" ]] && cp "$tmp/$f" "$CORPUS/$name/$f"
    done

    printf 'source: %s\nfetched_sha: %s\nfetched_at: %s\n' \
        "$url" "$sha" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        > "$CORPUS/$name/PROVENANCE"

    rm -rf "$tmp"
}

mkdir -p "$CORPUS"

# Tier 1 — inventory-focused, direct scope match.

fetch ocsf https://github.com/ocsf/ocsf-schema.git \
    objects events dictionary.json enums

fetch otel https://github.com/open-telemetry/semantic-conventions.git \
    model

fetch osquery https://github.com/osquery/osquery.git \
    specs

fetch ecs https://github.com/elastic/ecs.git \
    schemas

fetch ohai https://github.com/chef/ohai.git \
    lib/ohai/plugins lib/ohai/mixin spec/unit/plugins

fetch facter https://github.com/puppetlabs/facter.git \
    lib/facter

fetch redfish https://github.com/DMTF/Redfish-Publications.git \
    json-schema

# Redfish ships every historical version of every schema (~6700 files,
# ~230MB). We only want the latest unversioned schema per resource
# type (the bare name without `.v1_x_y`). Strip the rest.
find "$CORPUS/redfish/json-schema" -name '*.v[0-9]*' -delete 2>/dev/null || true

# Kubernetes: we only care about the NodeStatus / NodeInfo Go types.
# Pulling the whole API repo is overkill; grab the one file.
mkdir -p "$CORPUS/k8s"
echo "==> k8s"
gh api repos/kubernetes/api/contents/core/v1/types.go --jq .content |
    base64 -d >"$CORPUS/k8s/core_v1_types.go"
gh api repos/kubernetes/api/commits/master --jq .sha >"$CORPUS/k8s/COMMIT_SHA"
cat >"$CORPUS/k8s/PROVENANCE" <<EOF
source: https://github.com/kubernetes/api
path: core/v1/types.go
fetched_sha: $(cat "$CORPUS/k8s/COMMIT_SHA")
fetched_at: $(date -u +%Y-%m-%dT%H:%M:%SZ)
note: Only the NodeStatus / NodeInfo / NodeSystemInfo types are relevant to gohai.
EOF

echo "==> done"
