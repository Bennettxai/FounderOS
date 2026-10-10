defmodule OptimalEngine.Knowledge.Optimizer do
  @moduledoc """
  Cost-based query optimizer for SPARQL BGP evaluation.

  Reorders triple patterns to minimize intermediate result sizes using
  cardinality estimation from predicate histograms collected by `Stats`.

  Strategy selection:
  - <= 4 patterns  → brute-force permutation search (optimal, max 24 orderings)
  - 5–6 patterns   → greedy with selectivity heuristic (near-optimal, max 720)
  - > 6 patterns   → greedy (linear pass)

  The optimizer annotates the returned AST with a `:plan` key:
  - `:trie_join`    — pattern count >= 4 with shared variables (handed to TrieJoin)
  - `:nested_loop`  — everything else
  """

  alias OptimalEngine.Knowledge.Stats

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Optimize a parsed SPARQL SELECT AST by reordering its WHERE patterns.

  Returns the AST with `where` patterns reordered and a `:plan` annotation.
  Falls back to the original AST unchanged if stats collection fails.
  """
  @spec optimize(map(), module(), term()) :: map()
  def optimize(%{type: :select, where: patterns} = ast, backend, backend_state) do
    case Stats.collect(backend, backend_state) do
      {:ok, stats} ->
        {graph_patterns, plain_patterns} =
          Enum.split_with(patterns, fn
            {:graph_pattern, _, _} -> true
            _ -> false
          end)

        optimized_plain = reorder(plain_patterns, stats)

        plan =
          if length(optimized_plain) >= 4 and has_shared_variables?(optimized_plain) do
            :trie_join
          else
            :nested_loop
          end

        %{ast | where: optimized_plain ++ graph_patterns}
        |> Map.put(:plan, plan)

      {:error, _} ->
        # Stats unavailable — return unoptimized AST so the query still runs
        ast
    end
  end

  def optimize(ast, _backend, _state), do: ast

  @doc """
  Reorder a list of triple patterns by ascending estimated cardinality.

  Cheaper (more selective) patterns are placed first so the nested-loop
  executor produces smaller intermediate result sets.
  """
  @spec reorder([tuple()], Stats.t()) :: [tuple()]
  def reorder(patterns, _stats) when length(patterns) <= 1, do: patterns

  def reorder(patterns, stats) when length(patterns) <= 4 do
    patterns
    |> permutations()
    |> Enum.min_by(&total_cost(&1, stats))
  end

  def reorder(patterns, stats) do
    greedy_reorder(patterns, stats, [], MapSet.new())
  end

  # ---------------------------------------------------------------------------
  # Greedy reordering — O(n²) passes, picks cheapest available pattern next
  # ---------------------------------------------------------------------------

  defp greedy_reorder([], _stats, acc, _bound), do: Enum.reverse(acc)

  defp greedy_reorder(remaining, stats, acc, bound_vars) do
    {best, _cost} =
      remaining
      |> Enum.map(fn p -> {p, estimate_pattern_cost(p, stats, bound_vars)} end)
      |> Enum.min_by(fn {_, cost} -> cost end)

    new_bound =
      best
      |> pattern_vars()
      |> MapSet.new()
      |> MapSet.union(bound_vars)

    greedy_reorder(List.delete(remaining, best), stats, [best | acc], new_bound)
  end

  # ---------------------------------------------------------------------------
  # Cost model
  # ---------------------------------------------------------------------------

  defp total_cost(pattern_order, stats) do
    {_bound, cost} =
      Enum.reduce(pattern_order, {MapSet.new(), 0}, fn pattern, {bound, running_cost} ->
        pat_cost = estimate_pattern_cost(pattern, stats, bound)
        new_bound = pattern |> pattern_vars() |> MapSet.new() |> MapSet.union(bound)
        {new_bound, running_cost + pat_cost}
      end)

    cost
  end

  defp estimate_pattern_cost({s, p, o}, stats, bound_vars) do
    query =
      []
      |> add_cost_term(:subject, s, bound_vars)
      |> add_cost_term(:predicate, p, bound_vars)
      |> add_cost_term(:object, o, bound_vars)

    Stats.estimate_cardinality(query, stats)
  end

  # A variable that is already bound by a prior pattern acts like a constant —
  # push it into the query pattern so cardinality estimation benefits from it.
  defp add_cost_term(query, key, {:var, name}, bound_vars) do
    if MapSet.member?(bound_vars, name) do
      [{key, "__bound__"} | query]
    else
      query
    end
  end

  defp add_cost_term(query, key, {:uri, val}, _bound), do: [{key, val} | query]

  defp add_cost_term(query, key, {:literal, val}, _bound) when is_binary(val),
    do: [{key, val} | query]

  defp add_cost_term(query, key, {:literal, val}, _bound), do: [{key, to_string(val)} | query]
  defp add_cost_term(query, key, {:name, val}, _bound), do: [{key, val} | query]
  defp add_cost_term(query, key, {:bound, val}, _bound), do: [{key, val} | query]
  defp add_cost_term(query, _key, _term, _bound), do: query

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  defp pattern_vars({s, p, o}) do
    for term <- [s, p, o],
        match?({:var, _}, term),
        do: elem(term, 1)
  end

  defp has_shared_variables?(patterns) do
    patterns
    |> Enum.flat_map(&pattern_vars/1)
    |> Enum.frequencies()
    |> Enum.any?(fn {_, count} -> count > 1 end)
  end

  defp permutations([]), do: [[]]

  defp permutations(list) do
    for elem <- list,
        rest <- permutations(List.delete(list, elem)),
        do: [elem | rest]
  end
end
