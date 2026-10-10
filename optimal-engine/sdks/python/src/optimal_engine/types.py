"""Pydantic models that mirror the engine's JSON payloads.

Every public response type is a Pydantic v2 model with `extra="allow"` so
new server-side fields don't break existing clients. Use ``.model_dump()``
to drop back to a plain dict.
"""

from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field

# ---------------------------------------------------------------------------
# Type aliases
# ---------------------------------------------------------------------------

Bandwidth = Literal["l0", "low", "medium", "high", "l1", "full"]
"""Receiver bandwidth tier — controls response density on /api/rag."""

ResponseFormat = Literal["markdown", "json", "plain"]
"""Format the engine renders ask/recall envelopes in."""

Scale = Literal["document", "section", "paragraph", "sentence", "chunk"]
"""Granularity of a chunk in the four-scale memory hierarchy."""

SubscriptionScope = Literal["workspace", "node", "topic", "actor"]
"""Surfacing subscription scope dimension."""

WorkspaceStatus = Literal["active", "archived"]


# ---------------------------------------------------------------------------
# Base
# ---------------------------------------------------------------------------


class _Base(BaseModel):
    """Base model — permissive by default so the engine can evolve safely."""

    model_config = ConfigDict(extra="allow", populate_by_name=True)


# ---------------------------------------------------------------------------
# Retrieval (ask / search / grep / profile / recall)
# ---------------------------------------------------------------------------


class Citation(_Base):
    """A single citation attached to an envelope or claim."""

    slug: str | None = None
    title: str | None = None
    uri: str | None = None
    chunk_id: str | None = None


class AskResult(_Base):
    """Response from ``client.ask`` / ``POST /api/rag``.

    The engine returns a hot-cited envelope: a rendered body plus the
    citations that ground each claim. Field shape is permissive — the
    engine ships extra trace fields (signal scores, cluster ids, etc.).
    """

    body: str | None = None
    answer: str | None = None
    format: ResponseFormat | None = None
    audience: str | None = None
    bandwidth: Bandwidth | None = None
    citations: list[Citation] = Field(default_factory=list)
    workspace_id: str | None = None
    recall_query: str | None = None


class SearchResult(_Base):
    """One row of ``GET /api/search`` — context-level metadata."""

    id: str | None = None
    title: str | None = None
    node: str | None = None
    genre: str | None = None
    uri: str | None = None
    l0_abstract: str | None = None
    sn_ratio: float | None = None


class SearchResponse(_Base):
    """Wrapper for ``GET /api/search``."""

    query: str
    results: list[SearchResult] = Field(default_factory=list)


class GrepMatch(_Base):
    """One chunk-level match from ``GET /api/grep``."""

    slug: str | None = None
    scale: str | None = None
    intent: str | None = None
    sn_ratio: float | None = None
    modality: str | None = None
    snippet: str | None = None
    score: float | None = None


class GrepResponse(_Base):
    """Wrapper for ``GET /api/grep``."""

    query: str
    workspace_id: str | None = None
    results: list[GrepMatch] = Field(default_factory=list)


class Profile(_Base):
    """Four-tier workspace snapshot from ``GET /api/profile``."""

    workspace_id: str | None = None
    tenant_id: str | None = None
    audience: str | None = None
    static: Any | None = None
    dynamic: Any | None = None
    curated: Any | None = None
    activity: Any | None = None
    entities: Any | None = None
    generated_at: str | None = None


# ---------------------------------------------------------------------------
# Workspaces
# ---------------------------------------------------------------------------


class Workspace(_Base):
    """A workspace (a 'brain') under a tenant."""

    id: str
    tenant_id: str | None = None
    slug: str | None = None
    name: str | None = None
    description: str | None = None
    status: WorkspaceStatus | None = None
    created_at: str | None = None
    archived_at: str | None = None
    metadata: dict[str, Any] | None = None


class WorkspaceList(_Base):
    """Wrapper returned by ``GET /api/workspaces``."""

    tenant_id: str | None = None
    workspaces: list[Workspace] = Field(default_factory=list)


class WorkspaceConfig(_Base):
    """Wrapper for ``GET/PATCH /api/workspaces/:id/config``."""

    workspace_id: str
    config: dict[str, Any] = Field(default_factory=dict)


# ---------------------------------------------------------------------------
# Memory primitive
# ---------------------------------------------------------------------------


class Memory(_Base):
    """A single memory entry — an integrity-gated, cited fact."""

    id: str
    workspace_id: str | None = None
    content: str | None = None
    is_static: bool | None = None
    audience: str | None = None
    citation_uri: str | None = None
    source_chunk_id: str | None = None
    metadata: dict[str, Any] | None = None
    root_memory_id: str | None = None
    parent_memory_id: str | None = None
    version: int | None = None
    forgotten_at: str | None = None
    forget_reason: str | None = None
    created_at: str | None = None
    updated_at: str | None = None


class MemoryList(_Base):
    """Wrapper for ``GET /api/memory``."""

    workspace_id: str | None = None
    count: int = 0
    memories: list[Memory] = Field(default_factory=list)


class MemoryVersions(_Base):
    """Wrapper for ``GET /api/memory/:id/versions``."""

    memory_id: str
    root_id: str | None = None
    versions: list[Memory] = Field(default_factory=list)


class MemoryRelation(_Base):
    """A single edge in the memory relation graph."""

    source_id: str | None = None
    target_id: str | None = None
    relation: str | None = None
    metadata: dict[str, Any] | None = None


class MemoryRelations(_Base):
    """Wrapper for ``GET /api/memory/:id/relations``."""

    memory_id: str
    inbound: list[MemoryRelation] = Field(default_factory=list)
    outbound: list[MemoryRelation] = Field(default_factory=list)


# ---------------------------------------------------------------------------
# Wiki
# ---------------------------------------------------------------------------


class WikiPageSummary(_Base):
    """One row of ``GET /api/wiki``."""

    slug: str
    audience: str | None = None
    version: int | None = None
    last_curated: str | None = None
    curated_by: str | None = None
    size_bytes: int | None = None
    workspace_id: str | None = None


class WikiList(_Base):
    """Wrapper for ``GET /api/wiki``."""

    tenant_id: str | None = None
    workspace_id: str | None = None
    pages: list[WikiPageSummary] = Field(default_factory=list)


class WikiPage(_Base):
    """A rendered wiki page from ``GET /api/wiki/:slug``."""

    slug: str
    audience: str | None = None
    version: int | None = None
    workspace_id: str | None = None
    body: str | None = None
    warnings: list[Any] = Field(default_factory=list)


class Contradiction(_Base):
    """One contradiction event from ``GET /api/wiki/contradictions``."""

    page_slug: str | None = None
    contradictions: list[Any] = Field(default_factory=list)
    entities: list[Any] = Field(default_factory=list)
    score: float | None = None
    detected_at: str | None = None


class ContradictionList(_Base):
    """Wrapper for ``GET /api/wiki/contradictions``."""

    workspace_id: str | None = None
    contradictions: list[Contradiction] = Field(default_factory=list)
    count: int = 0


# ---------------------------------------------------------------------------
# Subscriptions / Surfacing
# ---------------------------------------------------------------------------


class Subscription(_Base):
    """A surfacing subscription — proactive push trigger."""

    id: str
    tenant_id: str | None = None
    workspace_id: str | None = None
    principal_id: str | None = None
    scope: SubscriptionScope | None = None
    scope_value: str | None = None
    categories: list[str] = Field(default_factory=list)
    activity: str | None = None
    status: str | None = None
    created_at: str | None = None


class SubscriptionList(_Base):
    """Wrapper for ``GET /api/subscriptions``."""

    workspace_id: str | None = None
    subscriptions: list[Subscription] = Field(default_factory=list)


class SurfaceEvent(_Base):
    """One server-sent event from ``GET /api/surface/stream``."""

    event: str
    """Event name (e.g. 'ready', 'surface', 'keepalive')."""

    data: dict[str, Any] = Field(default_factory=dict)
    """Decoded JSON payload (empty for keepalive comments)."""


# ---------------------------------------------------------------------------
# Status
# ---------------------------------------------------------------------------


class Status(_Base):
    """Engine liveness + readiness from ``GET /api/status``."""

    status: str | None = None
    ok: bool | None = Field(default=None, alias="ok?")
    checks: dict[str, Any] = Field(default_factory=dict)
    degraded: list[str] = Field(default_factory=list)


__all__ = [
    "AskResult",
    "Bandwidth",
    "Citation",
    "Contradiction",
    "ContradictionList",
    "GrepMatch",
    "GrepResponse",
    "Memory",
    "MemoryList",
    "MemoryRelation",
    "MemoryRelations",
    "MemoryVersions",
    "Profile",
    "ResponseFormat",
    "Scale",
    "SearchResponse",
    "SearchResult",
    "Status",
    "Subscription",
    "SubscriptionList",
    "SubscriptionScope",
    "SurfaceEvent",
    "WikiList",
    "WikiPage",
    "WikiPageSummary",
    "Workspace",
    "WorkspaceConfig",
    "WorkspaceList",
    "WorkspaceStatus",
]
