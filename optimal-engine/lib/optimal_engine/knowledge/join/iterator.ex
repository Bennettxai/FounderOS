defmodule OptimalEngine.Knowledge.Join.Iterator do
  @moduledoc """
  Sorted iterator behaviour for leapfrog triejoin.

  Iterators provide ordered access to a set of values (typically entity IDs
  from a column of a triple pattern result). The leapfrog algorithm requires:
  - `key/1` — current value, or nil when exhausted
  - `seek/2` — advance to first value >= target
  - `next/1` — advance to next value
  - `exhausted?/1` — check if past end

  All implementations must maintain the invariant that values are produced in
  sorted (ascending) order, and that `seek/2` on an exhausted iterator is a
  no-op that returns the exhausted state.
  """

  @type t :: term()
  @type key :: term()

  @callback key(t()) :: key() | nil
  @callback seek(t(), key()) :: t()
  @callback next(t()) :: t()
  @callback exhausted?(t()) :: boolean()
end
