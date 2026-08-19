#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v go > /dev/null 2>&1; then
  echo "skip (go missing)"
  exit 0
fi

echo "click_proxy_smoke: go test click proxy package tests"
go test -count=1 -run 'TestClickProxyDeliver|TestClickRedirect_ProxyMode|TestBuildProxyUpstreamURL' ./internal/ingestion/

echo "click_proxy_smoke: PASS"
