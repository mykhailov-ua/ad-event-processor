#!/usr/bin/env bash
# Role: Unit smoke for click-proxy deliver, redirect, and upstream URL builders.
# Execution context: Repo root go test; skips when go is missing.
# Env knobs: none.
# Verify: bash scripts/test/edge/click_smoke.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v go > /dev/null 2>&1; then
  echo "skip (go missing)"
  exit 0
fi

echo "click_proxy_smoke: go test click proxy package tests"
go test -count=1 -run 'TestClickProxyDeliver|TestClickRedirect_ProxyMode|TestBuildProxyUpstreamURL' ./internal/ingest/

echo "click_proxy_smoke: PASS"
