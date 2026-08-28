"""Repository paths shared across model subpackages."""

from __future__ import annotations

from pathlib import Path

MODEL_DIR = Path(__file__).resolve().parent
REPO_ROOT = MODEL_DIR.parent
TRACKED_FIXTURE_DIR = REPO_ROOT / "internal" / "fraud" / "testdata"
EPHEMERAL_FIXTURE_DIR = REPO_ROOT / "var" / "fraudscore" / "fixtures"
