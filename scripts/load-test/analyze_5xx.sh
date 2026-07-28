#!/usr/bin/env bash
# Compare k6 client status histogram with tracker ad_http_requests_total (5xx reconciliation).
# Usage: analyze_5xx.sh [session_dir]
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"
cd "$ROOT"

log() { printf 'analyze-5xx: %s\n' "$*"; }
die() { printf 'analyze-5xx: ERROR: %s\n' "$*" >&2; exit 1; }

OUT="${1:-}"
if [[ -z "$OUT" ]]; then
	die "usage: analyze_5xx.sh <var/load-test/session>"
fi
PROM="${PROMETHEUS_URL:-http://127.0.0.1:9190}"
REPORT="$OUT/5xx-report.md"

prom_scalar() {
	local q=$1
	curl -sfG --max-time 10 --data-urlencode "query=${q}" "${PROM}/api/v1/query" 2>/dev/null \
		| python3 -c 'import json,sys; d=json.load(sys.stdin); r=d.get("data",{}).get("result",[]); print(r[0]["value"][1] if r else "0")' 2>/dev/null \
		|| echo "0"
}

{
	echo "# eSPX 5xx reconciliation report"
	echo ""
	echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "Session: \`$OUT\`"
	echo "Prometheus: $PROM"
	echo ""

	if [[ -f "$OUT/k6-status-histogram.json" ]]; then
		echo ""
	echo "## k6 client (status histogram)"
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
    flag = " **5xx**" if status.isdigit() and int(status) >= 500 else ""
    print(f"| {status} | {count} | {pct:.1f}% |{flag}")
five = sum(v for k, v in by.items() if k.isdigit() and int(k) >= 500)
zero = by.get("0", 0)
print(f"\n- k6 status≥500: **{five}** ({100*five/total:.2f}%)")
print(f"- k6 status=0 (transport): **{zero}** ({100*zero/total:.2f}%)")
print("\nTop buckets (status × error):")
for row in data.get("histogram", [])[:15]:
    print(f"- `{row['status']}` / `{row['error']}`: {row['count']}")
PY
	else
		echo "_No k6-status-histogram.json — re-run with updated k6_dirty_traffic.js._"
	fi

	echo ""
	echo "## Tracker Prometheus (\`ad_http_requests_total\`, increase 15m)"
	echo ""
	echo "| status | POST /track count |"
	echo "|--------|-------------------|"
	python3 <<PY
import json, urllib.parse, urllib.request
prom = "${PROM}"
q = 'sum(increase(ad_http_requests_total{job="tracker",path="/track"}[15m])) by (status)'
url = prom + '/api/v1/query?' + urllib.parse.urlencode({'query': q})
try:
    d = json.load(urllib.request.urlopen(url, timeout=10))
except Exception as e:
    print(f"_Prometheus query failed: {e}_")
    raise SystemExit(0)
rows = []
for r in d.get('data', {}).get('result', []):
    st = r['metric'].get('status', '?')
    v = float(r['value'][1])
    if v >= 0.5:
        rows.append((v, st))
rows.sort(reverse=True)
five = 0
for v, st in rows:
    mark = ''
    if st.isdigit() and int(st) >= 500:
        five += v
        mark = ' **5xx**'
    print(f'| {st} | {v:.0f} |{mark}')
print(f'\n- tracker instrumented 5xx: **{five:.0f}**')
PY

	echo ""
	echo "## Correlation"
	echo ""
	k6_five="$(python3 -c "
import json
try:
    d=json.load(open('$OUT/k6-status-histogram.json'))
    by=d.get('by_status',{})
    print(sum(v for k,v in by.items() if k.isdigit() and int(k)>=500))
except Exception:
    print('na')
" 2>/dev/null || echo na)"
	tracker_five="$(prom_scalar 'sum(increase(ad_http_requests_total{job="tracker",path="/track",status=~"5.."}[15m]))')"
	echo "- k6 status≥500: **${k6_five}**"
	echo "- tracker \`ad_http_requests_total\` 5xx: **${tracker_five}**"
	if [[ "$k6_five" != "na" && "$k6_five" != "0" && "$tracker_five" == "0" ]]; then
		echo ""
		echo "> **Gap:** k6 sees 5xx but tracker hot-path counters show none."
		echo "> Dirty mix sends ~15% of requests to \`EDGE_URL\` (nginx, default :8180) — edge circuit breaker returns **503** under pressure."
		echo "> Compare direct tracker buckets (400/404/413) with \`status=0\` transport errors on keep-alive."
	fi
	echo ""
	echo "## Related metrics"
	echo ""
	echo "- worker_pool_reject increase: $(prom_scalar 'sum(increase(ad_worker_pool_reject_total{job="tracker"}[15m]))')"
	echo "- parse payload_too_large increase: $(prom_scalar 'sum(increase(ad_http_parse_errors_total{job="tracker",error_type="payload_too_large"}[15m]))')"
	echo "- filter_engine failures increase: $(prom_scalar 'sum(increase(ad_filter_internal_errors_total{job="tracker",kind="filter_engine"}[15m]))')"
} | tee "$REPORT"

printf 'analyze-5xx: wrote %s\n' "$REPORT"
