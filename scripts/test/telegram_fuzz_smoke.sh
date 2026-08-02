#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

FUZZTIME="${TELEGRAM_FUZZ_TIME:-5s}"

run_fuzz() {
	local pkg="$1"
	shift
	for name in "$@"; do
		echo "telegram_fuzz_smoke: $pkg Fuzz$name ($FUZZTIME)"
		go test -run='^$' -fuzz="Fuzz${name}" -fuzztime="$FUZZTIME" "$pkg"
	done
}

run_fuzz ./internal/ingestion/ ParseTgClickQuery
run_fuzz ./internal/controlplane/ ParseInitData

echo "telegram_fuzz_smoke: PASS"
