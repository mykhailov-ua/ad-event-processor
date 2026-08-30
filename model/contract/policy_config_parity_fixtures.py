"""Shared policy_config parity fixtures for Python and Go tests.

Role:
- Load internal/fraud/testdata/policy_config_parity.json cases.
- Both pytest and Go tests assert identical merge/env parsing outcomes.

Verify:
  pytest model/tests/test_policy_config_parity.py -q
  go test ./internal/fraud/ -short -run TestPolicyConfigParity -count=1
"""

from __future__ import annotations

import json
from typing import Any

from contract.fixture_catalog import TRACKED_FIXTURE_DIR

POLICY_CONFIG_PARITY_PATH = TRACKED_FIXTURE_DIR / "policy_config_parity.json"


def load_policy_config_parity_cases() -> list[dict[str, Any]]:
    """Return the ``cases`` list from policy_config_parity.json."""
    if not POLICY_CONFIG_PARITY_PATH.is_file():
        raise FileNotFoundError(f"missing {POLICY_CONFIG_PARITY_PATH}")
    payload = json.loads(POLICY_CONFIG_PARITY_PATH.read_text(encoding="utf-8"))
    cases = payload.get("cases")
    if not isinstance(cases, list) or not cases:
        raise ValueError("policy_config_parity.json must contain a non-empty cases list")
    return cases
