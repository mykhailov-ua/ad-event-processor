#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

BROKER_FRAUD_TOPIC="${BROKER_FRAUD_TOPIC:-ad-fraud-events}"
REDIS_GROUP_NAME="${REDIS_GROUP_NAME:-ad_event_processor}"
PROC_URL="${INGEST_METRICS_PROCESSOR_URL:-http://127.0.0.1:9091/metrics}"
TRACK_URL="${INGEST_METRICS_TRACKER_URL:-http://127.0.0.1:9090/metrics}"
CH_GROUP="${REDIS_GROUP_NAME}_ch_broker"
FRAUD_GROUP="${REDIS_GROUP_NAME}_fraud_broker"
FRAUD_STREAM="${FRAUD_STREAM_NAME:-ad:fraud:stream}"

log() { printf 'ingest-pipeline-metrics: %s\n' "$*"; }

print_queries() {
  cat << EOF

ad_ingest_fraud_path


ad_fraud_stream_pel_age_seconds{stream="${FRAUD_STREAM}"}


ad_broker_consumer_lag_messages{topic="${BROKER_TOPIC:-tracker-logs}",group="${CH_GROUP}"}
ad_broker_consumer_lag_messages{topic="${BROKER_FRAUD_TOPIC}",group="${FRAUD_GROUP}"}


sum by (topic, group) (rate(ad_broker_ingest_messages_total[5m]))
sum by (topic, group) (rate(ad_broker_ingest_commits_total[5m]))






EOF
}

fetch_metric() {
  local url="$1"
  local pattern="$2"
  if command -v curl > /dev/null 2>&1; then
    curl -fsS --max-time 3 "$url" 2> /dev/null | grep -E "$pattern" || true
  fi
}

log "metric catalog (Prometheus / Grafana)"
print_queries
echo

if [[ "${INGEST_METRICS_SKIP_LIVE:-0}" == "1" ]]; then
  log "live scrape skipped (INGEST_METRICS_SKIP_LIVE=1)"
  printf 'fault_proof fault=ingest_pipeline_metrics status=partial proof=catalog_only harness=script\n'
  exit 0
fi

log "live scrape processor: $PROC_URL"
fetch_metric "$PROC_URL" '^(ad_ingest_fraud_path|ad_broker_consumer_lag_messages|ad_broker_ingest_(messages|commits)_total|ad_processor_stream_lag_seconds)'
echo

log "live scrape tracker: $TRACK_URL"
fetch_metric "$TRACK_URL" '^(ad_ingest_fraud_path|ad_fraud_stream_pel_age_seconds|ad_broker_produced_events_total\{status="fraud_)'
echo

printf 'fault_proof fault=ingest_pipeline_metrics status=partial proof=catalog+live_scrape harness=script ch_ingest_source=%s fraud_topic=%s\n' \
  "${CH_INGEST_SOURCE:-}" "$BROKER_FRAUD_TOPIC"
log "done — save this output for before/after diff"
