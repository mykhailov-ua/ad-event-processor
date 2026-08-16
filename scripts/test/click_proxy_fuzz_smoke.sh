#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

FUZZTIME="${CLICK_PROXY_FUZZ_TIME:-5s}"

for name in ProxyUpstreamURL ProxyResponseHeaderParse; do
	echo "click_proxy_fuzz_smoke: Fuzz${name} ($FUZZTIME)"
	go test -run='^$' -fuzz="Fuzz${name}" -fuzztime="$FUZZTIME" ./internal/ingestion/
done

echo "click_proxy_fuzz_smoke: PASS"
