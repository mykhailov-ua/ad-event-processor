#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

default_skip_prefixes=(
	internal/ingestion/sqlc
	internal/identity/db
	internal/payment/db
	internal/billing/db
	internal/notifier/db
)

skip_prefixes=()

load_skip_prefixes_from_file() {
	local list="$ROOT/scripts/ci/comment_linter_skip_prefixes.txt"
	[[ -f "$list" ]] || return 0

	local line path
	while IFS= read -r line || [[ -n "$line" ]]; do
		path="${line%%#*}"
		path="${path#"${path%%[![:space:]]*}"}"
		path="${path%"${path##*[![:space:]]}"}"
		path="${path%/}"
		[[ -n "$path" ]] || continue
		skip_prefixes+=("$path")
	done <"$list"
}

if ((${#skip_prefixes[@]} == 0)); then
	load_skip_prefixes_from_file
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
