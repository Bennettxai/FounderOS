defmodule OptimalEngine.Graph.WorkspaceScopingTest do
  @moduledoc """
  T3 workspace isolation for the knowledge graph:

    * edge writes carry the originating context's workspace_id (the old
      @edge_insert_sql omitted it, backfilling every edge to 'default')
    * reads (edges_for / stats / subgraph / related_contexts) are scoped
    * per-workspace rebuild only touches that workspace's edges
    * Analyzer hubs/clusters never cross workspaces
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Context
  alias OptimalEngine.Graph
  alias OptimalEngine.Graph.Analyzer
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
    ws_a = "graph-ws-a-#{suffix}"
    ws_b = "graph-ws-b-#{suffix}"
    node = "graph-node-#{suffix}"

    a = insert_context("grapha#{suffix}", ws_a, node, ["GraphEntA#{suffix}"])
    b = insert_context("graphb#{suffix}", ws_b, node, ["GraphEntB#{suffix}"])

    {:ok, ws_a: ws_a, ws_b: ws_b, node: node, a: a, b: b, suffix: suffix}
  end

  test "edge writes carry the context's workspace_id", ctx do
    {:ok, rows} =
      Store.raw_query(
        "SELECT DISTINCT workspace_id FROM edges WHERE source_id = ?1 OR target_id = ?1",
        [ctx.a.id]
      )

    assert rows == [[ctx.ws_a]]
  end

  test "edges_for is workspace-scoped", ctx do
    # Both workspaces have a `lives_in` edge to the SAME node slug — scoping
    # is what keeps them apart.
    {:ok, a_edges} = Graph.edges_for(ctx.a.id, direction: :out, workspace_id: ctx.ws_a)
    assert Enum.any?(a_edges, &(&1.relation == "lives_in" and &1.target_id == ctx.node))

    {:ok, cross} = Graph.edges_for(ctx.a.id, direction: :out, workspace_id: ctx.ws_b)
    assert cross == []
  end

  test "related_contexts is workspace-scoped", ctx do
    {:ok, related} =
      Graph.related_contexts("GraphEntA#{ctx.suffix}", workspace_id: ctx.ws_a)

    assert ctx.a.id in related

    {:ok, other} =
      Graph.related_contexts("GraphEntA#{ctx.suffix}", workspace_id: ctx.ws_b)

    assert other == []
  end

  test "stats counts only the requested workspace", ctx do
    {:ok, stats_a} = Graph.stats(workspace_id: ctx.ws_a)
    # mentioned_in + lives_in for exactly one context
    assert stats_a["total_edges"] == 2
    assert stats_a["mentioned_in"] == 1
    assert stats_a["lives_in"] == 1
  end

  test "per-workspace rebuild only touches that workspace's edges", ctx do
    {:ok, [[b_before]]} =
      Store.raw_query("SELECT COUNT(*) FROM edges WHERE workspace_id = ?1", [ctx.ws_b])

    assert b_before == 2

    {:ok, _count} = Graph.rebuild(workspace_id: ctx.ws_a)

    # A's edges were recreated with correct attribution
    {:ok, rows} =
      Store.raw_query(
        "SELECT DISTINCT workspace_id FROM edges WHERE source_id = ?1 OR target_id = ?1",
        [ctx.a.id]
      )

    assert rows == [[ctx.ws_a]]

    # B's edges were untouched
    {:ok, [[b_after]]} =
      Store.raw_query("SELECT COUNT(*) FROM edges WHERE workspace_id = ?1", [ctx.ws_b])

    assert b_after == b_before
  end

  test "Analyzer hubs and clusters never include another workspace's ids", ctx do
    {:ok, hubs} = Analyzer.hubs(workspace_id: ctx.ws_a)
    hub_ids = Enum.map(hubs, & &1.id)
    refute ctx.b.id in hub_ids
    refute "GraphEntB#{ctx.suffix}" in hub_ids

    {:ok, clusters} = Analyzer.clusters(workspace_id: ctx.ws_a)
    all_members = Enum.flat_map(clusters, &MapSet.to_list/1)
    assert ctx.a.id in all_members
    refute ctx.b.id in all_members
  end
end
