#!/usr/bin/env python3
"""Write testdata/ml/features_*.json for cmd/ml-validate and bootstrap validate."""
from __future__ import annotations

import json
import os
from pathlib import Path

from feature_spec import FEATURE_NAMES, row_to_vector

REPO_ROOT = Path(__file__).resolve().parent.parent
OUT_DIR = REPO_ROOT / "testdata" / "ml"

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


def main() -> int:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    vectors = [row_to_vector(row) for _, row in FIXTURE_ROWS]

    for idx, (fixture_id, row) in enumerate(FIXTURE_ROWS):
        payload: dict[str, object] = {
            "id": fixture_id,
            "feature_names": list(FEATURE_NAMES),
            "row": row,
            "vector": vectors[idx],
        }

        out_path = OUT_DIR / f"features_{fixture_id}.json"
        with open(out_path, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, indent=2)
            handle.write("\n")
        print(f"wrote {out_path.relative_to(REPO_ROOT)}")

    return 0


if __name__ == "__main__":
    os.chdir(Path(__file__).resolve().parent)
    raise SystemExit(main())
