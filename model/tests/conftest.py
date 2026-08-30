"""Shared pytest hooks and fixtures for model package tests.

Role: Reset global policy_config singleton between tests so env/metadata cases do not leak.
Tier: fast (unit).
Infra: none; in-process contract.policy_config only.
Invariants proved: autouse fixture restores default_policy_config after every test.
Verify: cd model && python3 -m pytest tests/ -q
"""

# pyright: reportUnusedFunction=false

from __future__ import annotations

import pytest

from contract.policy_config import default_policy_config, set_policy_config

@pytest.fixture(autouse=True)
def reset_policy_config() -> None:
    set_policy_config(default_policy_config())
