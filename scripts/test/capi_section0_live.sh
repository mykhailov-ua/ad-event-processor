#!/usr/bin/env bash
# §0 live CAPI smoke: local compose + mock Meta receiver + postback-sender metrics.
# Proves ad_postback_dispatch_total{provider="facebook",status="success"} without real Meta.
#
# Real Meta staging (Events Manager test stream): set META_CAPI_URL + META_ACCESS_TOKEN
# and META_TEST_EVENT_CODE, then run scripts/test/capi_meta_staging_live.sh instead.
#
# Usage:
#   bash scripts/test/capi_section0_live.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
  echo "capi_section0_live: skip (docker unavailable)"
  exit 0
fi

if ! docker ps --format '{{.Names}}' | grep -q '^bidshard-db-1$'; then
  echo "capi_section0_live: starting compose db/redis..."
  docker compose up -d db redis-0 redis-1 redis-2 redis-3 > /dev/null
fi

if [[ ! -x "$ROOT/bin/postback-sender" ]]; then
  echo "capi_section0_live: building postback-sender..."
  go build -o "$ROOT/bin/postback-sender" ./cmd/postback-sender
fi

exec bash "$SCRIPTS/test/capi_meta_bootstrap.sh" run
