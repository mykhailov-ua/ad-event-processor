#!/usr/bin/env bash
# Seed local dev admin: .env defaults, platform bootstrap, admin user.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

NO_UP=0
for arg in "$@"; do
	case "$arg" in
	--no-up) NO_UP=1 ;;
	-h | --help)
		echo "usage: $0 [--no-up]"
		echo "  Ensures dev admin credentials in .env, bootstraps platform if needed,"
		echo "  creates admin user when platform is already bootstrapped."
		exit 0
		;;
	*)
		echo "seed-admin: unknown argument: $arg" >&2
		exit 2
		;;
	esac
done

DEFAULT_EMAIL="admin@test.local"
DEFAULT_PASSWORD="Password123!"

read_env_var() {
	local key="$1"
	if [[ -f .env ]]; then
		grep -m1 "^${key}=" .env 2>/dev/null | cut -d= -f2- || true
	else
		echo ""
	fi
}

log() { printf 'seed-admin: %s\n' "$*"; }
die() { printf 'seed-admin: ERROR: %s\n' "$*" >&2; exit 1; }

set_env_var() {
	local key="$1" value="$2"
	if [[ ! -f .env ]]; then
		cp .env.example .env
	fi
	if ! grep -q "^${key}=" .env; then
		echo "${key}=${value}" >> .env
		return 0
	fi
	local current
	current="$(read_env_var "$key")"
	if [[ -z "$current" ]]; then
		sed -i "s/^${key}=.*/${key}=${value}/" .env
		return 0
	fi
	return 1
}

ensure_env() {
	local token_generated=0
	if [[ ! -f .env ]]; then
		cp .env.example .env
	fi
	if set_env_var "ADMIN_BOOTSTRAP_EMAIL" "$DEFAULT_EMAIL"; then
		log "set ADMIN_BOOTSTRAP_EMAIL=$DEFAULT_EMAIL in .env"
	fi
	if set_env_var "ADMIN_BOOTSTRAP_PASSWORD" "$DEFAULT_PASSWORD"; then
		log "set ADMIN_BOOTSTRAP_PASSWORD in .env (dev default)"
	fi
	if ! grep -q '^INSTALL_BOOTSTRAP_TOKEN=' .env; then
		echo "INSTALL_BOOTSTRAP_TOKEN=$(openssl rand -hex 32)" >> .env
		token_generated=1
	elif [[ -z "$(read_env_var INSTALL_BOOTSTRAP_TOKEN)" ]]; then
		token="$(openssl rand -hex 32)"
		sed -i "s/^INSTALL_BOOTSTRAP_TOKEN=.*/INSTALL_BOOTSTRAP_TOKEN=${token}/" .env
		token_generated=1
	fi
	if [[ "$token_generated" -eq 1 ]]; then
		log "generated INSTALL_BOOTSTRAP_TOKEN in .env"
	fi
	echo "$token_generated"
}

wait_control() {
	local url="$1"
	local i=0
	while [[ $i -lt 120 ]]; do
		if curl -sf "${url}/health" >/dev/null 2>&1; then
			return 0
		fi
		sleep 2
		i=$((i + 1))
	done
	return 1
}

bootstrap_platform() {
	local url="$1" token="$2" email="$3" password="$4"
	log "bootstrapping platform"
	local body http_code
	body="$(curl -s -w '\n%{http_code}' -X POST "${url}/api/v1/settings/platform/bootstrap" \
		-H "Content-Type: application/json" \
		-H "X-Install-Token: ${token}" \
		-d "$(jq -n \
			--arg email "$email" \
			--arg password "$password" \
			'{
				config: {
					tracking_domain: "track.local",
					default_currency: "USD",
					timezone: "UTC",
					ingress_schema: "espx_native",
					profile: "single_vps",
					network_interface: "eth0",
					telemetry_enabled: true,
					edge_xdp: false,
					stripe: { enabled: false }
				},
				admin_email: $email,
				admin_password: $password
			}')")"
	http_code="${body##*$'\n'}"
	body="${body%$'\n'*}"
	if [[ "$http_code" != "200" ]]; then
		die "bootstrap failed (HTTP ${http_code}): ${body}"
	fi
}

load_env() {
	MANAGEMENT_PORT="$(read_env_var MANAGEMENT_PORT)"
	ADMIN_BOOTSTRAP_EMAIL="$(read_env_var ADMIN_BOOTSTRAP_EMAIL)"
	ADMIN_BOOTSTRAP_PASSWORD="$(read_env_var ADMIN_BOOTSTRAP_PASSWORD)"
	INSTALL_BOOTSTRAP_TOKEN="$(read_env_var INSTALL_BOOTSTRAP_TOKEN)"
	export DB_DSN="$(read_env_var DB_DSN)"
}

ensure_admin_user() {
	local email="$1" password="$2"
	if go run ./cmd/admin --env-path .env user get "$email" >/dev/null 2>&1; then
		log "admin user exists: $email"
		return 0
	fi
	log "creating admin user: $email"
	go run ./cmd/admin --env-path .env user create \
		--email "$email" \
		--password "$password" \
		--role A
}

token_generated="$(ensure_env)"
load_env

CONTROL_URL="${CONTROL_URL:-http://127.0.0.1:${MANAGEMENT_PORT:-8188}}"
EMAIL="${ADMIN_BOOTSTRAP_EMAIL:-$DEFAULT_EMAIL}"
PASSWORD="${ADMIN_BOOTSTRAP_PASSWORD:-$DEFAULT_PASSWORD}"
INSTALL_TOKEN="${INSTALL_BOOTSTRAP_TOKEN:-}"

[[ -n "$EMAIL" && -n "$PASSWORD" ]] || die "ADMIN_BOOTSTRAP_EMAIL and ADMIN_BOOTSTRAP_PASSWORD required"
[[ -n "$INSTALL_TOKEN" ]] || die "INSTALL_BOOTSTRAP_TOKEN missing after ensure_env"

control_up=0
if curl -sf "${CONTROL_URL}/health" >/dev/null 2>&1; then
	control_up=1
fi

if [[ "$NO_UP" -eq 0 && "$control_up" -eq 0 ]]; then
	log "control not healthy; starting ingest-only stack"
	CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 \
		CONTROL_ENABLE_NOTIFIER=0 CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
		docker compose --profile ingest_only up -d db redis-0 control
	if [[ "$token_generated" -eq 1 ]]; then
		docker compose --profile ingest_only up -d --force-recreate control
	fi
	wait_control "$CONTROL_URL" || die "control did not become healthy"
	control_up=1
	elif [[ "$token_generated" -eq 1 && "$control_up" -eq 1 ]]; then
	log "INSTALL_BOOTSTRAP_TOKEN was generated; recreating redis-0 and control to load it"
	CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 \
		CONTROL_ENABLE_NOTIFIER=0 CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
		docker compose --profile ingest_only up -d --force-recreate redis-0 control
	wait_control "$CONTROL_URL" || die "control did not become healthy after recreate"
fi

# Redis and control share host networking; recreate control without redis leaves stale sessions.
if [[ "$control_up" -eq 1 ]] && ! curl -sf "${CONTROL_URL}/health" >/dev/null 2>&1; then
	log "control unhealthy; recreating redis-0 and control"
	CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 \
		CONTROL_ENABLE_NOTIFIER=0 CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
		docker compose --profile ingest_only up -d --force-recreate redis-0 control
	wait_control "$CONTROL_URL" || die "control did not become healthy after redis recreate"
fi

if [[ "$control_up" -eq 0 ]]; then
	die "control not reachable at ${CONTROL_URL} (use without --no-up to start stack)"
fi

meta="$(curl -sf "${CONTROL_URL}/api/v1/meta" || true)"
[[ -n "$meta" ]] || die "GET /api/v1/meta failed"

if echo "$meta" | grep -q '"bootstrap_complete":true'; then
	log "platform already bootstrapped"
	ensure_admin_user "$EMAIL" "$PASSWORD"
else
	bootstrap_platform "$CONTROL_URL" "$INSTALL_TOKEN" "$EMAIL" "$PASSWORD"
	log "platform bootstrapped with admin $EMAIL"
fi

cat <<EOF

seed-admin: done
  Control:  ${CONTROL_URL}
  Login:    ${CONTROL_URL}/login
  Email:    ${EMAIL}
  Password: ${PASSWORD}

EOF
