defmodule OptimalEngine.Graph.PopulationTest do
  @moduledoc """
  Graph population layer:

    * a context mentioning >= 2 entities creates co_occurs edges (both directions)
      in that context's real workspace
    * claim->about->entity edges via Graph.create_claim_edges/3
    * Graph.graph_candidates/3 surfaces contexts reachable from query entities
    * Mix.Tasks.Optimal.GraphBackfill rebuilds edges idempotently
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Context
  alias OptimalEngine.Graph
  alias OptimalEngine.Store

  defp insert_context(id, workspace_id, node, entities) do
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    ctx = %Context{
      id: id,
      uri: "optimal://nodes/#{node}/#{id}.md",
      type: :signal,
      path: "/tmp/#{id}.md",
      title: "ctx #{id}",
      content: "content #{id}",
      l0_abstract: "SIGNAL | note | #{id}",
      l1_overview: "overview",
      signal: nil,
      node: node,
      sn_ratio: 0.7,
      entities: entities,
      created_at: now,
      modified_at: now,
      routed_to: [],
      workspace_id: workspace_id,
      metadata: %{}
    }

    :ok = Store.insert_context(ctx)
    ctx
  end

  setup do
    suffix = System.unique_integer([:positive])
    ws = "pop-ws-#{suffix}"
    node = "pop-node-#{suffix}"
    e1 = "PopEntOne#{suffix}"
    e2 = "PopEntTwo#{suffix}"

    ctx = insert_context("pop#{suffix}", ws, node, [e1, e2])

    {:ok, ws: ws, node: node, e1: e1, e2: e2, ctx: ctx, suffix: suffix}
  end

  test "context with two entities creates co_occurs edges both directions in its workspace", c do
    {:ok, out} = Graph.edges_for(c.e1, direction: :out, relation: "co_occurs", workspace_id: c.ws)
    assert Enum.any?(out, &(&1.target_id == c.e2))

    {:ok, back} = Graph.edges_for(c.e2, direction: :out, relation: "co_occurs", workspace_id: c.ws)
    assert Enum.any?(back, &(&1.target_id == c.e1))

    # no leakage into default
    {:ok, none} =
      Graph.edges_for(c.e1, direction: :out, relation: "co_occurs", workspace_id: "default")

    assert none == []
  end

  test "create_claim_edges writes claim->about->entity edges", c do
    claim_id = "popclaim#{c.suffix}"
    :ok = Graph.create_claim_edges(claim_id, [c.e1, c.e2], workspace_id: c.ws)

    {:ok, edges} = Graph.edges_for(claim_id, direction: :out, relation: "about", workspace_id: c.ws)
    targets = Enum.map(edges, & &1.target_id) |> Enum.sort()
    assert targets == Enum.sort([c.e1, c.e2])
  end

  test "graph_candidates surfaces contexts mentioning the query entity", c do
    {:ok, ids} = Graph.graph_candidates([c.e1], 1, workspace_id: c.ws)
    assert c.ctx.id in ids
  end

  test "graph_candidates with 2 hops reaches contexts via co_occurring entities", c do
    # A second context mentions e2 alongside a third entity. Starting from e1,
    # hop-1 reaches ctx (via e1), and hop-2 follows e1->co_occurs->e2 to reach ctx2.
    e3 = "PopEntThree#{c.suffix}"
    ctx2 = insert_context("pop2#{c.suffix}", c.ws, c.node, [c.e2, e3])

    {:ok, one_hop} = Graph.graph_candidates([c.e1], 1, workspace_id: c.ws)
    refute ctx2.id in one_hop

    {:ok, two_hop} = Graph.graph_candidates([c.e1], 2, workspace_id: c.ws)
    assert ctx2.id in two_hop
  end

  test "graph_backfill is idempotent", c do
    # Two runs must produce identical edge counts in this workspace.
    run = fn -> Mix.Tasks.Optimal.GraphBackfill.run(["--workspace", c.ws, "--no-claims"]) end

    run.()

    {:ok, [[after_first]]} =
      Store.raw_query("SELECT COUNT(*) FROM edges WHERE workspace_id = ?1", [c.ws])

    run.()

    {:ok, [[after_second]]} =
      Store.raw_query("SELECT COUNT(*) FROM edges WHERE workspace_id = ?1", [c.ws])

    assert after_first == after_second
    assert after_first > 0
  end
end
