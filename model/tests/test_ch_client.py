"""Tests for ClickHouse client lifecycle helpers."""

from __future__ import annotations

from ch_client import ch_client, close_client


class _FakeClient:
    def __init__(self) -> None:
        self.closed = False

    def close(self) -> None:
        self.closed = True


def test_close_client_invokes_close() -> None:
    client = _FakeClient()
    close_client(client)
    assert client.closed is True


def test_ch_client_closes_on_exit() -> None:
    created: list[_FakeClient] = []

    class _Factory:
        def connect(self):
            client = _FakeClient()
            created.append(client)
            return client

    import ch_client as mod

    original = mod.connect_client
    mod.connect_client = lambda _config=None: _Factory().connect()
    try:
        with ch_client():
            assert len(created) == 1
            assert created[0].closed is False
        assert created[0].closed is True
    finally:
        mod.connect_client = original
