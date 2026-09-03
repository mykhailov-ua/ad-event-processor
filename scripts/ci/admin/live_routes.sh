#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: report catalog keys must have SPA route + Go HTTP handler (or export-only contract).
# Execution context: CI via admin/web.sh when web/src exists.
# Invariants/contracts enforced: ReportCatalogEntries keys wire to /reports/:key runner and /api/v1/reports/*.
# Verify: bash scripts/ci/admin/live_routes.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ ! -d web/src ]]; then
  echo "report_live_routes_gate: skipped (web/src absent)"
  exit 0
fi

CATALOG_GO="internal/reports/catalog.go"
APP_ROUTES="web/src/app_routes.tsx"
REPORT_PATHS_TS="web/src/lib/report_paths.ts"
REPORTS_TREE="internal/reports"

for path in "$CATALOG_GO" "$APP_ROUTES" "$REPORT_PATHS_TS"; do
  if [[ ! -f "$path" ]]; then
    echo "Error: required path missing: $path"
    exit 1
  fi
done

if ! rg -q 'path="reports/:key"' "$APP_ROUTES"; then
  echo "Error: $APP_ROUTES missing Route path=\"reports/:key\" for ReportRunnerPage"
  exit 1
fi

if ! rg -q 'path="reports/click-log"' "$APP_ROUTES"; then
  echo "Error: $APP_ROUTES missing dedicated click-log report route"
  exit 1
fi

if ! rg -q 'path="reports/jobs"' "$APP_ROUTES"; then
  echo "Error: $APP_ROUTES missing reports/jobs route"
  exit 1
fi

if ! command -v python3 > /dev/null 2>&1; then
  echo "Error: python3 required for report catalog parity check"
  exit 1
fi

python3 - "$CATALOG_GO" "$REPORT_PATHS_TS" "$REPORTS_TREE" << 'PY'
import re
import sys
from pathlib import Path

catalog_path = Path(sys.argv[1])
report_paths_ts = Path(sys.argv[2])
reports_tree = Path(sys.argv[3])

catalog_src = catalog_path.read_text(encoding="utf-8")
keys = re.findall(r'Key:\s*"([^"]+)"', catalog_src)
if not keys:
    print(f"Error: no ReportCatalogEntries keys in {catalog_path}", file=sys.stderr)
    sys.exit(1)

rp_src = report_paths_ts.read_text(encoding="utf-8")
override_block = re.search(
    r"REPORT_KEY_PATH_OVERRIDES[^=]*=\s*\{([^}]*)\}",
    rp_src,
    re.DOTALL,
)
overrides: dict[str, str] = {}
if override_block:
    for m in re.finditer(r"'([^']+)':\s*'([^']+)'", override_block.group(1)):
        overrides[m.group(1)] = m.group(2)

export_only_keys: set[str] = set()
m = re.search(r"EXPORT_ONLY_REPORT_KEYS\s*=\s*new Set\(\[([^\]]*)\]", rp_src, re.DOTALL)
if m:
    export_only_keys = set(re.findall(r"'([^']+)'", m.group(1)))

handler_sources: list[str] = []
for path in reports_tree.rglob("*.go"):
    if path.name.endswith("_test.go"):
        continue
    handler_sources.append(path.read_text(encoding="utf-8", errors="replace"))
handler_blob = "\n".join(handler_sources)

missing_handler: list[str] = []
for key in keys:
    if key in export_only_keys:
        if f'"{key}"' not in handler_blob and f"'{key}'" not in handler_blob:
            # export-only keys must appear in export/register or job export switch
            if key not in handler_blob:
                missing_handler.append(f"{key} (export-only, no handler reference)")
        continue

    api_path = overrides.get(key, f"/api/v1/reports/{key}")
    quoted_key = f'"{key}"'
    if api_path in handler_blob or quoted_key in handler_blob or f"wrapReport({quoted_key}" in handler_blob:
        continue
    if f"WrapReport({quoted_key}" in handler_blob:
        continue
    missing_handler.append(f"{key} (expected {api_path})")

if missing_handler:
    print("Error: catalog keys without Go report handler:", file=sys.stderr)
    for item in missing_handler:
        print(f"  - {item}", file=sys.stderr)
    sys.exit(1)

print(f"Report live routes gate: OK ({len(keys)} catalog keys, SPA reports/:key)")
PY
