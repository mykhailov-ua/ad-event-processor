#!/usr/bin/env bash
# Role: Reverse-proxy close cohort: RP baseline tests, attestation fuzz, optional constrained load.
# Execution context: Repo root; load tier needs Docker and REVERSE_PROXY_SKIP_LOAD=0.
# Env knobs: REVERSE_PROXY_SKIP_LOAD (1 skips load/BPF); ATTESTATION_FUZZ_TIME (3s).
# Verify: bash scripts/test/edge/reverse_proxy_close_smoke.sh
set -euo pipefail
cd "$(dirname "$0")/../../.."
ROOT="$(pwd)"

echo "RP: baseline audit"
go test ./internal/ingest/ -short -count=1 -run 'TestRPBaseline'

echo "RP: click redirect + attestation unit"
go test ./internal/ingest/ -short -count=1 -run 'TestCIDR|TestClickRedirect_L1|TestClickRedirect_Attestation|TestClickProxy|TestMintAttestation'

echo "RP: attestation fuzz smoke"
ATTESTATION_FUZZ_TIME=3s bash scripts/test/attestation_fuzz_smoke.sh

echo "RP: click proxy smoke"
bash scripts/test/edge/click_smoke.sh

if [ "${REVERSE_PROXY_SKIP_LOAD:-1}" = "1" ]; then
  echo "reverse_proxy_close_smoke: OK (load/BPF skipped - set REVERSE_PROXY_SKIP_LOAD=0 + Docker for AC-1)"
  exit 0
fi

echo "RP: load cohort (needs constrained stack)"
bash scripts/test/load/prepare_constrained_stack.sh
PCT_CLICK_PROXY=10 AD_EVENT_PROCESSOR_BPF_PROBE=1 bash scripts/test/load/malformed.sh business
echo "reverse_proxy_close_smoke: load run complete - paste go run ./cmd/load-report all var/load-test/<ts>/"
