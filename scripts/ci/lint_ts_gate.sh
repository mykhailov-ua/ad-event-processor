#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

WEB_DIR="$ROOT/web"

if [[ ! -f "$WEB_DIR/package.json" ]]; then
  echo "lint_ts_gate: missing $WEB_DIR/package.json" >&2
  exit 1
fi

if [[ ! -d "$WEB_DIR/node_modules/typescript" ]]; then
  echo "lint_ts_gate: npm ci (web)..."
  (cd "$WEB_DIR" && npm ci)
fi

echo "lint_ts_gate: tsc --noEmit (app + tests)..."
(cd "$WEB_DIR" && npm run typecheck)

echo "lint_ts_gate: node --check on static JS..."
while IFS= read -r -d '' jsfile; do
  case "$jsfile" in
    */node_modules/* | */dist/* | */e2e/*) continue ;;
  esac
  node --check "$jsfile"
done < <(find "$ROOT" -name '*.js' -print0)

echo "lint_ts_gate: OK"
