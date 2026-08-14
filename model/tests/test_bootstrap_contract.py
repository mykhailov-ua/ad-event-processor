"""Bootstrap must fail loudly when synthetic training is invalid."""

from __future__ import annotations

import pytest


def test_bootstrap_synthetic_rejects_single_class_dataset() -> None:
    pytest.importorskip("lightgbm")
    import numpy as np

    import artifact_bootstrap as bootstrap_mod

    def _single_class(_count: int, seed: int = 42):
        rows = [
            {"events": 10, "clicks": 1, "spend_micro": 1, "budget_limit_micro": 1, "unique_users": 1, "unique_uas": 1}
        ]
        labels = np.zeros(1, dtype=np.int32)
        matrix = np.zeros((1, 16), dtype=np.float32)
        return matrix, labels, rows

    bootstrap_mod.synthetic_dataset = _single_class  # type: ignore[method-assign]
    with pytest.raises(ValueError, match="single class"):
        bootstrap_mod.bootstrap_synthetic()
