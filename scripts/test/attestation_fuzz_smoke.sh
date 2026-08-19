#!/usr/bin/env bash

set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"
FUZZ_TIME="${ATTESTATION_FUZZ_TIME:-3s}"
echo "attestation_fuzz_smoke: FuzzAttestationTokenParse ${FUZZ_TIME}"
go test ./internal/ingestion -run='^$' -fuzz=FuzzAttestationTokenParse -fuzztime="${FUZZ_TIME}" -count=1
echo "attestation_fuzz_smoke: FuzzAttestationHMAC ${FUZZ_TIME}"
go test ./internal/ingestion -run='^$' -fuzz=FuzzAttestationHMAC -fuzztime="${FUZZ_TIME}" -count=1
echo "attestation_fuzz_smoke: OK"
