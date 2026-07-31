#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

default_skip_prefixes=(
	internal/ingestion/sqlc
	internal/auth/db
	internal/payment/db
	internal/billing/db
	internal/notifier/db
)

skip_prefixes=()

load_skip_prefixes_from_ci_gates() {
	local gates="$ROOT/.cursor/CI_GATES.md"
	[[ -f "$gates" ]] || return 0

	local in_block=0
	local line path
	while IFS= read -r line || [[ -n "$line" ]]; do
		case "$line" in
		*'<!-- check-comments-skip-prefixes:start -->'*)
			in_block=1
			skip_prefixes=()
			continue
			;;
		*'<!-- check-comments-skip-prefixes:end -->'*)
			in_block=0
			continue
			;;
		esac
		if (( in_block )); then
			path="${line#- }"
			path="${path%/}"
			[[ -n "$path" ]] || continue
			skip_prefixes+=("$path")
		fi
	done <"$gates"
}

if ((${#skip_prefixes[@]} == 0)); then
	load_skip_prefixes_from_ci_gates
fi
if ((${#skip_prefixes[@]} == 0)); then
	skip_prefixes=("${default_skip_prefixes[@]}")
fi

export CHECK_COMMENTS_SKIP_PREFIXES
CHECK_COMMENTS_SKIP_PREFIXES="$(
	IFS=:
	echo "${skip_prefixes[*]}"
)"

exec go run ./cmd/check-comments
