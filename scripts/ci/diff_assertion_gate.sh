#!/usr/bin/env bash
# Fail PRs that delete test assertions (assert/require/t.Error/t.Fatal).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

BASE="${DIFF_ASSERTION_BASE:-}"
if [[ -z "$BASE" ]]; then
  if git rev-parse --verify origin/main >/dev/null 2>&1; then
    BASE="origin/main"
  else
    echo "diff-assertion-gate: SKIP (no origin/main; set DIFF_ASSERTION_BASE)"
    exit 0
  fi
fi

if ! git merge-base --is-ancestor "$BASE" HEAD 2>/dev/null; then
  echo "diff-assertion-gate: SKIP (cannot diff against $BASE)"
  exit 0
fi

pattern='^[[:space:]]*require\.|^[[:space:]]*assert\.|^[[:space:]]*t\.(Error|Fatal|Fail)\('

removed="$(
  git diff -U0 "${BASE}"...HEAD -- '*.go' '*_test.go' 2>/dev/null \
    | awk '/^--- / {next} /^+++ / {next} /^@@ / {next} /^-/ && !/^---/ {print}' \
    | rg "$pattern" || true
)"

if [[ -n "$removed" ]]; then
  echo "diff-assertion-gate: FAIL removed assertions in diff vs $BASE"
  echo "$removed"
  exit 1
fi

echo "diff-assertion-gate: OK"
