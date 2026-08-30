"""ClickHouse HTTP client from CH_DSN / CH_READONLY_DSN and CH_HTTP_PORT.

Role:
- Resolve connection params from DSN URL or CH_HOST/CH_USER/CH_PASSWORD fallbacks.
- Prefer CH_READONLY_DSN for training/export jobs (read-only CH user).

Env:
- CH_READONLY_DSN or CH_DSN: clickhouse://user:pass@host/dbname
- CH_HTTP_PORT: HTTP interface port (default 8123)
- CH_HOST, CH_USER, CH_PASSWORD, CH_NAME: used when DSN unset

Verify:
  pytest model/tests/test_clickhouse_client.py -q
  python3 -c "from data.clickhouse_client import clickhouse_config_from_env; print(clickhouse_config_from_env())"
"""

from __future__ import annotations

import os
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass
from typing import Any
from urllib.parse import unquote, urlparse


@dataclass(frozen=True)
class ClickHouseConfig:
    """Resolved ClickHouse HTTP endpoint and credentials."""

    host: str
    port: int
    username: str
    password: str
    database: str


def clickhouse_config_from_env() -> ClickHouseConfig:
    """Build config from env; readonly DSN wins over primary CH_DSN."""
    dsn = os.environ.get("CH_READONLY_DSN") or os.environ.get("CH_DSN", "")
    http_port = int(os.environ.get("CH_HTTP_PORT", "8123"))
    if dsn:
        parsed = urlparse(dsn)
        host = parsed.hostname or "127.0.0.1"
        database = (parsed.path or "/default").lstrip("/") or "default"
        username = unquote(parsed.username or os.environ.get("CH_USER", "default"))
        password = unquote(parsed.password or os.environ.get("CH_PASSWORD", ""))
        return ClickHouseConfig(
            host=host,
            port=http_port,
            username=username,
            password=password,
            database=database,
        )
    return ClickHouseConfig(
        host=os.environ.get("CH_HOST", "127.0.0.1"),
        port=http_port,
        username=os.environ.get("CH_USER", "default"),
        password=os.environ.get("CH_PASSWORD", ""),
        database=os.environ.get("CH_NAME", "ad_event_processor"),
    )


def connect_client(config: ClickHouseConfig | None = None) -> Any:
    """Return clickhouse_connect client; lazy-imports optional dependency."""
    import clickhouse_connect

    resolved_config = config or clickhouse_config_from_env()
    return clickhouse_connect.get_client(
        host=resolved_config.host,
        port=resolved_config.port,
        username=resolved_config.username,
        password=resolved_config.password,
        database=resolved_config.database,
    )


def close_client(client: Any) -> None:
    close = getattr(client, "close", None)
    if callable(close):
        close()


@contextmanager
def clickhouse_client(config: ClickHouseConfig | None = None) -> Iterator[Any]:
    """Open a ClickHouse client and close it when the block exits."""
    client = connect_client(config)
    try:
        yield client
    finally:
        close_client(client)


def ping_client(client: Any) -> bool:
    """Return True when SELECT 1 succeeds."""
    try:
        client.command("SELECT 1")
    except (OSError, ConnectionError, TimeoutError, ValueError):
        return False
    else:
        return True
