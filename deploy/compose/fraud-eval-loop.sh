#!/bin/sh
set -eu

INTERVAL_SEC="${FRAUD_EVAL_INTERVAL_SEC:-21600}"
cd /app/model

while true; do
  echo "fraud-shadow-eval: starting evaluate.py"
  if python3 evaluate.py --format both; then
    echo "fraud-shadow-eval: completed successfully"
  else
    echo "fraud-shadow-eval: evaluate.py failed" >&2
  fi
  echo "fraud-shadow-eval: sleeping ${INTERVAL_SEC}s"
  sleep "${INTERVAL_SEC}"
done
