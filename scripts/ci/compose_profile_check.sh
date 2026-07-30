#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "compose profile config: single_vps"
docker compose --profile single_vps config >/dev/null

echo "compose profile config: ingest_only"
docker compose --profile ingest_only config >/dev/null

echo "compose profile config: network_operator"
docker compose --profile network_operator config >/dev/null

echo "compose profile config: analytics_ml"
docker compose --profile analytics_ml --profile fraud-scorer config >/dev/null

echo "compose_profile_check: ok"
