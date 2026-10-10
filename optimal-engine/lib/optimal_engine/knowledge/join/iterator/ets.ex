defmodule OptimalEngine.Knowledge.Join.Iterator.ETS do
  @moduledoc """
  Sorted-list iterator over ETS query results.

  Given a list of values (pre-sorted), provides the Iterator behaviour
  (seek/next/key/exhausted?) for use in leapfrog joins. Internally uses
  `:array` for O(log n) binary-search seeks.

  ## Usage

      values = Enum.sort(["alice", "bob", "carol"])
      iter = ETSIter.new(values)
      ETSIter.key(iter)             # => "alice"
      iter = ETSIter.seek(iter, "bob")
      ETSIter.key(iter)             # => "bob"
      iter = ETSIter.next(iter)
      ETSIter.key(iter)             # => "carol"
      iter = ETSIter.next(iter)
      ETSIter.exhausted?(iter)      # => true
  """

  @behaviour OptimalEngine.Knowledge.Join.Iterator

  defstruct [:values, :pos]

  @doc "Create an iterator from a pre-sorted list of values."
  @spec new([term()]) :: t()
  def new(sorted_values) do
    %__MODULE__{values: :array.from_list(sorted_values), pos: 0}
  end

  @type t :: %__MODULE__{values: :array.array(), pos: non_neg_integer()}

  @impl true
  def key(%{pos: pos, values: values}) do
    size = :array.size(values)
    if pos < size, do: :array.get(pos, values), else: nil
  end

  @impl true
  def seek(%{exhausted?: true} = state, _target), do: state

  def seek(state, target) do
    size = :array.size(state.values)
    new_pos = binary_search(state.values, target, state.pos, size)
    %{state | pos: new_pos}
  end

  @impl true
  def next(%{pos: pos, values: values} = state) do
    %{state | pos: min(pos + 1, :array.size(values))}
  end

  @impl true
  def exhausted?(%{pos: pos, values: values}) do
    pos >= :array.size(values)
  end

  # Binary search: find leftmost position in [lo, hi) where arr[pos] >= target.
  defp binary_search(arr, target, lo, hi) when lo < hi do
    mid = div(lo + hi, 2)

    if :array.get(mid, arr) < target do
      binary_search(arr, target, mid + 1, hi)
    else
      binary_search(arr, target, lo, mid)
    end
  end

  defp binary_search(_arr, _target, lo, _hi), do: lo
end
