# model

Python ML training and offline evaluation for fraud scoring. **Cold path only** — production inference runs in `cmd/fraud-scorer`, not on tracker.

Cross-ref: [deploy/vendor/ANTIFRAUD.md](../deploy/vendor/ANTIFRAUD.md), [cmd/CMD.md](../cmd/CMD.md).

---

## Role

| Stage | Where | Output |
| :--- | :--- | :--- |
| Feature export | CH / batch jobs | Training matrices |
| Train | `model/train/` | Model artifact (LightGBM) |
| Evaluate | `model/eval/` | Metrics, calibration reports |
| Infer | `cmd/fraud-scorer` | Redis `ml:score:boost:{campaign_id}` |
| Hot read | tracker `SettingsWatcher` | In-memory boost snapshot |

**SKU:** `ml_fraud_boost` (Scale+). Requires ClickHouse + `analytics-ml` compose profile.

---

## Layout

```
model/
  train/           Training pipelines
  eval/            Offline evaluation
  contract/        Feature/schema contracts (shared with Go scorer)
  data/            Training data layout
  testdata/        Fixture datasets
  tests/           Python unit tests
  Dockerfile       Training image
  Dockerfile.evaluate
  pyproject.toml
  requirements.txt
  requirements-dev.txt
```

---

## Operate

### Environment

```bash
cd model
pip install -r requirements.txt
pip install -r requirements-dev.txt   # tests
```

Docker training image: `model/Dockerfile`.

### Train

Follow scripts/notebooks in `model/train/`. Features must match `contract/` definitions consumed by Go `fraud-scorer`.

### Deploy artifact

Production path (typical):

```
var/fraudscore/artifacts/<model_version>/
```

Configure on processor/fraud-scorer:

- `FRAUD_SCORING_ENABLED=true`
- Model path env vars (see `internal/config` and compose `analytics-ml` profile)

### Runtime flow

```
ClickHouse ml_features_1m
  → fraud-scorer batch (≤1000 events)
  → Redis ml:score:boost:*
  → SettingsWatcher → tracker fraud filter snapshot
```

**Latency:** Boost sync default ~10 s (`FRAUD_BOOST_FULL_RESYNC_SEC`). Not per-click ML.

---

## Limits

| Limit | Detail |
| :--- | :--- |
| No hot path import | Tracker never loads Python or ONNX directly |
| CH required | Features read from ClickHouse aggregates |
| Buyer-operated quality | Bootstrap model ok; production fit is operator responsibility |
| License gate | JWT `ml_fraud_boost` must be true |
| Outbox backpressure | IVT pauses when outbox `PENDING` > 500; scorer should not flood outbox |

---

## ML enforcement actions (via outbox)

| Action | Effect |
| :--- | :--- |
| `boost` | Redis score boost |
| `blacklist` | IP on `blacklist:fraud` |
| `silent_reject` | Per-IP blacklist add — does **not** flip campaign `silent_reject_enabled` |

Configured from `ivt-detector` rules and fraud admin — not from training code directly.

---

## Test

```bash
cd model && pytest tests/
go test ./internal/fraud/... -short -count=1
```

CI: `ci.yaml` conditional `fraud-model` job (Python 3.12).

Go integration with model file: fraud scoring rule tests with artifact on disk.

---

## Development rules

1. **Keep `contract/` in sync** with Go feature reader — version breaking changes.
2. **Do not train on PII plaintext** — use hashed columns matching CH schema.
3. **Document feature lag** — CH materialized views may lag minutes; not real-time.
4. **Eval before promote** — offline metrics in `model/eval/`; no production SLA from notebook loss alone.
5. **Regenerate artifacts** in `var/` — never commit large model binaries to git.

---

## Pitfalls

1. **Expecting ML on every `/track`** — batch sidecar only; cite `BenchmarkFilterFraudBoost` scope correctly (snapshot lookup, not LGBM).
2. **Running fraud-scorer without CH** — feature scan empty; boosts stale or zero.
3. **Confusing IVT and ML** — `ivt-detector` is rule-based on CH; `fraud-scorer` is model-based (both cold).
4. **Shipping new features without scorer update** — Go reader must understand contract version.

---

## Verification checklist

| Step | Command / check |
| :--- | :--- |
| Python tests | `pytest model/tests/` |
| Go fraud package | `go test ./internal/fraud/...` |
| End-to-end cold | compose `analytics-ml` + manual boost key in Redis |
| License | JWT includes `ml_fraud_boost` |
