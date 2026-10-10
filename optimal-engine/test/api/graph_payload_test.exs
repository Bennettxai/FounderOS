defmodule OptimalEngine.API.GraphPayloadTest do
  @moduledoc """
  Graph payload workspace-isolation regression tests.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.API.GraphPayload
  alias OptimalEngine.Context
  alias OptimalEngine.Graph
  alias OptimalEngine.Store

  defp insert_context(id, workspace_id, node, entities) do
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    %Context{
      id: id,
      uri: "optimal://nodes/#{node}/#{id}.md",
      type: :signal,
      path: "/tmp/#{id}.md",
      title: "ctx #{id}",
      content: "content #{id}",
      l0_abstract: "SIGNAL | note | #{id}",
      l1_overview: "overview #{id}",
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
    |> then(fn ctx ->
      :ok = Store.insert_context(ctx)
      :ok = Graph.create_edges_for_context(ctx)
      ctx
    end)
  end

  setup do
    suffix = System.unique_integer([:positive])
    ws_a = "payload-ws-a-#{suffix}"
    ws_b = "payload-ws-b-#{suffix}"
    node = "payload-node-#{suffix}"

    a1 = insert_context("pga1#{suffix}", ws_a, node, ["EntityA#{suffix}"])
    a2 = insert_context("pga2#{suffix}", ws_a, node, ["EntityA#{suffix}"])

    b1 = insert_context("pgb1#{suffix}", ws_b, node, ["EntityB#{suffix}"])
    b2 = insert_context("pgb2#{suffix}", ws_b, node, ["EntityB#{suffix}"])

    a3 = insert_context("pgtri-a1#{suffix}", ws_a, node, ["TriHubA#{suffix}", "SharedA#{suffix}"])
    a4 = insert_context("pgtri-a2#{suffix}", ws_a, node, ["TriHubA#{suffix}", "SharedA#{suffix}"])
    b3 = insert_context("pgtri-b1#{suffix}", ws_b, node, ["TriHubB#{suffix}", "SharedB#{suffix}"])
    b4 = insert_context("pgtri-b2#{suffix}", ws_b, node, ["TriHubB#{suffix}", "SharedB#{suffix}"])

    {
      :ok,
      ws_a: ws_a,
      ws_b: ws_b,
      node: node,
      a1: a1,
      a2: a2,
      b1: b1,
      b2: b2,
      a3: a3,
      a4: a4,
      b3: b3,
      b4: b4,
      suffix: suffix
    }
  end

  test "full_graph defaults and explicit workspace are scoped", ctx do
    scoped_a = GraphPayload.full_graph(ctx.ws_a)
    scoped_b = GraphPayload.full_graph(ctx.ws_b)
    scoped_default = GraphPayload.full_graph()

    assert scoped_a[:workspace_id] == ctx.ws_a
    assert scoped_b[:workspace_id] == ctx.ws_b
    assert scoped_default[:workspace_id] == "default"

    a_context_ids = Enum.map(scoped_a[:contexts], & &1[:id])
    b_context_ids = Enum.map(scoped_b[:contexts], & &1[:id])

    assert ctx.a1.id in a_context_ids
    assert ctx.a2.id in a_context_ids
    refute ctx.b1.id in a_context_ids
    refute ctx.b2.id in a_context_ids

    assert ctx.b1.id in b_context_ids
    assert ctx.b2.id in b_context_ids
    refute ctx.a1.id in b_context_ids

    a_entities = Enum.map(scoped_a[:entities], & &1[:name])
    b_entities = Enum.map(scoped_b[:entities], & &1[:name])

    assert "EntityA#{ctx.suffix}" in a_entities
    refute "EntityB#{ctx.suffix}" in a_entities

    assert "EntityB#{ctx.suffix}" in b_entities
    refute "EntityA#{ctx.suffix}" in b_entities

    a_edge_ids =
      scoped_a[:edges]
      |> Enum.flat_map(fn e -> [e[:source], e[:target]] end)

    b_edge_ids =
      scoped_b[:edges]
      |> Enum.flat_map(fn e -> [e[:source], e[:target]] end)

    refute ctx.b1.id in a_edge_ids
    refute ctx.b2.id in a_edge_ids

    refute ctx.a1.id in b_edge_ids
    refute ctx.a2.id in b_edge_ids
  end

  test "node payload includes contexts and edges from one workspace only", ctx do
    a_payload = GraphPayload.node(ctx.node, ctx.ws_a)
    b_payload = GraphPayload.node(ctx.node, ctx.ws_b)

    a_ids = Enum.map(a_payload[:contexts], & &1[:id])
    b_ids = Enum.map(b_payload[:contexts], & &1[:id])

    assert ctx.a1.id in a_ids
    assert ctx.a2.id in a_ids
    refute ctx.b1.id in a_ids

    assert ctx.b1.id in b_ids
    assert ctx.b2.id in b_ids
    refute ctx.a1.id in b_ids
  end

  test "hubs, triangles, clusters, and reflection are workspace-scoped", ctx do
    hubs_a = GraphPayload.hubs(ctx.ws_a)
    hubs_b = GraphPayload.hubs(ctx.ws_b)

    refute Enum.any?(
             hubs_a[:hubs],
             &String.contains?(to_string(&1[:entity]), "EntityB#{ctx.suffix}")
           )

    refute Enum.any?(
             hubs_b[:hubs],
             &String.contains?(to_string(&1[:entity]), "EntityA#{ctx.suffix}")
           )

    triangles_a = GraphPayload.triangles(20, ctx.ws_a, skip_llm: true)[:triangles]
    triangles_b = GraphPayload.triangles(20, ctx.ws_b, skip_llm: true)[:triangles]

    a_triangle_nodes = triangles_a |> Enum.flat_map(fn tri -> tri[:nodes] end) || []
    b_triangle_nodes = triangles_b |> Enum.flat_map(fn tri -> tri[:nodes] end) || []

    assert Enum.any?(a_triangle_nodes, &(&1 in [ctx.a3.id, ctx.a4.id]))
    assert Enum.any?(b_triangle_nodes, &(&1 in [ctx.b3.id, ctx.b4.id]))

    refute Enum.any?(a_triangle_nodes, &(&1 in [ctx.b1.id, ctx.b2.id, ctx.b3.id, ctx.b4.id]))
    refute Enum.any?(b_triangle_nodes, &(&1 in [ctx.a1.id, ctx.a2.id, ctx.a3.id, ctx.a4.id]))

    clusters_a = GraphPayload.clusters(ctx.ws_a)[:clusters]
    clusters_b = GraphPayload.clusters(ctx.ws_b)[:clusters]

    assert Enum.any?(clusters_a, fn cluster -> ctx.a1.id in cluster and ctx.a2.id in cluster end)
    refute Enum.any?(clusters_b, fn cluster -> ctx.a1.id in cluster end)
    assert Enum.any?(clusters_b, fn cluster -> ctx.b1.id in cluster and ctx.b2.id in cluster end)
    refute Enum.any?(clusters_a, fn cluster -> ctx.b1.id in cluster end)

    gaps_a = GraphPayload.reflection_gaps(2, ctx.ws_a)[:gaps]
    gaps_b = GraphPayload.reflection_gaps(2, ctx.ws_b)[:gaps]

    a_gap_entities = Enum.flat_map(gaps_a, fn g -> [g[:entity_a], g[:entity_b]] end)
    b_gap_entities = Enum.flat_map(gaps_b, fn g -> [g[:entity_a], g[:entity_b]] end)

    assert "TriHubA#{ctx.suffix}" in a_gap_entities
    assert "SharedA#{ctx.suffix}" in a_gap_entities
    refute "TriHubB#{ctx.suffix}" in a_gap_entities
    refute "SharedB#{ctx.suffix}" in a_gap_entities

    refute Enum.any?(b_gap_entities, &(&1 in ["TriHubA#{ctx.suffix}", "SharedA#{ctx.suffix}"]))
  end
end
