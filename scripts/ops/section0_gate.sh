#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-.env.example}"

echo "== section0: admin web build =="
node "$ROOT/web/scripts/build.mjs"

echo "== section0: CAPI staging gate =="
bash "$SCRIPTS/ci/capi_staging_gate.sh"

if [[ "${SECTION0_CAPI_LIVE:-}" == "1" ]]; then
  echo "== section0: CAPI live smoke (mock Meta + metrics) =="
  bash "$SCRIPTS/test/capi_section0_live.sh"
else
  echo "section0: CAPI live smoke skipped (set SECTION0_CAPI_LIVE=1 for local compose proof)"
fi

echo "== section0: redis topology =="
bash "$SCRIPTS/ops/verify_redis_topology.sh" "$ENV_FILE"

echo "== section0: admin_web CI =="
bash "$SCRIPTS/ci/admin_web.sh"

if [[ "${SECTION0_SKIP_PR_FAST:-}" == "1" ]]; then
  echo "section0: pr_fast skipped (SECTION0_SKIP_PR_FAST=1)"
else
  echo "== section0: pr_fast =="
  bash "$SCRIPTS/ci/pr_fast.sh"
fi

echo "section0_gate: OK (live Meta staging still requires scripts/test/capi_meta_staging.sh on stack)"
