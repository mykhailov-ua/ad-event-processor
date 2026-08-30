#!/usr/bin/env bash

set -euo pipefail

# Role: License gate: License route audit.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/route_audit.sh
cd "$(dirname "$0")/../../.."
go test ./internal/controlplane/ -run 'TestLicense_' -count=1
echo "license_route_audit_gate: OK"
