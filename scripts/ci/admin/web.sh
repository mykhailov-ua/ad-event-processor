#!/usr/bin/env bash
set -euo pipefail

# Role: Admin web gate while web/ absent: static stub embed and boot inject tests.
# Execution context: CI via pr_fast; does not run npm typecheck until web/ returns.
# Invariants/contracts enforced: go test TestAdminStaticRoutes and TestInjectAdminBoot must pass.
# Verify: bash scripts/ci/admin/web.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "admin: static stub embed + boot inject"
go test ./internal/controlplane/ -run 'TestAdminStaticRoutes|TestInjectAdminBoot' -count=1

echo "Admin web checks PASSED (UI rebuild; web/ absent)."
