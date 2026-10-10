defmodule OptimalEngine.Knowledge.Backend do
  @moduledoc """
  MIOSA Knowledge — behaviour for pluggable knowledge store backends.

  Each backend implements the required callbacks for triple assertion, retraction,
  pattern queries, and triple counting. Backends may optionally implement `sparql/2`
  for native SPARQL execution; backends that do not will automatically use the
  built-in pure-Elixir SPARQL engine.

  Graph-aware backends may optionally implement `assert/5` and `retract/5` to
  support named graphs (quad store). Backends that do not implement these callbacks
  will have triples stored in the default graph implicitly.

  Implementations:
  - `OptimalEngine.Knowledge.Backend.ETS` — In-memory, zero-config. Default for dev/test.
  - `OptimalEngine.Knowledge.Backend.Mnesia` — Distributed across BEAM nodes. Default for production.
  """

  @type state :: term()
  @type triple :: {String.t(), String.t(), String.t()}
  @type quad :: {String.t(), String.t(), String.t(), String.t()}
  @type statement :: triple() | quad()
  @type pattern :: [
          subject: String.t(),
          predicate: String.t(),
          object: String.t(),
          graph: String.t()
        ]

  @doc "Initialize the backend. Returns opaque state."
  @callback init(store_id :: String.t(), opts :: keyword()) :: {:ok, state()} | {:error, term()}

  @doc "Assert a triple into the store."
  @callback assert(state(), subject :: String.t(), predicate :: String.t(), object :: String.t()) ::
              {:ok, state()} | {:error, term()}

  @doc "Assert a triple into a named graph."
  @callback assert(
              state(),
              graph :: String.t(),
              subject :: String.t(),
              predicate :: String.t(),
              object :: String.t()
            ) ::
              {:ok, state()} | {:error, term()}

  @doc "Assert multiple triples in a batch."
  @callback assert_many(state(), [triple()]) :: {:ok, state()} | {:error, term()}

  @doc "Retract a triple from the store."
  @callback retract(
              state(),
              subject :: String.t(),
              predicate :: String.t(),
              object :: String.t()
            ) ::
              {:ok, state()} | {:error, term()}

  @doc "Retract a triple from a named graph."
  @callback retract(
              state(),
              graph :: String.t(),
              subject :: String.t(),
              predicate :: String.t(),
              object :: String.t()
            ) ::
              {:ok, state()} | {:error, term()}

  @doc "Query triples matching a pattern."
  @callback query(state(), pattern()) :: {:ok, [triple()]}

  @doc "Count triples in the store."
  @callback count(state()) :: {:ok, non_neg_integer()}

  @doc "Execute a SPARQL query. Return :sparql_not_supported if not available."
  @callback sparql(state(), String.t()) :: {:ok, term()} | {:error, :sparql_not_supported | term()}

  @doc "Clean up resources."
  @callback terminate(state()) :: :ok

  @optional_callbacks [sparql: 2, assert: 5, retract: 5]
end
