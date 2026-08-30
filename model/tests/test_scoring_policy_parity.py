"""Scoring policy parity vs internal/fraud/testdata/policy_parity.json.

Role: adjust_probability, decide, residential_proxy_signal match Go policy_parity.json rows.
Tier: fast (unit).
Infra: fixture JSON only; no ML model inference.
Invariants proved: tier, score, adjusted_probability, fp_guard flags per parametrized case id.
Verify: cd model && python3 -m pytest tests/test_scoring_policy_parity.py -q
"""

from __future__ import annotations

import math
from typing import Any

import pytest

from contract.policy_parity_fixtures import load_policy_parity_cases
from contract.scoring_policy import adjust_probability, decide, residential_proxy_signal

def _row(raw: dict[str, Any]) -> dict[str, int]:
    return {key: int(raw[key]) for key in raw}

@pytest.mark.parametrize("case", load_policy_parity_cases(), ids=lambda case: str(case["id"]))
def test_scoring_policy_parity_fixtures(case: dict[str, Any]) -> None:
    op = case["op"]
    row = _row(case["row"])
    want = case["want"]

    if op == "residential_proxy_signal":
        assert residential_proxy_signal(row) == bool(want)
        return

    ml_probability = float(case["ml_probability"])

    if op == "adjust_probability":
        adjusted, proxy, structural, fp_guard = adjust_probability(row, ml_probability)
        assert math.isclose(adjusted, float(want["adjusted_probability"]), abs_tol=1e-9)
        assert proxy == bool(want["residential_proxy"])
        assert structural == bool(want["structural_fraud"])
        assert fp_guard == bool(want["fp_guard_applied"])
        return

    if op == "decide":
        decision = decide(row, ml_probability)
        assert decision.tier == str(want["tier"])
        assert decision.score == int(want["score"])
        assert math.isclose(decision.adjusted_probability, float(want["adjusted_probability"]), abs_tol=1e-9)
        assert decision.residential_proxy == bool(want["residential_proxy"])
        assert decision.structural_fraud == bool(want["structural_fraud"])
        assert decision.fp_guard_applied == bool(want["fp_guard_applied"])
        return

    pytest.fail(f"unknown op {op!r}")
