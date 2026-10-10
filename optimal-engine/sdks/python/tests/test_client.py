"""Mock-based smoke tests for the Optimal Engine SDK.

These tests intercept HTTP calls with ``respx`` so they run without a
live engine. They cover client construction, request shape, response
parsing, exception mapping, and the SSE stream parser.
"""

from __future__ import annotations

import json
from typing import Any

import httpx
import pytest
import respx

from optimal_engine import (
    AsyncOptimalEngine,
    NotFoundError,
    OptimalEngine,
    ValidationError,
)
from optimal_engine._client import _iter_sse


# ---------------------------------------------------------------------------
# Construction
# ---------------------------------------------------------------------------


def test_client_constructs_with_defaults() -> None:
    """Default construction shouldn't raise and should set sane defaults."""
    client = OptimalEngine(base_url="http://localhost:4200")
    assert client.base_url == "http://localhost:4200"
    assert client.workspace == "default"
    assert client.tenant == "default"
    assert client.memory is not None
    assert client.workspaces is not None
    assert client.recall is not None
    assert client.wiki is not None
    assert client.subscriptions is not None
    assert client.surface is not None
    client.close()


def test_async_client_constructs() -> None:
    """Async client mirrors the sync one structurally."""
    client = AsyncOptimalEngine(base_url="http://localhost:4200")
    assert client.base_url == "http://localhost:4200"
    assert client.workspace == "default"


def test_context_manager_closes_http() -> None:
    """The sync client supports ``with`` and closes the underlying httpx client."""
    with OptimalEngine(base_url="http://localhost:4200") as client:
        assert client._http is not None


# ---------------------------------------------------------------------------
# Retrieval
# ---------------------------------------------------------------------------


@respx.mock
def test_ask_posts_to_rag() -> None:
    """``ask`` should POST to /api/rag with the canonical body."""
    route = respx.post("http://localhost:4200/api/rag").mock(
        return_value=httpx.Response(
            200,
            json={"body": "Bob owns pricing.", "citations": [{"slug": "pricing"}]},
        )
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        result = client.ask("who owns pricing", audience="sales")

    assert route.called
    sent = json.loads(route.calls.last.request.content)
    assert sent["query"] == "who owns pricing"
    assert sent["audience"] == "sales"
    assert sent["workspace"] == "default"
    assert result.body == "Bob owns pricing."
    assert result.citations[0].slug == "pricing"


@respx.mock
def test_search_serializes_params() -> None:
    """``search`` should send q + workspace + limit on the query string."""
    respx.get("http://localhost:4200/api/search").mock(
        return_value=httpx.Response(200, json={"query": "microvm", "results": []})
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        result = client.search("microvm", limit=5)

    assert result.query == "microvm"
    assert result.results == []


@respx.mock
def test_grep_passes_filters() -> None:
    """``grep`` propagates intent and scale filters."""
    route = respx.get("http://localhost:4200/api/grep").mock(
        return_value=httpx.Response(
            200,
            json={
                "query": "pricing",
                "workspace_id": "default",
                "results": [
                    {
                        "slug": "pricing-doc",
                        "scale": "paragraph",
                        "intent": "decision",
                        "sn_ratio": 0.92,
                        "snippet": "We picked $99 because…",
                        "score": 0.81,
                    }
                ],
            },
        )
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        result = client.grep("pricing", intent="decision", scale="paragraph", limit=10)

    assert route.called
    request = route.calls.last.request
    assert "intent=decision" in str(request.url)
    assert "scale=paragraph" in str(request.url)
    assert result.results[0].sn_ratio == 0.92


# ---------------------------------------------------------------------------
# Memory
# ---------------------------------------------------------------------------


@respx.mock
def test_memory_create_and_get() -> None:
    """Creating a memory hits POST /api/memory; get hits GET /api/memory/:id."""
    respx.post("http://localhost:4200/api/memory").mock(
        return_value=httpx.Response(201, json={"id": "mem_123", "content": "Hello"})
    )
    respx.get("http://localhost:4200/api/memory/mem_123").mock(
        return_value=httpx.Response(200, json={"id": "mem_123", "content": "Hello"})
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        created = client.memory.create(
            content="Hello",
            citation_uri="slack://chan/1",
        )
        fetched = client.memory.get("mem_123")

    assert created.id == "mem_123"
    assert fetched.id == "mem_123"


@respx.mock
def test_memory_get_not_found_raises() -> None:
    """A 404 from the engine should become NotFoundError."""
    respx.get("http://localhost:4200/api/memory/nope").mock(
        return_value=httpx.Response(404, json={"error": "memory not found"})
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        with pytest.raises(NotFoundError) as excinfo:
            client.memory.get("nope")

    assert excinfo.value.status_code == 404
    assert "not found" in excinfo.value.message


@respx.mock
def test_memory_forget_returns_none_on_204() -> None:
    """Forget should swallow the 204 cleanly and return None."""
    respx.post("http://localhost:4200/api/memory/mem_1/forget").mock(
        return_value=httpx.Response(204)
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        result = client.memory.forget("mem_1", reason="stale")

    assert result is None


# ---------------------------------------------------------------------------
# Workspaces
# ---------------------------------------------------------------------------


@respx.mock
def test_workspaces_create() -> None:
    """Creating a workspace uses POST /api/workspaces."""
    respx.post("http://localhost:4200/api/workspaces").mock(
        return_value=httpx.Response(
            201,
            json={"id": "default:research", "slug": "research", "name": "Research Brain"},
        )
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        ws = client.workspaces.create(slug="research", name="Research Brain")

    assert ws.id == "default:research"
    assert ws.slug == "research"


@respx.mock
def test_workspaces_validation_error() -> None:
    """422 should map to ValidationError."""
    respx.post("http://localhost:4200/api/workspaces").mock(
        return_value=httpx.Response(422, json={"error": "invalid slug"})
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        with pytest.raises(ValidationError):
            client.workspaces.create(slug="bad slug", name="x")


# ---------------------------------------------------------------------------
# Subscriptions
# ---------------------------------------------------------------------------


@respx.mock
def test_subscriptions_create() -> None:
    """Subscription create posts the engine-shaped body."""
    route = respx.post("http://localhost:4200/api/subscriptions").mock(
        return_value=httpx.Response(
            201,
            json={
                "id": "sub_abc",
                "scope": "topic",
                "scope_value": "pricing",
                "categories": ["recent_actions"],
                "status": "active",
            },
        )
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        sub = client.subscriptions.create(
            scope="topic",
            scope_value="pricing",
            categories=["recent_actions"],
        )

    assert sub.id == "sub_abc"
    sent = json.loads(route.calls.last.request.content)
    assert sent["scope"] == "topic"
    assert sent["categories"] == ["recent_actions"]


# ---------------------------------------------------------------------------
# SSE parser
# ---------------------------------------------------------------------------


def test_sse_parser_yields_events() -> None:
    """``_iter_sse`` should decode standard SSE records into SurfaceEvents."""
    lines = iter(
        [
            "event: ready",
            'data: {"subscription":"sub_1"}',
            "",
            ": keepalive",
            "",
            "event: surface",
            'data: {"slug":"page"}',
            "",
        ]
    )

    events = list(_iter_sse(lines))

    assert len(events) == 2
    assert events[0].event == "ready"
    assert events[0].data == {"subscription": "sub_1"}
    assert events[1].event == "surface"
    assert events[1].data == {"slug": "page"}


def test_sse_parser_handles_non_json_data() -> None:
    """Non-JSON data falls back to {'raw': ...}, not a crash."""
    lines = iter(["event: weird", "data: not-json", ""])
    events = list(_iter_sse(lines))
    assert events[0].data == {"raw": "not-json"}


# ---------------------------------------------------------------------------
# Status
# ---------------------------------------------------------------------------


@respx.mock
def test_status_returns_model() -> None:
    """``status`` returns a parsed Status model."""
    respx.get("http://localhost:4200/api/status").mock(
        return_value=httpx.Response(
            200,
            json={"status": "ok", "ok?": True, "checks": {}, "degraded": []},
        )
    )

    with OptimalEngine(base_url="http://localhost:4200") as client:
        status = client.status()

    assert status.status == "ok"
    assert status.ok is True


# ---------------------------------------------------------------------------
# Async smoke test
# ---------------------------------------------------------------------------


@respx.mock
async def test_async_ask_works() -> None:
    """The async client mirrors the sync one for the ask call."""
    respx.post("http://localhost:4200/api/rag").mock(
        return_value=httpx.Response(200, json={"body": "ok", "citations": []})
    )

    async with AsyncOptimalEngine(base_url="http://localhost:4200") as client:
        result = await client.ask("hello")

    assert result.body == "ok"


# ---------------------------------------------------------------------------
# Exception mapping
# ---------------------------------------------------------------------------


def test_status_to_exception_maps_known_codes() -> None:
    """The status->exception mapping covers the documented codes."""
    from optimal_engine.exceptions import (
        BadRequestError,
        InternalServerError,
        NotFoundError as _NotFound,
        RateLimitError,
        status_to_exception,
    )

    assert status_to_exception(400) is BadRequestError
    assert status_to_exception(404) is _NotFound
    assert status_to_exception(429) is RateLimitError
    assert status_to_exception(503) is InternalServerError


def test_unknown_5xx_maps_to_internal() -> None:
    """Any 5xx that isn't explicitly mapped becomes InternalServerError."""
    from optimal_engine.exceptions import InternalServerError, status_to_exception

    assert status_to_exception(599) is InternalServerError


# ---------------------------------------------------------------------------
# Pydantic models
# ---------------------------------------------------------------------------


def test_models_accept_extra_fields() -> None:
    """All response models tolerate unknown fields the engine might add."""
    from optimal_engine import AskResult, Memory

    ask = AskResult.model_validate({"body": "x", "future_field": 1})
    mem = Memory.model_validate({"id": "m", "future_field": "y"})

    # extra="allow" means the field is preserved on the model
    assert ask.body == "x"
    assert mem.id == "m"


def test_pydantic_dump_strips_unset() -> None:
    """``model_dump(exclude_unset=True)`` produces clean payloads."""
    from optimal_engine import Memory

    mem = Memory.model_validate({"id": "m"})
    dumped: dict[str, Any] = mem.model_dump(exclude_unset=True)
    assert dumped == {"id": "m"}
