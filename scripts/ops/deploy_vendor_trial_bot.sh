#!/usr/bin/env bash
# Role: Build and deploy vendor-trial-bot on vendor VPS.
# Env:
#   AED_TARGET (default root@45.94.158.106)
#   AED_SSH_PORT (default 2222)
#   AED_INSTALL_ROOT (default /opt/platform/ad-event-processor)
# Verify:
#   bash scripts/ops/deploy_vendor_trial_bot.sh --check
#   VENDOR_TRIAL_BOT_TOKEN=... bash scripts/ops/deploy_vendor_trial_bot.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

TARGET="${AED_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${AED_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"
ENV_FILE="/etc/ad-event-processor/vendor-trial-bot.env"
CHECK_ONLY=0

log() { printf 'deploy-vendor-trial-bot: %s\n' "$*"; }
die() {
  printf 'deploy-vendor-trial-bot: ERROR: %s\n' "$*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --check | -n)
      CHECK_ONLY=1
      shift
      ;;
    -h | --help)
      sed -n '1,14p' "$0" | tail -n +2
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

log "target ${TARGET} port ${SSH_PORT}"
remote "echo ok && hostname"

if [[ "$CHECK_ONLY" == "1" ]]; then
  log "ssh ok"
  exit 0
fi

TOKEN="${VENDOR_TRIAL_BOT_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(remote "if [[ -f '${ENV_FILE}' ]] && grep -q '^VENDOR_TRIAL_BOT_TOKEN=' '${ENV_FILE}'; then grep '^VENDOR_TRIAL_BOT_TOKEN=' '${ENV_FILE}' | head -n1 | cut -d= -f2-; fi" 2>/dev/null || true)"
fi
if [[ -z "$TOKEN" ]]; then
  die "VENDOR_TRIAL_BOT_TOKEN is required (set env or create ${ENV_FILE} on target)"
fi

log "build vendor-trial-bot"
mkdir -p "$ROOT/bin"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/vendor-trial-bot" ./cmd/vendor-trial-bot

log "sync binary + registry path"
remote "mkdir -p '${INSTALL_ROOT}/bin' '${INSTALL_ROOT}/deploy/vendor'"
rsync "${rsync_base[@]}" "$ROOT/bin/vendor-trial-bot" "${TARGET}:${INSTALL_ROOT}/bin/vendor-trial-bot"
if [[ -f "$ROOT/deploy/vendor/trial_registry.json" ]]; then
  rsync "${rsync_base[@]}" \
    "$ROOT/deploy/vendor/trial_registry.json" \
    "${TARGET}:${INSTALL_ROOT}/deploy/vendor/trial_registry.json"
fi

log "upload systemd unit"
scp -P "$SSH_PORT" -o BatchMode=yes -o ConnectTimeout=30 \
  "$ROOT/deploy/systemd/ad-event-processor-vendor-trial-bot.service" \
  "${TARGET}:/etc/systemd/system/ad-event-processor-vendor-trial-bot.service"

OFFER_META="${ROOT}/deploy/vendor/offer_meta.json"
if [[ -f "$OFFER_META" ]]; then
  OFFER_VERSION="$(python3 -c "import json; print(json.load(open('${OFFER_META}'))['version'])")"
else
  OFFER_VERSION="${VENDOR_OFFER_VERSION:-2026-09-09}"
fi
OFFER_URL="${VENDOR_OFFER_SUMMARY_URL:-https://bidshard.com/offer.html}"
REGISTRY="${INSTALL_ROOT}/deploy/vendor/trial_registry.json"

remote "touch '${REGISTRY}'
chmod 600 '${REGISTRY}' 2>/dev/null || true
cat > '${ENV_FILE}' <<EOF
VENDOR_TRIAL_BOT_TOKEN=${TOKEN}
VENDOR_TRIAL_REGISTRY=${REGISTRY}
VENDOR_OFFER_VERSION=${OFFER_VERSION}
VENDOR_OFFER_SUMMARY_URL=${OFFER_URL}
EOF
chmod 600 '${ENV_FILE}'
systemctl daemon-reload
systemctl enable ad-event-processor-vendor-trial-bot
systemctl restart ad-event-processor-vendor-trial-bot
systemctl is-active ad-event-processor-vendor-trial-bot"

log "done"
