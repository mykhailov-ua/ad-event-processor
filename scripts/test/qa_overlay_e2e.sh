#!/usr/bin/env bash
set -euo pipefail

# Role: Manual QA overlay regression bundle (Campaigns filters + Flows + clone API).
# Execution context: operator laptop with Docker; always tears stack down on exit.
# Verify: bash scripts/test/qa_overlay_e2e.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

STACK_DOWN=0
cleanup() {
  if [[ "$STACK_DOWN" == "0" ]]; then
    printf 'qa_overlay_e2e: stack down\n'
    ENV=development bash "$ROOT/scripts/dev/stack/stack.sh" down 2>&1 || true
    STACK_DOWN=1
  fi
}
trap cleanup EXIT

log() { printf 'qa_overlay_e2e: %s\n' "$*"; }
die() {
  printf 'qa_overlay_e2e: ERROR: %s\n' "$*" >&2
  exit 1
}

wait_control() {
  local i=0
  while [[ $i -lt 120 ]]; do
    if curl -sf "${CONTROL_URL:-http://127.0.0.1:8188}/health" > /dev/null 2>&1; then
      log "control healthy (${CONTROL_URL:-http://127.0.0.1:8188})"
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  return 1
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
  local install_token="${INSTALL_BOOTSTRAP_TOKEN:-}"
  local admin_email="${ADMIN_STACK_E2E_EMAIL:-${ADMIN_BOOTSTRAP_EMAIL:-admin@test.local}}"
  local admin_password="${ADMIN_STACK_E2E_PASSWORD:-${ADMIN_BOOTSTRAP_PASSWORD:-Password123!}}"
  [[ -n "$install_token" ]] || die "INSTALL_BOOTSTRAP_TOKEN required for bootstrap (set in .env)"
  log "bootstrapping platform"
  curl -sf -X POST "${CONTROL_URL}/api/v1/settings/platform/bootstrap" \
    -H "Content-Type: application/json" \
    -H "X-Install-Token: ${install_token}" \
    -d "$(jq -n \
      --arg email "$admin_email" \
      --arg password "$admin_password" \
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

CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:8188}"
export ENV=development

log "playwright install chromium (before stack)"
cd "$ROOT/web/e2e"
if ! npx playwright install chromium 2>&1; then
  die "playwright install chromium failed"
fi

log "stack up (ingest-only, ENV=development)"
ENV=development bash "$ROOT/scripts/dev/stack/stack.sh" ingest-only

wait_control || die "control did not become healthy at ${CONTROL_URL}"

bootstrap_if_needed

wait_control || die "control unhealthy before playwright at ${CONTROL_URL}"

log "playwright QA specs"
export PLAYWRIGHT_BASE_URL="$CONTROL_URL"
export ADMIN_E2E_BASE_URL="$CONTROL_URL"
export ADMIN_STACK_E2E_EMAIL="${ADMIN_STACK_E2E_EMAIL:-${ADMIN_BOOTSTRAP_EMAIL:-admin@test.local}}"
export ADMIN_STACK_E2E_PASSWORD="${ADMIN_STACK_E2E_PASSWORD:-${ADMIN_BOOTSTRAP_PASSWORD:-Password123!}}"
npx playwright test \
  campaigns_filters.spec.js \
  creative_flows.spec.js \
  campaign_single_clone.spec.js \
  --reporter=line

log "PASSED"
