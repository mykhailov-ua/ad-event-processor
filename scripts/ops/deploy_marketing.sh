#!/usr/bin/env bash
# Role: Deploy deploy/marketing static site to appliance and wire bidshard.com in Caddy ingress.
# Env:
#   AED_TARGET (default root@45.94.158.106)
#   AED_SSH_PORT (default 2222)
#   AED_INSTALL_ROOT (default /opt/platform/ad-event-processor)
#   MARKETING_DOMAIN (default bidshard.com)
#   CADDY_ACME_EMAIL (optional; used when patching global Caddy email)
# Verify:
#   bash scripts/ops/bundle_marketing_jetbrains_mono.sh
#   bash scripts/ops/deploy_marketing.sh --check
#   MARKETING_DOMAIN=bidshard.com bash scripts/ops/deploy_marketing.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

TARGET="${AED_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${AED_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"
MARKETING_DOMAIN="${MARKETING_DOMAIN:-bidshard.com}"
CHECK_ONLY=0
MARKER="# aed-marketing:${MARKETING_DOMAIN}"

log() { printf 'deploy-marketing: %s\n' "$*"; }
die() {
  printf 'deploy-marketing: ERROR: %s\n' "$*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check | -n)
      CHECK_ONLY=1
      shift
      ;;
    -h | --help)
      sed -n '1,16p' "$0" | tail -n +2
      exit 0
      ;;
    *)
      die "unknown arg: $1"
      ;;
  esac
done

ssh_base=(-o BatchMode=yes -o ConnectTimeout=30 -p "$SSH_PORT")
rsync_base=(-az --delete -e "ssh -p ${SSH_PORT} -o BatchMode=yes -o ConnectTimeout=30")

remote() {
  ssh "${ssh_base[@]}" "$TARGET" "$@"
}

log "target ${TARGET} port ${SSH_PORT} domain ${MARKETING_DOMAIN}"
remote "echo ok && hostname"

if [[ "$CHECK_ONLY" == "1" ]]; then
  log "ssh ok"
  exit 0
fi

log "render marketing Caddy snippet"
MARKETING_DOMAIN="$MARKETING_DOMAIN" bash "$ROOT/scripts/ops/render_marketing_caddy.sh"

log "render public offer (English)"
if [[ -f "$ROOT/deploy/vendor/offer_vendor.env" ]]; then
  # shellcheck disable=SC1091
  set -a
  source "$ROOT/deploy/vendor/offer_vendor.env"
  set +a
fi
python3 "$ROOT/scripts/ops/render_marketing_offer.py"

log "build obfuscated get.sh"
MARKETING_DOMAIN="$MARKETING_DOMAIN" bash "$ROOT/scripts/install/obfuscate_get.sh"

log "bundle JetBrains Mono webfont for marketing monospace"
bash "$ROOT/scripts/ops/bundle_marketing_jetbrains_mono.sh"

log "bundle marketing logo assets (product_avatar.svg)"
bash "$ROOT/scripts/ops/bundle_marketing_assets.sh"

log "sync marketing files"
remote "mkdir -p '${INSTALL_ROOT}/deploy/marketing' '${INSTALL_ROOT}/deploy/marketing/releases' '${INSTALL_ROOT}/deploy/ingress/caddy/generated'"
rsync "${rsync_base[@]}" \
  --exclude '.gitignore' \
  "$ROOT/deploy/marketing/" "${TARGET}:${INSTALL_ROOT}/deploy/marketing/"
if [[ -d "$ROOT/deploy/marketing/releases" ]] && ls "$ROOT/deploy/marketing/releases"/*.tar.gz > /dev/null 2>&1; then
  log "sync installer release tarballs"
  rsync "${rsync_base[@]}" \
    "$ROOT/deploy/marketing/releases/" "${TARGET}:${INSTALL_ROOT}/deploy/marketing/releases/"
fi

log "upload ingress overlay + marketing.caddy"
scp -P "$SSH_PORT" -o BatchMode=yes -o ConnectTimeout=30 \
  "$ROOT/deploy/compose/docker-compose.marketing-ingress.yaml" \
  "${TARGET}:${INSTALL_ROOT}/deploy/compose/docker-compose.marketing-ingress.yaml"
scp -P "$SSH_PORT" -o BatchMode=yes -o ConnectTimeout=30 \
  "$ROOT/deploy/ingress/caddy/generated/marketing.caddy" \
  "${TARGET}:${INSTALL_ROOT}/deploy/ingress/caddy/generated/marketing.caddy"

log "patch Caddyfile and reload ingress"
remote "set -euo pipefail
INSTALL_ROOT='${INSTALL_ROOT}'
MARKETING_DOMAIN='${MARKETING_DOMAIN}'
MARKER='${MARKER}'
CADDYFILE=\"\${INSTALL_ROOT}/deploy/ingress/caddy/generated/Caddyfile\"
IMPORT_LINE='import /etc/caddy/snippets/marketing.caddy'
COMPOSE_PROFILES=\"\${COMPOSE_PROFILES:-ingest_only}\"

if [[ ! -f \"\${CADDYFILE}\" ]]; then
  echo 'missing Caddyfile' >&2
  exit 1
fi

if ! grep -q \"\${MARKER}\" \"\${CADDYFILE}\"; then
  printf '\n%s\n%s\n' \"\${MARKER}\" \"\${IMPORT_LINE}\" >> \"\${CADDYFILE}\"
fi

cd \"\${INSTALL_ROOT}\"
set -a
source .env
set +a

IFS=',' read -r -a profile_args <<< \"\${COMPOSE_PROFILES}\"
compose_profiles=(--profile ingress)
for p in \"\${profile_args[@]}\"; do
  p=\"\${p// /}\"
  [[ -n \"\${p}\" ]] && compose_profiles+=(--profile \"\${p}\")
done

docker rm -f ad-event-processor-ingress-1 2>/dev/null || true
docker compose -p ad-event-processor \\
  -f deploy/compose/docker-compose.yaml \\
  -f deploy/compose/docker-compose.marketing-ingress.yaml \\
  \"\${compose_profiles[@]}\" \\
  up -d --no-deps ingress

sleep 2
docker exec ad-event-processor-ingress-1 caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
docker exec ad-event-processor-ingress-1 test -f /srv/marketing/index.html
docker exec ad-event-processor-ingress-1 caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile
"

log "done — set Namecheap DNS A @ and www -> $(remote 'curl -4 -s ifconfig.me || hostname -I | awk "{print \$1}"')"
log "after DNS propagates: https://${MARKETING_DOMAIN}/"
