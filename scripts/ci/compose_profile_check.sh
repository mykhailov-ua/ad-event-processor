#!/usr/bin/env bash
set -euo pipefail

# Role: Compose profile consistency check.
# Execution context: CI full_test.
# Invariants/contracts enforced: Profile/env mismatch fails.
# Verify: bash scripts/ci/compose_profile_check.sh
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

ENV_FILE=()
if [[ -f "$ROOT/.env.example" ]]; then
  ENV_FILE=(--env-file "$ROOT/.env.example")
fi

echo "compose default config (no cpu-isolation overlay)"
docker compose "${ENV_FILE[@]}" config > /dev/null
if ! docker compose "${ENV_FILE[@]}" config --services | grep -qx tracker-0; then
  echo "default compose must include tracker-0 without cpu-isolation profile" >&2
  exit 1
fi
if ! docker compose "${ENV_FILE[@]}" config --services | grep -qx broker; then
  echo "default compose must include broker (CH_INGEST_SOURCE defaults to broker)" >&2
  exit 1
fi
if ! grep -q 'create_host_path: false' "$ROOT/deploy/compose/docker-compose.yaml"; then
  echo "compose must set create_host_path: false on runtime bind mounts" >&2
  exit 1
fi

echo "compose release overlay (image-only services)"
RELEASE_ENV_FILE="$ROOT/.cache/compose-release-ci.env"
mkdir -p "$ROOT/.cache"
printf 'AD_EVENT_PROCESSOR_APP_IMAGE=ghcr.io/example/ad-event-processor:ci-test\n' > "$RELEASE_ENV_FILE"
RELEASE_CFG="$ROOT/.cache/compose-release-ci.yaml"
docker compose -f "$ROOT/deploy/compose/docker-compose.yaml" \
  -f "$ROOT/deploy/compose/docker-compose.release.yaml" \
  "${ENV_FILE[@]}" --env-file "$RELEASE_ENV_FILE" config > "$RELEASE_CFG" 2> /dev/null
if ! grep -q 'image: ghcr.io/example/ad-event-processor:ci-test' "$RELEASE_CFG"; then
  echo "release overlay must set AD_EVENT_PROCESSOR_APP_IMAGE on app services" >&2
  exit 1
fi
if awk '/^  tracker-0:/{p=1} p&&/^  [a-z]/{if(!/^  tracker-0:/)exit} p' "$RELEASE_CFG" | grep -q 'build:'; then
  echo "release overlay must remove build from tracker-0" >&2
  exit 1
fi

echo "compose load-test overlay (nginx depends on two trackers)"
docker compose -f "$ROOT/deploy/compose/docker-compose.yaml" \
  -f "$ROOT/deploy/compose/docker-compose.load-test.yaml" \
  "${ENV_FILE[@]}" config > /dev/null
if docker compose -f "$ROOT/deploy/compose/docker-compose.yaml" \
  -f "$ROOT/deploy/compose/docker-compose.load-test.yaml" \
  "${ENV_FILE[@]}" config 2> /dev/null | grep -A20 '^  nginx:' | grep -q 'tracker-2:'; then
  echo "load-test overlay must not require tracker-2/3 in nginx depends_on" >&2
  exit 1
fi
if ! grep -q 'LOCAL_QUOTA_MODE: live' "$ROOT/deploy/compose/docker-compose.load-test.yaml"; then
  echo "load-test overlay must set LOCAL_QUOTA_MODE=live on high-QPS trackers" >&2
  exit 1
fi
if ! grep -q 'QUOTA_MODE: live' "$ROOT/deploy/compose/docker-compose.load-test.yaml"; then
  echo "load-test overlay must set QUOTA_MODE=live with LOCAL_QUOTA_MODE=live" >&2
  exit 1
fi

echo "compose cpu-isolation overlay + profile"
docker compose -f "$ROOT/deploy/compose/docker-compose.yaml" \
  -f "$ROOT/deploy/compose/docker-compose.cpu-isolation.yaml" \
  "${ENV_FILE[@]}" --profile cpu-isolation config > /dev/null
if ! docker compose -f "$ROOT/deploy/compose/docker-compose.yaml" \
  -f "$ROOT/deploy/compose/docker-compose.cpu-isolation.yaml" \
  "${ENV_FILE[@]}" --profile cpu-isolation config --services | grep -qx tracker-0; then
  echo "cpu-isolation profile must include tracker-0" >&2
  exit 1
fi

echo "compose profile config: single_vps"
docker compose --profile single_vps config > /dev/null

echo "compose profile config: ingest_only"
docker compose --profile ingest_only config > /dev/null
if docker compose --profile ingest_only config --services | grep -qx clickhouse; then
  echo "ingest_only profile must not include clickhouse" >&2
  exit 1
fi
if docker compose --profile ingest_only config --services | grep -qx db-payment; then
  echo "ingest_only profile must not include db-payment" >&2
  exit 1
fi

echo "compose profile config: network_operator"
docker compose --profile network_operator config > /dev/null

echo "compose profile config: resilience"
docker compose --profile resilience config > /dev/null
if ! docker compose --profile resilience config --services | grep -qx control; then
  echo "resilience profile must include control" >&2
  exit 1
fi
if docker compose --profile resilience config --services | grep -qxE 'management|payment|billing|notifier'; then
  echo "resilience profile must not include legacy sidecars" >&2
  exit 1
fi

echo "compose profile config: crypto"
docker compose --profile crypto config > /dev/null
if ! docker compose --profile crypto config --services | grep -qx control; then
  echo "crypto profile must include control" >&2
  exit 1
fi

echo "compose profile config: analytics_ml"
docker compose --profile analytics_ml --profile fraud-scorer config > /dev/null
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
docker compose --profile enterprise-xdp config > /dev/null
if ! docker compose --profile enterprise-xdp config --services | grep -qx edge-xdp; then
  echo "enterprise-xdp profile must include edge-xdp" >&2
  exit 1
fi
if docker compose --profile single_vps config --services | grep -qx edge-xdp; then
  echo "single_vps profile must not include edge-xdp" >&2
  exit 1
fi
if ! docker compose --profile enterprise-xdp config 2> /dev/null | grep -q 'privileged: true'; then
  echo "enterprise-xdp edge-xdp service must be privileged" >&2
  exit 1
fi
if ! docker compose --profile enterprise-xdp config 2> /dev/null | grep -q 'network_mode: host'; then
  echo "enterprise-xdp edge-xdp service must use host network" >&2
  exit 1
fi

echo "compose_profile_check: ok"
