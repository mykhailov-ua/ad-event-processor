"""Load labeled feature rows for time-based model training.

Role:
- Read parquet/csv from features_export (window_start + ROW_FIELDS + label).
- Time-ordered train/val split; no random shuffle across windows.
- Optional manual label overrides from FRAUD_MANUAL_LABELS CSV by ip_hash_hex.

Required columns:
- window_start, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas
- label or is_fraud (0/1)

Verify:
  pytest model/tests/test_labeled_dataset.py -q
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import TYPE_CHECKING, Any

from contract.feature_spec import FEATURE_DIMS, FEATURE_NAMES, row_to_vector
from train.traffic_simulator import Row

if TYPE_CHECKING:
    import numpy as np

ROW_FIELDS: tuple[str, ...] = (
    "events",
    "clicks",
    "spend_micro",
    "budget_limit_micro",
    "unique_users",
    "unique_uas",
)
LABEL_COLUMNS: tuple[str, ...] = ("label", "is_fraud")
TIME_COLUMN = "window_start"

@dataclass(frozen=True)
class LabeledRecord:
    """Single training row with UTC window_start and label metadata."""

    window_start: datetime
    row: dict[str, int]
    label: int
    label_source: str

@dataclass(frozen=True)
class LabeledSplit:
    records: list[LabeledRecord]
    matrix: np.ndarray
    labels: np.ndarray
    probs: np.ndarray | None = None

def _parse_window_start(value: object) -> datetime:
    if isinstance(value, datetime):
        if value.tzinfo is None:
            return value.replace(tzinfo=UTC)
        return value.astimezone(UTC)
    if isinstance(value, str):
        text = value.strip()
    else:
        iso_attr = getattr(value, "isoformat", None)
        if callable(iso_attr):
            text = str(iso_attr())
        else:
            text = str(value).strip()
    if text.endswith("Z"):
        text = text[:-1] + "+00:00"
    parsed = datetime.fromisoformat(text)
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC)

def _read_csv_stdlib(path: Path) -> dict[str, list]:
    import csv

    with path.open(encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        columns: dict[str, list] = {}
        for row in reader:
            for key, value in row.items():
                if key is None:
                    continue
                columns.setdefault(key, []).append(value)
    return columns

def _read_table(path: Path) -> Any:
    suffix = path.suffix.lower()
    if suffix == ".csv":
        try:
            import pyarrow.csv as pacsv

            return pacsv.read_csv(path)
        except ImportError:
            return _read_csv_stdlib(path)
    if suffix == ".parquet":
        import pyarrow.parquet as pq

        return pq.read_table(path)
    raise ValueError(f"unsupported dataset format: {path.suffix}")

def _column_dict(table: Any) -> dict[str, list]:
    if isinstance(table, dict):
        return table
    return {name: table[name].to_pylist() for name in table.column_names}

def _resolve_label(columns: dict[str, list], idx: int) -> int:
    for name in LABEL_COLUMNS:
        if name not in columns:
            continue
        raw = columns[name][idx]
        if raw is None:
            continue
        label = int(raw)
        if label not in (0, 1):
            raise ValueError(f"row {idx}: {name} must be 0 or 1, got {raw}")
        return label
    raise ValueError(f"row {idx}: missing label column ({', '.join(LABEL_COLUMNS)})")

def load_labeled_dataset(path: Path, manual_labels_path: Path | None = None) -> list[LabeledRecord]:
    """Load parquet/csv exported from features_export plus a label column."""
    if not path.is_file():
        raise FileNotFoundError(path)

    table = _read_table(path)
    columns = _column_dict(table)
    if TIME_COLUMN not in columns:
        raise ValueError(f"dataset missing required column: {TIME_COLUMN}")

    for field in ROW_FIELDS:
        if field not in columns:
            raise ValueError(f"dataset missing required feature column: {field}")

    row_count = len(columns[TIME_COLUMN])
    if row_count == 0:
        raise ValueError("dataset is empty")

    label_sources = columns.get("label_source", ["unknown"] * row_count)
    ip_hashes = columns.get("ip_hash_hex", [None] * row_count)
    records: list[LabeledRecord] = []
    for row_index in range(row_count):
        row = {field: int(columns[field][row_index]) for field in ROW_FIELDS}
        records.append(
            LabeledRecord(
                window_start=_parse_window_start(columns[TIME_COLUMN][row_index]),
                row=row,
                label=_resolve_label(columns, row_index),
                label_source=str(label_sources[row_index] or "unknown"),
            )
        )

    if manual_labels_path and manual_labels_path.is_file():
        records = _apply_manual_labels(records, ip_hashes, manual_labels_path)

    records.sort(key=lambda item: item.window_start)
    return records

def _apply_manual_labels(
    records: list[LabeledRecord],
    ip_hashes: list[str | None],
    manual_labels_path: Path,
) -> list[LabeledRecord]:
    import csv

    manual_labels: dict[str, tuple[int, str]] = {}
    with manual_labels_path.open(encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            ip_hash = row.get("ip_hash")
            label = row.get("label")
            if ip_hash and label is not None:
                manual_labels[ip_hash.lower()] = (int(label), row.get("source", "manual"))

    if not manual_labels:
        return records

    updated_records: list[LabeledRecord] = []
    overrides = 0
    for row_index, record in enumerate[LabeledRecord](records):
        ip_hash = ip_hashes[row_index]
        if ip_hash and ip_hash.lower() in manual_labels:
            label, source = manual_labels[ip_hash.lower()]
            updated_records.append(
                LabeledRecord(
                    window_start=record.window_start,
                    row=record.row,
                    label=label,
                    label_source=f"override:{source}",
                )
            )
            overrides += 1
        else:
            updated_records.append(record)

    if overrides > 0:
        print(f"labeled_dataset: applied {overrides} manual label overrides", flush=True)

    return updated_records

def time_based_split(
    records: list[LabeledRecord],
    *,
    val_fraction: float = 0.2,
    train_until: datetime | None = None,
    val_from: datetime | None = None,
) -> tuple[list[LabeledRecord], list[LabeledRecord]]:
    """Split by time order; no random shuffle across windows."""
    if not records:
        raise ValueError("no records to split")

    if train_until is not None or val_from is not None:
        if train_until is not None and val_from is not None:
            train = [r for r in records if r.window_start < train_until]
            val = [r for r in records if r.window_start >= val_from]
        elif train_until is not None:
            train = [r for r in records if r.window_start < train_until]
            val = [r for r in records if r.window_start >= train_until]
        elif val_from is not None:
            train = [r for r in records if r.window_start < val_from]
            val = [r for r in records if r.window_start >= val_from]
    else:
        if not 0.0 < val_fraction < 1.0:
            raise ValueError("val_fraction must be between 0 and 1")
        # Chronological split: earliest (1-val_fraction) rows train, tail validates.
        split_idx = max(1, int(len(records) * (1.0 - val_fraction)))
        split_idx = min(split_idx, len(records) - 1)
        train = records[:split_idx]
        val = records[split_idx:]

    if not train or not val:
        raise ValueError("time split produced empty train or validation set")
    if train[-1].window_start >= val[0].window_start:
        raise ValueError("train/val time ranges overlap; adjust split boundaries")
    return train, val

def records_to_matrix(records: list[LabeledRecord]) -> tuple[np.ndarray, np.ndarray]:
    import numpy as np

    matrix = np.array([row_to_vector(record.row) for record in records], dtype=np.float32)
    labels = np.array([record.label for record in records], dtype=np.int32)
    return matrix, labels

def split_to_bundle(records: list[LabeledRecord]) -> LabeledSplit:
    matrix, labels = records_to_matrix(records)
    return LabeledSplit(records=records, matrix=matrix, labels=labels)

def label_source_counts(records: list[LabeledRecord]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for record in records:
        counts[record.label_source] = counts.get(record.label_source, 0) + 1
    return counts

def training_summary(train: list[LabeledRecord], val: list[LabeledRecord]) -> dict[str, object]:
    return {
        "feature_dims": FEATURE_DIMS,
        "feature_names": list(FEATURE_NAMES),
        "train_rows": len(train),
        "val_rows": len(val),
        "train_window_start": train[0].window_start.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "train_window_end": train[-1].window_start.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "val_window_start": val[0].window_start.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "val_window_end": val[-1].window_start.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "train_label_sources": label_source_counts(train),
        "val_label_sources": label_source_counts(val),
        "train_fraud_rate": float(sum(record.label for record in train) / len(train)),
        "val_fraud_rate": float(sum(record.label for record in val) / len(val)),
    }

def write_synthetic_dataset(path: Path, count: int = 2000, seed: int = 42) -> None:
    """Write a labeled dataset from the traffic simulator (tests / CI smoke)."""
    import csv

    from traffic_simulator import generate_network_batch

    rows, labels, archetypes = generate_network_batch(count, seed=seed)
    base = datetime(2026, 1, 1, tzinfo=UTC)
    fieldnames = [TIME_COLUMN, *ROW_FIELDS, "label", "label_source", "ip_hash_hex", "campaign_id"]
    records: list[dict[str, object]] = []

    for idx, row in enumerate[Row](rows):
        window = base + timedelta(minutes=idx)
        record: dict[str, object] = {
            TIME_COLUMN: window.strftime("%Y-%m-%dT%H:%M:%SZ"),
            "label": int(labels[idx]),
            "label_source": archetypes[idx],
            "ip_hash_hex": f"{idx:032x}",
            "campaign_id": "00000000-0000-0000-0000-000000000001",
        }
        for field in ROW_FIELDS:
            record[field] = int(row[field])
        records.append(record)

    path.parent.mkdir(parents=True, exist_ok=True)
    if path.suffix.lower() == ".parquet":
        import pyarrow as pa
        import pyarrow.parquet as pq

        columns: dict[str, list] = {name: [] for name in fieldnames}
        for record in records:
            for name in fieldnames:
                columns[name].append(record[name])
        pq.write_table(pa.table(columns), path)
        return

    if path.suffix.lower() != ".csv":
        path = path.with_suffix(".csv")

    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(records)

def write_synthetic_parquet(path: Path, count: int = 2000, seed: int = 42) -> None:
    write_synthetic_dataset(path, count=count, seed=seed)
