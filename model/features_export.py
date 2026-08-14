#!/usr/bin/env python3
"""Export ml_features_1m rows to parquet or CSV."""

from __future__ import annotations

import argparse
import csv
import os
import sys
from datetime import UTC, datetime, timedelta
from pathlib import Path

from ch_client import ch_config_from_env, connect_client, ping_client

EXPORT_SQL = """
SELECT
    window_start,
    hex(ip_hash) AS ip_hash_hex,
    toString(campaign_id) AS campaign_id,
    events,
    clicks,
    spend_micro,
    budget_limit_micro,
    unique_users,
    unique_uas
FROM ml_features_1m
WHERE window_start >= {since:DateTime}
  AND window_start < {until:DateTime}
ORDER BY window_start
"""

EXPORT_COLUMNS = (
    "window_start",
    "ip_hash_hex",
    "campaign_id",
    "events",
    "clicks",
    "spend_micro",
    "budget_limit_micro",
    "unique_users",
    "unique_uas",
)

LABEL_COLUMNS = ("label", "label_source")


def _parse_iso8601(value: str) -> datetime:
    text = value.strip()
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    parsed = datetime.fromisoformat(text)
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC)


def _default_window(since: str, until: str) -> tuple[datetime, datetime]:
    now = datetime.now(UTC)
    end = _parse_iso8601(until) if until else now
    start = _parse_iso8601(since) if since else end - timedelta(days=7)
    return start, end


def _load_manual_labels_from_pg(dsn: str) -> dict[str, tuple[int, str]]:
    """Load ml_manual_labels keyed by lower-case ip_hash hex."""
    import psycopg2

    labels: dict[str, tuple[int, str]] = {}
    conn = psycopg2.connect(dsn)
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT ip_hash, label, source FROM ml_manual_labels")
            for ip_hash, label, source in cur.fetchall():
                if ip_hash is None or label is None:
                    continue
                labels[str(ip_hash).lower()] = (int(label), str(source or "manual"))
    finally:
        conn.close()
    return labels


def _rows_with_labels(
    rows: list[tuple],
    manual_labels: dict[str, tuple[int, str]],
) -> list[tuple]:
    """LEFT JOIN manual labels on ip_hash_hex."""
    ip_idx = EXPORT_COLUMNS.index("ip_hash_hex")
    enriched: list[tuple] = []
    for row in rows:
        ip_hex = row[ip_idx]
        label_value: int | None = None
        label_source: str | None = None
        if ip_hex is not None:
            match = manual_labels.get(str(ip_hex).lower())
            if match is not None:
                label_value, label_source = match
        enriched.append((*row, label_value, label_source))
    return enriched


def export_features(
    output: Path,
    *,
    since: datetime,
    until: datetime,
    fmt: str,
    db_dsn: str = "",
) -> int:
    client = connect_client()
    if not ping_client(client):
        raise ConnectionError("clickhouse ping failed")

    params = {"since": since, "until": until}
    result = client.query(EXPORT_SQL, parameters=params)
    rows = result.result_rows
    columns = EXPORT_COLUMNS
    if db_dsn:
        manual_labels = _load_manual_labels_from_pg(db_dsn)
        rows = _rows_with_labels(rows, manual_labels)
        columns = EXPORT_COLUMNS + LABEL_COLUMNS
    row_count = len(rows)

    if fmt == "csv":
        if str(output) == "-":
            writer = csv.writer(sys.stdout)
            writer.writerow(columns)
            writer.writerows(rows)
        else:
            output.parent.mkdir(parents=True, exist_ok=True)
            with output.open("w", encoding="utf-8", newline="") as handle:
                writer = csv.writer(handle)
                writer.writerow(columns)
                writer.writerows(rows)
        return row_count

    if fmt == "parquet":
        import pyarrow as pa
        import pyarrow.parquet as pq

        column_data = {name: [] for name in columns}
        for row in rows:
            for idx, name in enumerate(columns):
                column_data[name].append(row[idx])
        table = pa.table(column_data)
        if str(output) == "-":
            pq.write_table(table, sys.stdout.buffer)
        else:
            output.parent.mkdir(parents=True, exist_ok=True)
            pq.write_table(table, output)
        return row_count

    raise ValueError(f"unsupported format: {fmt}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Export fraud training features from ClickHouse")
    parser.add_argument("--output", default="-", help="output path or - for stdout")
    parser.add_argument("--format", choices=("csv", "parquet"), default="", help="csv or parquet")
    parser.add_argument("--since", default="", help="ISO-8601 lower bound on window_start (UTC)")
    parser.add_argument("--until", default="", help="ISO-8601 upper bound on window_start (UTC)")
    parser.add_argument(
        "--smoke",
        action="store_true",
        help="connect + export empty window; 0 rows is success",
    )
    parser.add_argument(
        "--allow-offline",
        action="store_true",
        help="exit 0 when ClickHouse is unreachable",
    )
    args = parser.parse_args()

    fmt = args.format
    if not fmt:
        if args.output.endswith(".parquet"):
            fmt = "parquet"
        else:
            fmt = "csv"

    if args.smoke:
        since = datetime(1970, 1, 1, tzinfo=UTC)
        until = datetime(1970, 1, 1, 1, tzinfo=UTC)
    else:
        since, until = _default_window(args.since, args.until)

    db_dsn = os.environ.get("DB_DSN", "")

    try:
        rows = export_features(
            Path(args.output),
            since=since,
            until=until,
            fmt=fmt,
            db_dsn=db_dsn,
        )
    except ImportError as err:
        print(f"features_export: missing dependency: {err}", file=sys.stderr)
        return 1
    except ConnectionError as err:
        if args.allow_offline or args.smoke:
            clickhouse_config = ch_config_from_env()
            print(
                f"features_export: skip ({err}) host={clickhouse_config.host}:"
                f"{clickhouse_config.port} db={clickhouse_config.database}",
                file=sys.stderr,
            )
            return 0
        print(f"features_export: {err}", file=sys.stderr)
        return 1
    except OSError as err:
        if args.allow_offline:
            print(f"features_export: skip ({err})", file=sys.stderr)
            return 0
        print(f"features_export: {err}", file=sys.stderr)
        return 1

    label_note = " with manual labels" if db_dsn else ""
    print(f"features_export: {rows} rows{label_note}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
