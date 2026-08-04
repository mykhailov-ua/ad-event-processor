#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

YES=0
SKIP_PROVISION=0
ENV_FILE=""
CMD="up"

usage() {
	echo "usage: $0 [--yes] [--skip-provision] [--env-file PATH] {up|status|apply|doctor}" >&2
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
	--env-file)
		ENV_FILE="${2:-}"
		shift 2
		;;
	up | status | apply | doctor)
		CMD="$1"
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "bidshard-install: unknown argument: $1" >&2
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
			val="$(grep -m1 "^${key}=" "$file" 2>/dev/null | cut -d= -f2- || true)"
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
		echo "bidshard-install: env file not found: $ENV_FILE" >&2
		exit 1
	fi
	if [[ -f "$file" ]]; then
		set -a
		# shellcheck disable=SC1090
		source "$file"
		set +a
	fi
}

check_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		return 1
	fi
	docker info >/dev/null 2>&1
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
		echo "bidshard-install: no packages in deploy/installer/packages.yaml" >&2
		exit 1
	fi
	if [[ "$YES" != "1" ]]; then
		echo "Docker is missing or daemon not running. Install packages: ${packages[*]}"
		read -r -p "Run apt-get install now? [y/N]: " yn
		if [[ "${yn,,}" != "y" ]]; then
			echo "bidshard-install: install Docker or re-run with --yes" >&2
			exit 1
		fi
	fi
	if ! command -v sudo >/dev/null 2>&1; then
		echo "bidshard-install: sudo required for provisioning" >&2
		exit 1
	fi
	echo "bidshard-install: installing host packages..."
	sudo apt-get update -qq
	sudo apt-get install -y --no-upgrade "${packages[@]}"
	if ! check_docker; then
		echo "bidshard-install: docker still not reachable; try: sudo systemctl start docker" >&2
		exit 1
	fi
}

wait_control_health() {
	local port
	port="$(read_env_var MANAGEMENT_PORT)"
	port="${port:-8188}"
	local url="http://127.0.0.1:${port}/health"
	for _ in $(seq 1 60); do
		if curl -sf "$url" >/dev/null; then
			return 0
		fi
		sleep 2
	done
	echo "bidshard-install: control health check failed on ${url}" >&2
	echo "hint: docker compose logs control" >&2
	return 1
}

build_bootstrap_json() {
	python3 - "$@" <<'PY'
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

config = {
    "tracking_domain": tracking_domain,
    "default_currency": currency,
    "timezone": timezone,
    "ingress_schema": "espx_native",
    "telemetry_enabled": telemetry,
    "profile": "single_vps",
    "edge_xdp": False,
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

print(json.dumps(req))
PY
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
	local license_key="${ESPX_LICENSE_KEY:-}"

	if [[ "$YES" == "1" ]]; then
		if [[ -z "$admin_email" ]] || [[ -z "$admin_password" ]]; then
			echo "bidshard-install: set ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD in install.env or env" >&2
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

	PLATFORM_JSON="$(build_bootstrap_json \
		"$tracking_domain" "$currency" "$timezone" "$telemetry_flag" \
		"$stripe_flag" "$stripe_secret" "$stripe_webhook" \
		"$admin_email" "$admin_password" "$license_key")"

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
		echo "bidshard-install: bootstrap complete"
		return 0
	fi
	if [[ "$http_code" == "409" ]] || echo "$body" | grep -qi 'already bootstrapped'; then
		echo "bidshard-install: platform already bootstrapped (skipping)"
		return 0
	fi
	echo "bidshard-install: bootstrap failed (${http_code}): ${body}" >&2
	echo "hint: check INSTALL_BOOTSTRAP_TOKEN in .env matches control container" >&2
	return 1
}

apply_platform_config() {
	local port key url body http_code
	port="$(read_env_var MANAGEMENT_PORT)"
	port="${port:-8188}"
	key="$(read_env_var ADMIN_API_KEY)"
	if [[ -z "$key" ]]; then
		echo "bidshard-install: ADMIN_API_KEY missing in .env" >&2
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
		echo "bidshard-install: apply failed (${http_code}): ${body}" >&2
		return 1
	fi
	echo "bidshard-install: wrote install.compose.env — restarting stack"
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
		echo "bidshard-install: skip API doctor (ADMIN_API_KEY not set)" >&2
		return 0
	fi

	local raw
	if ! raw="$(curl -sf -H "X-Admin-API-Key: ${key}" "$url")"; then
		echo "bidshard-install: doctor API unreachable" >&2
		return 1
	fi

	python3 - "$raw" <<'PY'
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
		tracking_domain="$(python3 -c 'import json; print(json.load(open("platform_config.json")).get("tracking_domain",""))' 2>/dev/null || true)"
	fi
	echo ""
	echo "=== BidShard ready ==="
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
	provision_host
	if ! check_docker; then
		echo "bidshard-install: docker not available" >&2
		exit 1
	fi
	ensure_env
	load_install_env
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a

	collect_config

	if [[ "${ESPX_USE_RELEASE_IMAGES:-0}" == "1" ]] || [[ -n "${ESPX_APP_IMAGE:-}" ]]; then
		echo "bidshard-install: pulling release images (${ESPX_APP_IMAGE:-from .env})..."
	else
		echo "bidshard-install: building images (first run may take several minutes)..."
	fi
	bash scripts/dev/stack.sh build

	echo "bidshard-install: starting single_vps stack..."
	bash scripts/dev/stack.sh single-vps
	wait_control_health

	bootstrap_platform
	apply_platform_config || true

	run_doctor || true
	print_summary
}

cmd_status() {
	if ! check_docker; then
		echo "bidshard-install: docker not available" >&2
		exit 1
	fi
	bash scripts/dev/stack.sh status
	local port
	port="$(read_env_var MANAGEMENT_PORT)"
	port="${port:-8188}"
	if curl -sf "http://127.0.0.1:${port}/health" >/dev/null; then
		echo "control: healthy (http://127.0.0.1:${port})"
	else
		echo "control: not reachable (http://127.0.0.1:${port})"
		echo "hint: docker compose logs control" >&2
	fi
}

cmd_apply() {
	if ! check_docker; then
		echo "bidshard-install: docker not available" >&2
		exit 1
	fi
	ensure_env
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
	if [[ -x "$ROOT/bin/espx-install" ]]; then
		"$ROOT/bin/espx-install" apply
		return
	fi
	apply_platform_config
}

cmd_doctor() {
	ensure_env
	set -a
	# shellcheck disable=SC1091
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
*)
	usage
	exit 2
	;;
esac
