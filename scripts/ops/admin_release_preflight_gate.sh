#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-.env.example}"

echo "admin release preflight: CAPI staging gate"
bash "$SCRIPTS/ci/capi_staging_gate.sh"

if [[ "${ADMIN_RELEASE_CAPI_LIVE:-}" == "1" ]]; then
  echo "admin release preflight: CAPI live smoke (mock Meta + metrics)"
  bash "$SCRIPTS/test/capi_staging_live_smoke.sh"
else
  echo "admin_release_preflight: CAPI live smoke skipped (set ADMIN_RELEASE_CAPI_LIVE=1 for local compose proof)"
fi

echo "admin release preflight: redis topology"
bash "$SCRIPTS/ops/verify_redis_topology.sh" "$ENV_FILE"

if [[ "${ADMIN_RELEASE_SKIP_PR_FAST:-}" == "1" ]]; then
  echo "admin release preflight: admin_web (pr_fast skipped)"
  bash "$SCRIPTS/ci/admin_web.sh"
else
  echo "admin release preflight: pr_fast (includes admin_web)"
  bash "$SCRIPTS/ci/pr_fast.sh"
fi

echo "admin_release_preflight_gate: OK (live Meta staging still requires scripts/test/capi_meta_staging.sh on stack)"
