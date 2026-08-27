#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "report_live_routes_gate: skipped (web/ removed; live routes return with admin UI rebuild)"
