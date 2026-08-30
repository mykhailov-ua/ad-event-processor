"""16-dim feature vector from ml_features_1m row aggregates.

Role:
- Map raw CH/PG row counters to the LightGBM input vector.
- Same column order and formulas as internal/fraud/feature_spec.go FeatureRow.ToVector.

Input row keys (int, micro-units for spend fields):
- events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas

Derived indices (FEATURE_NAMES order):
- ctr = clicks/events (0 when events=0)
- spend_norm = spend_micro / 1e6
- spend_ratio = spend_micro / budget_limit_micro (0 when budget=0)
- impression_pressure = events / (clicks + 1)
- user_click_gap = unique_users / (clicks + 1)
- ua_diversity = unique_uas / events (0 when events=0)

Invariants:
- FEATURE_DIMS must equal booster.num_feature() and Go FraudFeatureDims.
- _safe_ratio returns 0.0 on non-positive denominator (matches Go).

Verify:
  pytest model/tests/test_feature_spec.py -q
  go test ./internal/fraud/ -short -run TestFeatureRowToVector -count=1
  python3 model/train/fixture_generator.py
  go run ./cmd/ml-validate -fixtures internal/fraud/testdata
"""

from __future__ import annotations

FEATURE_NAMES: tuple[str, ...] = (
    "events",
    "clicks",
    "ctr",
    "spend_norm",
    "spend_ratio",
    "unique_users",
    "unique_uas",
    "events_per_ua",
    "clicks_per_ua",
    "users_per_ua",
    "clicks_per_user",
    "spend_per_click",
    "ua_diversity",
    "events_per_user",
    "impression_pressure",
    "user_click_gap",
)

FEATURE_DIMS = len(FEATURE_NAMES)


def _safe_ratio(numerator: float, denominator: float) -> float:
    """Return numerator/denominator or 0.0 when denominator <= 0."""
    if denominator <= 0:
        return 0.0
    return numerator / denominator


def row_to_vector(row: dict[str, int]) -> list[float]:
    """Map ml_features_1m aggregate row to model input vector.

    Args:
        row: Counter dict with keys from ROW_FIELDS in labeled_dataset / CH export.

    Returns:
        List of length FEATURE_DIMS in FEATURE_NAMES order.
    """
    events = int(row.get("events", 0))
    clicks = int(row.get("clicks", 0))
    spend_micro = int(row.get("spend_micro", 0))
    budget_limit_micro = int(row.get("budget_limit_micro", 0))
    unique_users = int(row.get("unique_users", 0))
    unique_uas = int(row.get("unique_uas", 0))

    events_f = float(events)
    clicks_f = float(clicks)
    unique_users_f = float(unique_users)
    unique_uas_f = float(unique_uas)
    spend_norm = float(spend_micro) / 1e6

    ctr = _safe_ratio(clicks_f, events_f)
    spend_ratio = _safe_ratio(float(spend_micro), float(budget_limit_micro))

    return [
        events_f,
        clicks_f,
        ctr,
        spend_norm,
        spend_ratio,
        unique_users_f,
        unique_uas_f,
        _safe_ratio(events_f, unique_uas_f),
        _safe_ratio(clicks_f, unique_uas_f),
        _safe_ratio(unique_users_f, unique_uas_f),
        _safe_ratio(clicks_f, unique_users_f),
        _safe_ratio(spend_norm, clicks_f),
        _safe_ratio(unique_uas_f, events_f),
        _safe_ratio(events_f, unique_users_f),
        # +1.0 denominators avoid div-by-zero; matches Go feature_spec.go.
        _safe_ratio(events_f, clicks_f + 1.0),
        _safe_ratio(unique_users_f, clicks_f + 1.0),
    ]
