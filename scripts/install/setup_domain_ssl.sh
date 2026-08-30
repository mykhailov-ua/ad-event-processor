#!/usr/bin/env bash
# Role: Obtain Let's Encrypt cert and wire DOMAIN into install.env for Caddy ingress.
# Execution context: Production host with DNS pointing at machine; requires certbot.
# Env knobs: DOMAIN (arg or env); TLS_EMAIL from install.env or .env.
# Verify: bash scripts/install/setup_domain_ssl.sh track.example.com --help 2>&1 | head -1
set -euo pipefail

HOST="${1:-${DOMAIN:-}}"
if [[ -z "$HOST" ]]; then
  echo "usage: setup_domain_ssl.sh <hostname>" >&2
  exit 2
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

if [[ -f deploy/installer/install.env ]]; then
  set -a

  source deploy/installer/install.env
  set +a
fi
if [[ -f .env ]]; then
  set -a

  source .env
  set +a
fi

EMAIL="${CADDY_ACME_EMAIL:-}"
WEBROOT="${DOMAIN_SSL_WEBROOT:-/var/www/certbot}"

if command -v certbot > /dev/null 2>&1; then
  mkdir -p "$WEBROOT"
  args=(certonly --non-interactive --agree-tos --webroot -w "$WEBROOT" -d "$HOST")
  if [[ -n "$EMAIL" ]]; then
    args+=(--email "$EMAIL")
  else
    args+=(--register-unsafely-without-email)
  fi
  echo "setup_domain_ssl: running certbot for ${HOST}"
  certbot "${args[@]}"
elif command -v caddy > /dev/null 2>&1 && [[ -x scripts/install/render_ingress.sh ]]; then
  echo "setup_domain_ssl: certbot not found; rendering Caddy ingress with ACME for ${HOST}"
  export TRACKING_DOMAIN="${TRACKING_DOMAIN:-$HOST}"
  bash scripts/install/render_ingress.sh
  if command -v systemctl > /dev/null 2>&1; then
    systemctl reload caddy 2> /dev/null || systemctl restart caddy 2> /dev/null || true
  fi
else
  echo "setup_domain_ssl: neither certbot nor caddy ingress available" >&2
  exit 1
fi

if [[ -x scripts/install/render_ingress.sh ]]; then
  bash scripts/install/render_ingress.sh || true
fi

echo "setup_domain_ssl: completed for ${HOST}"
