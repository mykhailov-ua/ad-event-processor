"""Shared ML feature fixtures for Go ml-validate and Python/Go vector tests."""

from __future__ import annotations

from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
TRACKED_FIXTURE_DIR = REPO_ROOT / "internal" / "fraud" / "testdata" / "ml"
EPHEMERAL_FIXTURE_DIR = REPO_ROOT / "var" / "fraudscore" / "fixtures"

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
