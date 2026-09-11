#!/usr/bin/env bash
set -euo pipefail

# Role: Report catalog keys vs OpenAPI report GET operations parity.
# Execution context: CI via live_routes.sh when web/src exists.
# Verify: bash scripts/ci/admin/report_catalog_openapi_parity.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

CATALOG_GO="internal/reports/catalog.go"
OPENAPI_REPORTS="api/openapi/paths/ops_reports.yaml"
CAMPAIGN_OPENAPI="api/openapi/paths/campaigns.yaml"

for path in "$CATALOG_GO" "$OPENAPI_REPORTS" "$CAMPAIGN_OPENAPI"; do
  if [[ ! -f "$path" ]]; then
    echo "Error: required path missing: $path"
    exit 1
  fi
done

python3 - "$CATALOG_GO" "$OPENAPI_REPORTS" "$CAMPAIGN_OPENAPI" << 'PY'
import re
import sys
from pathlib import Path

catalog_path = Path(sys.argv[1])
openapi_reports = Path(sys.argv[2])
openapi_campaigns = Path(sys.argv[3])

catalog_keys = set(re.findall(r'\{Key:\s*"([^"]+)"', catalog_path.read_text(encoding="utf-8")))
if not catalog_keys:
    print(f"Error: no catalog keys in {catalog_path}", file=sys.stderr)
    sys.exit(1)

openapi_blob = openapi_reports.read_text(encoding="utf-8") + "\n" + openapi_campaigns.read_text(encoding="utf-8")

# Map OpenAPI path suffix -> catalog key (or alias target).
path_to_key: dict[str, str] = {}
for match in re.finditer(
    r"^\s+/api/v1/reports/([^:\s]+):\s*$",
    openapi_reports.read_text(encoding="utf-8"),
    re.MULTILINE,
):
    suffix = match.group(1).strip("/")
    if suffix.startswith("jobs"):
        continue
    if suffix == "catalog":
        continue
    if suffix == "notifications" or suffix.startswith("notifications/"):
        continue
    if suffix == "clicks":
        path_to_key[suffix] = "click-log"
        continue
    if suffix == "telegram/export":
        continue
    if suffix.startswith("telegram/"):
        path_to_key[suffix] = f"telegram/{suffix.split('/', 1)[1]}"
        continue
    if suffix.startswith("rtb/"):
        path_to_key[suffix] = suffix.replace("/", "-").replace("rtb-", "rtb-")
        # rtb/overview -> rtb-overview
        path_to_key[suffix] = "rtb-" + suffix.split("/", 1)[1].replace("/", "-")
        continue
    if suffix.startswith("ml/"):
        path_to_key[suffix] = suffix
        continue
    path_to_key[suffix] = suffix

# campaignsGetStats is catalog campaign-stats, not under /reports.
path_to_key["campaigns/{id}/stats"] = "campaign-stats"

openapi_keys = set(path_to_key.values())

# Legacy URL aliases documented in catalog/runner only.
openapi_keys.add("click-log")

# ghost-impression-funnel shares silent-reject handler; catalog uses canonical key.
if "silent-reject-impression-funnel" in catalog_keys:
    openapi_keys.add("silent-reject-impression-funnel")

missing_catalog = sorted(openapi_keys - catalog_keys)
extra_catalog = sorted(catalog_keys - openapi_keys)

# Export-only and metric-only keys may exist only in catalog when handler is registered elsewhere.
allowed_extra = {"fraud-evidence-pack-bulk"}

extra_catalog = [key for key in extra_catalog if key not in allowed_extra]

if missing_catalog:
    print("Error: OpenAPI report paths without catalog row:", file=sys.stderr)
    for key in missing_catalog:
        print(f"  - {key}", file=sys.stderr)
    sys.exit(1)

if extra_catalog:
    print("Error: catalog keys without OpenAPI GET report path:", file=sys.stderr)
    for key in extra_catalog:
        print(f"  - {key}", file=sys.stderr)
    sys.exit(1)

print(f"Report catalog OpenAPI parity: OK ({len(catalog_keys)} keys)")
PY
