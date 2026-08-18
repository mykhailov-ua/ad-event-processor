#!/usr/bin/env bash
# Cold path static bans: O(N) SQL in loops (diff-scoped), Redis KEYS/FLUSHALL.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0
ALLOWLIST="$SCRIPTS/ci/cold_path_n1_allowlist.txt"
N1_PATTERN='for _, \w+ := range \w+ \{[^{}]{0,300}(GetPool\(\)\.QueryRow|pool\.QueryRow|\.QueryRow\(|GetCustomerDTO|GetCampaign\(|ReadBudgetInvariant)'

fail() {
  echo "cold-path-static: FAIL $*"
  failed=1
}

is_allowlisted() {
  local path="$1"
  [[ -f "$ALLOWLIST" ]] || return 1
  while IFS= read -r line; do
    [[ -z "$line" || "$line" =~ ^# ]] && continue
    if [[ "$path" == *"$line"* ]]; then
      return 0
    fi
  done < "$ALLOWLIST"
  return 1
}

# Blocking Redis commands (full tree).
if rg -n '\.Keys\(|FLUSHALL|FLUSHDB|"KEYS |KEYS \*|\.FlushAll\(|\.FlushDB\(' \
  internal/ pkg/ --glob '*.go' --glob '!*_test.go' \
  --glob '!**/vendor/**' 2>/dev/null; then
  fail "blocking Redis KEYS/FLUSH* usage"
fi

BASE="${DIFF_ASSERTION_BASE:-}"
if [[ -z "$BASE" ]] && git rev-parse --verify origin/main >/dev/null 2>&1; then
  BASE="origin/main"
fi

if [[ -n "$BASE" ]] && git merge-base --is-ancestor "$BASE" HEAD 2>/dev/null; then
  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    is_allowlisted "$file" && continue
    if rg -U --multiline-dotall "$N1_PATTERN" "$file" 2>/dev/null; then
      fail "O(N) DB pattern introduced in $file"
    fi
  done < <(git diff --name-only "${BASE}"...HEAD -- internal/controlplane internal/payment internal/ledger -- '*.go' ':!*_test.go' 2>/dev/null || true)
else
  echo "cold-path-static: SKIP N+1 diff scan (no merge base)"
fi

if [[ "$failed" -ne 0 ]]; then
  echo "cold-path-static: see DEVELOPMENT.md#bpf-cold-gate"
  exit 1
fi

echo "cold-path-static: OK"
