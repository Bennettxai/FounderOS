"""Workspace resource — list, create, fetch, update, archive, configure."""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from .types import Workspace, WorkspaceConfig, WorkspaceList, WorkspaceStatus

if TYPE_CHECKING:
    from ._async_client import AsyncOptimalEngine
    from ._client import OptimalEngine


# ---------------------------------------------------------------------------
# Sync
# ---------------------------------------------------------------------------


class WorkspaceResource:
    """Sync handler for ``/api/workspaces``."""

    def __init__(self, client: OptimalEngine) -> None:
        self._client = client

    def list(
        self,
        *,
        tenant: str | None = None,
        status: WorkspaceStatus | str = "active",
    ) -> WorkspaceList:
        """List workspaces under a tenant; defaults to active workspaces."""
        params = {
            "tenant": tenant or self._client.tenant,
            "status": status,
        }
        data = self._client.get("/api/workspaces", params=params)
        return WorkspaceList.model_validate(data)

    def create(
        self,
        *,
        slug: str,
        name: str,
        description: str | None = None,
        tenant: str | None = None,
    ) -> Workspace:
        """Create a workspace under a tenant. Slug + name required."""
        body: dict[str, Any] = {
            "slug": slug,
            "name": name,
            "tenant": tenant or self._client.tenant,
        }
        if description is not None:
            body["description"] = description

        data = self._client.post("/api/workspaces", json_body=body)
        return Workspace.model_validate(data)

    def get(self, workspace_id: str) -> Workspace:
        """Fetch a single workspace by id."""
        data = self._client.get(f"/api/workspaces/{workspace_id}")
        return Workspace.model_validate(data)

    def update(
        self,
        workspace_id: str,
        *,
        name: str | None = None,
        description: str | None = None,
    ) -> Workspace:
        """Rename or re-describe a workspace."""
        body: dict[str, Any] = {}
        if name is not None:
            body["name"] = name
        if description is not None:
            body["description"] = description

        data = self._client.patch(f"/api/workspaces/{workspace_id}", json_body=body)
        return Workspace.model_validate(data)

    def archive(self, workspace_id: str) -> None:
        """Soft-delete a workspace. Memories stay queryable in :archived: status."""
        self._client.post(f"/api/workspaces/{workspace_id}/archive")

    def config(self, workspace_id: str) -> WorkspaceConfig:
        """Read the merged (defaults + on-disk) config for a workspace."""
        data = self._client.get(f"/api/workspaces/{workspace_id}/config")
        return WorkspaceConfig.model_validate(data)

    def update_config(
        self,
        workspace_id: str,
        config: dict[str, Any],
    ) -> WorkspaceConfig:
        """Deep-merge ``config`` into the workspace's on-disk YAML, returning the merged result."""
        data = self._client.patch(f"/api/workspaces/{workspace_id}/config", json_body=config)
        return WorkspaceConfig.model_validate(data)


# ---------------------------------------------------------------------------
# Async
# ---------------------------------------------------------------------------


class AsyncWorkspaceResource:
    """Async counterpart of :class:`WorkspaceResource`."""

    def __init__(self, client: AsyncOptimalEngine) -> None:
        self._client = client

    async def list(
        self,
        *,
        tenant: str | None = None,
        status: WorkspaceStatus | str = "active",
    ) -> WorkspaceList:
        """List workspaces under a tenant; defaults to active workspaces."""
        params = {
            "tenant": tenant or self._client.tenant,
            "status": status,
        }
        data = await self._client.get("/api/workspaces", params=params)
        return WorkspaceList.model_validate(data)

    async def create(
        self,
        *,
        slug: str,
        name: str,
        description: str | None = None,
        tenant: str | None = None,
    ) -> Workspace:
        """Create a workspace under a tenant. Slug + name required."""
        body: dict[str, Any] = {
            "slug": slug,
            "name": name,
            "tenant": tenant or self._client.tenant,
        }
        if description is not None:
            body["description"] = description

        data = await self._client.post("/api/workspaces", json_body=body)
        return Workspace.model_validate(data)

    async def get(self, workspace_id: str) -> Workspace:
        """Fetch a single workspace by id."""
        data = await self._client.get(f"/api/workspaces/{workspace_id}")
        return Workspace.model_validate(data)

    async def update(
        self,
        workspace_id: str,
        *,
        name: str | None = None,
        description: str | None = None,
    ) -> Workspace:
        """Rename or re-describe a workspace."""
        body: dict[str, Any] = {}
        if name is not None:
            body["name"] = name
        if description is not None:
            body["description"] = description

        data = await self._client.patch(f"/api/workspaces/{workspace_id}", json_body=body)
        return Workspace.model_validate(data)

    async def archive(self, workspace_id: str) -> None:
        """Soft-delete a workspace. Memories stay queryable in :archived: status."""
        await self._client.post(f"/api/workspaces/{workspace_id}/archive")

    async def config(self, workspace_id: str) -> WorkspaceConfig:
        """Read the merged (defaults + on-disk) config for a workspace."""
        data = await self._client.get(f"/api/workspaces/{workspace_id}/config")
        return WorkspaceConfig.model_validate(data)

    async def update_config(
        self,
        workspace_id: str,
        config: dict[str, Any],
    ) -> WorkspaceConfig:
        """Deep-merge ``config`` into the workspace's on-disk YAML, returning the merged result."""
        data = await self._client.patch(
            f"/api/workspaces/{workspace_id}/config", json_body=config
        )
        return WorkspaceConfig.model_validate(data)


__all__ = ["AsyncWorkspaceResource", "WorkspaceResource"]
