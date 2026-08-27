#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/load_test_env.sh"

ROOT="${ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

export DOLLAR='$'

CHECK_ONLY=0
if [[ "${1:-}" == "--check" ]]; then
  CHECK_ONLY=1
fi

render_assert_no_unexpanded_env() {
  local file=$1
  local label=$2
  if render_has_unexpanded_env "$file"; then
    printf 'render-load-test: unexpanded env vars in %s\n' "$label" >&2
    render_print_unexpanded_env "$file"
    return 1
  fi
  return 0
}

render_has_unexpanded_env() {
  local file=$1
  if command -v rg > /dev/null 2>&1; then
    rg -q '\$\{[A-Z][A-Z0-9_]*\}' "$file"
    return $?
  fi
  grep -qE '\$\{[A-Z][A-Z0-9_]*\}' "$file"
}

render_print_unexpanded_env() {
  local file=$1
  if command -v rg > /dev/null 2>&1; then
    rg -n '\$\{[A-Z][A-Z0-9_]*\}' "$file" >&2 || true
    return 0
  fi
  grep -nE '\$\{[A-Z][A-Z0-9_]*\}' "$file" >&2 || true
}

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
    if ! render_assert_no_unexpanded_env "$tmp" "$in_file"; then
      rm -f "$tmp"
      return 1
    fi
    if [[ -f "$out_file" ]]; then
      if ! diff -q "$tmp" "$out_file" > /dev/null 2>&1; then
        rm -f "$tmp"
        printf 'render-load-test: drift: %s (run: make load-test-config)\n' "$out_file" >&2
        return 1
      fi
    fi
    rm -f "$tmp"
    return 0
  fi
  mkdir -p "$(dirname "$out_file")"
  envsubst < "$in_file" > "$out_file"
  if ! render_assert_no_unexpanded_env "$out_file" "$in_file"; then
    return 1
  fi
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

if [[ "$CHECK_ONLY" == "1" ]]; then
  printf 'render-load-test: ok (templates)\n'
else
  printf 'render-load-test: wrote %d files (sources: *.load-test.*.in)\n' "${#pairs[@]}"
fi
