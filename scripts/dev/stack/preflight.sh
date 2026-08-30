#!/usr/bin/env bash
# Role: Dev stack readiness gate after clone or compose change.
# Execution context: Repo root; runs CI deps check then local HTTP/redis smoke unless smoke-only.
# Env knobs: PREFLIGHT_SMOKE (1 skips live stack, configs-only via validate_configs.sh).
# Verify: bash scripts/dev/stack/preflight.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

# PREFLIGHT_SMOKE=1 for CI config validation without a running compose stack.
if [[ "${PREFLIGHT_SMOKE:-0}" == "1" ]]; then
  bash "$SCRIPTS/ci/validate_configs.sh"
  echo "preflight smoke: OK (configs only; start stack for full dependency check)"
  exit 0
fi

bash "$SCRIPTS/ci/deps.sh"
bash "$SCRIPTS/dev/stack/smoke_local.sh"
