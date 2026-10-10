defmodule OptimalEngine.Memory.Store.ETS do
  @moduledoc """
  ETS-backed storage implementation for OptimalEngine.Memory.

  Stores entries in an ETS table keyed by `{collection, key}`.
  Optionally persists to disk as JSON files for durability across restarts.

  ## Configuration

      config :optimal_engine, OptimalEngine.Memory.Store.ETS,
        persist: true,
        persist_path: "~/.miosa/store"
  """

  use GenServer

  @behaviour OptimalEngine.Memory.Store

  @table :optimal_engine_memory_store
  @collections_table :optimal_engine_memory_collections

  # --- Client API (behaviour callbacks) ---

  @impl OptimalEngine.Memory.Store
  @spec put(String.t(), String.t(), term(), map()) :: :ok | {:error, term()}
  def put(collection, key, value, metadata \\ %{}) do
    now = DateTime.utc_now()

    full_metadata =
      Map.merge(
        %{created_at: now, updated_at: now, access_count: 0, tags: []},
        metadata
      )

    entry = %{key: key, value: value, metadata: full_metadata}
    :ets.insert(@table, {{collection, key}, entry})
    :ets.insert(@collections_table, {collection, true})

    if persist?(), do: GenServer.cast(__MODULE__, {:persist, collection})

    :ok
  end

  @impl OptimalEngine.Memory.Store
  @spec get(String.t(), String.t()) ::
          {:ok, OptimalEngine.Memory.Store.entry()} | {:error, :not_found}
  def get(collection, key) do
    case :ets.lookup(@table, {collection, key}) do
      [{{^collection, ^key}, entry}] ->
        updated =
          update_in(entry, [:metadata, :access_count], &((&1 || 0) + 1))
          |> put_in([:metadata, :updated_at], DateTime.utc_now())

        :ets.insert(@table, {{collection, key}, updated})
        {:ok, updated}

      [] ->
        {:error, :not_found}
    end
  end

  @impl OptimalEngine.Memory.Store
  @spec delete(String.t(), String.t()) :: :ok
  def delete(collection, key) do
    :ets.delete(@table, {collection, key})

    if persist?(), do: GenServer.cast(__MODULE__, {:persist, collection})

    :ok
  end

  @impl OptimalEngine.Memory.Store
  @spec list(String.t(), keyword()) :: {:ok, [OptimalEngine.Memory.Store.entry()]}
  def list(collection, opts \\ []) do
    limit = Keyword.get(opts, :limit)

    entries =
      :ets.match_object(@table, {{collection, :_}, :_})
      |> Enum.map(fn {_key, entry} -> entry end)
      |> Enum.sort_by(& &1.metadata.updated_at, {:desc, DateTime})

    entries = if limit, do: Enum.take(entries, limit), else: entries

    {:ok, entries}
  end

  @impl OptimalEngine.Memory.Store
  @spec search(String.t(), String.t()) :: {:ok, [OptimalEngine.Memory.Store.entry()]}
  def search(collection, query) do
    query_down = String.downcase(query)
    terms = String.split(query_down)

    {:ok, all} = list(collection)

    matches =
      Enum.filter(all, fn entry ->
        haystack =
          [
            to_searchable(entry.key),
            to_searchable(entry.value),
            Enum.join(Map.get(entry.metadata, :tags, []), " ")
          ]
          |> Enum.join(" ")
          |> String.downcase()

        Enum.any?(terms, &String.contains?(haystack, &1))
      end)

    {:ok, matches}
  end

  @impl OptimalEngine.Memory.Store
  @spec collections() :: {:ok, [String.t()]}
  def collections do
    cols =
      :ets.tab2list(@collections_table)
      |> Enum.map(fn {col, _} -> col end)
      |> Enum.sort()

    {:ok, cols}
  end

  # --- GenServer ---

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @impl GenServer
  def init(_opts) do
    @table |> create_table()
    @collections_table |> create_table()

    if persist?(), do: load_from_disk()

    {:ok, %{}}
  end

  @impl GenServer
  def handle_cast({:persist, collection}, state) do
    persist_collection(collection)
    {:noreply, state}
  end

  # --- Private ---

  defp create_table(name) do
    case :ets.info(name) do
      :undefined ->
        :ets.new(name, [:set, :public, :named_table, read_concurrency: true])

      _ ->
        name
    end
  end

  defp persist? do
    config()[:persist] == true
  end

  defp config do
    Application.get_env(:optimal_engine, __MODULE__, [])
  end

  defp persist_path do
    config()[:persist_path]
    |> Kernel.||("~/.miosa/store")
    |> Path.expand()
  end

  defp persist_collection(collection) do
    dir = persist_path()
    File.mkdir_p!(dir)

    {:ok, entries} = list(collection)

    data =
      Enum.map(entries, fn entry ->
        entry
        |> put_in([:metadata, :created_at], DateTime.to_iso8601(entry.metadata.created_at))
        |> put_in([:metadata, :updated_at], DateTime.to_iso8601(entry.metadata.updated_at))
      end)

    path = Path.join(dir, "#{collection}.json")
    File.write!(path, Jason.encode!(data, pretty: true))
  end

  defp load_from_disk do
    dir = persist_path()

    if File.dir?(dir) do
      dir
      |> File.ls!()
      |> Enum.filter(&String.ends_with?(&1, ".json"))
      |> Enum.each(fn file ->
        collection = String.trim_trailing(file, ".json")

        path = Path.join(dir, file)
        data = path |> File.read!() |> Jason.decode!(keys: :atoms)

        Enum.each(data, fn entry ->
          metadata =
            entry.metadata
            |> Map.update!(:created_at, &parse_datetime!/1)
            |> Map.update!(:updated_at, &parse_datetime!/1)

          restored = %{key: entry.key, value: entry.value, metadata: metadata}
          :ets.insert(@table, {{collection, entry.key}, restored})
          :ets.insert(@collections_table, {collection, true})
        end)
      end)
    end
  end

  defp parse_datetime!(dt) when is_binary(dt) do
    {:ok, parsed, _} = DateTime.from_iso8601(dt)
    parsed
  end

  defp parse_datetime!(%DateTime{} = dt), do: dt

  defp to_searchable(val) when is_binary(val), do: val
  defp to_searchable(val) when is_atom(val), do: Atom.to_string(val)
  defp to_searchable(val) when is_number(val), do: to_string(val)
  defp to_searchable(val), do: inspect(val)
end
