#!/usr/bin/env python3
"""Write tracked + ephemeral features_*.json fixtures for ml-validate."""

from __future__ import annotations

import json
from pathlib import Path

from contract.feature_spec import FEATURE_NAMES, row_to_vector
from contract.fixture_catalog import EPHEMERAL_FIXTURE_DIR, FIXTURE_ROWS, TRACKED_FIXTURE_DIR
from repo_paths import REPO_ROOT

def write_fixture_dirs(*dirs: Path) -> int:
    vectors = [row_to_vector(row) for _, row in FIXTURE_ROWS]
    written = 0

    for out_dir in dirs:
        out_dir.mkdir(parents=True, exist_ok=True)
        for idx, (fixture_id, row) in enumerate(FIXTURE_ROWS):
            payload: dict[str, object] = {
                "id": fixture_id,
                "feature_names": list(FEATURE_NAMES),
                "row": row,
                "vector": vectors[idx],
            }
            out_path = out_dir / f"features_{fixture_id}.json"
            with out_path.open("w", encoding="utf-8") as handle:
                json.dump(payload, handle, indent=2)
                handle.write("\n")
            written += 1
            print(f"wrote {out_path.relative_to(REPO_ROOT)}")

    return written

def main() -> int:
    write_fixture_dirs(TRACKED_FIXTURE_DIR, EPHEMERAL_FIXTURE_DIR)
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
