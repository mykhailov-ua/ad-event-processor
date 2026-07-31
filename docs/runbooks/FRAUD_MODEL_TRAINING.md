# Runbook: labeled fraud model training

Production model training when operator labels exist. Synthetic `bootstrap` is for dev only.

Related: `fraud_modeling/LABELS.md`, `fraud_modeling/README.md`, `docs/runbooks/FRAUD_SHADOW_PRECISION.md`.

---

## Workflow

| Step | Command |
| :--- | :--- |
| 1. Export features | `python3 fraud_modeling/features_export.py --output /var/fraudscore/training/features.parquet --format parquet` |
| 2. Join labels | See `LABELS.md` — add `label`, `label_source` |
| 3. Train | `python3 fraud_modeling/artifact_bootstrap.py fit` |
| 4. Smoke artifacts | `python3 fraud_modeling/artifact_bootstrap.py validate-artifacts` |
| 5. Fixture contract | `go run ./cmd/ml-validate` |
| 6. Promote | Insert `ml_model_versions` with artifact hash → `SYNCING` |

CronJob `fraud-scorer-retrain` runs `fit-validate`: uses `FRAUD_TRAIN_DATASET` when the file exists, otherwise falls back to synthetic `bootstrap`.

---

## Environment

```bash
export FRAUD_TRAIN_DATASET=/var/fraudscore/training/labeled.parquet
export FRAUD_FIT_VAL_FRACTION=0.2
export FRAUD_FIT_BOOST_ROUNDS=200
export FRAUD_FIT_MIN_ROWS=500
export FRAUD_BOOTSTRAP_IFOREST=0
export FRAUD_ARTIFACT_DIR=/var/fraudscore/artifacts
```

---

## Time split

Never use random row shuffle for train/val. Default: earliest 80% of `window_start` → train, latest 20% → validation.

Explicit boundaries:

```bash
python3 fraud_modeling/artifact_bootstrap.py fit \
  --train-until 2026-03-01T00:00:00Z \
  --val-from 2026-03-01T00:00:00Z
```

---

## Outputs

| Artifact | Path |
| :--- | :--- |
| LightGBM model | `var/fraudscore/artifacts/model.txt` |
| Metadata + metrics | `var/fraudscore/artifacts/metadata.json` |
| iforest placeholder | `var/fraudscore/artifacts/iforest.onnx` (disabled unless `FRAUD_BOOTSTRAP_IFOREST=1`) |

`metadata.json` includes `metrics.training` with window ranges and label-source counts.

---

## Verification before promotion

```bash
python3 fraud_modeling/artifact_bootstrap.py validate-artifacts
go run ./cmd/ml-validate
python3 fraud_modeling/evaluate.py --format both --min-labeled-rows 50
```

Hold promotion if shadow precision drops > 10 pp week-over-week (see shadow precision runbook).

---

## Manual CronJob trigger

```bash
kubectl -n espx create job --from=cronjob/fraud-scorer-retrain fraud-retrain-manual-$(date +%s)
```

Ensure `/var/fraudscore/training/labeled.parquet` exists on the PVC before triggering production fit.
