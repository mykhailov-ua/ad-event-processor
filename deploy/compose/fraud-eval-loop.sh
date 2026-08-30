#!/bin/sh
# Fraud shadow eval loop entrypoint (fraud-shadow-eval service).
# Built via model/Dockerfile.evaluate; profiles analytics_ml, fraud-eval.
# Cross-ref: deploy/DEPLOY.md, deploy/compose/docker-compose.yaml fraud-shadow-eval.
#
# Execution context:
# - Long-running container; cd /app/model then python3 -m eval.evaluate --format both.
# - Reads CH/DB DSN from container env (CH_DSN, DB_DSN, FRAUD_EVAL_HOURS).
#
# Env deps:
# - FRAUD_EVAL_INTERVAL_SEC (default 21600, 6 h between eval runs).
# - CH_DSN, DB_DSN required; artifacts under FRAUD_ARTIFACT_DIR mount.
#
# Exit codes:
# - Loop never exits on eval failure; logs stderr and sleeps FRAUD_EVAL_INTERVAL_SEC.
# - set -eu: shell error (missing python, bad cd) exits non-zero before loop.
#
# Forbidden:
# - Do not run on ingest-only stack (requires clickhouse + analytics_ml profile).
#
# Verify:
# bash scripts/dev/stack/stack.sh analytics-ml
# docker compose --profile fraud-eval ps fraud-shadow-eval
set -eu

INTERVAL_SEC="${FRAUD_EVAL_INTERVAL_SEC:-21600}"
cd /app/model

while true; do
  echo "fraud-shadow-eval: starting eval.evaluate"
  if python3 -m eval.evaluate --format both; then
    echo "fraud-shadow-eval: completed successfully"
  else
    echo "fraud-shadow-eval: eval.evaluate failed" >&2
  fi
  echo "fraud-shadow-eval: sleeping ${INTERVAL_SEC}s"
  sleep "${INTERVAL_SEC}"
done
