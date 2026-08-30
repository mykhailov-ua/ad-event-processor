#!/usr/bin/env bash
# Role: Short fuzz runs for click-proxy URL and header parsers on ingest hot path.
# Execution context: Repo root go test fuzz targets.
# Env knobs: CLICK_PROXY_FUZZ_TIME (5s per target).
# Verify: bash scripts/test/edge/click_fuzz_smoke.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

FUZZTIME="${CLICK_PROXY_FUZZ_TIME:-5s}"

for name in ProxyUpstreamURL ProxyResponseHeaderParse; do
  echo "click_proxy_fuzz_smoke: Fuzz${name} ($FUZZTIME)"
  go test -run='^$' -fuzz="Fuzz${name}" -fuzztime="$FUZZTIME" ./internal/ingest/
done

echo "click_proxy_fuzz_smoke: PASS"
