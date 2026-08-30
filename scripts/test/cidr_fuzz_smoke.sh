#!/usr/bin/env bash
set -euo pipefail

# Role: Timeboxed CIDR parser fuzz smoke (FuzzCIDRParse/Match/Build) on ingest hot path.
# Execution context: Nightly or release QA; no Docker.
# Invariants/contracts enforced: No panic within CIDR_FUZZ_TIME (default 5s) per target.
# Verify: bash scripts/test/cidr_fuzz_smoke.sh
# Env: CIDR_FUZZ_TIME
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

FUZZTIME="${CIDR_FUZZ_TIME:-5s}"

for name in CIDRParse CIDRMatch CIDRBuild; do
  echo "cidr_fuzz_smoke: Fuzz${name} ($FUZZTIME)"
  go test -run='^$' -fuzz="Fuzz${name}" -fuzztime="$FUZZTIME" ./internal/ingest/
done

echo "cidr_fuzz_smoke: PASS"
