defmodule OptimalEngine.Retrieval.IntentAnalyzer do
  @moduledoc """
  Analyzes search queries to understand user intent before retrieval.

  Stateless module — no GenServer. Uses LLM analysis when Ollama is available,
  falls back to regex-based heuristics when it is not.

  Always returns `{:ok, intent_map}`. Never crashes.

  ## Intent map structure

      %{
        expanded_query: String.t(),
        intent_type: :lookup | :comparison | :temporal | :decision | :exploration,
        key_entities: [String.t()],
        temporal_hint: :recent | :historical | :any,
        node_hints: [String.t()]
      }
  """

  alias OptimalEngine.Embed.Ollama
  alias OptimalEngine.MemoryCore.GovernedModel
  require Logger

  @typedoc "Analyzed intent for a search query."
  @type intent_map :: %{
          expanded_query: String.t(),
          intent_type: :lookup | :comparison | :temporal | :decision | :exploration,
          key_entities: [String.t()],
          temporal_hint: :recent | :historical | :any,
          node_hints: [String.t()]
        }

  @node_keywords %{
    "operator" => "operator",
    "personal" => "operator",
    "platform" => "product-customer-portal",
    "portal" => "product-customer-portal",
    "skyscraper" => "product-customer-portal",
    "entity-company" => "entity-company",
    "agency" => "entity-company",
    "ai masters" => "project-platform-launch",
    "course" => "project-platform-launch",
    "ed" => "project-platform-launch",
    "os architect" => "entity-company",
    "youtube" => "entity-company",
    "ahmed" => "entity-company",
    "agency accelerants" => "operation-delivery",
    "aa" => "operation-delivery",
    "founderos" => "operation-delivery",
    "cliniciq" => "operation-delivery",
    "community" => "team",
    "school group" => "team",
    "content" => "learning-research-library",
    "mosaic" => "learning-research-library",
    "podcast" => "learning-research-library",
    "revenue" => "operation-revenue",
    "money" => "operation-revenue",
    "pricing" => "operation-revenue",
    "deal" => "operation-revenue",
    "team" => "team",
    "hiring" => "team",
    "accelerator" => "operation-delivery"
  }

  @stopwords ~w[the a an is are was were what when where who how why did do does can could should would will about for with from this that these those it its and or but not]

  @llm_system_prompt "You are a query analyzer. Output valid JSON only, no explanation."

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Analyzes a search query and returns a structured intent map.

  Uses LLM-based analysis when Ollama is available; falls back to regex
  heuristics otherwise. Always returns `{:ok, intent_map}`.
  """
  @spec analyze(String.t()) :: {:ok, intent_map()}
  def analyze(query) when is_binary(query) do
    intent =
      if Ollama.available?() do
        llm_analyze(query)
      else
        fallback_analyze(query)
      end

    {:ok, intent}
  end

  # ---------------------------------------------------------------------------
  # LLM-based analysis
  # ---------------------------------------------------------------------------

  defp llm_analyze(query) do
    prompt = build_prompt(query)

    generate_model =
      Application.get_env(:optimal_engine, :ollama, [])[:generate_model] || "qwen3:8b"

    governed_generate =
      GovernedModel.call_model(
        "intent_analyzer.generate",
        %{"chars" => String.length(prompt)},
        fn -> Ollama.generate(prompt, system: @llm_system_prompt) end,
        model_id: generate_model,
        model_task_type: "generation"
      )

    case governed_generate do
      {:ok, raw_text} ->
        parse_llm_response(raw_text, query)

      {:error, reason} ->
        Logger.warning("IntentAnalyzer: LLM call failed (#{inspect(reason)}), falling back")
        fallback_analyze(query)
    end
  rescue
    e ->
      Logger.warning("IntentAnalyzer: LLM analysis raised #{inspect(e)}, falling back")
      fallback_analyze(query)
  end

  defp build_prompt(query) do
    """
    Analyze this search query for an operating engine. The system has these domain nodes:
    operator, product-customer-portal, entity-company, project-platform-launch, operation-delivery, team, learning-research-library, inbox, operation-revenue

    Query: "#{query}"

    Respond in JSON only:
    {
      "expanded_query": "original query plus related keywords",
      "intent_type": "lookup|comparison|temporal|decision|exploration",
      "key_entities": ["person or project names found"],
      "temporal_hint": "recent|historical|any",
      "node_hints": ["likely node ids from the list above"]
    }
    """
  end

  defp parse_llm_response(raw_text, original_query) do
    # Extract the first JSON object from the response text — models sometimes
    # prepend/append prose even when asked not to.
    json_text = extract_json(raw_text)

    case Jason.decode(json_text) do
      {:ok, decoded} ->
        build_intent_from_llm(decoded, original_query)

      {:error, reason} ->
        Logger.warning("IntentAnalyzer: JSON decode failed (#{inspect(reason)}), falling back")
        fallback_analyze(original_query)
    end
  end

  defp extract_json(text) do
    case Regex.run(~r/\{[\s\S]*\}/u, text) do
      [json | _] -> json
      nil -> text
    end
  end

  defp build_intent_from_llm(decoded, original_query) do
    %{
      expanded_query: string_or(decoded["expanded_query"], original_query),
      intent_type: parse_intent_type(decoded["intent_type"]),
      key_entities: list_of_strings(decoded["key_entities"]),
      temporal_hint: parse_temporal_hint(decoded["temporal_hint"]),
      node_hints: list_of_strings(decoded["node_hints"])
    }
  end

  defp string_or(value, _default) when is_binary(value) and byte_size(value) > 0, do: value
  defp string_or(_value, default), do: default

  defp list_of_strings(value) when is_list(value) do
    Enum.filter(value, &is_binary/1)
  end

  defp list_of_strings(_), do: []

  defp parse_intent_type("comparison"), do: :comparison
  defp parse_intent_type("temporal"), do: :temporal
  defp parse_intent_type("decision"), do: :decision
  defp parse_intent_type("exploration"), do: :exploration
  defp parse_intent_type(_), do: :lookup

  defp parse_temporal_hint("recent"), do: :recent
  defp parse_temporal_hint("historical"), do: :historical
  defp parse_temporal_hint(_), do: :any

  # ---------------------------------------------------------------------------
  # Regex-based fallback
  # ---------------------------------------------------------------------------

  defp fallback_analyze(query) do
    downcased = String.downcase(query)

    %{
      expanded_query: query,
      intent_type: detect_intent_type(downcased),
      key_entities: extract_entities(query),
      temporal_hint: detect_temporal_hint(downcased),
      node_hints: detect_node_hints(downcased)
    }
  end

  defp detect_intent_type(downcased) do
    cond do
      Regex.match?(~r/\bvs\b|compare|difference/, downcased) ->
        :comparison

      Regex.match?(
        ~r/\bwhen\b|\blast\b|\brecent\b|\btoday\b|\byesterday\b|\bthis week\b/,
        downcased
      ) ->
        :temporal

      Regex.match?(~r/\bdecision\b|\bdecided\b|\bchose\b|\bwhy did\b/, downcased) ->
        :decision

      Regex.match?(~r/\?|\bwhat\b|\bhow\b|\boverview\b|\btell me about\b/, downcased) ->
        :exploration

      true ->
        :lookup
    end
  end

  defp extract_entities(query) do
    query
    |> String.split(~r/\s+/)
    |> Enum.filter(fn word ->
      stripped = String.replace(word, ~r/[^A-Za-z]/, "")

      String.length(stripped) > 0 and
        Regex.match?(~r/^[A-Z]/, stripped) and
        String.downcase(stripped) not in @stopwords
    end)
    |> Enum.map(&String.replace(&1, ~r/[^A-Za-z]/, ""))
    |> Enum.uniq()
  end

  defp detect_temporal_hint(downcased) do
    cond do
      Regex.match?(~r/\brecent\b|\blatest\b|\btoday\b|\bthis week\b/, downcased) ->
        :recent

      Regex.match?(~r/\bold\b|\bfirst\b|\boriginal\b|\bhistory\b|\bback when\b/, downcased) ->
        :historical

      true ->
        :any
    end
  end

  defp detect_node_hints(downcased) do
    @node_keywords
    |> Enum.filter(fn {keyword, _node} -> String.contains?(downcased, keyword) end)
    |> Enum.map(fn {_keyword, node} -> node end)
    |> Enum.uniq()
  end
end
