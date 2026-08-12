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

echo "compose profile config: resilience"
docker compose --profile resilience config >/dev/null
if ! docker compose --profile resilience config --services | grep -qx control; then
	echo "resilience profile must include control" >&2
	exit 1
fi
if docker compose --profile resilience config --services | grep -qxE 'management|payment|billing|notifier'; then
	echo "resilience profile must not include legacy sidecars" >&2
	exit 1
fi

echo "compose profile config: crypto"
docker compose --profile crypto config >/dev/null
if ! docker compose --profile crypto config --services | grep -qx control; then
	echo "crypto profile must include control" >&2
	exit 1
fi

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

echo "compose profile config: enterprise-xdp"
docker compose --profile enterprise-xdp config >/dev/null
if ! docker compose --profile enterprise-xdp config --services | grep -qx edge-xdp; then
	echo "enterprise-xdp profile must include edge-xdp" >&2
	exit 1
fi
if docker compose --profile single_vps config --services | grep -qx edge-xdp; then
	echo "single_vps profile must not include edge-xdp" >&2
	exit 1
fi
if ! docker compose --profile enterprise-xdp config 2>/dev/null | grep -q 'privileged: true'; then
	echo "enterprise-xdp edge-xdp service must be privileged" >&2
	exit 1
fi
if ! docker compose --profile enterprise-xdp config 2>/dev/null | grep -q 'network_mode: host'; then
	echo "enterprise-xdp edge-xdp service must use host network" >&2
	exit 1
fi

echo "compose_profile_check: ok"
