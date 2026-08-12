#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

LOG="${RESILIENCE_LOG:-/tmp/espx-resilience.log}"
MIN_PROOFS="${RESILIENCE_MIN_PROOFS:-52}"
export BROKER_FAULT_LAB=1

if [[ "${CI:-}" == "true" ]] && command -v apt-get >/dev/null 2>&1 && ! command -v stress-ng >/dev/null 2>&1; then
	echo "run_resilience: installing stress-ng for CH spool fault gate"
	sudo apt-get update -qq
	sudo apt-get install -y -qq stress-ng
fi

go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0 generate
go fmt ./...

go test -count=1 -v -run 'Fault' -timeout 20m \
	./tests/... \
	./internal/database/... \
	./internal/identity/... \
	./internal/ingestion/... \
	./internal/payment/... \
	./internal/ledger/... \
	./internal/licensing/... \
	./internal/notify/... \
	./internal/ivtdetector/... \
	./internal/fraud/... \
	./pkg/broker/server/... \
	./internal/controlplane/... \
	./internal/edge/bpf/... \
	./internal/edge/perimeter/... \
	./internal/rtb/... \
	./internal/logevacuator/... \
	2>&1 | tee "$LOG"

PROOFS="$(grep -c 'fault_proof fault=' "$LOG" || true)"
echo "fault_proof lines: $PROOFS (min $MIN_PROOFS)"
test "$PROOFS" -ge "$MIN_PROOFS"

MR_PROOFS="$(grep -c 'fault_proof fault=mr_' "$LOG" || true)"
MIN_MR_PROOFS="${RESILIENCE_MIN_PROOFS_MR:-12}"
echo "mr fault_proof lines: $MR_PROOFS (min $MIN_MR_PROOFS)"
test "$MR_PROOFS" -ge "$MIN_MR_PROOFS"

bash "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/fault/milestone_gates.sh" "$LOG"
