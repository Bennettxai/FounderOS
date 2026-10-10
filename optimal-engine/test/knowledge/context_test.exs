defmodule OptimalEngine.Knowledge.ContextTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.{Context, Store}

  setup do
    store_id = "ctx_test_#{:erlang.unique_integer([:positive])}"

    {:ok, pid} =
      Store.start_link(
        store_id: store_id,
        name: :"ctx_store_#{store_id}"
      )

    # Seed knowledge graph
    :ok = Store.assert(pid, "agent:1", "role", "researcher")
    :ok = Store.assert(pid, "agent:1", "knows", "user:alice")
    :ok = Store.assert(pid, "agent:1", "knows", "user:bob")
    :ok = Store.assert(pid, "agent:1", "status", "active")
    :ok = Store.assert(pid, "user:alice", "role", "admin")

    on_exit(fn ->
      if Process.alive?(pid), do: GenServer.stop(pid)
    end)

    %{store: pid}
  end

  test "for_agent returns structured context", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "agent:1")

    assert ctx.agent_id == "agent:1"
    assert ctx.fact_count == 4
    assert {"agent:1", "role", "researcher"} in ctx.facts
    assert {"agent:1", "knows", "user:alice"} in ctx.facts
  end

  test "for_agent with multiple scopes", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "agent:1", scope: ["agent:1", "user:alice"])

    assert ctx.fact_count == 5
    assert {"user:alice", "role", "admin"} in ctx.facts
  end

  test "for_agent with max_facts limit", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "agent:1", max_facts: 2)
    assert ctx.fact_count == 2
  end

  test "for_agent builds relationships map", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "agent:1")

    assert Map.has_key?(ctx.relationships, "knows")
    assert "user:alice" in ctx.relationships["knows"]
    assert "user:bob" in ctx.relationships["knows"]
  end

  test "for_agent builds properties map", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "agent:1")

    assert ctx.properties["role"] == "researcher"
    assert ctx.properties["status"] == "active"
  end

  test "to_prompt renders formatted text", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "agent:1")
    prompt = Context.to_prompt(ctx)

    assert prompt =~ "Knowledge Context (agent:1)"
    assert prompt =~ "Facts: 4"
    assert prompt =~ "role: researcher"
    assert prompt =~ "knows:"
  end

  test "to_prompt handles empty context", %{store: store} do
    ctx = Context.for_agent(store, agent_id: "nonexistent:agent")
    prompt = Context.to_prompt(ctx)

    assert prompt =~ "No facts in knowledge graph"
  end
end
