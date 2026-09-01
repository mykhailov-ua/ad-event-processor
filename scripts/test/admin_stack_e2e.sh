#!/usr/bin/env bash
set -euo pipefail

# Role: Playwright admin stack E2E against control :8188; smoke bundle always, full matrix when ADMIN_WEB_E2E_NIGHTLY=1.
# Execution context: CI admin-stack-e2e workflow or operator with Docker; exits 0 when web/scripts/build.mjs absent.
# Invariants/contracts enforced: CONTROL_URL reachable; bootstrap credentials from ADMIN_STACK_E2E_* or INSTALL_BOOTSTRAP_TOKEN.
# Verify: bash scripts/test/admin_stack_e2e.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ ! -f "$ROOT/web/scripts/build.mjs" ]]; then
  log() { printf 'admin-stack-e2e: %s\n' "$*"; }
  log "skipped (web/ absent; OpenAPI contracts remain under api/openapi/)"
  exit 0
fi

if [[ -f "$ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a
fi

CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
ADMIN_STACK_E2E_EMAIL="${ADMIN_STACK_E2E_EMAIL:-${ADMIN_BOOTSTRAP_EMAIL:-}}"
ADMIN_STACK_E2E_PASSWORD="${ADMIN_STACK_E2E_PASSWORD:-${ADMIN_BOOTSTRAP_PASSWORD:-}}"
INSTALL_TOKEN="${INSTALL_BOOTSTRAP_TOKEN:-}"

COMPOSE_INGEST_ENV=(
  CH_ENABLED=0
  CONTROL_ENABLE_PAYMENT=0
  CONTROL_ENABLE_BILLING=0
  CONTROL_ENABLE_NOTIFIER=0
  CONTROL_ENABLE_MARGIN_GUARD=0
  CONTROL_ENABLE_COST_SYNC=0
)

log() { printf 'admin-stack-e2e: %s\n' "$*"; }
die() {
  printf 'admin-stack-e2e: ERROR: %s\n' "$*" >&2
  exit 1
}

wait_control() {
  local i=0
  while [[ $i -lt 120 ]]; do
    if curl -sf "${CONTROL_URL}/health" > /dev/null 2>&1; then
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  return 1
}

ensure_env_and_license() {
  if [[ ! -f "$ROOT/.env" ]]; then
    cp "$ROOT/.env.example" "$ROOT/.env"
    log "created .env from .env.example"
  fi
  set -a
  # shellcheck disable=SC1091
  source "$ROOT/.env"
  set +a

  if [[ -z "${INSTALL_BOOTSTRAP_TOKEN:-}" ]]; then
    INSTALL_BOOTSTRAP_TOKEN="$(openssl rand -hex 32)"
    echo "INSTALL_BOOTSTRAP_TOKEN=${INSTALL_BOOTSTRAP_TOKEN}" >> "$ROOT/.env"
    log "generated INSTALL_BOOTSTRAP_TOKEN in .env"
  fi
  INSTALL_TOKEN="${INSTALL_BOOTSTRAP_TOKEN}"

  ADMIN_STACK_E2E_EMAIL="${ADMIN_STACK_E2E_EMAIL:-${ADMIN_BOOTSTRAP_EMAIL:-admin@test.local}}"
  ADMIN_STACK_E2E_PASSWORD="${ADMIN_STACK_E2E_PASSWORD:-${ADMIN_BOOTSTRAP_PASSWORD:-Password123!}}"

  mkdir -p "$ROOT/var"
  if [[ ! -f "$ROOT/var/license.jwt" ]]; then
    log "issuing pilot license for stack e2e"
    go run ./cmd/license-issue \
      --sku pilot \
      --customer admin-stack-e2e \
      --out "$ROOT/var/license.jwt" \
      --force \
      --force-reason admin_stack_e2e
  fi
}

build_admin_ui() {
  log "building admin UI (dist + embed sync)"
  (cd "$ROOT/web" && npm ci && npm run build)
}

start_or_refresh_stack() {
  if curl -sf "${CONTROL_URL}/health" > /dev/null 2>&1; then
    log "control healthy; rebuilding control with fresh admin embed"
    "${COMPOSE_INGEST_ENV[@]}" docker compose --profile ingest_only up -d --build --force-recreate control
  else
    log "control not healthy; starting ingest-only stack"
    "${COMPOSE_INGEST_ENV[@]}" docker compose --profile ingest_only up -d --build db redis-0 control
  fi
  wait_control || die "control did not become healthy"
}

bootstrap_if_needed() {
  local meta
  meta="$(curl -sf "${CONTROL_URL}/api/v1/meta" || true)"
  if [[ -z "$meta" ]]; then
    die "GET /api/v1/meta failed"
  fi
  if echo "$meta" | grep -q '"bootstrap_complete":true'; then
    log "platform already bootstrapped"
    return 0
  fi
  [[ -n "$INSTALL_TOKEN" ]] || die "INSTALL_BOOTSTRAP_TOKEN required for bootstrap"
  [[ -n "$ADMIN_STACK_E2E_EMAIL" && -n "$ADMIN_STACK_E2E_PASSWORD" ]] \
    || die "ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD required for bootstrap"

  log "bootstrapping platform"
  curl -sf -X POST "${CONTROL_URL}/api/v1/settings/platform/bootstrap" \
    -H "Content-Type: application/json" \
    -H "X-Install-Token: ${INSTALL_TOKEN}" \
    -d "$(jq -n \
      --arg email "$ADMIN_STACK_E2E_EMAIL" \
      --arg password "$ADMIN_STACK_E2E_PASSWORD" \
      '{
        config: {
          tracking_domain: "track.local",
          default_currency: "USD",
          timezone: "UTC",
          ingress_schema: "ad_event_processor_native",
          profile: "single_vps",
          network_interface: "eth0",
          telemetry_enabled: true,
          edge_xdp: false,
          edge_expose_click: true,
          edge_expose_openrtb: false,
          stripe: { enabled: false }
        },
        admin_email: $email,
        admin_password: $password
      }')" > /dev/null
}

install_playwright_deps() {
  local e2e_dir="$ROOT/web/e2e"
  if [ -f "$e2e_dir/package-lock.json" ]; then
    (cd "$e2e_dir" && npm ci)
  else
    (cd "$e2e_dir" && npm install)
  fi
  (cd "$e2e_dir" && npx playwright install chromium)
}

ensure_env_and_license
build_admin_ui
start_or_refresh_stack
bootstrap_if_needed
install_playwright_deps

export ADMIN_STACK_E2E=1
export PLAYWRIGHT_BASE_URL="${CONTROL_URL}"
export ADMIN_E2E_BASE_URL="${CONTROL_URL}"
export ADMIN_STACK_E2E_EMAIL
export ADMIN_STACK_E2E_PASSWORD

log "running admin web smoke playwright bundle"
bash "$SCRIPTS/ci/admin/web_e2e_smoke.sh"

if [[ "${ADMIN_WEB_E2E_NIGHTLY:-0}" = "1" ]]; then
  log "running admin web nightly playwright matrix"
  bash "$SCRIPTS/ci/admin/web_e2e_nightly.sh"
fi

log "admin stack e2e PASSED"
