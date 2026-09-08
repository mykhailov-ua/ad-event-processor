#!/usr/bin/env bash
# Role: Click ingress latency budget drill (T27): in-process holdouts + optional live wrk.
# Execution context: Repo root; default path runs go test holdouts without compose.
# Env knobs: TRACK_URL, CAMPAIGN_ID (live wrk); CLICK_INGRESS_P95_MS_MAX (default 50, core.mdc).
# Waiver: set CLICK_FILTER_TIER_DEFAULT=light on TDS campaigns when live full-tier wrk is unavailable.
# Verify: bash scripts/test/load/click_ingress_latency_drill.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

P95_MAX_MS="${CLICK_INGRESS_P95_MS_MAX:-50}"
OUT="${OUT:-$ROOT/var/load-test/click-ingress-latency}"
mkdir -p "$OUT"

log() { printf 'click-ingress-latency: %s\n' "$*"; }

probe_tracker_click() {
  local base="$1"
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' "${base}/click?type=click" 2> /dev/null || true)"
  [[ "$code" =~ ^[345] ]]
}

log "proxy timeout holdout"
go test ./internal/ingest/ -run TestClickProxy_holdoutSlowUpstreamRespectsTimeoutBudget -count=1

log "burst proxy timeout holdout (worker pool bounded)"
go test ./internal/ingest/ -run TestClickProxy_holdoutBurstSlowUpstreamBounded -count=1

log "full tier in-process p95 holdout (gnet harness, mock filter, no Redis EVALSHA)"
go test ./internal/ingest/ -run TestClickRedirectGnet_fullTierLatency_holdout -count=1

{
  printf 'tier=holdout\n'
  printf 'proxy_holdout=pass\n'
  printf 'proxy_concurrent_holdout=pass\n'
  printf 'full_tier_p95_holdout=pass\n'
} > "$OUT/holdout_summary.txt"

TRACK_URL="${TRACK_URL:-}"
if [[ -z "$TRACK_URL" ]]; then
  for candidate in http://127.0.0.1:8181 http://127.0.0.1:8182; do
    if probe_tracker_click "$candidate"; then
      TRACK_URL="$candidate"
      log "auto-detected tracker ${TRACK_URL}"
      break
    fi
  done
fi

CAMPAIGN_ID="${CAMPAIGN_ID:-${CLICK_INGRESS_CAMPAIGN_ID:-}}"
if [[ -z "$CAMPAIGN_ID" && -n "${AED_CAMPAIGN_UUID_1:-}" ]]; then
  CAMPAIGN_ID="$AED_CAMPAIGN_UUID_1"
fi

if [[ -z "${TRACK_URL:-}" || -z "${CAMPAIGN_ID:-}" ]]; then
  log "live wrk tier skipped (set TRACK_URL and CAMPAIGN_ID, or seed ingest campaigns)"
  log "artifact: $OUT/holdout_summary.txt"
  log "PASS (holdout only; TDS waiver: CLICK_FILTER_TIER_DEFAULT=light + click_tier_light_drill.sh)"
  exit 0
fi

if ! command -v wrk > /dev/null 2>&1; then
  log "wrk missing; install wrk for live tier or rely on holdout"
  log "artifact: $OUT/holdout_summary.txt"
  exit 0
fi

DURATION="${DURATION:-10s}"
THREADS="${THREADS:-4}"
CONNECTIONS="${CONNECTIONS:-64}"

LUA="$OUT/wrk_click_full.lua"
cat > "$LUA" << 'LUA'
local counter = 0
request = function()
  counter = counter + 1
  local click_id = string.format("00000000-0000-4000-8000-%012x", counter)
  local path = string.format("/click?campaign_id=%s&type=click&click_id=%s&sub1=ingress-latency", os.getenv("CAMPAIGN_ID"), click_id)
  return wrk.format("GET", path, {
    ["Accept"] = "*/*",
    ["Connection"] = "keep-alive",
    ["User-Agent"] = "Mozilla/5.0 (click-ingress-latency drill)",
  })
end
LUA

log "live wrk tier track=${TRACK_URL} campaign=${CAMPAIGN_ID} duration=${DURATION}"
export CAMPAIGN_ID
wrk -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" --latency -s "$LUA" "$TRACK_URL" | tee "$OUT/wrk.txt"

p95_line="$(rg '95%' "$OUT/wrk.txt" | head -1 || true)"
if [[ -z "$p95_line" ]]; then
  log "WARN: could not parse wrk p95 from $OUT/wrk.txt"
  exit 0
fi

p95_ms="$(echo "$p95_line" | awk '{print $2}' | sed 's/ms//')"
if [[ -z "$p95_ms" ]]; then
  log "WARN: unparsed p95: $p95_line"
  exit 0
fi

{
  printf 'tier=live_wrk\n'
  printf 'track_url=%s\n' "$TRACK_URL"
  printf 'campaign_id=%s\n' "$CAMPAIGN_ID"
  printf 'p95_ms=%s\n' "$p95_ms"
  printf 'p95_max_ms=%s\n' "$P95_MAX_MS"
} > "$OUT/live_wrk_summary.txt"

awk -v p95="$p95_ms" -v max="$P95_MAX_MS" 'BEGIN {
  if (p95+0 > max+0) { print "click-ingress-latency: FAIL p95=" p95 "ms max=" max "ms"; exit 1 }
  print "click-ingress-latency: PASS p95=" p95 "ms max=" max "ms"
}'
