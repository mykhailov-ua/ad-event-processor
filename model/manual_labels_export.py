#!/usr/bin/env python3
"""Export manual labels from Postgres for ML training feedback loop."""
from __future__ import annotations

import csv
import os
import sys
from pathlib import Path

EXPORT_COLUMNS = ("ip_hash", "label", "source", "reason")


def export_manual_labels(output_path: Path, *, dsn: str) -> int:
    """Write ml_manual_labels rows to CSV; return row count."""
    import psycopg2

    conn = psycopg2.connect(dsn)
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT ip_hash, label, source, reason FROM ml_manual_labels")
            rows = cur.fetchall()
    finally:
        conn.close()

    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.writer(handle)
        writer.writerow(EXPORT_COLUMNS)
        writer.writerows(rows)
    return len(rows)


def main() -> int:
    try:
        import psycopg2
    except ImportError:
        print("manual_labels_export: psycopg2 missing, skipping", file=sys.stderr)
        return 0

    dsn = os.environ.get("DB_DSN", "")
    if not dsn:
        print("manual_labels_export: DB_DSN missing, skipping", file=sys.stderr)
        return 0

    output = Path(os.environ.get("FRAUD_MANUAL_LABELS", "var/fraudscore/training/manual_labels.csv"))
    try:
        count = export_manual_labels(output, dsn=dsn)
    except (psycopg2.Error, OSError) as exc:
        print(f"manual_labels_export: {exc}", file=sys.stderr)
        return 1

    print(f"manual_labels_export: exported {count} labels to {output}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
