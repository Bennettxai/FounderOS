"""OpenAI Agents SDK adapter.

Exports :func:`optimal_engine_tools`, which returns a list of
``function_tool``-decorated callables ready to drop into an Agent's
``tools=`` parameter. Each tool is named for an action verb (``ask_engine``,
``add_memory``, ``forget_memory`` …) and carries a one-line docstring
that follows the engine's flavor.

Install the extra::

    pip install "optimal-engine[openai]"

Usage::

    from agents import Agent
    from optimal_engine import OptimalEngine
    from optimal_engine.adapters.openai_agents import optimal_engine_tools

    client = OptimalEngine(base_url="http://localhost:4200")
    agent = Agent(
        name="brain",
        instructions="Answer using the second brain.",
        tools=optimal_engine_tools(client, workspace="default"),
    )
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, Callable

if TYPE_CHECKING:
    from .._client import OptimalEngine


def optimal_engine_tools(
    client: OptimalEngine,
    *,
    workspace: str | None = None,
) -> list[Any]:
    """Return the eight Optimal Engine verbs as OpenAI Agents tools.

    Args:
        client: A configured :class:`OptimalEngine` instance.
        workspace: Default workspace passed to every call. Falls back to
            the client's default if omitted.
    """
    try:
        from agents import function_tool  # type: ignore[import-not-found]
    except ImportError as e:
        raise ImportError(
            "openai-agents is required. Install with: pip install 'optimal-engine[openai]'"
        ) from e

    ws = workspace or client.workspace

    @function_tool
    def ask_engine(
        query: str,
        audience: str = "default",
        bandwidth: str = "medium",
    ) -> dict[str, Any]:
        """Ask the second brain. Wiki-first, hybrid-fallback, hot-cited envelope."""
        result = client.ask(query, workspace=ws, audience=audience, bandwidth=bandwidth)  # type: ignore[arg-type]
        return result.model_dump()

    @function_tool
    def search_engine(query: str, limit: int = 10) -> dict[str, Any]:
        """Full-text search across the workspace; returns context-level metadata."""
        return client.search(query, workspace=ws, limit=limit).model_dump()

    @function_tool
    def grep_engine(
        query: str,
        intent: str | None = None,
        scale: str | None = None,
        limit: int = 25,
    ) -> dict[str, Any]:
        """Hybrid semantic + literal grep; chunk-level matches with full signal trace."""
        return client.grep(
            query,
            workspace=ws,
            intent=intent,
            scale=scale,  # type: ignore[arg-type]
            limit=limit,
        ).model_dump()

    @function_tool
    def add_memory(
        content: str,
        citation_uri: str,
        audience: str = "default",
        is_static: bool = False,
    ) -> dict[str, Any]:
        """Add a cited fact to long-term memory. Integrity-gated — must include source URI."""
        return client.memory.create(
            content=content,
            citation_uri=citation_uri,
            audience=audience,
            is_static=is_static,
            workspace=ws,
        ).model_dump()

    @function_tool
    def update_memory(memory_id: str, content: str) -> dict[str, Any]:
        """Create a new version of a memory. Old version stays in the version chain."""
        return client.memory.update(memory_id, content=content).model_dump()

    @function_tool
    def forget_memory(memory_id: str, reason: str | None = None) -> str:
        """Soft-forget a memory; audit trail preserved."""
        client.memory.forget(memory_id, reason=reason)
        return f"forgot {memory_id}"

    @function_tool
    def get_wiki_page(slug: str, audience: str = "default") -> dict[str, Any]:
        """Render a curated wiki page for a specific audience."""
        return client.wiki.get(slug, audience=audience, workspace=ws).model_dump()

    @function_tool
    def recall_who(topic: str, role: str = "owner") -> dict[str, Any]:
        """Contact / ownership lookup — 'who owns X'."""
        return client.recall.who(topic=topic, role=role, workspace=ws).model_dump()

    return [
        ask_engine,
        search_engine,
        grep_engine,
        add_memory,
        update_memory,
        forget_memory,
        get_wiki_page,
        recall_who,
    ]


__all__: list[str] = ["optimal_engine_tools"]


# Re-export the typed Callable signature so static analyzers see it as
# a real symbol even when ``agents`` is not installed.
_optimal_engine_tools_t: Callable[..., list[Any]] = optimal_engine_tools
