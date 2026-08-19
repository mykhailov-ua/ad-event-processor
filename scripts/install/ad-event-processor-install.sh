#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
cd "$ROOT"

DEV_LICENSE_REL="var/license.jwt"

ensure_dev_license_file() {
  mkdir -p var
  if [[ ! -f "$DEV_LICENSE_REL" ]]; then
    : > "$DEV_LICENSE_REL"
    chmod 600 "$DEV_LICENSE_REL"
  fi
}

license_write_path() {
  local p
  p="$(read_env_var AD_EVENT_PROCESSOR_LICENSE_PATH)"
  if [[ -n "$p" && "$p" != "/etc/ad-event-processor/license.jwt" ]]; then
    echo "$p"
    return
  fi
  if [[ -f go.mod ]]; then
    echo "$DEV_LICENSE_REL"
    return
  fi
  echo "/etc/ad-event-processor/license.jwt"
}

write_license_token() {
  local token="$1"
  local dest
  dest="$(license_write_path)"
  if [[ "$dest" == /* ]]; then
    if ! command -v sudo > /dev/null 2>&1; then
      echo "ad-event-processor-install: sudo required to write $dest" >&2
      exit 1
    fi
    sudo mkdir -p "$(dirname "$dest")"
    printf '%s' "$token" | sudo tee "$dest" > /dev/null
    sudo chmod 600 "$dest"
    return
  fi
  mkdir -p "$(dirname "$dest")"
  printf '%s' "$token" > "$dest"
  chmod 600 "$dest"
}

YES=0
SKIP_PROVISION=0
SKIP_PREFLIGHT=0
ACCEPT_EULA=0
ENV_FILE=""
CMD="up"

usage() {
  echo "usage: $0 [--yes] [--skip-provision] [--skip-preflight] [--accept-eula] [--env-file PATH] {up|status|apply|doctor|license-apply [JWT]}" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes)
      YES=1
      shift
      ;;
    --skip-provision)
      SKIP_PROVISION=1
      shift
      ;;
    --skip-preflight)
      SKIP_PREFLIGHT=1
      shift
      ;;
    --accept-eula)
      ACCEPT_EULA=1
      shift
      ;;
    --env-file)
      ENV_FILE="${2:-}"
      shift 2
      ;;
    up | status | apply | doctor | license-apply)
      CMD="$1"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "ad-event-processor-install: unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

read_env_var() {
  local key="$1"
  local file val
  for file in .env deploy/installer/install.env; do
    if [[ -f "$file" ]]; then
      val="$(grep -m1 "^${key}=" "$file" 2> /dev/null | cut -d= -f2- || true)"
      if [[ -n "$val" ]]; then
        echo "$val"
        return 0
      fi
    fi
  done
  echo ""
}

load_install_env() {
  local file="${ENV_FILE:-deploy/installer/install.env}"
  if [[ -n "$ENV_FILE" ]] && [[ ! -f "$ENV_FILE" ]]; then
    echo "ad-event-processor-install: env file not found: $ENV_FILE" >&2
    exit 1
  fi
  if [[ -f "$file" ]]; then
    set -a

    source "$file"
    set +a
  fi
}

check_docker() {
  if ! command -v docker > /dev/null 2>&1; then
    return 1
  fi
  docker info > /dev/null 2>&1
}

ensure_env() {
  if [[ ! -f .env ]]; then
    cp .env.example .env
  fi
  if ! grep -q '^INSTALL_BOOTSTRAP_TOKEN=' .env; then
    echo "INSTALL_BOOTSTRAP_TOKEN=$(openssl rand -hex 32)" >> .env
  elif [[ -z "$(read_env_var INSTALL_BOOTSTRAP_TOKEN)" ]]; then
    token="$(openssl rand -hex 32)"
    sed -i "s/^INSTALL_BOOTSTRAP_TOKEN=.*/INSTALL_BOOTSTRAP_TOKEN=${token}/" .env
  fi
  ensure_dev_license_file
}

set_env_key() {
  local key="$1"
  local val="$2"
  if grep -q "^${key}=" .env; then
    sed -i "s|^${key}=.*|${key}=${val}|" .env
  else
    echo "${key}=${val}" >> .env
  fi
}

compute_deployment_fingerprint() {
  local mid="" id path
  for id in /etc/machine-id /var/lib/dbus/machine-id; do
    if [[ -r "$id" ]]; then
      mid="$(tr -d '[:space:]' < "$id")"
      break
    fi
  done
  local -a paths=()
  if [[ -n "${ROOT:-}" ]]; then
    paths+=("$(realpath -m "$ROOT" 2> /dev/null || echo "$ROOT")")
  fi
  paths+=("$(pwd)")
  local lic_path
  lic_path="$(installer_env_dual AD_EVENT_PROCESSOR_LICENSE_PATH AD_EVENT_PROCESSOR_LICENSE_PATH)"
  if [[ -z "$lic_path" ]]; then
    lic_path="/etc/ad-event-processor/license.jwt"
  fi
  paths+=("$(dirname "$lic_path")")
  python3 - "$mid" "${paths[@]}" << 'PY'
import hashlib, sys
h = hashlib.sha256()
h.update(sys.argv[1].encode())
seen = set()
for p in sys.argv[2:]:
    p = p.strip()
    if not p or p in seen:
        continue
    seen.add(p)
    h.update(b"\0")
    h.update(p.encode())
print(h.hexdigest())
PY
}

setup_offline_license() {
  local pub_file="$ROOT/deploy/vendor/license_public.key"
  local mode
  mode="$(installer_license_mode)"
  local jwt
  jwt="$(installer_license_key)"
  if [[ -z "$jwt" ]]; then
    jwt="${AD_EVENT_PROCESSOR_LICENSE_KEY:-}"
  fi

  set_env_key AD_EVENT_PROCESSOR_LICENSE_MODE "$mode"
  set_env_key AD_EVENT_PROCESSOR_LICENSE_SERVER ""
  if [[ -f go.mod ]]; then
    set_env_key AD_EVENT_PROCESSOR_LICENSE_PATH "$DEV_LICENSE_REL"
  else
    set_env_key AD_EVENT_PROCESSOR_LICENSE_PATH "/etc/ad-event-processor/license.jwt"
  fi
  set_env_key AD_EVENT_PROCESSOR_TELEMETRY_OPT_IN "0"
  set_env_key AD_EVENT_PROCESSOR_LICENSE_REQUIRED "$(installer_license_required)"

  local fp
  fp="$(compute_deployment_fingerprint)"
  set_env_key AD_EVENT_PROCESSOR_DEPLOYMENT_FINGERPRINT "$fp"
  echo "ad-event-processor-install: deployment fingerprint=$fp (include in vendor renewal ticket)"
  if command -v go > /dev/null 2>&1 && [[ -f go.mod ]]; then
    local hwid_line
    hwid_line="$(go run ./cmd/installer license host-id 2> /dev/null | grep '^hwid_v2=' || true)"
    if [[ -n "$hwid_line" ]]; then
      echo "ad-event-processor-install: deployment ${hwid_line} (preferred for new renewal JWTs)"
    fi
  fi

  if [[ -z "$(grep -m1 '^AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY=' .env 2> /dev/null | cut -d= -f2- | tr -d '[:space:]')" ]] && [[ -f "$pub_file" ]]; then
    local pub
    pub="$(tr -d '[:space:]' < "$pub_file")"
    set_env_key AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY "$pub"
  fi

  if [[ -z "$jwt" ]]; then
    echo "ad-event-processor-install: set AD_EVENT_PROCESSOR_LICENSE_KEY (or legacy AD_EVENT_PROCESSOR_LICENSE_KEY) to the monthly license JWT in install.env" >&2
    exit 1
  fi

  write_license_token "$jwt"
}

prompt_default() {
  local var="$1"
  local msg="$2"
  local default="${3:-}"
  local input=""
  if [[ -n "$default" ]]; then
    read -r -p "${msg} [${default}]: " input
    eval "$var=\"${input:-$default}\""
  else
    read -r -p "${msg}: " input
    eval "$var=\"$input\""
  fi
}

provision_host() {
  if [[ "$SKIP_PROVISION" == "1" ]]; then
    return 0
  fi
  if check_docker; then
    return 0
  fi
  local -a packages=()
  while IFS= read -r pkg; do
    [[ -n "$pkg" ]] && packages+=("$pkg")
  done < <(grep -E '^\s+- ' deploy/installer/packages.yaml | sed 's/^[[:space:]]*-[[:space:]]*//')
  if [[ ${#packages[@]} -eq 0 ]]; then
    echo "ad-event-processor-install: no packages in deploy/installer/packages.yaml" >&2
    exit 1
  fi
  if [[ "$YES" != "1" ]]; then
    echo "Docker is missing or daemon not running. Install packages: ${packages[*]}"
    read -r -p "Run apt-get install now? [y/N]: " yn
    if [[ "${yn,,}" != "y" ]]; then
      echo "ad-event-processor-install: install Docker or re-run with --yes" >&2
      exit 1
    fi
  fi
  if ! command -v sudo > /dev/null 2>&1; then
    echo "ad-event-processor-install: sudo required for provisioning" >&2
    exit 1
  fi
  echo "ad-event-processor-install: installing host packages..."
  sudo apt-get update -qq
  sudo apt-get install -y --no-upgrade "${packages[@]}"
  if ! check_docker; then
    echo "ad-event-processor-install: docker still not reachable; try: sudo systemctl start docker" >&2
    exit 1
  fi
}

run_preflight() {
  if [[ "$SKIP_PREFLIGHT" == "1" ]]; then
    return 0
  fi
  echo "ad-event-processor-install: running host preflight..."
  bash "$SCRIPTS/install/preflight.sh"
}

wait_control_health() {
  local port
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  local url="http://127.0.0.1:${port}/health"
  for _ in $(seq 1 60); do
    if curl -sf "$url" > /dev/null; then
      return 0
    fi
    sleep 2
  done
  echo "ad-event-processor-install: control health check failed on ${url}" >&2
  echo "hint: docker compose logs control" >&2
  return 1
}

build_bootstrap_json() {
  python3 - "$@" << 'PY'
import json, sys

tracking_domain = sys.argv[1]
currency = sys.argv[2]
timezone = sys.argv[3]
telemetry = sys.argv[4] == "1"
stripe_enabled = sys.argv[5] == "1"
stripe_secret = sys.argv[6]
stripe_webhook = sys.argv[7]
admin_email = sys.argv[8]
admin_password = sys.argv[9]
license_key = sys.argv[10]
eula_version = sys.argv[11]

config = {
    "tracking_domain": tracking_domain,
    "default_currency": currency,
    "timezone": timezone,
    "ingress_schema": "ad_event_processor_native",
    "telemetry_enabled": telemetry,
    "profile": "single_vps",
    "edge_xdp": False,
    "edge_expose_click": True,
    "edge_expose_openrtb": False,
    "network_interface": "eth0",
    "stripe": {
        "enabled": stripe_enabled,
    },
}
if stripe_enabled:
    config["stripe"]["secret_key"] = stripe_secret
    config["stripe"]["webhook_secret"] = stripe_webhook

req = {
    "config": config,
    "admin_email": admin_email,
    "admin_password": admin_password,
}
if license_key:
    req["license_key"] = license_key
if eula_version:
    req["eula_version"] = eula_version

print(json.dumps(req))
PY
}

EULA_VERSION="2026-01"

require_eula_acceptance() {
  if [[ "$ACCEPT_EULA" == "1" ]]; then
    return 0
  fi
  if [[ "$YES" == "1" ]]; then
    echo "ad-event-processor-install: re-run with --accept-eula (required for install)" >&2
    exit 1
  fi
  echo ""
  echo "ad-event-processor on-premise license agreement (version ${EULA_VERSION})"
  if [[ -f pkg/legal/EULA.txt ]]; then
    head -n 12 pkg/legal/EULA.txt
    echo "..."
  fi
  read -r -p "Accept license agreement? [y/N]: " yn
  if [[ "${yn,,}" != "y" ]]; then
    echo "ad-event-processor-install: install aborted — EULA not accepted" >&2
    exit 1
  fi
  ACCEPT_EULA=1
}

collect_config() {
  local tracking_domain="${TRACKING_DOMAIN:-}"
  local currency="${DEFAULT_CURRENCY:-USD}"
  local timezone="${TIMEZONE:-UTC}"
  local telemetry="${TELEMETRY_ENABLED:-true}"
  local stripe_enabled="${STRIPE_ENABLED:-false}"
  local stripe_secret="${STRIPE_SECRET_KEY:-}"
  local stripe_webhook="${STRIPE_WEBHOOK_SECRET:-}"
  local admin_email="${ADMIN_BOOTSTRAP_EMAIL:-}"
  local admin_password="${ADMIN_BOOTSTRAP_PASSWORD:-}"
  local license_key="${AD_EVENT_PROCESSOR_LICENSE_KEY:-}"

  if [[ "$YES" == "1" ]]; then
    if [[ -z "$admin_email" ]] || [[ -z "$admin_password" ]]; then
      echo "ad-event-processor-install: set ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD in install.env or env" >&2
      exit 1
    fi
  else
    prompt_default tracking_domain "Tracking domain (e.g. trk.example.com)" "$tracking_domain"
    prompt_default currency "Default currency" "$currency"
    prompt_default timezone "Timezone" "$timezone"

    read -r -p "Enable telemetry? (Y/n) [Y]: " telemetry_yn
    telemetry_yn="${telemetry_yn:-Y}"
    telemetry="true"
    if [[ "${telemetry_yn,,}" == "n" ]]; then
      telemetry="false"
    fi

    stripe_enabled="false"
    stripe_secret=""
    stripe_webhook=""
    read -r -p "Enable Stripe payments? (y/N): " stripe_yn
    if [[ "${stripe_yn,,}" == "y" ]]; then
      stripe_enabled="true"
      prompt_default stripe_secret "Stripe secret key" ""
      prompt_default stripe_webhook "Stripe webhook secret" ""
    fi

    prompt_default admin_email "Admin email" "$admin_email"
    prompt_default admin_password "Admin password" "$admin_password"
    prompt_default license_key "License key (optional)" "$license_key"
  fi

  case "${telemetry,,}" in
    1 | true | yes | y) telemetry_flag="1" ;;
    *) telemetry_flag="0" ;;
  esac
  case "${stripe_enabled,,}" in
    1 | true | yes | y) stripe_flag="1" ;;
    *) stripe_flag="0" ;;
  esac

  local eula_version=""
  if [[ "$ACCEPT_EULA" == "1" ]]; then
    eula_version="$EULA_VERSION"
  fi

  PLATFORM_JSON="$(build_bootstrap_json \
    "$tracking_domain" "$currency" "$timezone" "$telemetry_flag" \
    "$stripe_flag" "$stripe_secret" "$stripe_webhook" \
    "$admin_email" "$admin_password" "$license_key" "$eula_version")"

  echo "$PLATFORM_JSON" | python3 -c 'import json,sys; c=json.load(sys.stdin)["config"]; print(json.dumps(c))' > platform_config.json
}

bootstrap_platform() {
  local port token url
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  token="$(read_env_var INSTALL_BOOTSTRAP_TOKEN)"
  url="http://127.0.0.1:${port}/api/v1/settings/platform/bootstrap"

  local body http_code
  body="$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-Install-Token: ${token}" \
    -d "$PLATFORM_JSON" \
    "$url")"
  http_code="${body##*$'\n'}"
  body="${body%$'\n'*}"

  if [[ "$http_code" == "200" ]]; then
    echo "ad-event-processor-install: bootstrap complete"
    return 0
  fi
  if [[ "$http_code" == "409" ]] || echo "$body" | grep -qi 'already bootstrapped'; then
    echo "ad-event-processor-install: platform already bootstrapped (skipping)"
    return 0
  fi
  echo "ad-event-processor-install: bootstrap failed (${http_code}): ${body}" >&2
  echo "hint: check INSTALL_BOOTSTRAP_TOKEN in .env matches control container" >&2
  return 1
}

apply_platform_config() {
  local port key url body http_code
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  key="$(read_env_var ADMIN_API_KEY)"
  if [[ -z "$key" ]]; then
    echo "ad-event-processor-install: ADMIN_API_KEY missing in .env" >&2
    return 1
  fi
  url="http://127.0.0.1:${port}/api/v1/settings/platform/apply"
  body="$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-Admin-API-Key: ${key}" \
    -d '{}' \
    "$url")"
  http_code="${body##*$'\n'}"
  body="${body%$'\n'*}"
  if [[ "$http_code" != "200" ]]; then
    echo "ad-event-processor-install: apply failed (${http_code}): ${body}" >&2
    return 1
  fi
  echo "ad-event-processor-install: wrote install.compose.env — restarting stack"
  bash scripts/dev/stack.sh single-vps
}

run_doctor() {
  local port key url
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  key="$(read_env_var ADMIN_API_KEY)"
  url="http://127.0.0.1:${port}/api/v1/ops/doctor"

  if ! bash scripts/ci/deps.sh; then
    echo "hint: bash scripts/dev/stack.sh status" >&2
    return 1
  fi

  if [[ -z "$key" ]]; then
    echo "ad-event-processor-install: skip API doctor (ADMIN_API_KEY not set)" >&2
    return 0
  fi

  local raw
  if ! raw="$(curl -sf -H "X-Admin-API-Key: ${key}" "$url")"; then
    echo "ad-event-processor-install: doctor API unreachable" >&2
    return 1
  fi

  python3 - "$raw" << 'PY'
import json, sys

raw = sys.argv[1]
doc = json.loads(raw)
overall = doc.get("overall", "unknown")
print(f"doctor overall: {overall}")
hints = {
    "redis": "docker compose logs redis-0",
    "postgres": "docker compose logs db",
    "clickhouse": "docker compose logs clickhouse",
    "dns": "set DNS A-record for tracking domain to this host; configure nginx/Caddy for TLS",
}
failed = False
for c in doc.get("checks", []):
    st = c.get("status", "")
    if st in ("fail", "warn"):
        msg = f"  [{st}] {c.get('id')}: {c.get('message')}"
        hint = c.get("hint") or hints.get(c.get("id", ""), "")
        if hint:
            msg += f" | fix: {hint}"
        print(msg)
        if st == "fail":
            failed = True
if doc.get("click_url_template"):
    print(f"click template: {doc['click_url_template']}")
if failed:
    sys.exit(1)
PY
}

print_summary() {
  local port tracking_domain
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  tracking_domain="${TRACKING_DOMAIN:-}"
  if [[ -z "$tracking_domain" ]] && [[ -f platform_config.json ]]; then
    tracking_domain="$(python3 -c 'import json; print(json.load(open("platform_config.json")).get("tracking_domain",""))' 2> /dev/null || true)"
  fi
  echo ""
  echo "=== ad-event-processor ready ==="
  echo "Control UI:  http://127.0.0.1:${port}"
  echo "Login:       http://127.0.0.1:${port}/login"
  echo "Bootstrap UI: http://127.0.0.1:${port}/bootstrap (alternative to CLI install)"
  echo "Checklist:    http://127.0.0.1:${port}/install/done"
  local ingress_enabled="${INGRESS_ENABLED:-}"
  if [[ -z "$ingress_enabled" ]]; then
    ingress_enabled="$(read_env_var INGRESS_ENABLED)"
  fi
  if [[ -n "$tracking_domain" ]]; then
    if [[ "$ingress_enabled" == "1" ]]; then
      echo "Tracking:    https://${tracking_domain}/click?campaign_id={campaign_id}&sub1={sub1}"
      local admin_domain="${ADMIN_DOMAIN:-}"
      if [[ -n "$admin_domain" ]]; then
        echo "Control UI:  https://${admin_domain}"
      else
        echo "Control UI:  http://127.0.0.1:${port} (set ADMIN_DOMAIN for HTTPS admin host)"
      fi
    else
      echo "Click URL:   https://${tracking_domain}/click?campaign_id={campaign_id}&sub1={sub1}"
      echo "DNS:         point ${tracking_domain} A-record to this server"
      echo "TLS:         set INGRESS_ENABLED=1 in install.env for automatic HTTPS (Caddy)"
    fi
  else
    echo "Tracking:    set TRACKING_DOMAIN in install.env and run: $0 apply"
  fi
  echo "Status:      $0 status"
  echo "Doctor:      $0 doctor"
}

cmd_up() {
  run_preflight
  provision_host
  if ! check_docker; then
    echo "ad-event-processor-install: docker not available" >&2
    exit 1
  fi
  ensure_env
  load_install_env
  require_eula_acceptance
  setup_offline_license
  set -a

  source .env
  set +a

  collect_config

  if installer_use_release_images; then
    echo "ad-event-processor-install: pulling release images ($(installer_release_app_image):-from .env})..."
  else
    echo "ad-event-processor-install: building images (first run may take several minutes)..."
  fi
  bash scripts/dev/stack.sh build

  echo "ad-event-processor-install: starting single_vps stack..."
  bash scripts/dev/stack.sh single-vps
  wait_control_health

  bootstrap_platform
  apply_platform_config || true

  run_doctor || true
  print_summary
}

cmd_license_apply() {
  local token="${1:-}"
  ensure_env
  load_install_env
  if [[ -z "$token" ]]; then
    token="$(installer_license_key)"
  fi
  if [[ -z "$token" ]]; then
    echo "ad-event-processor-install: pass JWT argument or set AD_EVENT_PROCESSOR_LICENSE_KEY" >&2
    exit 2
  fi
  if ! check_docker; then
    echo "ad-event-processor-install: docker not available" >&2
    exit 1
  fi
  set -a

  source .env
  set +a
  local port key url body http_code
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  key="$(read_env_var ADMIN_API_KEY)"
  if [[ -z "$key" ]]; then
    echo "ad-event-processor-install: ADMIN_API_KEY missing in .env" >&2
    exit 1
  fi
  url="http://127.0.0.1:${port}/api/v1/license/apply"
  body="$(python3 -c 'import json,sys; print(json.dumps({"token": sys.argv[1]}))' "$token")"
  local resp
  resp="$(curl -s -w "\n%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-Admin-API-Key: ${key}" \
    -d "$body" \
    "$url")"
  http_code="${resp##*$'\n'}"
  resp="${resp%$'\n'*}"
  if [[ "$http_code" == "200" ]]; then
    echo "ad-event-processor-install: license applied"
    echo "$resp"
    write_license_token "$token"
    return 0
  fi
  echo "ad-event-processor-install: license apply failed (${http_code}): ${resp}" >&2
  return 1
}

cmd_status() {
  if ! check_docker; then
    echo "ad-event-processor-install: docker not available" >&2
    exit 1
  fi
  bash scripts/dev/stack.sh status
  local port
  port="$(read_env_var MANAGEMENT_PORT)"
  port="${port:-8188}"
  if curl -sf "http://127.0.0.1:${port}/health" > /dev/null; then
    echo "control: healthy (http://127.0.0.1:${port})"
  else
    echo "control: not reachable (http://127.0.0.1:${port})"
    echo "hint: docker compose logs control" >&2
  fi
}

cmd_apply() {
  if ! check_docker; then
    echo "ad-event-processor-install: docker not available" >&2
    exit 1
  fi
  ensure_env
  set -a

  source .env
  set +a
  if [[ -x "$ROOT/bin/ad-event-processor-install" ]]; then
    "$ROOT/bin/ad-event-processor-install" apply
    return
  fi
  apply_platform_config
}

cmd_doctor() {
  ensure_env
  set -a

  source .env
  set +a
  run_doctor
}

case "$CMD" in
  up)
    cmd_up
    ;;
  status)
    cmd_status
    ;;
  apply)
    cmd_apply
    ;;
  doctor)
    cmd_doctor
    ;;
  license-apply)
    cmd_license_apply "${1:-}"
    ;;
  *)
    usage
    exit 2
    ;;
esac
