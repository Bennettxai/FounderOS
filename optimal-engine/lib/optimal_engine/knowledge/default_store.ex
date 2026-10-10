defmodule OptimalEngine.Knowledge.DefaultStore do
  @moduledoc """
  Starts and hydrates the singleton default Knowledge.Store.

  This GenServer starts a named `Knowledge.Store` (store_id = "default") under
  the Knowledge.Registry, then asynchronously loads all rows from the SQLite
  `edges` table so the in-memory triple store mirrors durable state after a
  restart.

  The hydration is fire-and-forget: failures are logged but never crash the
  supervisor. New edges written after startup are fed synchronously via
  `Graph.maybe_feed_triple_store/4` (called from `Graph.create_edges_for_context`
  and `Graph.assert_edge`).
  """

  use GenServer

  require Logger

  alias OptimalEngine.Graph
  alias OptimalEngine.Knowledge.Store, as: KnowledgeStore

  @store_id "default"
  @backend_modules %{
    "ets" => OptimalEngine.Knowledge.Backend.ETS,
    "mnesia" => OptimalEngine.Knowledge.Backend.Mnesia,
    "rocksdb" => OptimalEngine.Knowledge.Backend.RocksDB
  }

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @impl true
  def init(_opts) do
    {backend, backend_opts} = configured_backend()

    # Start the underlying Knowledge.Store registered under the default store_id.
    case KnowledgeStore.start_link(
           store_id: @store_id,
           backend: backend,
           backend_opts: backend_opts
         ) do
      {:ok, store_pid} ->
        # Hydrate asynchronously so supervision tree startup is not blocked.
        Task.start(fn -> hydrate(store_pid) end)
        {:ok, %{store_pid: store_pid}}

      {:error, {:already_started, store_pid}} ->
        Task.start(fn -> hydrate(store_pid) end)
        {:ok, %{store_pid: store_pid}}

      {:error, reason} ->
        Logger.warning("[DefaultStore] Could not start Knowledge.Store: #{inspect(reason)}")
        # Still start this GenServer so the supervisor tree doesn't fail — the
        # triple store is a secondary index and its absence is non-fatal.
        {:ok, %{store_pid: nil}}
    end
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  defp configured_backend do
    config = Application.get_env(:optimal_engine, :knowledge, [])

    requested =
      config
      |> Keyword.get(:backend, "ets")
      |> to_string()
      |> String.downcase()

    backend = Map.get(@backend_modules, requested, OptimalEngine.Knowledge.Backend.ETS)

    cond do
      requested == "rocksdb" and not Code.ensure_loaded?(:rocksdb) ->
        Logger.warning(
          "[DefaultStore] OPTIMAL_KNOWLEDGE_BACKEND=rocksdb requested but :rocksdb is unavailable; falling back to ETS"
        )

        {OptimalEngine.Knowledge.Backend.ETS, []}

      requested == "rocksdb" ->
        path =
          Keyword.get(config, :rocksdb_path, Path.join(File.cwd!(), ".optimal/knowledge-rocksdb"))

        Logger.info("[DefaultStore] Knowledge backend: RocksDB at #{path}")
        {backend, path: path}

      requested == "mnesia" ->
        Logger.info("[DefaultStore] Knowledge backend: Mnesia")
        {backend, copies: :ram_copies}

      true ->
        Logger.info("[DefaultStore] Knowledge backend: ETS")
        {backend, []}
    end
  end

  defp hydrate(store_pid) do
    store_name = {:via, Registry, {OptimalEngine.Knowledge.Registry, @store_id}}

    target =
      if is_pid(store_pid) and Process.alive?(store_pid),
        do: store_pid,
        else: store_name

    case Graph.hydrate_triple_store(target) do
      {:ok, count} ->
        Logger.info("[DefaultStore] Triple store hydrated with #{count} edges from SQLite")

      {:error, reason} ->
        Logger.warning("[DefaultStore] Triple store hydration failed: #{inspect(reason)}")
    end
  rescue
    error ->
      Logger.warning("[DefaultStore] Triple store hydration exception: #{inspect(error)}")
  end
end
