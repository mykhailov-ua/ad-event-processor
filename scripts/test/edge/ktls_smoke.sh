#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a

  source "$ROOT/.env"
  set +a
fi

EDGE_KTLS="${EDGE_KTLS:-1}"
IMAGE="${OPENRESTY_IMAGE:-openresty/openresty:alpine}"

log() { printf 'nginx-ktls-smoke: %s\n' "$*"; }
die() {
  printf 'nginx-ktls-smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

if [[ "$EDGE_KTLS" == "0" && "${NGINX_KTLS_SMOKE_FORCE:-0}" != "1" ]]; then
  log "skip (EDGE_KTLS=0)"
  exit 0
fi

if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
  log "skip (docker unavailable)"
  exit 0
fi

log "nginx -t (full edge conf + kTLS snippet)"
docker run --rm \
  -v "$ROOT/deploy/nginx/nginx.conf:/usr/local/openresty/nginx/conf/nginx.conf:ro" \
  -v "$ROOT/deploy/nginx/lua:/etc/nginx/lua:ro" \
  -v "$ROOT/deploy/nginx/snippets:/etc/nginx/snippets:ro" \
  -v "$ROOT/deploy/nginx/certs:/etc/nginx/certs:ro" \
  "$IMAGE" nginx -t

if ! grep -q 'ssl_conf_command Options KTLS' "$ROOT/deploy/nginx/snippets/ssl_server.conf"; then
  die "ssl_server.conf missing ssl_conf_command Options KTLS"
fi
if ! grep -q 'snippets/ssl_server.conf' "$ROOT/deploy/nginx/nginx.conf"; then
  die "nginx.conf does not include ssl_server.conf"
fi

if [[ ! -r /proc/net/tls_stat ]]; then
  log "skip (kernel tls module not loaded - config ok; live offload not proven)"
  printf 'fault_proof fault=nginx_ktls status=partial proof=nginx_t kernel_tls=absent harness=config_only\n'
  exit 0
fi

if ! command -v curl > /dev/null 2>&1; then
  log "skip (curl missing for live offload probe - config ok)"
  printf 'fault_proof fault=nginx_ktls status=partial proof=nginx_t kernel_tls=present live=skipped harness=config_only\n'
  exit 0
fi

read_tx() {
  awk '/^TlsTxSw[[:space:]]/{print $2}' /proc/net/tls_stat 2> /dev/null || echo 0
}

NAME="nginx-ktls-smoke-$$"
LISTEN="127.0.0.1:18443"
cleanup() { docker rm -f "$NAME" > /dev/null 2>&1 || true; }
trap cleanup EXIT

TMP="$(mktemp -d)"
cat > "$TMP/nginx.conf" << 'EOF'
events { worker_connections 64; }
http {
    sendfile on;
    server {
        listen 127.0.0.1:18443 ssl;
        include /etc/nginx/snippets/ssl_server.conf;
        location / { return 200 "ktls-smoke\n"; }
    }
}
EOF

BEFORE="$(read_tx)"
docker run -d --name "$NAME" --network host \
  -v "$TMP/nginx.conf:/usr/local/openresty/nginx/conf/nginx.conf:ro" \
  -v "$ROOT/deploy/nginx/snippets:/etc/nginx/snippets:ro" \
  -v "$ROOT/deploy/nginx/certs:/etc/nginx/certs:ro" \
  "$IMAGE" > /dev/null

ok=0
for _ in $(seq 1 20); do
  if curl -sk --http1.1 --max-time 1 "https://${LISTEN}/" | grep -q ktls-smoke; then
    ok=1
    break
  fi
  sleep 0.1
done
[[ "$ok" -eq 1 ]] || die "HTTPS probe to $LISTEN failed"

for _ in 1 2 3 4 5; do
  curl -sk --http1.1 --max-time 1 "https://${LISTEN}/" > /dev/null
done
AFTER="$(read_tx)"

if [[ "${AFTER:-0}" -le "${BEFORE:-0}" ]]; then
  die "TlsTxSw did not increase (before=$BEFORE after=$AFTER) - kTLS TX not used"
fi

log "ok  TlsTxSw $BEFORE -> $AFTER"
printf 'fault_proof fault=nginx_ktls status=ok proof=tls_stat_tx harness=kernel_tx before=%s after=%s baseline_ok=true\n' "$BEFORE" "$AFTER"
rm -rf "$TMP"
