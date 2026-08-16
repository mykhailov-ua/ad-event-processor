#!/usr/bin/env bash
# Render Caddyfile for compose ingress profile.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

GEN_DIR="$ROOT/deploy/ingress/caddy/generated"
OUT="$GEN_DIR/Caddyfile"
mkdir -p "$GEN_DIR"

if [[ -f deploy/installer/install.env ]]; then
	set -a
	# shellcheck disable=SC1090
	source deploy/installer/install.env
	set +a
fi

if [[ -f .env ]]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

TRACKING_DOMAIN="${TRACKING_DOMAIN:-}"
ADMIN_DOMAIN="${ADMIN_DOMAIN:-}"
CADDY_ACME_EMAIL="${CADDY_ACME_EMAIL:-}"
INGRESS_TLS_MODE="${INGRESS_TLS_MODE:-acme}"
INGRESS_TLS_ISSUER="${INGRESS_TLS_ISSUER:-acme}"
INGRESS_HTTPS_PORT="${INGRESS_HTTPS_PORT:-443}"
TLS_CERT="${INGRESS_TLS_CERT:-/etc/caddy/certs/fullchain.pem}"
TLS_KEY="${INGRESS_TLS_KEY:-/etc/caddy/certs/privkey.pem}"
MANAGEMENT_PORT="${MANAGEMENT_PORT:-8188}"
CADDY_TLS_ASK_TOKEN="${CADDY_TLS_ASK_TOKEN:-}"

if [[ -z "$TRACKING_DOMAIN" ]]; then
	echo "render_ingress: TRACKING_DOMAIN is required when INGRESS_ENABLED=1" >&2
	exit 1
fi

tls_lines() {
	if [[ "$INGRESS_TLS_MODE" == "manual" ]]; then
		echo "	tls ${TLS_CERT} ${TLS_KEY}"
	fi
}

ask_url() {
	local q=""
	if [[ -n "$CADDY_TLS_ASK_TOKEN" ]]; then
		q="?token=${CADDY_TLS_ASK_TOKEN}"
	fi
	# Caddy appends ?domain=<SNI> (or &domain= when q is non-empty).
	echo "http://127.0.0.1:${MANAGEMENT_PORT}/api/v1/ops/domains/tls-allowed${q}"
}

{
	echo "{"
	if [[ -n "$CADDY_ACME_EMAIL" ]]; then
		echo "	email ${CADDY_ACME_EMAIL}"
	fi
	if [[ "$INGRESS_TLS_MODE" == "on_demand" ]]; then
		echo "	on_demand_tls {"
		echo "		ask $(ask_url)"
		echo "		interval 2m"
		echo "		burst 5"
		echo "	}"
	fi
	echo "}"
	echo ""

	if [[ "$INGRESS_TLS_MODE" == "on_demand" ]]; then
		echo ":${INGRESS_HTTPS_PORT} {"
		echo "	tls {"
		echo "		on_demand"
		if [[ "$INGRESS_TLS_ISSUER" == "internal" ]]; then
			echo "		issuer internal"
		fi
		echo "	}"
		if [[ -n "$ADMIN_DOMAIN" ]]; then
			echo "	@admin host ${ADMIN_DOMAIN}"
			echo "	handle @admin {"
			echo "		reverse_proxy 127.0.0.1:${MANAGEMENT_PORT}"
			echo "	}"
		fi
		echo "	handle {"
		echo "		reverse_proxy 127.0.0.1:8181"
		echo "	}"
		echo "}"
	else
		echo "${TRACKING_DOMAIN} {"
		tls_lines
		echo "	reverse_proxy 127.0.0.1:8181"
		echo "}"
		if [[ -n "$ADMIN_DOMAIN" ]]; then
			echo ""
			echo "${ADMIN_DOMAIN} {"
			tls_lines
			echo "	reverse_proxy 127.0.0.1:${MANAGEMENT_PORT}"
			echo "}"
		fi
	fi
} >"$OUT"

echo "render_ingress: wrote $OUT (INGRESS_TLS_MODE=${INGRESS_TLS_MODE})"
echo "  tracking: https://${TRACKING_DOMAIN}"
if [[ -n "$ADMIN_DOMAIN" ]]; then
	echo "  admin:    https://${ADMIN_DOMAIN}"
fi
if [[ "$INGRESS_TLS_MODE" == "on_demand" ]]; then
	echo "  on_demand ask: $(ask_url)"
fi
