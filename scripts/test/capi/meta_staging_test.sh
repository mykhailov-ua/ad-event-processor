#!/usr/bin/env bash
# Role: Negative/dry-run harness for meta_staging.sh contract (metric hint and TRACK_URL guard).
# Execution context: Standalone shell test; no live Meta or stack required.
# Env knobs: CAPI_STAGING_DRY_RUN (1 for dry-run path).
# Verify: bash scripts/test/capi/meta_staging_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SCRIPT="$ROOT/scripts/test/capi/meta_staging.sh"

out="$(CAPI_STAGING_DRY_RUN=1 bash "$SCRIPT" 2>&1)" || {
  echo "capi_meta_staging_test: dry-run failed" >&2
  echo "$out" >&2
  exit 1
}

echo "$out" | grep -q 'ad_postback_dispatch_total{provider="facebook",status="success"}' || {
  echo "capi_meta_staging_test: dry-run missing facebook success metric hint" >&2
  exit 1
}

if CAPI_STAGING_DRY_RUN=0 bash "$SCRIPT" 2> /dev/null; then
  echo "capi_meta_staging_test: expected failure without TRACK_URL" >&2
  exit 1
fi

echo "capi_meta_staging_test: OK"
