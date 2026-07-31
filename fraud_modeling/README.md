# fraud_modeling

Offline fraud ML: train artifacts, calibrate policy, run benchmarks. Production inference is Go (`cmd/fraud-scorer`, `internal/fraud`).

## Modules

| File | Purpose |
|------|---------|
| `feature_spec.py` | 16-dim vector; must match `internal/fraud/feature_spec.go` |
| `scoring_policy.py` | Post-ML heuristics and tier mapping |
| `policy_config.py` | `FRAUD_POLICY_*` and `metadata.json` policy |
| `policy_calibrate.py` | Policy grid search after bootstrap |
| `traffic_simulator.py` | Labeled synthetic traffic |
| `labeled_dataset.py` | Load labeled parquet/csv; time-based split |
| `artifact_bootstrap.py` | `bootstrap`, `fit`, `fit-validate`, `validate`, `validate-artifacts` |
| `simulation_benchmark.py` | Synthetic benchmark (`--simulate`) and CH shadow report |
| `evaluate.py` | Weekly shadow precision report (json + markdown) |
| `features_export.py` | ClickHouse `ml_features_1m` → csv/parquet |
| `ch_client.py` | ClickHouse HTTP client from env |
| `shadow_precision.py` | Proxy-label precision SQL |
| `fixture_generator.py` | Writes `testdata/ml/features_*.json` |

Docs: `LABELS.md` (training data contract), `docs/runbooks/FRAUD_SHADOW_PRECISION.md`.

Output directory: `var/fraudscore/artifacts/` (`FRAUD_ARTIFACT_DIR`). `model.txt` and `testdata/ml/features_*.json` are generated locally — not committed.

## Commands

```bash
pip install -r fraud_modeling/requirements.txt

# Dev / no labels
python3 fraud_modeling/artifact_bootstrap.py bootstrap
python3 fraud_modeling/artifact_bootstrap.py bootstrap-validate

# Production training (labeled parquet)
export FRAUD_TRAIN_DATASET=var/fraudscore/training/labeled.parquet
python3 fraud_modeling/artifact_bootstrap.py fit
python3 fraud_modeling/artifact_bootstrap.py fit-validate   # CronJob default

python3 fraud_modeling/artifact_bootstrap.py bootstrap
python3 fraud_modeling/fixture_generator.py
python3 fraud_modeling/artifact_bootstrap.py validate --model var/fraudscore/artifacts/model.txt
go run ./cmd/ml-validate -model var/fraudscore/artifacts/model.txt

make fraud-modeling-check   # optional local; not in main CI
```

Docker: `docker build -f deploy/ml/Dockerfile -t espx-ml-bootstrap:latest .`

## Labeled training (`fit`)

1. Export features from ClickHouse (`features_export.py`).
2. Join operator labels — see `LABELS.md`.
3. Time-based split (no random shuffle across `window_start`).

```bash
python3 fraud_modeling/artifact_bootstrap.py fit \
  --dataset var/fraudscore/training/labeled.parquet \
  --val-fraction 0.2
```

| Env | Default | Effect |
|-----|---------|--------|
| `FRAUD_TRAIN_DATASET` | — | Default `--dataset` for `fit` / CronJob |
| `FRAUD_FIT_VAL_FRACTION` | `0.2` | Validation tail fraction by time |
| `FRAUD_FIT_BOOST_ROUNDS` | `200` | LightGBM max rounds |
| `FRAUD_FIT_MIN_ROWS` | `500` | Minimum labeled rows |
| `FRAUD_BOOTSTRAP_IFOREST` | `0` | Skip iforest ONNX (prod uses LightGBM only) |

Promotion: `fit-validate` → register `metadata.json` hash in `ml_model_versions` → `SYNCING`.

## ClickHouse export

Uses `CH_READONLY_DSN` (or `CH_DSN`) + `CH_HTTP_PORT` (default `8123`).

## Feedback loop (manual labels)

| Env | Default | Effect |
|-----|---------|--------|
| `DB_DSN` | — | Postgres DSN; enables manual label join in `features_export.py` |
| `FRAUD_MANUAL_LABELS` | `var/fraudscore/training/manual_labels.csv` | Output path for `manual_labels_export.py` |

When `DB_DSN` is set, `features_export.py` adds `label` and `label_source` from `ml_manual_labels` (keyed on `ip_hash_hex`). `fit` can also apply overrides from `FRAUD_MANUAL_LABELS` CSV at train time.

## Shadow evaluation

```bash
export FRAUD_EVAL_HOURS=168
python3 fraud_modeling/evaluate.py --format both --min-labeled-rows 100
```

## Policy

`FRAUD_POLICY_SOURCE`: `env` | `metadata` | `auto`.

## Ops tooling (Phase D)

| Tool / endpoint | Purpose |
|-----------------|---------|
| `go run ./cmd/ml-replay` | Replay fixtures or CH through `ScoreBatch` → CSV |
| `GET /api/v1/ops/ml-model` | Model status, drift, precision/recall, feature importance |
| `GET /api/v1/ops/ml-model/labels` | Recent manual labels (limit 100) |
| `POST /api/v1/ops/ml-model/labels` | Submit manual fraud/clean label by `ip_hash` |
| Prometheus `ml_shadow_*` | Shadow scoring observability in `ivt-detector` |

```bash
# Fixtures → stdout CSV
go run ./cmd/ml-replay -fixtures testdata/ml

# ClickHouse window (requires CH_READONLY_DSN)
go run ./cmd/ml-replay -clickhouse -limit 500 -minutes 60 -output /tmp/replay.csv

# Shadow eval report (feeds ops API drift fields)
python3 fraud_modeling/evaluate.py --format both
```

Feedback loop env: `DB_DSN` (export labels from PG), `FRAUD_MANUAL_LABELS` (fit overrides), `FRAUD_EVAL_REPORT_PATH` (ops API drift).

## Feature contract change

1. `feature_spec.py` + `internal/fraud/feature_spec.go`
2. `python3 fraud_modeling/artifact_bootstrap.py bootstrap && python3 fraud_modeling/fixture_generator.py`
3. `python3 fraud_modeling/artifact_bootstrap.py validate --model var/fraudscore/artifacts/model.txt`
4. `go test ./internal/fraud/...`
