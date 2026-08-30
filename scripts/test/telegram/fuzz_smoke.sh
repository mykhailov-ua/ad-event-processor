#!/usr/bin/env bash
# Role: Short fuzz runs for Telegram click query and initData parsers.
# Execution context: Repo root go test fuzz on ingest and controlplane packages.
# Env knobs: TELEGRAM_FUZZ_TIME (5s per target).
# Verify: bash scripts/test/telegram/fuzz_smoke.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
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

run_fuzz ./internal/ingest/ ParseTgClickQuery DecodeTgBid
run_fuzz ./internal/controlplane/ ParseInitData

echo "telegram_fuzz_smoke: PASS"
