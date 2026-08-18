#!/usr/bin/env bash
# Referrer-policy closure smoke — baseline audit, attestation, proxy unit paths.
# Run from repo root: bash scripts/test/rp_close_smoke.sh
# Skip load/BPF lab: RP_SKIP_LOAD=1 (default when Docker stack unavailable)
set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"

echo "== RP: baseline audit =="
go test ./internal/ingestion/ -short -count=1 -run 'TestRPBaseline'

echo "== RP: click redirect + attestation unit =="
go test ./internal/ingestion/ -short -count=1 -run 'TestCIDR|TestClickRedirect_L1|TestClickRedirect_Attestation|TestClickProxy|TestMintAttestation'

echo "== RP: attestation fuzz smoke =="
ATTESTATION_FUZZ_TIME=3s bash scripts/test/attestation_fuzz_smoke.sh

echo "== RP: click proxy smoke =="
bash scripts/test/click_proxy_smoke.sh

if [ "${RP_SKIP_LOAD:-1}" = "1" ]; then
  echo "rp_close_smoke: OK (load/BPF skipped — set RP_SKIP_LOAD=0 + Docker for AC-1)"
  exit 0
fi

echo "== RP: load cohort (needs constrained stack) =="
bash scripts/test/prepare_constrained_stack.sh
PCT_CLICK_PROXY=10 ESPX_BPF_PROBE=1 bash scripts/test/malformed.sh business
echo "rp_close_smoke: load run complete — paste go run ./cmd/load-report all var/load-test/<ts>/"
