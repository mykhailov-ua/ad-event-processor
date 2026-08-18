#!/usr/bin/env bash
# Block merge when heap escapes grow in hot-path source files (ingestion/rtb handlers).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

BASELINE_FILE="${ESCAPE_HEAP_BASELINE:-$SCRIPTS/ci/baselines/escape_hot_path_heap_lines.count}"
REPORT="${ESCAPE_HEAP_REPORT:-escape_heap_gate.txt}"

mapfile -t HOT_FILES < <(
  find internal/ingestion internal/rtb \
    -name '*.go' ! -name '*_test.go' \
    \( \
      -path 'internal/ingestion/handler.go' -o \
      -path 'internal/ingestion/handler_http*.go' -o \
      -path 'internal/ingestion/filter*.go' -o \
      -path 'internal/ingestion/filters.go' -o \
      -path 'internal/rtb/*.go' \
    \) \
    ! -name '*_corpus.go' \
    ! -name '*_fault_corpus.go' \
    2> /dev/null | sort -u
)

if ((${#HOT_FILES[@]} == 0)); then
  echo "escape-heap-gate: ERROR no hot-path files resolved"
  exit 1
fi

mapfile -t HOT_PKGS < <(printf '%s\n' "${HOT_FILES[@]}" | xargs -I{} dirname {} | sort -u | sed 's|^|./|' | sed 's|$|/...|')

if [[ ! -f "$BASELINE_FILE" ]]; then
  echo "escape-heap-gate: ERROR missing baseline $BASELINE_FILE"
  exit 1
fi

BASELINE="$(tr -d '[:space:]' < "$BASELINE_FILE")"

go build -gcflags="-m" "${HOT_PKGS[@]}" 2>&1 | tee "$REPORT"

COUNT=0
for f in "${HOT_FILES[@]}"; do
  c="$(grep -F "$f" "$REPORT" | grep -c 'escapes to heap' || true)"
  COUNT=$((COUNT + c))
done

echo "escape-heap-gate: hot_path_heap_escape_lines=$COUNT baseline=$BASELINE (${#HOT_FILES[@]} files)"

if [[ "$COUNT" -gt "$BASELINE" ]]; then
  echo "escape-heap-gate: FAIL new heap escapes in hot-path files"
  for f in "${HOT_FILES[@]}"; do
    grep -F "$f" "$REPORT" | grep 'escapes to heap' || true
  done
  exit 1
fi

BASE="${DIFF_ASSERTION_BASE:-}"
if [[ -z "$BASE" ]] && git rev-parse --verify origin/main >/dev/null 2>&1; then
  BASE="origin/main"
fi

if [[ -n "$BASE" ]] && git merge-base --is-ancestor "$BASE" HEAD 2>/dev/null; then
  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    if grep -F "$file" "$REPORT" | grep -q 'escapes to heap'; then
      echo "escape-heap-gate: FAIL heap escape introduced in changed hot-path file $file"
      grep -F "$file" "$REPORT" | grep 'escapes to heap' || true
      exit 1
    fi
  done < <(git diff --name-only "${BASE}"...HEAD -- "${HOT_FILES[@]}" 2>/dev/null || true)
fi

echo "escape-heap-gate: OK"
