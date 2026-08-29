#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
  echo "capi_staging_live_smoke: skip (docker unavailable)"
  exit 0
fi

if ! docker ps --format '{{.Names}}' | grep -q '^ad-event-processor-db-1$'; then
  echo "capi_staging_live_smoke: starting compose db/redis..."
  docker compose up -d db redis-0 redis-1 redis-2 redis-3 > /dev/null
fi

if [[ ! -x "$ROOT/bin/postback-sender" ]]; then
  echo "capi_staging_live_smoke: building postback-sender..."
  go build -o "$ROOT/bin/postback-sender" ./cmd/postback-sender
fi

exec bash "$SCRIPTS/test/capi/meta_bootstrap.sh" run
