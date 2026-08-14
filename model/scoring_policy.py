"""Post-ML heuristics; mirrors internal/fraud/scoring_policy.go."""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from policy_config import PolicyConfig, get_policy_config


@dataclass(frozen=True)
class TierDecision:
    tier: str
    score: int
    ml_probability: float
    adjusted_probability: float
    residential_proxy: bool
    structural_fraud: bool
    fp_guard_applied: bool


def probability_to_score(probability: float) -> int:
    """[0,1] -> 0..100; same rounding as Go ProbabilityToFraudScore."""
    bounded = max(0.0, min(1.0, probability))
    return int(bounded * 100 + 0.5)


def map_probability_tier_with_thresholds(
    probability: float,
    tier_pass: int,
    tier_suspect: int,
    tier_ivt: int,
    tier_block: int,
) -> tuple[str, int]:
    score = probability_to_score(probability)
    if score <= tier_pass:
        return "pass", score
    if score <= tier_suspect:
        return "suspect", score
    if score <= tier_ivt:
        return "ivt", score
    if score <= tier_block:
        return "block", score
    return "block", score


def map_probability_tier(probability: float) -> tuple[str, int]:
    cfg = get_policy_config()
    return map_probability_tier_with_thresholds(
        probability,
        cfg.tier_pass,
        cfg.tier_suspect,
        cfg.tier_ivt,
        cfg.tier_block,
    )


def _row_metrics(row: dict[str, int], cfg: PolicyConfig) -> dict[str, float]:
    events = int(row.get("events", 0))
    clicks = int(row.get("clicks", 0))
    unique_users = int(row.get("unique_users", 0))
    unique_uas = int(row.get("unique_uas", 0))
    spend_micro = int(row.get("spend_micro", 0))
    budget_limit_micro = int(row.get("budget_limit_micro", 0))

    events_f = float(events)
    clicks_f = float(clicks)
    users_f = float(unique_users)
    uas_f = float(unique_uas)
    ctr = clicks_f / events_f if events > 0 else 0.0
    spend_ratio = float(spend_micro) / float(budget_limit_micro) if budget_limit_micro > 0 else 0.0

    return {
        "events": events_f,
        "clicks": clicks_f,
        "ctr": ctr,
        "spend_ratio": spend_ratio,
        "unique_users": users_f,
        "unique_uas": uas_f,
        "events_per_user": events_f / users_f if users_f > 0 else 0.0,
        "impression_pressure": events_f / (clicks_f + 1.0),
        "user_click_gap": users_f / (clicks_f + 1.0),
        "events_per_ua": events_f / uas_f if uas_f > 0 else 0.0,
        "clicks_per_user": clicks_f / users_f if users_f > 0 else 0.0,
        "users_per_ua": users_f / uas_f if uas_f > 0 else 0.0,
    }


def residential_proxy_signal_with_config(row: dict[str, int], cfg: PolicyConfig) -> bool:
    """Low CTR, high volume, dispersed users/UAs; residential proxy farm pattern."""
    m = _row_metrics(row, cfg)
    if m["events"] < cfg.proxy_min_events:
        return False
    if m["ctr"] > cfg.proxy_max_ctr:
        return False
    if m["unique_users"] < cfg.proxy_min_users:
        return False
    if m["user_click_gap"] < cfg.proxy_min_user_click_gap:
        return False
    if m["events_per_user"] < cfg.proxy_min_events_per_user:
        return False
    if m["impression_pressure"] < cfg.proxy_min_impression_pressure:
        return False
    if m["users_per_ua"] < cfg.proxy_min_users_per_ua:
        return False
    return not m["clicks"] < cfg.proxy_min_clicks


def residential_proxy_signal(row: dict[str, int]) -> bool:
    return residential_proxy_signal_with_config(row, get_policy_config())


def structural_fraud_signal_with_config(row: dict[str, int], cfg: PolicyConfig) -> bool:
    """Single UA, extreme CTR, or budget drain; exempt from FP-guard."""
    m = _row_metrics(row, cfg)
    if m["ctr"] > cfg.structural_high_ctr and m["unique_users"] <= cfg.structural_max_users:
        return True
    if m["unique_uas"] <= 1 and m["events"] >= cfg.structural_min_events:
        return True
    if m["events_per_ua"] >= cfg.structural_min_events_per_ua:
        return True
    if m["clicks_per_user"] >= cfg.structural_min_clicks_per_user:
        return True
    return m["spend_ratio"] > cfg.structural_spend_ratio and m["ctr"] > cfg.structural_spend_min_ctr


def structural_fraud_signal(row: dict[str, int]) -> bool:
    return structural_fraud_signal_with_config(row, get_policy_config())


def precompute_row_signals(
    rows: list[dict[str, int]],
    cfg: PolicyConfig,
) -> tuple[np.ndarray, np.ndarray]:
    """Proxy and structural flags per row; stable while calibration grid varies floors."""
    n = len(rows)
    proxy = np.empty(n, dtype=bool)
    structural = np.empty(n, dtype=bool)
    for i, row in enumerate(rows):
        proxy[i] = residential_proxy_signal_with_config(row, cfg)
        structural[i] = structural_fraud_signal_with_config(row, cfg)
    return proxy, structural


def policy_scores_vector(
    probs: np.ndarray,
    proxy: np.ndarray,
    structural: np.ndarray,
    proxy_floor: float,
    proxy_max_ml: float,
    fp_guard_cap: float,
    block_prob: float,
) -> np.ndarray:
    """Vectorized adjust_probability + score; matches row-wise policy for fixed tier thresholds."""
    adj = np.asarray(probs, dtype=np.float64).copy()
    boost = proxy & (adj < proxy_max_ml)
    adj[boost] = np.maximum(adj[boost], proxy_floor)
    cap = (adj >= block_prob) & (~structural) & (~proxy)
    adj[cap] = fp_guard_cap
    return (np.clip(adj, 0.0, 1.0) * 100.0 + 0.5).astype(np.int32)


def scores_suspect_positive(scores: np.ndarray, tier_pass: int) -> np.ndarray:
    return scores > tier_pass


def scores_block_positive(scores: np.ndarray, tier_ivt: int) -> np.ndarray:
    return scores > tier_ivt


def adjust_probability_with_config(
    row: dict[str, int],
    ml_probability: float,
    cfg: PolicyConfig,
) -> tuple[float, bool, bool, bool]:
    """Returns (adjusted_prob, proxy_hit, structural_hit, fp_guard_applied)."""
    prob = float(ml_probability)
    proxy = residential_proxy_signal_with_config(row, cfg)
    structural = structural_fraud_signal_with_config(row, cfg)
    fp_guard = False

    if proxy and prob < cfg.residential_proxy_max_ml:
        prob = max(prob, cfg.residential_proxy_floor)

    if prob >= cfg.block_probability() and not structural and not proxy:
        prob = cfg.fp_guard_cap
        fp_guard = True

    return prob, proxy, structural, fp_guard


def adjust_probability(row: dict[str, int], ml_probability: float) -> tuple[float, bool, bool, bool]:
    return adjust_probability_with_config(row, ml_probability, get_policy_config())


def decide_with_policy(row: dict[str, int], ml_probability: float, cfg: PolicyConfig) -> TierDecision:
    adjusted, proxy, structural, fp_guard = adjust_probability_with_config(row, ml_probability, cfg)
    tier, score = map_probability_tier_with_thresholds(
        adjusted,
        cfg.tier_pass,
        cfg.tier_suspect,
        cfg.tier_ivt,
        cfg.tier_block,
    )
    return TierDecision(
        tier=tier,
        score=score,
        ml_probability=ml_probability,
        adjusted_probability=adjusted,
        residential_proxy=proxy,
        structural_fraud=structural,
        fp_guard_applied=fp_guard,
    )


def decide_with_campaign(
    row: dict[str, int],
    ml_probability: float,
    tier_pass: int,
    tier_suspect: int,
    tier_ivt: int,
    tier_block: int,
) -> TierDecision:
    """Campaign tier overrides; 0 keeps global default."""
    base = get_policy_config()
    cfg = PolicyConfig(
        tier_pass=tier_pass or base.tier_pass,
        tier_suspect=tier_suspect or base.tier_suspect,
        tier_ivt=tier_ivt or base.tier_ivt,
        tier_block=tier_block or base.tier_block,
        ml_threshold=base.ml_threshold,
        residential_proxy_floor=base.residential_proxy_floor,
        residential_proxy_max_ml=base.residential_proxy_max_ml,
        fp_guard_cap=base.fp_guard_cap,
        proxy_min_events=base.proxy_min_events,
        proxy_max_ctr=base.proxy_max_ctr,
        proxy_min_users=base.proxy_min_users,
        proxy_min_user_click_gap=base.proxy_min_user_click_gap,
        proxy_min_events_per_user=base.proxy_min_events_per_user,
        proxy_min_impression_pressure=base.proxy_min_impression_pressure,
        proxy_min_users_per_ua=base.proxy_min_users_per_ua,
        proxy_min_clicks=base.proxy_min_clicks,
        structural_high_ctr=base.structural_high_ctr,
        structural_max_users=base.structural_max_users,
        structural_min_events=base.structural_min_events,
        structural_min_events_per_ua=base.structural_min_events_per_ua,
        structural_min_clicks_per_user=base.structural_min_clicks_per_user,
        structural_spend_ratio=base.structural_spend_ratio,
        structural_spend_min_ctr=base.structural_spend_min_ctr,
    )
    return decide_with_policy(row, ml_probability, cfg)


def decide(row: dict[str, int], ml_probability: float) -> TierDecision:
    return decide_with_policy(row, ml_probability, get_policy_config())


def action_fraud_positive(decision: TierDecision, *, block_only: bool = False) -> bool:
    if block_only:
        return decision.tier == "block"
    return decision.tier in {"suspect", "ivt", "block"}
