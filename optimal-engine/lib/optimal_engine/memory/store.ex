defmodule OptimalEngine.Memory.Store do
  @moduledoc """
  Storage backend behaviour for OptimalEngine.Memory.

  Defines the contract that all storage implementations must fulfill.
  The default implementation is `OptimalEngine.Memory.Store.ETS`.
  """

  @type collection :: String.t()
  @type key :: String.t()
  @type value :: term()
  @type metadata :: %{
          created_at: DateTime.t(),
          updated_at: DateTime.t(),
          access_count: non_neg_integer(),
          tags: [String.t()]
        }
  @type entry :: %{
          key: key(),
          value: value(),
          metadata: metadata()
        }
  @type opts :: keyword()

  @doc "Store a value under collection + key with metadata."
  @callback put(collection(), key(), value(), metadata()) :: :ok | {:error, term()}

  @doc "Retrieve an entry by collection + key."
  @callback get(collection(), key()) :: {:ok, entry()} | {:error, :not_found}

  @doc "Delete an entry by collection + key."
  @callback delete(collection(), key()) :: :ok

  @doc "List all entries in a collection. Supports opts like `limit`."
  @callback list(collection(), opts()) :: {:ok, [entry()]}

  @doc "Search entries in a collection matching query."
  @callback search(collection(), String.t()) :: {:ok, [entry()]}

  @doc "List all collection names."
  @callback collections() :: {:ok, [collection()]}

  @doc """
  Returns the configured store backend.

  Defaults to `OptimalEngine.Memory.Store.ETS`.
  """
  @spec backend() :: module()
  def backend do
    Application.get_env(:optimal_engine, :store_backend, OptimalEngine.Memory.Store.ETS)
  end

  @doc """
  Recall all memory content as a single string.

  Retrieves all entries from the default "memory" collection and concatenates
  their values. Used by `OptimalEngine.Memory.Cortex` for synthesis.
  """
  @spec recall() :: String.t()
  def recall do
    case backend().list("memory", []) do
      {:ok, entries} ->
        entries
        |> Enum.map(fn entry ->
          case entry do
            %{value: v} when is_binary(v) -> v
            %{value: v} -> inspect(v)
            _ -> ""
          end
        end)
        |> Enum.join("\n")

      _ ->
        ""
    end
  rescue
    _ -> ""
  end

  @doc """
  List all sessions with metadata. Delegates to `OptimalEngine.Memory.SessionStore.list_sessions/0`.
  """
  @spec list_sessions() :: [map()]
  def list_sessions do
    OptimalEngine.Memory.SessionStore.list_sessions()
  end

  @doc """
  Load a session's message history by ID. Delegates to `OptimalEngine.Memory.SessionStore.load/1`.
  """
  @spec load_session(String.t()) :: [map()]
  def load_session(session_id) do
    OptimalEngine.Memory.SessionStore.load(session_id)
  end
end
