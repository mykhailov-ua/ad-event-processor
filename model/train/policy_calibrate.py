"""Grid search for policy thresholds; persists best config to metadata.json."""

from __future__ import annotations

import os
from typing import Any

import numpy as np

from contract.policy_config import PolicyConfig, default_policy_config
from contract.scoring_policy import (
    policy_scores_vector,
    precompute_row_signals,
    scores_block_positive,
    scores_suspect_positive,
)

def _confusion(y_true: np.ndarray, y_pred: np.ndarray) -> tuple[int, int, int, int]:
    tp = int(np.sum((y_true == 1) & (y_pred == 1)))
    tn = int(np.sum((y_true == 0) & (y_pred == 0)))
    fp = int(np.sum((y_true == 0) & (y_pred == 1)))
    fn = int(np.sum((y_true == 1) & (y_pred == 0)))
    return tp, tn, fp, fn

def _rates(tp: int, tn: int, fp: int, fn: int) -> dict[str, float]:
    precision = tp / (tp + fp) if (tp + fp) else 0.0
    recall = tp / (tp + fn) if (tp + fn) else 0.0
    f1 = 2 * precision * recall / (precision + recall) if (precision + recall) else 0.0
    fpr = fp / (fp + tn) if (fp + tn) else 0.0
    return {"precision": precision, "recall": recall, "f1": f1, "false_positive_rate": fpr}

def _rates_from_preds(y_true: np.ndarray, y_pred: np.ndarray) -> dict[str, float]:
    tp, tn, fp, fn = _confusion(y_true, y_pred)
    return _rates(tp, tn, fp, fn)

def _calibration_grid() -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    coarse = os.environ.get("FRAUD_POLICY_CALIB_GRID", "full").lower() == "coarse"
    if coarse:
        return (
            np.array([0.35, 0.50, 0.65], dtype=np.float64),
            np.array([0.55, 0.62], dtype=np.float64),
            np.array([0.35, 0.45], dtype=np.float64),
            np.array([0.75, 0.79], dtype=np.float64),
        )
    return (
        np.arange(0.35, 0.76, 0.05),
        np.arange(0.55, 0.71, 0.02),
        np.arange(0.35, 0.56, 0.05),
        np.arange(0.75, 0.83, 0.02),
    )

def calibrate_policy(
    rows: list[dict[str, int]],
    labels: list[int],
    archetypes: list[str],
    probs: np.ndarray,
    *,
    max_fpr: float = 0.01,
    max_block_fpr: float = 0.005,
) -> tuple[PolicyConfig, dict[str, Any]]:
    """Select thresholds under FPR caps; maximize recall with proxy cohort weight."""
    base = default_policy_config()
    y_true = np.asarray(labels, dtype=np.int32)
    probability_vector = np.asarray(probs, dtype=np.float64)

    proxy_signals, structural_signals = precompute_row_signals(rows, base)
    proxy_mask = np.fromiter((a == "residential_proxy_bot" for a in archetypes), dtype=bool, count=len(archetypes))
    grey_mask = np.fromiter((a == "grey_noise" for a in archetypes), dtype=bool, count=len(archetypes))
    proxy_count = int(np.sum(proxy_mask))
    grey_noise_count = int(np.sum(grey_mask))

    block_prob = base.block_probability()
    ml_thresholds, proxy_floors, proxy_max_ml, fp_caps = _calibration_grid()

    best_policy_config = base
    best_score = -1.0
    best_metrics: dict[str, float] = {}
    best_proxy_recall = 0.0
    best_grey_fpr = 1.0

    for ml_threshold_candidate in ml_thresholds:
        for floor in proxy_floors:
            for max_ml in proxy_max_ml:
                for fp_guard_cap_candidate in fp_caps:
                    scores = policy_scores_vector(
                        probability_vector,
                        proxy_signals,
                        structural_signals,
                        float(floor),
                        float(max_ml),
                        float(fp_guard_cap_candidate),
                        block_prob,
                    )
                    suspect_pred = scores_suspect_positive(scores, base.tier_pass).astype(np.int32)
                    block_pred = scores_block_positive(scores, base.tier_ivt).astype(np.int32)

                    suspect_metrics = _rates_from_preds(y_true, suspect_pred)
                    if suspect_metrics["false_positive_rate"] > max_fpr:
                        continue

                    block_metrics = _rates_from_preds(y_true, block_pred)
                    if block_metrics["false_positive_rate"] > max_block_fpr:
                        continue

                    if proxy_count > 0:
                        proxy_recall = float(np.sum(suspect_pred[proxy_mask]) / proxy_count)
                    else:
                        proxy_recall = 0.0

                    if grey_noise_count > 0:
                        grey_fpr = float(np.sum(suspect_pred[grey_mask]) / grey_noise_count)
                    else:
                        grey_fpr = 0.0

                    objective = suspect_metrics["recall"] * 0.6 + proxy_recall * 0.3 - grey_fpr * 0.1
                    if objective > best_score:
                        best_score = objective
                        best_policy_config = PolicyConfig(
                            ml_threshold=float(ml_threshold_candidate),
                            residential_proxy_floor=float(floor),
                            residential_proxy_max_ml=float(max_ml),
                            fp_guard_cap=float(fp_guard_cap_candidate),
                        )
                        best_metrics = {
                            **{f"suspect_{key}": value for key, value in suspect_metrics.items()},
                            **{f"block_{key}": value for key, value in block_metrics.items()},
                        }
                        best_proxy_recall = proxy_recall
                        best_grey_fpr = grey_fpr

    report = {
        "objective_score": best_score,
        "max_fpr": max_fpr,
        "max_block_fpr": max_block_fpr,
        "proxy_cohort_recall": best_proxy_recall,
        "grey_noise_fpr": best_grey_fpr,
        "metrics": best_metrics,
        "policy": best_policy_config.to_dict(),
    }
    return best_policy_config, report
