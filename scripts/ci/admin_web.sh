#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

echo "admin: static stub embed + boot inject"
go test ./internal/controlplane/ -run 'TestAdminStaticRoutes|TestInjectAdminBoot' -count=1

echo "Admin web checks PASSED (UI rebuild; web/ absent)."
