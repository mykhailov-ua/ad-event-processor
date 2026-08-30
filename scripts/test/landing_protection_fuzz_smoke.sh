#!/usr/bin/env bash

set -euo pipefail

# Role: Landing protection fuzz (JA3 parse, referrer checks) on ingest; skips when go missing.
# Execution context: Nightly parser security tier; no Docker.
# Invariants/contracts enforced: No panic on FuzzJA3Parse (30s) and related ingest fuzz targets.
# Verify: bash scripts/test/landing_protection_fuzz_smoke.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v go > /dev/null 2>&1; then
  echo "skip (go not found)"
  exit 0
fi

echo "landing_protection_fuzz_smoke: FuzzJA3Parse 30s"
go test ./internal/ingest -run='^$' -fuzz=FuzzJA3Parse -fuzztime=30s -count=1

echo "landing_protection_fuzz_smoke: FuzzLinkSignerVerify 30s"
go test ./internal/ingest -run='^$' -fuzz=FuzzLinkSignerVerify -fuzztime=30s -count=1

echo "landing_protection_fuzz_smoke: FuzzSafePageVerifyParse 30s"
go test ./internal/ingest -run='^$' -fuzz=FuzzSafePageVerifyParse -fuzztime=30s -count=1

echo "landing_protection_fuzz_smoke: FuzzBanditWeightSnapshot 30s"
go test ./internal/flow -run='^$' -fuzz=FuzzBanditWeightSnapshot -fuzztime=30s -count=1

echo "landing_protection_fuzz_smoke: OK"
