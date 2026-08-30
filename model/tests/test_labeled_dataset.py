"""Time-based split invariants for labeled training sets.

Role: time_based_split preserves chronological order and rejects overlapping train/val windows.
Tier: fast (unit).
Infra: in-memory LabeledRecord rows only.
Invariants proved: val_fraction tail is strictly after train; overlapping train_until/val_from raises ValueError.
Verify: cd model && python3 -m pytest tests/test_labeled_dataset.py -q
"""

from __future__ import annotations

from datetime import UTC, datetime

import pytest

from train.labeled_dataset import LabeledRecord, time_based_split

def _record(minute: int, label: int = 0) -> LabeledRecord:
    window = datetime(2026, 1, 1, 0, minute, tzinfo=UTC)
    row = {
        "events": 10,
        "clicks": 1,
        "spend_micro": 100,
        "budget_limit_micro": 1000,
        "unique_users": 2,
        "unique_uas": 2,
    }
    return LabeledRecord(window_start=window, row=row, label=label, label_source="test")

def test_time_based_split_preserves_order() -> None:
    records = [_record(i) for i in range(10)]
    train, val = time_based_split(records, val_fraction=0.2)
    assert len(train) == 8
    assert len(val) == 2
    assert train[-1].window_start < val[0].window_start

def test_time_based_split_rejects_overlap() -> None:
    records = [_record(i) for i in range(6)]
    with pytest.raises(ValueError, match="overlap"):
        time_based_split(
            records, train_until=datetime(2026, 1, 1, 0, 4, tzinfo=UTC), val_from=datetime(2026, 1, 1, 0, 3, tzinfo=UTC)
        )
