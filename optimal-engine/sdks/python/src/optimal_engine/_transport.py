"""Shared HTTP transport plumbing for sync + async clients.

Both the blocking and asyncio clients route through the helpers here so
URL building, header injection, and status-to-exception mapping live in
exactly one place. This is composition-over-inheritance — the public
clients delegate to functions, they don't subclass them.
"""

from __future__ import annotations

from typing import Any
from urllib.parse import urlencode

import httpx

from .exceptions import (
    APIConnectionError,
    APITimeoutError,
    OptimalEngineError,
    status_to_exception,
)

DEFAULT_TIMEOUT = httpx.Timeout(60.0, connect=10.0)
DEFAULT_HEADERS: dict[str, str] = {
    "Accept": "application/json",
    "User-Agent": "optimal-engine-python/0.1.0",
}


def build_url(base_url: str, path: str) -> str:
    """Join the configured base URL with a relative API path."""
    return f"{base_url.rstrip('/')}/{path.lstrip('/')}"


def merge_query(params: dict[str, Any] | None) -> dict[str, Any]:
    """Drop ``None`` values so we don't send empty query parameters."""
    if not params:
        return {}
    return {k: v for k, v in params.items() if v is not None}


def encode_query(params: dict[str, Any] | None) -> str:
    """Encode a query dict, ignoring ``None``s. Useful for SSE URLs."""
    cleaned = merge_query(params)
    if not cleaned:
        return ""
    return urlencode({k: str(v) for k, v in cleaned.items()})


def raise_for_status(response: httpx.Response) -> None:
    """Translate non-2xx responses into the SDK's exception hierarchy."""
    if response.is_success:
        return

    try:
        body: Any = response.json()
        message = body.get("error") if isinstance(body, dict) else str(body)
    except Exception:
        body = response.text
        message = response.text or response.reason_phrase

    exc_cls = status_to_exception(response.status_code)
    raise exc_cls(
        message or f"HTTP {response.status_code}",
        status_code=response.status_code,
        body=body,
    )


def parse_json(response: httpx.Response) -> Any:
    """Decode the response body as JSON or raise a clean SDK error."""
    if response.status_code == 204 or not response.content:
        return None
    try:
        return response.json()
    except ValueError as e:
        raise OptimalEngineError(
            f"Engine returned non-JSON body: {response.text[:200]!r}",
            status_code=response.status_code,
            body=response.text,
        ) from e


def translate_httpx_error(exc: httpx.HTTPError) -> OptimalEngineError:
    """Convert an httpx transport error into the SDK's exception hierarchy."""
    if isinstance(exc, httpx.TimeoutException):
        return APITimeoutError(f"Request timed out: {exc}")
    return APIConnectionError(f"Could not reach engine: {exc}")


__all__ = [
    "DEFAULT_HEADERS",
    "DEFAULT_TIMEOUT",
    "build_url",
    "encode_query",
    "merge_query",
    "parse_json",
    "raise_for_status",
    "translate_httpx_error",
]
