#!/usr/bin/env bash
set -euo pipefail

# Role: Playwright campaigns_filters.spec.js against local dev UI or remote control embed.
# Execution context: Operator machine; local path uses :5173 when control redirects dev UI.
# Env: ADMIN_E2E_TARGET=local|vps|both (default both);
#   ADMIN_E2E_VPS_EMAIL/PASSWORD override VPS login (else remote seed-admin summary when SSH works);
#   ADMIN_E2E_SSH_TARGET (default root@45.94.158.106), ADMIN_E2E_SSH_PORT (default 2222).
# Verify: bash scripts/test/campaigns_filters_e2e.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

E2E_DIR="$ROOT/web/e2e"
TARGETS="${ADMIN_E2E_TARGET:-both}"
VPS_URL="${ADMIN_E2E_VPS_URL:-http://45.94.158.106:8188}"
LOCAL_DEV_URL="${ADMIN_E2E_LOCAL_URL:-http://127.0.0.1:5173}"
SSH_TARGET="${ADMIN_E2E_SSH_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${ADMIN_E2E_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"

log() { printf 'campaigns-filters-e2e: %s\n' "$*"; }
die() {
  printf 'campaigns-filters-e2e: ERROR: %s\n' "$*" >&2
  exit 1
}

load_local_credentials() {
  if [[ -f "$ROOT/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "$ROOT/.env"
    set +a
  fi
  export ADMIN_E2E_EMAIL="${ADMIN_E2E_EMAIL:-${ADMIN_STACK_E2E_EMAIL:-${ADMIN_BOOTSTRAP_EMAIL:-admin@test.local}}}"
  export ADMIN_E2E_PASSWORD="${ADMIN_E2E_PASSWORD:-${ADMIN_STACK_E2E_PASSWORD:-${ADMIN_BOOTSTRAP_PASSWORD:-Password123!}}}"
  [[ -n "$ADMIN_E2E_EMAIL" && -n "$ADMIN_E2E_PASSWORD" ]] \
    || die "set ADMIN_E2E_EMAIL and ADMIN_E2E_PASSWORD or ADMIN_BOOTSTRAP_* in .env"
}

load_vps_credentials() {
  if [[ -n "${ADMIN_E2E_VPS_EMAIL:-}" && -n "${ADMIN_E2E_VPS_PASSWORD:-}" ]]; then
    export ADMIN_E2E_EMAIL="$ADMIN_E2E_VPS_EMAIL"
    export ADMIN_E2E_PASSWORD="$ADMIN_E2E_VPS_PASSWORD"
    return 0
  fi

  local remote_out email password e2e_email e2e_password
  e2e_email="${ADMIN_E2E_VPS_BOOTSTRAP_EMAIL:-e2e-admin@test.local}"
  e2e_password="${ADMIN_E2E_VPS_BOOTSTRAP_PASSWORD:-Password123!}"
  if remote_out="$(
    ssh -o BatchMode=yes -o ConnectTimeout=20 -p "$SSH_PORT" "$SSH_TARGET" bash -s << EOF
set -euo pipefail
cd /opt/platform/ad-event-processor
email="${e2e_email}"
password="${e2e_password}"
login_code="\$(
  curl -s -o /dev/null -w '%{http_code}' -X POST 'http://127.0.0.1:8188/api/v1/auth/login' \
    -H 'Content-Type: application/json' \
    -d "{\\"email\\":\\"\${email}\\",\\"password\\":\\"\${password}\\"}"
)"
if [[ "\$login_code" != "200" ]]; then
  ./bin/admin --env-path .env user delete "\$email" > /dev/null 2>&1 || true
  ./bin/admin --env-path .env user create --email "\$email" --password "\$password" --role A > /dev/null
  db_container="\$(docker ps --format '{{.Names}}' | grep -E '(-db-1|^ad-event-processor-db$)' | head -n 1)"
  if [[ -n "\$db_container" ]]; then
    set -a
    # shellcheck disable=SC1091
    source .env
    set +a
    docker exec "\$db_container" psql -h localhost -p "\${DB_PORT:-5430}" -U "\${DB_USER:-ad_event_processor_user}" -d "\${DB_NAME:-ad_event_processor}" \
      -c "UPDATE users SET email_verified = TRUE WHERE email = '\$email';" > /dev/null
  fi
  login_code="\$(
    curl -s -o /dev/null -w '%{http_code}' -X POST 'http://127.0.0.1:8188/api/v1/auth/login' \
      -H 'Content-Type: application/json' \
      -d "{\\"email\\":\\"\${email}\\",\\"password\\":\\"\${password}\\"}"
  )"
fi
if [[ "\$login_code" != "200" ]]; then
  echo 'LOGIN_FAIL=1'
  exit 0
fi
printf 'EMAIL=%s\nPASSWORD_B64=%s\n' "\$email" "\$(printf '%s' "\$password" | base64 -w0)"
EOF
  )"; then
    if printf '%s\n' "$remote_out" | grep -q '^LOGIN_FAIL=1'; then
      remote_out=""
    fi
    email="$(printf '%s\n' "$remote_out" | sed -n 's/^EMAIL=//p' | tail -n 1)"
    password_b64="$(printf '%s\n' "$remote_out" | sed -n 's/^PASSWORD_B64=//p' | tail -n 1)"
    if [[ -n "$email" && -n "$password_b64" ]]; then
      password="$(printf '%s' "$password_b64" | base64 -d 2> /dev/null || true)"
    fi
    if [[ -n "$email" && -n "$password" ]]; then
      export ADMIN_E2E_EMAIL="$email"
      export ADMIN_E2E_PASSWORD="$password"
      log "VPS E2E admin ready (${email})"
      return 0
    fi
  fi

  load_local_credentials
  log "VPS credentials fallback to local ADMIN_BOOTSTRAP_* (${ADMIN_E2E_EMAIL})"
}

preflight_login() {
  local base_url="$1"
  local email="$2"
  local password="$3"
  local payload http_code
  payload="$(python3 -c 'import json,sys; print(json.dumps({"email":sys.argv[1],"password":sys.argv[2]}))' "$email" "$password")"
  http_code="$(
    curl -s -o /dev/null -w '%{http_code}' -X POST "${base_url}/api/v1/auth/login" \
      -H 'Content-Type: application/json' \
      -d "$payload"
  )"
  if [[ "$http_code" != "200" ]]; then
    die "login preflight failed for ${base_url} (HTTP ${http_code}); set ADMIN_E2E_VPS_EMAIL/PASSWORD"
  fi
}

install_playwright() {
  if [ ! -d "$E2E_DIR/node_modules/@playwright/test" ]; then
    (cd "$E2E_DIR" && npm ci)
  fi
  (cd "$E2E_DIR" && npx playwright install chromium)
}

run_suite() {
  local label="$1"
  local base_url="$2"
  local email="$3"
  local password="$4"
  log "playwright campaigns_filters.spec.js target=${label} base=${base_url}"
  preflight_login "$base_url" "$email" "$password"
  (
    cd "$E2E_DIR"
    ADMIN_E2E_BASE_URL="$base_url" PLAYWRIGHT_BASE_URL="$base_url" \
      ADMIN_E2E_EMAIL="$email" ADMIN_E2E_PASSWORD="$password" \
      npx playwright test campaigns_filters.spec.js --workers=1 --reporter=list
  )
  return $?
}

[[ -d "$E2E_DIR" ]] || die "missing web/e2e"
install_playwright

failures=0

run_local() {
  load_local_credentials
  curl -sf http://127.0.0.1:8188/health > /dev/null \
    || die "local control unhealthy; run: bash scripts/dev/admin_ui.sh stack"
  curl -sf "$LOCAL_DEV_URL/" > /dev/null \
    || die "local dev UI unreachable at ${LOCAL_DEV_URL}; run: bash scripts/dev/admin_ui.sh web"
  if run_suite local "$LOCAL_DEV_URL" "$ADMIN_E2E_EMAIL" "$ADMIN_E2E_PASSWORD"; then
    log "local PASSED"
  else
    log "local FAILED exit=$?"
    failures=$((failures + 1))
  fi
}

run_vps() {
  load_vps_credentials
  curl -sf "${VPS_URL}/health" > /dev/null || die "VPS health failed: ${VPS_URL}/health"
  if run_suite vps "$VPS_URL" "$ADMIN_E2E_EMAIL" "$ADMIN_E2E_PASSWORD"; then
    log "vps PASSED"
  else
    log "vps FAILED exit=$?"
    failures=$((failures + 1))
  fi
}

case "$TARGETS" in
  local) run_local ;;
  vps) run_vps ;;
  both)
    run_local
    run_vps
    ;;
  *)
    die "unknown ADMIN_E2E_TARGET=${TARGETS} (use local|vps|both)"
    ;;
esac

if [[ "$failures" -gt 0 ]]; then
  die "${failures} target(s) failed"
fi

log "PASSED"
