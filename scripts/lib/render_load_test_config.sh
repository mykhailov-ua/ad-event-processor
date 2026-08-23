#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/load_test_env.sh"

ROOT="${ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
CHECK_ONLY=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=1
fi

render_one() {
  local in_file=$1
  local out_file=$2
  if [[ ! -f "$in_file" ]]; then
    printf 'render-load-test: ERROR: missing template %s\n' "$in_file" >&2
    return 1
  fi
  if [[ "$CHECK_ONLY" == "1" ]]; then
    local tmp
    tmp="$(mktemp)"
    envsubst < "$in_file" > "$tmp"
    if ! diff -q "$tmp" "$out_file" > /dev/null 2>&1; then
      rm -f "$tmp"
      printf 'render-load-test: drift: %s (run: bash scripts/lib/render_load_test_config.sh)\n' "$out_file" >&2
      return 1
    fi
    rm -f "$tmp"
    return 0
  fi
  envsubst < "$in_file" > "$out_file"
}

load_test_bootstrap "$ROOT"

pairs=(
  "$ROOT/deploy/nginx/nginx.load-test.conf.in|$ROOT/deploy/nginx/nginx.load-test.conf"
  "$ROOT/deploy/nginx/lua/edge-tracker-peers.load-test.lua.in|$ROOT/deploy/nginx/lua/edge-tracker-peers.load-test.lua"
  "$ROOT/deploy/monitoring/prometheus.load-test.yaml.in|$ROOT/deploy/monitoring/prometheus.load-test.yaml"
  "$ROOT/deploy/monitoring/grafana-provisioning/datasources/datasource.load-test.yaml.in|$ROOT/deploy/monitoring/grafana-provisioning/datasources/datasource.load-test.yaml"
)

for pair in "${pairs[@]}"; do
  render_one "${pair%%|*}" "${pair##*|}"
done
