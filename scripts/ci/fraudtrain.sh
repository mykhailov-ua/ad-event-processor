#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

ARTIFACT_DIR="${FRAUD_ARTIFACT_DIR:-var/fraudscore/artifacts}"
MODEL_PATH="${ARTIFACT_DIR}/model.txt"
FIXTURES_DIR="testdata/ml"

mkdir -p "${ARTIFACT_DIR}"

echo "fraudtrain: bootstrap artifacts (ephemeral)"
if python3 -c "import lightgbm" 2>/dev/null; then
  python3 fraudtrain/artifact_bootstrap.py bootstrap
  python3 fraudtrain/fixture_generator.py
else
  python3 fraudtrain/artifact_bootstrap.py bootstrap
  echo "fraudtrain: skip fixture_generator (lightgbm not installed)"
fi

if [[ ! -f "${MODEL_PATH}" ]]; then
  echo "fraudtrain: missing ${MODEL_PATH} after bootstrap" >&2
  exit 1
fi

echo "fraudtrain: Go ml-validate"
go test ./cmd/ml-validate/... -count=1
go run ./cmd/ml-validate -model "${MODEL_PATH}" -fixtures "${FIXTURES_DIR}"

echo "fraudtrain: Go ml-replay"
go test ./cmd/ml-replay/... -count=1
go run ./cmd/ml-replay -model "${MODEL_PATH}" -fixtures "${FIXTURES_DIR}" > /dev/null

if python3 -c "import lightgbm" 2>/dev/null; then
  echo "fraudtrain: Python artifact validate"
  python3 fraudtrain/artifact_bootstrap.py validate --model "${MODEL_PATH}"
  echo "fraudtrain: fit smoke"
  python3 -c "
from pathlib import Path
import sys
sys.path.insert(0, 'fraudtrain')
from labeled_dataset import write_synthetic_dataset
write_synthetic_dataset(Path('var/fraudscore/training/fit_smoke.csv'), count=1500)
"
  python3 fraudtrain/manual_labels_export.py || true

  FRAUD_TRAIN_DATASET=var/fraudscore/training/fit_smoke.csv \
    FRAUD_FIT_MIN_ROWS=500 \
    FRAUD_FIT_BOOST_ROUNDS=30 \
    python3 fraudtrain/artifact_bootstrap.py fit-validate
else
  echo "fraudtrain: skip Python validate (pip install -r fraudtrain/requirements.txt)"
fi

if python3 -c "import clickhouse_connect" 2>/dev/null; then
  echo "fraudtrain: features_export smoke"
  python3 fraudtrain/features_export.py --smoke --allow-offline
else
  echo "fraudtrain: skip CH export smoke (clickhouse-connect not installed)"
fi

echo "fraudtrain: evaluate smoke"
python3 fraudtrain/evaluate.py --allow-offline --format json --hours 1

if command -v ruff >/dev/null 2>&1; then
  ruff check fraudtrain/
fi

echo "fraudtrain: OK"
