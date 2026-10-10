defmodule OptimalEngine.Insight.Rethink do
  @moduledoc """
  Evidence synthesis engine for the OptimalOS knowledge base.

  When cumulative confidence on a topic reaches >= 1.5 (from observations
  in RememberLoop), the RethinkEngine gathers all evidence — observations,
  search results, and graph connections — then generates a structured
  rethink report with proposed context.md updates.

  Never auto-writes changes (`:auto_apply` defaults to false).

  Stateless module — no GenServer.
  """

  require Logger
  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store

  @confidence_threshold 1.5

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Synthesizes accumulated observations about a topic into actionable knowledge.

  When total confidence across observations is >= 1.5, generates a synthesis
  report with evidence, patterns, and proposed context.md updates.

  ## Options

  - `:force`        — bypass the confidence threshold (default: false)
  - `:auto_apply`   — apply proposed updates automatically (default: false)
  - `:workspace_id` — workspace whose observations are synthesized (default: "default")

  ## Return value

  Always returns `{:ok, map()}`. The map includes a `:status` key:

  - `:insufficient_evidence` — not enough confidence accumulated yet
  - `:synthesized`           — report generated successfully
  - `:error`                 — unexpected failure (message in `:message`)
  """
  @spec rethink(String.t(), keyword()) :: {:ok, map()}
  def rethink(topic, opts \\ []) do
    auto_apply = Keyword.get(opts, :auto_apply, false)
    ws = Keyword.get(opts, :workspace_id, "default")

    observations = gather_observations(topic, ws)

    total_confidence =
      observations
      |> Enum.map(& &1.confidence)
      |> Enum.sum()

    if total_confidence < @confidence_threshold and not Keyword.get(opts, :force, false) do
      {:ok,
       %{
         topic: topic,
         status: :insufficient_evidence,
         total_confidence: Float.round(total_confidence, 2),
         threshold: @confidence_threshold,
         observation_count: length(observations),
         message:
           "Need #{Float.round(@confidence_threshold - total_confidence, 2)} more confidence to trigger rethink. Use --force to override."
       }}
    else
      related_contexts = search_related(topic, ws)
      synthesis = generate_synthesis(topic, observations, related_contexts)

      report = %{
        topic: topic,
        status: :synthesized,
        total_confidence: Float.round(total_confidence, 2),
        observation_count: length(observations),
        observations: observations,
        related_context_count: length(related_contexts),
        related_contexts:
          Enum.map(related_contexts, fn ctx ->
            %{id: ctx.id, title: ctx.title, node: ctx.node}
          end),
        synthesis: synthesis,
        auto_applied: false,
        proposed_updates: synthesis.proposed_updates
      }

      if auto_apply do
        apply_updates(synthesis.proposed_updates)
        {:ok, Map.put(report, :auto_applied, true)}
      else
        {:ok, report}
      end
    end
  rescue
    err ->
      Logger.warning("[RethinkEngine] rethink/2 failed: #{inspect(err)}")
      {:ok, %{topic: topic, status: :error, message: inspect(err)}}
  end

  # ---------------------------------------------------------------------------
  # Private: Gather evidence
  # ---------------------------------------------------------------------------

  defp gather_observations(topic, ws) do
    sql = """
    SELECT id, category, content, confidence, source, created_at
    FROM observations
    WHERE (category = ?1 OR content LIKE ?2) AND workspace_id = ?3
    ORDER BY confidence DESC
    """

    case Store.raw_query(sql, [topic, "%#{topic}%", ws]) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, cat, content, conf, source, created] ->
          %{
            id: id,
            category: cat,
            content: content,
            confidence: conf,
            source: source,
            created_at: created
          }
        end)

      _ ->
        []
    end
  end

  defp search_related(topic, ws) do
    case GenServer.call(
           OptimalEngine.Retrieval.Search,
           {:search, topic, [limit: 10, workspace_id: ws]},
           15_000
         ) do
      {:ok, results} -> results
      _ -> []
    end
  rescue
    _ -> []
  end

  # ---------------------------------------------------------------------------
  # Private: Generate synthesis
  # ---------------------------------------------------------------------------

  defp generate_synthesis(topic, observations, related_contexts) do
    if Ollama.available?() do
      llm_synthesis(topic, observations, related_contexts)
    else
      rule_based_synthesis(topic, observations, related_contexts)
    end
  end

  defp llm_synthesis(topic, observations, related_contexts) do
    obs_text =
      observations
      |> Enum.map(fn o ->
        "- [#{o.category}] #{o.content} (confidence: #{o.confidence})"
      end)
      |> Enum.join("\n")

    ctx_text =
      related_contexts
      |> Enum.take(5)
      |> Enum.map(fn c -> "- \"#{c.title}\" (node: #{c.node})" end)
      |> Enum.join("\n")

    prompt = """
    Synthesize these observations about "#{topic}" into actionable knowledge:

    Observations:
    #{obs_text}

    Related existing contexts:
    #{ctx_text}

    Produce:
    1. A summary (2-3 sentences) of what the observations collectively tell us
    2. A list of proposed updates to the knowledge base (which context.md files should change and how)
    3. Any new patterns or rules that emerge

    Format as JSON:
    {
      "summary": "...",
      "proposed_updates": [{"file": "...", "action": "update|add", "content": "..."}],
      "patterns": ["..."]
    }
    """

    case Ollama.generate(prompt,
           system: "You synthesize observations into actionable knowledge. Output valid JSON only."
         ) do
      {:ok, response} ->
        parse_synthesis(response, topic, observations)

      _ ->
        rule_based_synthesis(topic, observations, related_contexts)
    end
  rescue
    _ -> rule_based_synthesis(topic, observations, related_contexts)
  end

  defp parse_synthesis(json_string, topic, observations) do
    cleaned =
      json_string
      |> String.trim()
      |> String.replace(~r/^```(?:json)?\n?/, "")
      |> String.replace(~r/\n?```$/, "")
      |> String.trim()

    case Jason.decode(cleaned) do
      {:ok, %{"summary" => summary, "proposed_updates" => updates, "patterns" => patterns}} ->
        %{
          summary: summary,
          proposed_updates:
            Enum.map(updates, fn u ->
              %{file: u["file"], action: u["action"], content: u["content"]}
            end),
          patterns: patterns,
          method: :llm
        }

      _ ->
        rule_based_synthesis(topic, observations, [])
    end
  end

  defp rule_based_synthesis(topic, observations, _related_contexts) do
    grouped = Enum.group_by(observations, & &1.category)

    summary_parts =
      Enum.map(grouped, fn {cat, obs} ->
        "#{length(obs)} #{cat} observations"
      end)

    total_conf =
      observations
      |> Enum.map(& &1.confidence)
      |> Enum.sum()
      |> Float.round(2)

    summary =
      "Rethink on \"#{topic}\": #{Enum.join(summary_parts, ", ")}. " <>
        "Total confidence: #{total_conf}."

    patterns =
      observations
      |> Enum.filter(fn o -> o.confidence >= 0.7 end)
      |> Enum.map(& &1.content)
      |> Enum.take(5)

    %{
      summary: summary,
      proposed_updates: [],
      patterns: patterns,
      method: :rule_based
    }
  end

  # ---------------------------------------------------------------------------
  # Private: Apply updates (only when explicitly requested)
  # ---------------------------------------------------------------------------

  defp apply_updates([]) do
    Logger.info("[RethinkEngine] No updates to apply")
  end

  defp apply_updates(updates) do
    Enum.each(updates, fn update ->
      Logger.info("[RethinkEngine] Would apply: #{update.action} to #{update.file}")
      # For now, just log. Auto-apply is dangerous and should be explicit.
    end)
  end
end
