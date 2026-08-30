#!/usr/bin/env bash

set -euo pipefail

# Role: Timeboxed attestation token/HMAC fuzz smoke on ingest package.
# Execution context: Nightly or release QA; no Docker.
# Invariants/contracts enforced: No panic within ATTESTATION_FUZZ_TIME (default 3s) per fuzz target.
# Verify: bash scripts/test/attestation_fuzz_smoke.sh
# Env: ATTESTATION_FUZZ_TIME
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
FUZZ_TIME="${ATTESTATION_FUZZ_TIME:-3s}"
echo "attestation_fuzz_smoke: FuzzAttestationTokenParse ${FUZZ_TIME}"
go test ./internal/ingest -run='^$' -fuzz=FuzzAttestationTokenParse -fuzztime="${FUZZ_TIME}" -count=1
echo "attestation_fuzz_smoke: FuzzAttestationHMAC ${FUZZ_TIME}"
go test ./internal/ingest -run='^$' -fuzz=FuzzAttestationHMAC -fuzztime="${FUZZ_TIME}" -count=1
echo "attestation_fuzz_smoke: OK"
