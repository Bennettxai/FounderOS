defmodule OptimalEngine.Knowledge.Join.LeapfrogJoin do
  @moduledoc """
  Leapfrog join: worst-case optimal intersection of K sorted iterators.

  Given K iterators over sorted domains, finds all values present in every
  iterator. This is the building block for TrieJoin — one LeapfrogJoin per
  variable level.

  ## Algorithm

  This implementation uses the "seek all to candidate" approach, which is
  equivalent to the circular leapfrog algorithm but simpler to implement
  correctly when iterators may start with equal keys:

  1. Identify the current maximum key across all iterators.
  2. Seek every iterator to that maximum.
  3. If all iterators landed on the same key → MATCH. Yield it.
     Advance all iterators one step and repeat.
  4. If some iterator jumped past the previous maximum → the new maximum is
     the furthest-ahead iterator. Repeat from step 1.
  5. If any iterator is exhausted → done.

  ## Correctness

  A value V is in the intersection iff every iterator can be seeked to V.
  We find the minimum candidate by always seeking to the global maximum:
  any value less than the current maximum cannot be in the intersection
  (at least one iterator is already past it).

  ## Complexity

  O(N * K) seeks total in the worst case, where N = result size, K = iterators.
  For K small (typically 2-6 in join queries) this is optimal in practice.

  ## Usage

      iterators = [
        {ETSIter, ETSIter.new(["a", "b", "c"])},
        {ETSIter, ETSIter.new(["b", "c", "d"])}
      ]
      LeapfrogJoin.run(iterators)
      # => ["b", "c"]
  """

  alias OptimalEngine.Knowledge.Join.Iterator

  @doc """
  Run leapfrog join over a list of `{module, state}` iterator pairs.

  Each module must implement the `Iterator` behaviour. Returns all values
  that are present in every iterator's domain, in sorted order.
  """
  @spec run([{module(), Iterator.t()}]) :: [Iterator.key()]
  def run([]), do: []

  def run([{mod, state}]) do
    collect_single(mod, state, [])
  end

  def run(iterators) do
    if Enum.any?(iterators, fn {mod, state} -> mod.exhausted?(state) end) do
      []
    else
      leapfrog_search(iterators, [])
    end
  end

  # ---------------------------------------------------------------------------
  # Single-iterator fast path: return all values
  # ---------------------------------------------------------------------------

  defp collect_single(mod, state, acc) do
    if mod.exhausted?(state) do
      Enum.reverse(acc)
    else
      val = mod.key(state)
      collect_single(mod, mod.next(state), [val | acc])
    end
  end

  # ---------------------------------------------------------------------------
  # Core leapfrog search loop
  # ---------------------------------------------------------------------------

  defp leapfrog_search(iterators, acc) do
    # Find the maximum current key across all iterators.
    x_max =
      Enum.reduce(iterators, nil, fn {mod, state}, current_max ->
        key = mod.key(state)
        if current_max == nil or key > current_max, do: key, else: current_max
      end)

    # Seek every iterator to x_max.
    seeked =
      Enum.map(iterators, fn {mod, state} ->
        {mod, mod.seek(state, x_max)}
      end)

    if Enum.any?(seeked, fn {mod, state} -> mod.exhausted?(state) end) do
      # At least one iterator has no value >= x_max; intersection is empty.
      Enum.reverse(acc)
    else
      # Check if all iterators landed on x_max (true intersection point).
      all_match = Enum.all?(seeked, fn {mod, state} -> mod.key(state) == x_max end)

      if all_match do
        # x_max is in the intersection. Advance all iterators past x_max.
        advanced =
          Enum.map(seeked, fn {mod, state} -> {mod, mod.next(state)} end)

        if Enum.any?(advanced, fn {mod, state} -> mod.exhausted?(state) end) do
          Enum.reverse([x_max | acc])
        else
          leapfrog_search(advanced, [x_max | acc])
        end
      else
        # Some iterator jumped past x_max. The new maximum is computed in the
        # next iteration. Loop without adding to acc.
        leapfrog_search(seeked, acc)
      end
    end
  end
end
