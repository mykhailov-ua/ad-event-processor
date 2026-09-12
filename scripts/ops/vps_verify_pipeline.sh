#!/usr/bin/env bash
# Role: One command VPS verify+deploy pipeline (sync, build, tests, optional control swap, e2e).
# Env: AED_TARGET, AED_SSH_PORT, AED_INSTALL_ROOT (same as deploy_appliance.sh)
# Verify:
#   bash scripts/ops/vps_verify_pipeline.sh
#   ssh -p 2222 root@45.94.158.106 'tail -f /opt/platform/aed-verify/var/vps-pipeline.log'
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

TARGET="${AED_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${AED_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"
VERIFY_ROOT="${AED_VERIFY_ROOT:-/opt/platform/aed-verify}"
LOG_PATH="${VERIFY_ROOT}/var/vps-pipeline.log"
REMOTE_MODE=0

log() { printf 'vps-pipeline: %s\n' "$*"; }
die() {
  printf 'vps-pipeline: ERROR: %s\n' "$*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --remote)
      REMOTE_MODE=1
      shift
      ;;
    -h | --help)
      sed -n '2,8p' "$0" | tail -n +2
      exit 0
      ;;
    *)
      die "unknown arg: $1"
      ;;
  esac
done

ssh_base=(-o BatchMode=yes -o ConnectTimeout=30 -p "$SSH_PORT")

run_remote_pipeline() {
  ssh "${ssh_base[@]}" "$TARGET" "bash -s" << EOF
set -euo pipefail
mkdir -p '${VERIFY_ROOT}/var'
nohup bash '${VERIFY_ROOT}/scripts/ops/vps_verify_pipeline.sh' --remote >>'${LOG_PATH}' 2>&1 &
echo "\$!"
EOF
}

remote_pipeline() {
  set +e
  export PATH="/usr/local/go/bin:${PATH:-}"
  ROOT="${VERIFY_ROOT}"
  INSTALL="${INSTALL_ROOT}"
  LOG="${LOG_PATH}"
  mkdir -p "${ROOT}/var"

  exec >> "${LOG}" 2>&1

  failures=0
  log() { printf 'vps-pipeline: %s\n' "$*"; }

  log "remote start $(date -Is)"
  log "root=${ROOT} install=${INSTALL}"

  step() {
    local title=$1
    shift
    log "=== ${title} ==="
    "$@"
    local rc=$?
    if [[ ${rc} -eq 0 ]]; then
      log "=== OK: ${title} ==="
    else
      log "=== FAIL: ${title} exit=${rc} ==="
      failures=$((failures + 1))
    fi
    return 0
  }

  ensure_go() {
    if command -v go > /dev/null 2>&1; then
      go version
      return 0
    fi
    if [[ -x /usr/local/go/bin/go ]]; then
      export PATH="/usr/local/go/bin:$PATH"
      go version
      return 0
    fi
    log "install Go 1.25.12"
    local tmp
    tmp="$(mktemp -d)"
    curl -fsSL -o "${tmp}/go.tgz" https://go.dev/dl/go1.25.12.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "${tmp}/go.tgz"
    rm -rf "${tmp}"
    export PATH="/usr/local/go/bin:$PATH"
    go version
  }

  ensure_node22() {
    local ver
    ver="$(node -v 2> /dev/null || true)"
    case "${ver}" in
      v22.* | v23.* | v24.*)
        log "node ok: ${ver}"
        return 0
        ;;
    esac
    log "install Node.js 22"
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    apt-get install -y -qq nodejs
    node -v
  }

  ensure_playwright_deps() {
    cd "${ROOT}/web/e2e"
    if [[ ! -f /var/lib/aed-playwright-deps.ok ]]; then
      npx playwright install-deps chromium
      touch /var/lib/aed-playwright-deps.ok
    fi
    npx playwright install chromium
  }

  ensure_e2e_admin() {
    cd "${INSTALL}"
    local email="e2e-admin@test.local" password="Password123!" login_code db_container
    login_code="$(curl -s -o /dev/null -w '%{http_code}' -X POST 'http://127.0.0.1:8188/api/v1/auth/login' \
      -H 'Content-Type: application/json' \
      -d "{\"email\":\"${email}\",\"password\":\"${password}\"}")"
    if [[ "${login_code}" != "200" ]]; then
      ./bin/admin --env-path .env user delete "${email}" > /dev/null 2>&1 || true
      ./bin/admin --env-path .env user create --email "${email}" --password "${password}" --role A > /dev/null
      set -a
      # shellcheck disable=SC1091
      source .env
      set +a
      db_container="$(docker ps --format '{{.Names}}' | grep -E '(-db-1|^ad-event-processor-db$)' | head -n 1)"
      docker exec "${db_container}" psql -h localhost -p "${DB_PORT:-5430}" -U "${DB_USER:-ad_event_processor_user}" -d "${DB_NAME:-ad_event_processor}" \
        -c "UPDATE users SET email_verified = TRUE WHERE email = '${email}';" > /dev/null
    fi
  }

  wait_health() {
    local i code
    for i in 1 2 3 4 5 6 7 8 9 10; do
      code="$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8188/health || true)"
      if [[ "${code}" == "200" ]]; then
        log "health ok"
        return 0
      fi
      sleep 3
    done
    return 1
  }

  deploy_control_if_ready() {
    local sealed="${INSTALL}/internal/control/control_runtime_sealed.bin" mode="" bak
    if [[ -f "${INSTALL}/.env" ]]; then
      mode="$(grep -E '^AD_EVENT_PROCESSOR_LICENSE_MODE=' "${INSTALL}/.env" | cut -d= -f2- | tr -d '"' | tr '[:upper:]' '[:lower:]' || true)"
    fi
    if [[ ! -f "${sealed}" && "${mode}" != "dev" && "${mode}" != "development" && -n "${mode}" ]]; then
      log "deploy skip: missing ${sealed} (LICENSE_MODE=${mode:-empty}); bin/control.new kept"
      return 0
    fi
    if [[ ! -f "${INSTALL}/bin/control.new" ]]; then
      log "deploy skip: no control.new"
      return 0
    fi
    bak="${INSTALL}/bin/control.bak.pipeline.$(date +%s)"
    cp -f "${INSTALL}/bin/control" "${bak}"
    install -m 755 "${INSTALL}/bin/control.new" "${INSTALL}/bin/control"
    if [[ -f "${INSTALL}/bin/admin.new" ]]; then
      install -m 755 "${INSTALL}/bin/admin.new" "${INSTALL}/bin/admin"
    fi
    systemctl restart ad-event-processor-control
    if ! wait_health; then
      log "deploy rollback: health failed after control swap"
      cp -f "${bak}" "${INSTALL}/bin/control"
      systemctl restart ad-event-processor-control
      wait_health || true
      return 1
    fi
    log "deploy ok"
    return 0
  }

  ensure_go
  cd "${ROOT}"

  step "go test reportjob notify" go test ./internal/reportjob/ -short -run 'TestReportJobNotify_|TestNormalizeReportJobNotify' -count=1
  step "go test adops range" go test ./internal/dashboardadmin/ -short -run 'TestGetAdOpsDashboard_' -count=1
  step "go test smartalerts offset" go test ./internal/smartalerts/ -short -run 'TestHTTPHandlers_listHistory_passesOffset_holdout' -count=1
  step "go test team metrics range" go test ./internal/platformadmin/ -short -run 'TestParseTeamMetricsRange_holdout' -count=1
  step "go test openapi" go test ./internal/openapi/ -count=1

  ensure_node22
  cd "${ROOT}/web"
  step "npm ci web" npm ci
  step "npm run typecheck" npm run typecheck
  step "npm test web" npm test
  step "npm run build" npm run build

  cd "${ROOT}"
  step "go build control" env CGO_ENABLED=0 go build -o "${INSTALL}/bin/control.new" ./cmd/control
  step "go build admin" env CGO_ENABLED=0 go build -o "${INSTALL}/bin/admin.new" ./cmd/admin

  rsync -a "${ROOT}/internal/control/" "${INSTALL}/internal/control/" 2> /dev/null || true
  step "deploy control" deploy_control_if_ready

  step "playwright deps" ensure_playwright_deps
  ensure_e2e_admin
  cd "${INSTALL}"
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  db_container="$(docker ps --format '{{.Names}}' | grep -E '(-db-1|^ad-event-processor-db$)' | head -n 1)"
  customer_id="$(docker exec "${db_container}" psql -h localhost -p "${DB_PORT:-5430}" -U "${DB_USER:-ad_event_processor_user}" -d "${DB_NAME:-ad_event_processor}" -tAc 'select id::text from customers limit 1' | tr -d '[:space:]')"
  cd "${ROOT}/web/e2e"
  step "npm ci e2e" npm ci
  step "playwright dashboards_adops" env ADMIN_E2E_BASE_URL=http://127.0.0.1:8188 PLAYWRIGHT_BASE_URL=http://127.0.0.1:8188 \
    ADMIN_E2E_EMAIL=e2e-admin@test.local ADMIN_E2E_PASSWORD=Password123! \
    ADMIN_E2E_CUSTOMER_ID="${customer_id}" \
    npx playwright test dashboards_adops.spec.js --workers=1 --reporter=list

  log "remote done failures=${failures} $(date -Is)"
  log "log=${LOG}"
  exit "${failures}"
}

if [[ "$REMOTE_MODE" == "1" ]]; then
  remote_pipeline
fi

cd "$ROOT"
log "sync ${ROOT}/ -> ${TARGET}:${VERIFY_ROOT}/"
rsync -az --delete \
  -e "ssh -p ${SSH_PORT} -o BatchMode=yes -o ConnectTimeout=30" \
  --exclude '.git' \
  --exclude 'node_modules' \
  --exclude 'web/node_modules' \
  --exclude 'web/e2e/node_modules' \
  --exclude 'bin' \
  --exclude 'var' \
  --exclude '.cache' \
  "${ROOT}/" "${TARGET}:${VERIFY_ROOT}/"

log "launch remote pipeline (background on VPS)"
pid="$(run_remote_pipeline | tail -n 1)"
log "started remote pid=${pid}"
log "tail: ssh -p ${SSH_PORT} ${TARGET} 'tail -f ${LOG_PATH}'"
log "done (local exit 0; remote still running)"
