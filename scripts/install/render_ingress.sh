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
TLS_CERT="${INGRESS_TLS_CERT:-/etc/caddy/certs/fullchain.pem}"
TLS_KEY="${INGRESS_TLS_KEY:-/etc/caddy/certs/privkey.pem}"

if [[ -z "$TRACKING_DOMAIN" ]]; then
	echo "render_ingress: TRACKING_DOMAIN is required when INGRESS_ENABLED=1" >&2
	exit 1
fi

tls_lines() {
	if [[ "$INGRESS_TLS_MODE" == "manual" ]]; then
		echo "	tls ${TLS_CERT} ${TLS_KEY}"
	fi
}

{
	echo "{"
	if [[ -n "$CADDY_ACME_EMAIL" ]]; then
		echo "	email ${CADDY_ACME_EMAIL}"
	fi
	echo "}"
	echo ""
	echo "${TRACKING_DOMAIN} {"
	tls_lines
	echo "	reverse_proxy 127.0.0.1:8181"
	echo "}"
	if [[ -n "$ADMIN_DOMAIN" ]]; then
		echo ""
		echo "${ADMIN_DOMAIN} {"
		tls_lines
		echo "	reverse_proxy 127.0.0.1:8188"
		echo "}"
	fi
} >"$OUT"

echo "render_ingress: wrote $OUT"
echo "  tracking: https://${TRACKING_DOMAIN}"
if [[ -n "$ADMIN_DOMAIN" ]]; then
	echo "  admin:    https://${ADMIN_DOMAIN}"
fi
