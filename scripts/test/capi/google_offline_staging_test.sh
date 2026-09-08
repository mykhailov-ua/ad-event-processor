#!/usr/bin/env bash
# Role: Negative/dry-run harness for google_offline_staging.sh contract (metric hint and TRACK_URL guard).
# Execution context: Standalone shell test; no live Google Ads or stack required.
# Env knobs: GOOGLE_OFFLINE_STAGING_DRY_RUN (1 for dry-run path).
# Verify: bash scripts/test/capi/google_offline_staging_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SCRIPT="$ROOT/scripts/test/capi/google_offline_staging.sh"

out="$(GOOGLE_OFFLINE_STAGING_DRY_RUN=1 bash "$SCRIPT" 2>&1)" || {
  echo "google_offline_staging_test: dry-run failed" >&2
  echo "$out" >&2
  exit 1
}

echo "$out" | grep -q 'ad_postback_dispatch_total{provider="google",status="success"}' || {
  echo "google_offline_staging_test: dry-run missing google success metric hint" >&2
  exit 1
}

if GOOGLE_OFFLINE_STAGING_DRY_RUN=0 bash "$SCRIPT" 2> /dev/null; then
  echo "google_offline_staging_test: expected failure without TRACK_URL" >&2
  exit 1
fi

echo "google_offline_staging_test: OK"
