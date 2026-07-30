#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

BASE="${1:-origin/main}"
if ! git rev-parse --verify "$BASE" >/dev/null 2>&1; then
	BASE="HEAD~1"
fi

mapfile -t hits < <(
	git diff --name-only "$BASE"...HEAD \
		| rg -i 'chaos' || true
)

if ((${#hits[@]})); then
	echo "check_no_chaos_refs: forbidden 'chaos' in new/changed paths (use fault/resilience naming):"
	printf '  %s\n' "${hits[@]}"
	exit 1
fi

mapfile -t content_hits < <(
	git diff "$BASE"...HEAD -- '*.go' '*.sh' '*.lua' '*.yaml' '*.yml' Makefile \
		| rg -i '\bchaos\b' \
		| rg -v 'bannedChaosWord|check_no_chaos_refs|no_chaos_refs|word .chaos. in comment' || true
)

if ((${#content_hits[@]})); then
	echo "check_no_chaos_refs: forbidden word 'chaos' in diff (use fault/resilience):"
	printf '  %s\n' "${content_hits[@]}"
	exit 1
fi

echo "check_no_chaos_refs: ok"
