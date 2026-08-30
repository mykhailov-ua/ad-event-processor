"""FRAUD_POLICY_* env parsing and metadata.json policy section.

Role:
- Load tier thresholds and heuristic knobs for scoring_policy.py calibration.
- Merge env overrides onto metadata policy when FRAUD_POLICY_SOURCE=auto.

Go mirror: internal/fraud/policy_config.go (field names and defaults must match).

Env defaults (balanced preset tiers):
- FRAUD_POLICY_TIER_PASS=30, TIER_SUSPECT=60, TIER_IVT=80, TIER_BLOCK=100
- FRAUD_POLICY_ML_THRESHOLD=0.50
- FRAUD_POLICY_SOURCE=env | metadata | auto

Invariants:
- tier_pass <= tier_suspect <= tier_ivt <= tier_block (operator responsibility).
- merge_policy_config only applies env values that differ from default_policy_config().

Verify:
  pytest model/tests/test_policy_config_parity.py -q
  go test ./internal/fraud/ -short -run TestPolicyConfig -count=1
"""

from __future__ import annotations

import json
import os
from dataclasses import asdict, dataclass, fields
from pathlib import Path
from typing import Any, cast

FRAUD_POLICY_PRESET_DEFAULTS: dict[str, dict[str, int]] = {
    "conservative": {"pass": 40, "suspect": 70, "ivt": 90, "block": 100},
    "balanced": {"pass": 30, "suspect": 60, "ivt": 80, "block": 100},
    "aggressive": {"pass": 20, "suspect": 45, "ivt": 65, "block": 85},
    "enhanced_defense": {"pass": 20, "suspect": 45, "ivt": 65, "block": 85},
}

@dataclass(frozen=True)
class PolicyConfig:
    """Thresholds and heuristic floors for post-ML scoring_policy."""

    tier_pass: int = 30
    tier_suspect: int = 60
    tier_ivt: int = 80
    tier_block: int = 100

    ml_threshold: float = 0.50
    residential_proxy_floor: float = 0.62
    residential_proxy_max_ml: float = 0.45
    fp_guard_cap: float = 0.79

    proxy_min_events: float = 80.0
    proxy_max_ctr: float = 0.05
    proxy_min_users: float = 20.0
    proxy_min_user_click_gap: float = 5.0
    proxy_min_events_per_user: float = 5.0
    proxy_min_impression_pressure: float = 12.0
    proxy_min_users_per_ua: float = 2.5
    proxy_min_clicks: float = 2.0

    structural_high_ctr: float = 0.45
    structural_max_users: float = 5.0
    structural_min_events: float = 50.0
    structural_min_events_per_ua: float = 80.0
    structural_min_clicks_per_user: float = 15.0
    structural_spend_ratio: float = 0.9
    structural_spend_min_ctr: float = 0.4

    def block_probability(self) -> float:
        """Score threshold (as prob) above which FP-guard may apply.

        Prefer tier_block when set below 100; else tier_ivt; else 0.80 fallback.
        """
        if 0 < self.tier_block < 100:
            return self.tier_block / 100.0
        if self.tier_ivt > 0:
            return self.tier_ivt / 100.0
        return 0.80

    def to_dict(self) -> dict[str, float | int]:
        return asdict(self)

    @classmethod
    def from_dict(cls, raw: dict[str, object]) -> PolicyConfig:
        tier_fields = {"tier_pass", "tier_suspect", "tier_ivt", "tier_block"}
        int_kwargs: dict[str, int] = {}
        float_kwargs: dict[str, float] = {}
        for field in fields(cls):
            if field.name not in raw:
                continue
            value = raw[field.name]
            if field.name in tier_fields:
                int_kwargs[field.name] = int(cast(Any, value))
            else:
                float_kwargs[field.name] = float(cast(Any, value))
        return cls(**cast(Any, {**int_kwargs, **float_kwargs}))

def default_policy_config() -> PolicyConfig:
    return PolicyConfig()

def _env_float(key: str, fallback: float) -> float:
    raw = os.environ.get(key)
    if raw is None or raw == "":
        return fallback
    try:
        return float(raw)
    except ValueError:
        return fallback

def _env_int(key: str, fallback: int) -> int:
    raw = os.environ.get(key)
    if raw is None or raw == "":
        return fallback
    try:
        return int(raw)
    except ValueError:
        return fallback

def load_policy_config_from_env() -> PolicyConfig:
    """Parse FRAUD_POLICY_* into PolicyConfig."""
    base = default_policy_config()
    return PolicyConfig(
        tier_pass=_env_int("FRAUD_POLICY_TIER_PASS", base.tier_pass),
        tier_suspect=_env_int("FRAUD_POLICY_TIER_SUSPECT", base.tier_suspect),
        tier_ivt=_env_int("FRAUD_POLICY_TIER_IVT", base.tier_ivt),
        tier_block=_env_int("FRAUD_POLICY_TIER_BLOCK", base.tier_block),
        ml_threshold=_env_float("FRAUD_POLICY_ML_THRESHOLD", base.ml_threshold),
        residential_proxy_floor=_env_float("FRAUD_POLICY_PROXY_FLOOR", base.residential_proxy_floor),
        residential_proxy_max_ml=_env_float("FRAUD_POLICY_PROXY_MAX_ML", base.residential_proxy_max_ml),
        fp_guard_cap=_env_float("FRAUD_POLICY_FP_GUARD_CAP", base.fp_guard_cap),
        proxy_min_events=_env_float("FRAUD_POLICY_PROXY_MIN_EVENTS", base.proxy_min_events),
        proxy_max_ctr=_env_float("FRAUD_POLICY_PROXY_MAX_CTR", base.proxy_max_ctr),
        proxy_min_users=_env_float("FRAUD_POLICY_PROXY_MIN_USERS", base.proxy_min_users),
        proxy_min_user_click_gap=_env_float("FRAUD_POLICY_PROXY_MIN_USER_CLICK_GAP", base.proxy_min_user_click_gap),
        proxy_min_events_per_user=_env_float("FRAUD_POLICY_PROXY_MIN_EVENTS_PER_USER", base.proxy_min_events_per_user),
        proxy_min_impression_pressure=_env_float(
            "FRAUD_POLICY_PROXY_MIN_IMPRESSION_PRESSURE", base.proxy_min_impression_pressure
        ),
        proxy_min_users_per_ua=_env_float("FRAUD_POLICY_PROXY_MIN_USERS_PER_UA", base.proxy_min_users_per_ua),
        proxy_min_clicks=_env_float("FRAUD_POLICY_PROXY_MIN_CLICKS", base.proxy_min_clicks),
        structural_high_ctr=_env_float("FRAUD_POLICY_STRUCTURAL_HIGH_CTR", base.structural_high_ctr),
        structural_max_users=_env_float("FRAUD_POLICY_STRUCTURAL_MAX_USERS", base.structural_max_users),
        structural_min_events=_env_float("FRAUD_POLICY_STRUCTURAL_MIN_EVENTS", base.structural_min_events),
        structural_min_events_per_ua=_env_float(
            "FRAUD_POLICY_STRUCTURAL_MIN_EVENTS_PER_UA", base.structural_min_events_per_ua
        ),
        structural_min_clicks_per_user=_env_float(
            "FRAUD_POLICY_STRUCTURAL_MIN_CLICKS_PER_USER", base.structural_min_clicks_per_user
        ),
        structural_spend_ratio=_env_float("FRAUD_POLICY_STRUCTURAL_SPEND_RATIO", base.structural_spend_ratio),
        structural_spend_min_ctr=_env_float("FRAUD_POLICY_STRUCTURAL_SPEND_MIN_CTR", base.structural_spend_min_ctr),
    )

def load_policy_from_metadata(path: Path) -> PolicyConfig | None:
    """Read policy section from metadata.json; None if missing."""
    if not path.is_file():
        return None
    with open(path, encoding="utf-8") as handle:
        payload = json.load(handle)
    raw = payload.get("policy")
    if not isinstance(raw, dict):
        return None
    return PolicyConfig.from_dict(raw)

def resolve_policy_config(
    env_policy: PolicyConfig,
    metadata_path: Path,
    source: str,
) -> PolicyConfig:
    """Resolve active policy from env, metadata, or merged auto mode.

    Args:
        source: ``env`` | ``metadata`` | ``auto`` (metadata base + non-default env).
    """
    if source == "env":
        return env_policy
    metadata_policy = load_policy_from_metadata(metadata_path)
    if source == "metadata":
        return metadata_policy if metadata_policy is not None else env_policy
    if metadata_policy is not None:
        return merge_policy_config(metadata_policy, env_policy)
    return env_policy

def merge_policy_config(base: PolicyConfig, override: PolicyConfig) -> PolicyConfig:
    """Overlay env overrides onto metadata base; skip fields still at Go defaults."""

    def pick_int(name: str, current: int) -> int:
        new_val = getattr(override, name)
        default_val = getattr(default_policy_config(), name)
        return new_val if new_val != default_val else current

    def pick_float(name: str, current: float) -> float:
        new_val = getattr(override, name)
        default_val = getattr(default_policy_config(), name)
        return new_val if new_val != default_val else current

    return PolicyConfig(
        tier_pass=pick_int("tier_pass", base.tier_pass),
        tier_suspect=pick_int("tier_suspect", base.tier_suspect),
        tier_ivt=pick_int("tier_ivt", base.tier_ivt),
        tier_block=pick_int("tier_block", base.tier_block),
        ml_threshold=pick_float("ml_threshold", base.ml_threshold),
        residential_proxy_floor=pick_float("residential_proxy_floor", base.residential_proxy_floor),
        residential_proxy_max_ml=pick_float("residential_proxy_max_ml", base.residential_proxy_max_ml),
        fp_guard_cap=pick_float("fp_guard_cap", base.fp_guard_cap),
        proxy_min_events=pick_float("proxy_min_events", base.proxy_min_events),
        proxy_max_ctr=pick_float("proxy_max_ctr", base.proxy_max_ctr),
        proxy_min_users=pick_float("proxy_min_users", base.proxy_min_users),
        proxy_min_user_click_gap=pick_float("proxy_min_user_click_gap", base.proxy_min_user_click_gap),
        proxy_min_events_per_user=pick_float("proxy_min_events_per_user", base.proxy_min_events_per_user),
        proxy_min_impression_pressure=pick_float("proxy_min_impression_pressure", base.proxy_min_impression_pressure),
        proxy_min_users_per_ua=pick_float("proxy_min_users_per_ua", base.proxy_min_users_per_ua),
        proxy_min_clicks=pick_float("proxy_min_clicks", base.proxy_min_clicks),
        structural_high_ctr=pick_float("structural_high_ctr", base.structural_high_ctr),
        structural_max_users=pick_float("structural_max_users", base.structural_max_users),
        structural_min_events=pick_float("structural_min_events", base.structural_min_events),
        structural_min_events_per_ua=pick_float("structural_min_events_per_ua", base.structural_min_events_per_ua),
        structural_min_clicks_per_user=pick_float(
            "structural_min_clicks_per_user", base.structural_min_clicks_per_user
        ),
        structural_spend_ratio=pick_float("structural_spend_ratio", base.structural_spend_ratio),
        structural_spend_min_ctr=pick_float("structural_spend_min_ctr", base.structural_spend_min_ctr),
    )

def format_policy_env(cfg: PolicyConfig) -> str:
    """Printable .env block after bootstrap calibration."""
    lines = [
        "# Fraud scoring policy (calibrated)",
        "FRAUD_POLICY_SOURCE=metadata",
        f"FRAUD_POLICY_TIER_PASS={cfg.tier_pass}",
        f"FRAUD_POLICY_TIER_SUSPECT={cfg.tier_suspect}",
        f"FRAUD_POLICY_TIER_IVT={cfg.tier_ivt}",
        f"FRAUD_POLICY_TIER_BLOCK={cfg.tier_block}",
        f"FRAUD_POLICY_ML_THRESHOLD={cfg.ml_threshold:.4f}",
        f"FRAUD_POLICY_PROXY_FLOOR={cfg.residential_proxy_floor:.4f}",
        f"FRAUD_POLICY_PROXY_MAX_ML={cfg.residential_proxy_max_ml:.4f}",
        f"FRAUD_POLICY_FP_GUARD_CAP={cfg.fp_guard_cap:.4f}",
        f"FRAUD_POLICY_PROXY_MIN_EVENTS={cfg.proxy_min_events:.1f}",
        f"FRAUD_POLICY_PROXY_MAX_CTR={cfg.proxy_max_ctr:.4f}",
        f"FRAUD_POLICY_PROXY_MIN_USERS={cfg.proxy_min_users:.1f}",
        f"FRAUD_POLICY_PROXY_MIN_USER_CLICK_GAP={cfg.proxy_min_user_click_gap:.2f}",
        f"FRAUD_POLICY_PROXY_MIN_EVENTS_PER_USER={cfg.proxy_min_events_per_user:.2f}",
        f"FRAUD_POLICY_PROXY_MIN_IMPRESSION_PRESSURE={cfg.proxy_min_impression_pressure:.2f}",
        f"FRAUD_POLICY_PROXY_MIN_USERS_PER_UA={cfg.proxy_min_users_per_ua:.2f}",
        f"FRAUD_POLICY_PROXY_MIN_CLICKS={cfg.proxy_min_clicks:.1f}",
    ]
    return "\n".join(lines) + "\n"

# Process-wide cache; set_policy_config() used after bootstrap calibration.
_active_policy_config: PolicyConfig | None = None


def get_policy_config() -> PolicyConfig:
    global _active_policy_config
    if _active_policy_config is None:
        _active_policy_config = load_policy_config_from_env()
    return _active_policy_config

def set_policy_config(cfg: PolicyConfig) -> None:
    global _active_policy_config
    _active_policy_config = cfg
