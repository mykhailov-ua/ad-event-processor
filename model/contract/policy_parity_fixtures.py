"""Shared scoring_policy parity fixtures for Python and Go tests.

Role:
- Load internal/fraud/testdata/policy_parity.json (row, ml_prob, expected tier/score).
- Keeps heuristic adjustments aligned between Python train/eval and cmd/fraud-scorer.

Verify:
  pytest model/tests/test_scoring_policy_parity.py -q
  go test ./internal/fraud/ -short -run TestPolicyParity -count=1
"""

from __future__ import annotations

import json
from typing import Any

from contract.fixture_catalog import TRACKED_FIXTURE_DIR

POLICY_PARITY_PATH = TRACKED_FIXTURE_DIR / "policy_parity.json"


def load_policy_parity_cases() -> list[dict[str, Any]]:
    """Return the ``cases`` list from policy_parity.json."""
    if not POLICY_PARITY_PATH.is_file():
        raise FileNotFoundError(f"missing {POLICY_PARITY_PATH}; commit internal/fraud/testdata/policy_parity.json")
    payload = json.loads(POLICY_PARITY_PATH.read_text(encoding="utf-8"))
    cases = payload.get("cases")
    if not isinstance(cases, list) or not cases:
        raise ValueError("policy_parity.json must contain a non-empty cases list")
    return cases
