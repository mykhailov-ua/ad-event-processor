"""Metadata contract: no fabricated training metrics.

Role: write_metadata must not invent accuracy/AUC when metrics omitted.
Tier: fast (unit).
Infra: tempfile artifact dir only.
Invariants proved: metrics key absent unless explicitly passed; placeholder note allowed without fake scores.
Verify: cd model && python3 -m pytest tests/test_artifact_metadata.py -q
"""

from __future__ import annotations

import json
import tempfile
from pathlib import Path

import train.artifact_bootstrap as artifact_bootstrap_mod
from train.artifact_bootstrap import write_metadata

def test_write_metadata_omits_metrics_when_not_provided() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        artifact_bootstrap_mod.ARTIFACT_DIR = tmp
        model_path = Path(tmp) / "model.txt"
        onnx_path = Path(tmp) / "iforest.onnx"
        model_path.write_text("tree\n", encoding="utf-8")
        onnx_path.write_bytes(b"onnx")

        metadata = write_metadata(str(model_path), str(onnx_path))

        assert "metrics" not in metadata
        stored = json.loads((Path(tmp) / "metadata.json").read_text(encoding="utf-8"))
        assert "metrics" not in stored
        assert "accuracy" not in stored
        assert "auc" not in stored

def test_write_metadata_placeholder_note_only() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        artifact_bootstrap_mod.ARTIFACT_DIR = tmp
        model_path = Path(tmp) / "model.txt"
        onnx_path = Path(tmp) / "iforest.onnx"
        model_path.write_text("tree\n", encoding="utf-8")
        onnx_path.write_bytes(b"onnx")

        metrics = {"note": "placeholder_no_ml_deps"}
        metadata = write_metadata(str(model_path), str(onnx_path), metrics=metrics)

        assert metadata["metrics"] == metrics
        assert "accuracy" not in metadata["metrics"]
        assert "f1_score" not in metadata["metrics"]
        assert "auc" not in metadata["metrics"]
