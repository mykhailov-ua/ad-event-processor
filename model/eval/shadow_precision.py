"""Shadow precision report from ml_shadow_scores vs proxy labels in ClickHouse."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

PROXY_LABEL_METHOD = "proxy"
PROXY_LABEL_DEFINITION = (
    "positive: fraud_events with non-empty fraud_reason in the lookback window; "
    "negative: impressions in the window excluding positives (capped sample)"
)

AUDITED_LABEL_METHOD = "manual"
AUDITED_LABEL_DEFINITION = "human-reviewed labels from ml_manual_labels (buyer or ops UI)"
AUDITED_LOW_CONFIDENCE_ROWS = 30

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

AUDITED_PRECISION_SQL = """
WITH
    manual AS (
        SELECT
            t.1 AS ip_hash,
            t.2 AS label
        FROM (
            SELECT arrayJoin(
                arrayZip({ip_hashes:Array(FixedString(16))}, {labels:Array(UInt8)})
            ) AS t
        )
    ),
    shadow AS (
        SELECT
            ip_hash,
            max(score) AS ml_score
        FROM ml_shadow_scores
        WHERE created_at >= now() - INTERVAL {hours:UInt32} HOUR
          AND ip_hash IN (SELECT ip_hash FROM manual)
        GROUP BY ip_hash
    ),
    labeled AS (
        SELECT s.ml_score, m.label
        FROM shadow AS s
        INNER JOIN manual AS m USING (ip_hash)
    )
SELECT
    countIf(label = 1 AND ml_score >= {threshold:Float64}) AS tp,
    countIf(label = 0 AND ml_score >= {threshold:Float64}) AS fp,
    countIf(label = 1 AND ml_score < {threshold:Float64}) AS fn,
    countIf(label = 0 AND ml_score < {threshold:Float64}) AS tn,
    count() AS matched_rows
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

def audited_confidence(labeled_rows: int) -> str:
    """Return confidence band for manual-label sample size."""
    if labeled_rows < AUDITED_LOW_CONFIDENCE_ROWS:
        return "low"
    if labeled_rows < 100:
        return "medium"
    return "high"

def _metrics_from_confusion(
    tp: int,
    fp: int,
    fn: int,
    tn: int,
    *,
    labeled_rows: int,
    label_method: str,
    label_definition: str,
    hours: int,
    threshold: float,
    status: str,
    matched_rows: int | None = None,
) -> dict[str, Any]:
    precision = tp / (tp + fp) if (tp + fp) else 0.0
    recall = tp / (tp + fn) if (tp + fn) else 0.0
    f1 = (2 * precision * recall / (precision + recall)) if (precision + recall) else 0.0
    fpr = fp / (fp + tn) if (fp + tn) else 0.0

    out: dict[str, Any] = {
        "status": status,
        "labeled_rows": labeled_rows,
        "hours": hours,
        "threshold": threshold,
        "label_method": label_method,
        "label_definition": label_definition,
        "tp": tp,
        "fp": fp,
        "fn": fn,
        "tn": tn,
        "precision": precision,
        "recall": recall,
        "f1": f1,
        "false_positive_rate": fpr,
    }
    if matched_rows is not None:
        out["matched_rows"] = matched_rows
    if label_method == AUDITED_LABEL_METHOD:
        out["confidence"] = audited_confidence(labeled_rows)
    return out

def _empty_audited_metrics(*, hours: int, threshold: float, labeled_rows: int = 0) -> dict[str, Any]:
    return _metrics_from_confusion(
        0,
        0,
        0,
        0,
        labeled_rows=labeled_rows,
        label_method=AUDITED_LABEL_METHOD,
        label_definition=AUDITED_LABEL_DEFINITION,
        hours=hours,
        threshold=threshold,
        status="empty",
        matched_rows=0,
    )

def run_audited_precision(
    client: Any,
    labels: list[tuple[str, int]],
    *,
    hours: int = 24,
    threshold: float = 0.6,
) -> dict[str, Any]:
    """Run precision report on ml_manual_labels joined to shadow scores."""
    if not labels:
        return _empty_audited_metrics(hours=hours, threshold=threshold)

    ip_hashes: list[bytes] = []
    label_vals: list[int] = []
    for ip_hash, label in labels:
        normalized = ip_hash.strip().lower()
        if len(normalized) != 32:
            continue
        try:
            ip_hashes.append(bytes.fromhex(normalized))
        except ValueError:
            continue
        label_vals.append(int(label))

    labeled_rows = len(ip_hashes)
    if labeled_rows == 0:
        return _empty_audited_metrics(hours=hours, threshold=threshold)

    result = client.query(
        AUDITED_PRECISION_SQL,
        parameters={
            "hours": hours,
            "threshold": threshold,
            "ip_hashes": ip_hashes,
            "labels": label_vals,
        },
    )
    if not result.result_rows:
        empty = _empty_audited_metrics(hours=hours, threshold=threshold, labeled_rows=labeled_rows)
        empty["matched_rows"] = 0
        return empty

    row = result.result_rows[0]
    columns = result.column_names
    metrics = dict(zip(columns, row, strict=True))
    tp = int(metrics.get("tp", 0))
    fp = int(metrics.get("fp", 0))
    fn = int(metrics.get("fn", 0))
    tn = int(metrics.get("tn", 0))
    matched_rows = int(metrics.get("matched_rows", 0))

    return _metrics_from_confusion(
        tp,
        fp,
        fn,
        tn,
        labeled_rows=labeled_rows,
        label_method=AUDITED_LABEL_METHOD,
        label_definition=AUDITED_LABEL_DEFINITION,
        hours=hours,
        threshold=threshold,
        status="ok" if matched_rows > 0 else "empty",
        matched_rows=matched_rows,
    )

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

    out = _metrics_from_confusion(
        tp,
        fp,
        fn,
        tn,
        labeled_rows=labeled_rows,
        label_method=PROXY_LABEL_METHOD,
        label_definition=PROXY_LABEL_DEFINITION,
        hours=hours,
        threshold=threshold,
        status="ok" if labeled_rows > 0 else "empty",
    )
    out["generated_at"] = datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")
    return out

def format_markdown(report: dict[str, Any]) -> str:
    """Render operator-facing markdown summary."""
    status = str(report.get("status", "unknown"))
    proxy = report.get("proxy_metrics")
    audited = report.get("audited_metrics")
    if isinstance(proxy, dict):
        return _format_markdown_mixed(report, proxy, audited if isinstance(audited, dict) else None)

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

    return "\n".join(lines + _format_metrics_section(report))

def _format_markdown_mixed(
    report: dict[str, float | int | str],
    proxy: dict[str, Any],
    audited: dict[str, Any] | None,
) -> str:
    lines = [
        "# Fraud shadow eval report",
        "",
        f"- Status: `{report.get('status', 'unknown')}`",
        f"- Generated: {report.get('generated_at', 'n/a')}",
        f"- Window: last {report.get('hours', 'n/a')} hours",
        f"- Threshold: {report.get('threshold', 'n/a')}",
        "",
        "## Proxy metrics",
        "",
        f"- Label method: `{proxy.get('label_method', PROXY_LABEL_METHOD)}` (not accuracy; proxy labels only)",
        f"- Labeled rows: {proxy.get('labeled_rows', 0)}",
        "",
    ]
    proxy_definition = proxy.get("label_definition", PROXY_LABEL_DEFINITION)
    if proxy_definition:
        lines.extend([str(proxy_definition), ""])
    if str(proxy.get("status", "")) in {"skipped", "empty"}:
        lines.append("No proxy metrics in ClickHouse for this window.")
    else:
        lines.extend(_format_metrics_section(proxy))
    lines.extend(["", "## Audited metrics", ""])
    audited_block = audited or _empty_audited_metrics(
        hours=int(report.get("hours", 24) or 24),
        threshold=float(report.get("threshold", 0.6) or 0.6),
    )
    lines.append(
        f"- Label method: `{audited_block.get('label_method', AUDITED_LABEL_METHOD)}` "
        "(human labels; never reported as accuracy)"
    )
    lines.append(f"- Labeled rows: {audited_block.get('labeled_rows', 0)}")
    if "confidence" in audited_block:
        lines.append(f"- Confidence: `{audited_block.get('confidence')}`")
    lines.append("")
    audited_definition = audited_block.get("label_definition", AUDITED_LABEL_DEFINITION)
    if audited_definition:
        lines.extend([str(audited_definition), ""])
    if str(audited_block.get("status", "")) in {"skipped", "empty"}:
        lines.append("No audited rows matched shadow scores in this window.")
    else:
        lines.extend(_format_metrics_section(audited_block))

    drift = report.get("drift")
    if isinstance(drift, dict) and drift.get("status") == "ok":
        max_drift = float(drift.get("max_drift", 0.0))
        drift_detected = bool(drift.get("drift_detected", False))
        lines.extend(
            [
                "",
                "## Input drift",
                "",
                f"- Max relative drift: {max_drift:.2%}",
                f"- Drift detected: `{drift_detected}`",
                "",
            ]
        )
    return "\n".join(lines)

def _format_metrics_section(block: dict[str, Any]) -> list[str]:
    tp = int(block.get("tp", 0))
    fp = int(block.get("fp", 0))
    fn = int(block.get("fn", 0))
    tn = int(block.get("tn", 0))
    precision = float(block.get("precision", 0.0))
    recall = float(block.get("recall", 0.0))
    f1 = float(block.get("f1", 0.0))
    fpr = float(block.get("false_positive_rate", 0.0))
    return [
        "### Confusion matrix",
        "",
        "| | Predicted fraud | Predicted legit |",
        "| :--- | ---: | ---: |",
        f"| Actual fraud | {tp} | {fn} |",
        f"| Actual legit | {fp} | {tn} |",
        "",
        "### Metrics",
        "",
        "| Metric | Value |",
        "| :--- | ---: |",
        f"| Precision | {precision:.4f} |",
        f"| Recall | {recall:.4f} |",
        f"| F1 | {f1:.4f} |",
        f"| False positive rate | {fpr:.4f} |",
        "",
    ]

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
