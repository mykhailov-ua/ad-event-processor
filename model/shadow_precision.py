"""Shadow precision report from ml_shadow_scores vs proxy labels in ClickHouse."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

PROXY_LABEL_METHOD = "proxy"
PROXY_LABEL_DEFINITION = (
    "positive: fraud_events with non-empty fraud_reason in the lookback window; "
    "negative: impressions in the window excluding positives (capped sample)"
)

SHADOW_PRECISION_SQL = """
WITH
    shadow AS (
        SELECT
            ip_hash,
            max(score) AS ml_score
        FROM ml_shadow_scores
        WHERE created_at >= now() - INTERVAL {hours:UInt32} HOUR
        GROUP BY ip_hash
    ),
    positives AS (
        SELECT DISTINCT ip_hash
        FROM fraud_events
        WHERE created_at >= now() - INTERVAL {hours:UInt32} HOUR
          AND fraud_reason != ''
    ),
    negatives AS (
        SELECT DISTINCT i.ip_hash
        FROM impressions AS i
        WHERE i.created_at >= now() - INTERVAL {hours:UInt32} HOUR
          AND i.ip_hash NOT IN (SELECT ip_hash FROM positives)
        LIMIT 10000
    ),
    labeled AS (
        SELECT s.ip_hash, s.ml_score, toUInt8(1) AS label
        FROM shadow AS s
        INNER JOIN positives AS p USING (ip_hash)
        UNION ALL
        SELECT s.ip_hash, s.ml_score, toUInt8(0) AS label
        FROM shadow AS s
        INNER JOIN negatives AS n USING (ip_hash)
    )
SELECT
    countIf(label = 1 AND ml_score >= {threshold:Float64}) AS tp,
    countIf(label = 0 AND ml_score >= {threshold:Float64}) AS fp,
    countIf(label = 1 AND ml_score < {threshold:Float64}) AS fn,
    countIf(label = 0 AND ml_score < {threshold:Float64}) AS tn,
    count() AS labeled_rows
FROM labeled
"""

DRIFT_STATS_SQL = """
SELECT
    avg(events) AS events,
    avg(clicks) AS clicks,
    avg(spend_micro) AS spend_micro,
    avg(budget_limit_micro) AS budget_limit_micro,
    avg(unique_users) AS unique_users,
    avg(unique_uas) AS unique_uas
FROM ml_features_1m
WHERE window_start >= now() - INTERVAL {hours:UInt32} HOUR
"""


def run_shadow_precision(
    client: Any,
    *,
    hours: int = 24,
    threshold: float = 0.6,
) -> dict[str, Any]:
    """Run proxy-label precision report; labeled_rows=0 when CH has no shadow data."""
    result = client.query(
        SHADOW_PRECISION_SQL,
        parameters={"hours": hours, "threshold": threshold},
    )
    if not result.result_rows:
        return {
            "status": "empty",
            "labeled_rows": 0,
            "hours": hours,
            "threshold": threshold,
            "label_method": PROXY_LABEL_METHOD,
            "label_definition": PROXY_LABEL_DEFINITION,
        }

    row = result.result_rows[0]
    columns = result.column_names
    metrics = dict(zip(columns, row, strict=True))
    tp = int(metrics.get("tp", 0))
    fp = int(metrics.get("fp", 0))
    fn = int(metrics.get("fn", 0))
    tn = int(metrics.get("tn", 0))
    labeled_rows = int(metrics.get("labeled_rows", 0))

    precision = tp / (tp + fp) if (tp + fp) else 0.0
    recall = tp / (tp + fn) if (tp + fn) else 0.0
    f1 = (2 * precision * recall / (precision + recall)) if (precision + recall) else 0.0
    fpr = fp / (fp + tn) if (fp + tn) else 0.0

    return {
        "status": "ok" if labeled_rows > 0 else "empty",
        "labeled_rows": labeled_rows,
        "hours": hours,
        "threshold": threshold,
        "label_method": PROXY_LABEL_METHOD,
        "label_definition": PROXY_LABEL_DEFINITION,
        "tp": tp,
        "fp": fp,
        "fn": fn,
        "tn": tn,
        "precision": precision,
        "recall": recall,
        "f1": f1,
        "false_positive_rate": fpr,
        "generated_at": datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }


def format_markdown(report: dict[str, float | int | str]) -> str:
    """Render operator-facing markdown summary."""
    status = str(report.get("status", "unknown"))
    lines = [
        "# Fraud shadow precision report",
        "",
        f"- Status: `{status}`",
        f"- Generated: {report.get('generated_at', 'n/a')}",
        f"- Window: last {report.get('hours', 'n/a')} hours",
        f"- Threshold: {report.get('threshold', 'n/a')}",
        f"- Labeled rows: {report.get('labeled_rows', 0)}",
        f"- Label method: `{report.get('label_method', PROXY_LABEL_METHOD)}` (not human-audited ground truth)",
        "",
    ]
    label_definition = report.get("label_definition", PROXY_LABEL_DEFINITION)
    if label_definition:
        lines.extend(
            [
                "## Label definition",
                "",
                str(label_definition),
                "",
            ]
        )
    if status in {"skipped", "empty"}:
        if "reason" in report:
            lines.append(f"- Note: {report['reason']}")
        lines.append("")
        lines.append("No metrics: insufficient shadow scores or proxy labels in ClickHouse.")
        return "\n".join(lines)

    tp = int(report.get("tp", 0))
    fp = int(report.get("fp", 0))
    fn = int(report.get("fn", 0))
    tn = int(report.get("tn", 0))
    precision = float(report.get("precision", 0.0))
    recall = float(report.get("recall", 0.0))
    f1 = float(report.get("f1", 0.0))
    fpr = float(report.get("false_positive_rate", 0.0))

    lines.extend(
        [
            "## Confusion matrix",
            "",
            "| | Predicted fraud | Predicted legit |",
            "| :--- | ---: | ---: |",
            f"| Actual fraud | {tp} | {fn} |",
            f"| Actual legit | {fp} | {tn} |",
            "",
            "## Metrics",
            "",
            "| Metric | Value |",
            "| :--- | ---: |",
            f"| Precision | {precision:.4f} |",
            f"| Recall | {recall:.4f} |",
            f"| F1 | {f1:.4f} |",
            f"| False positive rate | {fpr:.4f} |",
            "",
        ]
    )

    drift = report.get("drift")
    if isinstance(drift, dict) and drift.get("status") == "ok":
        max_drift = float(drift.get("max_drift", 0.0))
        drift_detected = bool(drift.get("drift_detected", False))
        lines.extend(
            [
                "## Input drift",
                "",
                f"- Max relative drift: {max_drift:.2%}",
                f"- Drift detected: `{drift_detected}`",
                "",
            ]
        )
        drift_values = drift.get("drift")
        if isinstance(drift_values, dict) and drift_values:
            lines.extend(
                [
                    "| Raw feature | Relative drift |",
                    "| :--- | ---: |",
                ]
            )
            for name, value in sorted(drift_values.items()):
                lines.append(f"| {name} | {float(value):.2%} |")
            lines.append("")

    return "\n".join(lines)


def run_drift_analysis(
    client: Any,
    *,
    hours: int = 24,
    training_stats: dict[str, dict[str, float]],
) -> dict[str, Any]:
    """Calculate input drift by comparing ClickHouse averages with training stats."""
    result = client.query(DRIFT_STATS_SQL, parameters={"hours": hours})
    if not result.result_rows:
        return {"status": "empty", "hours": hours}

    row = result.result_rows[0]
    live_stats = dict(zip(result.column_names, row, strict=True))
    drift: dict[str, float] = {}
    max_drift = 0.0

    for name, train in training_stats.items():
        if name in live_stats:
            live_val = float(live_stats[name])
            train_mean = train["mean"]
            if abs(train_mean) > 1e-6:
                d = abs(live_val - train_mean) / train_mean
                drift[name] = d
                max_drift = max(max_drift, d)

    return {
        "status": "ok",
        "hours": hours,
        "drift": drift,
        "max_drift": max_drift,
        "drift_detected": max_drift > 0.3,
    }
