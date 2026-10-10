"""Wiki resource — list pages, render one, surface contradictions.

The wiki is the engine's curated, audience-tagged surface — the part of
the brain meant to be read directly. Profile (the four-tier snapshot)
lives on the client as :meth:`OptimalEngine.profile` because it doesn't
need a resource handler.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

from .types import ContradictionList, ResponseFormat, WikiList, WikiPage

if TYPE_CHECKING:
    from ._async_client import AsyncOptimalEngine
    from ._client import OptimalEngine


# ---------------------------------------------------------------------------
# Sync
# ---------------------------------------------------------------------------


class WikiResource:
    """Sync handler for ``/api/wiki``."""

    def __init__(self, client: OptimalEngine) -> None:
        self._client = client

    def list(
        self,
        *,
        tenant: str | None = None,
        workspace: str | None = None,
    ) -> WikiList:
        """List every wiki page in a workspace."""
        params = {
            "tenant": tenant or self._client.tenant,
            "workspace": workspace or self._client.workspace,
        }
        data = self._client.get("/api/wiki", params=params)
        return WikiList.model_validate(data)

    def get(
        self,
        slug: str,
        *,
        audience: str = "default",
        format: ResponseFormat = "markdown",
        workspace: str | None = None,
        tenant: str | None = None,
    ) -> WikiPage:
        """Render a wiki page for a specific audience."""
        params = {
            "tenant": tenant or self._client.tenant,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
            "format": format,
        }
        data = self._client.get(f"/api/wiki/{slug}", params=params)
        return WikiPage.model_validate(data)

    def contradictions(self, *, workspace: str | None = None) -> ContradictionList:
        """Active contradiction surfacing events from the last 90 days."""
        params = {"workspace": workspace or self._client.workspace}
        data = self._client.get("/api/wiki/contradictions", params=params)
        return ContradictionList.model_validate(data)


# ---------------------------------------------------------------------------
# Async
# ---------------------------------------------------------------------------


class AsyncWikiResource:
    """Async counterpart of :class:`WikiResource`."""

    def __init__(self, client: AsyncOptimalEngine) -> None:
        self._client = client

    async def list(
        self,
        *,
        tenant: str | None = None,
        workspace: str | None = None,
    ) -> WikiList:
        """List every wiki page in a workspace."""
        params = {
            "tenant": tenant or self._client.tenant,
            "workspace": workspace or self._client.workspace,
        }
        data = await self._client.get("/api/wiki", params=params)
        return WikiList.model_validate(data)

    async def get(
        self,
        slug: str,
        *,
        audience: str = "default",
        format: ResponseFormat = "markdown",
        workspace: str | None = None,
        tenant: str | None = None,
    ) -> WikiPage:
        """Render a wiki page for a specific audience."""
        params = {
            "tenant": tenant or self._client.tenant,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
            "format": format,
        }
        data = await self._client.get(f"/api/wiki/{slug}", params=params)
        return WikiPage.model_validate(data)

    async def contradictions(self, *, workspace: str | None = None) -> ContradictionList:
        """Active contradiction surfacing events from the last 90 days."""
        params = {"workspace": workspace or self._client.workspace}
        data = await self._client.get("/api/wiki/contradictions", params=params)
        return ContradictionList.model_validate(data)


__all__ = ["AsyncWikiResource", "WikiResource"]
