defmodule OptimalEngine.Knowledge.Join.Iterator.RocksDB do
  @moduledoc """
  Wraps `OptimalEngine.Knowledge.Backend.RocksDB.Iterator` to implement the
  `OptimalEngine.Knowledge.Join.Iterator` behaviour for leapfrog triejoin.

  The underlying RocksDB iterator operates on raw binary keys. This wrapper
  delegates all operations to the inner iterator's existing seek/next/key/exhausted?
  functions, providing a uniform interface for the join engine regardless of
  backend.

  ## Usage

      iter = RocksDBIter.new(db, cf_handle, prefix, dict_pid)
      RocksDBIter.key(iter)             # => current binary key or nil
      iter = RocksDBIter.seek(iter, target_key)
      iter = RocksDBIter.next(iter)
      RocksDBIter.exhausted?(iter)      # => boolean
  """

  @behaviour OptimalEngine.Knowledge.Join.Iterator

  alias OptimalEngine.Knowledge.Backend.RocksDB.Iterator, as: RocksIter

  defstruct [:inner]

  @type t :: %__MODULE__{inner: RocksIter.t()}

  @doc "Open a RocksDB iterator positioned at the first key matching the given prefix."
  @spec new(reference(), reference(), binary(), GenServer.server()) :: t()
  def new(db, cf_handle, prefix, dict_pid) do
    inner = RocksIter.new(db, cf_handle, prefix, dict_pid)
    %__MODULE__{inner: inner}
  end

  @impl true
  def key(%{inner: inner}), do: RocksIter.key(inner)

  @impl true
  def seek(%{inner: inner} = state, target) do
    %{state | inner: RocksIter.seek(inner, target)}
  end

  @impl true
  def next(%{inner: inner} = state) do
    %{state | inner: RocksIter.next(inner)}
  end

  @impl true
  def exhausted?(%{inner: inner}), do: RocksIter.exhausted?(inner)
end
