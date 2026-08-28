#!/usr/bin/env python3
"""Offline ML vs policy benchmark on synthetic traffic."""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

import numpy as np

from contract.feature_spec import FEATURE_DIMS, FEATURE_NAMES, row_to_vector
from contract.policy_config import (
    get_policy_config,
    load_policy_config_from_env,
    resolve_policy_config,
    set_policy_config,
)
from contract.scoring_policy import action_fraud_positive, decide
from data.clickhouse_client import clickhouse_config_from_env, connect_client, ping_client
from eval.shadow_precision import run_shadow_precision
from repo_paths import REPO_ROOT
from train.traffic_simulator import ARCHETYPES, generate_network_batch

DEFAULT_MODEL = REPO_ROOT / "var" / "fraudscore" / "artifacts" / "model.txt"
ARTIFACT_DIR = REPO_ROOT / os.environ.get("FRAUD_ARTIFACT_DIR", "var/fraudscore/artifacts")
FRAUD_THRESHOLD = float(os.environ.get("FRAUD_POLICY_ML_THRESHOLD", "0.5"))

def _load_booster(model_path: Path):
    try:
        import lightgbm as lgb
    except ImportError as err:
        raise SystemExit("evaluate: lightgbm required; pip install -r model/requirements.txt") from err
    booster = lgb.Booster(model_file=str(model_path))
    if booster.num_feature() != FEATURE_DIMS:
        raise SystemExit(f"evaluate: model num_feature={booster.num_feature()} want {FEATURE_DIMS}")
    return booster

def _train_booster(train_rows: list[dict[str, int]], train_labels: list[int]):
    import lightgbm as lgb

    matrix = np.array([row_to_vector(row) for row in train_rows], dtype=np.float64)
    labels = np.array(train_labels, dtype=np.int32)
    if labels.sum() == 0 or labels.sum() == len(labels):
        raise SystemExit("evaluate: train set has a single class; increase --train-size")

    negative_count = int(np.sum(labels == 0))
    positive_count = int(np.sum(labels == 1))
    scale_pos_weight = float(negative_count / positive_count) if positive_count > 0 else 1.0

    train_data = lgb.Dataset(matrix, label=labels, feature_name=list(FEATURE_NAMES))
    params = {
        "objective": "binary",
        "metric": "binary_logloss",
        "boosting_type": "gbdt",
        "learning_rate": 0.05,
        "num_leaves": 31,
        "min_data_in_leaf": 15,
        "feature_fraction": 0.9,
        "scale_pos_weight": scale_pos_weight,
        "verbose": -1,
    }
    return lgb.train(params, train_data, num_boost_round=80)

def _confusion(y_true: np.ndarray, y_pred: np.ndarray) -> dict[str, int]:
    tp = int(np.sum((y_true == 1) & (y_pred == 1)))
    tn = int(np.sum((y_true == 0) & (y_pred == 0)))
    fp = int(np.sum((y_true == 0) & (y_pred == 1)))
    fn = int(np.sum((y_true == 1) & (y_pred == 0)))
    return {"tp": tp, "tn": tn, "fp": fp, "fn": fn}

def _rates(confusion_counts: dict[str, int]) -> dict[str, float]:
    tp, tn, fp, fn = (
        confusion_counts["tp"],
        confusion_counts["tn"],
        confusion_counts["fp"],
        confusion_counts["fn"],
    )
    precision = tp / (tp + fp) if (tp + fp) else 0.0
    recall = tp / (tp + fn) if (tp + fn) else 0.0
    f1 = 2 * precision * recall / (precision + recall) if (precision + recall) else 0.0
    accuracy = (tp + tn) / (tp + tn + fp + fn) if (tp + tn + fp + fn) else 0.0
    fpr = fp / (fp + tn) if (fp + tn) else 0.0
    return {
        "accuracy": accuracy,
        "precision": precision,
        "recall": recall,
        "f1": f1,
        "false_positive_rate": fpr,
    }

def _score_rows(booster, rows: list[dict[str, int]]) -> np.ndarray:
    matrix = np.array([row_to_vector(row) for row in rows], dtype=np.float64)
    return booster.predict(matrix)

def _policy_predictions(
    rows: list[dict[str, int]],
    probs: np.ndarray,
    *,
    block_only: bool,
) -> np.ndarray:
    out = np.zeros(len(rows), dtype=np.int32)
    for idx, (row, prob) in enumerate(zip(rows, probs, strict=True)):
        decision = decide(row, float(prob))
        if action_fraud_positive(decision, block_only=block_only):
            out[idx] = 1
    return out

def _policy_stats(rows: list[dict[str, int]], probs: np.ndarray) -> dict[str, int]:
    proxy_hits = 0
    structural_hits = 0
    fp_guard_hits = 0
    for row, prob in zip(rows, probs, strict=True):
        decision = decide(row, float(prob))
        if decision.residential_proxy:
            proxy_hits += 1
        if decision.structural_fraud:
            structural_hits += 1
        if decision.fp_guard_applied:
            fp_guard_hits += 1
    return {
        "residential_proxy_signals": proxy_hits,
        "structural_fraud_signals": structural_hits,
        "fp_guard_downgrades": fp_guard_hits,
    }

def _evaluate_mode(
    y_true: np.ndarray,
    y_pred: np.ndarray,
    probs: np.ndarray,
) -> dict[str, object]:
    confusion_counts = _confusion(y_true, y_pred)
    rates = _rates(confusion_counts)
    from sklearn.metrics import average_precision_score, roc_auc_score

    try:
        auc = float(roc_auc_score(y_true, probs))
        pr_auc = float(average_precision_score(y_true, probs))
    except ValueError:
        auc = 0.0
        pr_auc = 0.0
    return {"confusion_matrix": confusion_counts, "metrics": {**rates, "roc_auc": auc, "pr_auc": pr_auc}}

def _archetype_report_policy(
    rows: list[dict[str, int]],
    archetypes: list[str],
    labels: list[int],
    probs: np.ndarray,
    *,
    block_only: bool,
) -> list[dict[str, object]]:
    preds = _policy_predictions(rows, probs, block_only=block_only)
    by_name: dict[str, dict[str, list]] = {}
    for name, label, pred in zip(archetypes, labels, preds, strict=True):
        bucket = by_name.setdefault(name, {"labels": [], "preds": []})
        bucket["labels"].append(label)
        bucket["preds"].append(int(pred))

    report: list[dict[str, object]] = []
    for archetype in ARCHETYPES:
        bucket = by_name.get(archetype.name)
        if not bucket:
            continue
        labels_batch = np.array(bucket["labels"], dtype=np.int32)
        predictions_batch = np.array(bucket["preds"], dtype=np.int32)
        confusion_counts = _confusion(labels_batch, predictions_batch)
        rates = _rates(confusion_counts)
        report.append(
            {
                "archetype": archetype.name,
                "is_fraud_cohort": archetype.is_fraud,
                "n": len(labels_batch),
                **rates,
                **confusion_counts,
            }
        )
    return report

def _archetype_report(
    archetypes: list[str],
    labels: list[int],
    probs: np.ndarray,
    threshold: float,
) -> list[dict[str, object]]:
    by_name: dict[str, dict[str, list]] = {}
    for name, label, prob in zip(archetypes, labels, probs, strict=True):
        bucket = by_name.setdefault(name, {"labels": [], "probs": []})
        bucket["labels"].append(label)
        bucket["probs"].append(float(prob))

    report: list[dict[str, object]] = []
    for archetype in ARCHETYPES:
        bucket = by_name.get(archetype.name)
        if not bucket:
            continue
        labels_batch = np.array(bucket["labels"], dtype=np.int32)
        probability_batch = np.array(bucket["probs"], dtype=np.float64)
        predictions_batch = (probability_batch >= threshold).astype(np.int32)
        confusion_counts = _confusion(labels_batch, predictions_batch)
        rates = _rates(confusion_counts)
        report.append(
            {
                "archetype": archetype.name,
                "is_fraud_cohort": archetype.is_fraud,
                "n": len(labels_batch),
                "fraud_rate_labeled": float(labels_batch.mean()),
                "score_mean": float(probability_batch.mean()),
                "score_p50": float(np.median(probability_batch)),
                "score_p95": float(np.quantile(probability_batch, 0.95)),
                **rates,
                **confusion_counts,
            }
        )
    return report

def _sample_misclassified(
    rows: list[dict[str, int]],
    labels: list[int],
    archetypes: list[str],
    probs: np.ndarray,
    threshold: float,
    limit: int = 5,
) -> dict[str, list[dict[str, object]]]:
    false_positives: list[dict[str, object]] = []
    false_negatives: list[dict[str, object]] = []

    for row, label, archetype, prob in zip(rows, labels, archetypes, probs, strict=True):
        pred = int(prob >= threshold)
        if label == 0 and pred == 1 and len(false_positives) < limit:
            false_positives.append(
                {
                    "archetype": archetype,
                    "score": round(float(prob), 4),
                    "row": row,
                    "vector": [round(v, 4) for v in row_to_vector(row)],
                }
            )
        if label == 1 and pred == 0 and len(false_negatives) < limit:
            false_negatives.append(
                {
                    "archetype": archetype,
                    "score": round(float(prob), 4),
                    "row": row,
                    "vector": [round(v, 4) for v in row_to_vector(row)],
                }
            )

    return {"false_positives": false_positives, "false_negatives": false_negatives}

def run_simulation(
    model_path: Path,
    train_size: int,
    test_size: int,
    train_seed: int,
    test_seed: int,
    threshold: float,
    retrain: bool,
    metrics_out: Path | None,
) -> dict[str, object]:
    """End-to-end benchmark on a synthetic test batch."""
    train_rows, train_labels, _ = generate_network_batch(train_size, seed=train_seed)
    test_rows, test_labels, test_archetypes = generate_network_batch(test_size, seed=test_seed)

    if retrain:
        booster = _train_booster(train_rows, train_labels)
        model_source = "fresh_train_on_network_batch"
    else:
        booster = _load_booster(model_path)
        model_source = str(model_path)

    y_true = np.array(test_labels, dtype=np.int32)
    probs = _score_rows(booster, test_rows)

    ml_preds = (probs >= threshold).astype(np.int32)
    policy_suspect_preds = _policy_predictions(test_rows, probs, block_only=False)
    policy_block_preds = _policy_predictions(test_rows, probs, block_only=True)

    ml_eval = _evaluate_mode(y_true, ml_preds, probs)
    policy_suspect_eval = _evaluate_mode(y_true, policy_suspect_preds, probs)
    policy_block_eval = _evaluate_mode(y_true, policy_block_preds, probs)

    result: dict[str, object] = {
        "model_source": model_source,
        "train_size": train_size,
        "test_size": test_size,
        "train_seed": train_seed,
        "test_seed": test_seed,
        "threshold": threshold,
        "policy_config": get_policy_config().to_dict(),
        "test_fraud_prevalence": float(y_true.mean()),
        "ml_only": ml_eval,
        "policy_suspect_plus": policy_suspect_eval,
        "policy_block_only": policy_block_eval,
        "policy_signals": _policy_stats(test_rows, probs),
        "score_distribution": {
            "legit_mean": float(probs[y_true == 0].mean()) if np.any(y_true == 0) else 0.0,
            "legit_p95": float(np.quantile(probs[y_true == 0], 0.95)) if np.any(y_true == 0) else 0.0,
            "fraud_mean": float(probs[y_true == 1].mean()) if np.any(y_true == 1) else 0.0,
            "fraud_p05": float(np.quantile(probs[y_true == 1], 0.05)) if np.any(y_true == 1) else 0.0,
        },
        "archetype_breakdown_ml": _archetype_report(test_archetypes, test_labels, probs, threshold),
        "archetype_breakdown_policy_suspect": _archetype_report_policy(
            test_rows, test_archetypes, test_labels, probs, block_only=False
        ),
        "archetype_breakdown_policy_block": _archetype_report_policy(
            test_rows, test_archetypes, test_labels, probs, block_only=True
        ),
        "misclassified_samples": _sample_misclassified(test_rows, test_labels, test_archetypes, probs, threshold),
        "archetype_mix": [{"name": a.name, "is_fraud": a.is_fraud, "weight": a.weight} for a in ARCHETYPES],
    }

    if metrics_out is not None:
        metrics_out.parent.mkdir(parents=True, exist_ok=True)
        with open(metrics_out, "w", encoding="utf-8") as handle:
            json.dump(result, handle, indent=2)
            handle.write("\n")

    return result

def run_shadow_precision_report(
    *,
    hours: int = 24,
    threshold: float = 0.6,
    metrics_out: Path | None = None,
    allow_offline: bool = False,
) -> dict:
    try:
        client = connect_client()
        if not ping_client(client):
            raise ConnectionError("clickhouse ping failed")
        report = run_shadow_precision(client, hours=hours, threshold=threshold)
    except (ConnectionError, OSError, TimeoutError, ValueError) as err:
        if allow_offline:
            clickhouse_config = clickhouse_config_from_env()
            report = {
                "status": "skipped",
                "reason": str(err),
                "host": clickhouse_config.host,
                "port": clickhouse_config.port,
                "database": clickhouse_config.database,
            }
        else:
            raise

    if metrics_out is not None:
        metrics_out.parent.mkdir(parents=True, exist_ok=True)
        with metrics_out.open("w", encoding="utf-8") as handle:
            json.dump(report, handle, indent=2)
            handle.write("\n")
    return report

def main() -> int:
    parser = argparse.ArgumentParser(description="Evaluate fraud model on realistic synthetic ad network traffic")
    parser.add_argument("--model", default=str(DEFAULT_MODEL.relative_to(REPO_ROOT)))
    parser.add_argument("--simulate", action="store_true", help="run chaotic network simulation")
    parser.add_argument("--train-size", type=int, default=12_000)
    parser.add_argument("--test-size", type=int, default=6_000)
    parser.add_argument("--train-seed", type=int, default=2026)
    parser.add_argument("--test-seed", type=int, default=4242)
    parser.add_argument("--threshold", type=float, default=FRAUD_THRESHOLD)
    parser.add_argument(
        "--retrain",
        action="store_true",
        help="train a fresh model on simulated train batch (recommended)",
    )
    parser.add_argument("--metrics-out", default="", help="write JSON report to this path")
    parser.add_argument(
        "--shadow-precision",
        action="store_true",
        help="run CH shadow precision report (proxy labels)",
    )
    parser.add_argument("--shadow-hours", type=int, default=24)
    parser.add_argument(
        "--allow-offline",
        action="store_true",
        help="exit 0 when ClickHouse is unreachable or empty",
    )
    args = parser.parse_args()

    if args.shadow_precision:
        metrics_out = (
            Path(args.metrics_out) if args.metrics_out else REPO_ROOT / "var/fraudscore/shadow_precision_report.json"
        )
        threshold = float(os.environ.get("FRAUD_POLICY_ML_THRESHOLD", str(args.threshold)))
        report = run_shadow_precision_report(
            hours=args.shadow_hours,
            threshold=threshold,
            metrics_out=metrics_out,
            allow_offline=args.allow_offline,
        )
        print(json.dumps(report, indent=2))
        if report.get("status") == "skipped":
            return 0
        if report.get("status") == "empty" and args.allow_offline:
            return 0
        if report.get("status") == "empty":
            print("evaluate: no labeled shadow rows in ClickHouse", file=sys.stderr)
            return 1
        return 0

    if not args.simulate:
        print("evaluate: pass --simulate to run network fraud benchmark", file=sys.stderr)
        return 1

    metadata_path = ARTIFACT_DIR / "metadata.json"
    env_policy = load_policy_config_from_env()
    source = os.environ.get("FRAUD_POLICY_SOURCE", "auto")
    set_policy_config(resolve_policy_config(env_policy, metadata_path, source))
    threshold = float(os.environ.get("FRAUD_POLICY_ML_THRESHOLD", str(args.threshold)))

    model_path = Path(args.model)
    if not model_path.is_file():
        model_path = REPO_ROOT / args.model

    metrics_out = Path(args.metrics_out) if args.metrics_out else REPO_ROOT / "var/fraudscore/sim_report.json"
    result = run_simulation(
        model_path=model_path,
        train_size=args.train_size,
        test_size=args.test_size,
        train_seed=args.train_seed,
        test_seed=args.test_seed,
        threshold=threshold,
        retrain=args.retrain,
        metrics_out=metrics_out,
    )

    print(json.dumps(result, indent=2))
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
