#!/usr/bin/env bash

set -euo pipefail

# Role: Admin gate: CPA route audit stub.
# Execution context: CI via admin/web.sh or pr_fast.
# Invariants/contracts enforced: Missing web/ uses stub embed checks; live routes need OpenAPI backend.
# Verify: bash scripts/ci/admin/cpa_route_audit.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"
go test ./internal/controlplane/ -run 'TestCPA_' -count=1
echo "cpa_route_audit_gate: OK"
