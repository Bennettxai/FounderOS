defmodule OptimalEngine.Knowledge.Backend.RocksDB.Iterator do
  @moduledoc """
  Prefix-bounded iterator over a RocksDB column family.

  Provides seek/next/key/exhausted? protocol for use by the leapfrog triejoin
  engine (Phase 3). Wraps a native RocksDB iterator with prefix-bounded scanning.

  ## Lifecycle

      iter = Iterator.new(db, cf_handle, prefix, dict_pid)
      iter = Iterator.seek(iter, target_key)
      key  = Iterator.key(iter)
      iter = Iterator.next(iter)
      Iterator.close(iter)

  The iterator is positioned at the first key >= prefix on `new/4`. If no key
  matches the prefix, `exhausted?/1` returns `true` immediately.

  All operations on an exhausted iterator are no-ops and return the exhausted
  state unchanged.
  """

  @compile {:no_warn_undefined,
            [
              {:rocksdb, :iterator, 3},
              {:rocksdb, :iterator_close, 1},
              {:rocksdb, :iterator_move, 2}
            ]}

  defstruct [:db, :cf_handle, :iter, :prefix, :current, :exhausted, :dict_pid]

  @type t :: %__MODULE__{
          db: reference(),
          cf_handle: reference(),
          iter: reference(),
          prefix: binary(),
          current: binary() | nil,
          exhausted: boolean(),
          dict_pid: GenServer.server()
        }

  @doc """
  Open an iterator positioned at the first key with the given prefix.

  Pass an empty binary prefix (`<<>>`) for a full-scan iterator.
  """
  @spec new(reference(), reference(), binary(), GenServer.server()) :: t()
  def new(db, cf_handle, prefix, dict_pid) do
    {:ok, iter} = :rocksdb.iterator(db, cf_handle, [])

    seek_target = if byte_size(prefix) == 0, do: :first, else: {:seek, prefix}

    state = %__MODULE__{
      db: db,
      cf_handle: cf_handle,
      iter: iter,
      prefix: prefix,
      current: nil,
      exhausted: false,
      dict_pid: dict_pid
    }

    case :rocksdb.iterator_move(iter, seek_target) do
      {:ok, key, _val} ->
        if prefix_match?(key, prefix) do
          %{state | current: key}
        else
          %{state | exhausted: true}
        end

      {:error, :invalid_iterator} ->
        %{state | exhausted: true}
    end
  end

  @doc "Seek to the first key >= target_key that also matches the prefix."
  @spec seek(t(), binary()) :: t()
  def seek(%{exhausted: true} = state, _target), do: state

  def seek(state, target_key) do
    case :rocksdb.iterator_move(state.iter, {:seek, target_key}) do
      {:ok, key, _val} ->
        if prefix_match?(key, state.prefix) do
          %{state | current: key}
        else
          %{state | current: nil, exhausted: true}
        end

      {:error, :invalid_iterator} ->
        %{state | current: nil, exhausted: true}
    end
  end

  @doc "Advance to the next key within the prefix."
  @spec next(t()) :: t()
  def next(%{exhausted: true} = state), do: state

  def next(state) do
    case :rocksdb.iterator_move(state.iter, :next) do
      {:ok, key, _val} ->
        if prefix_match?(key, state.prefix) do
          %{state | current: key}
        else
          %{state | current: nil, exhausted: true}
        end

      {:error, :invalid_iterator} ->
        %{state | current: nil, exhausted: true}
    end
  end

  @doc "Return the current raw key binary, or nil if exhausted."
  @spec key(t()) :: binary() | nil
  def key(%{current: key}), do: key

  @doc "True when the iterator has moved past all matching keys."
  @spec exhausted?(t()) :: boolean()
  def exhausted?(%{exhausted: true}), do: true
  def exhausted?(_), do: false

  @doc "Close the underlying RocksDB iterator. Must be called to release resources."
  @spec close(t()) :: :ok
  def close(state) do
    :rocksdb.iterator_close(state.iter)
    :ok
  end

  # Empty prefix = full-scan, every key matches.
  defp prefix_match?(_key, <<>>), do: true

  defp prefix_match?(key, prefix),
    do: :binary.longest_common_prefix([key, prefix]) == byte_size(prefix)
end
