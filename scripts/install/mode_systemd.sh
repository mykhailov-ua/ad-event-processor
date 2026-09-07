#!/usr/bin/env bash
# Role: Systemd appliance install: infra (optional docker), PG migrate, binaries, units, ingress, admin seed.
# Execution context: ad-event-processor-install.sh --mode systemd up.
# Env knobs: INSTALL_INFRA=docker|external; INGRESS_ENABLED; AD_EVENT_PROCESSOR_INSTALL_ROOT.
# Verify: bash scripts/install/mode_systemd.sh --help
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
source "$SCRIPTS/lib/redis_topology.sh"
source "$SCRIPTS/lib/dev_bind_mounts.sh"
cd "$ROOT"

INSTALL_ROOT="${AD_EVENT_PROCESSOR_INSTALL_ROOT:-$ROOT}"
SYSTEMD_INFRA="${INSTALL_INFRA:-docker}"

usage() {
  echo "usage: $0 {up|bootstrap-schema|status}" >&2
}

cmd="${1:-up}"

aed_install_root() {
  printf '%s' "$(cd "$INSTALL_ROOT" && pwd)"
}

require_binaries() {
  local root="$1"
  local missing=0
  for bin in control tracker processor; do
    if [[ ! -x "${root}/bin/${bin}" ]]; then
      echo "mode_systemd: missing executable ${root}/bin/${bin}" >&2
      missing=1
    fi
  done
  if [[ "$missing" -ne 0 ]]; then
    echo "mode_systemd: place garbled binaries in bin/ or set AD_EVENT_PROCESSOR_BIN_DIR" >&2
    exit 1
  fi
}

copy_binaries_if_requested() {
  local root="$1"
  local src="${AD_EVENT_PROCESSOR_BIN_DIR:-}"
  if [[ -z "$src" ]]; then
    return 0
  fi
  mkdir -p "${root}/bin"
  for bin in control tracker processor; do
    if [[ -x "${src}/${bin}" ]]; then
      install -m 0755 "${src}/${bin}" "${root}/bin/${bin}"
    fi
  done
}

render_systemd_units() {
  local root="$1"
  local unit_dir="/etc/systemd/system"
  local svc
  if ! command -v sudo > /dev/null 2>&1; then
    echo "mode_systemd: sudo required to install systemd units" >&2
    exit 1
  fi
  for svc in ad-event-processor-control ad-event-processor-tracker ad-event-processor-processor; do
    sed "s|__INSTALL_ROOT__|${root}|g" "${ROOT}/deploy/systemd/${svc}.service" \
      | sudo tee "${unit_dir}/${svc}.service" > /dev/null
    sudo chmod 644 "${unit_dir}/${svc}.service"
  done
}

render_secrets_env() {
  local dotenv="$1"
  if ! command -v sudo > /dev/null 2>&1; then
    echo "mode_systemd: sudo required to write /etc/ad-event-processor/secrets.env" >&2
    exit 1
  fi
  sudo mkdir -p /etc/ad-event-processor
  sudo cp "$dotenv" /etc/ad-event-processor/secrets.env
  sudo chmod 600 /etc/ad-event-processor/secrets.env
  if [[ -f var/license.jwt ]]; then
    sudo cp var/license.jwt /etc/ad-event-processor/license.jwt
    sudo chmod 600 /etc/ad-event-processor/license.jwt
  fi
}

start_infra_docker() {
  echo "mode_systemd: starting docker infra (db, redis, broker)..."
  bash "$SCRIPTS/dev/stack/stack.sh" infra-only
}

start_ingress_docker() {
  if [[ "$(installer_read_env INGRESS_ENABLED)" != "1" ]]; then
    return 0
  fi
  bash "$SCRIPTS/dev/stack/stack.sh" ingress-only
}

stop_conflicting_docker_apps() {
  if ! command -v docker > /dev/null 2>&1; then
    return 0
  fi
  bash "$SCRIPTS/dev/stack/stack.sh" stop-app-containers 2> /dev/null || true
}

systemd_up() {
  local root
  root="$(aed_install_root)"
  copy_binaries_if_requested "$root"
  require_binaries "$root"

  if [[ "$SYSTEMD_INFRA" == "docker" ]]; then
    if ! check_docker; then
      echo "mode_systemd: docker required for --infra docker (install Docker or use --infra external)" >&2
      exit 1
    fi
    start_infra_docker
  fi

  bash "$SCRIPTS/ops/bootstrap_pg_schema.sh"
  render_secrets_env "${ROOT}/.env"
  stop_conflicting_docker_apps
  render_systemd_units "$root"

  sudo systemctl daemon-reload
  sudo systemctl enable ad-event-processor-control ad-event-processor-tracker ad-event-processor-processor
  sudo systemctl restart ad-event-processor-control
  sleep 3
  sudo systemctl restart ad-event-processor-processor ad-event-processor-tracker

  start_ingress_docker

  if ! installer_ui_activation_enabled; then
    if [[ -f "$SCRIPTS/dev/stack/seed_admin.sh" ]]; then
      bash "$SCRIPTS/dev/stack/seed_admin.sh" || true
    fi
  fi

  local port admin_domain activate_url
  port="$(installer_read_env MANAGEMENT_PORT)"
  port="${port:-8188}"
  admin_domain="$(installer_read_env ADMIN_DOMAIN)"
  echo ""
  echo "ad-event-processor systemd install complete"
  echo "Install root: ${root}"
  if installer_ui_activation_enabled; then
    if [[ -n "$admin_domain" ]] && [[ "$(installer_read_env INGRESS_ENABLED)" == "1" ]]; then
      activate_url="https://${admin_domain}/activate"
    else
      activate_url="http://127.0.0.1:${port}/activate"
    fi
    echo "Activate:     ${activate_url}"
  elif [[ -n "$admin_domain" ]]; then
    echo "Admin UI:     https://${admin_domain}/login"
  else
    echo "Admin UI:     http://127.0.0.1:${port}/login"
  fi
  echo "Logs:         sudo journalctl -u ad-event-processor-control -f"
}

systemd_bootstrap_schema() {
  if [[ "$SYSTEMD_INFRA" == "docker" ]] && check_docker; then
    start_infra_docker
  fi
  bash "$SCRIPTS/ops/bootstrap_pg_schema.sh"
}

systemd_status() {
  sudo systemctl status ad-event-processor-control ad-event-processor-tracker ad-event-processor-processor --no-pager || true
  if check_docker; then
    bash "$SCRIPTS/dev/stack/stack.sh" status 2> /dev/null || true
  fi
}

check_docker() {
  command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1
}

case "$cmd" in
  up)
    systemd_up
    ;;
  bootstrap-schema)
    systemd_bootstrap_schema
    ;;
  status)
    systemd_status
    ;;
  -h | --help)
    usage
    ;;
  *)
    usage
    exit 2
    ;;
esac
