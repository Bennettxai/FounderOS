defmodule OptimalEngine.Knowledge.SPARQL.Parser do
  @moduledoc """
  Pure Elixir SPARQL parser using tokenization + recursive descent.

  Covers the 80% of SPARQL that agents actually use:
  - SELECT with variables and *
  - WHERE with basic graph patterns (BGP)
  - FILTER with comparisons (=, !=, <, >, <=, >=, CONTAINS, REGEX)
  - OPTIONAL blocks
  - LIMIT and OFFSET
  - ORDER BY (ASC/DESC)
  - PREFIX declarations
  - INSERT DATA and DELETE DATA
  """

  @type token ::
          {:keyword, String.t()}
          | {:var, String.t()}
          | {:uri, String.t()}
          | {:prefixed, String.t()}
          | {:string_literal, String.t()}
          | {:integer, integer()}
          | {:float, float()}
          | {:symbol, String.t()}

  @type ast :: map()

  @keywords ~w(SELECT WHERE FILTER OPTIONAL LIMIT OFFSET ORDER BY ASC DESC
               PREFIX INSERT DELETE DATA DISTINCT UNION BIND AS HAVING GROUP
               COUNT SUM AVG MIN MAX GRAPH)

  @comparison_ops ~w(= != < > <= >=)

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Parse a SPARQL query string into a structured AST map.

  Returns `{:ok, ast}` on success or `{:error, reason}` on failure.
  """
  @spec parse(String.t()) :: {:ok, ast()} | {:error, String.t()}
  def parse(input) when is_binary(input) do
    input
    |> tokenize()
    |> parse_query()
  rescue
    e in RuntimeError -> {:error, e.message}
  end

  # ---------------------------------------------------------------------------
  # Tokenizer
  # ---------------------------------------------------------------------------

  @doc false
  def tokenize(input) do
    input
    |> String.trim()
    |> do_tokenize([])
    |> Enum.reverse()
  end

  defp do_tokenize("", acc), do: acc

  # Skip whitespace
  defp do_tokenize(<<c, rest::binary>>, acc) when c in [?\s, ?\t, ?\n, ?\r] do
    do_tokenize(rest, acc)
  end

  # Skip single-line comments
  defp do_tokenize(<<"#", rest::binary>>, acc) do
    rest = skip_until_newline(rest)
    do_tokenize(rest, acc)
  end

  # Multi-character operators
  defp do_tokenize(<<"!=", rest::binary>>, acc), do: do_tokenize(rest, [{:symbol, "!="} | acc])
  defp do_tokenize(<<"<=", rest::binary>>, acc), do: do_tokenize(rest, [{:symbol, "<="} | acc])
  defp do_tokenize(<<">=", rest::binary>>, acc), do: do_tokenize(rest, [{:symbol, ">="} | acc])
  defp do_tokenize(<<"&&", rest::binary>>, acc), do: do_tokenize(rest, [{:symbol, "&&"} | acc])
  defp do_tokenize(<<"||", rest::binary>>, acc), do: do_tokenize(rest, [{:symbol, "||"} | acc])

  # Full URI: <...> — only if followed by a URI-like character (letter, /, #)
  # Otherwise treat < as a comparison operator
  defp do_tokenize(<<"<", c, _::binary>> = input, acc)
       when c in ?a..?z or c in ?A..?Z or c == ?/ or c == ?# or c == ?h do
    <<"<", rest::binary>> = input
    {uri, rest} = consume_until(rest, ">")
    do_tokenize(rest, [{:uri, uri} | acc])
  end

  # Variable: ?name
  defp do_tokenize(<<"?", rest::binary>>, acc) do
    {name, rest} = consume_name(rest)
    do_tokenize(rest, [{:var, name} | acc])
  end

  # String literal: "..."
  defp do_tokenize(<<"\"", rest::binary>>, acc) do
    {str, rest} = consume_string(rest)
    do_tokenize(rest, [{:string_literal, str} | acc])
  end

  # String literal: '...'
  defp do_tokenize(<<"'", rest::binary>>, acc) do
    {str, rest} = consume_string_single(rest)
    do_tokenize(rest, [{:string_literal, str} | acc])
  end

  # Single-character symbols
  defp do_tokenize(<<c, rest::binary>>, acc)
       when c in [?{, ?}, ?(, ?), ?., ?,, ?;, ?*, ?=, ?<, ?>, ?!, ?^] do
    do_tokenize(rest, [{:symbol, <<c>>} | acc])
  end

  # Numbers and names/keywords/prefixed URIs
  defp do_tokenize(<<c, _::binary>> = input, acc) when c in ?0..?9 do
    {num, rest} = consume_number(input)
    do_tokenize(rest, [num | acc])
  end

  defp do_tokenize(<<c, _::binary>> = input, acc) when c in ?a..?z or c in ?A..?Z or c == ?_ do
    {word, rest} = consume_name(input)

    token =
      cond do
        # Check for prefixed URI: prefix:local
        match?(":" <> _, rest) ->
          <<":", local_rest::binary>> = rest
          {local, rest2} = consume_name_or_empty(local_rest)
          {{:prefixed, "#{word}:#{local}"}, rest2}

        String.upcase(word) in @keywords ->
          {:keyword, String.upcase(word)}

        word in ["true", "false"] ->
          {:boolean, word == "true"}

        word == "a" ->
          {:uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"}

        true ->
          {:name, word}
      end

    case token do
      {{:prefixed, _} = t, r} -> do_tokenize(r, [t | acc])
      t -> do_tokenize(rest, [t | acc])
    end
  end

  # Colon at start (default prefix)
  defp do_tokenize(<<":", rest::binary>>, acc) do
    {local, rest2} = consume_name_or_empty(rest)
    do_tokenize(rest2, [{:prefixed, ":#{local}"} | acc])
  end

  defp do_tokenize(<<c, _rest::binary>>, _acc) do
    raise "Unexpected character: #{<<c>>}"
  end

  # --- Tokenizer helpers ---

  defp skip_until_newline(""), do: ""
  defp skip_until_newline(<<"\n", rest::binary>>), do: rest
  defp skip_until_newline(<<_, rest::binary>>), do: skip_until_newline(rest)

  defp consume_until("", _stop), do: raise("Unexpected end of input")

  defp consume_until(<<c, rest::binary>>, <<c>>) do
    {"", rest}
  end

  defp consume_until(input, stop) do
    <<c, rest::binary>> = input

    if <<c>> == stop do
      {"", rest}
    else
      {tail, rest2} = consume_until(rest, stop)
      {<<c, tail::binary>>, rest2}
    end
  end

  defp consume_string(input), do: consume_string_delim(input, ?", [])
  defp consume_string_single(input), do: consume_string_delim(input, ?', [])

  defp consume_string_delim("", _delim, _acc), do: raise("Unterminated string literal")

  defp consume_string_delim(<<delim, rest::binary>>, delim, acc) do
    {IO.iodata_to_binary(Enum.reverse(acc)), rest}
  end

  defp consume_string_delim(<<"\\", c, rest::binary>>, delim, acc) do
    escaped =
      case c do
        ?n -> "\n"
        ?t -> "\t"
        ?\\ -> "\\"
        ^delim -> <<delim>>
        other -> <<other>>
      end

    consume_string_delim(rest, delim, [escaped | acc])
  end

  defp consume_string_delim(<<c, rest::binary>>, delim, acc) do
    consume_string_delim(rest, delim, [<<c>> | acc])
  end

  defp consume_name(input), do: consume_name(input, [])

  defp consume_name(<<c, rest::binary>>, acc)
       when c in ?a..?z or c in ?A..?Z or c in ?0..?9 or c == ?_ or c == ?- do
    consume_name(rest, [<<c>> | acc])
  end

  defp consume_name(rest, acc) do
    {IO.iodata_to_binary(Enum.reverse(acc)), rest}
  end

  defp consume_name_or_empty(<<c, _::binary>> = input)
       when c in ?a..?z or c in ?A..?Z or c in ?0..?9 or c == ?_ do
    consume_name(input)
  end

  defp consume_name_or_empty(rest), do: {"", rest}

  defp consume_number(input), do: consume_number(input, [], false)

  defp consume_number(<<c, rest::binary>>, acc, is_float) when c in ?0..?9 do
    consume_number(rest, [<<c>> | acc], is_float)
  end

  defp consume_number(<<".", c, rest::binary>>, acc, false) when c in ?0..?9 do
    consume_number(rest, [<<c>>, "." | acc], true)
  end

  defp consume_number(rest, acc, is_float) do
    str = IO.iodata_to_binary(Enum.reverse(acc))

    token =
      if is_float do
        {:float, String.to_float(str)}
      else
        {:integer, String.to_integer(str)}
      end

    {token, rest}
  end

  # ---------------------------------------------------------------------------
  # Recursive Descent Parser
  # ---------------------------------------------------------------------------

  defp parse_query(tokens) do
    {prefixes, tokens} = parse_prefixes(tokens, %{})

    case tokens do
      [{:keyword, "SELECT"} | _] ->
        parse_select(tokens, prefixes)

      [{:keyword, "INSERT"} | _] ->
        parse_insert_data(tokens, prefixes)

      [{:keyword, "DELETE"} | _] ->
        parse_delete_data(tokens, prefixes)

      [{token_type, value} | _] ->
        {:error, "Expected SELECT, INSERT, or DELETE, got #{token_type}: #{inspect(value)}"}

      [] ->
        {:error, "Empty query"}
    end
  end

  # --- PREFIX ---

  defp parse_prefixes([{:keyword, "PREFIX"} | rest], prefixes) do
    case rest do
      [{:prefixed, prefix_name}, {:uri, uri} | rest2] ->
        # prefix_name is like "foaf:" — strip trailing colon handled by tokenizer
        clean_prefix = String.trim_trailing(prefix_name, ":")
        parse_prefixes(rest2, Map.put(prefixes, clean_prefix <> ":", uri))

      [{:name, prefix_name}, {:symbol, ":"}, {:uri, uri} | rest2] ->
        parse_prefixes(rest2, Map.put(prefixes, prefix_name <> ":", uri))

      _ ->
        raise "Malformed PREFIX declaration"
    end
  end

  defp parse_prefixes(tokens, prefixes), do: {prefixes, tokens}

  # --- SELECT ---

  defp parse_select([{:keyword, "SELECT"} | rest], prefixes) do
    {distinct, rest} = parse_distinct(rest)
    {variables, rest} = parse_select_variables(rest, [])

    ast = %{
      type: :select,
      prefixes: prefixes,
      distinct: distinct,
      variables: variables,
      where: [],
      filters: [],
      optionals: [],
      order_by: nil,
      limit: nil,
      offset: nil
    }

    {ast, rest} = parse_where_clause(ast, rest)
    {ast, rest} = parse_solution_modifiers(ast, rest)

    if rest != [] do
      {:ok, ast}
    else
      {:ok, ast}
    end
  end

  defp parse_distinct([{:keyword, "DISTINCT"} | rest]), do: {true, rest}
  defp parse_distinct(rest), do: {false, rest}

  defp parse_select_variables([{:symbol, "*"} | rest], _acc), do: {[:all], rest}

  defp parse_select_variables([{:var, name} | rest], acc) do
    parse_select_variables(rest, [{:var, name} | acc])
  end

  defp parse_select_variables(rest, acc), do: {Enum.reverse(acc), rest}

  # --- WHERE ---

  defp parse_where_clause(ast, [{:keyword, "WHERE"}, {:symbol, "{"} | rest]) do
    parse_group_body(ast, rest)
  end

  defp parse_where_clause(ast, [{:symbol, "{"} | rest]) do
    # WHERE keyword is optional
    parse_group_body(ast, rest)
  end

  defp parse_where_clause(ast, rest), do: {ast, rest}

  defp parse_group_body(ast, tokens) do
    parse_group_body(ast, tokens, [])
  end

  defp parse_group_body(ast, [{:symbol, "}"} | rest], patterns) do
    ast = %{ast | where: ast.where ++ Enum.reverse(patterns)}
    {ast, rest}
  end

  defp parse_group_body(ast, [{:keyword, "OPTIONAL"}, {:symbol, "{"} | rest], patterns) do
    {optional_patterns, rest} = parse_optional_body(rest, [])
    ast = %{ast | optionals: ast.optionals ++ [optional_patterns]}
    parse_group_body(ast, rest, patterns)
  end

  defp parse_group_body(ast, [{:keyword, "FILTER"} | rest], patterns) do
    {filter, rest} = parse_filter(rest)
    ast = %{ast | filters: ast.filters ++ [filter]}
    parse_group_body(ast, rest, patterns)
  end

  defp parse_group_body(ast, [{:keyword, "GRAPH"} | rest], patterns) do
    {graph_term, rest} = parse_graph_term(rest)
    rest = expect_symbol(rest, "{")
    {graph_patterns, rest} = parse_graph_body(rest, [], ast.prefixes)
    tagged = Enum.map(graph_patterns, fn pattern -> {:graph_pattern, graph_term, pattern} end)
    parse_group_body(ast, rest, tagged ++ patterns)
  end

  defp parse_group_body(ast, tokens, patterns) do
    case parse_triple_pattern(tokens) do
      {:ok, pattern, rest} ->
        # Consume optional dot separator
        rest = skip_dot(rest)
        parse_group_body(ast, rest, [resolve_pattern(pattern, ast.prefixes) | patterns])

      {:error, _} = err ->
        raise "Parse error in WHERE clause: #{inspect(err)}"
    end
  end

  defp parse_graph_term([{:var, name} | rest]), do: {{:var, name}, rest}
  defp parse_graph_term([{:uri, uri} | rest]), do: {{:uri, uri}, rest}
  defp parse_graph_term([{:prefixed, pname} | rest]), do: {{:prefixed, pname}, rest}

  defp parse_graph_term(tokens) do
    raise "Expected graph term (variable or URI), got #{inspect(Enum.take(tokens, 1))}"
  end

  defp parse_graph_body([{:symbol, "}"} | rest], acc, _prefixes) do
    {Enum.reverse(acc), rest}
  end

  defp parse_graph_body(tokens, acc, prefixes) do
    case parse_triple_pattern(tokens) do
      {:ok, pattern, rest} ->
        rest = skip_dot(rest)
        parse_graph_body(rest, [resolve_pattern(pattern, prefixes) | acc], prefixes)

      {:error, _} = err ->
        raise "Parse error in GRAPH block: #{inspect(err)}"
    end
  end

  defp parse_optional_body([{:symbol, "}"} | rest], patterns) do
    {Enum.reverse(patterns), rest}
  end

  defp parse_optional_body(tokens, patterns) do
    case parse_triple_pattern(tokens) do
      {:ok, pattern, rest} ->
        rest = skip_dot(rest)
        parse_optional_body(rest, [pattern | patterns])

      {:error, _} = err ->
        raise "Parse error in OPTIONAL: #{inspect(err)}"
    end
  end

  defp skip_dot([{:symbol, "."} | rest]), do: rest
  defp skip_dot(rest), do: rest

  # --- Triple Patterns ---

  defp parse_triple_pattern(tokens) do
    with {:ok, subject, rest} <- parse_term(tokens),
         {:ok, predicate, rest} <- parse_term(rest),
         {:ok, object, rest} <- parse_term(rest) do
      {:ok, {subject, predicate, object}, rest}
    end
  end

  defp parse_term([{:var, name} | rest]), do: {:ok, {:var, name}, rest}
  defp parse_term([{:uri, uri} | rest]), do: {:ok, {:uri, uri}, rest}
  defp parse_term([{:prefixed, pname} | rest]), do: {:ok, {:prefixed, pname}, rest}
  defp parse_term([{:string_literal, str} | rest]), do: {:ok, {:literal, str}, rest}
  defp parse_term([{:integer, n} | rest]), do: {:ok, {:literal, n}, rest}
  defp parse_term([{:float, n} | rest]), do: {:ok, {:literal, n}, rest}
  defp parse_term([{:boolean, b} | rest]), do: {:ok, {:literal, b}, rest}
  defp parse_term([{:name, n} | rest]), do: {:ok, {:name, n}, rest}

  defp parse_term(tokens) do
    token_desc =
      case tokens do
        [{type, val} | _] -> "#{type}: #{inspect(val)}"
        [] -> "end of input"
      end

    {:error, "Expected term, got #{token_desc}"}
  end

  # --- FILTER ---

  defp parse_filter([{:symbol, "("} | rest]) do
    {expr, rest} = parse_filter_expression(rest)
    rest = expect_symbol(rest, ")")
    {expr, rest}
  end

  defp parse_filter(rest) do
    # FILTER without parens — try function-style
    parse_filter_expression(rest)
  end

  defp parse_filter_expression(tokens) do
    {left, rest} = parse_filter_primary(tokens)
    parse_filter_rest(left, rest)
  end

  defp parse_filter_rest(left, [{:symbol, op} | rest]) when op in @comparison_ops do
    {right, rest} = parse_filter_primary(rest)
    expr = {String.to_atom(op), left, right}
    parse_filter_rest(expr, rest)
  end

  defp parse_filter_rest(left, [{:symbol, "&&"} | rest]) do
    {right, rest} = parse_filter_expression(rest)
    {{:and, left, right}, rest}
  end

  defp parse_filter_rest(left, [{:symbol, "||"} | rest]) do
    {right, rest} = parse_filter_expression(rest)
    {{:or, left, right}, rest}
  end

  defp parse_filter_rest(expr, rest), do: {expr, rest}

  defp parse_filter_primary([{:symbol, "!"}, {:symbol, "("} | rest]) do
    {expr, rest} = parse_filter_expression(rest)
    rest = expect_symbol(rest, ")")
    {{:not, expr}, rest}
  end

  defp parse_filter_primary([{:symbol, "("} | rest]) do
    {expr, rest} = parse_filter_expression(rest)
    rest = expect_symbol(rest, ")")
    {expr, rest}
  end

  defp parse_filter_primary([{:name, func_name}, {:symbol, "("} | rest])
       when func_name in [
              "CONTAINS",
              "contains",
              "REGEX",
              "regex",
              "BOUND",
              "bound",
              "STRSTARTS",
              "strstarts",
              "STRENDS",
              "strends",
              "LANG",
              "lang",
              "LANGMATCHES",
              "langmatches",
              "isIRI",
              "isiri",
              "isURI",
              "isuri",
              "isLiteral",
              "isliteral",
              "isBlank",
              "isblank",
              "STR",
              "str"
            ] do
    func = String.upcase(func_name) |> String.to_atom()
    {args, rest} = parse_function_args(rest, [])
    {{func, args}, rest}
  end

  defp parse_filter_primary([{:var, name} | rest]) do
    {{:var, name}, rest}
  end

  defp parse_filter_primary([{:string_literal, str} | rest]) do
    {{:literal, str}, rest}
  end

  defp parse_filter_primary([{:integer, n} | rest]) do
    {{:literal, n}, rest}
  end

  defp parse_filter_primary([{:float, n} | rest]) do
    {{:literal, n}, rest}
  end

  defp parse_filter_primary([{:boolean, b} | rest]) do
    {{:literal, b}, rest}
  end

  defp parse_filter_primary([{:uri, uri} | rest]) do
    {{:uri, uri}, rest}
  end

  defp parse_filter_primary(tokens) do
    raise "Unexpected token in FILTER: #{inspect(Enum.take(tokens, 3))}"
  end

  defp parse_function_args([{:symbol, ")"} | rest], acc) do
    {Enum.reverse(acc), rest}
  end

  defp parse_function_args([{:symbol, ","} | rest], acc) do
    parse_function_args(rest, acc)
  end

  defp parse_function_args(tokens, acc) do
    {arg, rest} = parse_filter_primary(tokens)
    parse_function_args(rest, [arg | acc])
  end

  # --- Solution Modifiers ---

  defp parse_solution_modifiers(ast, tokens) do
    {ast, tokens} = parse_order_by(ast, tokens)
    {ast, tokens} = parse_limit(ast, tokens)
    {ast, tokens} = parse_offset(ast, tokens)
    # Try limit after offset too (either order is valid)
    {ast, tokens} = parse_limit(ast, tokens)
    {ast, tokens}
  end

  defp parse_order_by(ast, [{:keyword, "ORDER"}, {:keyword, "BY"} | rest]) do
    {order_clauses, rest} = parse_order_clauses(rest, [])
    {%{ast | order_by: order_clauses}, rest}
  end

  defp parse_order_by(ast, rest), do: {ast, rest}

  defp parse_order_clauses(
         [{:keyword, "ASC"}, {:symbol, "("}, {:var, name}, {:symbol, ")"} | rest],
         acc
       ) do
    parse_order_clauses(rest, [{:asc, {:var, name}} | acc])
  end

  defp parse_order_clauses(
         [{:keyword, "DESC"}, {:symbol, "("}, {:var, name}, {:symbol, ")"} | rest],
         acc
       ) do
    parse_order_clauses(rest, [{:desc, {:var, name}} | acc])
  end

  defp parse_order_clauses([{:var, name} | rest], acc) do
    parse_order_clauses(rest, [{:asc, {:var, name}} | acc])
  end

  defp parse_order_clauses(rest, acc), do: {Enum.reverse(acc), rest}

  defp parse_limit(%{limit: nil} = ast, [{:keyword, "LIMIT"}, {:integer, n} | rest]) do
    {%{ast | limit: n}, rest}
  end

  defp parse_limit(ast, rest), do: {ast, rest}

  defp parse_offset(ast, [{:keyword, "OFFSET"}, {:integer, n} | rest]) do
    {%{ast | offset: n}, rest}
  end

  defp parse_offset(ast, rest), do: {ast, rest}

  # --- INSERT DATA ---

  defp parse_insert_data(
         [{:keyword, "INSERT"}, {:keyword, "DATA"}, {:symbol, "{"} | rest],
         prefixes
       ) do
    {triples, rest} = parse_data_body(rest, [], prefixes)

    ast = %{
      type: :insert_data,
      prefixes: prefixes,
      triples: triples
    }

    _ = rest
    {:ok, ast}
  end

  defp parse_insert_data(_, _), do: {:error, "Expected INSERT DATA { ... }"}

  # --- DELETE DATA ---

  defp parse_delete_data(
         [{:keyword, "DELETE"}, {:keyword, "DATA"}, {:symbol, "{"} | rest],
         prefixes
       ) do
    {triples, rest} = parse_data_body(rest, [], prefixes)

    ast = %{
      type: :delete_data,
      prefixes: prefixes,
      triples: triples
    }

    _ = rest
    {:ok, ast}
  end

  defp parse_delete_data(_, _), do: {:error, "Expected DELETE DATA { ... }"}

  defp parse_data_body([{:symbol, "}"} | rest], acc, _prefixes) do
    {Enum.reverse(acc), rest}
  end

  defp parse_data_body(tokens, acc, prefixes) do
    case parse_triple_pattern(tokens) do
      {:ok, pattern, rest} ->
        rest = skip_dot(rest)
        resolved = resolve_pattern(pattern, prefixes)
        parse_data_body(rest, [resolved | acc], prefixes)

      {:error, reason} ->
        raise "Parse error in DATA block: #{reason}"
    end
  end

  # --- Prefix Resolution ---

  @doc false
  def resolve_term({:prefixed, pname}, prefixes) do
    case String.split(pname, ":", parts: 2) do
      [prefix, local] ->
        case Map.get(prefixes, prefix <> ":") do
          nil -> {:prefixed, pname}
          base_uri -> {:uri, base_uri <> local}
        end

      _ ->
        {:prefixed, pname}
    end
  end

  def resolve_term(term, _prefixes), do: term

  defp resolve_pattern({s, p, o}, prefixes) do
    {resolve_term(s, prefixes), resolve_term(p, prefixes), resolve_term(o, prefixes)}
  end

  # --- Utility ---

  defp expect_symbol([{:symbol, expected} | rest], expected), do: rest

  defp expect_symbol(tokens, expected) do
    raise "Expected '#{expected}', got #{inspect(Enum.take(tokens, 1))}"
  end
end
