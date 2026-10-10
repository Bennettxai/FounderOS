defmodule OptimalEngine.Knowledge do
  @moduledoc """
  MIOSA Knowledge — semantic knowledge graph engine for the MIOSA ecosystem.

  Provides a pluggable backend behaviour over a triple-store model, with native
  SPARQL 1.1 execution and OWL 2 RL reasoning built entirely in pure Elixir.

  - Pluggable storage: ETS (dev/test), Mnesia (distributed/production)
  - Agent context injection: query the knowledge graph from agent loops
  - Signal integration: knowledge mutations emit telemetry events
  - Namespace-scoped stores: per-tenant, per-agent, or shared

  ## Quick Start

      # Open a store with ETS backend (default in dev/test)
      {:ok, store} = OptimalEngine.Knowledge.open("agent_context")

      # Assert facts
      :ok = OptimalEngine.Knowledge.assert(store, {"user:alice", "knows", "user:bob"})
      :ok = OptimalEngine.Knowledge.assert(store, {"user:alice", "role", "admin"})

      # Query
      {:ok, results} = OptimalEngine.Knowledge.query(store, subject: "user:alice")

      # SPARQL — executed by the native pure-Elixir SPARQL engine
      {:ok, results} = OptimalEngine.Knowledge.sparql(store, "SELECT ?o WHERE { <user:alice> <knows> ?o }")

      # Inject into agent context
      context = OptimalEngine.Knowledge.Context.for_agent(store, agent_id: "agent_1")
  """

  alias OptimalEngine.Knowledge.Store

  @type store :: GenServer.server()
  @type triple :: {String.t(), String.t(), String.t()}
  @type quad :: {String.t(), String.t(), String.t(), String.t()}
  @type pattern :: [
          subject: String.t(),
          predicate: String.t(),
          object: String.t(),
          graph: String.t()
        ]

  @doc """
  Opens a named knowledge store.

  ## Options

  - `:backend` - Backend module (default: `OptimalEngine.Knowledge.Backend.ETS`)
  - `:backend_opts` - Options passed to the backend
  - `:name` - Process name (default: derived from store_id)
  """
  @spec open(String.t(), keyword()) :: {:ok, store()} | {:error, term()}
  def open(store_id, opts \\ []) do
    Store.start_link(Keyword.merge(opts, store_id: store_id))
  end

  @doc "Close a store and release resources."
  @spec close(store()) :: :ok
  def close(store) do
    Store.stop(store)
  end

  @doc """
  Assert a triple or quad into the store.

  Accepts a 3-tuple `{subject, predicate, object}` (stored in the default graph)
  or a 4-tuple `{graph, subject, predicate, object}` (stored in the named graph).
  """
  @spec assert(store(), triple() | quad()) :: :ok | {:error, term()}
  def assert(store, {g, s, p, o})
      when is_binary(g) and is_binary(s) and is_binary(p) and is_binary(o) do
    Store.assert(store, g, s, p, o)
  end

  def assert(store, {s, p, o}) do
    Store.assert(store, s, p, o)
  end

  @doc "Assert multiple triples in a batch."
  @spec assert_many(store(), [triple()]) :: :ok | {:error, term()}
  def assert_many(store, triples) when is_list(triples) do
    Store.assert_many(store, triples)
  end

  @doc """
  Retract a triple or quad from the store.

  Accepts a 3-tuple `{subject, predicate, object}` (retracts from the default graph)
  or a 4-tuple `{graph, subject, predicate, object}` (retracts from the named graph).
  """
  @spec retract(store(), triple() | quad()) :: :ok | {:error, term()}
  def retract(store, {g, s, p, o})
      when is_binary(g) and is_binary(s) and is_binary(p) and is_binary(o) do
    Store.retract(store, g, s, p, o)
  end

  def retract(store, {s, p, o}) do
    Store.retract(store, s, p, o)
  end

  @doc """
  Query the store with a pattern.

  Returns all triples matching the given constraints.
  Unspecified positions are treated as wildcards.

  ## Examples

      # All triples about alice
      OptimalEngine.Knowledge.query(store, subject: "user:alice")

      # All "knows" relationships
      OptimalEngine.Knowledge.query(store, predicate: "knows")

      # Exact match
      OptimalEngine.Knowledge.query(store, subject: "user:alice", predicate: "knows")
  """
  @spec query(store(), pattern()) :: {:ok, [triple()]} | {:error, term()}
  def query(store, pattern \\ []) do
    Store.query(store, pattern)
  end

  @doc """
  Execute a SPARQL query against the store.

  All backends are supported: backends that implement the optional `sparql/2` callback
  use their native execution path; all others fall back to the built-in pure-Elixir
  SPARQL engine. Returns `{:error, :sparql_not_supported}` only if execution cannot
  be determined at runtime.
  """
  @spec sparql(store(), String.t()) :: {:ok, term()} | {:error, term()}
  def sparql(store, query_string) when is_binary(query_string) do
    Store.sparql(store, query_string)
  end

  @doc "Return the number of triples in the store."
  @spec count(store()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(store) do
    Store.count(store)
  end

  @doc "Check if a specific triple exists."
  @spec exists?(store(), triple()) :: boolean()
  def exists?(store, {s, p, o}) do
    case Store.query(store, subject: s, predicate: p, object: o) do
      {:ok, [_ | _]} -> true
      _ -> false
    end
  end
end
