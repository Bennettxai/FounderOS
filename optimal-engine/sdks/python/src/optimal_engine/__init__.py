"""Optimal Engine — Python SDK.

A pythonic, type-annotated client for the Optimal Engine, the
open-source memory infrastructure for LLM agents.

Quick start::

    from optimal_engine import OptimalEngine

    client = OptimalEngine(base_url="http://localhost:4200", workspace="default")
    answer = client.ask("what did we decide on pricing?", audience="sales")
    print(answer.body)

Async::

    from optimal_engine import AsyncOptimalEngine

    async with AsyncOptimalEngine() as client:
        answer = await client.ask("renewal pipeline status")
"""

from __future__ import annotations

from ._async_client import AsyncOptimalEngine
from ._client import OptimalEngine
from .exceptions import (
    APIConnectionError,
    APIStatusError,
    APITimeoutError,
    AuthenticationError,
    BadRequestError,
    ConflictError,
    InternalServerError,
    NotFoundError,
    OptimalEngineError,
    PermissionDeniedError,
    RateLimitError,
    ValidationError,
)
from .types import (
    AskResult,
    Bandwidth,
    Citation,
    Contradiction,
    ContradictionList,
    GrepMatch,
    GrepResponse,
    Memory,
    MemoryList,
    MemoryRelation,
    MemoryRelations,
    MemoryVersions,
    Profile,
    ResponseFormat,
    Scale,
    SearchResponse,
    SearchResult,
    Status,
    Subscription,
    SubscriptionList,
    SubscriptionScope,
    SurfaceEvent,
    WikiList,
    WikiPage,
    WikiPageSummary,
    Workspace,
    WorkspaceConfig,
    WorkspaceList,
    WorkspaceStatus,
)

__version__ = "0.1.0"

__all__ = [
    # Clients
    "AsyncOptimalEngine",
    "OptimalEngine",
    # Exceptions
    "APIConnectionError",
    "APIStatusError",
    "APITimeoutError",
    "AuthenticationError",
    "BadRequestError",
    "ConflictError",
    "InternalServerError",
    "NotFoundError",
    "OptimalEngineError",
    "PermissionDeniedError",
    "RateLimitError",
    "ValidationError",
    # Types
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
    "__version__",
]
