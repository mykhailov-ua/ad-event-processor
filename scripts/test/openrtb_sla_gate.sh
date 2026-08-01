#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

SESSION_DIR="${1:-}"
if [[ -z "$SESSION_DIR" ]]; then
	echo "usage: $0 <load-session-dir>" >&2
	exit 1
fi

PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
export LOAD_SLA_GATE=1

go run ./cmd/load-report sla "$SESSION_DIR" --prom "$PROM_URL"
echo "openrtb_sla_gate: PASS — $SESSION_DIR/sla-gate.md"
