#!/usr/bin/env bash
# Role: Verify light click_filter_tier redirect latency (in-process holdout + optional live wrk).
# Execution context: Repo root; default path runs go test holdout without compose.
# Env knobs: TRACK_URL, CAMPAIGN_ID (live wrk tier); CLICK_TIER_LIGHT_P99_MS_MAX (default 15).
# Verify: bash scripts/test/load/click_tier_light_drill.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

P99_MAX_MS="${CLICK_TIER_LIGHT_P99_MS_MAX:-15}"

log() { printf 'click-tier-light: %s\n' "$*"; }

log "in-process holdout (gnet harness, no Redis)"
go test ./internal/ingest/ -run TestClickRedirectGnet_lightTierLatency_holdout -count=1

if [[ -z "${TRACK_URL:-}" || -z "${CAMPAIGN_ID:-}" ]]; then
  log "live wrk tier skipped (set TRACK_URL and CAMPAIGN_ID for load-test p99)"
  log "PASS (holdout only)"
  exit 0
fi

if ! command -v wrk > /dev/null 2>&1; then
  log "wrk missing; install wrk for live tier or rely on holdout"
  exit 0
fi

OUT="${OUT:-$CI_ARTIFACT_DIR/click-tier-light}"
mkdir -p "$OUT"
DURATION="${DURATION:-10s}"
THREADS="${THREADS:-4}"
CONNECTIONS="${CONNECTIONS:-64}"

LUA="$OUT/wrk_click_light.lua"
cat > "$LUA" << 'LUA'
local counter = 0
request = function()
  counter = counter + 1
  local click_id = string.format("00000000-0000-4000-8000-%012x", counter)
  local path = string.format("/click?campaign_id=%s&type=click&click_id=%s&sub1=light-tier", os.getenv("CAMPAIGN_ID"), click_id)
  return wrk.format("GET", path, {
    ["Accept"] = "*/*",
    ["Connection"] = "keep-alive",
    ["User-Agent"] = "Mozilla/5.0 (click-tier-light drill)",
  })
end
LUA

log "live wrk tier track=${TRACK_URL} duration=${DURATION}"
export CAMPAIGN_ID
wrk -t"$THREADS" -c"$CONNECTIONS" -d"$DURATION" --latency -s "$LUA" "$TRACK_URL" | tee "$OUT/wrk.txt"

p99_line="$(rg '99%' "$OUT/wrk.txt" | head -1 || true)"
if [[ -z "$p99_line" ]]; then
  log "WARN: could not parse wrk p99 from $OUT/wrk.txt"
  exit 0
fi

p99_ms="$(echo "$p99_line" | awk '{print $2}' | sed 's/ms//')"
if [[ -z "$p99_ms" ]]; then
  log "WARN: unparsed p99: $p99_line"
  exit 0
fi

awk -v p99="$p99_ms" -v max="$P99_MAX_MS" 'BEGIN {
  if (p99+0 > max+0) { print "click-tier-light: FAIL p99=" p99 "ms max=" max "ms"; exit 1 }
  print "click-tier-light: PASS p99=" p99 "ms max=" max "ms"
}'
