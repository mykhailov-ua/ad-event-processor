#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ENV_FILE="${ENV_FILE:-.env}"
if [[ ! -f "$ENV_FILE" ]]; then
  cp .env.example "$ENV_FILE"
  if sed --version > /dev/null 2>&1; then
    sed -i 's/your_redis_password_here/sentinel_fault_test/' "$ENV_FILE"
  else
    sed -i '' 's/your_redis_password_here/sentinel_fault_test/' "$ENV_FILE"
  fi
fi

set -a

. "$ENV_FILE"
set +a

if [[ -z "${REDIS_PASSWORD:-}" ]]; then
  echo "sentinel_failover_env: REDIS_PASSWORD required in $ENV_FILE" >&2
  exit 1
fi

echo "sentinel_failover_env: ok (ENV_FILE=$ENV_FILE)"
