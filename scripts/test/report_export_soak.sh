#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/load_test_env.sh"
cd "$ROOT"

load_test_source_env "$ROOT" 2> /dev/null || true
load_test_export_derived 2> /dev/null || true

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env" 2> /dev/null || true
  set +a
fi

CONTROL_URL="${CONTROL_URL:-${LOAD_TEST_CONTROL_URL:-http://127.0.0.1:${LOAD_TEST_CONTROL_PORT:-8800}}}"
ADMIN_API_KEY="${ADMIN_API_KEY:-${MANAGEMENT_API_KEY:-test-secret}}"
OUT_DIR="${1:-}"
PARALLEL="${REPORT_EXPORT_SOAK_PARALLEL:-4}"
WAIT_SEC="${REPORT_EXPORT_SOAK_WAIT_SEC:-300}"

log() { printf 'report-export-soak: %s\n' "$*"; }
die() {
  printf 'report-export-soak: ERROR: %s\n' "$*" >&2
  exit 1
}

wait_control() {
  local i=0
  while [[ $i -lt 90 ]]; do
    if curl -sf "${CONTROL_URL}/health" > /dev/null 2>&1; then
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  return 1
}

wait_jobs() {
  local -a job_ids=("$@")
  local deadline=$((SECONDS + WAIT_SEC))
  local pending=1

  while [[ "$pending" -eq 1 && "$SECONDS" -lt "$deadline" ]]; do
    pending=0
    for job_id in "${job_ids[@]}"; do
      [[ -n "$job_id" ]] || continue
      local body status
      body="$(curl -sf -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
        "${CONTROL_URL}/api/v1/reports/jobs/${job_id}" || true)"
      status="$(printf '%s' "$body" | jq -r '.status // empty' 2> /dev/null || true)"
      case "$status" in
        COMPLETED)
          log "job ${job_id} COMPLETED"
          ;;
        FAILED)
          die "job ${job_id} FAILED: $(printf '%s' "$body" | jq -r '.error // empty' 2> /dev/null || true)"
          ;;
        RUNNING | PENDING | "")
          pending=1
          ;;
        *)
          pending=1
          log "job ${job_id} status=${status}"
          ;;
      esac
    done
    [[ "$pending" -eq 1 ]] && sleep 2
  done

  if [[ "$pending" -eq 1 ]]; then
    die "export jobs did not finish within ${WAIT_SEC}s"
  fi
}

if ! wait_control; then
  die "control ${CONTROL_URL}/health not ready (export soak requires live control)"
fi

FROM="$(date -u -d '7 days ago' +%Y-%m-%dT%H:%M:%SZ 2> /dev/null || date -u -v-7d +%Y-%m-%dT%H:%M:%SZ)"
TO="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

CUSTOMER_JSON="$(curl -sf -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
  "${CONTROL_URL}/api/v1/customers?limit=1" || true)"
CUSTOMER_ID="$(printf '%s' "$CUSTOMER_JSON" | jq -r '.items[0].id // empty' 2> /dev/null || true)"
if [[ -z "$CUSTOMER_ID" ]]; then
  log "WARN: no customer for export soak; skipping"
  exit 0
fi

enqueue_job() {
  local key="$1"
  local n="$2"
  local resp
  resp="$(curl -sf -X POST "${CONTROL_URL}/api/v1/reports/jobs" \
    -H "Content-Type: application/json" \
    -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
    -H "Idempotency-Key: report-soak-${key}-${n}-$(date +%s)" \
    -d "$(jq -n \
      --arg cid "$CUSTOMER_ID" \
      --arg key "$key" \
      --arg from "$FROM" \
      --arg to "$TO" \
      '{customer_id:$cid,report_key:$key,from:$from,to:$to,format:"csv"}')")"
  printf '%s' "$resp" | jq -r '.id // .job_id // empty' 2> /dev/null
}

log "enqueue ${PARALLEL} parallel exports (placements + geo-roi) for customer ${CUSTOMER_ID}"
job_ids=()
for i in $(seq 1 "$PARALLEL"); do
  key="placements"
  if ((i % 2 == 0)); then
    key="geo-roi"
  fi
  job_id="$(enqueue_job "$key" "$i")"
  if [[ -z "$job_id" ]]; then
    die "export job POST did not return job id"
  fi
  job_ids+=("$job_id")
done

log "waiting for ${#job_ids[@]} export jobs (timeout ${WAIT_SEC}s)"
wait_jobs "${job_ids[@]}"
sleep 5

if [[ -n "$OUT_DIR" ]]; then
  mkdir -p "$OUT_DIR"
  {
    echo "customer_id=$CUSTOMER_ID"
    echo "parallel=$PARALLEL"
    echo "from=$FROM"
    echo "to=$TO"
    printf 'job_ids=%s\n' "$(
      IFS=,
      echo "${job_ids[*]}"
    )"
  } >> "${OUT_DIR}/report_export_soak.txt"
fi

log "export soak complete"
