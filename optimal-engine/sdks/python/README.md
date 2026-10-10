# optimal-engine

Python client for the [Optimal Engine](https://github.com/optimalengine/optimal-engine) — open-source memory infrastructure for LLM agents.

```bash
pip install optimal-engine
```

Python 3.10+. Built on `httpx` and `pydantic` v2.

---

## Quick start

```python
from optimal_engine import OptimalEngine

client = OptimalEngine(base_url="http://localhost:4200", workspace="default")

answer = client.ask("what did we decide on pricing?", audience="sales")
print(answer.body)
for citation in answer.citations:
    print(f"  via {citation.slug}")
```

Async:

```python
from optimal_engine import AsyncOptimalEngine

async with AsyncOptimalEngine() as client:
    answer = await client.ask("renewal pipeline status")
    print(answer.body)
```

---

## API surface

### Retrieval

```python
result   = client.ask("pricing decisions", audience="sales")
results  = client.search("microvm", limit=10)
matches  = client.grep("pricing", intent="decision", scale="paragraph")
profile  = client.profile(audience="default", bandwidth="l1")
```

### Recall (Engramme-style cued recall)

```python
client.recall.actions(topic="pricing", since="2026-01-01")
client.recall.who(topic="microvm")
client.recall.when(event="renewal pipeline")
client.recall.where(thing="security audit notes")
client.recall.owns(actor="Bob")
```

### Memory

```python
mem = client.memory.create(
    content="Bob is the platform lead",
    citation_uri="slack://team/messages/1234",
    is_static=True,
    audience="sales",
)

client.memory.get(mem.id)
client.memory.list(workspace="default", limit=20)
client.memory.update(mem.id, content="Bob is now the engineering director")
client.memory.forget(mem.id, reason="superseded by org chart change")
client.memory.versions(mem.id)
client.memory.relations(mem.id)
```

### Workspaces

```python
ws       = client.workspaces.create(slug="research", name="Research Brain")
config   = client.workspaces.config("research")
client.workspaces.update_config("research", {"contradictions": {"policy": "reject"}})
client.workspaces.archive("research")
```

### Wiki

```python
pages = client.wiki.list()
page  = client.wiki.get("customer portal-pricing-decision", audience="sales")
client.wiki.contradictions()
```

### Subscriptions and surfacing (SSE)

```python
sub = client.subscriptions.create(
    scope="topic",
    scope_value="pricing",
    categories=["recent_actions"],
)

for event in client.surface.stream(sub.id):
    print(event.event, event.data)
```

Async stream:

```python
async for event in client.surface.stream(sub.id):
    print(event.event, event.data)
```

---

## Adapters

### OpenAI Agents SDK

```bash
pip install "optimal-engine[openai]"
```

```python
from agents import Agent
from optimal_engine import OptimalEngine
from optimal_engine.adapters.openai_agents import optimal_engine_tools

client = OptimalEngine()
agent = Agent(
    name="brain",
    instructions="Use the second brain to answer.",
    tools=optimal_engine_tools(client, workspace="default"),
)
```

### LangChain

```bash
pip install "optimal-engine[langchain]"
```

```python
from optimal_engine import OptimalEngine, AsyncOptimalEngine
from optimal_engine.adapters.langchain import optimal_engine_tools

client = OptimalEngine()
async_client = AsyncOptimalEngine()
tools = optimal_engine_tools(client, async_client=async_client)
```

---

## Errors

Every error inherits from `OptimalEngineError`:

```python
from optimal_engine import (
    OptimalEngineError,
    NotFoundError,
    ValidationError,
    RateLimitError,
    APIConnectionError,
)

try:
    client.memory.get("does-not-exist")
except NotFoundError:
    ...
except OptimalEngineError as e:
    print(e.status_code, e.message, e.body)
```

---

## License

MIT.
