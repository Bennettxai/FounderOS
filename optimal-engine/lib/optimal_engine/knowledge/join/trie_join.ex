defmodule OptimalEngine.Knowledge.Join.TrieJoin do
  @moduledoc """
  Trie-based multi-way join using LeapfrogJoin at each variable level.

  Given a set of triple patterns with shared variables, produces all satisfying
  binding maps. This is worst-case optimal for conjunctive queries (Ngo et al. 2012).

  ## Algorithm

  1. Analyze patterns to find all variables and which patterns constrain each.
  2. Order variables: most-constrained first (appears in the most patterns).
  3. For each variable at the current level, build iterators from relevant patterns
     given the bindings accumulated so far.
  4. Use LeapfrogJoin to intersect those iterators → all values satisfying all
     constraints simultaneously.
  5. For each matching value, extend the binding map and recurse on remaining variables.

  ## Pattern format

  Patterns are 3-tuples `{subject_term, predicate_term, object_term}` where each
  term is one of:
  - `{:var, name}` — unbound variable
  - `{:uri, value}` — URI constant
  - `{:literal, value}` — literal constant
  - `{:name, value}` — short name constant

  ## Usage

      patterns = [
        {{:var, "s"}, {:uri, "knows"}, {:var, "o"}},
        {{:var, "o"}, {:uri, "name"}, {:var, "n"}}
      ]
      TrieJoin.execute(patterns, ETS, ets_state)
      # => [%{"s" => "alice", "o" => "bob", "n" => "Bob"}, ...]
  """

  alias OptimalEngine.Knowledge.Join.LeapfrogJoin
  alias OptimalEngine.Knowledge.Join.Iterator.ETS, as: ETSIter

  @doc """
  Execute a trie join over triple patterns against a backend.

  Returns a list of binding maps (same format as nested-loop join in the
  SPARQL executor).
  """
  @spec execute([tuple()], module(), term()) :: [map()]
  def execute(patterns, backend, backend_state) do
    var_info = analyze_variables(patterns)

    if map_size(var_info) == 0 do
      check_all_exist(patterns, backend, backend_state)
    else
      # Most-constrained variable first: appears in the most patterns
      var_order =
        var_info
        |> Enum.sort_by(fn {_var, info} -> -info.pattern_count end)
        |> Enum.map(fn {var, _info} -> var end)

      trie_join_recursive(var_order, patterns, %{}, backend, backend_state)
    end
  end

  # ---------------------------------------------------------------------------
  # Variable analysis
  # ---------------------------------------------------------------------------

  # Returns %{var_name => %{pattern_count: integer, positions: [{idx, position}]}}
  defp analyze_variables(patterns) do
    patterns
    |> Enum.with_index()
    |> Enum.reduce(%{}, fn {pattern, idx}, acc ->
      pattern
      |> extract_vars()
      |> Enum.reduce(acc, fn {var_name, position}, acc2 ->
        info = Map.get(acc2, var_name, %{pattern_count: 0, positions: []})

        updated = %{
          info
          | pattern_count: info.pattern_count + 1,
            positions: [{idx, position} | info.positions]
        }

        Map.put(acc2, var_name, updated)
      end)
    end)
  end

  defp extract_vars({s, p, o}) do
    []
    |> maybe_add_var(s, :subject)
    |> maybe_add_var(p, :predicate)
    |> maybe_add_var(o, :object)
  end

  defp maybe_add_var(acc, {:var, name}, position), do: [{name, position} | acc]
  defp maybe_add_var(acc, _term, _position), do: acc

  # ---------------------------------------------------------------------------
  # Recursive trie traversal
  # ---------------------------------------------------------------------------

  defp trie_join_recursive([], _patterns, binding, _backend, _state) do
    [binding]
  end

  defp trie_join_recursive([var | rest_vars], patterns, binding, backend, backend_state) do
    # Find patterns that mention this variable
    relevant = Enum.filter(patterns, fn pattern -> mentions_var?(pattern, var) end)

    # Build one iterator per relevant pattern
    iterators =
      relevant
      |> Enum.map(fn pattern ->
        get_values_for_var(pattern, var, binding, backend, backend_state)
      end)
      |> Enum.reject(&is_nil/1)

    if iterators == [] do
      # No iterators means variable is unconstrained by any pattern — should
      # not happen in a well-formed BGP, but fall through gracefully
      trie_join_recursive(rest_vars, patterns, binding, backend, backend_state)
    else
      matching_values = LeapfrogJoin.run(iterators)

      Enum.flat_map(matching_values, fn value ->
        new_binding = Map.put(binding, var, value)
        trie_join_recursive(rest_vars, patterns, new_binding, backend, backend_state)
      end)
    end
  end

  # ---------------------------------------------------------------------------
  # Query helpers
  # ---------------------------------------------------------------------------

  defp mentions_var?({s, p, o}, var) do
    match?({:var, ^var}, s) or match?({:var, ^var}, p) or match?({:var, ^var}, o)
  end

  # Return a {ETSIter, iterator_state} for the values of `var` in `pattern`
  # given the current `binding`, or nil if the query fails/is impossible.
  defp get_values_for_var(pattern, var, binding, backend, backend_state) do
    # Substitute already-bound variables
    {s, p, o} = substitute(pattern, binding)

    # Which position does the target variable occupy?
    position =
      cond do
        match?({:var, ^var}, s) -> :subject
        match?({:var, ^var}, p) -> :predicate
        match?({:var, ^var}, o) -> :object
        true -> nil
      end

    if is_nil(position) do
      # Variable doesn't appear in this pattern (binding already handled it)
      nil
    else
      query_pattern = build_query_pattern(s, p, o)

      case backend.query(backend_state, query_pattern) do
        {:ok, triples} ->
          values =
            triples
            |> Enum.map(fn {ts, tp, to_val} ->
              case position do
                :subject -> ts
                :predicate -> tp
                :object -> to_val
              end
            end)
            |> Enum.sort()
            |> Enum.uniq()

          {ETSIter, ETSIter.new(values)}

        {:error, _} ->
          nil
      end
    end
  end

  # Substitute bound variable values from the current binding into a pattern.
  # Variables bound in `binding` become {:bound, value}; unbound stay {:var, name}.
  defp substitute({s, p, o}, binding) do
    {sub_term(s, binding), sub_term(p, binding), sub_term(o, binding)}
  end

  defp sub_term({:var, name} = term, binding) do
    case Map.get(binding, name) do
      nil -> term
      value -> {:bound, value}
    end
  end

  defp sub_term(term, _binding), do: term

  # Build a keyword-list query pattern for the backend (nil values omitted).
  defp build_query_pattern(s, p, o) do
    []
    |> add_constraint(:subject, s)
    |> add_constraint(:predicate, p)
    |> add_constraint(:object, o)
  end

  defp add_constraint(pattern, key, {:bound, val}), do: [{key, val} | pattern]
  defp add_constraint(pattern, key, {:uri, val}), do: [{key, val} | pattern]
  defp add_constraint(pattern, key, {:name, val}), do: [{key, val} | pattern]

  defp add_constraint(pattern, key, {:literal, val}) when is_binary(val),
    do: [{key, val} | pattern]

  defp add_constraint(pattern, key, {:literal, val}),
    do: [{key, to_string(val)} | pattern]

  defp add_constraint(pattern, _key, {:var, _}), do: pattern
  defp add_constraint(pattern, _key, _), do: pattern

  # ---------------------------------------------------------------------------
  # No-variable case: existence check
  # ---------------------------------------------------------------------------

  defp check_all_exist(patterns, backend, backend_state) do
    all_exist =
      Enum.all?(patterns, fn {s, p, o} ->
        query = build_query_pattern(s, p, o)

        case backend.query(backend_state, query) do
          {:ok, [_ | _]} -> true
          _ -> false
        end
      end)

    if all_exist, do: [%{}], else: []
  end
end
