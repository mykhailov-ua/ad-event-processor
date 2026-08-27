#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

OUT_JSON="${1:?targets.json path required}"
WANT="${2:-tracker,nginx,redis,processor}"
STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
SAMPLE_RATE="${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-1}"
NATIVE="${AD_EVENT_PROCESSOR_BPF_NATIVE:-0}"

mkdir -p "$(dirname "$OUT_JSON")"

role_tracker=1
role_nginx=2
role_redis=3
role_loadgen=4
role_processor=5

entries=""
seen_pids=""

cgroup_id_for_pid() {
  local pid=$1
  local rel path
  rel="$(awk -F: '$1=="0" {gsub(/^\/+/, "", $2); print $2}' "/proc/${pid}/cgroup" 2> /dev/null || true)"
  if [[ -z "$rel" ]]; then
    printf '0'
    return
  fi
  path="/sys/fs/cgroup/${rel}"
  if [[ ! -d "$path" ]]; then
    path="/sys/fs/cgroup${rel}"
  fi
  stat -c '%i' "$path" 2> /dev/null || printf '0'
}

add_entry() {
  local pid=$1 role=$2 name=$3
  local cgroup_id=0
  if [[ -z "$pid" || "$pid" == "0" ]]; then
    return 0
  fi
  if [[ " $seen_pids " == *" $pid "* ]]; then
    return 0
  fi
  seen_pids+=" $pid"
  cgroup_id="$(cgroup_id_for_pid "$pid")"
  entries+=$(printf '{"pid":%s,"cgroup_id":%s,"role":%s,"name":"%s"},' "$pid" "$cgroup_id" "$role" "$name")
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
  pid="$(docker inspect -f '{{.State.Pid}}' "$cid" 2> /dev/null || echo 0)"
  add_entry "$pid" "$role" "$name"
}

resolve_native_pattern() {
  local pattern=$1
  local role=$2
  local label=$3
  local pid
  pid="$(pgrep -n -f "$pattern" 2> /dev/null || true)"
  if [[ -z "$pid" ]]; then
    return 0
  fi
  add_entry "$pid" "$role" "$label"
}

resolve_native_defaults() {
  if [[ "$WANT" == *tracker* ]]; then
    resolve_native_pattern '[c]md/tracker' "$role_tracker" 'native-tracker'
    resolve_native_pattern '/bin/tracker' "$role_tracker" 'native-tracker'
  fi
  if [[ "$WANT" == *nginx* ]]; then
    resolve_native_pattern '[n]ginx: master' "$role_nginx" 'native-nginx'
    resolve_native_pattern 'openresty' "$role_nginx" 'native-nginx'
  fi
  if [[ "$WANT" == *redis* ]]; then
    resolve_native_pattern '[r]edis-server' "$role_redis" 'native-redis'
  fi
  if [[ "$WANT" == *processor* ]]; then
    resolve_native_pattern '[c]md/processor' "$role_processor" 'native-processor'
    resolve_native_pattern '/bin/processor' "$role_processor" 'native-processor'
  fi
}

resolve_native_custom() {
  local spec item role pattern
  spec="${AD_EVENT_PROCESSOR_BPF_NATIVE_PATTERNS:-}"
  [[ -z "$spec" ]] && return 0
  IFS=',' read -r -a items <<< "$spec"
  for item in "${items[@]}"; do
    role="${item%%:*}"
    pattern="${item#*:}"
    [[ -z "$role" || -z "$pattern" || "$pattern" == "$item" ]] && continue
    resolve_native_pattern "$pattern" "$role" "native-${pattern}"
  done
}

parse_extra_targets() {
  local spec item pid role name
  spec="${AD_EVENT_PROCESSOR_BPF_EXTRA_TARGETS:-}"
  [[ -z "$spec" ]] && return 0
  IFS=',' read -r -a items <<< "$spec"
  for item in "${items[@]}"; do
    pid="${item%%:*}"
    rest="${item#*:}"
    role="${rest%%:*}"
    name="${rest#*:}"
    [[ -z "$pid" || -z "$role" || "$name" == "$rest" ]] && continue
    add_entry "$pid" "$role" "$name"
  done
}

if [[ "$WANT" == *tracker* ]]; then
  for i in 0 1 2 3; do
    resolve_container "ad-event-processor-tracker-${i}" "$role_tracker" || true
    resolve_container "tracker-${i}" "$role_tracker" || true
  done
fi
if [[ "$WANT" == *nginx* ]]; then
  resolve_container 'ad-event-processor-nginx' "$role_nginx" || true
fi
if [[ "$WANT" == *redis* ]]; then
  for i in 0 1 2 3; do
    resolve_container "ad-event-processor-redis-${i}" "$role_redis" || true
  done
fi
if [[ "$WANT" == *processor* ]]; then
  resolve_container 'ad-event-processor-processor' "$role_processor" || true
fi

if [[ "$NATIVE" == "1" ]]; then
  resolve_native_defaults
  resolve_native_custom
fi

parse_extra_targets

entries="${entries%,}"
if [[ -z "$entries" ]]; then
  printf 'bpf-resolve-targets: WARN no PIDs found (stack running? set AD_EVENT_PROCESSOR_BPF_NATIVE=1 or AD_EVENT_PROCESSOR_BPF_EXTRA_TARGETS)\n' >&2
fi

cat > "$OUT_JSON" << EOF
{
  "started_at": "$STARTED_AT",
  "sample_rate": $SAMPLE_RATE,
  "roles_wanted": "$WANT",
  "targets": [${entries}]
}
EOF

printf 'bpf-resolve-targets: wrote %s\n' "$OUT_JSON"
