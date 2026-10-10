defmodule OptimalEngine.Memory.Search do
  @moduledoc """
  Relevance-based retrieval of memory entries.

  Implements keyword scoring with recency and importance decay.
  Delegates index lookups to `OptimalEngine.Memory.Index`.
  """

  alias OptimalEngine.Memory.Index

  @chars_per_token 4

  @category_importance %{
    "decision" => 1.0,
    "preference" => 0.9,
    "architecture" => 0.95,
    "bug" => 0.8,
    "insight" => 0.85,
    "contact" => 0.7,
    "workflow" => 0.75,
    "general" => 0.5,
    "note" => 0.4
  }

  @doc """
  Retrieve memories relevant to a query within a token budget.

  Scores entries by keyword overlap, recency, and category importance.
  Returns formatted string or "" if nothing matches.
  """
  @spec recall_relevant(String.t(), pos_integer()) :: String.t()
  def recall_relevant(message, max_tokens \\ 2000) do
    keywords = Index.extract_keywords(message)

    if keywords == [] do
      get_recent_entries(max_tokens)
    else
      entry_ids = Index.query_keywords(keywords)

      if map_size(entry_ids) == 0 do
        get_recent_entries(max_tokens)
      else
        scored =
          entry_ids
          |> Enum.map(fn {entry_id, keyword_hits} ->
            case Index.get_entry(entry_id) do
              nil -> nil
              entry -> {score(entry, keyword_hits, length(keywords)), entry}
            end
          end)
          |> Enum.reject(&is_nil/1)
          |> Enum.sort_by(fn {s, _} -> s end, :desc)

        select_within_budget(scored, max_tokens * @chars_per_token)
      end
    end
  end

  @doc """
  Search memories by keyword or category.

  ## Options
    - `:category` — filter by category string
    - `:limit` — max results (default 10)
    - `:sort` — `:relevance` (default), `:recency`, `:importance`
  """
  @spec search(String.t(), keyword()) :: [map()]
  def search(query, opts \\ []) do
    category_filter = Keyword.get(opts, :category)
    limit = Keyword.get(opts, :limit, 10)
    sort = Keyword.get(opts, :sort, :relevance)

    keywords = Index.extract_keywords(query)

    all_entries =
      Index.all_entries()
      |> maybe_filter_category(category_filter)

    scored =
      if keywords != [] do
        all_entries
        |> Enum.map(fn entry ->
          entry_keywords = Index.extract_keywords(entry[:content] || "")
          entry_keyword_set = MapSet.new(entry_keywords)
          overlap = Enum.count(keywords, &MapSet.member?(entry_keyword_set, &1))
          s = score(entry, overlap, length(keywords))
          {s, entry}
        end)
        |> Enum.filter(fn {s, _} -> s > 0.05 end)
      else
        Enum.map(all_entries, fn entry -> {0.5, entry} end)
      end

    sorted =
      case sort do
        :recency ->
          Enum.sort_by(scored, fn {_, entry} -> entry[:timestamp] || "" end, :desc)

        :importance ->
          Enum.sort_by(
            scored,
            fn {_, entry} ->
              Map.get(@category_importance, entry[:category] || "general", 0.5)
            end,
            :desc
          )

        _ ->
          Enum.sort_by(scored, fn {s, _} -> s end, :desc)
      end

    sorted
    |> Enum.take(limit)
    |> Enum.map(fn {s, entry} -> Map.put(entry, :relevance_score, Float.round(s, 3)) end)
  end

  # ── Private ──────────────────────────────────────────────────────────

  defp score(entry, keyword_hits, total_keywords) do
    overlap = keyword_hits / max(total_keywords, 1)
    recency = recency_score(entry[:timestamp])
    importance = Map.get(@category_importance, entry[:category] || "general", 0.5)
    overlap * 0.5 + recency * 0.3 + importance * 0.2
  end

  defp recency_score(nil), do: 0.3

  defp recency_score(ts) when is_binary(ts) do
    case DateTime.from_iso8601(ts) do
      {:ok, dt, _} ->
        age_hours = DateTime.diff(DateTime.utc_now(), dt, :second) / 3600.0
        :math.exp(-0.693 * age_hours / 48.0)

      _ ->
        0.3
    end
  end

  defp recency_score(_), do: 0.3

  defp select_within_budget(scored_entries, max_chars) do
    {selected, _} =
      Enum.reduce_while(scored_entries, {[], max_chars}, fn {_score, entry}, {acc, budget} ->
        content = entry[:content] || ""
        header = "## [#{entry[:category] || "general"}] #{entry[:timestamp] || "unknown"}"
        full_text = "#{header}\n#{content}\n"
        size = byte_size(full_text)

        if size <= budget do
          {:cont, {[full_text | acc], budget - size}}
        else
          {:halt, {acc, budget}}
        end
      end)

    selected |> Enum.reverse() |> Enum.join("\n")
  end

  defp get_recent_entries(max_tokens) do
    max_chars = max_tokens * @chars_per_token

    scored =
      Index.all_entries()
      |> Enum.sort_by(fn entry -> entry[:timestamp] || "" end, :desc)
      |> Enum.take(20)
      |> Enum.map(fn entry -> {1.0, entry} end)

    select_within_budget(scored, max_chars)
  end

  defp maybe_filter_category(entries, nil), do: entries

  defp maybe_filter_category(entries, category) do
    Enum.filter(entries, fn entry -> entry[:category] == category end)
  end
end
