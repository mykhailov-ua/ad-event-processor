#!/usr/bin/env python3
"""Export manual labels from Postgres for ML training feedback loop.

Role:
- Read ml_manual_labels (ip_hash, label, source, reason) to CSV.
- Used by features_export LEFT JOIN and artifact_bootstrap manual override path.

Env:
- DB_DSN: Postgres connection string (required for export)
- FRAUD_MANUAL_LABELS: output path (default var/fraudscore/training/manual_labels.csv)

Verify:
  DB_DSN=postgres://... python3 model/data/manual_labels_export.py
"""

from __future__ import annotations

import csv
import os
import sys
from pathlib import Path

EXPORT_COLUMNS = ("ip_hash", "label", "source", "reason")

def load_manual_labels(dsn: str) -> list[tuple[str, int]]:
    """Load ip_hash and label pairs from ml_manual_labels."""
    import psycopg2

    conn = psycopg2.connect(dsn)
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT ip_hash, label FROM ml_manual_labels")
            rows = cur.fetchall()
    finally:
        conn.close()

    out: list[tuple[str, int]] = []
    for ip_hash, label in rows:
        if not ip_hash:
            continue
        normalized = str(ip_hash).strip().lower()
        # ml_manual_labels stores 128-bit ip_hash as 32-char hex.
        if len(normalized) != 32:
            continue
        out.append((normalized, int(label)))
    return out

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
    except (psycopg2.Error, OSError) as err:
        print(f"manual_labels_export: {err}", file=sys.stderr)
        return 1

    print(f"manual_labels_export: exported {count} labels to {output}", file=sys.stderr)
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
