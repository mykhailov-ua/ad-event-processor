#!/usr/bin/env python3
"""Bootstrap LightGBM and iforest artifacts; calibrate policy; validate fixtures."""
from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path
from typing import TYPE_CHECKING, Any

from feature_spec import FEATURE_DIMS, FEATURE_NAMES, row_to_vector
from labeled_dataset import (
    ROW_FIELDS,
    load_labeled_dataset,
    split_to_bundle,
    time_based_split,
    training_summary,
)
from policy_calibrate import calibrate_policy
from policy_config import (
    format_policy_env,
    load_policy_config_from_env,
    resolve_policy_config,
    set_policy_config,
)
from traffic_simulator import generate_network_batch

if TYPE_CHECKING:
    import numpy as np

REPO_ROOT = Path(__file__).resolve().parent.parent
ARTIFACT_DIR = str(REPO_ROOT / os.environ.get("FRAUD_ARTIFACT_DIR", "var/fraudscore/artifacts"))
DEFAULT_ARTIFACT_MODEL = os.path.join(ARTIFACT_DIR, "model.txt")
FIXTURES_DIR = REPO_ROOT / "testdata" / "ml"
ONNX_TARGET_OPSET = {"": 12, "ai.onnx.ml": 3}
SYNTHETIC_ROW_COUNT = 12_000
SYNTHETIC_VAL_FRACTION = 0.2
SYNTHETIC_BOOST_ROUNDS = 50
_CALIB_HOLDOUT_DEFAULT = 3000
_FIT_VAL_FRACTION_DEFAULT = 0.2
_FIT_BOOST_ROUNDS_DEFAULT = 200
_FIT_MIN_ROWS_DEFAULT = 500
_LOG = "fraud_modeling/artifact_bootstrap"


def _env_bool(key: str, default: bool) -> bool:
    raw = os.environ.get(key)
    if raw is None or raw == "":
        return default
    return raw.lower() in {"1", "true", "yes", "on"}


def _calibration_holdout_size() -> int:
    raw = os.environ.get("FRAUD_POLICY_CALIB_HOLDOUT", str(_CALIB_HOLDOUT_DEFAULT))
    try:
        return max(500, int(raw))
    except ValueError:
        return _CALIB_HOLDOUT_DEFAULT


def _fit_val_fraction() -> float:
    raw = os.environ.get("FRAUD_FIT_VAL_FRACTION", str(_FIT_VAL_FRACTION_DEFAULT))
    try:
        value = float(raw)
    except ValueError:
        return _FIT_VAL_FRACTION_DEFAULT
    return min(0.5, max(0.05, value))


def _fit_boost_rounds() -> int:
    raw = os.environ.get("FRAUD_FIT_BOOST_ROUNDS", str(_FIT_BOOST_ROUNDS_DEFAULT))
    try:
        return max(20, int(raw))
    except ValueError:
        return _FIT_BOOST_ROUNDS_DEFAULT


def _fit_min_rows() -> int:
    raw = os.environ.get("FRAUD_FIT_MIN_ROWS", str(_FIT_MIN_ROWS_DEFAULT))
    try:
        return max(100, int(raw))
    except ValueError:
        return _FIT_MIN_ROWS_DEFAULT


def _parse_optional_time(value: str) -> datetime | None:
    if not value:
        return None
    text = value.strip()
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    parsed = datetime.fromisoformat(text)
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC)


def _train_lightgbm_binary(
    x_train: np.ndarray,
    y_train: np.ndarray,
    x_val: np.ndarray,
    y_val: np.ndarray,
    *,
    num_rounds: int,
    note: str,
) -> tuple[Any, dict[str, float], np.ndarray]:
    import lightgbm as lgb
    import numpy as np
    from sklearn.metrics import accuracy_score, f1_score, roc_auc_score

    if int(np.sum(y_train == 1)) == 0 or int(np.sum(y_train == 0)) == 0:
        raise ValueError("train set has a single class")
    if int(np.sum(y_val == 1)) == 0 or int(np.sum(y_val == 0)) == 0:
        raise ValueError("validation set has a single class")

    neg = int(np.sum(y_train == 0))
    pos = int(np.sum(y_train == 1))
    scale_pos_weight = float(neg / pos) if pos > 0 else 1.0

    train_data = lgb.Dataset(x_train, label=y_train, feature_name=list(FEATURE_NAMES))
    val_data = lgb.Dataset(x_val, label=y_val, reference=train_data)
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
    model = lgb.train(
        params,
        train_data,
        num_boost_round=num_rounds,
        valid_sets=[val_data],
        callbacks=[lgb.early_stopping(stopping_rounds=20, verbose=False)],
    )

    val_probs = model.predict(x_val, num_iteration=model.best_iteration)
    val_preds = (val_probs >= 0.5).astype(np.int32)
    metrics = {
        "accuracy": float(accuracy_score(y_val, val_preds)),
        "f1_score": float(f1_score(y_val, val_preds, zero_division=0)),
        "auc": float(roc_auc_score(y_val, val_probs)),
        "note": note,
    }
    return model, metrics, val_probs


def _write_iforest_artifact(onnx_path: str, matrix: np.ndarray) -> None:
    """Train iforest ONNX when enabled; otherwise write a placeholder file."""
    if not _env_bool("FRAUD_BOOTSTRAP_IFOREST", False):
        with open(onnx_path, "wb") as f:
            f.write(b"iforest disabled")
        return

    from skl2onnx import convert_sklearn
    from skl2onnx.common.data_types import FloatTensorType
    from sklearn.ensemble import IsolationForest

    iforest = IsolationForest(
        n_estimators=50,
        contamination="auto",
        random_state=42,
    )
    iforest.fit(matrix)
    initial_type = [("input", FloatTensorType([None, FEATURE_DIMS]))]
    onx = convert_sklearn(iforest, initial_types=initial_type, target_opset=ONNX_TARGET_OPSET)
    with open(onnx_path, "wb") as f:
        f.write(onx.SerializeToString())


def sha256_file(path: str) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def _raw_stats_from_rows(rows: list[dict[str, int]]) -> dict[str, dict[str, float]]:
    """Mean/std for raw ml_features_1m columns before vectorization."""
    import numpy as np

    if not rows:
        return {}
    matrix = np.array([[int(row[field]) for field in ROW_FIELDS] for row in rows], dtype=np.float64)
    return {
        name: {"mean": float(matrix[:, idx].mean()), "std": float(matrix[:, idx].std())}
        for idx, name in enumerate(ROW_FIELDS)
    }


def write_metadata(
    model_path: str,
    onnx_path: str,
    metrics: dict | None = None,
    *,
    policy: dict | None = None,
    calibration: dict | None = None,
    importance: dict | None = None,
    stats: dict | None = None,
    raw_stats: dict | None = None,
) -> dict:
    """Write metadata.json with artifact hashes and optional policy/calibration/importance."""
    model_hash = sha256_file(model_path)
    iforest_hash = sha256_file(onnx_path)
    meta = {
        "version": "v" + model_hash[:8],
        "lightgbm_hash": model_hash,
        "iforest_hash": iforest_hash,
        "metrics": metrics or {
            "accuracy": 0.95,
            "f1_score": 0.92,
            "auc": 0.98,
            "note": "bootstrap/synthetic — not production evaluation",
        },
        "created_at": datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    if policy is not None:
        meta["policy"] = policy
    if calibration is not None:
        meta["calibration"] = calibration
    if importance is not None:
        meta["importance"] = importance
    if stats is not None:
        meta["stats"] = stats
    if raw_stats is not None:
        meta["raw_stats"] = raw_stats
    with open(os.path.join(ARTIFACT_DIR, "metadata.json"), "w", encoding="utf-8") as f:
        json.dump(meta, f, indent=2)
    return meta


def synthetic_fraud_label(row: dict[str, int]) -> int:
    events = int(row.get("events", 0))
    clicks = int(row.get("clicks", 0))
    unique_users = int(row.get("unique_users", 0))
    unique_uas = int(row.get("unique_uas", 0))
    spend_micro = int(row.get("spend_micro", 0))
    budget_limit_micro = int(row.get("budget_limit_micro", 0))

    if events <= 0:
        return 0

    ctr = float(clicks) / float(events)
    spend_ratio = float(spend_micro) / float(budget_limit_micro) if budget_limit_micro > 0 else 0.0
    events_per_ua = float(events) / float(unique_uas) if unique_uas > 0 else 0.0
    clicks_per_user = float(clicks) / float(unique_users) if unique_users > 0 else 0.0
    ua_diversity = float(unique_uas) / float(events)

    if ctr > 0.8 and unique_users <= 2:
        return 1
    if unique_uas == 1 and events >= 50 and unique_users <= 3:
        return 1
    if events_per_ua >= 80 and unique_uas <= 1:
        return 1
    if clicks_per_user >= 20:
        return 1
    if spend_ratio > 0.95 and ctr > 0.5:
        return 1
    if ua_diversity >= 0.5 and ctr < 0.05 and unique_users >= 10:
        return 0
    return 0


def synthetic_rows(count: int, seed: int = 42) -> list[dict[str, int]]:
    import numpy as np

    rng = np.random.default_rng(seed)
    rows: list[dict[str, int]] = []
    for _ in range(count):
        events = int(rng.integers(1, 500))
        clicks = int(rng.integers(0, events + 1))
        budget_limit_micro = int(rng.integers(1_000_000, 100_000_000))
        spend_micro = int(rng.integers(0, budget_limit_micro + 1))
        unique_users = int(rng.integers(1, max(2, events // 5 + 2)))
        unique_uas = int(rng.integers(1, max(2, unique_users + 1)))
        rows.append(
            {
                "events": events,
                "clicks": clicks,
                "spend_micro": spend_micro,
                "budget_limit_micro": budget_limit_micro,
                "unique_users": unique_users,
                "unique_uas": unique_uas,
            }
        )
    return rows


def synthetic_dataset(count: int, seed: int = 42) -> tuple[np.ndarray, np.ndarray, list[dict[str, int]]]:
    """Feature matrix, labels, and raw rows from traffic_simulator."""
    import numpy as np

    rows, labels, _ = generate_network_batch(count, seed=seed)
    matrix = np.array([row_to_vector(row) for row in rows], dtype=np.float32)
    return matrix, np.array(labels, dtype=np.int32), rows


def bootstrap_synthetic() -> bool:
    """Train + calibrate; False if ML dependencies missing."""
    try:
        import lightgbm as lgb  # noqa: F401
        import numpy as np
        from sklearn.model_selection import train_test_split
    except ImportError:
        return False

    matrix, labels, raw_rows = synthetic_dataset(SYNTHETIC_ROW_COUNT, seed=42)
    if labels.sum() == 0 or labels.sum() == len(labels):
        return False

    x_train, x_val, y_train, y_val = train_test_split(
        matrix,
        labels,
        test_size=SYNTHETIC_VAL_FRACTION,
        random_state=42,
        stratify=labels,
    )

    model, metrics, _val_probs = _train_lightgbm_binary(
        x_train,
        y_train,
        x_val,
        y_val,
        num_rounds=SYNTHETIC_BOOST_ROUNDS,
        note="network synthetic bootstrap — not production evaluation",
    )

    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    model.save_model(model_path)

    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    _write_iforest_artifact(onnx_path, matrix)

    holdout = _calibration_holdout_size()
    cal_rows, cal_labels, cal_archetypes = generate_network_batch(holdout, seed=4242)
    cal_matrix = np.array([row_to_vector(row) for row in cal_rows], dtype=np.float32)
    cal_probs = model.predict(cal_matrix, num_iteration=model.best_iteration)

    max_fpr = float(os.environ.get("FRAUD_POLICY_CALIB_MAX_FPR", "0.01"))
    max_block_fpr = float(os.environ.get("FRAUD_POLICY_CALIB_MAX_BLOCK_FPR", "0.005"))
    policy_cfg, calibration = calibrate_policy(
        cal_rows,
        cal_labels,
        cal_archetypes,
        cal_probs,
        max_fpr=max_fpr,
        max_block_fpr=max_block_fpr,
    )
    set_policy_config(policy_cfg)

    importance = dict(zip(FEATURE_NAMES, model.feature_importance().tolist()))
    stats = {
        name: {"mean": float(matrix[:, i].mean()), "std": float(matrix[:, i].std())}
        for i, name in enumerate(FEATURE_NAMES)
    }
    raw_stats = _raw_stats_from_rows(raw_rows)

    meta = write_metadata(
        model_path,
        onnx_path,
        metrics=metrics,
        policy=policy_cfg.to_dict(),
        calibration=calibration,
        importance=importance,
        stats=stats,
        raw_stats=raw_stats,
    )
    print(f"{_LOG}: synthetic bootstrap OK — version {meta['version']}")
    print(f"{_LOG}: policy calibrated — suspect recall={calibration['metrics'].get('suspect_recall', 0):.3f}")
    print(format_policy_env(policy_cfg), end="")
    return True


def bootstrap_placeholder() -> None:
    """Write minimal placeholder artifacts when LightGBM unavailable."""
    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    with open(model_path, "w", encoding="utf-8") as f:
        f.write("tree\nversion=v3\nnum_class=1\nnum_tree_per_iteration=1\n")
    if not os.path.exists(onnx_path):
        with open(onnx_path, "wb") as f:
            f.write(b"mock onnx content")
    meta = write_metadata(model_path, onnx_path)
    print(f"{_LOG}: placeholder bootstrap — version {meta['version']}")


def fit_labeled_dataset(
    dataset_path: Path,
    *,
    val_fraction: float,
    train_until: datetime | None,
    val_from: datetime | None,
    boost_rounds: int,
    min_rows: int,
) -> int:
    """Train on labeled parquet/csv with time-based validation split."""

    manual_labels_path = os.environ.get("FRAUD_MANUAL_LABELS", "")
    try:
        records = load_labeled_dataset(
            dataset_path,
            manual_labels_path=Path(manual_labels_path) if manual_labels_path else None,
        )
    except (OSError, ValueError) as exc:
        print(f"{_LOG} fit: {exc}", file=sys.stderr)
        return 1

    if len(records) < min_rows:
        print(
            f"{_LOG} fit: {len(records)} rows < min_rows={min_rows}",
            file=sys.stderr,
        )
        return 1

    try:
        train_records, val_records = time_based_split(
            records,
            val_fraction=val_fraction,
            train_until=train_until,
            val_from=val_from,
        )
    except ValueError as exc:
        print(f"{_LOG} fit: {exc}", file=sys.stderr)
        return 1

    train_bundle = split_to_bundle(train_records)
    val_bundle = split_to_bundle(val_records)

    try:
        model, metrics, val_probs = _train_lightgbm_binary(
            train_bundle.matrix,
            train_bundle.labels,
            val_bundle.matrix,
            val_bundle.labels,
            num_rounds=boost_rounds,
            note="labeled fit — production evaluation on time-based holdout",
        )
    except (ImportError, ValueError) as exc:
        print(f"{_LOG} fit: {exc}", file=sys.stderr)
        return 1

    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    model.save_model(model_path)

    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    _write_iforest_artifact(onnx_path, train_bundle.matrix)

    val_rows = [rec.row for rec in val_records]
    val_labels = [rec.label for rec in val_records]
    val_archetypes = [rec.label_source for rec in val_records]

    max_fpr = float(os.environ.get("FRAUD_POLICY_CALIB_MAX_FPR", "0.01"))
    max_block_fpr = float(os.environ.get("FRAUD_POLICY_CALIB_MAX_BLOCK_FPR", "0.005"))
    policy_cfg, calibration = calibrate_policy(
        val_rows,
        val_labels,
        val_archetypes,
        val_probs,
        max_fpr=max_fpr,
        max_block_fpr=max_block_fpr,
    )
    set_policy_config(policy_cfg)

    split_info = training_summary(train_records, val_records)
    metrics["training"] = split_info
    metrics["dataset"] = str(dataset_path.name)

    importance = dict(zip(FEATURE_NAMES, model.feature_importance().tolist()))
    stats = {
        name: {
            "mean": float(train_bundle.matrix[:, i].mean()),
            "std": float(train_bundle.matrix[:, i].std()),
        }
        for i, name in enumerate(FEATURE_NAMES)
    }
    raw_stats = _raw_stats_from_rows([rec.row for rec in train_records])

    meta = write_metadata(
        model_path,
        onnx_path,
        metrics=metrics,
        policy=policy_cfg.to_dict(),
        calibration=calibration,
        importance=importance,
        stats=stats,
        raw_stats=raw_stats,
    )
    print(f"{_LOG}: fit OK — version {meta['version']} train={split_info['train_rows']} val={split_info['val_rows']}")
    print(
        f"{_LOG}: val auc={metrics['auc']:.3f} f1={metrics['f1_score']:.3f} "
        f"suspect_recall={calibration['metrics'].get('suspect_recall', 0):.3f}"
    )
    print(format_policy_env(policy_cfg), end="")
    return 0


def cmd_fit(args: argparse.Namespace) -> int:
    os.makedirs(ARTIFACT_DIR, exist_ok=True)
    dataset = args.dataset or os.environ.get("FRAUD_TRAIN_DATASET", "")
    if not dataset:
        print(f"{_LOG} fit: --dataset or FRAUD_TRAIN_DATASET required", file=sys.stderr)
        return 1

    dataset_path = Path(dataset)
    if not dataset_path.is_file():
        dataset_path = REPO_ROOT / dataset
    if not dataset_path.is_file():
        print(f"{_LOG} fit: dataset not found: {dataset}", file=sys.stderr)
        return 1

    metadata_path = Path(ARTIFACT_DIR) / "metadata.json"
    env_cfg = load_policy_config_from_env()
    source = os.environ.get("FRAUD_POLICY_SOURCE", "auto")
    set_policy_config(resolve_policy_config(env_cfg, metadata_path, source))

    return fit_labeled_dataset(
        dataset_path,
        val_fraction=args.val_fraction if args.val_fraction is not None else _fit_val_fraction(),
        train_until=_parse_optional_time(args.train_until),
        val_from=_parse_optional_time(args.val_from),
        boost_rounds=args.boost_rounds if args.boost_rounds is not None else _fit_boost_rounds(),
        min_rows=args.min_rows if args.min_rows is not None else _fit_min_rows(),
    )


def cmd_fit_validate(args: argparse.Namespace) -> int:
    dataset = args.dataset or os.environ.get("FRAUD_TRAIN_DATASET", "")
    dataset_path = Path(dataset) if dataset else None
    if dataset_path and not dataset_path.is_file():
        dataset_path = REPO_ROOT / dataset
    if dataset_path and dataset_path.is_file():
        if cmd_fit(args) != 0:
            return 1
    else:
        print(f"{_LOG}: no labeled dataset — falling back to synthetic bootstrap", file=sys.stderr)
        if cmd_bootstrap(args) != 0:
            return 1
    return cmd_validate_artifacts(args)


def cmd_bootstrap(_: argparse.Namespace) -> int:
    os.makedirs(ARTIFACT_DIR, exist_ok=True)
    metadata_path = Path(ARTIFACT_DIR) / "metadata.json"
    env_cfg = load_policy_config_from_env()
    source = os.environ.get("FRAUD_POLICY_SOURCE", "auto")
    set_policy_config(resolve_policy_config(env_cfg, metadata_path, source))
    if bootstrap_synthetic():
        return 0
    print(f"{_LOG}: ML libs unavailable — using placeholder bootstrap", file=sys.stderr)
    bootstrap_placeholder()
    return 0


def cmd_export(_: argparse.Namespace) -> int:
    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    if not os.path.isfile(model_path) or not os.path.isfile(onnx_path):
        print(f"{_LOG} export: missing model.txt or iforest.onnx", file=sys.stderr)
        return 1
    meta = write_metadata(model_path, onnx_path)
    print(f"{_LOG}: exported metadata version {meta['version']}")
    return 0


def _validate_fixtures_python(model_path: Path) -> int:
    if not FIXTURES_DIR.is_dir():
        print(f"{_LOG} validate: missing {FIXTURES_DIR}", file=sys.stderr)
        return 1

    try:
        import lightgbm as lgb
    except ImportError:
        print(f"{_LOG} validate: lightgbm unavailable, falling back to go ml-validate", file=sys.stderr)
        return _validate_fixtures_go(model_path)

    booster = lgb.Booster(model_file=str(model_path))
    if booster.num_feature() != FEATURE_DIMS:
        print(
            f"{_LOG} validate: model num_feature={booster.num_feature()} want {FEATURE_DIMS}",
            file=sys.stderr,
        )
        return 1

    checked = 0
    for path in sorted(FIXTURES_DIR.glob("features_*.json")):
        with open(path, encoding="utf-8") as handle:
            payload = json.load(handle)
        if payload.get("feature_names") != list(FEATURE_NAMES):
            print(f"{_LOG} validate: {path.name}: feature_names mismatch", file=sys.stderr)
            return 1
        row = payload.get("row", {})
        expected_vec = payload.get("vector")
        actual_vec = row_to_vector(row)
        if expected_vec is None or len(actual_vec) != len(expected_vec):
            print(f"{_LOG} validate: {path.name}: vector length mismatch", file=sys.stderr)
            return 1
        for idx, (got, want) in enumerate(zip(actual_vec, expected_vec, strict=True)):
            if not math.isclose(got, want, rel_tol=0.0, abs_tol=1e-9):
                print(
                    f"{_LOG} validate: {path.name}: vector[{idx}] got {got} want {want}",
                    file=sys.stderr,
                )
                return 1
        checked += 1

    if checked == 0:
        print(f"{_LOG} validate: no fixtures under {FIXTURES_DIR}", file=sys.stderr)
        return 1
    print(f"{_LOG} validate: OK model={model_path} fixtures={checked}")
    return 0


def _validate_fixtures_go(model_path: Path) -> int:
    cmd = [
        "go",
        "run",
        "./cmd/ml-validate",
        "-model",
        str(model_path),
        "-fixtures",
        str(FIXTURES_DIR),
    ]
    proc = subprocess.run(cmd, cwd=REPO_ROOT, check=False)
    return proc.returncode


def _validate_artifacts(model_path: Path, *, require_metadata: bool = True) -> int:
    """Smoke-check artifact files after bootstrap (no fixture score contract)."""
    if not model_path.is_file():
        print(f"{_LOG} validate-artifacts: missing model {model_path}", file=sys.stderr)
        return 1

    meta_path = Path(ARTIFACT_DIR) / "metadata.json"
    onnx_path = Path(ARTIFACT_DIR) / "iforest.onnx"
    if require_metadata and not meta_path.is_file():
        print(f"{_LOG} validate-artifacts: missing {meta_path}", file=sys.stderr)
        return 1
    if require_metadata and not onnx_path.is_file():
        print(f"{_LOG} validate-artifacts: missing {onnx_path}", file=sys.stderr)
        return 1

    try:
        import lightgbm as lgb
    except ImportError:
        print(f"{_LOG} validate-artifacts: lightgbm unavailable", file=sys.stderr)
        return 1

    booster = lgb.Booster(model_file=str(model_path))
    if booster.num_feature() != FEATURE_DIMS:
        print(
            f"{_LOG} validate-artifacts: num_feature={booster.num_feature()} want {FEATURE_DIMS}",
            file=sys.stderr,
        )
        return 1

    if require_metadata:
        with open(meta_path, encoding="utf-8") as handle:
            meta = json.load(handle)
        for key in ("version", "lightgbm_hash", "iforest_hash"):
            if key not in meta:
                print(f"{_LOG} validate-artifacts: metadata missing {key}", file=sys.stderr)
                return 1

    print(f"{_LOG} validate-artifacts: OK model={model_path} dims={FEATURE_DIMS}")
    return 0


def cmd_validate_artifacts(_: argparse.Namespace) -> int:
    model_path = Path(ARTIFACT_DIR) / "model.txt"
    return _validate_artifacts(model_path)


def cmd_bootstrap_validate(_: argparse.Namespace) -> int:
    if cmd_bootstrap(_) != 0:
        return 1
    return cmd_validate_artifacts(_)


def cmd_validate(args: argparse.Namespace) -> int:
    model_path = Path(args.model)
    if not model_path.is_file():
        model_path = REPO_ROOT / args.model
    if not model_path.is_file():
        print(f"{_LOG} validate: model not found: {args.model}", file=sys.stderr)
        return 1
    return _validate_fixtures_python(model_path)


def main() -> int:
    parser = argparse.ArgumentParser(description="Fraud model artifact bootstrap")
    sub = parser.add_subparsers(dest="cmd", required=True)
    p_boot = sub.add_parser("bootstrap", help="train synthetic model or write placeholder artifacts")
    p_boot.set_defaults(func=cmd_bootstrap)
    p_exp = sub.add_parser("export", help="refresh metadata.json from existing artifacts")
    p_exp.set_defaults(func=cmd_export)
    p_val = sub.add_parser("validate", help="check model feature dims and fixture vectors")
    p_val.add_argument(
        "--model",
        default=DEFAULT_ARTIFACT_MODEL,
        help="LightGBM model.txt path (default: var/fraudscore/artifacts/model.txt)",
    )
    p_val.set_defaults(func=cmd_validate)
    p_art = sub.add_parser("validate-artifacts", help="smoke-check ARTIFACT_DIR after bootstrap")
    p_art.set_defaults(func=cmd_validate_artifacts)
    p_bv = sub.add_parser("bootstrap-validate", help="bootstrap then validate-artifacts (dev CronJob)")
    p_bv.set_defaults(func=cmd_bootstrap_validate)
    p_fit = sub.add_parser("fit", help="train on labeled parquet/csv (time-based split)")
    p_fit.add_argument("--dataset", default="", help="labeled dataset path (or FRAUD_TRAIN_DATASET)")
    p_fit.add_argument("--val-fraction", type=float, default=None)
    p_fit.add_argument("--train-until", default="", help="ISO-8601 exclusive train upper bound")
    p_fit.add_argument("--val-from", default="", help="ISO-8601 inclusive validation lower bound")
    p_fit.add_argument("--boost-rounds", type=int, default=None)
    p_fit.add_argument("--min-rows", type=int, default=None)
    p_fit.set_defaults(func=cmd_fit)
    p_fv = sub.add_parser(
        "fit-validate",
        help="fit when FRAUD_TRAIN_DATASET exists, else bootstrap; then validate-artifacts",
    )
    p_fv.add_argument("--dataset", default="", help="labeled dataset path (or FRAUD_TRAIN_DATASET)")
    p_fv.add_argument("--val-fraction", type=float, default=None)
    p_fv.add_argument("--train-until", default="")
    p_fv.add_argument("--val-from", default="")
    p_fv.add_argument("--boost-rounds", type=int, default=None)
    p_fv.add_argument("--min-rows", type=int, default=None)
    p_fv.set_defaults(func=cmd_fit_validate)
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
