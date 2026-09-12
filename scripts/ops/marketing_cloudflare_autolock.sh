#!/usr/bin/env bash
# Role: When marketing domain DNS is proxied via Cloudflare, optionally lock 80/443 to CF ranges.
# Env:
#   MARKETING_DOMAIN (default bidshard.com)
#   MARKETING_ORIGIN_IPV4 (default: curl -4 ifconfig.me on host)
#   MARKETING_CF_AUTOLOCK=1 required (off by default; shared ingress on same IP)
# Verify:
#   MARKETING_CF_AUTOLOCK=1 MARKETING_CF_DRY_RUN=1 bash scripts/ops/marketing_cloudflare_autolock.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)/.."

log() { printf 'marketing-cloudflare-autolock: %s\n' "$*"; }

AUTOLOCK="${MARKETING_CF_AUTOLOCK:-0}"
if [[ "$AUTOLOCK" != "1" ]]; then
  exit 0
fi

MARKETING_DOMAIN="${MARKETING_DOMAIN:-bidshard.com}"
ORIGIN_IP="${MARKETING_ORIGIN_IPV4:-$(curl -4 -fsSL --max-time 10 ifconfig.me 2> /dev/null || true)}"
if [[ -z "$ORIGIN_IP" ]]; then
  log "skip: origin IPv4 unknown"
  exit 0
fi

resolve_v4() {
  dig +short A "$1" 2> /dev/null | sed '/\./!d' | head -n 1
}

resolved="$(resolve_v4 "$MARKETING_DOMAIN")"
if [[ -z "$resolved" ]]; then
  log "skip: ${MARKETING_DOMAIN} has no A record yet"
  exit 0
fi

if [[ "$resolved" == "$ORIGIN_IP" ]]; then
  log "skip: ${MARKETING_DOMAIN} still points at origin ${ORIGIN_IP}"
  exit 0
fi

log "${MARKETING_DOMAIN} -> ${resolved} (origin ${ORIGIN_IP}); applying Cloudflare HTTP lock"
MARKETING_CF_LOCK_HTTP=1 exec bash "$SCRIPT_DIR/marketing_cloudflare_firewall.sh"
