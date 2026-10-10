"""Asynchronous Optimal Engine client.

Mirror of :mod:`optimal_engine._client` but using :class:`httpx.AsyncClient`.
Every resource handler exposes an async-prefixed method so callers can
``await`` without leaving the asyncio event loop.
"""

from __future__ import annotations

import json
from types import TracebackType
from typing import Any, AsyncIterator, Mapping

import httpx
from typing_extensions import Self

from ._transport import (
    DEFAULT_HEADERS,
    DEFAULT_TIMEOUT,
    build_url,
    encode_query,
    merge_query,
    parse_json,
    raise_for_status,
    translate_httpx_error,
)
from .memory import AsyncMemoryResource
from .profile import AsyncWikiResource
from .retrieval import AsyncRecallResource, AsyncRetrievalMixin
from .surface import AsyncSubscriptionsResource, AsyncSurfaceResource
from .types import SurfaceEvent
from .workspace import AsyncWorkspaceResource


class AsyncOptimalEngine(AsyncRetrievalMixin):
    """Asynchronous Python client for the Optimal Engine.

    Usage::

        from optimal_engine import AsyncOptimalEngine

        async with AsyncOptimalEngine(base_url="http://localhost:4200") as client:
            result = await client.ask("pricing decisions", audience="sales")
    """

    def __init__(
        self,
        *,
        base_url: str = "http://localhost:4200",
        workspace: str = "default",
        tenant: str = "default",
        timeout: float | httpx.Timeout | None = None,
        headers: Mapping[str, str] | None = None,
        http_client: httpx.AsyncClient | None = None,
    ) -> None:
        """Create a new async client. See :class:`OptimalEngine` for params."""
        self._base_url = base_url
        self.workspace = workspace
        self.tenant = tenant

        merged_headers = dict(DEFAULT_HEADERS)
        if headers:
            merged_headers.update(headers)

        if http_client is not None:
            self._http = http_client
            self._owns_http = False
        else:
            self._http = httpx.AsyncClient(
                timeout=timeout or DEFAULT_TIMEOUT,
                headers=merged_headers,
            )
            self._owns_http = True

        self.memory = AsyncMemoryResource(self)
        self.workspaces = AsyncWorkspaceResource(self)
        self.recall = AsyncRecallResource(self)
        self.wiki = AsyncWikiResource(self)
        self.subscriptions = AsyncSubscriptionsResource(self)
        self.surface = AsyncSurfaceResource(self)

    # ---------------------------------------------------------------------
    # Lifecycle
    # ---------------------------------------------------------------------

    async def close(self) -> None:
        """Close the underlying async HTTP client."""
        if self._owns_http:
            await self._http.aclose()

    async def __aenter__(self) -> Self:
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        await self.close()

    # ---------------------------------------------------------------------
    # HTTP verbs
    # ---------------------------------------------------------------------

    @property
    def base_url(self) -> str:
        """The engine base URL this client targets."""
        return self._base_url

    async def _request(
        self,
        method: str,
        path: str,
        *,
        params: dict[str, Any] | None = None,
        json_body: Any | None = None,
    ) -> Any:
        """Execute an HTTP request, raise on non-2xx, return decoded JSON."""
        url = build_url(self._base_url, path)
        try:
            response = await self._http.request(
                method,
                url,
                params=merge_query(params),
                json=json_body,
            )
        except httpx.HTTPError as e:
            raise translate_httpx_error(e) from e

        raise_for_status(response)
        return parse_json(response)

    async def get(self, path: str, *, params: dict[str, Any] | None = None) -> Any:
        """Send a GET and return the decoded JSON body."""
        return await self._request("GET", path, params=params)

    async def post(self, path: str, *, json_body: Any | None = None) -> Any:
        """Send a POST and return the decoded JSON body."""
        return await self._request("POST", path, json_body=json_body)

    async def patch(self, path: str, *, json_body: Any | None = None) -> Any:
        """Send a PATCH and return the decoded JSON body."""
        return await self._request("PATCH", path, json_body=json_body)

    async def delete(self, path: str) -> Any:
        """Send a DELETE and return the decoded JSON body (or ``None``)."""
        return await self._request("DELETE", path)

    # ---------------------------------------------------------------------
    # Status
    # ---------------------------------------------------------------------

    async def status(self) -> Any:
        """Return engine liveness + readiness from ``GET /api/status``."""
        from .types import Status

        data = await self.get("/api/status")
        return Status.model_validate(data)

    # ---------------------------------------------------------------------
    # SSE
    # ---------------------------------------------------------------------

    async def _stream_sse(
        self,
        path: str,
        *,
        params: dict[str, Any] | None = None,
    ) -> AsyncIterator[SurfaceEvent]:
        """Async iterator over an SSE stream — yields :class:`SurfaceEvent`."""
        qs = encode_query(params)
        url = build_url(self._base_url, path) + (f"?{qs}" if qs else "")

        try:
            async with self._http.stream("GET", url) as response:
                raise_for_status(response)
                event_name = "message"
                data_buf: list[str] = []

                async for raw_line in response.aiter_lines():
                    line = raw_line.rstrip("\r")

                    if line == "":
                        if data_buf:
                            payload = _decode_sse_data("\n".join(data_buf))
                            yield SurfaceEvent(event=event_name, data=payload)
                        event_name = "message"
                        data_buf = []
                        continue

                    if line.startswith(":"):
                        continue

                    if line.startswith("event:"):
                        event_name = line[len("event:"):].strip()
                    elif line.startswith("data:"):
                        data_buf.append(line[len("data:"):].lstrip())
        except httpx.HTTPError as e:
            raise translate_httpx_error(e) from e


def _decode_sse_data(blob: str) -> dict[str, Any]:
    """Best-effort JSON decode for an SSE data payload."""
    if not blob:
        return {}
    try:
        decoded = json.loads(blob)
    except json.JSONDecodeError:
        return {"raw": blob}
    if isinstance(decoded, dict):
        return decoded
    return {"value": decoded}


__all__ = ["AsyncOptimalEngine"]
