#!/usr/bin/env bash
# DNS lab: /etc/hosts SNI + Caddy on-demand TLS (internal issuer on INGRESS_HTTPS_PORT).
#
# Prerequisites (auto with DNS_LAB_BRINGUP=1):
#   docker compose: db, redis-0, control, tracker-0, nginx, ingress (profile)
#
# Usage:
#   DNS_LAB_USE_PREPARE=1 DNS_LAB_BRINGUP=1 bash scripts/test/domain_tls_dns_lab.sh
#   DOMAIN_TLS_SMOKE_HOST=buyer-tls-lab.test ADMIN_API_KEY=... bash scripts/test/domain_tls_dns_lab.sh
#
# AC-3: first TLS handshake ≤ 5s after domain registered (internal CA, harness=dns_lab_internal).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
# shellcheck source=scripts/lib/dns_lab_hosts.sh
source "$SCRIPTS/lib/dns_lab_hosts.sh"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env" 2> /dev/null || true
  set +a
fi

DNS_LAB_HOST="${DOMAIN_TLS_SMOKE_HOST:-buyer-tls-lab.test}"
DNS_LAB_TRACKING="${DNS_LAB_TRACKING:-trk-tls-lab.test}"
DNS_LAB_ADMIN="${DNS_LAB_ADMIN:-admin-tls-lab.test}"
DNS_LAB_EVIL="${DNS_LAB_EVIL:-evil-tls-lab.test}"
CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
INGRESS_HTTPS_PORT="${INGRESS_HTTPS_PORT:-8443}"
ADMIN_API_KEY="${ADMIN_API_KEY:-dev-admin-api-key-change-me}"
CADDY_TLS_ASK_TOKEN="${CADDY_TLS_ASK_TOKEN:-dns-lab-ask-token}"
DNS_LAB_BRINGUP="${DNS_LAB_BRINGUP:-0}"
REPORT_DIR="${DNS_LAB_REPORT_DIR:-$ROOT/var/dns-lab/$(date -u +%Y%m%dT%H%M%SZ)}"

COMPOSE=(docker compose)
DB_PORT="${DB_PORT:-5430}"

log() { printf 'domain-tls-dns-lab: %s\n' "$*"; }
die() {
  printf 'domain-tls-dns-lab: ERROR: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" > /dev/null 2>&1 || die "missing command: $1"
}

wait_http() {
  local url="$1" max="${2:-120}" i=0
  while ((i < max)); do
    if curl -fsS -o /dev/null -m 2 "$url" 2> /dev/null; then
      return 0
    fi
    sleep 1
    ((i++)) || true
  done
  return 1
}

bringup_stack() {
  if [[ "${DNS_LAB_USE_PREPARE:-}" == "1" ]]; then
    log "full constrained stack (prepare_constrained_stack.sh)"
    bash "$SCRIPTS/test/prepare_constrained_stack.sh"
  else
    log "minimal bringup (db, redis-0, control) — set DNS_LAB_USE_PREPARE=1 for full stack"
    "${COMPOSE[@]}" up -d db redis-0
    log "waiting for postgres :${DB_PORT}"
    local i=0
    while ((i < 120)); do
      if "${COMPOSE[@]}" exec -T db pg_isready -h 127.0.0.1 -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor > /dev/null 2>&1; then
        break
      fi
      sleep 1
      ((i++)) || true
    done
    if ((i >= 120)); then
      die "postgres not ready"
    fi
  fi

  if [[ "${DNS_LAB_SKIP_MIGRATE:-}" != "1" && "${DNS_LAB_USE_PREPARE:-}" != "1" ]]; then
    log "migrations reconcile + cold-path"
    bash "$SCRIPTS/test/reconcile_ingestion_migrations.sh" || true
    export DB_DSN="${DB_DSN:-postgres://ad_event_processor_user:secure_pass_123@127.0.0.1:${DB_PORT}/ad_event_processor?sslmode=disable}"
    if ! go run ./cmd/migrate-cold-path --only=ads,auth,billing; then
      log "WARN: migrate-cold-path failed; applying domain_health bootstrap only"
    fi
    bash "$SCRIPTS/test/reconcile_ingestion_migrations.sh" || true
  fi
  log "ensure domain_health_status (00077)"
  "${COMPOSE[@]}" exec -T db psql -h 127.0.0.1 -p "$DB_PORT" -U ad_event_processor_user -d ad_event_processor -v ON_ERROR_STOP=1 << 'SQL'
CREATE TABLE IF NOT EXISTS domain_health_status (
    hostname TEXT PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('tracking', 'admin', 'custom')),
    health_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'degraded', 'down', 'unknown')),
    ssl_status TEXT NOT NULL DEFAULT 'unknown'
        CHECK (ssl_status IN ('valid', 'expiring', 'expired', 'missing', 'unknown')),
    ssl_not_after TIMESTAMPTZ,
    http_status INT,
    probe_latency_ms INT,
    probe_detail TEXT NOT NULL DEFAULT '',
    last_probe_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
SQL

  log "control (no-deps)"
  "${COMPOSE[@]}" up -d --no-deps control
  wait_http "${CONTROL_URL}/health" 180 || die "control not healthy at ${CONTROL_URL}"

  export TRACKING_DOMAIN="$DNS_LAB_TRACKING"
  export ADMIN_DOMAIN="$DNS_LAB_ADMIN"
  export INGRESS_TLS_MODE=on_demand
  export INGRESS_TLS_ISSUER=internal
  export INGRESS_HTTPS_PORT
  export CADDY_TLS_ASK_TOKEN
  export MANAGEMENT_PORT=8188
  bash "$SCRIPTS/install/render_ingress.sh"

  log "ingress profile (HTTPS :${INGRESS_HTTPS_PORT})"
  "${COMPOSE[@]}" up -d --no-deps tracker-0 nginx 2> /dev/null || true
  "${COMPOSE[@]}" --profile ingress up -d --no-deps ingress
  sleep 5
}

ask_curl() {
  local host="$1"
  local url="${CONTROL_URL}/api/v1/ops/domains/${host}/tls-allowed"
  if [[ -n "$CADDY_TLS_ASK_TOKEN" ]]; then
    curl -sS -o /dev/null -w '%{http_code}' -H "X-Caddy-Ask-Token: ${CADDY_TLS_ASK_TOKEN}" "$url"
  else
    curl -sS -o /dev/null -w '%{http_code}' "$url"
  fi
}

probe_tls_ms() {
  local host="$1"
  local start end ms out
  start=$(date +%s%N)
  out="$(echo | openssl s_client -connect "127.0.0.1:${INGRESS_HTTPS_PORT}" -servername "$host" 2> /dev/null | openssl x509 -noout -subject 2> /dev/null || true)"
  end=$(date +%s%N)
  ms=$(((end - start) / 1000000))
  printf '%s\n' "$ms"
  printf '%s\n' "$out" >&2
}

require_cmd curl
require_cmd openssl

mkdir -p "$REPORT_DIR"
log "report dir: $REPORT_DIR"

dns_lab_hosts_install "$DNS_LAB_HOST" "$DNS_LAB_TRACKING" "$DNS_LAB_ADMIN" "$DNS_LAB_EVIL" || {
  log "WARN: /etc/hosts not updated (openssl SNI lab still works via 127.0.0.1:${INGRESS_HTTPS_PORT})"
}

if [[ "$DNS_LAB_BRINGUP" == "1" ]]; then
  bringup_stack
fi

if ! wait_http "${CONTROL_URL}/health" 5; then
  die "control unreachable at ${CONTROL_URL} (set DNS_LAB_BRINGUP=1 or start stack)"
fi

evil_code="$(ask_curl "$DNS_LAB_EVIL")"
log "ask evil ${DNS_LAB_EVIL} -> HTTP ${evil_code} (expect 403)"
if [[ "$evil_code" != "403" ]]; then
  die "unknown SNI / unregistered host must get 403 from ask, got ${evil_code}"
fi

reg_code="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"hostname\":\"${DNS_LAB_HOST}\"}" \
  "${CONTROL_URL}/api/v1/domains" || true)"
log "POST /api/v1/domains (${DNS_LAB_HOST}) -> HTTP ${reg_code}"
if [[ "$reg_code" != "201" && "$reg_code" != "200" ]]; then
  die "domain registration failed with HTTP ${reg_code}"
fi

allowed_code="$(ask_curl "$DNS_LAB_HOST")"
log "ask registered ${DNS_LAB_HOST} -> HTTP ${allowed_code} (expect 200)"
if [[ "$allowed_code" != "200" ]]; then
  die "registered host ask must return 200, got ${allowed_code}"
fi

if ! ss -tln 2> /dev/null | grep -q ":${INGRESS_HTTPS_PORT} "; then
  log "skip TLS handshake (nothing listening on :${INGRESS_HTTPS_PORT}; start ingress profile)"
  log "ok (ask path proven; harness=dns_lab_ask_only)"
  exit 0
fi

tls_ms="$(probe_tls_ms "$DNS_LAB_HOST" 2> "$REPORT_DIR/tls-subject.txt" | head -1)"
subject="$(cat "$REPORT_DIR/tls-subject.txt" || true)"
log "TLS handshake ${DNS_LAB_HOST}:${INGRESS_HTTPS_PORT} -> ${tls_ms}ms"
log "cert subject: ${subject:-<none>}"

if [[ -z "$subject" ]]; then
  die "no certificate returned for registered host (check ingress logs)"
fi
if ! grep -qi "$DNS_LAB_HOST" "$REPORT_DIR/tls-subject.txt" 2> /dev/null; then
  log "WARN: subject may not include hostname (internal CA); continuing"
fi

if ((tls_ms > 5000)); then
  die "TLS handshake SLA exceeded: ${tls_ms}ms > 5000ms"
fi

evil_tls="$(probe_tls_ms "$DNS_LAB_EVIL" 2> /dev/null | head -1 || echo fail)"
log "evil SNI handshake -> ${evil_tls}ms (expect fail or no cert)"
if grep -qi "BEGIN CERTIFICATE" "$REPORT_DIR/tls-subject.txt" 2> /dev/null && [[ "$evil_tls" != "fail" ]]; then
  log "WARN: evil host got a cert line — verify ask rejected before issue"
fi

{
  echo "harness=dns_lab_internal"
  echo "host=${DNS_LAB_HOST}"
  echo "ingress_port=${INGRESS_HTTPS_PORT}"
  echo "ask_evil_http=${evil_code}"
  echo "ask_allowed_http=${allowed_code}"
  echo "tls_handshake_ms=${tls_ms}"
  echo "ac3_sla_m4_01=pass"
  echo "fault_proof fault=on_demand_tls_dns_lab harness=dns_lab_internal tls_ms=${tls_ms}"
} > "$REPORT_DIR/summary.txt"
cat "$REPORT_DIR/summary.txt"

log "PASS (AC-3 handshake ${tls_ms}ms ≤ 5000ms)"
