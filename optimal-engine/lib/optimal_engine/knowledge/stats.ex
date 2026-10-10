defmodule OptimalEngine.Knowledge.Stats do
  @moduledoc """
  Statistics collection for cost-based query optimization.

  Maintains per-predicate histograms and cardinality estimates
  used by the Optimizer to reorder BGP patterns.
  """

  @type t :: %__MODULE__{
          total: non_neg_integer(),
          predicate_counts: %{String.t() => non_neg_integer()},
          subject_count: non_neg_integer(),
          object_count: non_neg_integer()
        }

  defstruct total: 0,
            predicate_counts: %{},
            subject_count: 0,
            object_count: 0

  @doc """
  Build statistics from a backend state by scanning all triples.
  """
  @spec collect(module(), term()) :: {:ok, t()} | {:error, term()}
  def collect(backend, backend_state) do
    case backend.query(backend_state, []) do
      {:ok, triples} ->
        base =
          Enum.reduce(triples, %__MODULE__{}, fn {_s, p, _o}, acc ->
            %{
              acc
              | total: acc.total + 1,
                predicate_counts: Map.update(acc.predicate_counts, p, 1, &(&1 + 1))
            }
          end)

        subjects = triples |> Enum.map(&elem(&1, 0)) |> Enum.uniq() |> length()
        objects = triples |> Enum.map(&elem(&1, 2)) |> Enum.uniq() |> length()

        {:ok, %{base | subject_count: subjects, object_count: objects}}

      {:error, _} = err ->
        err
    end
  end

  @doc """
  Estimate cardinality (expected result count) for a triple pattern.

  `pattern` is a keyword list with any subset of `:subject`, `:predicate`,
  `:object` bound to concrete values. Unbound positions are omitted.

  Returns a float so that tie-breaking between patterns of equal integer
  selectivity favours the one with lower raw predicate frequency.

  Uses an independence assumption with histogram-based estimation.
  """
  @spec estimate_cardinality(keyword(), t()) :: number()
  def estimate_cardinality(pattern, %__MODULE__{} = stats) do
    s = Keyword.get(pattern, :subject)
    p = Keyword.get(pattern, :predicate)
    o = Keyword.get(pattern, :object)

    pred_count = if p != nil, do: Map.get(stats.predicate_counts, p, 0), else: nil

    # When two patterns produce the same integer estimate we want to break ties
    # in favour of the pattern with the lower raw predicate count.  Adding a
    # tiny fractional component achieves this without changing integer ordering.
    tie_break = if pred_count != nil, do: pred_count / (stats.total + 1) * 0.001, else: 0.0

    base =
      cond do
        # All three bound → at most 1 result
        s != nil and p != nil and o != nil ->
          1

        # S + P bound → rows for this predicate divided by distinct subjects
        s != nil and p != nil ->
          max(1, div(pred_count, max(stats.subject_count, 1)))

        # P + O bound → rows for this predicate divided by distinct objects
        p != nil and o != nil ->
          max(1, div(pred_count, max(stats.object_count, 1)))

        # S + O bound (no predicate selectivity) → small fraction of total
        s != nil and o != nil ->
          max(1, div(stats.total, max(stats.subject_count * max(stats.object_count, 1), 1)))

        # P only → frequency of that predicate in the store
        p != nil ->
          pred_count

        # S only → average triples per subject
        s != nil ->
          max(1, div(stats.total, max(stats.subject_count, 1)))

        # O only → average triples per object
        o != nil ->
          max(1, div(stats.total, max(stats.object_count, 1)))

        # Wildcard → full scan
        true ->
          stats.total
      end

    base + tie_break
  end
end
