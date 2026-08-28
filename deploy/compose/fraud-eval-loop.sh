#!/bin/sh
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
