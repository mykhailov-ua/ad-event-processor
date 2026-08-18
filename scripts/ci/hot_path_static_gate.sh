#!/usr/bin/env bash
# Hot-path static bans: fmt.Sprintf, interface{} boxing, context.WithValue in tracker handlers.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

failed=0

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

ALLOW_WITH_VALUE=(
  internal/ingestion/filter_context.go
)

is_allowed_with_value() {
  local path="$1"
  local allow
  for allow in "${ALLOW_WITH_VALUE[@]}"; do
    if [[ "$path" == "$allow" ]]; then
      return 0
    fi
  done
  return 1
}

if ((${#HOT_FILES[@]} == 0)); then
  echo "hot-path-static: ERROR no hot-path files resolved"
  exit 1
fi

check_pattern() {
  local desc="$1"
  local pattern="$2"
  if rg -n "$pattern" "${HOT_FILES[@]}" 2> /dev/null; then
    echo "hot-path-static: FAIL $desc"
    failed=1
  fi
}

check_pattern 'fmt.Sprintf' 'fmt\.Sprintf'
check_pattern 'interface{} boxing' 'interface\{\}'

for path in "${HOT_FILES[@]}"; do
  if is_allowed_with_value "$path"; then
    continue
  fi
  if rg -n 'context\.WithValue' "$path" 2> /dev/null; then
    echo "hot-path-static: FAIL context.WithValue in $path"
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  echo "hot-path-static: see DEVELOPMENT.md#bpf-hot-static and ingestion.mdc hot-path rules"
  exit 1
fi

echo "hot-path-static: OK (${#HOT_FILES[@]} files)"
