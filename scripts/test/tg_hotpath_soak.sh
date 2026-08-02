#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PREPARE="${PREPARE:-1}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-200}"
TRACKER_BASES="${TRACKER_BASES:-http://127.0.0.1:8181}"
OUT="${OUT:-$ROOT/var/load-test/tg-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"

CAMPAIGN_ID="${TG_SOAK_CAMPAIGN_ID:-00000000-0000-0000-0000-000000000001}"
CLICK_ID="${TG_SOAK_CLICK_ID:-d5671191-236b-4e94-825e-399185a9bc8d}"
BRIDGE_TOKEN="${TG_SOAK_BRIDGE_TOKEN:-token_abc123_}"

log() { printf 'tg-hotpath-soak: %s\n' "$*"; }

if [[ "$PREPARE" == "1" ]]; then
	log "preparing constrained stack"
	bash "$SCRIPTS/test/prepare_constrained_stack.sh" 2>&1 | tee "$OUT/compose.log"
fi

IFS=',' read -r -a BASES <<<"$TRACKER_BASES"

BPF_PID=""
if [[ "${ESPX_BPF_PROBE:-0}" == "1" ]]; then
	bash "$SCRIPTS/test/bpf_probe_session.sh" start "$OUT" || log "WARN: BPF start failed (need sudo + bpf-dev)"
	[[ -f "$OUT/bpf/collector.pid" ]] && BPF_PID="$(cat "$OUT/bpf/collector.pid")"
fi

log "soak GET /tg/click rate=$RATE duration=$DURATION"
end=$((SECONDS + $(printf '%s' "$DURATION" | sed -E 's/s$//')))
ok=0
fail=0
iter=0
declare -A STATUS_COUNTS=()
	while [[ $SECONDS -lt $end ]]; do
	base="${BASES[$((iter % ${#BASES[@]}))]}"
	click_id="$(printf '00000000-0000-4000-8000-%012x' "$iter")"
	path="/tg/click?campaign_id=${CAMPAIGN_ID}&click_id=${click_id}&bridge_token=${BRIDGE_TOKEN}"
	code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 --http1.1 --keepalive-time 120 "${base}${path}" || echo 000)"
	STATUS_COUNTS["$code"]=$(( ${STATUS_COUNTS[$code]:-0} + 1 ))
	if [[ "$code" == "302" || "$code" == "404" || "$code" == "409" ]]; then
		ok=$((ok + 1))
	else
		fail=$((fail + 1))
	fi
	iter=$((iter + 1))
	if (( iter % RATE == 0 )); then
		sleep 1
	fi
done

{
	echo "duration=$DURATION rate=$RATE"
	echo "requests=$iter ok=$ok fail=$fail"
	echo "status_histogram=$(printf '%s' "${!STATUS_COUNTS[@]}" | tr ' ' ',' | sed 's/^/[/;s/$/]/')"
	for code in "${!STATUS_COUNTS[@]}"; do
		echo "status_${code}=${STATUS_COUNTS[$code]}"
	done
	echo "campaign_id=$CAMPAIGN_ID click_id=$CLICK_ID"
} | tee "$OUT/summary.txt"

[[ -n "$BPF_PID" ]] && bash "$SCRIPTS/test/bpf_probe_session.sh" stop "$OUT" "$BPF_PID" || true

if [[ "${STATUS_COUNTS[405]:-0}" -gt 0 || "${STATUS_COUNTS[000]:-0}" -gt 0 ]]; then
	log "ERROR: route missing or unreachable (405/000 in histogram)"
	exit 1
fi
if [[ "$ok" -lt $((iter * 8 / 10)) ]]; then
	log "ERROR: success rate too low ($ok/$iter); want >=80% 302/404/409"
	exit 1
fi

log "PASS — $OUT"
