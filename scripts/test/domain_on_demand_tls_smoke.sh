#!/usr/bin/env bash
# On-demand TLS smoke: Caddy on-demand TLS ask endpoint + optional live SNI probe.
# Usage:
#   CONTROL_URL=http://127.0.0.1:8188 bash scripts/test/domain_on_demand_tls_smoke.sh
#   DOMAIN_TLS_SMOKE_HOST=buyer.example.com CONTROL_URL=... bash scripts/test/domain_on_demand_tls_smoke.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
ASK_HOST="${DOMAIN_TLS_SMOKE_HOST:-buyer-on-demand-smoke.invalid}"
CADDY_TLS_ASK_TOKEN="${CADDY_TLS_ASK_TOKEN:-}"

log() { printf 'domain-on-demand-tls-smoke: %s\n' "$*"; }
die() {
  printf 'domain-on-demand-tls-smoke: ERROR: %s\n' "$*" >&2
  exit 1
}

if ! command -v curl > /dev/null 2>&1; then
  log "skip (curl missing)"
  exit 0
fi

ask_url="${CONTROL_URL}/api/v1/ops/domains/${ASK_HOST}/tls-allowed"
curl_args=(-sS -o /dev/null -w '%{http_code}')
if [[ -n "$CADDY_TLS_ASK_TOKEN" ]]; then
  curl_args+=(-H "X-Caddy-Ask-Token: ${CADDY_TLS_ASK_TOKEN}")
fi

code="$(curl "${curl_args[@]}" "$ask_url" || true)"
log "GET ${ask_url} -> HTTP ${code} (unregistered host expect 403)"

if [[ "$code" == "000" ]]; then
  die "control plane unreachable at ${CONTROL_URL}"
fi

if [[ -z "${DOMAIN_TLS_SMOKE_HOST:-}" ]]; then
  log "skip (no DNS lab: set DOMAIN_TLS_SMOKE_HOST or run scripts/test/domain_tls_dns_lab.sh)"
  log "ok (ask endpoint reachable)"
  exit 0
fi

# Register custom domain then ask again (requires admin API).
if [[ -z "${ADMIN_API_KEY:-}" ]]; then
  log "skip (ADMIN_API_KEY unset; cannot register ${ASK_HOST})"
  exit 0
fi

reg_code="$(curl -sS -o /dev/null -w '%{http_code}' \
  -H "X-Admin-API-Key: ${ADMIN_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"hostname\":\"${ASK_HOST}\"}" \
  "${CONTROL_URL}/api/v1/domains" || true)"
log "POST /api/v1/domains -> HTTP ${reg_code}"

allowed_code="$(curl "${curl_args[@]}" "$ask_url" || true)"
log "GET ask (registered) -> HTTP ${allowed_code}"
if [[ "$allowed_code" != "200" ]]; then
  die "expected 200 after domain registration, got ${allowed_code}"
fi

if command -v openssl > /dev/null 2>&1; then
  log "openssl s_client -servername ${ASK_HOST} (first ClientHello; needs ingress on_demand_tls)"
  openssl s_client -connect "${ASK_HOST}:443" -servername "${ASK_HOST}" < /dev/null 2> /dev/null | head -5 || true
fi

log "ok"
