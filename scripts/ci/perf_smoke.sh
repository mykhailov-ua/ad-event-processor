#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

export PERF_GATE_STRICT=false
bash "$SCRIPTS/test/gate_run.sh"
