#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import sys
from pathlib import Path

from feature_spec import FEATURE_DIMS, FEATURE_NAMES, row_to_vector

REPO_ROOT = Path(__file__).resolve().parent.parent
OUT_DIR = REPO_ROOT / "testdata" / "ml"
MODEL_PATH = REPO_ROOT / "internal" / "fraudscoring" / "testdata" / "model.txt"

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
]

KNOWN_SCORES: dict[str, float] = {
    "basic": 0.52497,
    "high_volume": 0.71094,
}


def score_vectors(vectors: list[list[float]]) -> list[float] | None:
    try:
        import lightgbm as lgb
    except ImportError:
        return None
    if not MODEL_PATH.is_file():
        return None
    booster = lgb.Booster(model_file=str(MODEL_PATH))
    if booster.num_feature() != FEATURE_DIMS:
        raise SystemExit(f"model num_feature={booster.num_feature()} want {FEATURE_DIMS}")
    import numpy as np

    matrix = np.array(vectors, dtype=np.float64)
    return [float(x) for x in booster.predict(matrix)]


def main() -> int:
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    vectors = [row_to_vector(row) for _, row in FIXTURE_ROWS]
    scores = score_vectors(vectors)

    for idx, (fixture_id, row) in enumerate(FIXTURE_ROWS):
        payload: dict[str, object] = {
            "id": fixture_id,
            "feature_names": list(FEATURE_NAMES),
            "row": row,
            "vector": vectors[idx],
        }
        if scores is not None:
            payload["score"] = scores[idx]
        elif fixture_id in KNOWN_SCORES:
            payload["score"] = KNOWN_SCORES[fixture_id]

        out_path = OUT_DIR / f"features_{fixture_id}.json"
        with open(out_path, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, indent=2)
            handle.write("\n")
        print(f"wrote {out_path.relative_to(REPO_ROOT)}")

    return 0


if __name__ == "__main__":
    os.chdir(Path(__file__).resolve().parent)
    raise SystemExit(main())
