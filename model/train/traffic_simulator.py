"""Labeled synthetic traffic archetypes for bootstrap and offline benchmarks.

Role:
- Sample realistic ml_features_1m-shaped rows with known fraud/legit labels.
- ARCHETYPES weights drive class balance in bootstrap_synthetic() holdout calibration.

Archetype cohorts:
- Legit: organic_display, mobile_in_app, search_brand, affiliate_long_tail, grey_noise
- Fraud: click_farm, bot_script, residential_proxy_bot, budget_drain

Verify:
  python3 model/eval/simulation_benchmark.py --simulate --retrain
  pytest model/tests/test_labeled_dataset.py -q
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass

import numpy as np

Row = dict[str, int]

@dataclass(frozen=True)
class TrafficArchetype:
    """Named sampler with fraud label and mixture weight."""

    name: str
    is_fraud: bool
    weight: float
    sampler: Callable[[np.random.Generator], Row]

def _clamp_int(value: float, lo: int, hi: int) -> int:
    return int(max(lo, min(hi, round(value))))

def _budget_spend(rng: np.random.Generator, spend_ratio_mean: float) -> tuple[int, int]:
    budget = int(rng.integers(2_000_000, 250_000_000))
    ratio = float(np.clip(rng.beta(spend_ratio_mean * 20, 20), 0.0, 1.0))
    spend = int(budget * ratio)
    return spend, budget

def sample_organic_display(rng: np.random.Generator) -> Row:
    events = _clamp_int(rng.lognormal(mean=4.2, sigma=1.1), 20, 8000)
    ctr = float(rng.beta(1.5, 350))
    clicks = _clamp_int(events * ctr, 0, events)
    unique_users = _clamp_int(rng.lognormal(2.8, 0.7), 3, max(3, events // 3))
    unique_uas = _clamp_int(unique_users * rng.uniform(0.75, 1.15) + rng.normal(0, 1.5), 2, unique_users + 8)
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.25)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_mobile_in_app(rng: np.random.Generator) -> Row:
    events = _clamp_int(rng.lognormal(mean=5.0, sigma=0.9), 50, 12000)
    ctr = float(rng.beta(2.5, 120))
    clicks = _clamp_int(events * ctr, 0, events)
    unique_users = _clamp_int(rng.lognormal(3.5, 0.5), 10, max(10, events // 4))
    unique_uas = _clamp_int(unique_users * rng.uniform(0.85, 1.05), unique_users, unique_users + 12)
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.35)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_search_brand(rng: np.random.Generator) -> Row:
    events = _clamp_int(rng.lognormal(mean=3.8, sigma=0.8), 10, 3000)
    ctr = float(rng.beta(3.0, 80))
    clicks = _clamp_int(events * ctr, 0, events)
    unique_users = _clamp_int(rng.lognormal(2.5, 0.6), 2, max(2, events // 2))
    unique_uas = _clamp_int(unique_users * rng.uniform(0.9, 1.2), unique_users, unique_users + 5)
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.4)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_affiliate_long_tail(rng: np.random.Generator) -> Row:
    events = _clamp_int(rng.lognormal(mean=3.2, sigma=1.3), 5, 1500)
    ctr = float(rng.beta(1.2, 200))
    clicks = _clamp_int(events * ctr, 0, events)
    unique_users = _clamp_int(rng.lognormal(1.8, 0.9), 1, max(1, events // 2))
    unique_uas = _clamp_int(unique_users * rng.uniform(0.6, 1.4) + rng.integers(0, 4), 1, unique_users + 10)
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.18)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_click_farm(rng: np.random.Generator) -> Row:
    events = int(rng.integers(120, 1200))
    ctr = float(rng.uniform(0.06, 0.28))
    clicks = _clamp_int(events * ctr, 1, events)
    unique_uas = int(rng.integers(1, 4))
    unique_users = int(rng.integers(unique_uas * 3, unique_uas * 15 + 2))
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.55)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_bot_script(rng: np.random.Generator) -> Row:
    events = int(rng.integers(150, 3500))
    ctr = float(rng.uniform(0.35, 0.98))
    clicks = _clamp_int(events * ctr, 1, events)
    unique_uas = 1
    unique_users = int(rng.integers(1, 4))
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.45)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_residential_proxy_bot(rng: np.random.Generator) -> Row:
    events = int(rng.integers(120, 950))
    ctr = float(rng.beta(1.5, 120))
    clicks = _clamp_int(events * ctr, 1, max(1, events // 20))
    unique_uas = int(rng.integers(4, 14))
    unique_users = int(rng.integers(20, 140))
    spend, budget = _budget_spend(rng, spend_ratio_mean=0.28)
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_budget_drain(rng: np.random.Generator) -> Row:
    events = int(rng.integers(60, 600))
    ctr = float(rng.uniform(0.45, 0.92))
    clicks = _clamp_int(events * ctr, 1, events)
    unique_uas = int(rng.integers(1, 3))
    unique_users = int(rng.integers(1, 6))
    budget = int(rng.integers(10_000_000, 80_000_000))
    spend = int(budget * rng.uniform(0.88, 0.995))
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

def sample_grey_noise(rng: np.random.Generator) -> Row:
    events = int(rng.integers(0, 2500))
    clicks = int(rng.integers(0, max(1, events + 1)))
    clicks = min(clicks, events)
    unique_users = int(rng.integers(1, max(2, events // 3 + 2)))
    unique_uas = int(rng.integers(1, max(2, unique_users + 3)))
    budget = int(rng.integers(0, 200_000_000))
    spend = int(rng.integers(0, max(1, budget))) if budget > 0 else 0
    return {
        "events": events,
        "clicks": clicks,
        "spend_micro": spend,
        "budget_limit_micro": budget,
        "unique_users": unique_users,
        "unique_uas": unique_uas,
    }

ARCHETYPES: tuple[TrafficArchetype, ...] = (
    TrafficArchetype("organic_display", False, 0.36, sample_organic_display),
    TrafficArchetype("mobile_in_app", False, 0.14, sample_mobile_in_app),
    TrafficArchetype("search_brand", False, 0.10, sample_search_brand),
    TrafficArchetype("affiliate_long_tail", False, 0.12, sample_affiliate_long_tail),
    TrafficArchetype("grey_noise", False, 0.06, sample_grey_noise),
    TrafficArchetype("click_farm", True, 0.12, sample_click_farm),
    TrafficArchetype("bot_script", True, 0.06, sample_bot_script),
    TrafficArchetype("residential_proxy_bot", True, 0.03, sample_residential_proxy_bot),
    TrafficArchetype("budget_drain", True, 0.01, sample_budget_drain),
)
# Weights sum to 1.0; residential_proxy_bot used in calibrate_policy proxy_recall objective.

def generate_network_batch(
    count: int,
    seed: int,
) -> tuple[list[Row], list[int], list[str]]:
    """Sample count rows by ARCHETYPES weights; return rows, labels, archetype names."""
    rng = np.random.default_rng(seed)
    weights = np.array([a.weight for a in ARCHETYPES], dtype=np.float64)
    weights /= weights.sum()

    rows: list[Row] = []
    labels: list[int] = []
    archetypes: list[str] = []

    for _ in range(count):
        idx = int(rng.choice(len(ARCHETYPES), p=weights))
        archetype = ARCHETYPES[idx]
        row = archetype.sampler(rng)
        rows.append(row)
        labels.append(1 if archetype.is_fraud else 0)
        archetypes.append(archetype.name)

    return rows, labels, archetypes
