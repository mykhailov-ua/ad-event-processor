#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

CELL_ID=""
CUSTOMER_NAME=""
DEPLOYMENT_ID=""
SKIP_UP=0
DRY_RUN=0

usage() {
  echo "usage: $0 --cell-id <slug> --customer <name> [--deployment-id <uuid>] [--skip-up] [--dry-run]" >&2
  echo "  Provision one vendor-managed SaaS cell (isolated compose project + managed_saas JWT)." >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cell-id)
      CELL_ID="${2:-}"
      shift 2
      ;;
    --customer)
      CUSTOMER_NAME="${2:-}"
      shift 2
      ;;
    --deployment-id)
      DEPLOYMENT_ID="${2:-}"
      shift 2
      ;;
    --skip-up)
      SKIP_UP=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "managed-saas-cell-bootstrap: unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$CELL_ID" || -z "$CUSTOMER_NAME" ]]; then
  usage
  exit 2
fi

if [[ -z "$DEPLOYMENT_ID" ]]; then
  DEPLOYMENT_ID="$(uuidgen 2> /dev/null || cat /proc/sys/kernel/random/uuid)"
fi

log() { printf 'managed-saas-cell-bootstrap: %s\n' "$*"; }
warn() { printf 'managed-saas-cell-bootstrap: WARN %s\n' "$*" >&2; }
die() {
  printf 'managed-saas-cell-bootstrap: ERROR %s\n' "$*" >&2
  exit 1
}

PROJECT_NAME="saas-${CELL_ID}"
LICENSE_PATH="var/saas-cells/${CELL_ID}/license.jwt"
ENV_PATH="var/saas-cells/${CELL_ID}/.env"

log "cell_id=${CELL_ID} project=${PROJECT_NAME} deployment_id=${DEPLOYMENT_ID}"

if [[ "$DRY_RUN" -eq 1 ]]; then
  log "dry-run: would write ${LICENSE_PATH} and ${ENV_PATH}"
  exit 0
fi

mkdir -p "var/saas-cells/${CELL_ID}"
if [[ ! -f .env ]]; then
  cp .env.example .env
fi

if [[ ! -f deploy/vendor/license_private.key ]]; then
  die "deploy/vendor/license_private.key missing; cannot issue managed_saas JWT"
fi

log "issuing managed_saas license"
ad_event_processor_go_run ./cmd/license-issue \
  --sku managed_saas \
  --customer "$CUSTOMER_NAME" \
  --deployment-id "$DEPLOYMENT_ID" \
  --out "$LICENSE_PATH"

cat > "$ENV_PATH" << EOF
COMPOSE_PROJECT_NAME=${PROJECT_NAME}
DEPLOYMENT_MODE=managed_saas
MANAGED_SAAS_CELL_ID=${CELL_ID}
AD_EVENT_PROCESSOR_LICENSE_PATH=${LICENSE_PATH}
AD_EVENT_PROCESSOR_LICENSE_MODE=file
EOF

log "wrote cell env at ${ENV_PATH}"
log "start stack: COMPOSE_PROJECT_NAME=${PROJECT_NAME} DEPLOYMENT_MODE=managed_saas docker compose -f deploy/compose/docker-compose.yaml -f deploy/compose/docker-compose.managed-saas-cell.yaml --profile single_vps up -d"

if [[ "$SKIP_UP" -eq 0 ]]; then
  export COMPOSE_PROJECT_NAME="$PROJECT_NAME"
  export DEPLOYMENT_MODE=managed_saas
  export MANAGED_SAAS_CELL_ID="$CELL_ID"
  export AD_EVENT_PROCESSOR_LICENSE_PATH="$LICENSE_PATH"
  export AD_EVENT_PROCESSOR_LICENSE_MODE=file
  docker compose \
    --project-directory "$ROOT" \
    -f "$ROOT/deploy/compose/docker-compose.yaml" \
    -f "$ROOT/deploy/compose/docker-compose.managed-saas-cell.yaml" \
    --profile single_vps up -d
  log "compose up complete for ${PROJECT_NAME}"
fi

log "data export: GET /api/v1/billing/usage/export and POST /api/v1/billing/exports (see docs/MANAGED_SAAS.md)"
