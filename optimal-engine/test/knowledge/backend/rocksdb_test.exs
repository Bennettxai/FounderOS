defmodule OptimalEngine.Knowledge.Backend.RocksDBTest do
  use ExUnit.Case, async: false

  @moduletag :rocksdb

  alias OptimalEngine.Knowledge.Backend.RocksDB

  setup do
    store_id = "test_#{:erlang.unique_integer([:positive])}"
    path = "/tmp/miosa_knowledge_rocksdb_test/#{store_id}"

    {:ok, state} = RocksDB.init(store_id, path: path)

    on_exit(fn ->
      RocksDB.terminate(state)
      File.rm_rf!(path)
    end)

    %{state: state}
  end

  describe "assert/4 (triple)" do
    test "inserts a triple and makes it queryable", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "bob")
      {:ok, results} = RocksDB.query(state, subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "deduplicates identical triples", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "bob")
      {:ok, count} = RocksDB.count(state)
      assert count == 1
    end
  end

  describe "assert/5 (quad)" do
    test "inserts a quad into a named graph", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "graph1", "alice", "knows", "bob")
      {:ok, results} = RocksDB.query(state, graph: "graph1", subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "same triple in two graphs are separate quads", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g2", "alice", "knows", "bob")
      {:ok, count} = RocksDB.count(state)
      assert count == 2
    end
  end

  describe "assert_many/2" do
    test "inserts multiple triples", %{state: state} do
      triples = [
        {"alice", "knows", "bob"},
        {"alice", "knows", "carol"},
        {"bob", "knows", "carol"}
      ]

      {:ok, state} = RocksDB.assert_many(state, triples)
      {:ok, count} = RocksDB.count(state)
      assert count == 3
    end

    test "accepts quads in the list", %{state: state} do
      statements = [
        {"g1", "alice", "knows", "bob"},
        {"g2", "carol", "knows", "dave"}
      ]

      {:ok, state} = RocksDB.assert_many(state, statements)
      {:ok, count} = RocksDB.count(state)
      assert count == 2
    end
  end

  describe "retract/4 (triple)" do
    test "removes a triple from all indices", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "bob")
      {:ok, state} = RocksDB.retract(state, "alice", "knows", "bob")
      {:ok, count} = RocksDB.count(state)
      assert count == 0
    end

    test "is a no-op for nonexistent triples", %{state: state} do
      assert {:ok, _state} = RocksDB.retract(state, "alice", "knows", "bob")
    end
  end

  describe "retract/5 (quad)" do
    test "removes only the specified graph's quad", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g2", "alice", "knows", "bob")
      {:ok, state} = RocksDB.retract(state, "g1", "alice", "knows", "bob")

      {:ok, count} = RocksDB.count(state)
      assert count == 1

      {:ok, results} = RocksDB.query(state, graph: "g2", subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "is a no-op when quad does not exist", %{state: state} do
      assert {:ok, _state} = RocksDB.retract(state, "g1", "alice", "knows", "nobody")
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

      {:ok, state} = RocksDB.assert_many(state, triples)
      %{state: state}
    end

    test "subject only", %{state: state} do
      {:ok, results} = RocksDB.query(state, subject: "alice")
      assert length(results) == 3
      assert {"alice", "knows", "bob"} in results
      assert {"alice", "knows", "carol"} in results
      assert {"alice", "role", "admin"} in results
    end

    test "predicate only", %{state: state} do
      {:ok, results} = RocksDB.query(state, predicate: "knows")
      assert length(results) == 4
    end

    test "object only", %{state: state} do
      {:ok, results} = RocksDB.query(state, object: "carol")
      assert length(results) == 2
      assert {"alice", "knows", "carol"} in results
      assert {"bob", "knows", "carol"} in results
    end

    test "subject + predicate", %{state: state} do
      {:ok, results} = RocksDB.query(state, subject: "alice", predicate: "knows")
      assert length(results) == 2
    end

    test "predicate + object", %{state: state} do
      {:ok, results} = RocksDB.query(state, predicate: "knows", object: "carol")
      assert length(results) == 2
    end

    test "subject + object", %{state: state} do
      {:ok, results} = RocksDB.query(state, subject: "alice", object: "carol")
      assert length(results) == 1
      assert {"alice", "knows", "carol"} in results
    end

    test "exact match (all bound)", %{state: state} do
      {:ok, results} = RocksDB.query(state, subject: "alice", predicate: "knows", object: "bob")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "no match returns empty list", %{state: state} do
      {:ok, results} = RocksDB.query(state, subject: "nobody")
      assert results == []
    end

    test "wildcard returns all triples", %{state: state} do
      {:ok, results} = RocksDB.query(state, [])
      assert length(results) == 6
    end
  end

  describe "query/2 named graphs" do
    test "graph-scoped query returns only that graph's triples", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g2", "alice", "knows", "carol")
      {:ok, results} = RocksDB.query(state, graph: "g1")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "cross-graph subject query returns all graphs", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g2", "alice", "knows", "carol")
      {:ok, results} = RocksDB.query(state, subject: "alice")
      assert length(results) == 2
    end

    test "triple assert lands in default graph", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "bob")
      {:ok, results} = RocksDB.query(state, graph: "default", subject: "alice")
      assert results == [{"alice", "knows", "bob"}]
    end

    test "cross-graph predicate query spans all graphs", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g2", "carol", "knows", "dave")
      {:ok, results} = RocksDB.query(state, predicate: "knows")
      assert length(results) == 2
    end

    test "graph-scoped subject+predicate query", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "carol")
      {:ok, state} = RocksDB.assert(state, "g2", "alice", "knows", "dave")
      {:ok, results} = RocksDB.query(state, graph: "g1", subject: "alice", predicate: "knows")
      assert length(results) == 2
      assert {"alice", "knows", "bob"} in results
      assert {"alice", "knows", "carol"} in results
    end

    test "graph-scoped predicate+object query uses gpos index", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "likes", "pizza")
      {:ok, state} = RocksDB.assert(state, "g1", "bob", "likes", "pizza")
      {:ok, state} = RocksDB.assert(state, "g2", "carol", "likes", "pizza")
      {:ok, results} = RocksDB.query(state, graph: "g1", predicate: "likes", object: "pizza")
      assert length(results) == 2
    end

    test "graph-scoped object query uses gosp index", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "g1", "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g1", "carol", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "g2", "dave", "knows", "bob")
      {:ok, results} = RocksDB.query(state, graph: "g1", object: "bob")
      assert length(results) == 2
    end
  end

  describe "sparql/2" do
    test "returns not supported", %{state: state} do
      assert {:error, :sparql_not_supported} =
               RocksDB.sparql(state, "SELECT ?s WHERE { ?s ?p ?o }")
    end
  end

  describe "count/1" do
    test "returns 0 for empty store", %{state: state} do
      {:ok, count} = RocksDB.count(state)
      assert count == 0
    end

    test "counts quads, not index entries", %{state: state} do
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "bob")
      {:ok, state} = RocksDB.assert(state, "alice", "knows", "carol")
      {:ok, count} = RocksDB.count(state)
      assert count == 2
    end
  end
end
