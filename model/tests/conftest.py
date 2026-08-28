"""Shared pytest hooks and fixtures for model package tests."""

# pyright: reportUnusedFunction=false

from __future__ import annotations

import pytest

from contract.policy_config import default_policy_config, set_policy_config

@pytest.fixture(autouse=True)
def reset_policy_config() -> None:
    set_policy_config(default_policy_config())
