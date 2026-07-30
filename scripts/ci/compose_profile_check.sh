#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "compose profile config: single_vps"
docker compose --profile single_vps config >/dev/null

echo "compose profile config: ingest_only"
docker compose --profile ingest_only config >/dev/null
if docker compose --profile ingest_only config --services | grep -qx clickhouse; then
	echo "ingest_only profile must not include clickhouse" >&2
	exit 1
fi
if docker compose --profile ingest_only config --services | grep -qx db-payment; then
	echo "ingest_only profile must not include db-payment" >&2
	exit 1
fi

echo "compose profile config: network_operator"
docker compose --profile network_operator config >/dev/null

echo "compose profile config: analytics_ml"
docker compose --profile analytics_ml --profile fraud-scorer config >/dev/null
if ! docker compose --profile analytics_ml --profile fraud-scorer config --services | grep -qx clickhouse; then
	echo "analytics_ml profile must include clickhouse" >&2
	exit 1
fi
if ! docker compose --profile analytics_ml --profile fraud-scorer config --services | grep -qx ivt-detector; then
	echo "analytics_ml profile must include ivt-detector" >&2
	exit 1
fi
if ! docker compose --profile analytics_ml --profile fraud-scorer config --services | grep -qx fraud-scorer; then
	echo "analytics_ml profile must include fraud-scorer" >&2
	exit 1
fi

echo "compose_profile_check: ok"
