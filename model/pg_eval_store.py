"""Persist shadow eval reports to Postgres for control plane readers."""

from __future__ import annotations

import json
import os
from datetime import UTC, datetime
from typing import Any

EVAL_REPORT_ID = "shadow_eval"


def _parse_generated_at(report: dict[str, Any]) -> datetime:
    raw = report.get("generated_at")
    if isinstance(raw, str) and raw:
        try:
            return datetime.fromisoformat(raw).astimezone(UTC)
        except ValueError:
            pass
    return datetime.now(UTC)


def upsert_ml_eval_report(report: dict[str, Any]) -> None:
    """Upsert the latest shadow eval row; no-op when DB_DSN is unset."""
    dsn = os.environ.get("DB_DSN", "").strip()
    if not dsn:
        return

    import psycopg2

    status = str(report.get("status", "error"))
    if status not in {"ok", "empty", "error", "skipped"}:
        status = "error"

    precision = float(report.get("precision", 0.0) or 0.0)
    recall = float(report.get("recall", 0.0) or 0.0)
    proxy = report.get("proxy_metrics")
    if isinstance(proxy, dict):
        precision = float(proxy.get("precision", precision) or precision)
        recall = float(proxy.get("recall", recall) or recall)
    label_method = str(report.get("label_method") or "proxy")
    drift_payload = report.get("drift")
    if drift_payload is None:
        drift_json = "{}"
    else:
        drift_json = json.dumps(drift_payload)

    report_payload = {
        "status": status,
        "generated_at": report.get("generated_at"),
        "hours": report.get("hours"),
        "threshold": report.get("threshold"),
        "proxy_metrics": report.get("proxy_metrics", {}),
        "audited_metrics": report.get("audited_metrics", {}),
        "drift": drift_payload,
        "drift_detected": report.get("drift_detected"),
        "precision": precision,
        "recall": recall,
        "label_method": label_method,
        "labeled_rows": report.get("labeled_rows", 0),
    }
    report_json = json.dumps(report_payload)

    generated_at = _parse_generated_at(report)

    conn = psycopg2.connect(dsn)
    try:
        with conn, conn.cursor() as cur:
            cur.execute(
                """
                    INSERT INTO ml_eval_reports (
                        id, generated_at, precision, recall, drift_json, status, label_method, report_json
                    ) VALUES (%s, %s, %s, %s, %s::jsonb, %s, %s, %s::jsonb)
                    ON CONFLICT (id) DO UPDATE SET
                        generated_at = EXCLUDED.generated_at,
                        precision = EXCLUDED.precision,
                        recall = EXCLUDED.recall,
                        drift_json = EXCLUDED.drift_json,
                        status = EXCLUDED.status,
                        label_method = EXCLUDED.label_method,
                        report_json = EXCLUDED.report_json,
                        created_at = NOW()
                    """,
                (
                    EVAL_REPORT_ID,
                    generated_at,
                    precision,
                    recall,
                    drift_json,
                    status,
                    label_method,
                    report_json,
                ),
            )
    finally:
        conn.close()
