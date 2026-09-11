#!/usr/bin/env bash
# Role: Restrict HTTP/HTTPS on the host to Cloudflare edge IP ranges (origin hide).
# Context: Host-network Caddy ingress; locking 80/443 affects every vhost on this IP.
# Env:
#   MARKETING_CF_LOCK_HTTP=1 required to apply (fail-closed without explicit opt-in)
#   MARKETING_CF_DRY_RUN=1 print ufw commands only
# Verify:
#   MARKETING_CF_DRY_RUN=1 bash scripts/ops/marketing_cloudflare_firewall.sh
#   MARKETING_CF_LOCK_HTTP=1 bash scripts/ops/marketing_cloudflare_firewall.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

log() { printf 'marketing-cloudflare-firewall: %s\n' "$*"; }
die() {
  printf 'marketing-cloudflare-firewall: ERROR: %s\n' "$*" >&2
  exit 1
}

LOCK_HTTP="${MARKETING_CF_LOCK_HTTP:-0}"
DRY_RUN="${MARKETING_CF_DRY_RUN:-0}"
STATE_DIR="${MARKETING_CF_STATE_DIR:-/var/lib/aed/marketing-cloudflare}"
STATE_FILE="${STATE_DIR}/http_locked"

if [[ "$LOCK_HTTP" != "1" ]]; then
  die "refusing to lock 80/443 without MARKETING_CF_LOCK_HTTP=1 (shared ingress may break direct access)"
fi

command -v ufw >/dev/null 2>&1 || die "ufw not installed"
command -v curl >/dev/null 2>&1 || die "curl not installed"

run() {
  if [[ "$DRY_RUN" == "1" ]]; then
    printf '+ %s\n' "$*"
    return 0
  fi
  "$@"
}

fetch_cidrs() {
  curl -fsSL --max-time 20 https://www.cloudflare.com/ips-v4
  curl -fsSL --max-time 20 https://www.cloudflare.com/ips-v6
}

if [[ -f "$STATE_FILE" && "$DRY_RUN" != "1" ]]; then
  log "already locked (state ${STATE_FILE})"
  exit 0
fi

mapfile -t cidrs < <(fetch_cidrs | sed '/^[[:space:]]*$/d')
if [[ "${#cidrs[@]}" -lt 8 ]]; then
  die "cloudflare IP list looks truncated (${#cidrs[@]} rows)"
fi

log "allow Cloudflare CIDRs on 80/443 (${#cidrs[@]} ranges)"
for cidr in "${cidrs[@]}"; do
  run ufw allow from "$cidr" to any port 80 proto tcp comment 'aed-cf-http'
  run ufw allow from "$cidr" to any port 443 proto tcp comment 'aed-cf-https'
done

log "deny direct HTTP/HTTPS (after Cloudflare allows)"
run ufw deny 80/tcp comment 'aed-cf-lock-http'
run ufw deny 443/tcp comment 'aed-cf-lock-https'

if [[ "$DRY_RUN" != "1" ]]; then
  mkdir -p "$STATE_DIR"
  date -u +%Y-%m-%dT%H:%M:%SZ >"$STATE_FILE"
  ufw status numbered | head -n 40 || true
  log "locked; verify bidshard.com resolves to Cloudflare before dropping direct IP tests"
fi
