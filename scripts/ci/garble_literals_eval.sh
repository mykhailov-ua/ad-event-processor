#!/usr/bin/env bash
# Compare tracker binary size with/without garble -literals (S3.4).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${1:-/tmp/garble_literals_eval}"
mkdir -p "$OUT"

echo "garble_literals_eval: building baseline (no -literals)..."
RELEASE_GARBLE=1 GARBLE_LITERALS=0 bash scripts/ci/release_garble.sh "$OUT/baseline" tracker

echo "garble_literals_eval: building with -literals..."
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
echo "Note: garble -literals is global; hot-path packages use //garble:ignore where needed."
echo "Run make test-alloc-gate on a literals build before enabling in release."
