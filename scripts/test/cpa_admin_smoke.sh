#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
bash scripts/ci/cpa_route_gap_gate.sh
bash scripts/ci/check_ui_slop.sh
bash scripts/ci/report_live_routes_gate.sh
bash scripts/ci/admin_web.sh
bash scripts/test/billing_export_smoke.sh
echo "cpa_admin_smoke: OK"
