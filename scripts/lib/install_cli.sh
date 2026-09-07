#!/usr/bin/env bash
# Role: Shared installer CLI flags (mode, domains, license) for docker and systemd paths.
# Execution context: Sourced by scripts/install/install.sh and ad-event-processor-install.sh.
# Verify: INSTALL_MODE=systemd bash -c 'source scripts/lib/install_cli.sh && install_cli_apply'

install_cli_defaults() {
  : "${INSTALL_MODE:=docker}"
  : "${INSTALL_INFRA:=docker}"
}

install_cli_usage() {
  cat <<'EOF'
usage: install.sh <docker|systemd> [command] [options]

Commands (default: up):
  up                 Install and start
  bootstrap-schema   Apply Postgres migrations only
  status             Stack / service status
  doctor             Health checks

Config: edit deploy/installer/install.env (license, domains, admin login).
Env overrides: AD_EVENT_PROCESSOR_LICENSE_KEY, ADMIN_DOMAIN, TRACKING_DOMAIN, ...

Options:
  --yes              Non-interactive (implies EULA acceptance)
  --infra external   Systemd with your own Postgres/Redis (default: docker infra)
  --env-file PATH    Alternate install.env
  --skip-provision   Skip apt/docker provisioning
  --skip-preflight   Skip host preflight

Examples:
  bash install.sh docker up
  bash install.sh systemd up
  curl -fsSL https://releases.example.com/install.sh | bash -s docker up
EOF
}

install_cli_maybe_accept_eula() {
  if [[ "${YES:-0}" == "1" ]]; then
    ACCEPT_EULA=1
  fi
}

# Non-interactive appliance install (owner completes /activate in the browser).
install_cli_autodetect_yes() {
  if [[ "${YES:-0}" == "1" ]]; then
    install_cli_maybe_accept_eula
    return 0
  fi
  if installer_ui_activation_enabled; then
    YES=1
    ACCEPT_EULA=1
    return 0
  fi
  local install_env="${ROOT}/deploy/installer/install.env"
  local lic admin_email admin_pass
  if [[ -f "${ENV_FILE:-}" ]]; then
    install_env="$ENV_FILE"
  fi
  if [[ ! -f "$install_env" ]]; then
    return 0
  fi
  set -a
  # shellcheck disable=SC1090
  source "$install_env"
  set +a
  lic="${AD_EVENT_PROCESSOR_LICENSE_KEY:-}"
  admin_email="${ADMIN_BOOTSTRAP_EMAIL:-}"
  admin_pass="${ADMIN_BOOTSTRAP_PASSWORD:-}"
  if [[ -n "$lic" && -n "$admin_email" && -n "$admin_pass" && "$admin_pass" != "change-me-strong-password" ]]; then
    YES=1
    ACCEPT_EULA=1
  fi
}

install_cli_parse() {
  install_cli_defaults
  INSTALL_CMD="up"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --mode)
        INSTALL_MODE="${2:-}"
        shift 2
        ;;
      --infra)
        INSTALL_INFRA="${2:-}"
        shift 2
        ;;
      --license | --license-key)
        AD_EVENT_PROCESSOR_LICENSE_KEY="${2:-}"
        shift 2
        ;;
      --admin-domain)
        ADMIN_DOMAIN="${2:-}"
        shift 2
        ;;
      --tracking-domain)
        TRACKING_DOMAIN="${2:-}"
        shift 2
        ;;
      --acme-email)
        CADDY_ACME_EMAIL="${2:-}"
        shift 2
        ;;
      --admin-email)
        ADMIN_BOOTSTRAP_EMAIL="${2:-}"
        shift 2
        ;;
      --admin-password)
        ADMIN_BOOTSTRAP_PASSWORD="${2:-}"
        shift 2
        ;;
      --yes)
        YES=1
        ACCEPT_EULA=1
        shift
        ;;
      --accept-eula)
        ACCEPT_EULA=1
        shift
        ;;
      docker)
        INSTALL_MODE=docker
        shift
        ;;
      systemd)
        INSTALL_MODE=systemd
        shift
        ;;
      external)
        INSTALL_INFRA=external
        shift
        ;;
      --env-file)
        ENV_FILE="${2:-}"
        shift 2
        ;;
      --skip-provision)
        SKIP_PROVISION=1
        shift
        ;;
      --skip-preflight)
        SKIP_PREFLIGHT=1
        shift
        ;;
      up | status | bootstrap-schema | doctor | apply | license-apply)
        INSTALL_CMD="$1"
        shift
        break
        ;;
      -h | --help)
        install_cli_usage
        exit 0
        ;;
      *)
        return 1
        ;;
    esac
  done
  case "$INSTALL_MODE" in
    docker | systemd) ;;
    *)
      echo "install: invalid --mode ${INSTALL_MODE} (want docker or systemd)" >&2
      exit 2
      ;;
  esac
  case "$INSTALL_INFRA" in
    docker | external) ;;
    *)
      echo "install: invalid --infra ${INSTALL_INFRA} (want docker or external)" >&2
      exit 2
      ;;
  esac
  return 0
}

install_cli_set_env_key() {
  local file="$1"
  local key="$2"
  local val="$3"
  mkdir -p "$(dirname "$file")"
  touch "$file"
  if grep -q "^${key}=" "$file" 2> /dev/null; then
    sed -i "s|^${key}=.*|${key}=${val}|" "$file"
  else
    echo "${key}=${val}" >> "$file"
  fi
}

install_cli_ensure_secret() {
  local file="$1"
  local key="$2"
  local gen="$3"
  local cur
  cur="$(grep -m1 "^${key}=" "$file" 2> /dev/null | cut -d= -f2- || true)"
  if [[ -z "$cur" || "$cur" == "dev-admin-api-key-change-me" || "$cur" == "01234567890123456789012345678901" || "$cur" == "your_redis_password_here" ]]; then
    install_cli_set_env_key "$file" "$key" "$gen"
  fi
}

install_cli_apply() {
  local install_env="${ROOT}/deploy/installer/install.env"
  local dotenv="${ROOT}/.env"
  mkdir -p "${ROOT}/deploy/installer" "${ROOT}/var"

  if [[ ! -f "$dotenv" ]]; then
    cp "${ROOT}/.env.example" "$dotenv"
  fi
  if [[ ! -f "$install_env" ]]; then
    cp "${ROOT}/deploy/installer/install.env.example" "$install_env" 2> /dev/null \
      || touch "$install_env"
  fi

  if [[ -n "${AD_EVENT_PROCESSOR_LICENSE_KEY:-}" ]]; then
    install_cli_set_env_key "$install_env" AD_EVENT_PROCESSOR_LICENSE_KEY "$AD_EVENT_PROCESSOR_LICENSE_KEY"
    install_cli_set_env_key "$install_env" AD_EVENT_PROCESSOR_LICENSE_MODE file
    install_cli_set_env_key "$install_env" AD_EVENT_PROCESSOR_LICENSE_REQUIRED 1
    install_cli_set_env_key "$install_env" AD_EVENT_PROCESSOR_UI_ACTIVATION 0
    install_cli_set_env_key "$dotenv" AD_EVENT_PROCESSOR_UI_ACTIVATION 0
  fi
  if [[ -n "${ADMIN_DOMAIN:-}" ]]; then
    install_cli_set_env_key "$install_env" ADMIN_DOMAIN "$ADMIN_DOMAIN"
    install_cli_set_env_key "$dotenv" ADMIN_DOMAIN "$ADMIN_DOMAIN"
    install_cli_set_env_key "$install_env" INGRESS_ENABLED 1
    install_cli_set_env_key "$dotenv" INGRESS_ENABLED 1
  fi
  if [[ -n "${TRACKING_DOMAIN:-}" ]]; then
    install_cli_set_env_key "$install_env" TRACKING_DOMAIN "$TRACKING_DOMAIN"
    install_cli_set_env_key "$dotenv" TRACKING_DOMAIN "$TRACKING_DOMAIN"
    install_cli_set_env_key "$install_env" INGRESS_ENABLED 1
    install_cli_set_env_key "$dotenv" INGRESS_ENABLED 1
  fi
  if [[ -n "${CADDY_ACME_EMAIL:-}" ]]; then
    install_cli_set_env_key "$install_env" CADDY_ACME_EMAIL "$CADDY_ACME_EMAIL"
    install_cli_set_env_key "$dotenv" CADDY_ACME_EMAIL "$CADDY_ACME_EMAIL"
  fi
  if [[ -n "${ADMIN_BOOTSTRAP_EMAIL:-}" ]]; then
    install_cli_set_env_key "$install_env" ADMIN_BOOTSTRAP_EMAIL "$ADMIN_BOOTSTRAP_EMAIL"
    install_cli_set_env_key "$dotenv" ADMIN_BOOTSTRAP_EMAIL "$ADMIN_BOOTSTRAP_EMAIL"
  fi
  if [[ -n "${ADMIN_BOOTSTRAP_PASSWORD:-}" ]]; then
    install_cli_set_env_key "$install_env" ADMIN_BOOTSTRAP_PASSWORD "$ADMIN_BOOTSTRAP_PASSWORD"
    install_cli_set_env_key "$dotenv" ADMIN_BOOTSTRAP_PASSWORD "$ADMIN_BOOTSTRAP_PASSWORD"
  fi

  install_cli_set_env_key "$dotenv" AD_EVENT_PROCESSOR_INSTALL_MODE "$INSTALL_MODE"
  install_cli_set_env_key "$dotenv" AD_EVENT_PROCESSOR_INSTALL_INFRA "$INSTALL_INFRA"
  install_cli_set_env_key "$install_env" AD_EVENT_PROCESSOR_UI_ACTIVATION "${AD_EVENT_PROCESSOR_UI_ACTIVATION:-1}"
  install_cli_set_env_key "$dotenv" AD_EVENT_PROCESSOR_UI_ACTIVATION "${AD_EVENT_PROCESSOR_UI_ACTIVATION:-1}"

  if [[ "$INSTALL_MODE" == "docker" ]]; then
    install_cli_set_env_key "$install_env" AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES "${AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES:-1}"
  fi

  if [[ "$INSTALL_MODE" == "systemd" && "$INSTALL_INFRA" == "docker" ]]; then
    install_cli_apply_systemd_host_tcp_defaults "$dotenv"
  fi

  install_cli_ensure_secret "$dotenv" TOKEN_SYMMETRIC_KEY "$(openssl rand -hex 16)"
  install_cli_ensure_secret "$dotenv" ADMIN_API_KEY "$(openssl rand -hex 24)"
  install_cli_ensure_secret "$dotenv" REDIS_PASSWORD "$(openssl rand -hex 16)"
  install_cli_set_env_key "$dotenv" TRUSTED_PROXIES "127.0.0.1/32,::1/128"
}

install_cli_apply_systemd_host_tcp_defaults() {
  local dotenv="$1"
  local db_user db_pass db_name db_port redis_pass shard_count i port addrs
  db_user="$(grep -m1 '^DB_USER=' "$dotenv" | cut -d= -f2-)"
  db_pass="$(grep -m1 '^DB_PASSWORD=' "$dotenv" | cut -d= -f2-)"
  db_name="$(grep -m1 '^DB_NAME=' "$dotenv" | cut -d= -f2-)"
  db_port="$(grep -m1 '^DB_PORT=' "$dotenv" | cut -d= -f2-)"
  db_port="${db_port:-5430}"
  db_user="${db_user:-ad_event_processor_user}"
  db_name="${db_name:-ad_event_processor}"
  redis_pass="$(grep -m1 '^REDIS_PASSWORD=' "$dotenv" | cut -d= -f2-)"
  shard_count="$(grep -m1 '^REDIS_SHARD_COUNT=' "$dotenv" | cut -d= -f2-)"
  shard_count="${shard_count:-4}"

  install_cli_set_env_key "$dotenv" DB_DSN "postgres://${db_user}:${db_pass}@127.0.0.1:${db_port}/${db_name}?sslmode=disable"
  install_cli_set_env_key "$dotenv" PAYMENT_DB_DSN "postgres://${db_user}:${db_pass}@127.0.0.1:${db_port}/${db_name}?sslmode=disable"

  addrs=""
  i=0
  while [[ "$i" -lt "$shard_count" ]]; do
    port=$((6479 + i))
    if [[ -n "$redis_pass" ]]; then
      entry="redis://:${redis_pass}@127.0.0.1:${port}/0"
    else
      entry="127.0.0.1:${port}"
    fi
    if [[ -n "$addrs" ]]; then
      addrs="${addrs},${entry}"
    else
      addrs="$entry"
    fi
    i=$((i + 1))
  done
  install_cli_set_env_key "$dotenv" REDIS_ADDRS "$addrs"
  install_cli_set_env_key "$dotenv" ENV development
}
