defmodule OptimalEngine.Insight.Reweave do
  @moduledoc """
  Backward pass: given a topic, find related contexts that may need updating.

  The reweaver searches for contexts related to a topic, examines their staleness
  (how long since last modified), and cross-references with the knowledge graph
  to find contexts that mention the same entities but may be outdated.

  With Ollama: generates specific diff suggestions for each stale context.
  Without Ollama: flags potentially outdated contexts with newer context titles.

  Stateless module — no GenServer.
  """

  require Logger
  alias OptimalEngine.Graph
  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store

  @default_max_results 10
  @default_staleness_days 30

  @doc """
  Finds contexts related to `topic` that may need updating.

  Returns `{:ok, [map()]}` where each map contains:
  - `:context_id`  — the context ID
  - `:title`       — context title
  - `:node`        — node the context belongs to
  - `:staleness`   — float 0.0 (fresh) to 1.0 (very stale)
  - `:days_old`    — integer days since last modification
  - `:suggestion`  — update suggestion (LLM-generated or rule-based)
  - `:type`        — `:llm_suggestion` or `:flag`

  ## Options
  - `:max_results`    — max contexts to return (default 10)
  - `:staleness_days` — days before a context is considered fully stale (default 30)
  - `:workspace_id`   — workspace to scope to (default "default")
  """
  @spec reweave(String.t(), keyword()) :: {:ok, [map()]}
  def reweave(topic, opts \\ []) do
    max_results = Keyword.get(opts, :max_results, @default_max_results)
    staleness_days = Keyword.get(opts, :staleness_days, @default_staleness_days)
    ws = Keyword.get(opts, :workspace_id, "default")

    # Step 1: Search for related contexts
    search_results = do_search(topic, max_results * 3, ws)

    # Step 2: Find graph-connected contexts via entity matching
    entity_contexts = find_entity_contexts(topic, ws)

    # Step 3: Merge and deduplicate
    all_context_ids =
      (Enum.map(search_results, & &1.id) ++ entity_contexts)
      |> Enum.uniq()

    # Step 4: Load context metadata, score staleness, filter and rank
    contexts_with_staleness =
      all_context_ids
      |> Enum.map(&load_context_meta(&1, ws))
      |> Enum.reject(&is_nil/1)
      |> Enum.map(fn ctx ->
        days_old = compute_days_old(ctx)
        staleness = compute_staleness(ctx, staleness_days)
        ctx |> Map.put(:staleness, staleness) |> Map.put(:days_old, days_old)
      end)
      |> Enum.filter(fn ctx -> ctx.staleness > 0.0 end)
      |> Enum.sort_by(& &1.staleness, :desc)
      |> Enum.take(max_results)

    # Step 5: Generate suggestions
    suggestions =
      if Ollama.available?() do
        newest = find_newest_context(search_results)

        Enum.map(contexts_with_staleness, fn ctx ->
          generate_llm_suggestion(ctx, topic, newest)
        end)
      else
        recent_titles =
          search_results
          |> Enum.take(3)
          |> Enum.map(& &1.title)

        Enum.map(contexts_with_staleness, fn ctx ->
          %{
            context_id: ctx.id,
            title: ctx.title,
            node: ctx.node,
            staleness: Float.round(ctx.staleness, 2),
            days_old: ctx.days_old,
            suggestion:
              "Potentially outdated — newer contexts exist: #{Enum.join(recent_titles, ", ")}",
            type: :flag
          }
        end)
      end

    {:ok, suggestions}
  rescue
    err ->
      Logger.warning("[Reweaver] reweave/2 failed: #{inspect(err)}")
      {:ok, []}
  end

  # ---------------------------------------------------------------------------
  # Private: Search
  # ---------------------------------------------------------------------------

  defp do_search(query, limit, ws) do
    case GenServer.call(
           OptimalEngine.Retrieval.Search,
           {:search, query, [limit: limit, workspace_id: ws]},
           15_000
         ) do
      {:ok, results} -> results
      _ -> []
    end
  rescue
    _ -> []
  end

  # ---------------------------------------------------------------------------
  # Private: Graph traversal
  # ---------------------------------------------------------------------------

  defp find_entity_contexts(topic, ws) do
    # Entity → workspace membership resolved via contexts (source of truth).
    sql = """
    SELECT DISTINCT e.name FROM entities e
    JOIN contexts c ON c.id = e.context_id AND c.workspace_id = ?2
    WHERE e.name LIKE ?1
    """

    case Store.raw_query(sql, ["%#{topic}%", ws]) do
      {:ok, rows} ->
        rows
        |> List.flatten()
        |> Enum.flat_map(fn name ->
          case Graph.edges_for(name, direction: :out, relation: "mentioned_in", workspace_id: ws) do
            {:ok, edges} -> Enum.map(edges, & &1.target_id)
            _ -> []
          end
        end)

      _ ->
        []
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Metadata loading
  # ---------------------------------------------------------------------------

  defp load_context_meta(id, ws) do
    sql =
      "SELECT id, title, node, modified_at, l0_abstract FROM contexts WHERE id = ?1 AND workspace_id = ?2"

    case Store.raw_query(sql, [id, ws]) do
      {:ok, [[id, title, node, modified_at, l0]]} ->
        %{id: id, title: title, node: node, modified_at: modified_at, l0: l0}

      _ ->
        nil
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Staleness scoring
  # ---------------------------------------------------------------------------

  defp compute_staleness(ctx, staleness_days) do
    case parse_datetime(ctx.modified_at) do
      nil ->
        0.5

      modified ->
        days_old = DateTime.diff(DateTime.utc_now(), modified, :day)
        min(days_old / staleness_days, 1.0)
    end
  end

  defp compute_days_old(ctx) do
    case parse_datetime(ctx.modified_at) do
      nil -> -1
      modified -> DateTime.diff(DateTime.utc_now(), modified, :day)
    end
  end

  defp parse_datetime(nil), do: nil

  defp parse_datetime(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _} ->
        dt

      _ ->
        case NaiveDateTime.from_iso8601(str) do
          {:ok, ndt} -> DateTime.from_naive!(ndt, "Etc/UTC")
          _ -> nil
        end
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Suggestion generation
  # ---------------------------------------------------------------------------

  defp find_newest_context([]), do: nil
  defp find_newest_context(results), do: List.first(results)

  defp generate_llm_suggestion(ctx, topic, newest) do
    newest_info =
      if newest, do: "Newest related context: \"#{newest.title}\"", else: ""

    prompt = """
    Context "#{ctx.title}" (last updated #{ctx.days_old} days ago) may need updating \
    regarding topic "#{topic}".

    Current L0 abstract: #{ctx.l0 || "(none)"}
    #{newest_info}

    Suggest a specific update in 1-2 sentences. Focus on what likely changed.
    """

    suggestion =
      case Ollama.generate(prompt,
             system:
               "You suggest updates for outdated knowledge base entries. Be specific and concise."
           ) do
        {:ok, text} -> String.trim(text)
        _ -> "Review and update — #{ctx.days_old} days since last modification"
      end

    %{
      context_id: ctx.id,
      title: ctx.title,
      node: ctx.node,
      staleness: Float.round(ctx.staleness, 2),
      days_old: ctx.days_old,
      suggestion: suggestion,
      type: :llm_suggestion
    }
  end
end
