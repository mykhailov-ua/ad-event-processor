"""Feature vector contract: must match internal/fraud/features.go and tracked fixtures."""

from __future__ import annotations

import json
import math
from pathlib import Path

import pytest

from contract.feature_spec import FEATURE_DIMS, FEATURE_NAMES, row_to_vector
from contract.fixture_catalog import FIXTURE_ROWS, TRACKED_FIXTURE_DIR

def _feature_fixture_paths() -> list[Path]:
    paths = sorted(TRACKED_FIXTURE_DIR.glob("features_*.json"))
    if not paths:
        pytest.fail(f"no fixtures under {TRACKED_FIXTURE_DIR}; run python3 -m train.fixture_generator")
    return paths

def test_feature_dims_match_go_contract() -> None:
    assert FEATURE_DIMS == 16
    assert len(FEATURE_NAMES) == FEATURE_DIMS

def test_row_to_vector_matches_tracked_fixtures() -> None:
    for path in _feature_fixture_paths():
        payload = json.loads(path.read_text(encoding="utf-8"))
        assert payload["feature_names"] == list(FEATURE_NAMES), path.name
        expected = payload["vector"]
        actual = row_to_vector(payload["row"])
        assert len(actual) == FEATURE_DIMS, path.name
        for index, (got, want) in enumerate(zip(actual, expected, strict=True)):
            assert math.isclose(got, want, rel_tol=0.0, abs_tol=1e-9), f"{path.name}[{index}]"

def test_fixture_catalog_rows_match_tracked_files() -> None:
    catalog_ids = {fixture_id for fixture_id, _ in FIXTURE_ROWS}
    file_ids = {path.stem.replace("features_", "") for path in _feature_fixture_paths()}
    assert catalog_ids == file_ids

def test_row_to_vector_changes_when_formula_breaks() -> None:
    """Fails if row_to_vector is stubbed or FEATURE_NAMES order drifts."""
    _, row = FIXTURE_ROWS[0]
    vector = row_to_vector(row)
    assert vector[0] == float(row["events"])
    assert vector[1] == float(row["clicks"])
    assert vector[2] == row["clicks"] / row["events"]
