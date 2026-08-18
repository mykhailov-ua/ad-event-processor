#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ "${PREFLIGHT_SMOKE:-0}" == "1" ]]; then
  bash "$SCRIPTS/ci/validate_configs.sh"
  echo "preflight smoke: OK (configs only; start stack for full dependency check)"
  exit 0
fi

bash "$SCRIPTS/ci/deps.sh"
bash "$SCRIPTS/dev/smoke_local.sh"
