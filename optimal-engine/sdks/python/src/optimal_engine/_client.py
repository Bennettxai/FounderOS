"""Synchronous Optimal Engine client.

The :class:`OptimalEngine` class is the entry point for the blocking SDK.
Resource handlers (``memory``, ``workspaces``, ``recall``, ``wiki``,
``subscriptions``, ``surface``) are attached as attributes — the client
itself only owns the HTTP transport and the four verbs (GET / POST /
PATCH / DELETE) that resources call into.
"""

from __future__ import annotations

import json
from types import TracebackType
from typing import Any, Iterator, Mapping

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
from .memory import MemoryResource
from .profile import WikiResource
from .retrieval import RecallResource, RetrievalMixin
from .surface import SubscriptionsResource, SurfaceResource
from .types import SurfaceEvent
from .workspace import WorkspaceResource


class OptimalEngine(RetrievalMixin):
    """Synchronous Python client for the Optimal Engine.

    Usage::

        from optimal_engine import OptimalEngine

        client = OptimalEngine(base_url="http://localhost:4200", workspace="default")
        result = client.ask("pricing decisions", audience="sales")
        client.close()

    Or as a context manager::

        with OptimalEngine(base_url="http://localhost:4200") as client:
            ...
    """

    def __init__(
        self,
        *,
        base_url: str = "http://localhost:4200",
        workspace: str = "default",
        tenant: str = "default",
        timeout: float | httpx.Timeout | None = None,
        headers: Mapping[str, str] | None = None,
        http_client: httpx.Client | None = None,
    ) -> None:
        """Create a new client.

        Args:
            base_url: The engine HTTP base URL (no trailing /api).
            workspace: Default workspace id used when a call doesn't pass one.
            tenant: Default tenant id (single-tenant deployments use ``"default"``).
            timeout: Per-request timeout; accepts a number of seconds or
                an :class:`httpx.Timeout`. Defaults to 60s read / 10s connect.
            headers: Extra headers merged on every request.
            http_client: Bring-your-own :class:`httpx.Client`. The SDK will
                NOT close it for you in that case.
        """
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
            self._http = httpx.Client(
                timeout=timeout or DEFAULT_TIMEOUT,
                headers=merged_headers,
            )
            self._owns_http = True

        # Resource handlers — composition, not inheritance.
        self.memory = MemoryResource(self)
        self.workspaces = WorkspaceResource(self)
        self.recall = RecallResource(self)
        self.wiki = WikiResource(self)
        self.subscriptions = SubscriptionsResource(self)
        self.surface = SurfaceResource(self)

    # ---------------------------------------------------------------------
    # Lifecycle
    # ---------------------------------------------------------------------

    def close(self) -> None:
        """Close the underlying HTTP client (no-op for borrowed clients)."""
        if self._owns_http:
            self._http.close()

    def __enter__(self) -> Self:
        return self

    def __exit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        self.close()

    # ---------------------------------------------------------------------
    # HTTP verbs (used by resource handlers)
    # ---------------------------------------------------------------------

    @property
    def base_url(self) -> str:
        """The engine base URL this client targets."""
        return self._base_url

    def _request(
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
            response = self._http.request(
                method,
                url,
                params=merge_query(params),
                json=json_body,
            )
        except httpx.HTTPError as e:
            raise translate_httpx_error(e) from e

        raise_for_status(response)
        return parse_json(response)

    def get(self, path: str, *, params: dict[str, Any] | None = None) -> Any:
        """Send a GET and return the decoded JSON body."""
        return self._request("GET", path, params=params)

    def post(self, path: str, *, json_body: Any | None = None) -> Any:
        """Send a POST and return the decoded JSON body."""
        return self._request("POST", path, json_body=json_body)

    def patch(self, path: str, *, json_body: Any | None = None) -> Any:
        """Send a PATCH and return the decoded JSON body."""
        return self._request("PATCH", path, json_body=json_body)

    def delete(self, path: str) -> Any:
        """Send a DELETE and return the decoded JSON body (or ``None``)."""
        return self._request("DELETE", path)

    # ---------------------------------------------------------------------
    # Status
    # ---------------------------------------------------------------------

    def status(self) -> Any:
        """Return engine liveness + readiness from ``GET /api/status``."""
        from .types import Status

        data = self.get("/api/status")
        return Status.model_validate(data)

    # ---------------------------------------------------------------------
    # SSE — used by SurfaceResource.stream
    # ---------------------------------------------------------------------

    def _stream_sse(
        self,
        path: str,
        *,
        params: dict[str, Any] | None = None,
    ) -> Iterator[SurfaceEvent]:
        """Iterate over a Server-Sent Events stream as :class:`SurfaceEvent`.

        The engine emits standard SSE — `event:` line, `data:` line, blank
        separator. We yield one ``SurfaceEvent`` per complete record and
        ignore comment lines (e.g. the ``: keepalive`` heartbeat).
        """
        qs = encode_query(params)
        url = build_url(self._base_url, path) + (f"?{qs}" if qs else "")

        try:
            with self._http.stream("GET", url) as response:
                raise_for_status(response)
                yield from _iter_sse(response.iter_lines())
        except httpx.HTTPError as e:
            raise translate_httpx_error(e) from e


def _iter_sse(lines: Iterator[str]) -> Iterator[SurfaceEvent]:
    """Decode an SSE byte/line stream into structured events.

    Implements the minimal subset of the SSE grammar the engine emits:

    * ``event: <name>``      sets the next event's name
    * ``data: <json>``       single-line JSON payload
    * blank line             dispatch the buffered event
    * ``:`` comment lines    ignored (keepalives)
    """
    event_name = "message"
    data_buf: list[str] = []

    for raw_line in lines:
        line = raw_line.rstrip("\r")

        if line == "":
            if data_buf:
                payload = _decode_sse_data("\n".join(data_buf))
                yield SurfaceEvent(event=event_name, data=payload)
            event_name = "message"
            data_buf = []
            continue

        if line.startswith(":"):
            # SSE comment — keepalive heartbeat, skip.
            continue

        if line.startswith("event:"):
            event_name = line[len("event:"):].strip()
        elif line.startswith("data:"):
            data_buf.append(line[len("data:"):].lstrip())


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


__all__ = ["OptimalEngine"]
