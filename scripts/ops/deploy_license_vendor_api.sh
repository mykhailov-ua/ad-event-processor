#!/usr/bin/env bash
# Role: Build and deploy license-vendor-api to vendor VPS (localhost :8199).
# Env:
#   AED_TARGET (default root@45.94.158.106)
#   AED_SSH_PORT (default 2222)
#   AED_INSTALL_ROOT (default /opt/platform/ad-event-processor)
#   LICENSE_VENDOR_API_TOKEN (optional; generated on first deploy if unset)
# Verify:
#   bash scripts/ops/deploy_license_vendor_api.sh --check
#   LICENSE_VENDOR_API_TOKEN=$(openssl rand -hex 32) bash scripts/ops/deploy_license_vendor_api.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

TARGET="${AED_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${AED_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"
ENV_FILE="/etc/ad-event-processor/license-vendor-api.env"
CHECK_ONLY=0

log() { printf 'deploy-license-vendor-api: %s\n' "$*"; }
die() {
  printf 'deploy-license-vendor-api: ERROR: %s\n' "$*" >&2
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

OFFER_META="${ROOT}/deploy/vendor/offer_meta.json"
if [[ ! -f "$OFFER_META" ]]; then
  die "missing ${OFFER_META}"
fi
OFFER_VERSION="$(python3 -c "import json; print(json.load(open('${OFFER_META}'))['version'])")"
KEY_ID="${LICENSE_VENDOR_KEY_ID:-2026-01}"

if [[ ! -f "$ROOT/deploy/vendor/license_private.key" ]]; then
  die "missing deploy/vendor/license_private.key (see deploy/vendor/KEYS.md)"
fi

log "build license-vendor-api"
mkdir -p "$ROOT/bin"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$ROOT/bin/license-vendor-api" ./cmd/license-vendor-api

log "sync binary and vendor files"
remote "mkdir -p '${INSTALL_ROOT}/bin' '${INSTALL_ROOT}/deploy/vendor' /etc/ad-event-processor"
rsync "${rsync_base[@]}" "$ROOT/bin/license-vendor-api" "${TARGET}:${INSTALL_ROOT}/bin/license-vendor-api"
rsync "${rsync_base[@]}" \
  "$ROOT/deploy/vendor/sku.yaml" \
  "${TARGET}:${INSTALL_ROOT}/deploy/vendor/sku.yaml"
rsync "${rsync_base[@]}" \
  "$ROOT/deploy/vendor/license_private.key" \
  "${TARGET}:${INSTALL_ROOT}/deploy/vendor/license_private.key"
if [[ -f "$ROOT/deploy/vendor/trial_registry.json" ]]; then
  rsync "${rsync_base[@]}" \
    "$ROOT/deploy/vendor/trial_registry.json" \
    "${TARGET}:${INSTALL_ROOT}/deploy/vendor/trial_registry.json"
fi
scp -P "$SSH_PORT" -o BatchMode=yes -o ConnectTimeout=30 \
  "$ROOT/deploy/systemd/ad-event-processor-license-vendor-api.service" \
  "${TARGET}:/etc/systemd/system/ad-event-processor-license-vendor-api.service"

TOKEN="${LICENSE_VENDOR_API_TOKEN:-}"
if [[ -z "$TOKEN" ]]; then
  TOKEN="$(remote "if [[ -f '${ENV_FILE}' ]] && grep -q '^LICENSE_VENDOR_API_TOKEN=' '${ENV_FILE}'; then grep '^LICENSE_VENDOR_API_TOKEN=' '${ENV_FILE}' | head -n1 | cut -d= -f2-; else openssl rand -hex 32; fi")"
fi

remote "set -euo pipefail
INSTALL_ROOT='${INSTALL_ROOT}'
ENV_FILE='${ENV_FILE}'
TOKEN='${TOKEN}'
REGISTRY=\"\${INSTALL_ROOT}/deploy/vendor/trial_registry.json\"
touch \"\${REGISTRY}\"
chmod 600 \"\${REGISTRY}\" 2>/dev/null || true
if [[ -f \"\${INSTALL_ROOT}/deploy/vendor/license_private.key\" ]]; then
  chmod 600 \"\${INSTALL_ROOT}/deploy/vendor/license_private.key\"
fi
if [[ ! -f \"\${INSTALL_ROOT}/deploy/vendor/license_private.key\" ]]; then
  echo 'deploy-license-vendor-api: missing license_private.key on target' >&2
  exit 1
fi
cat > \"\${ENV_FILE}\" <<EOF
LICENSE_VENDOR_API_TOKEN=\${TOKEN}
LICENSE_VENDOR_API_LISTEN=127.0.0.1:8199
LICENSE_VENDOR_SKU_FILE=\${INSTALL_ROOT}/deploy/vendor/sku.yaml
LICENSE_VENDOR_PRIVATE_KEY_FILE=\${INSTALL_ROOT}/deploy/vendor/license_private.key
LICENSE_VENDOR_KEY_ID=${KEY_ID}
VENDOR_TRIAL_REGISTRY=\${REGISTRY}
VENDOR_OFFER_VERSION=${OFFER_VERSION}
EOF
chmod 600 \"\${ENV_FILE}\"
systemctl daemon-reload
systemctl enable ad-event-processor-license-vendor-api
systemctl restart ad-event-processor-license-vendor-api
sleep 1
systemctl is-active ad-event-processor-license-vendor-api
curl -sf http://127.0.0.1:8199/health
curl -sf -H \"Authorization: Bearer \${TOKEN}\" http://127.0.0.1:8199/api/v1/vendor/plans/buttons | head -c 200
"

log "deployed — API token (save for Telegram bot):"
log "${TOKEN}"
log "contract: deploy/vendor/LICENSE_VENDOR_API.openapi.yaml"
log "health: ssh -p ${SSH_PORT} ${TARGET} curl -sf http://127.0.0.1:8199/health"
