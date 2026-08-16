#!/usr/bin/env bash
# Smoke: CPU isolation profile (cpuset pin, no cpu_quota).
#
# Usage:
#   bash scripts/test/cpu_isolation_smoke.sh
#
# Env:
#   CPU_ISOLATION_ENABLED=1   skip when 0 (unless CPU_ISOLATION_SMOKE_FORCE=1)
#   CPU_ISOLATION_SMOKE_FORCE=1
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

CPU_ISOLATION_ENABLED="${CPU_ISOLATION_ENABLED:-1}"

log() { printf 'cpu-isolation-smoke: %s\n' "$*"; }

if [[ "$CPU_ISOLATION_ENABLED" != "1" && "${CPU_ISOLATION_SMOKE_FORCE:-0}" != "1" ]]; then
	log "skip (CPU_ISOLATION_ENABLED!=1)"
	exit 0
fi

GO_BIN="$(cd "$ROOT" && go env GOROOT)/bin/go"
if [[ ! -x "$GO_BIN" ]]; then
	GO_BIN="$(command -v go)"
fi
log "unit: pkg/cpuset"
"$GO_BIN" test ./pkg/cpuset/ -count=1

if [[ "$CPU_ISOLATION_ENABLED" != "1" ]]; then
	log "skip live (CPU_ISOLATION_ENABLED!=1; unit ok)"
	printf 'fault_proof fault=cpu_isolation_smoke status=partial proof=unit_only harness=cpuset_parse isolation=disabled\n'
	exit 0
fi

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
	log "skip live (docker unavailable; unit ok)"
	printf 'fault_proof fault=cpu_isolation_smoke status=partial proof=unit_only harness=cpuset_parse docker=absent\n'
	exit 0
fi

bash "$SCRIPTS/ops/cpu_isolation.sh" verify
log "passed (live cpuset verify — see fault_proof from cpu_isolation.sh)"
