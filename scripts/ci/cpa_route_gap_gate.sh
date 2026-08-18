#!/usr/bin/env bash
# Route/report gap audit — live reports must have API + UI routes.
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
go test ./internal/controlplane/ -run 'TestCPA_' -count=1
echo "cpa_route_gap_gate: OK"
