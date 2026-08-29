#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

FUZZTIME="${CIDR_FUZZ_TIME:-5s}"

for name in CIDRParse CIDRMatch CIDRBuild; do
  echo "cidr_fuzz_smoke: Fuzz${name} ($FUZZTIME)"
  go test -run='^$' -fuzz="Fuzz${name}" -fuzztime="$FUZZTIME" ./internal/ingest/
done

echo "cidr_fuzz_smoke: PASS"
