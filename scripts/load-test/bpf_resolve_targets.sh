#!/usr/bin/env bash
# Resolve docker container host PIDs for BPF target map.
# Usage: bpf_resolve_targets.sh <targets.json> [tracker,nginx,redis,processor]
set -euo pipefail

source "$(cd "$(dirname "$0")/../lib" && pwd)/paths.sh"

OUT_JSON="${1:?targets.json path required}"
WANT="${2:-tracker,nginx,redis,processor}"
STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
SAMPLE_RATE="${ESPX_BPF_SAMPLE_RATE:-1}"

mkdir -p "$(dirname "$OUT_JSON")"

role_tracker=1
role_nginx=2
role_redis=3
role_processor=5

entries=""
seen_pids=""

add_entry() {
	local pid=$1 role=$2 name=$3
	if [[ -z "$pid" || "$pid" == "0" ]]; then
		return 0
	fi
	if [[ " $seen_pids " == *" $pid "* ]]; then
		return 0
	fi
	seen_pids+=" $pid"
	entries+=$(printf '{"pid":%s,"role":%s,"name":"%s"},' "$pid" "$role" "$name")
}

resolve_container() {
	local pattern=$1
	local role=$2
	local cid name pid
	cid="$(docker ps --format '{{.ID}} {{.Names}}' | awk -v p="$pattern" '$2 ~ p {print $1; exit}')"
	if [[ -z "$cid" ]]; then
		return 0
	fi
	name="$(docker inspect -f '{{.Name}}' "$cid" | sed 's#^/##')"
	pid="$(docker inspect -f '{{.State.Pid}}' "$cid" 2>/dev/null || echo 0)"
	add_entry "$pid" "$role" "$name"
}

if [[ "$WANT" == *tracker* ]]; then
	for pat in 'espx-tracker-0' 'espx-tracker-1' 'tracker-0' 'tracker-1'; do
		resolve_container "$pat" "$role_tracker" || true
	done
fi
if [[ "$WANT" == *nginx* ]]; then
	resolve_container 'espx-nginx' "$role_nginx" || true
fi
if [[ "$WANT" == *redis* ]]; then
	for i in 0 1 2 3; do
		resolve_container "espx-redis-${i}" "$role_redis" || true
	done
fi
if [[ "$WANT" == *processor* ]]; then
	resolve_container 'espx-processor' "$role_processor" || true
fi

entries="${entries%,}"
if [[ -z "$entries" ]]; then
	printf 'bpf-resolve-targets: WARN no container PIDs found (stack running?)\n' >&2
fi

cat >"$OUT_JSON" <<EOF
{
  "started_at": "$STARTED_AT",
  "sample_rate": $SAMPLE_RATE,
  "targets": [${entries}]
}
EOF

printf 'bpf-resolve-targets: wrote %s\n' "$OUT_JSON"
