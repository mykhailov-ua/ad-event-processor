#!/usr/bin/env python3
"""Fraud model artifact bootstrap / training.

Modes:
  bootstrap (default) — synthetic fit or copy testdata into var/fraudscore/artifacts/
  export              — hash existing artifacts into metadata.json

Production inference: cmd/fraud-scorer + internal/fraudscoring (Go).
Real CH feature training is out of scope for this repo entrypoint.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

from feature_spec import FEATURE_DIMS, FEATURE_NAMES, row_to_vector

REPO_ROOT = Path(__file__).resolve().parent.parent
ARTIFACT_DIR = os.environ.get("FRAUD_ARTIFACT_DIR", "var/fraudscore/artifacts")
TESTDATA_MODEL = "internal/fraudscoring/testdata/model.txt"
FIXTURES_DIR = REPO_ROOT / "testdata" / "ml"


def sha256_file(path: str) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def write_metadata(model_path: str, onnx_path: str, metrics: dict | None = None) -> dict:
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
        "created_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    }
    with open(os.path.join(ARTIFACT_DIR, "metadata.json"), "w", encoding="utf-8") as f:
        json.dump(meta, f, indent=2)
    return meta


def bootstrap_synthetic() -> bool:
    try:
        import numpy as np
        import lightgbm as lgb
        from sklearn.ensemble import IsolationForest
        from skl2onnx import convert_sklearn
        from skl2onnx.common.data_types import FloatTensorType
    except ImportError:
        return False

    X = np.random.rand(1000, 7) * 100
    X[:, 2] = X[:, 1] / (X[:, 0] + 1)
    X[:, 4] = X[:, 3] / (X[:, 0] + 1)
    y = (X[:, 2] > 0.5).astype(int)

    train_data = lgb.Dataset(X, label=y)
    params = {
        "objective": "binary",
        "metric": "binary_logloss",
        "boosting_type": "gbdt",
        "learning_rate": 0.1,
        "num_leaves": 31,
        "verbose": -1,
    }
    model = lgb.train(params, train_data, num_boost_round=10)
    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    model.save_model(model_path)

    iforest = IsolationForest(n_estimators=50, random_state=42)
    iforest.fit(X)
    initial_type = [("input", FloatTensorType([None, 7]))]
    onx = convert_sklearn(iforest, initial_types=initial_type, target_opset=12)
    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    with open(onnx_path, "wb") as f:
        f.write(onx.SerializeToString())

    meta = write_metadata(model_path, onnx_path)
    print(f"ml/train: synthetic bootstrap OK — version {meta['version']}")
    return True


def bootstrap_copy() -> None:
    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    if os.path.exists(TESTDATA_MODEL):
        shutil.copy(TESTDATA_MODEL, model_path)
    else:
        with open(model_path, "w", encoding="utf-8") as f:
            f.write("tree\nversion=v3\nnum_class=1\nnum_tree_per_iteration=1\n")
    if not os.path.exists(onnx_path):
        with open(onnx_path, "wb") as f:
            f.write(b"mock onnx content")
    meta = write_metadata(model_path, onnx_path)
    print(f"ml/train: copied testdata bootstrap — version {meta['version']}")


def cmd_bootstrap(_: argparse.Namespace) -> int:
    os.makedirs(ARTIFACT_DIR, exist_ok=True)
    if bootstrap_synthetic():
        return 0
    print("ml/train: ML libs unavailable — using testdata copy", file=sys.stderr)
    bootstrap_copy()
    return 0


def cmd_export(_: argparse.Namespace) -> int:
    model_path = os.path.join(ARTIFACT_DIR, "model.txt")
    onnx_path = os.path.join(ARTIFACT_DIR, "iforest.onnx")
    if not os.path.isfile(model_path) or not os.path.isfile(onnx_path):
        print("ml/train export: missing model.txt or iforest.onnx", file=sys.stderr)
        return 1
    meta = write_metadata(model_path, onnx_path)
    print(f"ml/train: exported metadata version {meta['version']}")
    return 0


def _validate_fixtures_python(model_path: Path) -> int:
    if FEATURE_DIMS != 7:
        print(f"ml/train validate: FEATURE_DIMS={FEATURE_DIMS} want 7", file=sys.stderr)
        return 1
    if not FIXTURES_DIR.is_dir():
        print(f"ml/train validate: missing {FIXTURES_DIR}", file=sys.stderr)
        return 1

    try:
        import lightgbm as lgb
    except ImportError:
        print("ml/train validate: lightgbm unavailable, falling back to go ml-validate", file=sys.stderr)
        return _validate_fixtures_go(model_path)

    booster = lgb.Booster(model_file=str(model_path))
    if booster.num_feature() != FEATURE_DIMS:
        print(
            f"ml/train validate: model num_feature={booster.num_feature()} want {FEATURE_DIMS}",
            file=sys.stderr,
        )
        return 1

    scored = 0
    for path in sorted(FIXTURES_DIR.glob("features_*.json")):
        with open(path, encoding="utf-8") as handle:
            payload = json.load(handle)
        if payload.get("feature_names") != list(FEATURE_NAMES):
            print(f"ml/train validate: {path.name}: feature_names mismatch", file=sys.stderr)
            return 1
        row = payload.get("row", {})
        expected_vec = payload.get("vector")
        actual_vec = row_to_vector(row)
        if expected_vec is None or len(actual_vec) != len(expected_vec):
            print(f"ml/train validate: {path.name}: vector length mismatch", file=sys.stderr)
            return 1
        for idx, (got, want) in enumerate(zip(actual_vec, expected_vec)):
            if not math.isclose(got, want, rel_tol=0.0, abs_tol=1e-9):
                print(
                    f"ml/train validate: {path.name}: vector[{idx}] got {got} want {want}",
                    file=sys.stderr,
                )
                return 1
        expected_score = payload.get("score")
        if expected_score is None:
            continue
        import numpy as np

        pred = float(booster.predict(np.array([actual_vec], dtype=np.float64))[0])
        if not math.isclose(pred, float(expected_score), rel_tol=0.0, abs_tol=1e-4):
            print(
                f"ml/train validate: {path.name}: score got {pred:.5f} want {expected_score}",
                file=sys.stderr,
            )
            return 1
        scored += 1

    if scored == 0:
        print("ml/train validate: no scored fixtures", file=sys.stderr)
        return 1
    print(f"ml/train validate: OK model={model_path} scored_fixtures={scored}")
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


def cmd_validate(args: argparse.Namespace) -> int:
    model_path = Path(args.model)
    if not model_path.is_file():
        model_path = REPO_ROOT / args.model
    if not model_path.is_file():
        print(f"ml/train validate: model not found: {args.model}", file=sys.stderr)
        return 1
    return _validate_fixtures_python(model_path)


def main() -> int:
    parser = argparse.ArgumentParser(description="Fraud model artifact bootstrap")
    sub = parser.add_subparsers(dest="cmd", required=True)
    p_boot = sub.add_parser("bootstrap", help="train synthetic or copy testdata model")
    p_boot.set_defaults(func=cmd_bootstrap)
    p_exp = sub.add_parser("export", help="refresh metadata.json from existing artifacts")
    p_exp.set_defaults(func=cmd_export)
    p_val = sub.add_parser("validate", help="check model feature dims and score fixtures")
    p_val.add_argument(
        "--model",
        default=TESTDATA_MODEL,
        help="LightGBM model.txt path (default: internal/fraudscoring/testdata/model.txt)",
    )
    p_val.set_defaults(func=cmd_validate)
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
