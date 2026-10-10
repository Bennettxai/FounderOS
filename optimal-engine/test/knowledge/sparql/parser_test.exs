defmodule OptimalEngine.Knowledge.SPARQL.ParserTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Knowledge.SPARQL.Parser

  describe "simple SELECT" do
    test "parses SELECT with single variable" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s ?p ?o }")

      assert ast.type == :select
      assert ast.variables == [{:var, "s"}]
      assert length(ast.where) == 1

      [{s, p, o}] = ast.where
      assert s == {:var, "s"}
      assert p == {:var, "p"}
      assert o == {:var, "o"}
    end

    test "parses SELECT with multiple variables" do
      {:ok, ast} = Parser.parse("SELECT ?s ?p ?o WHERE { ?s ?p ?o }")

      assert ast.variables == [{:var, "s"}, {:var, "p"}, {:var, "o"}]
    end

    test "parses SELECT *" do
      {:ok, ast} = Parser.parse("SELECT * WHERE { ?s ?p ?o }")

      assert ast.variables == [:all]
    end

    test "parses SELECT DISTINCT" do
      {:ok, ast} = Parser.parse("SELECT DISTINCT ?s WHERE { ?s ?p ?o }")

      assert ast.distinct == true
      assert ast.variables == [{:var, "s"}]
    end

    test "parses WHERE with URI terms" do
      {:ok, ast} =
        Parser.parse(
          ~s|SELECT ?name WHERE { <http://example.org/alice> <http://xmlns.com/foaf/0.1/name> ?name }|
        )

      [{s, p, o}] = ast.where
      assert s == {:uri, "http://example.org/alice"}
      assert p == {:uri, "http://xmlns.com/foaf/0.1/name"}
      assert o == {:var, "name"}
    end

    test "parses WHERE with multiple triple patterns" do
      query = """
      SELECT ?name ?age WHERE {
        ?s <name> ?name .
        ?s <age> ?age .
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert length(ast.where) == 2
    end

    test "parses WHERE with string literal objects" do
      {:ok, ast} = Parser.parse(~s|SELECT ?s WHERE { ?s <name> "Alice" }|)

      [{_s, _p, o}] = ast.where
      assert o == {:literal, "Alice"}
    end

    test "parses WHERE with integer literal objects" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s <age> 30 }")

      [{_s, _p, o}] = ast.where
      assert o == {:literal, 30}
    end

    test "implicit WHERE (no keyword)" do
      {:ok, ast} = Parser.parse("SELECT ?s { ?s ?p ?o }")

      assert ast.type == :select
      assert length(ast.where) == 1
    end
  end

  describe "FILTER" do
    test "parses simple comparison filter" do
      query = """
      SELECT ?s ?age WHERE {
        ?s <age> ?age .
        FILTER (?age > 30)
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.filters == [{:>, {:var, "age"}, {:literal, 30}}]
    end

    test "parses equality filter" do
      query = """
      SELECT ?s WHERE {
        ?s <status> ?status .
        FILTER (?status = "active")
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.filters == [{:=, {:var, "status"}, {:literal, "active"}}]
    end

    test "parses inequality filter" do
      query = """
      SELECT ?s WHERE {
        ?s <role> ?role .
        FILTER (?role != "guest")
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.filters == [{:!=, {:var, "role"}, {:literal, "guest"}}]
    end

    test "parses CONTAINS filter" do
      query = """
      SELECT ?s WHERE {
        ?s <name> ?name .
        FILTER (CONTAINS(?name, "ali"))
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert [{:CONTAINS, [_var, _lit]}] = ast.filters
    end

    test "parses REGEX filter" do
      query = """
      SELECT ?s WHERE {
        ?s <email> ?email .
        FILTER (REGEX(?email, "^admin"))
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert [{:REGEX, [_var, _pattern]}] = ast.filters
    end

    test "parses compound AND filter" do
      query = """
      SELECT ?s WHERE {
        ?s <age> ?age .
        FILTER (?age > 18 && ?age < 65)
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert [{:and, {:>, _, _}, {:<, _, _}}] = ast.filters
    end

    test "parses less-than-or-equal filter" do
      query = """
      SELECT ?s WHERE {
        ?s <score> ?score .
        FILTER (?score <= 100)
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert [{:<=, {:var, "score"}, {:literal, 100}}] = ast.filters
    end

    test "parses greater-than-or-equal filter" do
      query = """
      SELECT ?s WHERE {
        ?s <score> ?score .
        FILTER (?score >= 0)
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert [{:>=, {:var, "score"}, {:literal, 0}}] = ast.filters
    end
  end

  describe "OPTIONAL" do
    test "parses single OPTIONAL block" do
      query = """
      SELECT ?s ?name ?email WHERE {
        ?s <name> ?name .
        OPTIONAL { ?s <email> ?email }
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert length(ast.where) == 1
      assert length(ast.optionals) == 1

      [opt_patterns] = ast.optionals
      assert length(opt_patterns) == 1
    end

    test "parses multiple OPTIONAL blocks" do
      query = """
      SELECT ?s ?name ?email ?phone WHERE {
        ?s <name> ?name .
        OPTIONAL { ?s <email> ?email }
        OPTIONAL { ?s <phone> ?phone }
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert length(ast.optionals) == 2
    end
  end

  describe "PREFIX" do
    test "parses single PREFIX declaration" do
      query = """
      PREFIX foaf: <http://xmlns.com/foaf/0.1/>
      SELECT ?name WHERE { ?s foaf:name ?name }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.prefixes == %{"foaf:" => "http://xmlns.com/foaf/0.1/"}

      [{_s, p, _o}] = ast.where
      assert p == {:uri, "http://xmlns.com/foaf/0.1/name"}
    end

    test "parses multiple PREFIX declarations" do
      query = """
      PREFIX foaf: <http://xmlns.com/foaf/0.1/>
      PREFIX ex: <http://example.org/>
      SELECT ?name WHERE { ex:alice foaf:name ?name }
      """

      {:ok, ast} = Parser.parse(query)

      assert Map.has_key?(ast.prefixes, "foaf:")
      assert Map.has_key?(ast.prefixes, "ex:")

      [{s, p, _o}] = ast.where
      assert s == {:uri, "http://example.org/alice"}
      assert p == {:uri, "http://xmlns.com/foaf/0.1/name"}
    end

    test "resolves prefixed URIs in WHERE clause" do
      query = """
      PREFIX schema: <http://schema.org/>
      SELECT ?name WHERE {
        ?person schema:name ?name .
        ?person schema:age ?age .
      }
      """

      {:ok, ast} = Parser.parse(query)

      [{_s1, p1, _o1}, {_s2, p2, _o2}] = ast.where
      assert p1 == {:uri, "http://schema.org/name"}
      assert p2 == {:uri, "http://schema.org/age"}
    end
  end

  describe "LIMIT and OFFSET" do
    test "parses LIMIT" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s ?p ?o } LIMIT 10")

      assert ast.limit == 10
    end

    test "parses OFFSET" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s ?p ?o } OFFSET 5")

      assert ast.offset == 5
    end

    test "parses LIMIT and OFFSET together" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s ?p ?o } LIMIT 10 OFFSET 20")

      assert ast.limit == 10
      assert ast.offset == 20
    end

    test "parses OFFSET before LIMIT" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s ?p ?o } OFFSET 5 LIMIT 25")

      assert ast.offset == 5
      assert ast.limit == 25
    end
  end

  describe "ORDER BY" do
    test "parses ORDER BY with variable" do
      {:ok, ast} = Parser.parse("SELECT ?s ?name WHERE { ?s <name> ?name } ORDER BY ?name")

      assert ast.order_by == [{:asc, {:var, "name"}}]
    end

    test "parses ORDER BY ASC" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s <age> ?age } ORDER BY ASC(?age)")

      assert ast.order_by == [{:asc, {:var, "age"}}]
    end

    test "parses ORDER BY DESC" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s <age> ?age } ORDER BY DESC(?age)")

      assert ast.order_by == [{:desc, {:var, "age"}}]
    end

    test "parses ORDER BY with LIMIT" do
      {:ok, ast} = Parser.parse("SELECT ?s WHERE { ?s <age> ?age } ORDER BY DESC(?age) LIMIT 5")

      assert ast.order_by == [{:desc, {:var, "age"}}]
      assert ast.limit == 5
    end
  end

  describe "INSERT DATA" do
    test "parses simple INSERT DATA" do
      query = """
      INSERT DATA {
        <http://example.org/alice> <http://xmlns.com/foaf/0.1/name> "Alice" .
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.type == :insert_data
      assert length(ast.triples) == 1

      [{s, p, o}] = ast.triples
      assert s == {:uri, "http://example.org/alice"}
      assert p == {:uri, "http://xmlns.com/foaf/0.1/name"}
      assert o == {:literal, "Alice"}
    end

    test "parses INSERT DATA with multiple triples" do
      query = """
      INSERT DATA {
        <ex:alice> <knows> <ex:bob> .
        <ex:alice> <name> "Alice" .
        <ex:bob> <name> "Bob" .
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.type == :insert_data
      assert length(ast.triples) == 3
    end

    test "parses INSERT DATA with PREFIX" do
      query = """
      PREFIX foaf: <http://xmlns.com/foaf/0.1/>
      INSERT DATA {
        <http://example.org/alice> foaf:name "Alice" .
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.type == :insert_data
      [{_s, p, _o}] = ast.triples
      assert p == {:uri, "http://xmlns.com/foaf/0.1/name"}
    end
  end

  describe "DELETE DATA" do
    test "parses simple DELETE DATA" do
      query = """
      DELETE DATA {
        <http://example.org/alice> <http://xmlns.com/foaf/0.1/name> "Alice" .
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.type == :delete_data
      assert length(ast.triples) == 1
    end

    test "parses DELETE DATA with multiple triples" do
      query = """
      DELETE DATA {
        <ex:alice> <knows> <ex:bob> .
        <ex:alice> <knows> <ex:carol> .
      }
      """

      {:ok, ast} = Parser.parse(query)

      assert ast.type == :delete_data
      assert length(ast.triples) == 2
    end
  end

  describe "GRAPH clause parsing" do
    test "parses GRAPH with URI" do
      query = ~s|SELECT ?s WHERE { GRAPH <http://example.org/g1> { ?s <knows> <bob> . } }|
      {:ok, ast} = Parser.parse(query)
      assert [{:graph_pattern, {:uri, "http://example.org/g1"}, {_, _, _}}] = ast.where
    end

    test "parses GRAPH with variable" do
      query = ~s|SELECT ?g ?s WHERE { GRAPH ?g { ?s <knows> <bob> } }|
      {:ok, ast} = Parser.parse(query)
      assert [{:graph_pattern, {:var, "g"}, _}] = ast.where
    end

    test "parses GRAPH with multiple patterns inside" do
      query =
        ~s|SELECT ?s ?name WHERE { GRAPH <http://example.org/g1> { ?s <knows> <bob> . ?s <name> ?name } }|

      {:ok, ast} = Parser.parse(query)
      assert length(ast.where) == 2

      assert Enum.all?(ast.where, fn
               {:graph_pattern, _, _} -> true
               _ -> false
             end)
    end

    test "mixes GRAPH and non-GRAPH patterns" do
      query =
        ~s|SELECT ?s ?name WHERE { ?s <role> "admin" . GRAPH <http://example.org/g1> { ?s <knows> <bob> } }|

      {:ok, ast} = Parser.parse(query)
      assert length(ast.where) == 2

      assert Enum.any?(ast.where, fn
               {:graph_pattern, _, _} -> true
               _ -> false
             end)

      assert Enum.any?(ast.where, fn
               {_, _, _} = t -> not match?({:graph_pattern, _, _}, t)
               _ -> false
             end)
    end
  end

  describe "error handling" do
    test "returns error for empty input" do
      assert {:error, _} = Parser.parse("")
    end

    test "returns error for invalid keyword" do
      assert {:error, _} = Parser.parse("FROBNICATE ?s WHERE { ?s ?p ?o }")
    end

    test "returns error for unterminated string" do
      assert {:error, _} = Parser.parse(~s|SELECT ?s WHERE { ?s <name> "unclosed }|)
    end

    test "returns error for unterminated URI" do
      assert {:error, _} = Parser.parse("SELECT ?s WHERE { <unclosed ?p ?o }")
    end
  end

  describe "tokenizer" do
    test "tokenizes variables" do
      tokens = Parser.tokenize("?subject ?predicate ?object")

      assert tokens == [
               {:var, "subject"},
               {:var, "predicate"},
               {:var, "object"}
             ]
    end

    test "tokenizes URIs" do
      tokens = Parser.tokenize("<http://example.org/alice>")
      assert tokens == [{:uri, "http://example.org/alice"}]
    end

    test "tokenizes keywords" do
      tokens = Parser.tokenize("SELECT WHERE FILTER OPTIONAL LIMIT")

      assert tokens == [
               {:keyword, "SELECT"},
               {:keyword, "WHERE"},
               {:keyword, "FILTER"},
               {:keyword, "OPTIONAL"},
               {:keyword, "LIMIT"}
             ]
    end

    test "tokenizes integers" do
      tokens = Parser.tokenize("42 100")
      assert tokens == [{:integer, 42}, {:integer, 100}]
    end

    test "tokenizes string literals" do
      tokens = Parser.tokenize(~s{"hello world"})
      assert tokens == [{:string_literal, "hello world"}]
    end

    test "tokenizes comparison operators" do
      tokens = Parser.tokenize("!= <= >=")

      assert tokens == [
               {:symbol, "!="},
               {:symbol, "<="},
               {:symbol, ">="}
             ]
    end

    test "skips comments" do
      tokens = Parser.tokenize("SELECT # this is a comment\n?s")
      assert tokens == [{:keyword, "SELECT"}, {:var, "s"}]
    end
  end
end
