#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

NGINX_IMAGE="${NGINX_IMAGE:-openresty/openresty:alpine}"
COMPOSE_FILE="$ROOT/deploy/compose/docker-compose.yaml"
ENV_FILE="$ROOT/.env.example"

run_compose_config() {
  if ! command -v docker > /dev/null 2>&1; then
    echo "lint_configs_gate: docker required for compose config check" >&2
    exit 1
  fi
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "lint_configs_gate: missing $ENV_FILE" >&2
    exit 1
  fi
  echo "lint_configs_gate: docker compose config (--env-file .env.example)..."
  local out
  out="$(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" config 2>&1)"
  local warnings
  warnings="$(printf '%s\n' "$out" | rg -c 'level=warning' || true)"
  if [[ "$warnings" -gt 0 ]]; then
    printf '%s\n' "$out" | rg 'level=warning' | head -20 >&2
    echo "lint_configs_gate: compose emitted $warnings warning(s); set defaults in .env.example or compose" >&2
    exit 1
  fi
}

run_nginx_config() {
  if ! command -v docker > /dev/null 2>&1; then
    echo "lint_configs_gate: docker required for nginx -t" >&2
    exit 1
  fi
  echo "lint_configs_gate: nginx -t (edge conf, no deprecation warnings)..."
  local cert_dir
  cert_dir="$(mktemp -d)"
  trap 'rm -rf "$cert_dir"' EXIT
  openssl req -x509 -nodes -newkey rsa:2048 \
    -keyout "$cert_dir/edge-dev.key" \
    -out "$cert_dir/edge-dev.crt" \
    -days 1 -subj '/CN=localhost' 2> /dev/null
  local out
  out="$(docker run --rm \
    -v "$ROOT/deploy/nginx:/etc/nginx:ro" \
    -v "$cert_dir:/etc/nginx/certs:ro" \
    "$NGINX_IMAGE" \
    /usr/local/openresty/nginx/sbin/nginx -t -c /etc/nginx/nginx.conf 2>&1)"
  printf '%s\n' "$out"
  if printf '%s\n' "$out" | grep -qE '\[warn\]'; then
    echo "lint_configs_gate: nginx -t reported warnings" >&2
    exit 1
  fi
  if ! printf '%s\n' "$out" | grep -q 'test is successful'; then
    echo "lint_configs_gate: nginx -t failed" >&2
    exit 1
  fi
  trap - EXIT
  rm -rf "$cert_dir"
}

run_compose_config
run_nginx_config
bash "$SCRIPTS/ci/admin/openapi.sh"
echo "lint_configs_gate: OK"
