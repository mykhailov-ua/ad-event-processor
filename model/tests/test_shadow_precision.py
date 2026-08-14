"""Shadow precision reports must disclose proxy labels."""

from __future__ import annotations

from shadow_precision import (
    PROXY_LABEL_DEFINITION,
    PROXY_LABEL_METHOD,
    format_markdown,
    run_shadow_precision,
)


def test_run_shadow_precision_empty_includes_proxy_label_metadata() -> None:
    class _EmptyResult:
        def __init__(self) -> None:
            self.result_rows: list[object] = []

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
