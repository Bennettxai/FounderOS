defmodule OptimalEngine.Retrieval.ContextAssembler do
  @moduledoc """
  Tiered context assembly following tiered context systems's hierarchy.

  ## Tiers (the correct model)

  - **L0** — Structural inventory: what exists in the library.
    Nodes, skills, resources, memory counts, system state.
    Always loaded. The agent sees "here's what's available."

  - **L1** — Per-file summaries/abstracts: one-liner per relevant file.
    Query-driven. BM25 + vector + graph fusion returns matching files
    with their L0 abstracts. The agent sees "here are the relevant files
    and what each one is about."

  - **L2** — Full content: loaded on demand for top results.
    Only for the files the agent actually needs to read in full.
    Deep retrieval with decision history.

  ## Usage

      # Full tiered assembly for a query
      {:ok, context} = ContextAssembler.assemble("Platform Launch pricing")

      # Just the inventory
      {:ok, l0} = ContextAssembler.l0()

      # Just summaries for a query
      {:ok, l1} = ContextAssembler.l1("pricing")
  """

  require Logger

  alias OptimalEngine.Retrieval.L0Cache, as: L0Cache
  alias OptimalEngine.Retrieval.Search, as: SearchEngine
  alias OptimalEngine.Store
  alias OptimalEngine.Bridge.Knowledge
  alias OptimalEngine.Retrieval.MCTS

  @default_budgets %{l0: 3_000, l1: 10_000, l2: 50_000}
  @rrf_k 60

  @type assembled :: %{
          l0: String.t(),
          l1: String.t(),
          l2: String.t(),
          total_tokens: non_neg_integer(),
          sources: [String.t()],
          search_scores: [map()]
        }

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Assembles full tiered context for a query.

  L0 = inventory (what exists)
  L1 = relevant file summaries (what matches)
  L2 = full content (what you need to read)

  Return shape (cleanup rule 7): `{:ok, map}` of token-budgeted markdown tiers
  built from **raw search hits** over compatibility `contexts` rows. This is a
  precursor surface — not the governed
  `OptimalEngine.MemoryCore.ContextPackage`; for that, use
  `OptimalEngine.MemoryCore.RetrievalCoordinator.retrieve/2`.
  """
  @spec assemble(String.t(), keyword()) :: {:ok, assembled()}
  def assemble(query, opts \\ []) do
    budgets = Keyword.get(opts, :tier_budgets, @default_budgets)

    # L0 — structural inventory (always loaded, query-independent)
    l0 = build_l0(budgets.l0)

    # L1 — per-file summaries for matching files (query-driven)
    {l1, search_scores} = build_l1(query, budgets.l1, opts)

    # L2 — full content for top results (deep retrieval)
    l2 = build_l2(query, search_scores, budgets.l2, opts)

    total_tokens = estimate_tokens(l0) + estimate_tokens(l1) + estimate_tokens(l2)
    sources = extract_sources(search_scores)

    {:ok,
     %{
       l0: l0,
       l1: l1,
       l2: l2,
       total_tokens: total_tokens,
       sources: sources,
       search_scores: search_scores
     }}
  end

  @doc "Returns L0 — the structural inventory of the library."
  @spec l0() :: {:ok, String.t()}
  def l0 do
    content = build_l0(@default_budgets.l0)
    {:ok, content}
  end

  @doc "Returns L1 — per-file summaries for a query."
  @spec l1(String.t(), keyword()) :: {:ok, String.t()}
  def l1(query, opts \\ []) do
    budget = Keyword.get(opts, :budget, @default_budgets.l1)
    {content, _scores} = build_l1(query, budget, opts)
    {:ok, content}
  end

  @doc """
  Performs real Reciprocal Rank Fusion across **independent** candidate
  lists:

    * FTS5 / BM25 lexical recall (`Search` with vectors disabled)
    * vector / semantic recall (`Search.search` hybrid path, when embeddings
      are healthy — degrades gracefully to FTS otherwise)
    * graph 1-hop boosted ranking (when the knowledge graph is available)
    * temporal-decay ranking (freshness-first ordering of the union)

  Each list contributes `1/(k + rank)` per document; the fused score is the
  sum across lists. Documents that appear in multiple independent lists rise
  to the top — the canonical RRF property. Lists that are empty (e.g. no
  embeddings) simply contribute nothing, so fusion degrades to FTS-only
  cleanly.
  """
  @spec fused_search(String.t(), keyword()) :: {:ok, [map()]}
  def fused_search(query, opts \\ []) do
    limit = Keyword.get(opts, :limit, 20)
    recall = limit * 3

    # FTS-only list: force vectors off so this list is a genuinely
    # independent lexical ranking, not the already-fused hybrid output.
    fts_results =
      case SearchEngine.search(query, limit: recall, vector_enabled: false) do
        {:ok, results} -> results
        _ -> []
      end

    # Semantic list: the hybrid path. When embeddings are absent this
    # returns the FTS ranking, which is fine — RRF tolerates correlated
    # lists, and the dedicated FTS list above keeps lexical signal intact.
    vector_results =
      case SearchEngine.search(query, limit: recall) do
        {:ok, results} -> results
        _ -> []
      end

    # Graph 1-hop list: reorder the union by graph adjacency to the query.
    graph_results = safe_graph_boost(fts_results ++ vector_results, query)

    # Temporal list: freshness-first ordering of everything seen.
    temporal_results = temporal_ranked(fts_results ++ vector_results ++ graph_results)

    fused =
      reciprocal_rank_fusion([
        fts_results,
        vector_results,
        graph_results,
        temporal_results
      ])

    {:ok, Enum.take(fused, limit)}
  end

  # Order the union by recency (modified_at, then created_at). Dedupe by id
  # so a document gets a single temporal rank.
  defp temporal_ranked(results) do
    results
    |> Enum.uniq_by(&Map.get(&1, :id))
    |> Enum.sort_by(&temporal_key/1, {:desc, DateTime})
  end

  defp temporal_key(result) do
    Map.get(result, :modified_at) || Map.get(result, :created_at) || ~U[1970-01-01 00:00:00Z]
  end

  defp safe_graph_boost(results, query) do
    Knowledge.graph_boost(results, query)
  rescue
    _ -> results
  catch
    :exit, _ -> results
  end

  # ---------------------------------------------------------------------------
  # Private: Tier Builders
  # ---------------------------------------------------------------------------

  # L0 — Structural inventory from cache
  defp build_l0(budget) do
    content = L0Cache.get()
    truncate_to_budget(content, budget)
  end

  # L1 — Per-file abstracts/summaries for relevant files
  defp build_l1(query, budget, opts) do
    limit = Keyword.get(opts, :limit, 20)

    {:ok, results} = fused_search(query, limit: limit)

    # Build L1 as a list of file summaries — one entry per matching file
    {content, _} =
      Enum.reduce(results, {"", 0}, fn result, {acc, tokens} ->
        title = Map.get(result, :title, "Untitled")
        node = Map.get(result, :node, "")
        score = Map.get(result, :score, 0)
        l0_abstract = Map.get(result, :l0_abstract, "") || ""
        uri = Map.get(result, :uri, "")
        type = Map.get(result, :type, :signal)

        # L1 shows the abstract (one-liner) per file, not the full overview
        entry =
          "- **#{title}** [#{type} | #{node} | score: #{Float.round(score, 3)}]\n" <>
            "  #{l0_abstract}\n" <>
            if(uri != "", do: "  `#{uri}`\n", else: "")

        entry_tokens = estimate_tokens(entry)

        if tokens + entry_tokens <= budget do
          {acc <> entry, tokens + entry_tokens}
        else
          {acc, tokens}
        end
      end)

    header = "## Matching Files (#{length(results)} results)\n\n"

    scores =
      Enum.map(results, fn r ->
        %{
          id: Map.get(r, :id),
          title: Map.get(r, :title),
          node: Map.get(r, :node),
          score: Map.get(r, :score),
          uri: Map.get(r, :uri)
        }
      end)

    {header <> content, scores}
  end

  # L2 — Full content for top results.
  #
  # Selection is budget-aware via MCTS (config `retrieval.mcts_enabled`,
  # default on): instead of blindly taking the top-5, MCTS maximizes concept
  # coverage within the L2 token budget so near-duplicate top hits don't
  # crowd out distinct, lower-ranked context. Falls back to greedy when the
  # flag is off.
  defp build_l2(_query, search_scores, budget, _opts) do
    top_ids =
      search_scores
      |> select_l2_candidates(budget)
      |> Enum.map(& &1.id)
      |> Enum.reject(&is_nil/1)

    {content, _} =
      Enum.reduce(top_ids, {"", 0}, fn id, {acc, tokens} ->
        case Store.get_context(id) do
          {:ok, ctx} ->
            full_text = Map.get(ctx, :content, "") || ""
            title = Map.get(ctx, :title, "")
            node = Map.get(ctx, :node, "")
            uri = Map.get(ctx, :uri, "")

            entry =
              "## #{title} (#{node})\n" <>
                if(uri != "", do: "> `#{uri}`\n\n", else: "\n") <>
                full_text <> "\n\n---\n"

            entry_tokens = estimate_tokens(entry)

            if tokens + entry_tokens <= budget do
              {acc <> entry, tokens + entry_tokens}
            else
              {acc, tokens}
            end

          _ ->
            {acc, tokens}
        end
      end)

    # Decision history for related topics
    decision_content = load_related_decisions(top_ids, budget - estimate_tokens(content))
    content <> decision_content
  end

  # Build budget-aware L2 candidates and select with MCTS (or greedy fallback).
  # Candidates carry a token estimate (full content length when cheaply
  # available, else l0_abstract proxy) and concepts (from title + abstract)
  # so MCTS can reason about coverage vs. budget.
  defp select_l2_candidates(search_scores, budget) do
    candidates =
      search_scores
      |> Enum.reject(&is_nil(&1[:id]))
      |> Enum.map(fn s ->
        {tokens, concepts_text} =
          case Store.get_context(s.id) do
            {:ok, ctx} ->
              content = Map.get(ctx, :content) || ""
              text = "#{Map.get(ctx, :title, "")} #{Map.get(ctx, :l0_abstract, "")}"
              {max(div(String.length(content), 4), 1), text}

            _ ->
              {1, to_string(s[:title] || "")}
          end

        %{
          id: s.id,
          score: (s[:score] || 0.0) * 1.0,
          tokens: tokens,
          concepts: concept_tokens(concepts_text)
        }
      end)

    result =
      if MCTS.enabled?() do
        MCTS.select(candidates, budget)
      else
        MCTS.greedy(candidates, budget)
      end

    result.selected
  end

  defp concept_tokens(text) do
    text
    |> String.downcase()
    |> String.split(~r/[^\p{L}\p{N}]+/u, trim: true)
    |> Enum.reject(&(String.length(&1) < 3))
    |> MapSet.new()
  end

  # ---------------------------------------------------------------------------
  # Reciprocal Rank Fusion
  # ---------------------------------------------------------------------------

  @doc """
  Fuse independent ranked lists via Reciprocal Rank Fusion. Each list
  contributes `1/(k + rank)` per document; documents appearing across
  multiple lists accumulate score and rise to the top. Public for testing
  the fusion property directly.
  """
  @spec fuse([[map()]]) :: [map()]
  def fuse(result_lists), do: reciprocal_rank_fusion(result_lists)

  defp reciprocal_rank_fusion(result_lists) do
    score_map =
      result_lists
      |> Enum.reduce(%{}, fn results, acc ->
        results
        |> Enum.with_index(1)
        |> Enum.reduce(acc, fn {result, rank}, inner_acc ->
          # Stable key so the *same* document fuses across independent lists.
          # Falls back to uri/title before a ref so id-less rows still merge.
          id = Map.get(result, :id) || Map.get(result, :uri) || Map.get(result, :title)
          rrf_score = 1.0 / (rrf_k() + rank)

          Map.update(inner_acc, id, {rrf_score, result}, fn {existing_score, existing_result} ->
            {existing_score + rrf_score, existing_result}
          end)
        end)
      end)

    score_map
    |> Enum.map(fn {_id, {fused_score, result}} ->
      Map.put(result, :score, Float.round(fused_score, 6))
    end)
    |> Enum.sort_by(& &1.score, :desc)
  end

  # ---------------------------------------------------------------------------
  # Private: Helpers
  # ---------------------------------------------------------------------------

  defp load_related_decisions(context_ids, remaining_budget) when remaining_budget > 0 do
    placeholders =
      context_ids
      |> Enum.with_index(1)
      |> Enum.map_join(", ", fn {_, i} -> "?#{i}" end)

    if placeholders == "" do
      ""
    else
      sql = """
      SELECT topic, decision, rationale, decided_at
      FROM decisions
      WHERE context_id IN (#{placeholders})
      ORDER BY decided_at DESC
      LIMIT 5
      """

      case Store.raw_query(sql, context_ids) do
        {:ok, rows} when rows != [] ->
          lines =
            Enum.map(rows, fn [topic, decision, rationale, date] ->
              "- **#{topic}** (#{date}): #{decision}\n  _Rationale: #{rationale}_"
            end)

          "\n## Related Decisions\n\n" <> Enum.join(lines, "\n")

        _ ->
          ""
      end
    end
  end

  defp load_related_decisions(_, _), do: ""

  defp extract_sources(scores) do
    scores
    |> Enum.map(fn s -> s[:uri] || s[:title] || "unknown" end)
    |> Enum.uniq()
  end

  defp rrf_k do
    Application.get_env(:optimal_engine, :retrieval, [])
    |> Keyword.get(:rrf_k, @rrf_k)
  end

  defp estimate_tokens(text) when is_binary(text), do: div(String.length(text), 4)
  defp estimate_tokens(_), do: 0

  defp truncate_to_budget(text, budget) do
    max_chars = budget * 4

    if String.length(text) > max_chars do
      String.slice(text, 0, max_chars) <> "\n\n[...truncated at token budget]"
    else
      text
    end
  end
end
