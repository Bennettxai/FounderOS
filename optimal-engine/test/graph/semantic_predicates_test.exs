defmodule OptimalEngine.Graph.SemanticPredicatesTest do
  @moduledoc """
  Asserts that the ingest pipeline and graph layer write semantic predicates
  beyond 'lives_in', and that the in-memory triple store hydrates from SQLite.

  Two behaviours under test:

  1. After `Store.insert_context/1` (which calls `Graph.create_edges_for_context_db`),
     the `edges` table contains 'mentioned_in' and 'co_occurs' rows — not only
     'lives_in'.

  2. `Graph.hydrate_triple_store/2` populates a fresh Knowledge.Store from the
     SQLite edges table, including those non-'lives_in' predicates, so triple
     store state survives a simulated restart.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Context
  alias OptimalEngine.Graph
  alias OptimalEngine.Knowledge.Store, as: KStore
  alias OptimalEngine.Store

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  defp unique_ws, do: "sem-pred-ws-#{System.unique_integer([:positive])}"

  defp insert_ctx(id, ws, node, entities) do
    now = DateTime.utc_now() |> DateTime.truncate(:second)

    ctx = %Context{
      id: id,
      uri: "optimal://nodes/#{node}/#{id}.md",
      type: :signal,
      path: "/tmp/#{id}.md",
      title: "test ctx #{id}",
      content: "Alice and Bob discussed the platform.",
      l0_abstract: "SIGNAL | note | #{id}",
      l1_overview: "overview #{id}",
      signal: nil,
      node: node,
      sn_ratio: 0.8,
      entities: entities,
      created_at: now,
      modified_at: now,
      routed_to: [node],
      workspace_id: ws,
      metadata: %{}
    }

    :ok = Store.insert_context(ctx)
    ctx
  end

  # ---------------------------------------------------------------------------
  # Tests: non-'lives_in' predicates in SQLite edges
  # ---------------------------------------------------------------------------

  test "insert_context writes 'mentioned_in' edges for each entity" do
    ws = unique_ws()
    node = "sem-pred-node-#{System.unique_integer([:positive])}"
    e1 = "SemanticAlice#{System.unique_integer([:positive])}"
    id = "sem-ctx-#{System.unique_integer([:positive])}"

    insert_ctx(id, ws, node, [e1])

    {:ok, edges} =
      Graph.edges_for(e1, direction: :out, relation: "mentioned_in", workspace_id: ws)

    assert Enum.any?(edges, &(&1.target_id == id)),
           "expected mentioned_in edge from #{e1} -> #{id}, got: #{inspect(edges)}"
  end

  test "insert_context writes 'co_occurs' edges for entity pairs" do
    ws = unique_ws()
    node = "sem-pred-node-#{System.unique_integer([:positive])}"
    suffix = System.unique_integer([:positive])
    e1 = "SemEnt1#{suffix}"
    e2 = "SemEnt2#{suffix}"
    id = "sem-ctx2-#{suffix}"

    insert_ctx(id, ws, node, [e1, e2])

    {:ok, fwd} = Graph.edges_for(e1, direction: :out, relation: "co_occurs", workspace_id: ws)
    {:ok, rev} = Graph.edges_for(e2, direction: :out, relation: "co_occurs", workspace_id: ws)

    assert Enum.any?(fwd, &(&1.target_id == e2)),
           "expected co_occurs edge #{e1} -> #{e2}, got: #{inspect(fwd)}"

    assert Enum.any?(rev, &(&1.target_id == e1)),
           "expected co_occurs edge #{e2} -> #{e1}, got: #{inspect(rev)}"
  end

  test "edges table after ingest contains relations other than 'lives_in'" do
    ws = unique_ws()
    node = "sem-pred-node-#{System.unique_integer([:positive])}"
    suffix = System.unique_integer([:positive])
    e1 = "SemEntA#{suffix}"
    e2 = "SemEntB#{suffix}"
    id = "sem-ctx3-#{suffix}"

    insert_ctx(id, ws, node, [e1, e2])

    {:ok, rows} =
      Store.raw_query(
        "SELECT DISTINCT relation FROM edges WHERE workspace_id = ?1",
        [ws]
      )

    relations = Enum.map(rows, fn [r] -> r end)

    refute relations == ["lives_in"],
           "expected varied predicates, got only: #{inspect(relations)}"

    assert "mentioned_in" in relations,
           "expected 'mentioned_in' predicate, got: #{inspect(relations)}"

    assert "co_occurs" in relations,
           "expected 'co_occurs' predicate, got: #{inspect(relations)}"
  end

  # ---------------------------------------------------------------------------
  # Tests: triple-store hydration survives restart
  # ---------------------------------------------------------------------------

  test "hydrate_triple_store loads non-'lives_in' predicates into a fresh Knowledge.Store" do
    ws = unique_ws()
    node = "sem-hydrate-node-#{System.unique_integer([:positive])}"
    suffix = System.unique_integer([:positive])
    e1 = "HydEnt1#{suffix}"
    e2 = "HydEnt2#{suffix}"
    id = "hyd-ctx-#{suffix}"

    # Write edges into SQLite.
    insert_ctx(id, ws, node, [e1, e2])

    # Open a fresh (empty) Knowledge.Store that is NOT the default singleton.
    store_id = "hydrate-test-#{suffix}"
    {:ok, store} = KStore.start_link(store_id: store_id)

    # Verify it is empty before hydration.
    {:ok, before_count} = KStore.count(store)
    assert before_count == 0

    # Hydrate only the edges for our test workspace.
    {:ok, loaded} = Graph.hydrate_triple_store(store, workspace_id: ws)
    assert loaded > 0, "expected at least one edge loaded, got 0"

    # Triple store should now contain 'mentioned_in' triples.
    {:ok, mentioned_in_triples} =
      KStore.query(store, predicate: "mentioned_in")

    assert length(mentioned_in_triples) >= 2,
           "expected >=2 mentioned_in triples, got: #{inspect(mentioned_in_triples)}"

    # Triple store should contain 'co_occurs' triples.
    {:ok, co_occurs_triples} =
      KStore.query(store, predicate: "co_occurs")

    assert length(co_occurs_triples) >= 2,
           "expected >=2 co_occurs triples (both directions), got: #{inspect(co_occurs_triples)}"

    KStore.stop(store)
  end

  test "Graph.assert_edge writes to SQLite and feeds the triple store" do
    ws = unique_ws()
    suffix = System.unique_integer([:positive])
    source = "AssertSource#{suffix}"
    target = "AssertTarget#{suffix}"
    predicate = "test_predicate"

    :ok = Graph.assert_edge(source, target, predicate, workspace_id: ws)

    # Verify SQLite persistence.
    {:ok, rows} =
      Store.raw_query(
        "SELECT source_id, target_id, relation FROM edges WHERE workspace_id = ?1 AND relation = ?2",
        [ws, predicate]
      )

    assert Enum.any?(rows, fn [s, t, r] -> s == source and t == target and r == predicate end),
           "expected edge in SQLite, got: #{inspect(rows)}"
  end
end
