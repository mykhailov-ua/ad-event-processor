#!/usr/bin/env bash
# Validates analyze_bpf.sh disk durability section against the GAP-DB-01/02 fixture.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
FIXTURE="$ROOT/scripts/load-test/fixtures/bpf_disk_gate"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

cp -a "$FIXTURE/bpf" "$OUT/bpf"
"$ROOT/scripts/load-test/analyze_bpf.sh" "$OUT"

grep -q "Group-commit coalescing: PASS" "$OUT/bpf-report.md"
grep -q "writev" "$OUT/bpf-report.md"
grep -q "sync reduction vs 1:1 baseline: 98.4%" "$OUT/bpf-report.md"

echo "analyze_bpf disk gate fixture: PASS"
