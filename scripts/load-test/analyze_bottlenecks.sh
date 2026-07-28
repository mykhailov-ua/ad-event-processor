#!/usr/bin/env bash
# Post-load bottleneck analysis: Prometheus hot-path metrics, FD snapshots, syscall summaries.
# Usage: analyze_bottlenecks.sh [output_dir]
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

OUT="${1:-$ROOT/var/load-test/$(date -u +%Y%m%dT%H%M%SZ)}"
PROM="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
mkdir -p "$OUT"

log() { printf 'analyze-bottlenecks: %s\n' "$*"; }
warn() { printf 'analyze-bottlenecks: WARN: %s\n' "$*" >&2; }

prom_scalar() {
	local q=$1
	local v
	v="$(curl -sfG --max-time 10 --data-urlencode "query=${q}" "${PROM}/api/v1/query" 2>/dev/null \
		| python3 -c 'import json,sys; d=json.load(sys.stdin); r=d.get("data",{}).get("result",[]); print(r[0]["value"][1] if r else "")' 2>/dev/null || true)"
	v="${v:-na}"
	printf '%s' "$v"
}

prom_vector() {
	local q=$1
	curl -sfG --max-time 10 --data-urlencode "query=${q}" "${PROM}/api/v1/query" 2>/dev/null \
		| python3 -c '
import json,sys
d=json.load(sys.stdin)
for r in d.get("data",{}).get("result",[]):
    m=r.get("metric",{})
    lbl=",".join(f"{k}={v}" for k,v in sorted(m.items()) if k!="__name__")
    print(f"{r[\"value\"][1]}\t{lbl}")
' 2>/dev/null || true
}

REPORT="$OUT/bottleneck-report.md"
{
	echo "# eSPX Dirty Load Bottleneck Report"
	echo ""
	echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "Prometheus: $PROM"
	echo "Grafana: http://127.0.0.1:3100"
	echo ""
	echo "## Ingress (tracker gnet)"
	echo ""
	echo "| Metric | Value |"
	echo "|--------|-------|"
	printf '| Tracker RPS | %s |\n' "$(prom_scalar 'sum(rate(ad_http_request_duration_seconds_count{job="tracker"}[5m]))')"
	printf '| Tracker p95 (ms) | %s |\n' "$(prom_scalar 'histogram_quantile(0.95, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000')"
	printf '| Tracker p99 (ms) | %s |\n' "$(prom_scalar 'histogram_quantile(0.99, sum(rate(ad_http_request_duration_seconds_bucket{job="tracker"}[5m])) by (le)) * 1000')"
	printf '| gnet active connections | %s |\n' "$(prom_scalar 'sum(ad_gnet_active_connections{job="tracker"})')"
	printf '| Worker pool rejects/s | %s |\n' "$(prom_scalar 'sum(rate(ad_worker_pool_reject_total{job="tracker"}[5m]))')"
	printf '| HTTP parse errors/s | %s |\n' "$(prom_scalar 'sum(rate(ad_http_parse_errors_total{job="tracker"}[5m]))')"
	echo ""
	echo "## Redis Lua (user/kernel boundary: TCP + epoll + write)"
	echo ""
	echo "| Shard | Lua p99 (ms) | ops/s | NOSCRIPT/s |"
	echo "|-------|--------------|-------|------------|"
	while IFS=$'\t' read -r val shard; do
		[[ -n "$shard" ]] || continue
		sh="${shard#shard=}"
		p99="$(prom_scalar "histogram_quantile(0.99, sum(rate(ad_redis_lua_duration_seconds_bucket{job=\"tracker\",shard=\"$sh\"}[5m])) by (le)) * 1000")"
		ops="$(prom_scalar "sum(rate(ad_redis_ops_total{job=\"tracker\",shard=\"$sh\"}[5m]))")"
		noscript="$(prom_scalar "sum(rate(ad_redis_lua_noscript_total{job=\"tracker\",shard=\"$sh\"}[5m]))")"
		printf '| %s | %s | %s | %s |\n' "$sh" "${p99:-na}" "${ops:-na}" "${noscript:-na}"
	done < <(prom_vector 'sum(rate(ad_redis_ops_total{job="tracker"}[5m])) by (shard)')
	echo ""
	echo "## Filter / fraud path"
	echo ""
	echo "- Fraud stream drops/s: $(prom_scalar 'sum(rate(ad_fraud_stream_drop_total{job="tracker"}[5m]))')"
	echo "- Events dropped (Redis ingest)/s: $(prom_scalar 'sum(rate(ad_events_dropped_total{job="tracker"}[5m]))')"
	echo "- Fraud tier block/s: $(prom_scalar 'sum(rate(ad_filter_blocked_total{reason="fraud"}[5m]))')"
	echo "- Budget cache miss PG/s: $(prom_scalar 'sum(rate(ad_budget_cache_miss_pg_total{job="tracker"}[5m]))')"
	echo ""
	echo "### Filter blocks by reason"
	echo '```'
	prom_vector 'sum(rate(ad_filter_blocked_total{job="tracker"}[5m])) by (reason)' | sort -rn | head -20
	echo '```'
	echo ""
	echo "## Cold path: pgx (Postgres) + clickhouse-go"
	echo ""
	echo "| Store | p99 batch write (ms) | errors/s |"
	echo "|-------|---------------------|----------|"
	for typ in postgres clickhouse; do
		p99="$(prom_scalar "histogram_quantile(0.99, sum(rate(ad_db_write_duration_seconds_bucket{job=\"processor\",type=\"$typ\"}[5m])) by (le)) * 1000")"
		err="$(prom_scalar "sum(rate(ad_db_write_errors_total{job=\"processor\",type=\"$typ\"}[5m]))")"
		printf '| %s | %s | %s |\n' "$typ" "${p99:-na}" "${err:-0}"
	done
	echo ""
	echo "- Tracker stream ingest/s: $(prom_scalar 'sum(rate(ad_events_processed_total{job=~"tracker.*"}[5m]))')"
	echo "- Processor PG write errors/s: $(prom_scalar 'sum(rate(ad_db_write_errors_total{job="processor",type="postgres"}[5m]))')"
	echo "- Processor CH write errors/s: $(prom_scalar 'sum(rate(ad_db_write_errors_total{job="processor",type="clickhouse"}[5m]))')"
	echo "- Processor stream lag (sum xlen): $(prom_scalar 'sum(ad_processor_stream_xlen{job="processor"})')"
	echo "- DLQ size: $(prom_scalar 'sum(ad_dlq_size_total{job="processor"})')"
	echo ""
	echo "## Edge (nginx OpenResty)"
	echo ""
	echo "- Phase1 pass/s: $(prom_scalar 'sum(rate(espx_edge_phase1_pass_total[5m]))')"
	echo "- Circuit reject/s: $(prom_scalar 'sum(rate(espx_edge_circuit_reject_total[5m]))')"
	echo "- Blocked IP/s: $(prom_scalar 'sum(rate(espx_edge_blocked_ip_total[5m]))')"
	echo ""
	echo "## File descriptors & syscalls"
	echo ""
	if compgen -G "$OUT/*-strace-*.txt" >/dev/null 2>&1; then
		echo "Strace summaries captured under \`$OUT\`. Top syscalls per service:"
		echo '```'
		for f in "$OUT"/*-strace-*.txt; do
			echo "=== $(basename "$f") ==="
			grep -E '%' "$f" 2>/dev/null | head -8 || true
		done
		echo '```'
	else
		echo "_No strace samples. Re-run with snapshot_runtime.sh during load._"
	fi
	if compgen -G "$OUT/espx-*.txt" >/dev/null 2>&1; then
		echo ""
		echo "FD counts:"
		echo '```'
		grep -h 'fd_count\|## fd_count' -A1 "$OUT"/espx-*.txt 2>/dev/null | paste - - | head -10 || true
		echo '```'
	fi
	echo ""
	echo "## Interpretation hints"
	echo ""
	echo "1. **Redis Lua p99 > 15ms** — kernel TCP/epoll or Redis single-threaded shard saturation; check \`ad_redis_ops_total\` per shard."
	echo "2. **clickhouse p99 >> postgres** — async_insert batching or LSM merge pressure; check CH \`system.parts\` and processor \`ad_db_write_errors{type=clickhouse}\`."
	echo "3. **pgx p99 spikes** — Postgres WAL/fsync or pool exhaustion (\`DB_PROCESSOR_MAX_CONNS\`); strace shows \`write\`/\`fsync\`/\`epoll_wait\` dominance."
	echo "4. **gnet connections near ulimit** — raise \`worker_rlimit_nofile\` / container ulimits; k6 keep-alive reduces FD churn."
	echo "5. **fraud_stream_drop > 0** — fraud ring (4096) overflow; hot path lossy by design under dirty traffic."
	echo "6. **worker_pool_reject** — pinned worker queue full; ingestion exceeds parse+filter capacity."
	if [[ -f "$OUT/k6-status-histogram.json" ]]; then
		echo ""
		echo "## k6 HTTP status histogram"
		echo ""
		python3 - "$OUT/k6-status-histogram.json" <<'PY'
import json, sys
path = sys.argv[1]
with open(path) as f:
    data = json.load(f)
by = data.get("by_status", {})
total = sum(by.values()) or 1
print("| status | count | % |")
print("|--------|-------|---|")
for status, count in sorted(by.items(), key=lambda kv: -kv[1]):
    pct = 100.0 * count / total
    mark = " **5xx**" if status.isdigit() and int(status) >= 500 else ""
    print(f"| {status} | {count} | {pct:.1f}% |{mark}")
print("")
print("Top status×error buckets:")
for row in data.get("histogram", [])[:12]:
    print(f"- status={row['status']} error={row['error']}: {row['count']}")
five = sum(v for k, v in by.items() if k.isdigit() and int(k) >= 500)
print(f"\n_k6 5xx total: {five} ({100*five/total:.2f}% of classified responses)_")
PY
	fi
	if [[ -f "$OUT/bpf-report.md" ]]; then
		echo ""
		echo "## BPF probe (dev)"
		echo ""
		echo "Kernel/session detail: [bpf-report.md](bpf-report.md)."
		loadgen_line="$(grep -m1 'loadgen share of tracked on-CPU' "$OUT/bpf-report.md" 2>/dev/null || true)"
		if [[ -z "$loadgen_line" ]]; then
			loadgen_line="$(grep -m1 'k6 share of tracked on-CPU' "$OUT/bpf-report.md" 2>/dev/null || true)"
		fi
		if [[ -n "$loadgen_line" ]]; then
			echo "- $loadgen_line"
		fi
	fi
} | tee "$REPORT"

log "wrote $REPORT"
log "Grafana dashboard: http://127.0.0.1:3100/d/espx-main/espx-operations (or browse Dashboards tagged load-test)"
