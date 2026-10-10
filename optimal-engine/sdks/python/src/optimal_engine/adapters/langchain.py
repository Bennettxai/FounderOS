"""LangChain adapter.

Exposes one :class:`BaseTool` subclass per Optimal Engine verb, each with
a Pydantic ``args_schema`` so LangChain can validate the LLM's argument
payloads before dispatching. Sync and async (``_run`` / ``_arun``) are
both implemented when an :class:`AsyncOptimalEngine` is provided.

Install the extra::

    pip install "optimal-engine[langchain]"

Usage::

    from optimal_engine import OptimalEngine
    from optimal_engine.adapters.langchain import optimal_engine_tools

    client = OptimalEngine()
    tools = optimal_engine_tools(client)
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, Field

if TYPE_CHECKING:
    from .._async_client import AsyncOptimalEngine
    from .._client import OptimalEngine


def _import_basetool() -> Any:
    try:
        from langchain_core.tools import BaseTool
    except ImportError as e:
        raise ImportError(
            "langchain-core is required. Install with: "
            "pip install 'optimal-engine[langchain]'"
        ) from e
    return BaseTool


# ---------------------------------------------------------------------------
# Argument schemas (pydantic models LangChain validates against)
# ---------------------------------------------------------------------------


class AskInput(BaseModel):
    query: str = Field(description="The natural-language question to ask the second brain.")
    audience: str = Field(default="default", description="Audience tag for the response.")
    bandwidth: str = Field(default="medium", description="Density: l0 / low / medium / high / l1 / full.")


class SearchInput(BaseModel):
    query: str = Field(description="Full-text search query.")
    limit: int = Field(default=10, description="Max number of context-level results.")


class GrepInput(BaseModel):
    query: str = Field(description="Hybrid semantic + literal grep query.")
    intent: str | None = Field(default=None, description="Filter by intent (one of 10 canonical values).")
    scale: str | None = Field(default=None, description="document | section | paragraph | chunk")
    limit: int = Field(default=25, description="Max number of chunk-level matches.")


class AddMemoryInput(BaseModel):
    content: str = Field(description="The fact to remember.")
    citation_uri: str = Field(description="REQUIRED. Source URI grounding the fact.")
    audience: str = Field(default="default", description="Audience tag.")
    is_static: bool = Field(default=False, description="True if the fact is unlikely to change.")


class UpdateMemoryInput(BaseModel):
    memory_id: str = Field(description="The memory's id.")
    content: str = Field(description="The new content; previous version stays in the chain.")


class ForgetMemoryInput(BaseModel):
    memory_id: str = Field(description="The memory's id.")
    reason: str | None = Field(default=None, description="Why the memory is being forgotten.")


class WikiGetInput(BaseModel):
    slug: str = Field(description="The wiki page slug.")
    audience: str = Field(default="default", description="Audience tag for the rendered page.")


class RecallWhoInput(BaseModel):
    topic: str = Field(description="The topic / artifact to look up ownership for.")
    role: str = Field(default="owner", description="The role of interest (default: owner).")


# ---------------------------------------------------------------------------
# Tool factory
# ---------------------------------------------------------------------------


def optimal_engine_tools(
    client: OptimalEngine,
    *,
    async_client: AsyncOptimalEngine | None = None,
    workspace: str | None = None,
) -> list[Any]:
    """Return all eight Optimal Engine verbs as LangChain ``BaseTool`` instances.

    Args:
        client: Sync :class:`OptimalEngine` used by ``_run``.
        async_client: Optional :class:`AsyncOptimalEngine` used by ``_arun``.
            When omitted, async invocations fall back to running the sync
            client in a worker thread (LangChain handles this).
        workspace: Default workspace passed to every call.
    """
    BaseTool = _import_basetool()
    ws = workspace or client.workspace

    class AskEngineTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "ask_engine"
        description: str = (
            "Ask the second brain. Wiki-first, hybrid-fallback, hot-cited envelope."
        )
        args_schema: type[BaseModel] = AskInput

        def _run(self, query: str, audience: str = "default", bandwidth: str = "medium") -> dict[str, Any]:
            return client.ask(
                query, workspace=ws, audience=audience, bandwidth=bandwidth  # type: ignore[arg-type]
            ).model_dump()

        async def _arun(self, query: str, audience: str = "default", bandwidth: str = "medium") -> dict[str, Any]:
            if async_client is None:
                return self._run(query, audience, bandwidth)
            result = await async_client.ask(
                query, workspace=ws, audience=audience, bandwidth=bandwidth  # type: ignore[arg-type]
            )
            return result.model_dump()

    class SearchEngineTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "search_engine"
        description: str = "Full-text search across the workspace; returns context-level metadata."
        args_schema: type[BaseModel] = SearchInput

        def _run(self, query: str, limit: int = 10) -> dict[str, Any]:
            return client.search(query, workspace=ws, limit=limit).model_dump()

        async def _arun(self, query: str, limit: int = 10) -> dict[str, Any]:
            if async_client is None:
                return self._run(query, limit)
            result = await async_client.search(query, workspace=ws, limit=limit)
            return result.model_dump()

    class GrepEngineTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "grep_engine"
        description: str = "Hybrid semantic + literal grep; chunk-level matches with full signal trace."
        args_schema: type[BaseModel] = GrepInput

        def _run(
            self, query: str, intent: str | None = None, scale: str | None = None, limit: int = 25
        ) -> dict[str, Any]:
            return client.grep(
                query, workspace=ws, intent=intent, scale=scale, limit=limit  # type: ignore[arg-type]
            ).model_dump()

        async def _arun(
            self, query: str, intent: str | None = None, scale: str | None = None, limit: int = 25
        ) -> dict[str, Any]:
            if async_client is None:
                return self._run(query, intent, scale, limit)
            result = await async_client.grep(
                query, workspace=ws, intent=intent, scale=scale, limit=limit  # type: ignore[arg-type]
            )
            return result.model_dump()

    class AddMemoryTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "add_memory"
        description: str = (
            "Add a cited fact to long-term memory. Integrity-gated — must include source URI."
        )
        args_schema: type[BaseModel] = AddMemoryInput

        def _run(
            self, content: str, citation_uri: str, audience: str = "default", is_static: bool = False
        ) -> dict[str, Any]:
            return client.memory.create(
                content=content,
                citation_uri=citation_uri,
                audience=audience,
                is_static=is_static,
                workspace=ws,
            ).model_dump()

        async def _arun(
            self, content: str, citation_uri: str, audience: str = "default", is_static: bool = False
        ) -> dict[str, Any]:
            if async_client is None:
                return self._run(content, citation_uri, audience, is_static)
            result = await async_client.memory.create(
                content=content,
                citation_uri=citation_uri,
                audience=audience,
                is_static=is_static,
                workspace=ws,
            )
            return result.model_dump()

    class UpdateMemoryTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "update_memory"
        description: str = "Create a new version of a memory. Old version stays in the chain."
        args_schema: type[BaseModel] = UpdateMemoryInput

        def _run(self, memory_id: str, content: str) -> dict[str, Any]:
            return client.memory.update(memory_id, content=content).model_dump()

        async def _arun(self, memory_id: str, content: str) -> dict[str, Any]:
            if async_client is None:
                return self._run(memory_id, content)
            result = await async_client.memory.update(memory_id, content=content)
            return result.model_dump()

    class ForgetMemoryTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "forget_memory"
        description: str = "Soft-forget a memory; audit trail preserved."
        args_schema: type[BaseModel] = ForgetMemoryInput

        def _run(self, memory_id: str, reason: str | None = None) -> str:
            client.memory.forget(memory_id, reason=reason)
            return f"forgot {memory_id}"

        async def _arun(self, memory_id: str, reason: str | None = None) -> str:
            if async_client is None:
                return self._run(memory_id, reason)
            await async_client.memory.forget(memory_id, reason=reason)
            return f"forgot {memory_id}"

    class WikiGetTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "get_wiki_page"
        description: str = "Render a curated wiki page for a specific audience."
        args_schema: type[BaseModel] = WikiGetInput

        def _run(self, slug: str, audience: str = "default") -> dict[str, Any]:
            return client.wiki.get(slug, audience=audience, workspace=ws).model_dump()

        async def _arun(self, slug: str, audience: str = "default") -> dict[str, Any]:
            if async_client is None:
                return self._run(slug, audience)
            result = await async_client.wiki.get(slug, audience=audience, workspace=ws)
            return result.model_dump()

    class RecallWhoTool(BaseTool):  # type: ignore[misc, valid-type]
        name: str = "recall_who"
        description: str = "Contact / ownership lookup — 'who owns X'."
        args_schema: type[BaseModel] = RecallWhoInput

        def _run(self, topic: str, role: str = "owner") -> dict[str, Any]:
            return client.recall.who(topic=topic, role=role, workspace=ws).model_dump()

        async def _arun(self, topic: str, role: str = "owner") -> dict[str, Any]:
            if async_client is None:
                return self._run(topic, role)
            result = await async_client.recall.who(topic=topic, role=role, workspace=ws)
            return result.model_dump()

    return [
        AskEngineTool(),
        SearchEngineTool(),
        GrepEngineTool(),
        AddMemoryTool(),
        UpdateMemoryTool(),
        ForgetMemoryTool(),
        WikiGetTool(),
        RecallWhoTool(),
    ]


__all__ = [
    "AddMemoryInput",
    "AskInput",
    "ForgetMemoryInput",
    "GrepInput",
    "RecallWhoInput",
    "SearchInput",
    "UpdateMemoryInput",
    "WikiGetInput",
    "optimal_engine_tools",
]
