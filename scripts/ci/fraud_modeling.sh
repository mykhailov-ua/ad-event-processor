#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ARTIFACT_DIR="${FRAUD_ARTIFACT_DIR:-var/fraudscore/artifacts}"
MODEL_PATH="${ARTIFACT_DIR}/model.txt"
FIXTURES_DIR="testdata/ml"

mkdir -p "${ARTIFACT_DIR}"

echo "fraud_modeling: bootstrap artifacts (ephemeral)"
if python3 -c "import lightgbm" 2>/dev/null; then
  python3 fraud_modeling/artifact_bootstrap.py bootstrap
  python3 fraud_modeling/fixture_generator.py
else
  python3 fraud_modeling/artifact_bootstrap.py bootstrap
  echo "fraud_modeling: skip fixture_generator (lightgbm not installed)"
fi

if [[ ! -f "${MODEL_PATH}" ]]; then
  echo "fraud_modeling: missing ${MODEL_PATH} after bootstrap" >&2
  exit 1
fi

echo "fraud_modeling: Go ml-validate"
go test ./cmd/ml-validate/... -count=1
go run ./cmd/ml-validate -model "${MODEL_PATH}" -fixtures "${FIXTURES_DIR}"

echo "fraud_modeling: Go ml-replay"
go test ./cmd/ml-replay/... -count=1
go run ./cmd/ml-replay -model "${MODEL_PATH}" -fixtures "${FIXTURES_DIR}" > /dev/null

if python3 -c "import lightgbm" 2>/dev/null; then
  echo "fraud_modeling: Python artifact validate"
  python3 fraud_modeling/artifact_bootstrap.py validate --model "${MODEL_PATH}"
  echo "fraud_modeling: fit smoke"
  python3 -c "
from pathlib import Path
import sys
sys.path.insert(0, 'fraud_modeling')
from labeled_dataset import write_synthetic_dataset
write_synthetic_dataset(Path('var/fraudscore/training/fit_smoke.csv'), count=1500)
"
  python3 fraud_modeling/manual_labels_export.py || true

  FRAUD_TRAIN_DATASET=var/fraudscore/training/fit_smoke.csv \
    FRAUD_FIT_MIN_ROWS=500 \
    FRAUD_FIT_BOOST_ROUNDS=30 \
    python3 fraud_modeling/artifact_bootstrap.py fit-validate
else
  echo "fraud_modeling: skip Python validate (pip install -r fraud_modeling/requirements.txt)"
fi

if python3 -c "import clickhouse_connect" 2>/dev/null; then
  echo "fraud_modeling: features_export smoke"
  python3 fraud_modeling/features_export.py --smoke --allow-offline
else
  echo "fraud_modeling: skip CH export smoke (clickhouse-connect not installed)"
fi

echo "fraud_modeling: evaluate smoke"
python3 fraud_modeling/evaluate.py --allow-offline --format json --hours 1

if command -v ruff >/dev/null 2>&1; then
  ruff check fraud_modeling/
fi

echo "fraud_modeling: OK"
