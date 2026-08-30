"""Offline evaluation, shadow precision reports, and Postgres eval persistence.

Role:
- shadow_precision: CH SQL for proxy and audited precision vs ml_shadow_scores.
- evaluate: CLI composing proxy + audited + drift; writes JSON/markdown reports.
- postgres_eval_store: upsert ml_eval_reports for control plane readers.
- simulation_benchmark: synthetic traffic ML vs policy benchmark.

Verify:
  pytest model/tests/test_shadow_precision.py model/tests/test_postgres_eval_store.py -q
  python3 model/eval/evaluate.py --allow-offline --hours 24
"""
