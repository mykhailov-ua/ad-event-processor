#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${1:-/tmp/garble_literals_eval}"
mkdir -p "$OUT"

echo "garble_literals_eval: building baseline tracker (GARBLE_LITERALS=0)..."
RELEASE_GARBLE=1 GARBLE_LITERALS=0 bash scripts/ci/release_garble.sh "$OUT/baseline" tracker

echo "garble_literals_eval: building tracker with -literals (GARBLE_LITERALS=1 override)..."
RELEASE_GARBLE=1 GARBLE_LITERALS=1 bash scripts/ci/release_garble.sh "$OUT/literals" tracker

baseline_size=$(stat -c%s "$OUT/baseline/tracker")
literals_size=$(stat -c%s "$OUT/literals/tracker")
delta=$((literals_size - baseline_size))
pct=0
if [[ "$baseline_size" -gt 0 ]]; then
  pct=$((delta * 100 / baseline_size))
fi

echo ""
echo "garble_literals_eval summary"
echo "  baseline bytes:  $baseline_size"
echo "  literals bytes:  $literals_size"
echo "  delta:           $delta ($pct%)"
echo ""
echo "Note: release builds use per-binary policy (tracker=0, control/processor=1)."
echo "      internal/ingestion has //garble:ignore when tracker literals eval is forced."
echo "      p99 acceptance (+10%) requires load-test-bpf on literals tracker — not run here."
