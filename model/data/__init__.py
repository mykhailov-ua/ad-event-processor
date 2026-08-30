"""ClickHouse and Postgres data export for ML training pipelines.

Role:
- clickhouse_client: CH_DSN / CH_READONLY_DSN HTTP client for batch reads.
- features_export: ml_features_1m window export to parquet/csv.
- manual_labels_export: ml_manual_labels feedback loop from Postgres.

Cold path only; tracker hot path reads Redis boost snapshot, not these modules.

Verify:
  pytest model/tests/test_clickhouse_client.py -q
  python3 model/data/features_export.py --smoke --allow-offline
  DB_DSN=... python3 model/data/manual_labels_export.py
"""
