#!/usr/bin/env bash
# Fail when a live report catalog entry lacks an explicit route (would mount report_stub.js).
# Requires python3 or node on PATH for live-key extraction (admin_web.sh already needs node for build).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

REPORT_JS="$ROOT/web/src/models/report.ts"
if [ ! -f "$REPORT_JS" ]; then
  REPORT_JS="$ROOT/web/src/models/report.js"
fi
APP_ROUTES_JS="$ROOT/web/src/app_routes.tsx"
if [ ! -f "$APP_ROUTES_JS" ]; then
  APP_ROUTES_JS="$ROOT/web/src/app_routes.js"
fi

if [ ! -f "$REPORT_JS" ] || [ ! -f "$APP_ROUTES_JS" ]; then
  echo "Error: report.ts/js or app_routes missing"
  exit 1
fi

extract_live_report_keys() {
  local report_file="$1"
  if command -v python3 > /dev/null 2>&1; then
    python3 - "$report_file" << 'PY'
import re, sys
src = open(sys.argv[1], encoding="utf-8").read()
for m in re.finditer(r"\{\s*key:\s*'([^']+)'[^}]*live:\s*true", src):
    print(m.group(1))
PY
  elif command -v node > /dev/null 2>&1; then
    node -e "
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const re = /\\{\\s*key:\\s*'([^']+)'[^}]*live:\\s*true/g;
let m;
const keys = new Set();
while ((m = re.exec(src)) !== null) keys.add(m[1]);
for (const k of keys) console.log(k);
" "$report_file"
  else
    echo "Error: python3 or node required for live report key extraction"
    exit 1
  fi
}

live_markers=$(grep -cE 'live:[[:space:]]*true' "$REPORT_JS" || true)
keys_tmp=$(mktemp)
extract_live_report_keys "$REPORT_JS" > "$keys_tmp"
key_count=$(wc -l < "$keys_tmp" | tr -d ' ')

if [ "$live_markers" -gt 0 ] && [ "$key_count" -eq 0 ]; then
  echo "Error: found live:true in ${REPORT_JS#$ROOT/} but extracted zero keys (parser/extractor failure)"
  rm -f "$keys_tmp"
  exit 1
fi

missing=0
while IFS= read -r key; do
  [ -z "$key" ] && continue
  if grep -q "/reports/${key}" "$APP_ROUTES_JS"; then
    :
  else
    echo "Error: live report '${key}' has no explicit route in app_routes"
    missing=1
  fi
done < "$keys_tmp"
rm -f "$keys_tmp"

if [ "$missing" -ne 0 ]; then
  exit 1
fi

echo "Report live routes gate: OK"
