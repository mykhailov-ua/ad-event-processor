#!/usr/bin/env bash
# Role: Audit edge TLS server persona (Mozilla Intermediate cipher profile, HTTP/2 on :443).
# Execution context: CI config-only; optional Docker live openssl probe.
# Env knobs: TLS_PERSONA_SNIPPET (default deploy/nginx/snippets/ssl_server.conf);
#   TLS_PERSONA_MANIFEST (default deploy/nginx/baselines/tls_server_persona.manifest);
#   TLS_PERSONA_LIVE (1 to run openssl probe); OPENRESTY_IMAGE.
# Verify:
#   bash scripts/test/edge/tls_server_persona_audit.sh --config
#   bash scripts/test/edge/tls_server_persona_audit.sh --holdout
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

SNIPPET="${TLS_PERSONA_SNIPPET:-$ROOT/deploy/nginx/snippets/ssl_server.conf}"
MANIFEST="${TLS_PERSONA_MANIFEST:-$ROOT/deploy/nginx/baselines/tls_server_persona.manifest}"
NGINX_CONF="${TLS_PERSONA_NGINX_CONF:-$ROOT/deploy/nginx/nginx.conf}"
IMAGE="${OPENRESTY_IMAGE:-openresty/openresty:alpine}"
ARTIFACT_DIR="${CI_ARTIFACT_DIR:-$ROOT/var/ci/tls_persona}"
MODE="${1:---config}"

log() { printf 'tls-persona-audit: %s\n' "$*"; }
die() {
  printf 'tls-persona-audit: ERROR: %s\n' "$*" >&2
  exit 1
}

read_directive() {
  local file="$1" key="$2"
  local line
  line="$(grep -E "^[[:space:]]*${key}[[:space:]]" "$file" | head -1 || true)"
  [[ -n "$line" ]] || return 1
  printf '%s' "$line" | sed -E "s/^[[:space:]]*${key}[[:space:]]+//;s/[[:space:]]*;[[:space:]]*$//;s/[[:space:]]+$//"
}

audit_snippet_against_manifest() {
  local snippet="$1" manifest="$2"
  [[ -r "$snippet" ]] || {
    printf 'tls-persona-audit: snippet not readable: %s\n' "$snippet" >&2
    return 1
  }
  [[ -r "$manifest" ]] || {
    printf 'tls-persona-audit: manifest not readable: %s\n' "$manifest" >&2
    return 1
  }

  local failures=0
  while IFS= read -r row || [[ -n "$row" ]]; do
    [[ -z "$row" || "$row" =~ ^# ]] && continue
    local key="${row%%=*}"
    local want="${row#*=}"
    local got=""
    case "$key" in
      nginx_tls_listener_http2)
        if grep -q 'listen 443 ssl' "$NGINX_CONF" && grep -q 'http2 on' "$NGINX_CONF"; then
          got="on"
        else
          got="off"
        fi
        ;;
      *)
        got="$(read_directive "$snippet" "$key" || true)"
        ;;
    esac
    if [[ "$got" != "$want" ]]; then
      printf 'tls-persona-audit: mismatch %s want=%q got=%q\n' "$key" "$want" "$got" >&2
      failures=$((failures + 1))
    fi
  done < "$manifest"

  if [[ "$failures" -gt 0 ]]; then
    return 1
  fi
  log "manifest ok ($snippet)"
  return 0
}

live_openssl_probe() {
  if ! command -v docker > /dev/null 2>&1 || ! docker info > /dev/null 2>&1; then
    log "skip live probe (docker unavailable)"
    return 0
  fi
  if ! command -v openssl > /dev/null 2>&1; then
    log "skip live probe (openssl missing)"
    return 0
  fi

  local name="tls-persona-live-$$"
  local listen="127.0.0.1:19443"
  local tmp
  tmp="$(mktemp -d)"
  cleanup() {
    docker rm -f "$name" > /dev/null 2>&1 || true
    rm -rf "$tmp"
  }
  trap cleanup EXIT

  cat > "$tmp/nginx.conf" << 'EOF'
events { worker_connections 64; }
http {
    server {
        listen 127.0.0.1:19443 ssl;
        http2 on;
        include /etc/nginx/snippets/ssl_server.conf;
        location / { return 200 "tls-persona\n"; }
    }
}
EOF

  docker run -d --name "$name" --network host \
    -v "$tmp/nginx.conf:/usr/local/openresty/nginx/conf/nginx.conf:ro" \
    -v "$ROOT/deploy/nginx/snippets:/etc/nginx/snippets:ro" \
    -v "$ROOT/deploy/nginx/certs:/etc/nginx/certs:ro" \
    "$IMAGE" > /dev/null

  local ok=0 i
  for i in $(seq 1 30); do
    if openssl s_client -connect "$listen" -servername localhost -tls1_2 < /dev/null 2> /dev/null | grep -q 'Cipher'; then
      ok=1
      break
    fi
    sleep 0.1
  done
  [[ "$ok" -eq 1 ]] || die "openssl probe to $listen failed"

  local fingerprint
  fingerprint="$(
    openssl s_client -connect "$listen" -servername localhost -tls1_2 < /dev/null 2> /dev/null \
      | awk '/^Protocol|^Cipher|^Server Temp Key/ {print}' \
      | sha256sum | awk '{print $1}'
  )"
  mkdir -p "$ARTIFACT_DIR"
  printf '%s\n' "$fingerprint" > "$ARTIFACT_DIR/openssl_tls12_fingerprint.sha256"
  log "live probe ok fingerprint=$fingerprint artifact=$ARTIFACT_DIR/openssl_tls12_fingerprint.sha256"
}

case "$MODE" in
  --holdout)
    if audit_snippet_against_manifest \
      "$ROOT/deploy/nginx/testfixtures/ssl_server_cipher_reorder.conf" \
      "$MANIFEST" 2> /dev/null; then
      die "holdout failed: cipher reorder fixture must not pass manifest audit"
    fi
    log "holdout ok (bad cipher order rejected)"
    exit 0
    ;;
  --config)
    audit_snippet_against_manifest "$SNIPPET" "$MANIFEST" || die "production ssl_server.conf failed manifest audit"
    if [[ "${TLS_PERSONA_LIVE:-0}" == "1" ]]; then
      live_openssl_probe
    fi
    log "ok"
    exit 0
    ;;
  --live)
    audit_snippet_against_manifest "$SNIPPET" "$MANIFEST" || die "production ssl_server.conf failed manifest audit"
    live_openssl_probe
    log "ok"
    exit 0
    ;;
  *)
    die "usage: $0 [--config|--holdout|--live]"
    ;;
esac
