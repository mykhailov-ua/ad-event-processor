#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CMD="${1:-up}"

read_env_var() {
	local key="$1"
	if [[ -f .env ]]; then
		grep -m1 "^${key}=" .env 2>/dev/null | cut -d= -f2- || true
	else
		echo ""
	fi
}

check_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "bidshard-install: docker not found" >&2
		exit 1
	fi
	if ! docker info >/dev/null 2>&1; then
		echo "bidshard-install: docker daemon not reachable" >&2
		exit 1
	fi
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

cmd_up() {
	check_docker
	ensure_env
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a

	local tracking_domain currency timezone telemetry stripe_enabled
	local stripe_secret stripe_webhook admin_email admin_password license_key

	prompt_default tracking_domain "Tracking domain (e.g. trk.example.com)" ""
	prompt_default currency "Default currency" "USD"
	prompt_default timezone "Timezone" "UTC"

	read -r -p "Enable telemetry? (Y/n) [Y]: " telemetry_yn
	telemetry_yn="${telemetry_yn:-Y}"
	telemetry="1"
	if [[ "${telemetry_yn,,}" == "n" ]]; then
		telemetry="0"
	fi

	stripe_enabled="0"
	stripe_secret=""
	stripe_webhook=""
	read -r -p "Enable Stripe payments? (y/N): " stripe_yn
	if [[ "${stripe_yn,,}" == "y" ]]; then
		stripe_enabled="1"
		prompt_default stripe_secret "Stripe secret key" ""
		prompt_default stripe_webhook "Stripe webhook secret" ""
	fi

	prompt_default admin_email "Admin email" ""
	prompt_default admin_password "Admin password" ""
	prompt_default license_key "License key (optional)" ""

	local platform_json
	platform_json="$(build_bootstrap_json \
		"$tracking_domain" "$currency" "$timezone" "$telemetry" \
		"$stripe_enabled" "$stripe_secret" "$stripe_webhook" \
		"$admin_email" "$admin_password" "$license_key")"

	echo "$platform_json" | python3 -c 'import json,sys; c=json.load(sys.stdin)["config"]; print(json.dumps(c))' > platform_config.json

	bash scripts/dev/stack.sh single-vps
	wait_control_health

	local port token url
	port="$(read_env_var MANAGEMENT_PORT)"
	port="${port:-8188}"
	token="$(read_env_var INSTALL_BOOTSTRAP_TOKEN)"
	url="http://127.0.0.1:${port}/api/v1/settings/platform/bootstrap"

	curl -sf -X POST \
		-H "Content-Type: application/json" \
		-H "X-Install-Token: ${token}" \
		-d "$platform_json" \
		"$url"

	echo ""
	echo "Control URL: http://127.0.0.1:${port}"
	if [[ -n "$tracking_domain" ]]; then
		echo "Click URL template: https://${tracking_domain}/click?campaign_id={campaign_id}&sub1={sub1}"
	fi
}

cmd_status() {
	check_docker
	bash scripts/dev/stack.sh status
	local port
	port="$(read_env_var MANAGEMENT_PORT)"
	port="${port:-8188}"
	if curl -sf "http://127.0.0.1:${port}/health" >/dev/null; then
		echo "control: healthy (http://127.0.0.1:${port})"
	else
		echo "control: not reachable (http://127.0.0.1:${port})"
	fi
}

cmd_apply() {
	check_docker
	ensure_env
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
	go run ./cmd/installer apply
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
*)
	echo "usage: $0 {up|status|apply}" >&2
	exit 2
	;;
esac
