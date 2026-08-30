"""Shadow precision reports must disclose proxy labels.

Role: eval reports distinguish PROXY vs AUDITED label methods in JSON and markdown output.
Tier: fast (unit).
Infra: fake ClickHouse client; no live CH queries on empty audited path.
Invariants proved: proxy labels disclosed as not human-audited; audited empty skips CH; mixed markdown never sells proxy as accuracy.
Verify: cd model && python3 -m pytest tests/test_shadow_precision.py -q
"""

from __future__ import annotations

from eval.shadow_precision import (
    AUDITED_LABEL_DEFINITION,
    AUDITED_LABEL_METHOD,
    PROXY_LABEL_DEFINITION,
    PROXY_LABEL_METHOD,
    format_markdown,
    run_audited_precision,
    run_shadow_precision,
)

def test_run_shadow_precision_empty_includes_proxy_label_metadata() -> None:
    class _EmptyResult:
        def __init__(self) -> None:
            self.result_rows: list[object] = []
            self.column_names: list[str] = []

    class _Client:
        def query(self, *_args, **_kwargs):
            return _EmptyResult()

    report = run_shadow_precision(_Client(), hours=24, threshold=0.6)
    assert report["label_method"] == PROXY_LABEL_METHOD
    assert report["label_definition"] == PROXY_LABEL_DEFINITION

def test_format_markdown_discloses_proxy_labels() -> None:
    markdown = format_markdown(
        {
            "status": "empty",
            "labeled_rows": 0,
            "hours": 24,
            "threshold": 0.6,
            "label_method": PROXY_LABEL_METHOD,
            "label_definition": PROXY_LABEL_DEFINITION,
        }
    )
    assert "proxy" in markdown.lower()
    assert "not human-audited" in markdown.lower()
    assert PROXY_LABEL_DEFINITION in markdown

def test_run_audited_precision_empty_labels() -> None:
    class _Client:
        def query(self, *_args, **_kwargs):
            raise AssertionError("clickhouse should not be queried without labels")

    report = run_audited_precision(_Client(), [], hours=24, threshold=0.6)
    assert report["labeled_rows"] == 0
    assert report["label_method"] == AUDITED_LABEL_METHOD
    assert report["confidence"] == "low"
    assert report["status"] == "empty"

def test_run_audited_precision_with_fixture_labels() -> None:
    class _Result:
        def __init__(self) -> None:
            self.column_names = ["tp", "fp", "fn", "tn", "matched_rows"]
            self.result_rows = [(1, 0, 1, 2, 3)]

    class _Client:
        def query(self, _sql, *, parameters):
            assert len(parameters["ip_hashes"]) == 2
            assert parameters["labels"] == [1, 0]
            return _Result()

    labels = [
        ("0123456789abcdef0123456789abcdef", 1),
        ("fedcba9876543210fedcba9876543210", 0),
    ]
    report = run_audited_precision(_Client(), labels, hours=48, threshold=0.7)
    assert report["labeled_rows"] == 2
    assert report["matched_rows"] == 3
    assert report["precision"] == 1.0
    assert report["label_method"] == AUDITED_LABEL_METHOD
    assert report["label_definition"] == AUDITED_LABEL_DEFINITION
    assert report["confidence"] == "low"

def test_format_markdown_mixed_never_calls_proxy_accuracy() -> None:
    markdown = format_markdown(
        {
            "status": "ok",
            "generated_at": "2026-01-01T00:00:00Z",
            "hours": 24,
            "threshold": 0.6,
            "proxy_metrics": {
                "status": "ok",
                "labeled_rows": 100,
                "label_method": PROXY_LABEL_METHOD,
                "label_definition": PROXY_LABEL_DEFINITION,
                "tp": 1,
                "fp": 0,
                "fn": 0,
                "tn": 1,
                "precision": 1.0,
                "recall": 1.0,
                "f1": 1.0,
                "false_positive_rate": 0.0,
            },
            "audited_metrics": {
                "status": "empty",
                "labeled_rows": 0,
                "label_method": AUDITED_LABEL_METHOD,
                "confidence": "low",
            },
        }
    )
    assert "Audited metrics" in markdown
    assert "not accuracy" in markdown.lower()
    assert "labeled rows: 0" in markdown.lower()
