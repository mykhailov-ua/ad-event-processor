# Runbook: fraud shadow precision check

Weekly proxy-label evaluation for the cold-path ML shadow scorer (`ivt-detector` / `fraud-scorer`).

Related: `fraud_modeling/README.md`, `docs/sql/explain/fraud_shadow_precision_report.sql`.

---

## When to run

| Trigger | Schedule |
| :--- | :--- |
| Operator | Weekly (Monday morning) |
| CI / CronJob | `fraud-shadow-eval` CronJob (`0 6 * * 1` UTC) when `analytics_ml` profile is enabled |
| After model deploy | Once within 24 h of `ml_model_versions` sync |

**Goal:** confirm shadow scores align with known fraud signals before promoting a new `model.txt`.

---

## Prerequisites

1. ClickHouse `analytics_ml` profile running with `ml_shadow_scores`, `fraud_events`, `impressions`.
2. `ivt-detector` or `fraud-scorer` writing shadow scores (check `ml_shadow_scores` row count).
3. Read-only CH credentials:

```bash
export CH_READONLY_DSN=clickhouse://default:secure_ch_pass@127.0.0.1:9000/ad_event_processor
export CH_HTTP_PORT=8123
export FRAUD_POLICY_ML_THRESHOLD=0.35   # optional; default from policy
export FRAUD_EVAL_HOURS=168             # 7-day window
```

4. Python deps: `pip install -r fraud_modeling/requirements.txt`

---

## Run evaluation

```bash
python3 fraud_modeling/evaluate.py --format both
```

Outputs:

| File | Content |
| :--- | :--- |
| `var/fraudscore/shadow_eval_report.json` | Machine-readable metrics |
| `var/fraudscore/shadow_eval_report.md` | Operator summary |

Custom window / threshold:

```bash
python3 fraud_modeling/evaluate.py \
  --hours 24 \
  --threshold 0.6 \
  --output /tmp/shadow_report \
  --format both
```

Enforce minimum sample size (fail for alerting):

```bash
python3 fraud_modeling/evaluate.py --min-labeled-rows 100
```

---

## Interpret results

### Status `ok`

| Metric | Healthy range (guidance) |
| :--- | :--- |
| `labeled_rows` | ≥ 100 for stable estimates |
| `precision` | ≥ 0.70 at production threshold |
| `recall` | ≥ 0.50 (proxy labels under-count fraud) |
| `false_positive_rate` | ≤ 0.05 on impression negatives |

Proxy positives come from `fraud_events` with non-empty `fraud_reason`. Negatives are random impression IPs — expect conservative precision vs real-world IVT.

### Status `empty`

No overlap between `ml_shadow_scores` and proxy labels in the window.

| Check | Action |
| :--- | :--- |
| `SELECT count() FROM ml_shadow_scores WHERE created_at > now() - INTERVAL 24 HOUR` | Enable shadow rule / fraud-scorer |
| `SELECT count() FROM fraud_events WHERE created_at > now() - INTERVAL 24 HOUR` | Verify IVT pipeline |
| Window too narrow | Increase `--hours` |

### Status `skipped`

ClickHouse unreachable. Verify `CH_READONLY_DSN`, `CH_HTTP_PORT`, network from runner pod.

---

## Kubernetes CronJob

Manifest: `deploy/k8s/apps/deployment-fraud-scorer.yaml` (`fraud-shadow-eval`).

Manual one-off inside cluster:

```bash
kubectl -n espx create job --from=cronjob/fraud-shadow-eval fraud-shadow-eval-manual-$(date +%s)
kubectl -n espx logs -l job-name=fraud-shadow-eval-manual-... --tail=50
```

Reports land in the `ml-artifacts` PVC under `/var/fraudscore/` when `--output` points there.

---

## SQL reference

Ad-hoc query (same logic as `evaluate.py`):

```bash
clickhouse-client --multiquery < docs/sql/explain/fraud_shadow_precision_report.sql
```

---

## Escalation

| Symptom | Action |
| :--- | :--- |
| Precision drops > 10 pp week-over-week | Hold `ml_model_versions` promotion; re-run `artifact_bootstrap validate` |
| `labeled_rows` = 0 for 2+ weeks | Shadow path not wired; check `ivt-detector` logs and CH ingest |
| High FPR on negatives | Review `FRAUD_POLICY_*` thresholds; run `simulation_benchmark.py --simulate` |
