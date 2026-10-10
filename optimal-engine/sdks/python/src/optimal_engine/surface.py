"""Subscriptions and surfacing — proactive push events.

Subscriptions tell the engine *what* to surface; the SSE stream tells you
*when* it pushes. Two resources here:

* :class:`SubscriptionsResource` — CRUD over ``/api/subscriptions``
* :class:`SurfaceResource` — connects to ``/api/surface/stream`` and
  yields decoded :class:`SurfaceEvent` objects from a generator.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, AsyncIterator, Iterator, List

from .types import Subscription, SubscriptionList, SubscriptionScope, SurfaceEvent

if TYPE_CHECKING:
    from ._async_client import AsyncOptimalEngine
    from ._client import OptimalEngine


# ---------------------------------------------------------------------------
# Subscriptions — sync
# ---------------------------------------------------------------------------


class SubscriptionsResource:
    """Sync handler for ``/api/subscriptions``."""

    def __init__(self, client: OptimalEngine) -> None:
        self._client = client

    def list(self, *, workspace: str | None = None) -> SubscriptionList:
        """List subscriptions for a workspace."""
        params = {"workspace": workspace or self._client.workspace}
        data = self._client.get("/api/subscriptions", params=params)
        return SubscriptionList.model_validate(data)

    def create(
        self,
        *,
        scope: SubscriptionScope = "workspace",
        scope_value: str | None = None,
        categories: List[str] | None = None,
        principal_id: str | None = None,
        activity: str | None = None,
        workspace: str | None = None,
    ) -> Subscription:
        """Create a surfacing subscription. Engine will push events whenever it matches."""
        body: dict[str, object] = {
            "workspace": workspace or self._client.workspace,
            "scope": scope,
        }
        if scope_value is not None:
            body["scope_value"] = scope_value
        if categories is not None:
            body["categories"] = categories
        if principal_id is not None:
            body["principal_id"] = principal_id
        if activity is not None:
            body["activity"] = activity

        data = self._client.post("/api/subscriptions", json_body=body)
        return Subscription.model_validate(data)

    def pause(self, subscription_id: str) -> None:
        """Pause a subscription — no further events until resumed."""
        self._client.post(f"/api/subscriptions/{subscription_id}/pause")

    def resume(self, subscription_id: str) -> None:
        """Resume a previously paused subscription."""
        self._client.post(f"/api/subscriptions/{subscription_id}/resume")

    def delete(self, subscription_id: str) -> None:
        """Delete a subscription. No undo."""
        self._client.delete(f"/api/subscriptions/{subscription_id}")


# ---------------------------------------------------------------------------
# Surface stream — sync
# ---------------------------------------------------------------------------


class SurfaceResource:
    """Sync handler for ``/api/surface/stream`` and ``/api/surface/test``."""

    def __init__(self, client: OptimalEngine) -> None:
        self._client = client

    def stream(self, subscription_id: str) -> Iterator[SurfaceEvent]:
        """Iterate over server-sent events for a subscription.

        The generator yields :class:`SurfaceEvent` objects until the engine
        closes the stream or the caller breaks out of the loop. Keepalive
        comments are filtered out — only real events are surfaced.
        """
        return self._client._stream_sse(
            "/api/surface/stream",
            params={"subscription": subscription_id},
        )

    def test(self, *, subscription_id: str, slug: str) -> None:
        """Trigger a synthetic surface push to all listeners of a subscription."""
        body = {"subscription": subscription_id, "slug": slug}
        self._client.post("/api/surface/test", json_body=body)


# ---------------------------------------------------------------------------
# Subscriptions — async
# ---------------------------------------------------------------------------


class AsyncSubscriptionsResource:
    """Async counterpart of :class:`SubscriptionsResource`."""

    def __init__(self, client: AsyncOptimalEngine) -> None:
        self._client = client

    async def list(self, *, workspace: str | None = None) -> SubscriptionList:
        """List subscriptions for a workspace."""
        params = {"workspace": workspace or self._client.workspace}
        data = await self._client.get("/api/subscriptions", params=params)
        return SubscriptionList.model_validate(data)

    async def create(
        self,
        *,
        scope: SubscriptionScope = "workspace",
        scope_value: str | None = None,
        categories: List[str] | None = None,
        principal_id: str | None = None,
        activity: str | None = None,
        workspace: str | None = None,
    ) -> Subscription:
        """Create a surfacing subscription. Engine will push events whenever it matches."""
        body: dict[str, object] = {
            "workspace": workspace or self._client.workspace,
            "scope": scope,
        }
        if scope_value is not None:
            body["scope_value"] = scope_value
        if categories is not None:
            body["categories"] = categories
        if principal_id is not None:
            body["principal_id"] = principal_id
        if activity is not None:
            body["activity"] = activity

        data = await self._client.post("/api/subscriptions", json_body=body)
        return Subscription.model_validate(data)

    async def pause(self, subscription_id: str) -> None:
        """Pause a subscription — no further events until resumed."""
        await self._client.post(f"/api/subscriptions/{subscription_id}/pause")

    async def resume(self, subscription_id: str) -> None:
        """Resume a previously paused subscription."""
        await self._client.post(f"/api/subscriptions/{subscription_id}/resume")

    async def delete(self, subscription_id: str) -> None:
        """Delete a subscription. No undo."""
        await self._client.delete(f"/api/subscriptions/{subscription_id}")


# ---------------------------------------------------------------------------
# Surface stream — async
# ---------------------------------------------------------------------------


class AsyncSurfaceResource:
    """Async counterpart of :class:`SurfaceResource`."""

    def __init__(self, client: AsyncOptimalEngine) -> None:
        self._client = client

    def stream(self, subscription_id: str) -> AsyncIterator[SurfaceEvent]:
        """Async iterator over server-sent events for a subscription."""
        return self._client._stream_sse(
            "/api/surface/stream",
            params={"subscription": subscription_id},
        )

    async def test(self, *, subscription_id: str, slug: str) -> None:
        """Trigger a synthetic surface push to all listeners of a subscription."""
        body = {"subscription": subscription_id, "slug": slug}
        await self._client.post("/api/surface/test", json_body=body)


__all__ = [
    "AsyncSubscriptionsResource",
    "AsyncSurfaceResource",
    "SubscriptionsResource",
    "SurfaceResource",
]
