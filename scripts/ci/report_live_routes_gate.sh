#!/usr/bin/env bash
# Fail when a live report catalog entry lacks an explicit route (would mount report_stub.js).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

REPORT_JS="$ROOT/web/src/models/report.ts"
if [ ! -f "$REPORT_JS" ]; then
  REPORT_JS="$ROOT/web/src/models/report.js"
fi
ROUTES_JS="$ROOT/web/src/lib/routes.ts"
if [ ! -f "$ROUTES_JS" ]; then
  ROUTES_JS="$ROOT/web/src/lib/routes.js"
fi

if [ ! -f "$REPORT_JS" ] || [ ! -f "$ROUTES_JS" ]; then
  echo "Error: report.ts/js or routes.ts/js missing"
  exit 1
fi

missing=0
while IFS= read -r key; do
  [ -z "$key" ] && continue
  if ! grep -q "/reports/${key}" "$ROUTES_JS"; then
    echo "Error: live report '${key}' has no explicit route in ${ROUTES_JS#$ROOT/}"
    missing=1
  fi
done < <(node -e "
const fs = require('fs');
const src = fs.readFileSync(process.argv[1], 'utf8');
const re = /\\{\\s*key:\\s*'([^']+)'[^}]*live:\\s*true/g;
let m;
const keys = new Set();
while ((m = re.exec(src)) !== null) keys.add(m[1]);
for (const k of keys) console.log(k);
" "$REPORT_JS")

if [ "$missing" -ne 0 ]; then
  exit 1
fi

echo "Report live routes gate: OK"
