#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

MODEL_DIR="${ROOT}/model"
export PYTHONPATH="${MODEL_DIR}"
ARTIFACT_DIR="${FRAUD_ARTIFACT_DIR:-var/fraudscore/artifacts}"
MODEL_PATH="${ARTIFACT_DIR}/model.txt"
FIXTURES_DIR="${FRAUD_FIXTURES_DIR:-var/fraudscore/fixtures}"

mkdir -p "${ARTIFACT_DIR}" "${FIXTURES_DIR}"

echo "fraudtrain: bootstrap artifacts (ephemeral)"
python3 -m train.artifact_bootstrap bootstrap

echo "fraudtrain: sync ML fixtures (tracked + ephemeral)"
python3 -m train.fixture_generator

if [[ ! -f "${MODEL_PATH}" ]]; then
  echo "fraudtrain: missing ${MODEL_PATH} after bootstrap" >&2
  exit 1
fi

echo "fraudtrain: Python contract tests"
if python3 -c "import pytest" 2> /dev/null; then
  (cd "${MODEL_DIR}" && python3 -m pytest tests/ -q)
else
  echo "fraudtrain: skip pytest (pip install -r model/requirements.txt)" >&2
  exit 1
fi

echo "fraudtrain: Go feature_spec golden"
go test ./internal/fraud/ -run 'TestFeatureSpec' -count=1

echo "fraudtrain: Go scoring_policy parity"
go test ./internal/fraud/ -run 'TestScoringPolicyParity' -count=1

echo "fraudtrain: Go policy_config parity"
go test ./internal/fraud/ -run 'TestPolicyConfigParity' -count=1

echo "fraudtrain: Go ml-validate"
go test ./cmd/ml-validate/... -count=1
go run ./cmd/ml-validate -model "${MODEL_PATH}" -fixtures "${FIXTURES_DIR}"

echo "fraudtrain: Go ml-replay"
go test ./cmd/ml-replay/... -count=1
go run ./cmd/ml-replay -model "${MODEL_PATH}" -fixtures "${FIXTURES_DIR}" > /dev/null

if python3 -c "import lightgbm" 2> /dev/null; then
  echo "fraudtrain: Python artifact validate"
  python3 -m train.artifact_bootstrap validate --model "${MODEL_PATH}"
  echo "fraudtrain: fit smoke"
  python3 -c "
from pathlib import Path
from train.labeled_dataset import write_synthetic_dataset
write_synthetic_dataset(Path('var/fraudscore/training/fit_smoke.csv'), count=1500)
"
  python3 -m data.manual_labels_export

  FRAUD_TRAIN_DATASET=var/fraudscore/training/fit_smoke.csv \
    FRAUD_FIT_MIN_ROWS=500 \
    FRAUD_FIT_BOOST_ROUNDS=30 \
    python3 -m train.artifact_bootstrap fit-validate
else
  echo "fraudtrain: skip Python validate (pip install -r model/requirements.txt)"
fi

if python3 -c "import clickhouse_connect" 2> /dev/null; then
  if python3 -c "
from data.clickhouse_client import connect_client, ping_client
client = connect_client()
import sys
sys.exit(0 if ping_client(client) else 1)
" 2> /dev/null; then
    echo "fraudtrain: features_export smoke"
    python3 -m data.features_export --smoke
    echo "fraudtrain: evaluate smoke"
    python3 -m eval.evaluate --format json --hours 1
  else
    echo "fraudtrain: skip CH smokes (ClickHouse unreachable)"
  fi
else
  echo "fraudtrain: skip CH smokes (clickhouse-connect not installed)"
fi

if command -v ruff > /dev/null 2>&1; then
  ruff check "${MODEL_DIR}/"
fi

echo "fraudtrain: OK"
