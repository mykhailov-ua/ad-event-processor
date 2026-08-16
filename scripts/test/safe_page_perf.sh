#!/usr/bin/env bash
# Safe-page (cloak companion) micro-bench + optional BPF session during gnet load.
# Preconditions: go 1.25+; BPF needs sudo/BTF (see bpf_requirements.sh).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

OUT="${OUT:-$ROOT/var/load-test/safe-page-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUT"

log() { printf 'safe-page-perf: %s\n' "$*"; }

export GOMAXPROCS="${GOMAXPROCS:-1}"
BENCH_PATTERN='ClickRedirect|SafePage|TrackVerify|ParseClick|ResolveSafePage'

log "micro-bench (ingestion, pattern=$BENCH_PATTERN)"
go test -run='^$' -bench="$BENCH_PATTERN" -benchmem -count="${BENCH_COUNT:-5}" ./internal/ingestion/ \
	| tee "$OUT/micro_bench.txt"

if [[ "${ESPX_BPF_PROBE:-0}" == "1" ]]; then
	if bash "$SCRIPTS/test/bpf_requirements.sh"; then
		log "BPF session during sustained bench"
		bash "$SCRIPTS/test/bpf_probe_session.sh" start "$OUT" || log "WARN: BPF start failed"
		BPF_PID=""
		[[ -f "$OUT/bpf/collector.pid" ]] && BPF_PID="$(cat "$OUT/bpf/collector.pid")"
		go test -run='^$' -bench='ClickRedirectGnet_forceSafe|SafePageStubGnet|TrackVerifyGnet' \
			-benchtime=5s -count=2 ./internal/ingestion/ \
			| tee -a "$OUT/micro_bench_sustained.txt" || true
		if [[ -n "$BPF_PID" ]]; then
			bash "$SCRIPTS/test/bpf_probe_session.sh" stop "$OUT" "$BPF_PID" || true
		fi
		if [[ -f "$OUT/bpf/maps/summary.json" ]]; then
			go run ./cmd/load-report bpf "$OUT" || true
		fi
	else
		log "skip BPF (requirements failed)"
	fi
else
	log "skip BPF (set ESPX_BPF_PROBE=1 to enable)"
fi

log "done — $OUT"
