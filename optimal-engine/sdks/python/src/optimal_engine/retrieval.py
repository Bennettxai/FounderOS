"""Retrieval-side resources — ask, search, grep, profile, and recall.

The four retrieval verbs (``ask`` / ``search`` / ``grep`` / ``profile``)
live as mixins on the client itself for ergonomic access — you write
``client.ask(...)``, not ``client.retrieval.ask(...)``. The five recall
verbs are grouped under ``client.recall`` because they share a common
question shape (Engramme-style cued recall).
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from .types import (
    AskResult,
    Bandwidth,
    GrepResponse,
    Profile,
    ResponseFormat,
    Scale,
    SearchResponse,
)

if TYPE_CHECKING:
    from ._async_client import AsyncOptimalEngine
    from ._client import OptimalEngine


# ---------------------------------------------------------------------------
# Sync — mixed into OptimalEngine
# ---------------------------------------------------------------------------


class RetrievalMixin:
    """Adds ``ask`` / ``search`` / ``grep`` / ``profile`` to the sync client."""

    # Concrete methods are defined on OptimalEngine but we need the type checker
    # to know about them — see the conditional imports above.
    workspace: str
    tenant: str

    def get(self, path: str, *, params: dict[str, Any] | None = None) -> Any: ...
    def post(self, path: str, *, json_body: Any | None = None) -> Any: ...

    def ask(
        self,
        query: str,
        *,
        workspace: str | None = None,
        audience: str = "default",
        format: ResponseFormat = "markdown",
        bandwidth: Bandwidth = "medium",
    ) -> AskResult:
        """Ask the second brain. Wiki-first, hybrid-fallback, hot-cited envelope.

        Sends ``POST /api/rag``. Returns the rendered answer plus the
        citations the engine used to ground each claim.
        """
        body = {
            "query": query,
            "workspace": workspace or self.workspace,
            "audience": audience,
            "format": format,
            "bandwidth": bandwidth,
        }
        data = self.post("/api/rag", json_body=body)
        return AskResult.model_validate(data)

    def search(
        self,
        query: str,
        *,
        workspace: str | None = None,
        limit: int = 10,
    ) -> SearchResponse:
        """Full-text search across context-level metadata.

        Returns one row per matching context (title / node / genre / sn_ratio).
        Use :meth:`grep` for chunk-level matches with the full signal trace.
        """
        params = {
            "q": query,
            "workspace": workspace or self.workspace,
            "limit": limit,
        }
        data = self.get("/api/search", params=params)
        return SearchResponse.model_validate(data)

    def grep(
        self,
        query: str,
        *,
        workspace: str | None = None,
        intent: str | None = None,
        scale: Scale | None = None,
        modality: str | None = None,
        limit: int = 25,
        literal: bool = False,
        path: str | None = None,
    ) -> GrepResponse:
        """Hybrid semantic + literal grep with the full signal trace.

        Each match carries slug, scale, intent, S/N ratio, modality, and
        a 200-char snippet — enough to decide whether to drill further.
        """
        params: dict[str, Any] = {
            "q": query,
            "workspace": workspace or self.workspace,
            "intent": intent,
            "scale": scale,
            "modality": modality,
            "limit": limit,
            "literal": "true" if literal else "false",
            "path": path,
        }
        data = self.get("/api/grep", params=params)
        return GrepResponse.model_validate(data)

    def profile(
        self,
        *,
        workspace: str | None = None,
        audience: str = "default",
        bandwidth: Bandwidth = "l1",
        node: str | None = None,
        tenant: str | None = None,
    ) -> Profile:
        """Four-tier workspace snapshot — static / dynamic / curated / activity."""
        params = {
            "workspace": workspace or self.workspace,
            "audience": audience,
            "bandwidth": bandwidth,
            "node": node,
            "tenant": tenant or self.tenant,
        }
        data = self.get("/api/profile", params=params)
        return Profile.model_validate(data)


class RecallResource:
    """Cued recall verbs — actions / who / when / where / owns.

    Each verb is a typed shortcut over ``POST /api/rag`` with a question
    shape that lets the engine's IntentAnalyzer decode intent at maximum
    confidence. Shape matches recognized enterprise memory-failure patterns.
    """

    def __init__(self, client: OptimalEngine) -> None:
        self._client = client

    def actions(
        self,
        *,
        actor: str | None = None,
        topic: str | None = None,
        since: str | None = None,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Past actions / decisions / commitments matching the cue."""
        params = {
            "actor": actor,
            "topic": topic,
            "since": since,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = self._client.get("/api/recall/actions", params=params)
        return AskResult.model_validate(data)

    def who(
        self,
        *,
        topic: str,
        role: str = "owner",
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Contact / ownership lookup — 'who owns X'."""
        params = {
            "topic": topic,
            "role": role,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = self._client.get("/api/recall/who", params=params)
        return AskResult.model_validate(data)

    def when(
        self,
        *,
        event: str,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Schedule / temporal lookup — 'when does X happen'."""
        params = {
            "event": event,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = self._client.get("/api/recall/when", params=params)
        return AskResult.model_validate(data)

    def where(
        self,
        *,
        thing: str,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Object-location lookup — 'where is X kept / discussed'."""
        params = {
            "thing": thing,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = self._client.get("/api/recall/where", params=params)
        return AskResult.model_validate(data)

    def owns(
        self,
        *,
        actor: str,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Open-task lookup — 'what is X currently committed to'."""
        params = {
            "actor": actor,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = self._client.get("/api/recall/owns", params=params)
        return AskResult.model_validate(data)


# ---------------------------------------------------------------------------
# Async — mixed into AsyncOptimalEngine
# ---------------------------------------------------------------------------


class AsyncRetrievalMixin:
    """Async equivalents of :class:`RetrievalMixin`."""

    workspace: str
    tenant: str

    async def get(self, path: str, *, params: dict[str, Any] | None = None) -> Any: ...
    async def post(self, path: str, *, json_body: Any | None = None) -> Any: ...

    async def ask(
        self,
        query: str,
        *,
        workspace: str | None = None,
        audience: str = "default",
        format: ResponseFormat = "markdown",
        bandwidth: Bandwidth = "medium",
    ) -> AskResult:
        """Ask the second brain. Wiki-first, hybrid-fallback, hot-cited envelope."""
        body = {
            "query": query,
            "workspace": workspace or self.workspace,
            "audience": audience,
            "format": format,
            "bandwidth": bandwidth,
        }
        data = await self.post("/api/rag", json_body=body)
        return AskResult.model_validate(data)

    async def search(
        self,
        query: str,
        *,
        workspace: str | None = None,
        limit: int = 10,
    ) -> SearchResponse:
        """Full-text search across context-level metadata."""
        params = {
            "q": query,
            "workspace": workspace or self.workspace,
            "limit": limit,
        }
        data = await self.get("/api/search", params=params)
        return SearchResponse.model_validate(data)

    async def grep(
        self,
        query: str,
        *,
        workspace: str | None = None,
        intent: str | None = None,
        scale: Scale | None = None,
        modality: str | None = None,
        limit: int = 25,
        literal: bool = False,
        path: str | None = None,
    ) -> GrepResponse:
        """Hybrid semantic + literal grep with the full signal trace."""
        params: dict[str, Any] = {
            "q": query,
            "workspace": workspace or self.workspace,
            "intent": intent,
            "scale": scale,
            "modality": modality,
            "limit": limit,
            "literal": "true" if literal else "false",
            "path": path,
        }
        data = await self.get("/api/grep", params=params)
        return GrepResponse.model_validate(data)

    async def profile(
        self,
        *,
        workspace: str | None = None,
        audience: str = "default",
        bandwidth: Bandwidth = "l1",
        node: str | None = None,
        tenant: str | None = None,
    ) -> Profile:
        """Four-tier workspace snapshot — static / dynamic / curated / activity."""
        params = {
            "workspace": workspace or self.workspace,
            "audience": audience,
            "bandwidth": bandwidth,
            "node": node,
            "tenant": tenant or self.tenant,
        }
        data = await self.get("/api/profile", params=params)
        return Profile.model_validate(data)


class AsyncRecallResource:
    """Async counterpart of :class:`RecallResource`."""

    def __init__(self, client: AsyncOptimalEngine) -> None:
        self._client = client

    async def actions(
        self,
        *,
        actor: str | None = None,
        topic: str | None = None,
        since: str | None = None,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Past actions / decisions / commitments matching the cue."""
        params = {
            "actor": actor,
            "topic": topic,
            "since": since,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = await self._client.get("/api/recall/actions", params=params)
        return AskResult.model_validate(data)

    async def who(
        self,
        *,
        topic: str,
        role: str = "owner",
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Contact / ownership lookup — 'who owns X'."""
        params = {
            "topic": topic,
            "role": role,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = await self._client.get("/api/recall/who", params=params)
        return AskResult.model_validate(data)

    async def when(
        self,
        *,
        event: str,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Schedule / temporal lookup — 'when does X happen'."""
        params = {
            "event": event,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = await self._client.get("/api/recall/when", params=params)
        return AskResult.model_validate(data)

    async def where(
        self,
        *,
        thing: str,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Object-location lookup — 'where is X kept / discussed'."""
        params = {
            "thing": thing,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = await self._client.get("/api/recall/where", params=params)
        return AskResult.model_validate(data)

    async def owns(
        self,
        *,
        actor: str,
        workspace: str | None = None,
        audience: str = "default",
    ) -> AskResult:
        """Open-task lookup — 'what is X currently committed to'."""
        params = {
            "actor": actor,
            "workspace": workspace or self._client.workspace,
            "audience": audience,
        }
        data = await self._client.get("/api/recall/owns", params=params)
        return AskResult.model_validate(data)


__all__ = [
    "AsyncRecallResource",
    "AsyncRetrievalMixin",
    "RecallResource",
    "RetrievalMixin",
]
