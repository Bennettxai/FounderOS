defmodule OptimalEngine.Knowledge.SPARQL do
  @moduledoc """
  Native SPARQL query engine for MIOSA knowledge graphs.

  Pure Elixir implementation — no NIFs, no Rust dependencies. Covers the 80%
  of SPARQL 1.1 that agents actually use:

  - `SELECT` queries with variables, `DISTINCT`, and `*`
  - `WHERE` clauses with basic graph patterns (BGP)
  - `FILTER` with comparisons (`=`, `!=`, `<`, `>`, `CONTAINS`, `REGEX`)
  - `OPTIONAL` patterns (left outer join)
  - `LIMIT` and `OFFSET`
  - `ORDER BY` with `ASC`/`DESC`
  - `PREFIX` declarations with automatic URI expansion
  - `INSERT DATA` and `DELETE DATA` for simple updates

  ## Usage

      # Parse a SPARQL query into an AST
      {:ok, ast} = OptimalEngine.Knowledge.SPARQL.parse("SELECT ?s WHERE { ?s ?p ?o } LIMIT 10")

      # Execute against a backend
      {:ok, results} = OptimalEngine.Knowledge.SPARQL.execute(ast, backend_module, backend_state)

      # One-shot: parse + execute
      {:ok, results} = OptimalEngine.Knowledge.SPARQL.query("SELECT ?s WHERE { ?s ?p ?o }", backend, state)

  ## AST Structure

  The parser produces structured maps:

      %{
        type: :select,
        prefixes: %{"foaf:" => "http://xmlns.com/foaf/0.1/"},
        distinct: false,
        variables: [{:var, "s"}, {:var, "name"}],
        where: [
          {{:var, "s"}, {:uri, "http://xmlns.com/foaf/0.1/name"}, {:var, "name"}}
        ],
        filters: [{:>, {:var, "age"}, {:literal, 30}}],
        optionals: [],
        order_by: [{:asc, {:var, "name"}}],
        limit: 10,
        offset: nil
      }
  """

  alias OptimalEngine.Knowledge.SPARQL.{Parser, Executor}
  alias OptimalEngine.Knowledge.Optimizer

  @doc """
  Parse a SPARQL query string into an AST.

  Returns `{:ok, ast}` on success, `{:error, reason}` on parse failure.

  ## Examples

      iex> OptimalEngine.Knowledge.SPARQL.parse("SELECT ?s WHERE { ?s ?p ?o }")
      {:ok, %{type: :select, variables: [{:var, "s"}], ...}}

      iex> OptimalEngine.Knowledge.SPARQL.parse("INVALID STUFF")
      {:error, "Expected SELECT, INSERT, or DELETE, got ..."}
  """
  @spec parse(String.t()) :: {:ok, map()} | {:error, String.t()}
  def parse(query_string) when is_binary(query_string) do
    Parser.parse(query_string)
  end

  @doc """
  Execute a parsed SPARQL AST against a backend.

  ## Parameters
  - `ast` — Parsed AST from `parse/1`
  - `backend` — Module implementing `OptimalEngine.Knowledge.Backend`
  - `backend_state` — Opaque backend state from `backend.init/2`

  ## Returns
  - `{:ok, [%{var_name => value, ...}]}` for SELECT queries
  - `{:ok, :inserted, count}` for INSERT DATA
  - `{:ok, :deleted, count}` for DELETE DATA
  """
  @spec execute(map(), module(), term()) :: Executor.result()
  def execute(ast, backend, backend_state) do
    Executor.execute(ast, backend, backend_state)
  end

  @doc """
  Parse and execute a SPARQL query string in one step.

  Convenience function combining `parse/1` and `execute/3`.

  ## Examples

      {:ok, results} = OptimalEngine.Knowledge.SPARQL.query(
        "SELECT ?name WHERE { ?s <knows> ?name }",
        OptimalEngine.Knowledge.Backend.ETS,
        ets_state
      )
  """
  @spec query(String.t(), module(), term()) :: Executor.result() | {:error, String.t()}
  def query(query_string, backend, backend_state) when is_binary(query_string) do
    case parse(query_string) do
      {:ok, ast} ->
        optimized_ast = Optimizer.optimize(ast, backend, backend_state)
        execute(optimized_ast, backend, backend_state)

      {:error, _} = err ->
        err
    end
  end
end
