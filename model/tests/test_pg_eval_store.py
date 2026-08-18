"""Tests for Postgres eval report persistence."""

from __future__ import annotations

from pg_eval_store import EVAL_REPORT_ID, upsert_ml_eval_report


def test_upsert_noop_without_dsn(monkeypatch) -> None:
    monkeypatch.delenv("DB_DSN", raising=False)
    upsert_ml_eval_report({"status": "ok", "precision": 0.9, "recall": 0.4})


def test_upsert_calls_postgres(monkeypatch) -> None:
    calls: list[tuple] = []

    class _Cursor:
        def execute(self, sql, params):
            calls.append((sql, params))

        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

    class _Conn:
        def cursor(self):
            return _Cursor()

        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def close(self) -> None:
            return None

    monkeypatch.setenv("DB_DSN", "postgres://user:pass@localhost/db")

    import psycopg2

    monkeypatch.setattr(psycopg2, "connect", lambda _dsn: _Conn())

    upsert_ml_eval_report(
        {
            "status": "error",
            "precision": 0.0,
            "recall": 0.0,
            "label_method": "proxy",
            "generated_at": "2026-03-01T10:00:00Z",
            "drift": {"drift_detected": False},
        }
    )
    assert len(calls) == 1
    assert calls[0][1][0] == EVAL_REPORT_ID
    assert calls[0][1][5] == "error"
