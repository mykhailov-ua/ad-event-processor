# Labeled training data contract

Production `fit` reads parquet or CSV produced from ClickHouse features plus operator labels.

## Required columns

| Column | Type | Description |
| :--- | :--- | :--- |
| `window_start` | datetime (UTC) | Feature window; used for **time-based** train/val split |
| `events` | int | `ml_features_1m.events` |
| `clicks` | int | `ml_features_1m.clicks` |
| `spend_micro` | int | Spend in micro-units |
| `budget_limit_micro` | int | Campaign budget limit in micro-units |
| `unique_users` | int | Distinct users in window |
| `unique_uas` | int | Distinct user agents in window |
| `label` | 0 or 1 | **1 = fraud / IVT**, **0 = legitimate |

`is_fraud` is accepted as an alias for `label`.

## Optional columns

| Column | Type | Description |
| :--- | :--- | :--- |
| `label_source` | string | Provenance tag (see below) |
| `ip_hash_hex` | string | Hex IP hash; joins `ml_manual_labels.ip_hash` for manual overrides |
| `campaign_id` | string | For audit joins only |

## Label sources (operator-defined)

| `label_source` | Meaning | Typical origin |
| :--- | :--- | :--- |
| `chargeback` | Confirmed billing dispute | Finance ledger |
| `manual_ivt` | Human reviewer marked IVT | Ops queue |
| `rule_outcome` | Deterministic IVT rule fired | `fraud_events.fraud_reason` |
| `ghost_ivt` | Ghost-mode block confirmed later | Campaign ghost tier |
| `proxy_label` | Weak label from shadow precision SQL | Offline eval only — avoid training |
| `unknown` | Default when column omitted | — |

**Do not** train on `proxy_label` rows mixed with production promotion decisions unless explicitly intended — precision on impression negatives is biased.

## Building a dataset

1. Export features (optionally joined with Postgres manual labels when `DB_DSN` is set):

```bash
export DB_DSN=postgres://user:pass@127.0.0.1:5432/ad_event_processor
python3 fraud_modeling/features_export.py \
  --since 2026-01-01T00:00:00Z \
  --output var/fraudscore/training/features.parquet \
  --format parquet
```

When `DB_DSN` is set, `features_export.py` LEFT JOINs `ml_manual_labels` on `ip_hash_hex` and adds `label` / `label_source` columns. Rows without a Postgres label get empty values for operator fill-in.

2. Join labels in ClickHouse or offline (example keys: `window_start`, `ip_hash_hex`, `campaign_id`), or export manual overrides:

```bash
export DB_DSN=postgres://user:pass@127.0.0.1:5432/ad_event_processor
export FRAUD_MANUAL_LABELS=var/fraudscore/training/manual_labels.csv
python3 fraud_modeling/manual_labels_export.py
```

3. Add `label` and `label_source` columns if not already present; write `labeled.parquet`.

4. Train:

```bash
export FRAUD_TRAIN_DATASET=var/fraudscore/training/labeled.parquet
python3 fraud_modeling/artifact_bootstrap.py fit
python3 fraud_modeling/artifact_bootstrap.py validate-artifacts
```

## Time split rules

`fit` never shuffles rows randomly across time.

| Mode | Behavior |
| :--- | :--- |
| Default | First `(1 - val_fraction)` of rows by `window_start` → train; remainder → validation |
| `--train-until` | Train rows with `window_start < train_until` |
| `--val-from` | Validation rows with `window_start >= val_from` |

Env: `FRAUD_FIT_VAL_FRACTION` (default `0.2`).

## Promotion workflow

1. `fit` or `fit-validate` → `var/fraudscore/artifacts/model.txt` + `metadata.json`
2. `go run ./cmd/ml-validate -model var/fraudscore/artifacts/model.txt` — fixture vector contract
3. `artifact_bootstrap.py validate-artifacts` — artifact smoke on `ARTIFACT_DIR`
4. Register hash in Postgres `ml_model_versions` → `SYNCING` (existing `fraud-scorer` / management sync)

Isolation Forest (`iforest.onnx`) stays **disabled** in production until ONNX ensemble path is validated (`FRAUD_BOOTSTRAP_IFOREST=0`).
