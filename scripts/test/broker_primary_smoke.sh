#!/usr/bin/env bash
# Smoke: broker-primary ClickHouse ingest (CH_INGEST_SOURCE=broker).
#
# Unit gates (always when go available):
#   - BrokerProducer hot-path tests
#   - BrokerConsumerGroup batch/offset tests
#
# Live stack (optional, when docker + running compose):
#   - broker health unix probe
#   - processor logs show Redis _ch disabled and broker bridge enabled
#
# Usage:
#   bash scripts/test/broker_primary_smoke.sh
#
# Env:
#   CH_INGEST_SOURCE=broker     skip when not broker (unless BROKER_PRIMARY_SMOKE_FORCE=1)
#   BROKER_PRIMARY_SMOKE_FORCE=1
#   BROKER_PRIMARY_SKIP_LIVE=1  unit tests only
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
# shellcheck source=../lib/go.sh
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

CH_INGEST_SOURCE="${CH_INGEST_SOURCE:-broker}"
COMPOSE_FILE="${BROKER_PRIMARY_COMPOSE_FILE:-deploy/compose/docker-compose.yaml}"

log() { printf 'broker-primary-smoke: %s\n' "$*"; }
die() { printf 'broker-primary-smoke: ERROR: %s\n' "$*" >&2; exit 1; }

if [[ "$CH_INGEST_SOURCE" != "broker" && "${BROKER_PRIMARY_SMOKE_FORCE:-0}" != "1" ]]; then
	log "skip (CH_INGEST_SOURCE=$CH_INGEST_SOURCE)"
	exit 0
fi

GO_BIN="$(ad_event_processor_go_bin)"
log "unit: BrokerProducer"
"$GO_BIN" test ./internal/ingestion/ -run '^TestBrokerProducer_' -count=1

log "unit: BrokerConsumerGroup"
"$GO_BIN" test ./cmd/processor/ -run '^TestBrokerConsumerGroup_' -count=1

if [[ "${BROKER_PRIMARY_SKIP_LIVE:-0}" == "1" ]]; then
	log "live stack skipped (BROKER_PRIMARY_SKIP_LIVE=1)"
	printf 'fault_proof fault=broker_primary_smoke status=partial proof=unit_only harness=go_test\n'
	exit 0
fi

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
	log "skip live (docker unavailable; unit gates passed)"
	printf 'fault_proof fault=broker_primary_smoke status=partial proof=unit_only harness=go_test docker=absent\n'
	exit 0
fi

COMPOSE=(docker compose -f "$COMPOSE_FILE")
BROKER_CID="$("${COMPOSE[@]}" ps -q broker 2>/dev/null | head -1 || true)"
if [[ -z "$BROKER_CID" ]]; then
	log "skip live (broker container not running; unit gates passed)"
	printf 'fault_proof fault=broker_primary_smoke status=partial proof=unit_only harness=go_test broker=down\n'
	exit 0
fi

if ! docker exec "$BROKER_CID" /broker --health-probe /run/ad-event-processor/broker/health.sock >/dev/null 2>&1; then
	die "broker health probe failed"
fi
log "ok  broker health"

PROC_CID="$("${COMPOSE[@]}" ps -q processor 2>/dev/null | head -1 || true)"
PROC_OK=0
if [[ -n "$PROC_CID" ]]; then
	logs="$(docker logs "$PROC_CID" 2>&1 | tail -200)"
	if echo "$logs" | grep -qE 'Redis _ch StreamConsumer disabled \(CH_INGEST_SOURCE=broker\)|Redis SettlementWorker disabled \(CH_INGEST_SOURCE=broker\)'; then
		log "ok  processor disabled Redis ingest path"
	else
		die "processor logs missing broker-primary disabled marker (_ch or SettlementWorker)"
	fi
	if echo "$logs" | grep -q 'Redis _fraud StreamConsumer disabled (CH_INGEST_SOURCE=broker)'; then
		log "ok  processor disabled Redis fraud stream consumer"
	else
		die "processor logs missing Redis _fraud disabled marker"
	fi
	if echo "$logs" | grep -q 'broker ingest bridge enabled'; then
		log "ok  processor broker bridge"
		PROC_OK=1
	else
		die "processor logs missing broker ingest bridge marker"
	fi
else
	log "skip processor log check (processor not running)"
fi

if [[ "$PROC_OK" -eq 1 ]]; then
	printf 'fault_proof fault=broker_primary_smoke status=ok proof=unit+broker_health+processor_logs harness=compose baseline_ok=true\n'
else
	printf 'fault_proof fault=broker_primary_smoke status=partial proof=unit+broker_health harness=compose processor=skipped\n'
fi
log "passed"
