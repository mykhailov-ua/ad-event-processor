#!/usr/bin/env bash

set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"
go test ./internal/controlplane/ -run 'TestCPA_' -count=1
echo "cpa_route_gap_gate: OK"
