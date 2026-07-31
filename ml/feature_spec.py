from __future__ import annotations

FEATURE_NAMES: tuple[str, ...] = (
    "events",
    "clicks",
    "ctr",
    "spend_norm",
    "spend_ratio",
    "unique_users",
    "unique_uas",
)

FEATURE_DIMS = len(FEATURE_NAMES)


def row_to_vector(row: dict[str, int]) -> list[float]:
    events = int(row.get("events", 0))
    clicks = int(row.get("clicks", 0))
    spend_micro = int(row.get("spend_micro", 0))
    budget_limit_micro = int(row.get("budget_limit_micro", 0))
    unique_users = int(row.get("unique_users", 0))
    unique_uas = int(row.get("unique_uas", 0))

    ctr = float(clicks) / float(events) if events > 0 else 0.0
    spend_norm = float(spend_micro) / 1e6
    spend_ratio = float(spend_micro) / float(budget_limit_micro) if budget_limit_micro > 0 else 0.0

    return [
        float(events),
        float(clicks),
        ctr,
        spend_norm,
        spend_ratio,
        float(unique_users),
        float(unique_uas),
    ]
