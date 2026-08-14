"""policy_config parity vs internal/fraud/testdata/ml/policy_config_parity.json."""

from __future__ import annotations

import json
import math
import os
from pathlib import Path
from typing import Any

import pytest

from policy_config import (
    load_policy_config_from_env,
    load_policy_from_metadata,
    resolve_policy_config,
)
from policy_config_parity_fixtures import load_policy_config_parity_cases


@pytest.mark.parametrize("case", load_policy_config_parity_cases(), ids=lambda case: str(case["id"]))
def test_policy_config_parity_fixtures(case: dict[str, Any], tmp_path: Path) -> None:
    metadata_path = tmp_path / "metadata.json"
    metadata_path.write_text(json.dumps({"policy": case["policy"]}), encoding="utf-8")
    want = case["want"]
    check = case["check"]

    if check == "load_metadata":
        loaded = load_policy_from_metadata(metadata_path)
        assert loaded is not None
        assert math.isclose(loaded.ml_threshold, float(want["ml_threshold"]), abs_tol=1e-9)
        assert math.isclose(loaded.residential_proxy_floor, float(want["residential_proxy_floor"]), abs_tol=1e-9)
        return

    if check == "resolve_auto":
        env = case.get("env", {})
        previous: dict[str, str | None] = {}
        for key, value in env.items():
            previous[key] = os.environ.get(key)
            os.environ[key] = str(value)
        try:
            env_policy = load_policy_config_from_env()
            resolved = resolve_policy_config(env_policy, metadata_path, "auto")
        finally:
            for key, value in previous.items():
                if value is None:
                    os.environ.pop(key, None)
                else:
                    os.environ[key] = value
        assert math.isclose(resolved.ml_threshold, float(want["ml_threshold"]), abs_tol=1e-9)
        assert math.isclose(resolved.fp_guard_cap, float(want["fp_guard_cap"]), abs_tol=1e-9)
        return

    pytest.fail(f"unknown check {check!r}")
