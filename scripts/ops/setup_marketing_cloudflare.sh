#!/usr/bin/env bash
# Role: Prepare appliance ingress for Cloudflare-proxied marketing (operator DNS card + optional firewall helpers).
# Does not lock 80/443 by default (shared ingress with tracker/admin hostnames on same IP).
# Env:
#   AED_TARGET, AED_SSH_PORT, AED_INSTALL_ROOT, MARKETING_DOMAIN (see deploy_marketing.sh)
#   MARKETING_CF_AUTOLOCK=1 to install cron that locks after DNS leaves origin IP
# Verify:
#   bash scripts/ops/setup_marketing_cloudflare.sh --check
#   MARKETING_DOMAIN=bidshard.com bash scripts/ops/setup_marketing_cloudflare.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

TARGET="${AED_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${AED_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"
MARKETING_DOMAIN="${MARKETING_DOMAIN:-bidshard.com}"
CHECK_ONLY=0
AUTOLOCK="${MARKETING_CF_AUTOLOCK:-0}"

log() { printf 'setup-marketing-cloudflare: %s\n' "$*"; }
die() {
  printf 'setup-marketing-cloudflare: ERROR: %s\n' "$*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check | -n)
      CHECK_ONLY=1
      shift
      ;;
    -h | --help)
      sed -n '1,18p' "$0" | tail -n +2
      exit 0
      ;;
    *)
      die "unknown arg: $1"
      ;;
  esac
done

ssh_base=(-o BatchMode=yes -o ConnectTimeout=30 -p "$SSH_PORT")
rsync_base=(-az -e "ssh -p ${SSH_PORT} -o BatchMode=yes -o ConnectTimeout=30")

remote() {
  ssh "${ssh_base[@]}" "$TARGET" "$@"
}

log "target ${TARGET} port ${SSH_PORT} domain ${MARKETING_DOMAIN}"
remote "echo ok && hostname"

if [[ "$CHECK_ONLY" == "1" ]]; then
  exit 0
fi

ORIGIN_IP="$(remote "curl -4 -fsSL --max-time 10 ifconfig.me 2>/dev/null || hostname -I | awk '{print \$1}'")"
[[ -n "$ORIGIN_IP" ]] || die "could not detect origin IPv4"

log "render marketing Caddy snippet"
MARKETING_DOMAIN="$MARKETING_DOMAIN" bash "$ROOT/scripts/ops/render_marketing_caddy.sh"

log "sync cloudflare helper scripts"
remote "mkdir -p '${INSTALL_ROOT}/scripts/ops' '${INSTALL_ROOT}/deploy/marketing'"
rsync "${rsync_base[@]}" \
  "$ROOT/scripts/ops/marketing_cloudflare_firewall.sh" \
  "$ROOT/scripts/ops/marketing_cloudflare_autolock.sh" \
  "${TARGET}:${INSTALL_ROOT}/scripts/ops/"
remote "chmod +x '${INSTALL_ROOT}/scripts/ops/marketing_cloudflare_firewall.sh' '${INSTALL_ROOT}/scripts/ops/marketing_cloudflare_autolock.sh'"

OPERATOR_CARD="${ROOT}/deploy/marketing/cloudflare.operator.txt"
cat >"$OPERATOR_CARD" <<EOF
Cloudflare setup for ${MARKETING_DOMAIN} (server-side ingress is ready)

Origin IPv4 (A record content): ${ORIGIN_IP}

1) Cloudflare dashboard -> Add site -> ${MARKETING_DOMAIN} -> Free plan
2) At registrar (Namecheap): set nameservers to Cloudflare (Custom DNS)
3) Cloudflare DNS -> Records (orange cloud = Proxied):

   Type   Name   Content            Proxy
   A      @      ${ORIGIN_IP}       Proxied
   A      www    ${ORIGIN_IP}       Proxied

4) SSL/TLS -> Overview -> Full (strict)

5) Security -> Settings: Bot Fight Mode ON; Security Level Medium
   (Under Attack Mode only while actively DDoSed)

Optional origin hide (ONLY after every hostname on this IP uses Cloudflare):
   ssh to server, then:
   MARKETING_CF_LOCK_HTTP=1 bash ${INSTALL_ROOT}/scripts/ops/marketing_cloudflare_firewall.sh

Shared ingress note: this host also serves other vhosts on :443.
Do not lock 80/443 until tracker/admin hostnames are behind Cloudflare too.

Verify:
   dig +short A ${MARKETING_DOMAIN}     # Cloudflare anycast IP, not ${ORIGIN_IP}
   curl -sI https://${MARKETING_DOMAIN} | grep -i server   # cloudflare
EOF

rsync "${rsync_base[@]}" \
  "$OPERATOR_CARD" \
  "${TARGET}:${INSTALL_ROOT}/deploy/marketing/cloudflare.operator.txt"

log "upload marketing.caddy and reload ingress"
scp -P "$SSH_PORT" -o BatchMode=yes -o ConnectTimeout=30 \
  "$ROOT/deploy/ingress/caddy/generated/marketing.caddy" \
  "${TARGET}:${INSTALL_ROOT}/deploy/ingress/caddy/generated/marketing.caddy"

remote "set -euo pipefail
INSTALL_ROOT='${INSTALL_ROOT}'
CADDYFILE=\"\${INSTALL_ROOT}/deploy/ingress/caddy/generated/Caddyfile\"
CF_MARKER='aed-cloudflare-trusted-proxies'
if [[ -f \"\${CADDYFILE}\" ]] && grep -q \"\${CF_MARKER}\" \"\${CADDYFILE}\"; then
  python3 - \"\${CADDYFILE}\" <<'PY'
import pathlib
import re
import sys

path = pathlib.Path(sys.argv[1])
text = path.read_text(encoding='utf-8')
text = re.sub(
    r'\n\tservers \{\n\t\ttrusted_proxies cloudflare # aed-cloudflare-trusted-proxies\n\t\}',
    '',
    text,
    count=1,
)
path.write_text(text, encoding='utf-8')
PY
fi
cd \"\${INSTALL_ROOT}\"
docker exec ad-event-processor-ingress-1 caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
docker exec ad-event-processor-ingress-1 caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile
mkdir -p /var/lib/aed/marketing-cloudflare
"

if [[ "$AUTOLOCK" == "1" ]]; then
  log "install autolock cron (MARKETING_CF_AUTOLOCK=1)"
  remote "set -euo pipefail
CRON_LINE='*/10 * * * * MARKETING_DOMAIN=${MARKETING_DOMAIN} MARKETING_ORIGIN_IPV4=${ORIGIN_IP} MARKETING_CF_AUTOLOCK=1 ${INSTALL_ROOT}/scripts/ops/marketing_cloudflare_autolock.sh >> /var/log/aed-marketing-cf-autolock.log 2>&1'
mkdir -p /var/log
touch /var/log/aed-marketing-cf-autolock.log
( crontab -l 2>/dev/null | grep -v 'marketing_cloudflare_autolock.sh' || true; echo \"\${CRON_LINE}\" ) | crontab -
"
else
  log "autolock cron skipped (set MARKETING_CF_AUTOLOCK=1 to enable)"
  remote "crontab -l 2>/dev/null | grep -v 'marketing_cloudflare_autolock.sh' | crontab - 2>/dev/null || true"
fi

log "done — operator card: ${INSTALL_ROOT}/deploy/marketing/cloudflare.operator.txt"
log "Cloudflare DNS: A @ and www -> ${ORIGIN_IP} (Proxied); SSL Full (strict)"
cat "$OPERATOR_CARD"
