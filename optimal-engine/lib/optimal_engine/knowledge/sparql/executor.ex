defmodule OptimalEngine.Knowledge.SPARQL.Executor do
  @moduledoc """
  Executes parsed SPARQL ASTs against a `OptimalEngine.Knowledge.Backend` state.

  Takes the AST produced by `OptimalEngine.Knowledge.SPARQL.Parser` and evaluates it
  by resolving BGP patterns, applying filters, joins, OPTIONAL patterns,
  and solution modifiers (ORDER BY, LIMIT, OFFSET).

  Results are returned as lists of binding maps, e.g.:
      [%{"s" => "alice", "name" => "Alice"}, ...]
  """

  @type binding :: %{String.t() => term()}
  @type result ::
          {:ok, [binding()]}
          | {:ok, :inserted, non_neg_integer()}
          | {:ok, :deleted, non_neg_integer()}
          | {:error, term()}

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Execute a parsed AST against a backend module + state.

  ## Parameters
  - `ast` — Parsed AST map from `Parser.parse/1`
  - `backend` — Backend module implementing `OptimalEngine.Knowledge.Backend`
  - `backend_state` — Opaque backend state

  ## Returns
  - `{:ok, bindings}` for SELECT queries
  - `{:ok, :inserted, count}` for INSERT DATA
  - `{:ok, :deleted, count}` for DELETE DATA
  - `{:error, reason}` on failure
  """
  @spec execute(map(), module(), term()) :: result()
  def execute(%{type: :select} = ast, backend, backend_state) do
    execute_select(ast, backend, backend_state)
  rescue
    e -> {:error, Exception.message(e)}
  end

  def execute(%{type: :insert_data} = ast, backend, backend_state) do
    execute_insert(ast, backend, backend_state)
  rescue
    e -> {:error, Exception.message(e)}
  end

  def execute(%{type: :delete_data} = ast, backend, backend_state) do
    execute_delete(ast, backend, backend_state)
  rescue
    e -> {:error, Exception.message(e)}
  end

  def execute(%{type: type}, _backend, _state) do
    {:error, "Unsupported query type: #{inspect(type)}"}
  end

  # ---------------------------------------------------------------------------
  # SELECT Execution
  # ---------------------------------------------------------------------------

  defp execute_select(ast, backend, backend_state) do
    # 1. Evaluate basic graph patterns (BGP join)
    bindings = evaluate_bgp(ast.where, backend, backend_state)

    # 2. Evaluate OPTIONAL patterns
    bindings = evaluate_optionals(bindings, ast.optionals, backend, backend_state)

    # 3. Apply FILTER
    bindings = apply_filters(bindings, ast.filters)

    # 4. Project variables (before DISTINCT so only selected vars are compared)
    bindings = project(bindings, ast.variables)

    # 5. Apply DISTINCT
    bindings = if ast.distinct, do: Enum.uniq(bindings), else: bindings

    # 6. Apply ORDER BY
    bindings = apply_order_by(bindings, ast.order_by)

    # 7. Apply OFFSET
    bindings = apply_offset(bindings, ast.offset)

    # 8. Apply LIMIT
    bindings = apply_limit(bindings, ast.limit)

    {:ok, bindings}
  end

  # ---------------------------------------------------------------------------
  # BGP Evaluation — Pattern-by-Pattern Join
  # ---------------------------------------------------------------------------

  defp evaluate_bgp([], _backend, _state), do: [%{}]

  defp evaluate_bgp(patterns, backend, state) do
    # Use leapfrog triejoin for complex BGPs: >= 4 plain triple patterns with
    # at least one shared variable (worst-case optimal for conjunctive queries).
    plain_patterns =
      Enum.filter(patterns, fn
        {:graph_pattern, _, _} -> false
        _ -> true
      end)

    if length(plain_patterns) >= 4 and has_shared_variables?(plain_patterns) do
      OptimalEngine.Knowledge.Join.TrieJoin.execute(plain_patterns, backend, state)
    else
      Enum.reduce(patterns, [%{}], fn pattern, bindings ->
        join_pattern(bindings, pattern, backend, state)
      end)
    end
  end

  defp has_shared_variables?(patterns) do
    all_vars =
      Enum.flat_map(patterns, fn {s, p, o} ->
        vars = []

        vars =
          case s do
            {:var, name} -> [name | vars]
            _ -> vars
          end

        vars =
          case p do
            {:var, name} -> [name | vars]
            _ -> vars
          end

        vars =
          case o do
            {:var, name} -> [name | vars]
            _ -> vars
          end

        vars
      end)

    all_vars
    |> Enum.frequencies()
    |> Enum.any?(fn {_, count} -> count > 1 end)
  end

  defp join_pattern(bindings, {:graph_pattern, graph_term, triple_pattern}, backend, state) do
    Enum.flat_map(bindings, fn binding ->
      {s_pat, p_pat, o_pat} = substitute_pattern(triple_pattern, binding)
      g_pat = substitute_term(graph_term, binding)
      query_pattern = build_query_pattern(s_pat, p_pat, o_pat) ++ build_graph_constraint(g_pat)

      case backend.query(state, query_pattern) do
        {:ok, triples} ->
          triples
          |> Enum.flat_map(fn {s, p, o} ->
            case unify(triple_pattern, {s, p, o}, binding) do
              {:ok, new_binding} ->
                {:ok, final_binding} = unify_graph_term(graph_term, g_pat, new_binding)
                [final_binding]

              :fail ->
                []
            end
          end)

        {:error, _} ->
          []
      end
    end)
  end

  defp join_pattern(bindings, pattern, backend, state) do
    Enum.flat_map(bindings, fn binding ->
      {s_pat, p_pat, o_pat} = substitute_pattern(pattern, binding)
      query_pattern = build_query_pattern(s_pat, p_pat, o_pat)

      case backend.query(state, query_pattern) do
        {:ok, triples} ->
          triples
          |> Enum.flat_map(fn {s, p, o} ->
            case unify(pattern, {s, p, o}, binding) do
              {:ok, new_binding} -> [new_binding]
              :fail -> []
            end
          end)

        {:error, _} ->
          []
      end
    end)
  end

  defp build_graph_constraint({:bound, value}), do: [graph: value]
  defp build_graph_constraint({:uri, value}), do: [graph: value]
  defp build_graph_constraint({:var, _}), do: []
  defp build_graph_constraint(_), do: []

  defp unify_graph_term({:var, name}, _resolved, binding) do
    # Graph variables require the backend to return graph info in results.
    # In Phase 1B the backend returns triples not quads, so we cannot bind
    # the graph variable from the result. We pass through the binding unchanged;
    # graph URI constraints are already pushed into the query_pattern above.
    case Map.get(binding, name) do
      nil -> {:ok, binding}
      _ -> {:ok, binding}
    end
  end

  defp unify_graph_term(_graph_term, _resolved, binding), do: {:ok, binding}

  defp substitute_pattern({s, p, o}, binding) do
    {substitute_term(s, binding), substitute_term(p, binding), substitute_term(o, binding)}
  end

  defp substitute_term({:var, name}, binding) do
    case Map.get(binding, name) do
      nil -> {:var, name}
      value -> {:bound, value}
    end
  end

  defp substitute_term(term, _binding), do: term

  defp build_query_pattern(s, p, o) do
    pattern = []
    pattern = add_constraint(pattern, :subject, s)
    pattern = add_constraint(pattern, :predicate, p)
    pattern = add_constraint(pattern, :object, o)
    pattern
  end

  defp add_constraint(pattern, key, {:bound, value}), do: [{key, value} | pattern]
  defp add_constraint(pattern, key, {:uri, value}), do: [{key, value} | pattern]

  defp add_constraint(pattern, key, {:literal, value}) when is_binary(value),
    do: [{key, value} | pattern]

  defp add_constraint(pattern, key, {:literal, value}), do: [{key, to_string(value)} | pattern]
  defp add_constraint(pattern, key, {:name, value}), do: [{key, value} | pattern]
  defp add_constraint(pattern, _key, {:var, _}), do: pattern
  defp add_constraint(pattern, _key, _), do: pattern

  defp unify({s_pat, p_pat, o_pat}, {s, p, o}, binding) do
    with {:ok, binding} <- unify_term(s_pat, s, binding),
         {:ok, binding} <- unify_term(p_pat, p, binding),
         {:ok, binding} <- unify_term(o_pat, o, binding) do
      {:ok, binding}
    end
  end

  defp unify_term({:var, name}, value, binding) do
    case Map.get(binding, name) do
      nil -> {:ok, Map.put(binding, name, value)}
      ^value -> {:ok, binding}
      _ -> :fail
    end
  end

  defp unify_term({:uri, uri}, value, binding) do
    if uri == value, do: {:ok, binding}, else: :fail
  end

  defp unify_term({:literal, lit}, value, binding) when is_binary(lit) do
    if lit == value, do: {:ok, binding}, else: :fail
  end

  defp unify_term({:literal, lit}, value, binding) do
    if to_string(lit) == value, do: {:ok, binding}, else: :fail
  end

  defp unify_term({:name, name}, value, binding) do
    if name == value, do: {:ok, binding}, else: :fail
  end

  defp unify_term({:bound, val}, value, binding) do
    if val == value, do: {:ok, binding}, else: :fail
  end

  defp unify_term({:prefixed, _pname}, _value, binding) do
    # Prefixed URIs should have been resolved by the parser.
    # If they weren't, treat as opaque match.
    {:ok, binding}
  end

  # ---------------------------------------------------------------------------
  # OPTIONAL Evaluation — Left Outer Join
  # ---------------------------------------------------------------------------

  defp evaluate_optionals(bindings, [], _backend, _state), do: bindings

  defp evaluate_optionals(bindings, optional_groups, backend, state) do
    Enum.reduce(optional_groups, bindings, fn opt_patterns, current_bindings ->
      left_outer_join(current_bindings, opt_patterns, backend, state)
    end)
  end

  defp left_outer_join(bindings, patterns, backend, state) do
    Enum.flat_map(bindings, fn binding ->
      extended =
        Enum.reduce(patterns, [binding], fn pattern, acc_bindings ->
          join_pattern(acc_bindings, pattern, backend, state)
        end)

      case extended do
        [] -> [binding]
        results -> results
      end
    end)
  end

  # ---------------------------------------------------------------------------
  # FILTER Evaluation
  # ---------------------------------------------------------------------------

  defp apply_filters(bindings, []), do: bindings

  defp apply_filters(bindings, filters) do
    Enum.filter(bindings, fn binding ->
      Enum.all?(filters, fn filter ->
        evaluate_filter(filter, binding)
      end)
    end)
  end

  defp evaluate_filter({:==, left, right}, binding) do
    resolve_filter_value(left, binding) == resolve_filter_value(right, binding)
  end

  defp evaluate_filter({:=, left, right}, binding) do
    resolve_filter_value(left, binding) == resolve_filter_value(right, binding)
  end

  defp evaluate_filter({:!=, left, right}, binding) do
    resolve_filter_value(left, binding) != resolve_filter_value(right, binding)
  end

  defp evaluate_filter({:<, left, right}, binding) do
    compare_values(left, right, binding, &Kernel.</2)
  end

  defp evaluate_filter({:>, left, right}, binding) do
    compare_values(left, right, binding, &Kernel.>/2)
  end

  defp evaluate_filter({:<=, left, right}, binding) do
    compare_values(left, right, binding, &Kernel.<=/2)
  end

  defp evaluate_filter({:>=, left, right}, binding) do
    compare_values(left, right, binding, &Kernel.>=/2)
  end

  defp evaluate_filter({:and, left, right}, binding) do
    evaluate_filter(left, binding) and evaluate_filter(right, binding)
  end

  defp evaluate_filter({:or, left, right}, binding) do
    evaluate_filter(left, binding) or evaluate_filter(right, binding)
  end

  defp evaluate_filter({:not, expr}, binding) do
    not evaluate_filter(expr, binding)
  end

  defp evaluate_filter({:CONTAINS, args}, binding) do
    [haystack, needle] = Enum.map(args, &resolve_filter_value(&1, binding))
    is_binary(haystack) and is_binary(needle) and String.contains?(haystack, needle)
  end

  defp evaluate_filter({:REGEX, args}, binding) do
    case args do
      [source, pattern | flags] ->
        source_val = resolve_filter_value(source, binding)
        pattern_val = resolve_filter_value(pattern, binding)
        flag_str = if flags != [], do: resolve_filter_value(hd(flags), binding), else: ""

        opts = if String.contains?(to_string(flag_str), "i"), do: "i", else: ""

        case Regex.compile(pattern_val, opts) do
          {:ok, regex} -> Regex.match?(regex, to_string(source_val))
          _ -> false
        end

      _ ->
        false
    end
  end

  defp evaluate_filter({:BOUND, args}, binding) do
    [{:var, name}] = args
    Map.has_key?(binding, name)
  end

  defp evaluate_filter({:STRSTARTS, args}, binding) do
    [source, prefix] = Enum.map(args, &resolve_filter_value(&1, binding))
    is_binary(source) and is_binary(prefix) and String.starts_with?(source, prefix)
  end

  defp evaluate_filter({:STRENDS, args}, binding) do
    [source, suffix] = Enum.map(args, &resolve_filter_value(&1, binding))
    is_binary(source) and is_binary(suffix) and String.ends_with?(source, suffix)
  end

  defp evaluate_filter(_, _binding), do: true

  defp resolve_filter_value({:var, name}, binding) do
    case Map.get(binding, name) do
      nil -> nil
      val -> maybe_parse_number(val)
    end
  end

  defp resolve_filter_value({:literal, val}, _binding), do: val
  defp resolve_filter_value({:uri, uri}, _binding), do: uri
  defp resolve_filter_value(val, _binding), do: val

  defp maybe_parse_number(val) when is_binary(val) do
    case Integer.parse(val) do
      {n, ""} ->
        n

      _ ->
        case Float.parse(val) do
          {f, ""} -> f
          _ -> val
        end
    end
  end

  defp maybe_parse_number(val), do: val

  defp compare_values(left, right, binding, comparator) do
    l = resolve_filter_value(left, binding)
    r = resolve_filter_value(right, binding)

    cond do
      is_number(l) and is_number(r) -> comparator.(l, r)
      is_binary(l) and is_binary(r) -> comparator.(l, r)
      true -> false
    end
  end

  # ---------------------------------------------------------------------------
  # Solution Modifiers
  # ---------------------------------------------------------------------------

  defp apply_order_by(bindings, nil), do: bindings

  defp apply_order_by(bindings, order_clauses) do
    Enum.sort_by(
      bindings,
      fn binding ->
        Enum.map(order_clauses, fn
          {:asc, {:var, name}} -> {0, Map.get(binding, name, "")}
          {:desc, {:var, name}} -> {1, Map.get(binding, name, "")}
        end)
      end,
      fn keys_a, keys_b ->
        compare_sort_keys(keys_a, keys_b)
      end
    )
  end

  defp compare_sort_keys([], []), do: true

  defp compare_sort_keys([{dir_a, val_a} | rest_a], [{_dir_b, val_b} | rest_b]) do
    cond do
      val_a == val_b -> compare_sort_keys(rest_a, rest_b)
      dir_a == 0 -> val_a <= val_b
      true -> val_a >= val_b
    end
  end

  defp apply_offset(bindings, nil), do: bindings
  defp apply_offset(bindings, offset), do: Enum.drop(bindings, offset)

  defp apply_limit(bindings, nil), do: bindings
  defp apply_limit(bindings, limit), do: Enum.take(bindings, limit)

  # ---------------------------------------------------------------------------
  # Projection
  # ---------------------------------------------------------------------------

  defp project(bindings, [:all]), do: bindings

  defp project(bindings, variables) do
    var_names = Enum.map(variables, fn {:var, name} -> name end)

    Enum.map(bindings, fn binding ->
      Map.take(binding, var_names)
    end)
  end

  # ---------------------------------------------------------------------------
  # INSERT DATA Execution
  # ---------------------------------------------------------------------------

  defp execute_insert(%{triples: triples}, backend, backend_state) do
    resolved_triples =
      Enum.map(triples, fn {s, p, o} ->
        {term_to_string(s), term_to_string(p), term_to_string(o)}
      end)

    case backend.assert_many(backend_state, resolved_triples) do
      {:ok, _new_state} -> {:ok, :inserted, length(resolved_triples)}
      {:error, _} = err -> err
    end
  end

  # ---------------------------------------------------------------------------
  # DELETE DATA Execution
  # ---------------------------------------------------------------------------

  defp execute_delete(%{triples: triples}, backend, backend_state) do
    resolved_triples =
      Enum.map(triples, fn {s, p, o} ->
        {term_to_string(s), term_to_string(p), term_to_string(o)}
      end)

    errors =
      Enum.reduce(resolved_triples, [], fn {s, p, o}, acc ->
        case backend.retract(backend_state, s, p, o) do
          {:ok, _} -> acc
          {:error, reason} -> [reason | acc]
        end
      end)

    case errors do
      [] -> {:ok, :deleted, length(resolved_triples)}
      _ -> {:error, {:partial_delete, errors}}
    end
  end

  # ---------------------------------------------------------------------------
  # Helpers
  # ---------------------------------------------------------------------------

  defp term_to_string({:uri, uri}), do: uri
  defp term_to_string({:literal, val}) when is_binary(val), do: val
  defp term_to_string({:literal, val}), do: to_string(val)
  defp term_to_string({:prefixed, pname}), do: pname
  defp term_to_string({:name, name}), do: name
  defp term_to_string({:var, _}), do: raise("Variables not allowed in INSERT/DELETE DATA")
end
