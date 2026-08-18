#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ALLOWLIST="$ROOT/internal/database/ch_direct_allowlist.txt"
if [[ ! -f "$ALLOWLIST" ]]; then
  echo "missing allowlist: $ALLOWLIST"
  exit 1
fi

declare -A allowed=()
while IFS= read -r line; do
  [[ -z "$line" || "$line" =~ ^# ]] && continue
  allowed["$line"]=1
done < "$ALLOWLIST"

pattern='\b(conn|chWrite|writeConn|j\.conn)\.(Query|QueryRow|Exec)\('
violations=0

while IFS= read -r file; do
  rel="${file#./}"
  if [[ -n "${allowed[$rel]:-}" ]]; then
    continue
  fi
  if rg -n "$pattern" "$file" > /dev/null; then
    echo "raw ClickHouse driver call outside CHQuery: $rel"
    rg -n "$pattern" "$file" || true
    violations=$((violations + 1))
  fi
done < <(rg -l 'github.com/ClickHouse/clickhouse-go' internal/ --glob '*.go' --glob '!*_test.go')

if [[ "$violations" -gt 0 ]]; then
  echo "check_ch_direct: $violations violation(s); add allowlist entry only for write/DDL paths"
  exit 1
fi

echo "check_ch_direct: ok"
