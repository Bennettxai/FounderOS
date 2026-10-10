defmodule OptimalEngine.Knowledge.Backend.ETSTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Backend.ETS

  setup do
    {:ok, state} = ETS.init("test_#{:erlang.unique_integer([:positive])}", [])
    on_exit(fn -> ETS.terminate(state) end)
    %{state: state}
  end

  describe "assert/4" do
    test "inserts a triple into all three indices", %{state: state} do
      {:ok, state} = ETS.assert(state, "alice", "knows", "bob")
      {:ok, results} = ETS.query(state, subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "deduplicates identical triples", %{state: state} do
      {:ok, state} = ETS.assert(state, "alice", "knows", "bob")
      {:ok, state} = ETS.assert(state, "alice", "knows", "bob")
      {:ok, count} = ETS.count(state)
      assert count == 1
    end
  end

  describe "assert_many/2" do
    test "inserts multiple triples", %{state: state} do
      triples = [
        {"alice", "knows", "bob"},
        {"alice", "knows", "carol"},
        {"bob", "knows", "carol"}
      ]

      {:ok, state} = ETS.assert_many(state, triples)
      {:ok, count} = ETS.count(state)
      assert count == 3
    end
  end

  describe "retract/4" do
    test "removes a triple from all indices", %{state: state} do
      {:ok, state} = ETS.assert(state, "alice", "knows", "bob")
      {:ok, state} = ETS.retract(state, "alice", "knows", "bob")
      {:ok, count} = ETS.count(state)
      assert count == 0
    end

    test "is a no-op for nonexistent triples", %{state: state} do
      {:ok, _state} = ETS.retract(state, "alice", "knows", "bob")
    end
  end

  describe "query/2 pattern matching" do
    setup %{state: state} do
      triples = [
        {"alice", "knows", "bob"},
        {"alice", "knows", "carol"},
        {"alice", "role", "admin"},
        {"bob", "knows", "carol"},
        {"bob", "role", "user"},
        {"carol", "knows", "alice"}
      ]

      {:ok, state} = ETS.assert_many(state, triples)
      %{state: state}
    end

    test "subject only", %{state: state} do
      {:ok, results} = ETS.query(state, subject: "alice")
      assert length(results) == 3
      assert {"alice", "knows", "bob"} in results
      assert {"alice", "knows", "carol"} in results
      assert {"alice", "role", "admin"} in results
    end

    test "predicate only", %{state: state} do
      {:ok, results} = ETS.query(state, predicate: "knows")
      assert length(results) == 4
    end

    test "object only", %{state: state} do
      {:ok, results} = ETS.query(state, object: "carol")
      assert length(results) == 2
      assert {"alice", "knows", "carol"} in results
      assert {"bob", "knows", "carol"} in results
    end

    test "subject + predicate", %{state: state} do
      {:ok, results} = ETS.query(state, subject: "alice", predicate: "knows")
      assert length(results) == 2
    end

    test "predicate + object", %{state: state} do
      {:ok, results} = ETS.query(state, predicate: "knows", object: "carol")
      assert length(results) == 2
    end

    test "subject + object", %{state: state} do
      {:ok, results} = ETS.query(state, subject: "alice", object: "carol")
      assert length(results) == 1
      assert {"alice", "knows", "carol"} in results
    end

    test "exact match (all bound)", %{state: state} do
      {:ok, results} = ETS.query(state, subject: "alice", predicate: "knows", object: "bob")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "no match returns empty", %{state: state} do
      {:ok, results} = ETS.query(state, subject: "nobody")
      assert results == []
    end

    test "wildcard returns all", %{state: state} do
      {:ok, results} = ETS.query(state, [])
      assert length(results) == 6
    end
  end

  describe "sparql/2" do
    test "returns not supported", %{state: state} do
      assert {:error, :sparql_not_supported} = ETS.sparql(state, "SELECT ?s WHERE { ?s ?p ?o }")
    end
  end

  describe "quad store (named graphs)" do
    test "assert with graph and query scoped to graph", %{state: state} do
      {:ok, state} = ETS.assert(state, "graph1", "alice", "knows", "bob")
      {:ok, results} = ETS.query(state, graph: "graph1", subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "cross-graph query returns all", %{state: state} do
      {:ok, state} = ETS.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = ETS.assert(state, "g2", "alice", "knows", "carol")
      {:ok, results} = ETS.query(state, subject: "alice")
      assert length(results) == 2
    end

    test "default graph for triple assert", %{state: state} do
      {:ok, state} = ETS.assert(state, "alice", "knows", "bob")
      {:ok, results} = ETS.query(state, graph: "default", subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "same triple in different graphs counted separately", %{state: state} do
      {:ok, state} = ETS.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = ETS.assert(state, "g2", "alice", "knows", "bob")
      {:ok, count} = ETS.count(state)
      assert count == 2
    end

    test "graph-scoped query does not return triples from other graphs", %{state: state} do
      {:ok, state} = ETS.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = ETS.assert(state, "g2", "alice", "knows", "carol")
      {:ok, results} = ETS.query(state, graph: "g1")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "retract from named graph removes only that quad", %{state: state} do
      {:ok, state} = ETS.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = ETS.assert(state, "g2", "alice", "knows", "bob")
      {:ok, state} = ETS.retract(state, "g1", "alice", "knows", "bob")
      {:ok, count} = ETS.count(state)
      assert count == 1
      {:ok, results} = ETS.query(state, graph: "g2", subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "cross-graph predicate query returns results from all graphs", %{state: state} do
      {:ok, state} = ETS.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = ETS.assert(state, "g2", "carol", "knows", "dave")
      {:ok, results} = ETS.query(state, predicate: "knows")
      assert length(results) == 2
    end
  end
end
