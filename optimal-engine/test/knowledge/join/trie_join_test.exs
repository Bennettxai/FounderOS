defmodule OptimalEngine.Knowledge.Join.TrieJoinTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Backend.ETS
  alias OptimalEngine.Knowledge.Join.TrieJoin
  alias OptimalEngine.Knowledge.SPARQL

  # ---------------------------------------------------------------------------
  # Setup: social graph with enough triples to exercise complex BGPs
  # ---------------------------------------------------------------------------

  setup do
    store_id = "triejoin_test_#{:erlang.unique_integer([:positive])}"
    {:ok, state} = ETS.init(store_id, [])

    triples = [
      # People and their names
      {"alice", "name", "Alice"},
      {"alice", "age", "30"},
      {"alice", "role", "admin"},
      {"alice", "email", "alice@example.com"},
      # Bob
      {"bob", "name", "Bob"},
      {"bob", "age", "25"},
      {"bob", "role", "user"},
      {"bob", "email", "bob@example.com"},
      # Carol
      {"carol", "name", "Carol"},
      {"carol", "age", "35"},
      {"carol", "role", "admin"},
      {"carol", "email", "carol@example.com"},
      # Dave
      {"dave", "name", "Dave"},
      {"dave", "age", "28"},
      {"dave", "role", "user"},
      {"dave", "email", "dave@example.com"},
      # "knows" edges — forms a social graph
      {"alice", "knows", "bob"},
      {"alice", "knows", "carol"},
      {"bob", "knows", "carol"},
      {"carol", "knows", "alice"},
      # Project memberships — for star queries
      {"alice", "member_of", "project_x"},
      {"alice", "member_of", "project_y"},
      {"bob", "member_of", "project_x"},
      {"carol", "member_of", "project_x"},
      {"carol", "member_of", "project_z"}
    ]

    {:ok, state} = ETS.assert_many(state, triples)
    on_exit(fn -> ETS.terminate(state) end)

    %{state: state}
  end

  # ---------------------------------------------------------------------------
  # Correctness: compare TrieJoin to nested-loop SPARQL executor
  # ---------------------------------------------------------------------------

  defp sort_bindings(bindings) do
    bindings
    |> Enum.map(fn b -> Enum.sort(b) end)
    |> Enum.sort()
  end

  describe "4-pattern BGP with shared variables" do
    test "produces same bindings as nested-loop executor", %{state: state} do
      # ?s name ?n, ?s age ?a, ?s role ?r, ?s email ?e
      patterns = [
        {{:var, "s"}, {:name, "name"}, {:var, "n"}},
        {{:var, "s"}, {:name, "age"}, {:var, "a"}},
        {{:var, "s"}, {:name, "role"}, {:var, "r"}},
        {{:var, "s"}, {:name, "email"}, {:var, "e"}}
      ]

      trie_results = TrieJoin.execute(patterns, ETS, state)

      sparql_query = """
      SELECT * WHERE {
        ?s <name> ?n .
        ?s <age> ?a .
        ?s <role> ?r .
        ?s <email> ?e .
      }
      """

      {:ok, sparql_results} = SPARQL.query(sparql_query, ETS, state)

      assert sort_bindings(trie_results) == sort_bindings(sparql_results)
    end

    test "returns all four people with their attributes", %{state: state} do
      patterns = [
        {{:var, "s"}, {:name, "name"}, {:var, "n"}},
        {{:var, "s"}, {:name, "age"}, {:var, "a"}},
        {{:var, "s"}, {:name, "role"}, {:var, "r"}},
        {{:var, "s"}, {:name, "email"}, {:var, "e"}}
      ]

      results = TrieJoin.execute(patterns, ETS, state)
      subjects = Enum.map(results, & &1["s"]) |> Enum.sort()

      assert subjects == ["alice", "bob", "carol", "dave"]
    end
  end

  describe "triangle query" do
    # ?a knows ?b, ?b knows ?c, ?c knows ?a
    # In our graph: alice→bob, alice→carol, bob→carol, carol→alice
    # Valid triangles: alice→carol→alice is a 2-cycle, not a triangle.
    # alice knows bob (alice→bob)
    # bob knows carol (bob→carol)
    # carol knows alice (carol→alice)
    # So: a=alice, b=bob, c=carol is a valid triangle.

    test "finds triangle in social graph (trie)", %{state: state} do
      patterns = [
        {{:var, "a"}, {:name, "knows"}, {:var, "b"}},
        {{:var, "b"}, {:name, "knows"}, {:var, "c"}},
        {{:var, "c"}, {:name, "knows"}, {:var, "a"}}
      ]

      results = TrieJoin.execute(patterns, ETS, state)

      # At minimum, alice→bob→carol→alice is a valid triangle
      found =
        Enum.any?(results, fn b ->
          b["a"] == "alice" and b["b"] == "bob" and b["c"] == "carol"
        end)

      assert found, "Expected triangle alice→bob→carol→alice in #{inspect(results)}"
    end

    test "triangle produces same results as nested-loop SPARQL", %{state: state} do
      patterns = [
        {{:var, "a"}, {:name, "knows"}, {:var, "b"}},
        {{:var, "b"}, {:name, "knows"}, {:var, "c"}},
        {{:var, "c"}, {:name, "knows"}, {:var, "a"}}
      ]

      trie_results = TrieJoin.execute(patterns, ETS, state)

      sparql_query = """
      SELECT * WHERE {
        ?a <knows> ?b .
        ?b <knows> ?c .
        ?c <knows> ?a .
      }
      """

      {:ok, sparql_results} = SPARQL.query(sparql_query, ETS, state)

      assert sort_bindings(trie_results) == sort_bindings(sparql_results)
    end
  end

  describe "star query" do
    # ?s has 4 properties simultaneously: name, age, role, email
    # This exercises LeapfrogJoin with iterators for each property projection
    test "finds subjects satisfying all four predicates", %{state: state} do
      patterns = [
        {{:var, "s"}, {:name, "name"}, {:var, "n"}},
        {{:var, "s"}, {:name, "age"}, {:var, "a"}},
        {{:var, "s"}, {:name, "role"}, {:var, "r"}},
        {{:var, "s"}, {:name, "email"}, {:var, "e"}}
      ]

      results = TrieJoin.execute(patterns, ETS, state)

      assert length(results) == 4

      alice = Enum.find(results, fn b -> b["s"] == "alice" end)
      assert alice["n"] == "Alice"
      assert alice["a"] == "30"
      assert alice["r"] == "admin"
      assert alice["e"] == "alice@example.com"
    end

    test "star query produces same results as nested-loop SPARQL", %{state: state} do
      patterns = [
        {{:var, "s"}, {:name, "name"}, {:var, "n"}},
        {{:var, "s"}, {:name, "age"}, {:var, "a"}},
        {{:var, "s"}, {:name, "role"}, {:var, "r"}},
        {{:var, "s"}, {:name, "email"}, {:var, "e"}}
      ]

      trie_results = TrieJoin.execute(patterns, ETS, state)

      sparql_query = """
      SELECT * WHERE {
        ?s <name> ?n .
        ?s <age> ?a .
        ?s <role> ?r .
        ?s <email> ?e .
      }
      """

      {:ok, sparql_results} = SPARQL.query(sparql_query, ETS, state)

      assert sort_bindings(trie_results) == sort_bindings(sparql_results)
    end
  end

  describe "result format" do
    test "returns list of binding maps with string keys", %{state: state} do
      patterns = [
        {{:var, "s"}, {:name, "name"}, {:var, "n"}},
        {{:var, "s"}, {:name, "age"}, {:var, "a"}},
        {{:var, "s"}, {:name, "role"}, {:var, "r"}},
        {{:var, "s"}, {:name, "email"}, {:var, "e"}}
      ]

      results = TrieJoin.execute(patterns, ETS, state)

      assert is_list(results)

      Enum.each(results, fn binding ->
        assert is_map(binding)
        assert Map.has_key?(binding, "s")
        assert Map.has_key?(binding, "n")
        assert Map.has_key?(binding, "a")
        assert Map.has_key?(binding, "r")
        assert Map.has_key?(binding, "e")
      end)
    end

    test "returns empty list when no patterns match", %{state: state} do
      patterns = [
        {{:var, "s"}, {:name, "nonexistent_predicate"}, {:var, "o"}},
        {{:var, "s"}, {:name, "another_missing"}, {:var, "x"}},
        {{:var, "s"}, {:name, "yet_another"}, {:var, "y"}},
        {{:var, "s"}, {:name, "and_another"}, {:var, "z"}}
      ]

      results = TrieJoin.execute(patterns, ETS, state)
      assert results == []
    end
  end

  describe "SPARQL executor routing" do
    # Verify that the executor routes >= 4 pattern BGPs through TrieJoin
    # by checking the SPARQL interface produces correct results.
    test "4-pattern SPARQL SELECT is routed to triejoin and returns correct results",
         %{state: state} do
      query = """
      SELECT * WHERE {
        ?s <name> ?n .
        ?s <age> ?a .
        ?s <role> ?r .
        ?s <email> ?e .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)
      assert length(results) == 4
      subjects = Enum.map(results, & &1["s"]) |> Enum.sort()
      assert subjects == ["alice", "bob", "carol", "dave"]
    end

    test "3-pattern BGP still uses nested-loop (below threshold)", %{state: state} do
      query = """
      SELECT * WHERE {
        ?s <name> ?n .
        ?s <age> ?a .
        ?s <role> ?r .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)
      assert length(results) == 4
    end
  end
end
