defmodule OptimalEngine.Knowledge.OptimizerTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Backend.ETS
  alias OptimalEngine.Knowledge.{Optimizer, Stats, SPARQL}

  # ---------------------------------------------------------------------------
  # Test data
  #
  # Skewed predicate distribution:
  #   "type"    — 6 triples (most common)
  #   "name"    — 4 triples
  #   "manages" — 2 triples
  #   "ceo"     — 1 triple  (rarest)
  # ---------------------------------------------------------------------------

  setup do
    store_id = "opt_test_#{:erlang.unique_integer([:positive])}"
    {:ok, state} = ETS.init(store_id, [])

    triples = [
      {"alice", "type", "Person"},
      {"bob", "type", "Person"},
      {"carol", "type", "Person"},
      {"dave", "type", "Person"},
      {"eve", "type", "Person"},
      {"frank", "type", "Person"},
      {"alice", "name", "Alice"},
      {"bob", "name", "Bob"},
      {"carol", "name", "Carol"},
      {"dave", "name", "Dave"},
      {"alice", "manages", "bob"},
      {"alice", "manages", "carol"},
      {"alice", "ceo", "true"}
    ]

    {:ok, state} = ETS.assert_many(state, triples)
    on_exit(fn -> ETS.terminate(state) end)
    %{state: state}
  end

  # ---------------------------------------------------------------------------
  # Stats.collect/2
  # ---------------------------------------------------------------------------

  describe "Stats.collect/2" do
    test "counts total triples", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      assert stats.total == 13
    end

    test "builds predicate histogram", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      assert stats.predicate_counts["type"] == 6
      assert stats.predicate_counts["name"] == 4
      assert stats.predicate_counts["manages"] == 2
      assert stats.predicate_counts["ceo"] == 1
    end

    test "counts distinct subjects", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      # alice, bob, carol, dave, eve, frank
      assert stats.subject_count == 6
    end

    test "counts distinct objects", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      # Person, Alice, Bob, Carol, Dave, bob, carol, true
      assert stats.object_count > 0
    end
  end

  # ---------------------------------------------------------------------------
  # Stats.estimate_cardinality/2
  # ---------------------------------------------------------------------------

  describe "Stats.estimate_cardinality/2" do
    test "wildcard pattern estimates total triples", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      assert Stats.estimate_cardinality([], stats) == 13
    end

    test "predicate-only uses histogram frequency", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      assert trunc(Stats.estimate_cardinality([predicate: "type"], stats)) == 6
      assert trunc(Stats.estimate_cardinality([predicate: "ceo"], stats)) == 1
    end

    test "all-bound pattern returns 1", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)

      estimate =
        Stats.estimate_cardinality(
          [subject: "alice", predicate: "name", object: "Alice"],
          stats
        )

      assert trunc(estimate) == 1
    end

    test "rare predicate estimates lower than common predicate", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      type_est = Stats.estimate_cardinality([predicate: "type"], stats)
      ceo_est = Stats.estimate_cardinality([predicate: "ceo"], stats)
      assert ceo_est < type_est
    end
  end

  # ---------------------------------------------------------------------------
  # Optimizer.reorder/2
  # ---------------------------------------------------------------------------

  describe "Optimizer.reorder/2" do
    test "places rare predicate before common predicate", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)

      # "ceo" appears 1 time, "type" appears 6 times.
      # With only variable subjects and literal objects, P-only cardinality
      # is used when the object variable is unbound.
      # ceo estimate (1) < type estimate (6), so ceo must come before type.
      patterns = [
        {{:var, "s"}, {:uri, "type"}, {:var, "t"}},
        {{:var, "s"}, {:uri, "ceo"}, {:var, "c"}}
      ]

      reordered = Optimizer.reorder(patterns, stats)
      preds = Enum.map(reordered, fn {_, {:uri, p}, _} -> p end)
      ceo_pos = Enum.find_index(preds, &(&1 == "ceo"))
      type_pos = Enum.find_index(preds, &(&1 == "type"))
      assert ceo_pos < type_pos, "Expected ceo (rare) before type (common), got: #{inspect(preds)}"
    end

    test "single pattern is unchanged", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)
      pattern = {{:var, "s"}, {:uri, "type"}, {:literal, "Person"}}
      assert Optimizer.reorder([pattern], stats) == [pattern]
    end

    test "returns same patterns (no additions or removals)", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)

      patterns = [
        {{:var, "s"}, {:uri, "type"}, {:literal, "Person"}},
        {{:var, "s"}, {:uri, "ceo"}, {:literal, "true"}},
        {{:var, "s"}, {:uri, "name"}, {:var, "name"}}
      ]

      reordered = Optimizer.reorder(patterns, stats)
      assert length(reordered) == length(patterns)
      assert Enum.sort(reordered) == Enum.sort(patterns)
    end

    test "handles all-variable patterns without raising", %{state: state} do
      {:ok, stats} = Stats.collect(ETS, state)

      patterns = [
        {{:var, "s"}, {:var, "p"}, {:var, "o"}},
        {{:var, "s"}, {:uri, "ceo"}, {:literal, "true"}}
      ]

      reordered = Optimizer.reorder(patterns, stats)
      assert length(reordered) == 2
    end
  end

  # ---------------------------------------------------------------------------
  # Optimizer.optimize/3 — AST-level integration
  # ---------------------------------------------------------------------------

  describe "Optimizer.optimize/3" do
    test "annotates AST with :plan key", %{state: state} do
      {:ok, ast} =
        SPARQL.parse("""
        SELECT ?s WHERE {
          ?s <type> "Person" .
          ?s <name> ?n .
          ?s <manages> ?o .
          ?o <type> "Person" .
        }
        """)

      optimized = Optimizer.optimize(ast, ETS, state)
      assert Map.has_key?(optimized, :plan)
      assert optimized.plan in [:trie_join, :nested_loop]
    end

    test "non-select AST is returned unchanged", %{state: state} do
      {:ok, ast} = SPARQL.parse("INSERT DATA { <a> <b> <c> }")
      assert Optimizer.optimize(ast, ETS, state) == ast
    end

    test "pattern count preserved after optimization", %{state: state} do
      {:ok, ast} =
        SPARQL.parse("""
        SELECT ?s ?n WHERE {
          ?s <type> "Person" .
          ?s <name> ?n .
          ?s <ceo> "true" .
        }
        """)

      optimized = Optimizer.optimize(ast, ETS, state)
      assert length(optimized.where) == length(ast.where)
    end
  end

  # ---------------------------------------------------------------------------
  # End-to-end: optimized query == unoptimized query results
  # ---------------------------------------------------------------------------

  describe "query result correctness" do
    test "optimized and unoptimized paths return identical result sets", %{state: state} do
      query = """
      SELECT ?s ?name WHERE {
        ?s <type> "Person" .
        ?s <name> ?name .
        ?s <manages> ?o .
      }
      """

      # Unoptimized path: parse then execute directly (no optimizer)
      {:ok, ast} = SPARQL.parse(query)
      {:ok, unoptimized} = SPARQL.execute(ast, ETS, state)

      # Optimized path: full query/3 pipeline
      {:ok, optimized} = SPARQL.query(query, ETS, state)

      assert Enum.sort_by(optimized, &Map.get(&1, "s")) ==
               Enum.sort_by(unoptimized, &Map.get(&1, "s"))
    end

    test "query with only common predicates still returns correct results", %{state: state} do
      query = """
      SELECT ?s ?name WHERE {
        ?s <type> "Person" .
        ?s <name> ?name .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)
      # alice, bob, carol, dave all have both type=Person and a name
      assert length(results) == 4
      names = Enum.map(results, &Map.get(&1, "name")) |> Enum.sort()
      assert names == ["Alice", "Bob", "Carol", "Dave"]
    end

    test "query with rare predicate filter returns correct results", %{state: state} do
      query = """
      SELECT ?s WHERE {
        ?s <ceo> "true" .
        ?s <type> "Person" .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)
      assert length(results) == 1
      assert hd(results)["s"] == "alice"
    end
  end
end
