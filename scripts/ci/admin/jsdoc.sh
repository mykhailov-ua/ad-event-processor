#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: JSDoc on exported functions.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/jsdoc.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_SRC="$ROOT/web/src"
FAIL=0

while IFS= read -r -d '' srcfile; do
  case "$srcfile" in
    *.test.js | */icon.js) continue ;;
  esac

  case "$srcfile" in
    *.ts) continue ;;
  esac
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    lineno="${line%%:*}"
    [[ "$lineno" =~ ^[0-9]+$ ]] || continue
    prev=$((lineno - 1))
    if [[ "$prev" -lt 1 ]]; then
      echo "Missing JSDoc: $srcfile:$lineno (export)"
      FAIL=1
      continue
    fi
    if ! sed -n "${prev}p" "$srcfile" | grep -qE '^\s*(\*|/\*\*)'; then
      if ! sed -n "$((prev - 1))p" "$srcfile" | grep -qE '^\s*/\*\*'; then
        echo "Missing JSDoc: $srcfile:$lineno (export)"
        FAIL=1
      fi
    fi
  done < <(rg -n --no-heading '^export (async )?function ' "$srcfile" 2> /dev/null || true)
done < <(find "$WEB_SRC" \( -name '*.js' -o -name '*.ts' \) ! -name '*.test.js' ! -name '*.test.ts' -print0)

if [[ "$FAIL" -ne 0 ]]; then
  echo "JSDoc check failed. Add /** ... */ above each exported function."
  exit 1
fi

echo "JSDoc export check: OK"
