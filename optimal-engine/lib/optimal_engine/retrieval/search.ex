defmodule OptimalEngine.Retrieval.Search do
  @moduledoc """
  Hybrid search: SQLite FTS5 BM25 + temporal decay + S/N ratio scoring.

  ## Scoring formula

      final_score = bm25_score * temporal_factor * sn_ratio_boost

  Where:
  - `bm25_score`       — native SQLite FTS5 BM25 rank (negated, lower is better in SQLite)
  - `temporal_factor`  — exponential decay based on age and genre half-life
  - `sn_ratio_boost`   — linear boost from signal quality (0.0–1.0)

  Each result carries a `score` field on the Context struct.

  ## Options

  - `:type`   — filter by context type atom (`:signal`, `:resource`, `:memory`, `:skill`)
  - `:node`   — filter by node ID (e.g. `"operator"`)
  - `:genre`  — filter by genre (signals only)
  - `:uri`    — scope to a URI prefix (e.g. `"optimal://nodes/project-platform-launch/"`)
  - `:limit`  — max results (default 10)
  - `:offset` — pagination offset (default 0)
  - `:min_score` — drop results below this score (default 0.0)

  ## Backward compatibility

  `search/2` returns `{:ok, [%Context{}]}`. Each context has a `.signal` field
  if it is of type `:signal`, so callers that need `Signal.t()` can use
  `OptimalEngine.Context.to_signal/1` on each result.
  """

  use GenServer
  require Logger

  alias OptimalEngine.Context
  alias OptimalEngine.Retrieval.IntentAnalyzer, as: IntentAnalyzer
  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store
  alias OptimalEngine.Routing
  alias OptimalEngine.Store.Vectors, as: VectorStore
  alias OptimalEngine.Bridge.Knowledge, as: BridgeKnowledge
  alias OptimalEngine.Bridge.Memory, as: BridgeMemory

  @default_limit 10
  @default_half_life 720

  # ---------------------------------------------------------------------------
  # Client API
  # ---------------------------------------------------------------------------

  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Executes a hybrid search query across all context types.

  Return shape (cleanup rule 7): **raw search hits** over compatibility
  `contexts` rows — not a `RetrievalPackage` or `ContextPackage`. Governed
  recall lives in `OptimalEngine.MemoryCore.RetrievalCoordinator`.

  Returns `{:ok, [%Context{}]}` with `:score` set on each result,
  or `{:error, reason}`.

  ## Examples

      # Search everything
      SearchEngine.search("Platform Launch pricing")

      # Only signals
      SearchEngine.search("Platform Launch", type: :signal)

      # Resources only
      SearchEngine.search("API docs", type: :resource)

      # Scoped to a URI prefix
      SearchEngine.search("context", uri: "optimal://nodes/project-platform-launch/")
  """
  @spec search(String.t(), keyword()) :: {:ok, [Context.t()]} | {:error, term()}
  def search(query, opts \\ []) when is_binary(query) do
    GenServer.call(__MODULE__, {:search, query, opts}, 15_000)
  end

  @doc "Searches within a specific node. Convenience wrapper around `search/2`."
  @spec search_node(String.t(), String.t(), keyword()) ::
          {:ok, [Context.t()]} | {:error, term()}
  def search_node(node, query, opts \\ []) do
    search(query, Keyword.put(opts, :node, node))
  end

  # ---------------------------------------------------------------------------
  # GenServer Callbacks
  # ---------------------------------------------------------------------------

  @impl true
  def init(_opts) do
    topology =
      case Routing.load() do
        {:ok, t} -> t
        {:error, _} -> %{half_lives: %{"default" => @default_half_life}}
      end

    {:ok, %{topology: topology}}
  end

  @impl true
  def handle_call({:search, query, opts}, _from, state) do
    start = System.monotonic_time(:millisecond)

    # `embed_healthy?` is stronger than `available?` — it verifies Ollama
    # actually returns a non-empty vector, not just that /api/tags
    # responds. When embeddings are flaky (missing model / partial
    # load / cold start) we skip the whole hybrid path and serve
    # FTS-only results so retrieval stays sub-second instead of
    # queuing behind a slow embed.
    # Per-call override: callers (e.g. the RRF fusion in ContextAssembler)
    # can force a pure FTS list with `vector_enabled: false` so they get a
    # genuinely independent lexical ranking to fuse against.
    vector_allowed = Keyword.get(opts, :vector_enabled, true)

    raw_result =
      if vector_allowed and hybrid_enabled?() and Ollama.embed_healthy?() do
        case do_hybrid_search(query, opts, state.topology) do
          {:ok, _} = ok -> ok
          {:error, _} -> do_search(query, opts, state.topology)
        end
      else
        do_search(query, opts, state.topology)
      end

    # Phase 1: principal-scoped ACL filter. No `:principal` → no filter
    # (backwards-compatible with pre-Phase-1 callers). When `:principal`
    # is set, drop any result the principal cannot read per ACLs.
    result = filter_by_principal(raw_result, opts)

    elapsed = System.monotonic_time(:millisecond) - start

    :telemetry.execute(
      [:optimal_engine, :search, :query],
      %{duration_ms: elapsed, result_count: result_count(result)},
      %{query: query}
    )

    emit_audit(query, opts, result, elapsed)

    {:reply, result, state}
  end

  # ---------------------------------------------------------------------------
  # Private: principal-scoped ACL filter
  # ---------------------------------------------------------------------------

  defp filter_by_principal({:ok, hits}, opts) when is_list(hits) do
    case Keyword.get(opts, :principal) do
      nil ->
        {:ok, hits}

      principal_id when is_binary(principal_id) ->
        tenant_id = Keyword.get(opts, :tenant_id, OptimalEngine.Tenancy.Tenant.default_id())

        filtered =
          Enum.filter(hits, fn hit ->
            uri = Map.get(hit, :uri) || Map.get(hit, :id)

            uri == nil or
              OptimalEngine.Identity.ACL.can?(principal_id, uri, :read, tenant_id: tenant_id)
          end)

        {:ok, filtered}
    end
  end

  defp filter_by_principal(other, _opts), do: other

  defp emit_audit(query, opts, result, elapsed) do
    case Keyword.get(opts, :principal) do
      nil ->
        :ok

      principal_id when is_binary(principal_id) ->
        count =
          case result do
            {:ok, hits} -> length(hits)
            _ -> 0
          end

        OptimalEngine.Audit.Logger.log("retrieval.executed",
          tenant_id: Keyword.get(opts, :tenant_id, OptimalEngine.Tenancy.Tenant.default_id()),
          principal: principal_id,
          latency_ms: elapsed,
          metadata: %{query: query, result_count: count}
        )

        :ok
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Hybrid Search Config
  # ---------------------------------------------------------------------------

  defp hybrid_enabled? do
    config = Application.get_env(:optimal_engine, :hybrid_search, [])
    Keyword.get(config, :vector_enabled, true)
  end

  defp hybrid_alpha do
    config = Application.get_env(:optimal_engine, :hybrid_search, [])
    Keyword.get(config, :alpha, 0.6)
  end

  # ---------------------------------------------------------------------------
  # Private: Hybrid Search Pipeline
  # ---------------------------------------------------------------------------

  defp do_hybrid_search(query, opts, topology) do
    limit = Keyword.get(opts, :limit, @default_limit)
    min_score = Keyword.get(opts, :min_score, 0.0)
    alpha = hybrid_alpha()

    # Step 1: Intent analysis. LLM-based expansion costs 5-8 seconds on a
    # local generate call, which is not acceptable on the hot retrieval
    # path. Use the fast regex fallback by default; opt in with
    # `expand_intent: true` for offline or background workloads.
    intent =
      if Keyword.get(opts, :expand_intent, false) do
        case IntentAnalyzer.analyze(query) do
          {:ok, i} -> i
          _ -> %{expanded_query: query, key_entities: [], node_hints: []}
        end
      else
        %{expanded_query: query, key_entities: [], node_hints: []}
      end

    # Step 2: Run FTS5 search (existing pipeline) — get more results for merging
    fts_opts = Keyword.put(opts, :limit, limit * 3)

    fts_results =
      case do_search(intent.expanded_query, fts_opts, topology) do
        {:ok, results} -> results
        _ -> []
      end

    # Step 3: Run vector search. Ollama sometimes returns an empty
    # embedding list for degenerate queries (empty strings, very short
    # tokens, partial model load). Skip the vector hop in that case so
    # we don't chase a zero-vector through VectorStore.
    {vector_results, chunk_results} =
      case Ollama.embed(query) do
        {:ok, query_embedding} when is_list(query_embedding) and query_embedding != [] ->
          vector_opts = [
            limit: limit * 3,
            min_similarity: 0.1,
            workspace_id: Keyword.get(opts, :workspace_id, "default")
          ]

          # Add type/node filters if present in opts
          vector_opts =
            if t = Keyword.get(opts, :type),
              do: Keyword.put(vector_opts, :type_filter, to_string(t)),
              else: vector_opts

          vector_opts =
            if n = Keyword.get(opts, :node),
              do: Keyword.put(vector_opts, :node_filter, n),
              else: vector_opts

          context_pairs =
            case VectorStore.search(query_embedding, vector_opts) do
              {:ok, pairs} -> pairs
              _ -> []
            end

          candidate_ids = Enum.map(fts_results, & &1.id)

          chunk_pairs =
            case VectorStore.rerank_contexts(query_embedding, candidate_ids,
                   limit: limit * 3,
                   workspace_id: Keyword.get(opts, :workspace_id, "default")
                 ) do
              {:ok, pairs} -> pairs
              _ -> []
            end

          {context_pairs, chunk_pairs}

        _ ->
          {[], []}
      end

    # Step 4: Merge results
    # Build a map of context_id -> fts_score (normalized)
    max_fts = fts_results |> Enum.map(& &1.score) |> Enum.max(fn -> 1.0 end)
    fts_map = Map.new(fts_results, fn ctx -> {ctx.id, ctx.score / max(max_fts, 0.001)} end)

    # Build a map of context_id -> vector_similarity
    vector_map =
      (vector_results ++ chunk_results)
      |> Enum.reduce(%{}, fn {id, score}, scores ->
        Map.update(scores, id, score, &max(&1, score))
      end)

    # Union of all context IDs
    all_ids = MapSet.union(MapSet.new(Map.keys(fts_map)), MapSet.new(Map.keys(vector_map)))

    # Score each ID: alpha * fts_normalized + (1-alpha) * vector_similarity
    scored =
      Enum.map(all_ids, fn id ->
        fts_score = Map.get(fts_map, id, 0.0)
        vec_score = Map.get(vector_map, id, 0.0)
        combined = alpha * fts_score + (1.0 - alpha) * vec_score
        {id, combined}
      end)

    # Apply intent-based adjustments
    scored = apply_intent_adjustments(scored, intent)

    # Sort, filter, limit
    scored =
      scored
      |> Enum.filter(fn {_id, score} -> score >= min_score end)
      |> Enum.sort_by(fn {_id, score} -> score end, :desc)
      |> Enum.take(limit)

    # Resolve full Context structs for results
    # First, build a lookup from FTS results (which already have full Context structs)
    fts_lookup = Map.new(fts_results, fn ctx -> {ctx.id, ctx} end)

    final_results =
      Enum.map(scored, fn {id, score} ->
        case Map.get(fts_lookup, id) do
          nil ->
            # This result came from vector search only — need to load the context
            case load_context(id, Keyword.get(opts, :workspace_id, "default")) do
              {:ok, ctx} -> %{ctx | score: Float.round(score, 4)}
              _ -> nil
            end

          ctx ->
            %{ctx | score: Float.round(score, 4)}
        end
      end)
      |> Enum.reject(&is_nil/1)

    # Apply graph boost (same as FTS-only path)
    final_results = try_graph_boost(final_results, query)

    {:ok, final_results}
  rescue
    e ->
      Logger.warning("[SearchEngine] Hybrid search failed: #{inspect(e)}, falling back to FTS")
      do_search(query, opts, topology)
  end

  defp load_context(id, workspace_id) do
    sql = """
    SELECT id, uri, type, path, title,
      l0_abstract, l1_overview, content,
      mode, genre, signal_type, format, structure,
      node, sn_ratio, entities,
      created_at, modified_at, valid_from, valid_until, supersedes,
      routed_to, metadata, workspace_id
    FROM contexts WHERE id = ?1 AND workspace_id = ?2
    """

    case Store.raw_query(sql, [id, workspace_id]) do
      {:ok, [row]} -> {:ok, Context.from_row(row)}
      _ -> {:error, :not_found}
    end
  end

  defp apply_intent_adjustments(scored, intent) do
    Enum.map(scored, fn {id, score} ->
      adjusted =
        if id_matches_node_hints?(id, intent.node_hints) do
          score * 1.2
        else
          score
        end

      {id, adjusted}
    end)
  end

  # Check if a context ID matches any node hints.
  # For efficiency, skip the DB query when no hints are provided.
  defp id_matches_node_hints?(_id, []), do: false

  defp id_matches_node_hints?(id, node_hints) do
    case Store.raw_query("SELECT node FROM contexts WHERE id = ?1", [id]) do
      {:ok, [[node]]} -> node in node_hints
      _ -> false
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Search Pipeline
  # ---------------------------------------------------------------------------

  defp do_search(query, opts, topology) do
    limit = Keyword.get(opts, :limit, @default_limit)
    offset = Keyword.get(opts, :offset, 0)
    type_filter = Keyword.get(opts, :type)
    node_filter = Keyword.get(opts, :node)
    genre_filter = Keyword.get(opts, :genre)
    uri_prefix = Keyword.get(opts, :uri)
    min_score = Keyword.get(opts, :min_score, 0.0)
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    # Resolve URI prefix to node filter if given
    {node_filter, type_filter} = apply_uri_filter(uri_prefix, node_filter, type_filter)

    fts_query = sanitize_fts_query(query)

    {sql, params} =
      build_fts_sql(
        fts_query,
        type_filter,
        node_filter,
        genre_filter,
        workspace_id,
        limit * 3,
        offset
      )

    with {:ok, rows} <- Store.raw_query(sql, params) do
      now = DateTime.utc_now()

      results =
        rows
        |> Enum.map(&build_result(&1, now, topology))
        |> Enum.filter(&(&1.score >= min_score))
        |> Enum.sort_by(& &1.score, :desc)
        |> Enum.take(limit)

      # Graph boost via OptimalEngine.Knowledge (non-blocking — falls back to unmodified results)
      results = try_graph_boost(results, query)

      # Record search event in episodic memory
      observe_search(query, length(results))

      {:ok, results}
    end
  end

  # Resolve a URI prefix into (node_filter, type_filter) hints
  defp apply_uri_filter(nil, node_filter, type_filter), do: {node_filter, type_filter}

  defp apply_uri_filter(uri_prefix, node_filter, type_filter) do
    alias OptimalEngine.URI

    case URI.parse(uri_prefix) do
      {:ok, parsed} ->
        inferred_type = URI.context_type(parsed)
        inferred_node = URI.node_id(parsed)

        {
          node_filter || inferred_node,
          type_filter || inferred_type
        }

      _ ->
        {node_filter, type_filter}
    end
  end

  # Build the FTS SQL with dynamic WHERE clauses.
  # We query contexts_fts (which has a `type` column) not `signals_fts`.
  # workspace_id is always applied (defaults to "default").
  defp build_fts_sql(fts_query, type_filter, node_filter, genre_filter, workspace_id, limit, offset) do
    base_sql = """
    SELECT
      c.id, c.uri, c.type, c.path, c.title,
      c.l0_abstract, c.l1_overview, c.content,
      c.mode, c.genre, c.signal_type, c.format, c.structure,
      c.node, c.sn_ratio, c.entities,
      c.created_at, c.modified_at, c.valid_from, c.valid_until, c.supersedes,
      c.routed_to, c.metadata, c.workspace_id,
      -bm25(contexts_fts) as bm25_rank
    FROM contexts_fts
    JOIN contexts c ON c.id = contexts_fts.id
    WHERE contexts_fts MATCH ?1
    """

    # workspace_id is always filtered; remaining filters are optional.
    # Build dynamic WHERE clauses with positional params ?2, ?3, ...
    filters =
      [
        {"c.workspace_id = ?", workspace_id},
        type_filter && {"c.type = ?", to_string(type_filter)},
        node_filter && {"c.node = ?", node_filter},
        genre_filter && {"c.genre = ?", genre_filter}
      ]
      |> Enum.reject(&is_nil/1)

    {extra_clauses, extra_values} = Enum.unzip(filters)

    # Renumber placeholders sequentially starting at ?2
    {conditions_sql, _} =
      Enum.reduce(extra_clauses, {"", 2}, fn clause, {acc_sql, n} ->
        numbered = String.replace(clause, "?", "?#{n}")
        {acc_sql <> " AND " <> numbered, n + 1}
      end)

    limit_n = length(extra_values) + 2
    offset_n = limit_n + 1

    final_sql =
      base_sql <>
        conditions_sql <>
        " ORDER BY bm25_rank DESC LIMIT ?#{limit_n} OFFSET ?#{offset_n}"

    params = [fts_query] ++ extra_values ++ [limit, offset]
    {final_sql, params}
  end

  defp build_result(row, now, topology) do
    # Last element is bm25_rank; rest is context columns (24 columns including workspace_id)
    {ctx_row, [bm25_rank]} = Enum.split(row, 24)

    ctx = Context.from_row(ctx_row)

    temporal = temporal_factor(ctx, now, topology)
    sn = ctx.sn_ratio || 0.5
    bm25 = bm25_rank || 1.0

    final_score = bm25 * temporal * (0.5 + sn * 0.5)

    %{ctx | score: Float.round(final_score, 4)}
  end

  @doc false
  def temporal_factor(ctx_or_signal, now, topology) do
    modified_at = extract_modified_at(ctx_or_signal)
    genre = extract_genre(ctx_or_signal)
    compute_decay(modified_at, genre, now, topology)
  end

  defp extract_modified_at(%Context{modified_at: m, created_at: c}), do: m || c
  defp extract_modified_at(%{modified_at: m, created_at: c}), do: m || c

  defp extract_genre(%Context{signal: %{genre: g}}) when is_binary(g), do: g
  defp extract_genre(%Context{}), do: "note"
  defp extract_genre(%{genre: g}) when is_binary(g), do: g
  defp extract_genre(_), do: "note"

  defp compute_decay(nil, _genre, _now, _topology), do: 0.5

  defp compute_decay(modified_at, genre, now, topology) do
    hours_old = DateTime.diff(now, modified_at, :second) / 3600.0
    half_life = Routing.half_life_for(topology, genre)
    decay_constant = :math.log(2) / half_life
    :math.exp(-decay_constant * hours_old)
  end

  defp sanitize_fts_query(query) do
    query
    |> String.replace(~r/[\"*^()]/u, " ")
    |> String.trim()
    |> case do
      "" -> "*"
      q -> q
    end
  end

  # Apply knowledge graph boost to search results (non-blocking)
  defp try_graph_boost(results, query) do
    BridgeKnowledge.graph_boost(results, query)
  rescue
    _ -> results
  catch
    :exit, _ -> results
  end

  # Record search event in episodic memory + SICA
  defp observe_search(query, result_count) do
    BridgeMemory.record_event(:search, %{
      query: query,
      result_count: result_count
    })
  rescue
    _ -> :ok
  end

  defp result_count({:ok, list}), do: length(list)
  defp result_count(_), do: 0
end
