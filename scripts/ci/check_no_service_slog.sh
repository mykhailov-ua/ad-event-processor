#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

mapfile -t hits < <(
	rg -n 'slog\.' internal/controlplane/service_*.go \
		--glob '!internal/controlplane/service.go' 2>/dev/null || true
)

if ((${#hits[@]})); then
	echo "check_no_service_slog: slog forbidden in service_*.go (except service.go lifecycle):"
	printf '  %s\n' "${hits[@]}"
	exit 1
fi

echo "check_no_service_slog: OK"
