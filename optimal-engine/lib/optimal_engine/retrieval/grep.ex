defmodule OptimalEngine.Retrieval.Grep do
  @moduledoc """
  Hybrid grep: semantic (vector + BM25) + literal substring match over a workspace.

  This is the engine behind `mix optimal.grep` and `GET /api/grep`. Unlike a
  plain file-system grep, every match carries the full signal trace — slug,
  scale, intent, sn_ratio, modality — so the caller can reason about *why*
  a chunk surfaced, not just that it did.

  ## What makes this ours

  - Returns the full signal trace per match (slug, scale, intent, sn_ratio, modality)
  - Honors workspace isolation — no cross-workspace leakage
  - Allows typed intent / scale / modality filters (cued recall)
  - Both literal AND semantic in one pass, controlled by the `:literal` flag
  - Path-prefix scoping restricts results to a node slug or slug prefix

  ## Match shape

      %{
        slug:     "operation-weekly-review/pricing",  # context slug / node
        scale:    :section,              # document | section | paragraph | chunk
        intent:   :record_fact,          # one of the 10 canonical intent values
        sn_ratio: 0.82,
        modality: :text,
        snippet:  "…200-char window…",
        score:    0.943
      }

  ## Options

  - `:workspace_id` — workspace to search (default: "default")
  - `:path_prefix`  — restrict to node slugs that start with this string
  - `:intent`       — filter by intent atom (e.g. `:record_fact`)
  - `:scale`        — filter by scale atom (`:document` | `:section` | `:paragraph` | `:chunk`)
  - `:modality`     — filter by modality atom (e.g. `:text`)
  - `:limit`        — max results (default 25)
  - `:literal`      — when true, skip semantic search and match only FTS BM25
  - `:snippet_len`  — bytes for snippet window (default 200)
  """

  require Logger

  alias OptimalEngine.Retrieval.Search, as: SearchEngine
  alias OptimalEngine.Store

  @default_limit 25
  @default_snippet_len 200

  @valid_intents ~w(
    request_info propose_decision record_fact express_concern commit_action
    reference narrate reflect specify measure
  )a

  @valid_scales ~w(document section paragraph chunk)a

  @type match :: %{
          slug: String.t(),
          scale: atom(),
          intent: atom() | nil,
          sn_ratio: float() | nil,
          modality: atom() | nil,
          snippet: String.t(),
          score: float()
        }

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Runs a hybrid grep over the given workspace.

  Returns `{:ok, [match()]}` or `{:error, reason}`.
  """
  @spec grep(String.t(), keyword()) :: {:ok, [match()]} | {:error, term()}
  def grep(query, opts \\ []) when is_binary(query) do
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    limit = Keyword.get(opts, :limit, @default_limit)
    literal? = Keyword.get(opts, :literal, false)
    intent_filter = validate_intent(Keyword.get(opts, :intent))
    scale_filter = validate_scale(Keyword.get(opts, :scale))
    modality_filter = Keyword.get(opts, :modality)
    path_prefix = Keyword.get(opts, :path_prefix)
    snippet_len = Keyword.get(opts, :snippet_len, @default_snippet_len)

    # Semantic search returns context-level hits (each context has many chunks).
    # We then explode down to the chunk level for fine-grained filtering,
    # snippet extraction, and signal-trace annotation.
    search_opts =
      [workspace_id: workspace_id, limit: limit * 4]
      |> maybe_put(:node, extract_node_filter(path_prefix))

    raw_result =
      try do
        if literal? do
          SearchEngine.search(query, Keyword.put(search_opts, :expand_intent, false))
        else
          SearchEngine.search(query, search_opts)
        end
      catch
        :exit, {:timeout, _} ->
          Logger.warning("Grep search timed out for query=#{inspect(query)}")
          {:ok, []}

        :exit, reason ->
          Logger.warning("Grep search exited for query=#{inspect(query)} reason=#{inspect(reason)}")
          {:error, reason}
      end

    case raw_result do
      {:ok, contexts} ->
        matches =
          contexts
          |> Enum.flat_map(
            &explode_to_chunks(&1, query, intent_filter, scale_filter, modality_filter, snippet_len)
          )
          |> apply_path_prefix(path_prefix)
          |> Enum.sort_by(& &1.score, :desc)
          |> Enum.take(limit)

        {:ok, matches}

      {:error, _} = err ->
        err
    end
  end

  # ---------------------------------------------------------------------------
  # Private: chunk explosion + annotation
  # ---------------------------------------------------------------------------

  # For each context returned by Search, pull its stored chunks from the
  # `chunks` table, apply filters, extract a snippet, and build match maps.
  #
  # When no chunks exist (pre-Phase-1 data or empty FTS hit), we fall back
  # to a document-level pseudo-chunk using the context's own content.
  defp explode_to_chunks(ctx, query, intent_filter, scale_filter, modality_filter, snippet_len) do
    node = Map.get(ctx, :node) || ""
    sn = Map.get(ctx, :sn_ratio)
    base_score = Map.get(ctx, :score) || 0.0

    chunks = fetch_chunks(ctx.id, scale_filter, modality_filter)

    if chunks == [] do
      # Fallback: emit one document-level pseudo-match from the context
      content = Map.get(ctx, :content) || Map.get(ctx, :l0_abstract) || ""

      case snippet_for(content, query, snippet_len) do
        nil ->
          []

        snippet ->
          [
            %{
              slug: node,
              scale: :document,
              intent: nil,
              sn_ratio: sn,
              modality: :text,
              snippet: snippet,
              score: Float.round(base_score, 4)
            }
          ]
      end
    else
      chunks
      |> maybe_filter_intent(intent_filter)
      |> Enum.flat_map(fn chunk ->
        text = Map.get(chunk, :text, "")

        case snippet_for(text, query, snippet_len) do
          nil ->
            []

          snippet ->
            [
              %{
                slug: node,
                scale: atomize(Map.get(chunk, :scale), :chunk),
                intent: atomize(Map.get(chunk, :intent), nil),
                sn_ratio: sn,
                modality: atomize(Map.get(chunk, :modality), :text),
                snippet: snippet,
                score: Float.round(chunk_score(base_score, chunk), 4)
              }
            ]
        end
      end)
    end
  end

  # Pull chunk rows with optional scale + modality filters, also join
  # the intent from the `intents` table. Placeholders are numbered
  # contiguously starting at ?1 (signal_id), then ?2/?3 if scale and
  # modality filters are present — SQLite rejects gaps in numbering.
  defp fetch_chunks(context_id, scale_filter, modality_filter) do
    next_placeholder = 2

    {scale_clause, scale_val, next_placeholder} =
      if scale_filter do
        {" AND ch.scale = ?#{next_placeholder}", [to_string(scale_filter)], next_placeholder + 1}
      else
        {"", [], next_placeholder}
      end

    {modality_clause, modality_val} =
      if modality_filter do
        {" AND ch.modality = ?#{next_placeholder}", [to_string(modality_filter)]}
      else
        {"", []}
      end

    sql = """
    SELECT ch.id, ch.scale, ch.modality, ch.text,
           i.intent, i.confidence
    FROM chunks ch
    LEFT JOIN intents i ON i.chunk_id = ch.id
    WHERE ch.signal_id = ?1
      AND ch.scale != 'document'
      #{scale_clause}
      #{modality_clause}
    ORDER BY CASE ch.scale
      WHEN 'section'   THEN 1
      WHEN 'paragraph' THEN 2
      WHEN 'chunk'     THEN 3
      ELSE 4
    END, ch.id
    LIMIT 50
    """

    params = [context_id] ++ scale_val ++ modality_val

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, scale, modality, text, intent, _conf] ->
          %{
            id: id,
            scale: scale,
            modality: modality,
            text: text || "",
            intent: intent
          }
        end)

      _ ->
        []
    end
  end

  # Apply intent filter at the Elixir level (simpler than re-querying)
  defp maybe_filter_intent(chunks, nil), do: chunks

  defp maybe_filter_intent(chunks, intent_atom) do
    intent_str = to_string(intent_atom)
    Enum.filter(chunks, fn ch -> Map.get(ch, :intent) == intent_str end)
  end

  # ---------------------------------------------------------------------------
  # Private: snippet extraction
  # ---------------------------------------------------------------------------

  # Find the query terms in the text and return a window of `len` bytes
  # centered on the first match. Returns nil if the text is empty or the
  # query is empty.
  defp snippet_for("", _query, _len), do: nil
  defp snippet_for(nil, _query, _len), do: nil

  defp snippet_for(text, query, len) do
    # Try to find the first query term in the text (case-insensitive)
    terms =
      query
      |> String.downcase()
      |> String.split(~r/\s+/)
      |> Enum.reject(&(byte_size(&1) < 2))

    text_lower = String.downcase(text)

    offset =
      terms
      |> Enum.find_value(fn term ->
        case :binary.match(text_lower, term) do
          {pos, _len} -> pos
          :nomatch -> nil
        end
      end)

    start =
      case offset do
        nil -> 0
        pos -> max(pos - div(len, 4), 0)
      end

    raw = String.slice(text, start, len)
    trimmed = String.trim(raw)

    if trimmed == "", do: nil, else: trimmed
  end

  # ---------------------------------------------------------------------------
  # Private: helpers
  # ---------------------------------------------------------------------------

  # Weight a chunk's contribution to the base context score. Finer-grained
  # chunks score a little lower than section-level hits since they are
  # narrower in scope and may match on incidental terms.
  defp chunk_score(base, %{scale: "section"}), do: base * 1.0
  defp chunk_score(base, %{scale: "paragraph"}), do: base * 0.95
  defp chunk_score(base, %{scale: "chunk"}), do: base * 0.90
  defp chunk_score(base, _), do: base * 0.85

  defp apply_path_prefix(matches, nil), do: matches

  defp apply_path_prefix(matches, prefix) do
    Enum.filter(matches, fn m -> String.starts_with?(m.slug, prefix) end)
  end

  # Derive a node filter from a path prefix. The prefix may include a trailing
  # slash (e.g. "operation-weekly-review/") — strip it before passing as a node slug.
  # If the prefix contains a slash in the middle (e.g. "operation-weekly-review/pricing")
  # we take the first segment as the node and apply path_prefix post-filter.
  defp extract_node_filter(nil), do: nil

  defp extract_node_filter(prefix) do
    prefix
    |> String.trim_trailing("/")
    |> String.split("/", parts: 2)
    |> List.first()
    |> case do
      "" -> nil
      node -> node
    end
  end

  defp maybe_put(opts, _key, nil), do: opts
  defp maybe_put(opts, key, val), do: Keyword.put(opts, key, val)

  defp validate_intent(nil), do: nil
  defp validate_intent(atom) when is_atom(atom) and atom in @valid_intents, do: atom

  defp validate_intent(str) when is_binary(str) do
    try do
      atom = String.to_existing_atom(str)
      if atom in @valid_intents, do: atom, else: nil
    rescue
      ArgumentError -> nil
    end
  end

  defp validate_intent(_), do: nil

  defp validate_scale(nil), do: nil
  defp validate_scale(atom) when is_atom(atom) and atom in @valid_scales, do: atom

  defp validate_scale(str) when is_binary(str) do
    try do
      atom = String.to_existing_atom(str)
      if atom in @valid_scales, do: atom, else: nil
    rescue
      ArgumentError -> nil
    end
  end

  defp validate_scale(_), do: nil

  defp atomize(nil, default), do: default
  defp atomize(val, _default) when is_atom(val), do: val

  defp atomize(str, default) when is_binary(str) do
    try do
      String.to_existing_atom(str)
    rescue
      ArgumentError -> default
    end
  end
end
