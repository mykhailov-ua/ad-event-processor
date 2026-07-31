#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

fail() {
	echo "docs layout: $1" >&2
	exit 1
}

for forbidden in docs/MULTI_REGION.md docs/COMPLIANCE_MATRIX.md; do
	if [[ -f "$forbidden" ]]; then
		fail "$forbidden must not exist (moved to .cursor/)"
	fi
done

for required in .cursor/MULTI_REGION.md .cursor/COMPLIANCE_MATRIX.md docs/MILESTONE.md; do
	if [[ ! -f "$required" ]]; then
		fail "missing $required"
	fi
done

if ! grep -q '.cursor/BACKLOG.md' docs/MILESTONE.md; then
	fail 'docs/MILESTONE.md must redirect to .cursor/BACKLOG.md'
fi

allowed_root=(
	ARCHITECTURE.md
	DEVELOPMENT.md
	MILESTONE.md
	SELF_HOSTED.md
	PROTECTION.md
	LICENSE_COMMERCE.md
	TELEMETRY_AND_TRUST.md
)

for path in docs/*.md; do
	[[ -e "$path" ]] || continue
	base="$(basename "$path")"
	ok=0
	for name in "${allowed_root[@]}"; do
		if [[ "$base" == "$name" ]]; then
			ok=1
			break
		fi
	done
	if [[ "$ok" -eq 0 ]]; then
		fail "unexpected docs/$base"
	fi
done

if ! [[ -d docs/openapi ]]; then
	fail 'missing docs/openapi/'
fi

if ! [[ -d docs/runbooks ]]; then
	fail 'missing docs/runbooks/'
fi

echo "docs layout: OK"
