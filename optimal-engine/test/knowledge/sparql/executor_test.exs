defmodule OptimalEngine.Knowledge.SPARQL.ExecutorTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.Backend.ETS
  alias OptimalEngine.Knowledge.SPARQL

  setup do
    store_id = "sparql_test_#{:erlang.unique_integer([:positive])}"
    {:ok, state} = ETS.init(store_id, [])

    # Seed test data
    triples = [
      {"alice", "name", "Alice"},
      {"alice", "age", "30"},
      {"alice", "knows", "bob"},
      {"alice", "knows", "carol"},
      {"alice", "email", "alice@example.com"},
      {"alice", "role", "admin"},
      {"bob", "name", "Bob"},
      {"bob", "age", "25"},
      {"bob", "knows", "carol"},
      {"bob", "role", "user"},
      {"carol", "name", "Carol"},
      {"carol", "age", "35"},
      {"carol", "role", "admin"},
      {"carol", "phone", "555-1234"},
      {"dave", "name", "Dave"},
      {"dave", "age", "28"},
      {"dave", "role", "guest"}
    ]

    {:ok, state} = ETS.assert_many(state, triples)
    on_exit(fn -> ETS.terminate(state) end)

    %{state: state}
  end

  describe "basic BGP execution" do
    test "single pattern with one variable", %{state: state} do
      {:ok, results} = SPARQL.query("SELECT ?name WHERE { <alice> <name> ?name }", ETS, state)

      assert results == [%{"name" => "Alice"}]
    end

    test "single pattern with all variables (wildcard)", %{state: state} do
      {:ok, results} = SPARQL.query("SELECT * WHERE { <alice> <knows> ?o }", ETS, state)

      objects = Enum.map(results, & &1["o"])
      assert "bob" in objects
      assert "carol" in objects
      assert length(results) == 2
    end

    test "returns empty for no matches", %{state: state} do
      {:ok, results} = SPARQL.query("SELECT ?o WHERE { <nobody> <knows> ?o }", ETS, state)

      assert results == []
    end
  end

  describe "multi-pattern join" do
    test "joins two patterns on shared variable", %{state: state} do
      query = """
      SELECT ?name WHERE {
        <alice> <knows> ?person .
        ?person <name> ?name .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert "Bob" in names
      assert "Carol" in names
      assert length(results) == 2
    end

    test "three-pattern join", %{state: state} do
      query = """
      SELECT ?friend_name ?friend_role WHERE {
        <alice> <knows> ?friend .
        ?friend <name> ?friend_name .
        ?friend <role> ?friend_role .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      assert length(results) == 2

      bob_result = Enum.find(results, &(&1["friend_name"] == "Bob"))
      assert bob_result["friend_role"] == "user"

      carol_result = Enum.find(results, &(&1["friend_name"] == "Carol"))
      assert carol_result["friend_role"] == "admin"
    end

    test "join with fixed predicate and object", %{state: state} do
      query = """
      SELECT ?person ?name WHERE {
        ?person <role> <admin> .
        ?person <name> ?name .
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert "Alice" in names
      assert "Carol" in names
      assert length(results) == 2
    end
  end

  describe "FILTER application" do
    test "numeric greater-than filter", %{state: state} do
      query = """
      SELECT ?name WHERE {
        ?s <name> ?name .
        ?s <age> ?age .
        FILTER (?age > 28)
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert "Alice" in names
      assert "Carol" in names
      refute "Bob" in names
      refute "Dave" in names
    end

    test "equality filter", %{state: state} do
      query = """
      SELECT ?s WHERE {
        ?s <role> ?role .
        FILTER (?role = "admin")
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      subjects = Enum.map(results, & &1["s"])
      assert "alice" in subjects
      assert "carol" in subjects
      assert length(results) == 2
    end

    test "inequality filter", %{state: state} do
      query = """
      SELECT ?s WHERE {
        ?s <role> ?role .
        FILTER (?role != "guest")
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      subjects = Enum.map(results, & &1["s"])
      refute "dave" in subjects
      assert length(results) > 0
    end

    test "CONTAINS filter", %{state: state} do
      query = """
      SELECT ?name WHERE {
        ?s <name> ?name .
        FILTER (CONTAINS(?name, "ol"))
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      assert results == [%{"name" => "Carol"}]
    end

    test "REGEX filter", %{state: state} do
      query = """
      SELECT ?name WHERE {
        ?s <name> ?name .
        FILTER (REGEX(?name, "^[AB]"))
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert "Alice" in names
      assert "Bob" in names
      refute "Carol" in names
    end

    test "compound AND filter", %{state: state} do
      query = """
      SELECT ?name WHERE {
        ?s <name> ?name .
        ?s <age> ?age .
        FILTER (?age >= 28 && ?age <= 30)
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert "Alice" in names
      assert "Dave" in names
      refute "Bob" in names
      refute "Carol" in names
    end
  end

  describe "OPTIONAL (left outer join)" do
    test "includes optional bindings when present", %{state: state} do
      query = """
      SELECT ?name ?email WHERE {
        ?s <name> ?name .
        ?s <role> <admin> .
        OPTIONAL { ?s <email> ?email }
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      alice = Enum.find(results, &(&1["name"] == "Alice"))
      assert alice["email"] == "alice@example.com"

      carol = Enum.find(results, &(&1["name"] == "Carol"))
      refute Map.has_key?(carol, "email")
    end

    test "returns base bindings when optional has no match", %{state: state} do
      query = """
      SELECT ?name ?phone WHERE {
        ?s <name> ?name .
        ?s <role> <user> .
        OPTIONAL { ?s <phone> ?phone }
      }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)

      assert length(results) == 1
      bob = hd(results)
      assert bob["name"] == "Bob"
      refute Map.has_key?(bob, "phone")
    end
  end

  describe "LIMIT and OFFSET" do
    test "LIMIT restricts result count", %{state: state} do
      query = "SELECT ?s WHERE { ?s <name> ?name } LIMIT 2"
      {:ok, results} = SPARQL.query(query, ETS, state)

      assert length(results) == 2
    end

    test "OFFSET skips results", %{state: state} do
      query = "SELECT ?name WHERE { ?s <name> ?name } ORDER BY ?name"

      {:ok, all_results} = SPARQL.query(query <> " LIMIT 100", ETS, state)
      {:ok, offset_results} = SPARQL.query(query <> " OFFSET 2", ETS, state)

      assert length(offset_results) == length(all_results) - 2
    end

    test "LIMIT and OFFSET together for pagination", %{state: state} do
      base_query = "SELECT ?name WHERE { ?s <name> ?name } ORDER BY ?name"

      {:ok, page1} = SPARQL.query(base_query <> " LIMIT 2 OFFSET 0", ETS, state)
      {:ok, page2} = SPARQL.query(base_query <> " LIMIT 2 OFFSET 2", ETS, state)

      assert length(page1) == 2
      assert length(page2) == 2

      page1_names = Enum.map(page1, & &1["name"])
      page2_names = Enum.map(page2, & &1["name"])
      assert MapSet.disjoint?(MapSet.new(page1_names), MapSet.new(page2_names))
    end
  end

  describe "ORDER BY" do
    test "ascending order", %{state: state} do
      query = "SELECT ?name WHERE { ?s <name> ?name } ORDER BY ASC(?name)"
      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert names == Enum.sort(names)
    end

    test "descending order", %{state: state} do
      query = "SELECT ?name WHERE { ?s <name> ?name } ORDER BY DESC(?name)"
      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert names == Enum.sort(names, :desc)
    end

    test "default order is ascending", %{state: state} do
      query = "SELECT ?name WHERE { ?s <name> ?name } ORDER BY ?name"
      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert names == Enum.sort(names)
    end

    test "ORDER BY with LIMIT", %{state: state} do
      query = "SELECT ?name WHERE { ?s <name> ?name } ORDER BY ?name LIMIT 2"
      {:ok, results} = SPARQL.query(query, ETS, state)

      names = Enum.map(results, & &1["name"])
      assert length(names) == 2
      assert names == Enum.sort(names)
    end
  end

  describe "DISTINCT" do
    test "removes duplicate bindings", %{state: state} do
      # Query for roles — alice and carol are both admin
      query = "SELECT DISTINCT ?role WHERE { ?s <role> ?role }"
      {:ok, results} = SPARQL.query(query, ETS, state)

      roles = Enum.map(results, & &1["role"])
      assert length(roles) == length(Enum.uniq(roles))
    end
  end

  describe "INSERT DATA execution" do
    test "inserts triples into backend", %{state: state} do
      query = """
      INSERT DATA {
        <eve> <name> "Eve" .
        <eve> <age> "22" .
      }
      """

      {:ok, :inserted, 2} = SPARQL.query(query, ETS, state)

      # Verify insertion
      {:ok, results} = SPARQL.query("SELECT ?name WHERE { <eve> <name> ?name }", ETS, state)
      assert results == [%{"name" => "Eve"}]
    end

    test "inserts with PREFIX resolution", %{state: state} do
      query = """
      PREFIX ex: <http://example.org/>
      INSERT DATA {
        ex:frank ex:name "Frank" .
      }
      """

      {:ok, :inserted, 1} = SPARQL.query(query, ETS, state)

      {:ok, results} =
        SPARQL.query(
          "SELECT ?name WHERE { <http://example.org/frank> <http://example.org/name> ?name }",
          ETS,
          state
        )

      assert results == [%{"name" => "Frank"}]
    end
  end

  describe "DELETE DATA execution" do
    test "deletes triples from backend", %{state: state} do
      # Verify alice knows bob before deletion
      {:ok, before} = SPARQL.query("SELECT ?o WHERE { <alice> <knows> ?o }", ETS, state)
      assert Enum.any?(before, &(&1["o"] == "bob"))

      query = """
      DELETE DATA {
        <alice> <knows> <bob> .
      }
      """

      {:ok, :deleted, 1} = SPARQL.query(query, ETS, state)

      # Verify deletion
      {:ok, after_del} = SPARQL.query("SELECT ?o WHERE { <alice> <knows> ?o }", ETS, state)
      refute Enum.any?(after_del, &(&1["o"] == "bob"))
    end

    test "deletes multiple triples", %{state: state} do
      query = """
      DELETE DATA {
        <alice> <knows> <bob> .
        <alice> <knows> <carol> .
      }
      """

      {:ok, :deleted, 2} = SPARQL.query(query, ETS, state)

      {:ok, results} = SPARQL.query("SELECT ?o WHERE { <alice> <knows> ?o }", ETS, state)
      assert results == []
    end
  end

  describe "PREFIX resolution in queries" do
    test "resolves prefixed URIs in SELECT patterns", %{state: state} do
      # Insert data with full URIs
      insert = """
      INSERT DATA {
        <http://example.org/person1> <http://schema.org/name> "Test Person" .
      }
      """

      {:ok, :inserted, 1} = SPARQL.query(insert, ETS, state)

      # Query with PREFIX
      query = """
      PREFIX schema: <http://schema.org/>
      PREFIX ex: <http://example.org/>
      SELECT ?name WHERE { ex:person1 schema:name ?name }
      """

      {:ok, results} = SPARQL.query(query, ETS, state)
      assert results == [%{"name" => "Test Person"}]
    end
  end

  describe "GRAPH clause" do
    test "GRAPH with URI scopes query to named graph", %{state: _state} do
      query = ~s|SELECT ?s ?p WHERE { GRAPH <http://example.org/graph1> { ?s ?p <bob> } }|
      {:ok, ast} = OptimalEngine.Knowledge.SPARQL.Parser.parse(query)
      assert ast.type == :select

      assert Enum.any?(ast.where, fn
               {:graph_pattern, {:uri, "http://example.org/graph1"}, _} -> true
               _ -> false
             end)
    end

    test "GRAPH with variable produces valid AST", %{state: _state} do
      query = ~s|SELECT ?g ?s WHERE { GRAPH ?g { ?s <knows> <bob> } }|
      {:ok, ast} = OptimalEngine.Knowledge.SPARQL.Parser.parse(query)

      assert Enum.any?(ast.where, fn
               {:graph_pattern, {:var, "g"}, _} -> true
               _ -> false
             end)
    end

    test "GRAPH URI constraint is passed to backend", %{state: state} do
      # The ETS backend ignores unknown keys in the query pattern, so this
      # returns normal results while confirming the executor doesn't crash.
      query = ~s|SELECT ?s WHERE { GRAPH <http://example.org/g1> { ?s <name> ?name } }|
      assert {:ok, _results} = SPARQL.query(query, ETS, state)
    end
  end

  describe "error handling" do
    test "returns error for invalid query", %{state: state} do
      assert {:error, _} = SPARQL.query("NOT A VALID QUERY", ETS, state)
    end
  end

  describe "edge cases" do
    test "handles empty WHERE clause", %{state: state} do
      {:ok, results} = SPARQL.query("SELECT ?s WHERE { }", ETS, state)
      assert results == [%{}]
    end

    test "handles query with only unbound variables matching nothing", %{state: state} do
      {:ok, results} =
        SPARQL.query("SELECT ?x WHERE { <nonexistent> <nothing> ?x }", ETS, state)

      assert results == []
    end
  end
end
