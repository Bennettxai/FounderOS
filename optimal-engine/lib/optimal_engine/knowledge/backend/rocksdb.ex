defmodule OptimalEngine.Knowledge.Backend.RocksDB do
  @moduledoc """
  RocksDB-backed persistent knowledge store.

  Uses 6 column families for quad indexing:
  - Graph-scoped: gspo, gpos, gosp (for queries within a named graph)
  - Cross-graph: spog, posg, ospg (for queries across all graphs)

  Keys are 32-byte binary: 4 × 8-byte big-endian dictionary-encoded IDs.
  Values are empty binaries — the key IS the quad.

  The `"default"` column family is required by RocksDB but unused for data.

  ## Dictionary

  Term-to-ID mapping is handled by `OptimalEngine.Knowledge.Dictionary`, a GenServer started
  per store instance. The process is started by `init/2` and terminated by
  `terminate/1`. All encode/decode operations go through this process.

  ## Availability

  This module compiles unconditionally but requires the `:rocksdb` NIF (provided
  by the `exrocksdb` hex package) to be installed at runtime. Use
  `Code.ensure_loaded?(:rocksdb)` to guard test execution or feature flags.
  """

  @behaviour OptimalEngine.Knowledge.Backend
  @compile {:no_warn_undefined,
            [
              {:rocksdb, :batch, 0},
              {:rocksdb, :batch_delete, 3},
              {:rocksdb, :batch_put, 4},
              {:rocksdb, :close, 1},
              {:rocksdb, :get, 4},
              {:rocksdb, :iterator, 3},
              {:rocksdb, :iterator_close, 1},
              {:rocksdb, :iterator_move, 2},
              {:rocksdb, :open, 3},
              {:rocksdb, :release_batch, 1},
              {:rocksdb, :write_batch, 3}
            ]}

  alias OptimalEngine.Knowledge.Dictionary

  @default_graph "default"

  # Column family names — order matters for :rocksdb.open/3 with CF descriptors.
  # "default" must be first; RocksDB requires it in the descriptor list.
  # Names are stored as Elixir binaries for Map keys; the NIF receives charlists.
  @cf_names ["default", "gspo", "gpos", "gosp", "spog", "posg", "ospg"]

  defstruct [:db, :cf_handles, :dict_pid, :path, :store_id]

  # ---------------------------------------------------------------------------
  # Backend callbacks
  # ---------------------------------------------------------------------------

  @impl true
  def init(store_id, opts) do
    path = Keyword.get(opts, :path, "/tmp/miosa_knowledge_rocksdb/#{store_id}")

    with :ok <- File.mkdir_p(path),
         {:ok, dict_pid} <- Dictionary.start_link([]),
         {:ok, db, cf_handles} <- open_db(path) do
      cf_map = Enum.zip(@cf_names, cf_handles) |> Map.new()

      {:ok,
       %__MODULE__{
         db: db,
         cf_handles: cf_map,
         dict_pid: dict_pid,
         path: path,
         store_id: store_id
       }}
    else
      {:error, reason} -> {:error, reason}
      error -> {:error, error}
    end
  end

  @impl true
  def assert(state, s, p, o) do
    assert(state, @default_graph, s, p, o)
  end

  @impl true
  def assert(state, g, s, p, o) do
    with {:ok, g_id} <- Dictionary.encode(state.dict_pid, g),
         {:ok, s_id} <- Dictionary.encode(state.dict_pid, s),
         {:ok, p_id} <- Dictionary.encode(state.dict_pid, p),
         {:ok, o_id} <- Dictionary.encode(state.dict_pid, o),
         {:ok, batch} <- :rocksdb.batch() do
      # Graph-scoped indices
      :rocksdb.batch_put(batch, cf(state, "gspo"), encode_key([g_id, s_id, p_id, o_id]), "")
      :rocksdb.batch_put(batch, cf(state, "gpos"), encode_key([g_id, p_id, o_id, s_id]), "")
      :rocksdb.batch_put(batch, cf(state, "gosp"), encode_key([g_id, o_id, s_id, p_id]), "")

      # Cross-graph indices
      :rocksdb.batch_put(batch, cf(state, "spog"), encode_key([s_id, p_id, o_id, g_id]), "")
      :rocksdb.batch_put(batch, cf(state, "posg"), encode_key([p_id, o_id, s_id, g_id]), "")
      :rocksdb.batch_put(batch, cf(state, "ospg"), encode_key([o_id, s_id, p_id, g_id]), "")

      :ok = :rocksdb.write_batch(state.db, batch, [])
      :rocksdb.release_batch(batch)
      {:ok, state}
    end
  end

  @impl true
  def assert_many(state, triples) do
    Enum.reduce_while(triples, {:ok, state}, fn triple, {:ok, acc} ->
      result =
        case triple do
          {g, s, p, o} -> assert(acc, g, s, p, o)
          {s, p, o} -> assert(acc, s, p, o)
        end

      case result do
        {:ok, _} = ok -> {:cont, ok}
        error -> {:halt, error}
      end
    end)
  end

  @impl true
  def retract(state, s, p, o) do
    retract(state, @default_graph, s, p, o)
  end

  @impl true
  def retract(state, g, s, p, o) do
    # Encode all terms. If any term has never been seen, it can't be in the DB,
    # so we can skip the delete entirely. We use encode/2 here rather than a
    # hypothetical "lookup-only" function because Dictionary.encode is idempotent
    # — calling it on an unknown term inserts it. We instead check whether the
    # specific key exists before issuing the batch delete.
    with {:ok, g_id} <- Dictionary.encode(state.dict_pid, g),
         {:ok, s_id} <- Dictionary.encode(state.dict_pid, s),
         {:ok, p_id} <- Dictionary.encode(state.dict_pid, p),
         {:ok, o_id} <- Dictionary.encode(state.dict_pid, o) do
      gspo_key = encode_key([g_id, s_id, p_id, o_id])

      # Check whether the quad actually exists before issuing deletes.
      # An absent key means there is nothing to retract.
      case :rocksdb.get(state.db, cf(state, "gspo"), gspo_key, []) do
        {:ok, _} ->
          {:ok, batch} = :rocksdb.batch()

          :rocksdb.batch_delete(batch, cf(state, "gspo"), encode_key([g_id, s_id, p_id, o_id]))
          :rocksdb.batch_delete(batch, cf(state, "gpos"), encode_key([g_id, p_id, o_id, s_id]))
          :rocksdb.batch_delete(batch, cf(state, "gosp"), encode_key([g_id, o_id, s_id, p_id]))
          :rocksdb.batch_delete(batch, cf(state, "spog"), encode_key([s_id, p_id, o_id, g_id]))
          :rocksdb.batch_delete(batch, cf(state, "posg"), encode_key([p_id, o_id, s_id, g_id]))
          :rocksdb.batch_delete(batch, cf(state, "ospg"), encode_key([o_id, s_id, p_id, g_id]))

          :ok = :rocksdb.write_batch(state.db, batch, [])
          :rocksdb.release_batch(batch)
          {:ok, state}

        :not_found ->
          {:ok, state}

        {:error, reason} ->
          {:error, reason}
      end
    end
  end

  @impl true
  def query(state, pattern) do
    g = Keyword.get(pattern, :graph)
    s = Keyword.get(pattern, :subject)
    p = Keyword.get(pattern, :predicate)
    o = Keyword.get(pattern, :object)

    results = do_query(state, g, s, p, o)
    {:ok, results}
  end

  @impl true
  def count(state) do
    {:ok, count_cf(state, "gspo")}
  end

  @impl true
  def sparql(_state, _query) do
    {:error, :sparql_not_supported}
  end

  @impl true
  def terminate(state) do
    :rocksdb.close(state.db)
    if Process.alive?(state.dict_pid), do: GenServer.stop(state.dict_pid)
    :ok
  end

  # ---------------------------------------------------------------------------
  # Private — DB open
  # ---------------------------------------------------------------------------

  defp open_db(path) do
    db_opts = [create_if_missing: true, create_missing_column_families: true]

    # CF names must be charlists per the rocksdb Erlang NIF type spec: cf_descriptor() :: {string(), cf_options()}
    cf_descriptors = Enum.map(@cf_names, fn name -> {to_charlist(name), []} end)
    :rocksdb.open(to_charlist(path), db_opts, cf_descriptors)
  end

  # ---------------------------------------------------------------------------
  # Private — key encoding
  # ---------------------------------------------------------------------------

  defp cf(state, name), do: Map.fetch!(state.cf_handles, name)

  defp encode_key(ids) do
    for id <- ids, into: <<>>, do: <<id::unsigned-big-integer-size(64)>>
  end

  defp decode_key(<<a::64-big, b::64-big, c::64-big, d::64-big>>), do: [a, b, c, d]

  defp encode_key_partial(ids) do
    for id <- ids, into: <<>>, do: <<id::unsigned-big-integer-size(64)>>
  end

  # ---------------------------------------------------------------------------
  # Private — query dispatch
  # ---------------------------------------------------------------------------

  defp do_query(state, g, s, p, o) do
    {g_id, s_id, p_id, o_id} = encode_query_terms(state, g, s, p, o)
    {cf_name, prefix, decode_fn} = select_index(g_id, s_id, p_id, o_id)
    scan_cf(state, cf_name, prefix, decode_fn)
  end

  # Encode bound query terms. Returns nil for unbound, :unknown when the term
  # has never been seen (so no results can match).
  defp encode_query_terms(state, g, s, p, o) do
    encode = fn
      nil ->
        nil

      term ->
        case Dictionary.encode(state.dict_pid, term) do
          {:ok, id} -> id
          _ -> :unknown
        end
    end

    {encode.(g), encode.(s), encode.(p), encode.(o)}
  end

  # ---------------------------------------------------------------------------
  # Index selection: {cf_name, prefix_binary | :no_results, decode_fn}
  # decode_fn: [a, b, c, d] (key components in CF order) -> {s, p, o}
  # ---------------------------------------------------------------------------

  # :unknown in any bound position => no possible results
  defp select_index(:unknown, _, _, _), do: {"gspo", :no_results, &gspo_decode/1}
  defp select_index(_, :unknown, _, _), do: {"gspo", :no_results, &gspo_decode/1}
  defp select_index(_, _, :unknown, _), do: {"gspo", :no_results, &gspo_decode/1}
  defp select_index(_, _, _, :unknown), do: {"gspo", :no_results, &gspo_decode/1}

  # Graph-scoped — g bound
  defp select_index(g, s, p, o) when g != nil and s != nil and p != nil and o != nil do
    {"gspo", encode_key([g, s, p, o]), &gspo_decode/1}
  end

  defp select_index(g, s, p, nil) when g != nil and s != nil and p != nil do
    {"gspo", encode_key_partial([g, s, p]), &gspo_decode/1}
  end

  defp select_index(g, s, nil, nil) when g != nil and s != nil do
    {"gspo", encode_key_partial([g, s]), &gspo_decode/1}
  end

  defp select_index(g, nil, p, o) when g != nil and p != nil and o != nil do
    {"gpos", encode_key_partial([g, p, o]), &gpos_decode/1}
  end

  defp select_index(g, nil, p, nil) when g != nil and p != nil do
    {"gpos", encode_key_partial([g, p]), &gpos_decode/1}
  end

  defp select_index(g, s, nil, o) when g != nil and s != nil and o != nil do
    {"gosp", encode_key_partial([g, o, s]), &gosp_decode/1}
  end

  defp select_index(g, nil, nil, o) when g != nil and o != nil do
    {"gosp", encode_key_partial([g, o]), &gosp_decode/1}
  end

  defp select_index(g, nil, nil, nil) when g != nil do
    {"gspo", encode_key_partial([g]), &gspo_decode/1}
  end

  # Cross-graph — g unbound
  defp select_index(nil, s, p, o) when s != nil and p != nil and o != nil do
    {"spog", encode_key_partial([s, p, o]), &spog_decode/1}
  end

  defp select_index(nil, s, p, nil) when s != nil and p != nil do
    {"spog", encode_key_partial([s, p]), &spog_decode/1}
  end

  defp select_index(nil, s, nil, nil) when s != nil do
    {"spog", encode_key_partial([s]), &spog_decode/1}
  end

  defp select_index(nil, nil, p, o) when p != nil and o != nil do
    {"posg", encode_key_partial([p, o]), &posg_decode/1}
  end

  defp select_index(nil, nil, p, nil) when p != nil do
    {"posg", encode_key_partial([p]), &posg_decode/1}
  end

  defp select_index(nil, s, nil, o) when s != nil and o != nil do
    {"ospg", encode_key_partial([o, s]), &ospg_decode/1}
  end

  defp select_index(nil, nil, nil, o) when o != nil do
    {"ospg", encode_key_partial([o]), &ospg_decode/1}
  end

  # Wildcard — full scan on gspo
  defp select_index(nil, nil, nil, nil) do
    {"gspo", <<>>, &gspo_decode/1}
  end

  # Decode: CF key order [a, b, c, d] → {s, p, o}
  defp gspo_decode([_g, s, p, o]), do: {s, p, o}
  defp gpos_decode([_g, p, o, s]), do: {s, p, o}
  defp gosp_decode([_g, o, s, p]), do: {s, p, o}
  defp spog_decode([s, p, o, _g]), do: {s, p, o}
  defp posg_decode([p, o, s, _g]), do: {s, p, o}
  defp ospg_decode([o, s, p, _g]), do: {s, p, o}

  # ---------------------------------------------------------------------------
  # Private — scan
  # ---------------------------------------------------------------------------

  defp scan_cf(_state, _cf_name, :no_results, _decode_fn), do: []

  defp scan_cf(state, cf_name, prefix, decode_fn) do
    cf_handle = cf(state, cf_name)
    {:ok, iter} = :rocksdb.iterator(state.db, cf_handle, [])

    seek_target = if byte_size(prefix) == 0, do: :first, else: {:seek, prefix}

    results =
      case :rocksdb.iterator_move(iter, seek_target) do
        {:ok, key, _val} ->
          if prefix_match?(key, prefix) do
            decoded = decode_result(key, state, decode_fn)
            collect_prefix(iter, prefix, state, decode_fn, [decoded])
          else
            []
          end

        {:error, :invalid_iterator} ->
          []
      end

    :rocksdb.iterator_close(iter)
    Enum.uniq(results)
  end

  defp collect_prefix(iter, prefix, state, decode_fn, acc) do
    case :rocksdb.iterator_move(iter, :next) do
      {:ok, key, _val} ->
        if prefix_match?(key, prefix) do
          decoded = decode_result(key, state, decode_fn)
          collect_prefix(iter, prefix, state, decode_fn, [decoded | acc])
        else
          acc
        end

      {:error, :invalid_iterator} ->
        acc
    end
  end

  # Empty prefix = full scan, matches everything.
  defp prefix_match?(_key, <<>>), do: true

  defp prefix_match?(key, prefix),
    do: :binary.longest_common_prefix([key, prefix]) == byte_size(prefix)

  defp decode_result(key, state, decode_fn) do
    ids = decode_key(key)
    {s_id, p_id, o_id} = decode_fn.(ids)

    {:ok, s} = Dictionary.decode(state.dict_pid, s_id)
    {:ok, p} = Dictionary.decode(state.dict_pid, p_id)
    {:ok, o} = Dictionary.decode(state.dict_pid, o_id)

    {s, p, o}
  end

  # ---------------------------------------------------------------------------
  # Private — count
  # ---------------------------------------------------------------------------

  defp count_cf(state, cf_name) do
    cf_handle = cf(state, cf_name)
    {:ok, iter} = :rocksdb.iterator(state.db, cf_handle, [])

    count =
      case :rocksdb.iterator_move(iter, :first) do
        {:ok, _, _} -> count_iter(iter, 1)
        {:error, :invalid_iterator} -> 0
      end

    :rocksdb.iterator_close(iter)
    count
  end

  defp count_iter(iter, acc) do
    case :rocksdb.iterator_move(iter, :next) do
      {:ok, _, _} -> count_iter(iter, acc + 1)
      {:error, :invalid_iterator} -> acc
    end
  end
end
