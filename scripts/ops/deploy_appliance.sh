#!/usr/bin/env bash
# Role: Build (optional) and deploy control/admin to appliance over SSH.
# Env:
#   AED_TARGET (default root@45.94.158.106)
#   AED_SSH_PORT (default 2222)
#   AED_INSTALL_ROOT (default /opt/platform/ad-event-processor)
#   SKIP_BUILD=1, SKIP_SEED=1, SKIP_PG_MIGRATE=1
# Verify:
#   bash scripts/ops/deploy_appliance.sh --check
#   make deploy-appliance
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

TARGET="${AED_TARGET:-root@${AED_TARGET_HOST:-45.94.158.106}}"
SSH_PORT="${AED_SSH_PORT:-2222}"
INSTALL_ROOT="${AED_INSTALL_ROOT:-/opt/platform/ad-event-processor}"
CONTROL_BIN="${DEPLOY_CONTROL_BIN:-/tmp/aed-control-linux}"
ADMIN_BIN="${DEPLOY_ADMIN_BIN:-/tmp/aed-admin-linux}"
CHECK_ONLY=0

log() { printf 'deploy-appliance: %s\n' "$*"; }
die() {
  printf 'deploy-appliance: ERROR: %s\n' "$*" >&2
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

ssh_base=(
  -o BatchMode=yes
  -o ConnectTimeout=30
  -p "$SSH_PORT"
)

scp_base=(
  -o BatchMode=yes
  -o ConnectTimeout=30
  -P "$SSH_PORT"
)

remote() {
  ssh "${ssh_base[@]}" "$TARGET" "$@"
}

upload() {
  scp "${scp_base[@]}" "$1" "${TARGET}:$2"
}

log "target ${TARGET} port ${SSH_PORT}"
remote "echo ok && hostname"

if [[ "$CHECK_ONLY" == "1" ]]; then
  log "ssh ok"
  exit 0
fi

if [[ "${SKIP_BUILD:-}" != "1" ]]; then
  log "building web embed"
  (cd "$ROOT/web" && npm run typecheck)
  (cd "$ROOT/web" && npm test)
  (cd "$ROOT/web" && npm run build)
  log "building linux control + admin"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$CONTROL_BIN" ./cmd/control
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$ADMIN_BIN" ./cmd/admin
else
  log "SKIP_BUILD=1 (using ${CONTROL_BIN} and ${ADMIN_BIN})"
  [[ -x "$CONTROL_BIN" ]] || die "missing ${CONTROL_BIN}"
  [[ -x "$ADMIN_BIN" ]] || die "missing ${ADMIN_BIN}"
fi

log "upload binaries"
upload "$CONTROL_BIN" "${INSTALL_ROOT}/bin/control.new"
upload "$ADMIN_BIN" "${INSTALL_ROOT}/bin/admin.new"
upload "$ROOT/scripts/ops/seed_appliance_demo.sh" "${INSTALL_ROOT}/scripts/ops/seed_appliance_demo.sh"
upload "$ROOT/scripts/ops/bootstrap_pg_schema.sh" "${INSTALL_ROOT}/scripts/ops/bootstrap_pg_schema.sh"

log "sync clickhouse config (must be files, not docker-created directories)"
remote "mkdir -p '${INSTALL_ROOT}/deploy/clickhouse'"
for ch_file in config.yaml config.unix.yaml users.yaml init.sql recon_materialized_views.sql; do
  remote "rm -rf '${INSTALL_ROOT}/deploy/clickhouse/${ch_file}'"
  upload "$ROOT/deploy/clickhouse/${ch_file}" "${INSTALL_ROOT}/deploy/clickhouse/${ch_file}"
done

log "sync postgres migrations"
remote "mkdir -p '${INSTALL_ROOT}/internal/ingest/migrations'"
scp "${scp_base[@]}" -r "$ROOT/internal/ingest/migrations/." "${TARGET}:${INSTALL_ROOT}/internal/ingest/migrations/"

log "install on target"
remote "set -euo pipefail
cd '${INSTALL_ROOT}'
chmod +x bin/control.new bin/admin.new scripts/ops/seed_appliance_demo.sh
mv -f bin/control bin/control.bak.\$(date +%s) 2>/dev/null || true
mv -f bin/admin bin/admin.bak.\$(date +%s) 2>/dev/null || true
mv -f bin/control.new bin/control
mv -f bin/admin.new bin/admin
chmod +x bin/control bin/admin
"

if [[ "${SKIP_PG_MIGRATE:-}" != "1" ]]; then
  log "apply postgres migrations"
  remote "bash '${INSTALL_ROOT}/scripts/ops/bootstrap_pg_schema.sh'"
else
  log "SKIP_PG_MIGRATE=1"
fi

if [[ "${SKIP_SEED:-}" != "1" ]]; then
  log "run synthetic PG seed"
  remote "bash '${INSTALL_ROOT}/scripts/ops/seed_appliance_demo.sh'"
else
  log "SKIP_SEED=1"
fi

log "restart control"
remote "systemctl restart ad-event-processor-control && sleep 2 && systemctl is-active ad-event-processor-control && curl -sf -o /dev/null -w 'health:%{http_code}\n' http://127.0.0.1:8188/health || true"

log "done"
