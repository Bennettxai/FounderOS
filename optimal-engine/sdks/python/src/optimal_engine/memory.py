"""Memory primitive — create, read, list, version, derive, forget.

The memory API is the engine's integrity-gated long-term store. Every
mutation is audited; every "delete" is a soft-forget (the audit trail
is preserved). Hard delete via :meth:`MemoryResource.delete` is reserved
for compliance flows (GDPR, etc.).
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from .types import Memory, MemoryList, MemoryRelations, MemoryVersions

if TYPE_CHECKING:
    from ._async_client import AsyncOptimalEngine
    from ._client import OptimalEngine


# ---------------------------------------------------------------------------
# Sync
# ---------------------------------------------------------------------------


class MemoryResource:
    """Sync handler for ``/api/memory`` and its sub-routes."""

    def __init__(self, client: OptimalEngine) -> None:
        self._client = client

    def create(
        self,
        *,
        content: str,
        workspace: str | None = None,
        is_static: bool | None = None,
        audience: str | None = None,
        citation_uri: str | None = None,
        source_chunk_id: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Add a cited fact to long-term memory. Integrity-gated — must include source URI."""
        body: dict[str, Any] = {
            "content": content,
            "workspace": workspace or self._client.workspace,
        }
        if is_static is not None:
            body["is_static"] = is_static
        if audience is not None:
            body["audience"] = audience
        if citation_uri is not None:
            body["citation_uri"] = citation_uri
        if source_chunk_id is not None:
            body["source_chunk_id"] = source_chunk_id
        if metadata is not None:
            body["metadata"] = metadata

        data = self._client.post("/api/memory", json_body=body)
        return Memory.model_validate(data)

    def get(self, memory_id: str) -> Memory:
        """Fetch one memory by id."""
        data = self._client.get(f"/api/memory/{memory_id}")
        return Memory.model_validate(data)

    def list(
        self,
        *,
        workspace: str | None = None,
        audience: str | None = None,
        include_forgotten: bool = False,
        include_old_versions: bool = False,
        limit: int = 50,
    ) -> MemoryList:
        """List memories for a workspace, with audience and version filters."""
        params: dict[str, Any] = {
            "workspace": workspace or self._client.workspace,
            "audience": audience,
            "include_forgotten": "true" if include_forgotten else "false",
            "include_old_versions": "true" if include_old_versions else "false",
            "limit": limit,
        }
        data = self._client.get("/api/memory", params=params)
        return MemoryList.model_validate(data)

    def forget(
        self,
        memory_id: str,
        *,
        reason: str | None = None,
        forget_after: str | None = None,
    ) -> None:
        """Soft-forget a memory; audit trail preserved."""
        body: dict[str, Any] = {}
        if reason is not None:
            body["reason"] = reason
        if forget_after is not None:
            body["forget_after"] = forget_after

        self._client.post(f"/api/memory/{memory_id}/forget", json_body=body)

    def update(
        self,
        memory_id: str,
        *,
        content: str,
        audience: str | None = None,
        citation_uri: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Create a new version of a memory — old version stays in the chain."""
        return self._mutate(memory_id, "update", content, audience, citation_uri, metadata)

    def extend(
        self,
        memory_id: str,
        *,
        content: str,
        audience: str | None = None,
        citation_uri: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Create a child memory linked back via an :extends relation."""
        return self._mutate(memory_id, "extend", content, audience, citation_uri, metadata)

    def derive(
        self,
        memory_id: str,
        *,
        content: str,
        audience: str | None = None,
        citation_uri: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Create a derived memory linked back via a :derives relation."""
        return self._mutate(memory_id, "derive", content, audience, citation_uri, metadata)

    def versions(self, memory_id: str) -> MemoryVersions:
        """Full version chain for a memory — chronological order."""
        data = self._client.get(f"/api/memory/{memory_id}/versions")
        return MemoryVersions.model_validate(data)

    def relations(self, memory_id: str) -> MemoryRelations:
        """Inbound + outbound relation graph for a memory."""
        data = self._client.get(f"/api/memory/{memory_id}/relations")
        return MemoryRelations.model_validate(data)

    def delete(self, memory_id: str) -> None:
        """Hard-delete a memory. Used for compliance flows only."""
        self._client.delete(f"/api/memory/{memory_id}")

    # ------------------------------------------------------------------

    def _mutate(
        self,
        memory_id: str,
        verb: str,
        content: str,
        audience: str | None,
        citation_uri: str | None,
        metadata: dict[str, Any] | None,
    ) -> Memory:
        body: dict[str, Any] = {"content": content}
        if audience is not None:
            body["audience"] = audience
        if citation_uri is not None:
            body["citation_uri"] = citation_uri
        if metadata is not None:
            body["metadata"] = metadata

        data = self._client.post(f"/api/memory/{memory_id}/{verb}", json_body=body)
        return Memory.model_validate(data)


# ---------------------------------------------------------------------------
# Async
# ---------------------------------------------------------------------------


class AsyncMemoryResource:
    """Async counterpart of :class:`MemoryResource`."""

    def __init__(self, client: AsyncOptimalEngine) -> None:
        self._client = client

    async def create(
        self,
        *,
        content: str,
        workspace: str | None = None,
        is_static: bool | None = None,
        audience: str | None = None,
        citation_uri: str | None = None,
        source_chunk_id: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Add a cited fact to long-term memory. Integrity-gated — must include source URI."""
        body: dict[str, Any] = {
            "content": content,
            "workspace": workspace or self._client.workspace,
        }
        if is_static is not None:
            body["is_static"] = is_static
        if audience is not None:
            body["audience"] = audience
        if citation_uri is not None:
            body["citation_uri"] = citation_uri
        if source_chunk_id is not None:
            body["source_chunk_id"] = source_chunk_id
        if metadata is not None:
            body["metadata"] = metadata

        data = await self._client.post("/api/memory", json_body=body)
        return Memory.model_validate(data)

    async def get(self, memory_id: str) -> Memory:
        """Fetch one memory by id."""
        data = await self._client.get(f"/api/memory/{memory_id}")
        return Memory.model_validate(data)

    async def list(
        self,
        *,
        workspace: str | None = None,
        audience: str | None = None,
        include_forgotten: bool = False,
        include_old_versions: bool = False,
        limit: int = 50,
    ) -> MemoryList:
        """List memories for a workspace, with audience and version filters."""
        params: dict[str, Any] = {
            "workspace": workspace or self._client.workspace,
            "audience": audience,
            "include_forgotten": "true" if include_forgotten else "false",
            "include_old_versions": "true" if include_old_versions else "false",
            "limit": limit,
        }
        data = await self._client.get("/api/memory", params=params)
        return MemoryList.model_validate(data)

    async def forget(
        self,
        memory_id: str,
        *,
        reason: str | None = None,
        forget_after: str | None = None,
    ) -> None:
        """Soft-forget a memory; audit trail preserved."""
        body: dict[str, Any] = {}
        if reason is not None:
            body["reason"] = reason
        if forget_after is not None:
            body["forget_after"] = forget_after

        await self._client.post(f"/api/memory/{memory_id}/forget", json_body=body)

    async def update(
        self,
        memory_id: str,
        *,
        content: str,
        audience: str | None = None,
        citation_uri: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Create a new version of a memory — old version stays in the chain."""
        return await self._mutate(memory_id, "update", content, audience, citation_uri, metadata)

    async def extend(
        self,
        memory_id: str,
        *,
        content: str,
        audience: str | None = None,
        citation_uri: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Create a child memory linked back via an :extends relation."""
        return await self._mutate(memory_id, "extend", content, audience, citation_uri, metadata)

    async def derive(
        self,
        memory_id: str,
        *,
        content: str,
        audience: str | None = None,
        citation_uri: str | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> Memory:
        """Create a derived memory linked back via a :derives relation."""
        return await self._mutate(memory_id, "derive", content, audience, citation_uri, metadata)

    async def versions(self, memory_id: str) -> MemoryVersions:
        """Full version chain for a memory — chronological order."""
        data = await self._client.get(f"/api/memory/{memory_id}/versions")
        return MemoryVersions.model_validate(data)

    async def relations(self, memory_id: str) -> MemoryRelations:
        """Inbound + outbound relation graph for a memory."""
        data = await self._client.get(f"/api/memory/{memory_id}/relations")
        return MemoryRelations.model_validate(data)

    async def delete(self, memory_id: str) -> None:
        """Hard-delete a memory. Used for compliance flows only."""
        await self._client.delete(f"/api/memory/{memory_id}")

    # ------------------------------------------------------------------

    async def _mutate(
        self,
        memory_id: str,
        verb: str,
        content: str,
        audience: str | None,
        citation_uri: str | None,
        metadata: dict[str, Any] | None,
    ) -> Memory:
        body: dict[str, Any] = {"content": content}
        if audience is not None:
            body["audience"] = audience
        if citation_uri is not None:
            body["citation_uri"] = citation_uri
        if metadata is not None:
            body["metadata"] = metadata

        data = await self._client.post(f"/api/memory/{memory_id}/{verb}", json_body=body)
        return Memory.model_validate(data)


__all__ = ["AsyncMemoryResource", "MemoryResource"]
