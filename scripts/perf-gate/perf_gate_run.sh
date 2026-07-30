#!/usr/bin/env bash
set -euo pipefail
export PERF_GATE_STRICT="${PERF_GATE_STRICT:-false}"
exec "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/perf/gate_run.sh" "$@"
