#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail_on_hit="${FIND_OBVIOUS_COMMENTS_FAIL:-0}"

mapfile -t hits < <(
	rg -n -i \
		'^\s*//\s*(this function|this method|this struct|this (file|package)|returns the|return the|gets the|sets the|handles the|creates the|initializes the|loops through|check if the|iterate over|step [0-9]|first,|then,|next,|finally,)' \
		internal cmd pkg tests \
		--glob '*.go' \
		--glob '!**/pb/**' \
		--glob '!**/sqlc/**' \
		--glob '!**/*.pb.go' \
		--glob '!**/*_test.go' 2>/dev/null || true
)

count="${#hits[@]}"
if (( count > 0 )); then
	echo "find_obvious_comments: ${count} hit(s)"
	printf '  %s\n' "${hits[@]}"
	if [[ "$fail_on_hit" == "1" ]]; then
		exit 1
	fi
	echo "find_obvious_comments: warning only (set FIND_OBVIOUS_COMMENTS_FAIL=1 to fail)"
	exit 0
fi

echo "find_obvious_comments: OK (0 hits)"
