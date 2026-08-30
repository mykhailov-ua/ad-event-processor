"""Shared ML feature fixtures for Go ml-validate and Python/Go vector tests.

Role:
- FIXTURE_ROWS: named (id, row) pairs covering edge cases (zero budget, bot UA, proxy farm).
- Writers emit features_<id>.json with row + precomputed vector for cross-language checks.

Fixture ids:
- basic, high_volume: nominal traffic
- zero_events, zero_budget: denominator guards in row_to_vector
- bot_single_ua, residential_proxy: policy heuristic regression rows

Verify:
  python3 model/train/fixture_generator.py
  go run ./cmd/ml-validate -fixtures internal/fraud/testdata
  pytest model/tests/test_bootstrap_contract.py -q
"""

from __future__ import annotations

from repo_paths import EPHEMERAL_FIXTURE_DIR, REPO_ROOT, TRACKED_FIXTURE_DIR

__all__ = [
    "EPHEMERAL_FIXTURE_DIR",
    "FIXTURE_ROWS",
    "REPO_ROOT",
    "TRACKED_FIXTURE_DIR",
]

FIXTURE_ROWS: list[tuple[str, dict[str, int]]] = [
    (
        "basic",
        {
            "events": 10,
            "clicks": 2,
            "spend_micro": 1_000_000,
            "budget_limit_micro": 5_000_000,
            "unique_users": 1,
            "unique_uas": 1,
        },
    ),
    (
        "high_volume",
        {
            "events": 100,
            "clicks": 10,
            "spend_micro": 10_000_000,
            "budget_limit_micro": 50_000_000,
            "unique_users": 5,
            "unique_uas": 2,
        },
    ),
    (
        "zero_events",
        {
            "events": 0,
            "clicks": 0,
            "spend_micro": 0,
            "budget_limit_micro": 0,
            "unique_users": 0,
            "unique_uas": 0,
        },
    ),
    (
        "zero_budget",
        {
            "events": 50,
            "clicks": 5,
            "spend_micro": 2_000_000,
            "budget_limit_micro": 0,
            "unique_users": 3,
            "unique_uas": 2,
        },
    ),
    (
        "bot_single_ua",
        {
            "events": 200,
            "clicks": 180,
            "spend_micro": 50_000_000,
            "budget_limit_micro": 60_000_000,
            "unique_users": 2,
            "unique_uas": 1,
        },
    ),
    (
        "residential_proxy",
        {
            "events": 275,
            "clicks": 4,
            "spend_micro": 6_130_354,
            "budget_limit_micro": 22_060_077,
            "unique_users": 32,
            "unique_uas": 11,
        },
    ),
]
