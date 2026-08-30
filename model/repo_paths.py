"""Repository path constants for the model/ cold-path tree.

Role:
- Resolve REPO_ROOT from this file location (model/ is one level below repo root).
- Point fixture writers and parity tests at tracked Go testdata vs ephemeral var/ output.

Paths:
- TRACKED_FIXTURE_DIR: internal/fraud/testdata (committed; consumed by cmd/ml-validate).
- EPHEMERAL_FIXTURE_DIR: var/fraudscore/fixtures (local bootstrap output).

Verify:
  python3 -c "from repo_paths import REPO_ROOT; print(REPO_ROOT)"
  pytest model/tests/test_feature_spec.py -q
"""

from __future__ import annotations

from pathlib import Path

MODEL_DIR = Path(__file__).resolve().parent
REPO_ROOT = MODEL_DIR.parent
# Committed vectors for Go/Python parity (policy_parity.json, features_*.json).
TRACKED_FIXTURE_DIR = REPO_ROOT / "internal" / "fraud" / "testdata"
# Local-only fixture output; same schema as TRACKED_FIXTURE_DIR.
EPHEMERAL_FIXTURE_DIR = REPO_ROOT / "var" / "fraudscore" / "fixtures"
