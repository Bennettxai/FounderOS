defmodule OptimalEngine.Insight.Verify do
  @moduledoc """
  Cold-read L0 fidelity test.

  Samples contexts from the store, takes only the title and L0 abstract,
  then evaluates how well the L0 represents the full content.

  With Ollama: LLM predicts what the content should contain based on title+L0,
  then scores the prediction against actual content.
  Without Ollama: Uses Jaccard similarity between L0 keywords and content keywords.

  Stateless module — no GenServer.
  """

  require Logger
  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store

  @default_sample_size 10

  @stopwords ~w(this that with from they have been were will would could should about after before under over between through during)

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Samples contexts and scores their L0 abstract fidelity.

  ## Options
  - `:sample`       — number of contexts to sample (default: #{@default_sample_size})
  - `:node`         — restrict sampling to a specific node (default: all nodes)
  - `:workspace_id` — workspace to sample from (default: "default")

  ## Returns
  `{:ok, map()}` where the map contains:
  - `:scores`       — list of per-context score maps
  - `:aggregate`    — mean fidelity score (0.0–1.0)
  - `:sample_size`  — number of contexts evaluated
  - `:message`      — human-readable fidelity summary
  """
  @spec verify(keyword()) :: {:ok, map()}
  def verify(opts \\ []) do
    sample_size = Keyword.get(opts, :sample, @default_sample_size)
    node_filter = Keyword.get(opts, :node)
    ws = Keyword.get(opts, :workspace_id, "default")

    contexts = sample_contexts(sample_size, node_filter, ws)

    if contexts == [] do
      {:ok, %{scores: [], aggregate: 0.0, sample_size: 0, message: "No contexts to verify"}}
    else
      scores = Enum.map(contexts, &score_context/1)

      valid_scores = Enum.reject(scores, fn s -> s.score == nil end)

      aggregate =
        if valid_scores == [] do
          0.0
        else
          Enum.sum(Enum.map(valid_scores, & &1.score)) / length(valid_scores)
        end

      {:ok,
       %{
         scores: scores,
         aggregate: Float.round(aggregate, 3),
         sample_size: length(contexts),
         message: fidelity_message(aggregate)
       }}
    end
  rescue
    err ->
      Logger.warning("[VerifyEngine] verify/1 failed: #{inspect(err)}")

      {:ok,
       %{
         scores: [],
         aggregate: 0.0,
         sample_size: 0,
         message: "Verification failed: #{inspect(err)}"
       }}
  end

  # ---------------------------------------------------------------------------
  # Private: Sampling
  # ---------------------------------------------------------------------------

  defp sample_contexts(n, nil, ws) do
    sql = """
    SELECT id, title, l0_abstract, content, node
    FROM contexts
    WHERE l0_abstract != '' AND content != '' AND workspace_id = ?2
    ORDER BY RANDOM()
    LIMIT ?1
    """

    case Store.raw_query(sql, [n, ws]) do
      {:ok, rows} -> Enum.map(rows, &row_to_ctx/1)
      _ -> []
    end
  end

  defp sample_contexts(n, node, ws) do
    sql = """
    SELECT id, title, l0_abstract, content, node
    FROM contexts
    WHERE l0_abstract != '' AND content != '' AND node = ?2 AND workspace_id = ?3
    ORDER BY RANDOM()
    LIMIT ?1
    """

    case Store.raw_query(sql, [n, node, ws]) do
      {:ok, rows} -> Enum.map(rows, &row_to_ctx/1)
      _ -> []
    end
  end

  defp row_to_ctx([id, title, l0, content, node]) do
    %{id: id, title: title, l0: l0, content: content, node: node}
  end

  # ---------------------------------------------------------------------------
  # Private: Scoring
  # ---------------------------------------------------------------------------

  defp score_context(ctx) do
    score =
      if Ollama.available?() do
        llm_score(ctx)
      else
        jaccard_score(ctx)
      end

    %{
      id: ctx.id,
      title: ctx.title,
      node: ctx.node,
      score: score,
      grade: score_grade(score)
    }
  end

  defp llm_score(ctx) do
    prompt = """
    Based on this title and abstract, predict what the full content covers.

    Title: #{ctx.title}
    Abstract: #{ctx.l0}

    Now here is the actual content (first 500 chars):
    #{String.slice(ctx.content, 0, 500)}

    Score how well the abstract represents the content on a scale of 0.0 to 1.0.
    Reply with ONLY a number like 0.85
    """

    case Ollama.generate(prompt,
           system: "You evaluate abstract quality. Reply with only a decimal number 0.0-1.0."
         ) do
      {:ok, response} ->
        case Float.parse(String.trim(response)) do
          {score, _} when score >= 0.0 and score <= 1.0 -> score
          _ -> jaccard_score(ctx)
        end

      _ ->
        jaccard_score(ctx)
    end
  rescue
    _ -> jaccard_score(ctx)
  end

  defp jaccard_score(ctx) do
    l0_words = extract_keywords(ctx.l0)
    content_words = extract_keywords(ctx.content)

    if MapSet.size(l0_words) == 0 or MapSet.size(content_words) == 0 do
      0.0
    else
      intersection = MapSet.intersection(l0_words, content_words) |> MapSet.size()
      union = MapSet.union(l0_words, content_words) |> MapSet.size()

      if union == 0, do: 0.0, else: intersection / union
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Keyword extraction
  # ---------------------------------------------------------------------------

  defp extract_keywords(text) when is_binary(text) do
    text
    |> String.downcase()
    |> String.replace(~r/[^a-z0-9\s]/, " ")
    |> String.split(~r/\s+/)
    |> Enum.reject(fn w -> String.length(w) < 4 end)
    |> Enum.reject(&stopword?/1)
    |> MapSet.new()
  end

  defp extract_keywords(_), do: MapSet.new()

  defp stopword?(word), do: word in @stopwords

  # ---------------------------------------------------------------------------
  # Private: Grading helpers
  # ---------------------------------------------------------------------------

  defp score_grade(nil), do: "N/A"
  defp score_grade(s) when s >= 0.8, do: "A"
  defp score_grade(s) when s >= 0.6, do: "B"
  defp score_grade(s) when s >= 0.4, do: "C"
  defp score_grade(s) when s >= 0.2, do: "D"
  defp score_grade(_), do: "F"

  defp fidelity_message(avg) when avg >= 0.8,
    do: "Excellent L0 fidelity — abstracts accurately represent content"

  defp fidelity_message(avg) when avg >= 0.6,
    do: "Good L0 fidelity — most abstracts are representative"

  defp fidelity_message(avg) when avg >= 0.4,
    do: "Fair L0 fidelity — some abstracts need improvement"

  defp fidelity_message(_),
    do: "Poor L0 fidelity — many abstracts don't represent their content well"
end
