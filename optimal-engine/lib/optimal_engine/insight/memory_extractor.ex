defmodule OptimalEngine.Insight.MemoryExtractor do
  @moduledoc """
  Extracts structured memories from session transcripts using a 6-category system.

  Categories:
  - `:fact`         — Concrete statements about the world
  - `:preference`   — How someone prefers things to work
  - `:decision`     — Choices that were made
  - `:relationship` — How people/entities relate to each other
  - `:skill`        — Learned capabilities or processes
  - `:context`      — Situational/temporal information

  Extraction strategy:
  1. If Ollama is available, use LLM-based extraction (higher accuracy).
  2. If Ollama is unavailable, fall back to regex-based extraction.
  3. Filter results below 0.7 confidence threshold.
  4. Cap at 20 memories per extraction call.

  This module is stateless — no GenServer, no process state.
  All failures are caught; the function always returns `{:ok, []}` in the worst case.
  """

  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Routing
  require Logger

  @confidence_threshold 0.7
  @max_memories 20
  @max_prompt_chars 3_000

  @valid_categories ~w(fact preference decision relationship skill context)

  @stopwords ~w(
    The A An And But Or For Nor So Yet Both Either Neither Not Only Such
    As At By For In Of On To Up Via With
  )

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Extracts structured memories from a session transcript string.

  Returns `{:ok, [memory_map()]}` where each map has the shape:

      %{
        category:   :fact | :preference | :decision | :relationship | :skill | :context,
        content:    String.t(),
        confidence: float(),
        entities:   [String.t()]
      }

  Always succeeds — failures fall back through regex, then to empty list.
  """
  @spec extract(String.t()) :: {:ok, [map()]}
  def extract(text) when is_binary(text) do
    memories =
      if Ollama.available?() do
        llm_extract(text)
      else
        regex_extract(text)
      end

    filtered =
      memories
      |> Enum.filter(&(&1.confidence >= @confidence_threshold))
      |> Enum.take(@max_memories)

    {:ok, filtered}
  rescue
    err ->
      Logger.warning("MemoryExtractor.extract/1 failed: #{inspect(err)}")
      {:ok, []}
  end

  @doc """
  Extracts entity names from a piece of text.

  Loads known people from Topology, scans for name matches, and also
  extracts capitalized proper nouns not in the stopword list.

  Returns a deduplicated list of entity strings.
  """
  @spec extract_entities(String.t()) :: [String.t()]
  def extract_entities(text) when is_binary(text) do
    known = known_people()

    topology_matches =
      Enum.filter(known, fn name ->
        String.contains?(String.downcase(text), String.downcase(name))
      end)

    proper_nouns = extract_proper_nouns(text)

    (topology_matches ++ proper_nouns)
    |> Enum.uniq()
  rescue
    err ->
      Logger.warning("MemoryExtractor.extract_entities/1 failed: #{inspect(err)}")
      []
  end

  # ---------------------------------------------------------------------------
  # LLM-based extraction
  # ---------------------------------------------------------------------------

  defp llm_extract(text) do
    truncated = truncate(text, @max_prompt_chars)

    prompt = """
    Extract structured memories from this conversation. For each memory, classify into one of 6 categories:

    - fact: Concrete statements about the world ("Ed wants $2K per seat", "Budget is $50K")
    - preference: How someone prefers things ("Alice prefers specs for devs", "Use Slack not email")
    - decision: Choices that were made ("Decided to use Firecracker", "Agreed on Q2 launch")
    - relationship: How people/entities relate ("Ed is Platform Launch partner", "Dan reports to Alice")
    - skill: Learned capabilities or processes ("Use mix optimal.search for context", "Deploy via GitHub Actions")
    - context: Situational/temporal info ("Platform Launch launching Q2", "Team is 5 people currently")

    Conversation:
    #{truncated}

    Respond with a JSON array only:
    [{"category": "fact", "content": "...", "confidence": 0.9, "entities": ["Ed"]}]
    """

    system =
      "You are a memory extractor for a cognitive operating system. " <>
        "Extract only clear, factual memories. Set confidence based on how explicit the " <>
        "statement was (0.0-1.0). Output valid JSON array only."

    case Ollama.generate(prompt, system: system) do
      {:ok, response} ->
        parse_memories(response)

      {:error, reason} ->
        Logger.info("MemoryExtractor LLM unavailable (#{inspect(reason)}), using regex fallback")
        regex_extract(text)
    end
  rescue
    err ->
      Logger.warning("MemoryExtractor llm_extract failed: #{inspect(err)}, falling back to regex")
      regex_extract(text)
  end

  # ---------------------------------------------------------------------------
  # Regex-based fallback extraction
  # ---------------------------------------------------------------------------

  defp regex_extract(text) do
    sentences = extract_sentences(text)

    decisions = extract_decisions(sentences)
    facts = extract_facts(sentences)
    relationships = extract_relationships(sentences)
    preferences = extract_preferences(sentences)

    (decisions ++ facts ++ relationships ++ preferences)
    |> deduplicate_by_content()
  end

  defp extract_decisions(sentences) do
    patterns = ~w(decided agreed confirmed chose we'll\ go\ with the\ plan\ is)

    Enum.flat_map(sentences, fn sentence ->
      lower = String.downcase(sentence)

      if Enum.any?(patterns, &String.contains?(lower, &1)) do
        [
          %{
            category: :decision,
            content: String.trim(sentence),
            confidence: 0.8,
            entities: extract_entities(sentence)
          }
        ]
      else
        []
      end
    end)
  end

  defp extract_facts(sentences) do
    # Patterns for "X is Y", "X has Y", "X costs $Y", "X wants Y"
    pattern =
      ~r/\b[A-Z][a-zA-Z]+\s+(?:is|has|costs?|wants?|needs?|owns?|runs?|leads?|manages?)\s+\S/

    Enum.flat_map(sentences, fn sentence ->
      if Regex.match?(pattern, sentence) do
        [
          %{
            category: :fact,
            content: String.trim(sentence),
            confidence: 0.6,
            entities: extract_entities(sentence)
          }
        ]
      else
        []
      end
    end)
  end

  defp extract_relationships(sentences) do
    people = known_people()

    Enum.flat_map(sentences, fn sentence ->
      lower = String.downcase(sentence)

      matched_people =
        Enum.filter(people, fn name ->
          String.contains?(lower, String.downcase(name))
        end)

      if length(matched_people) >= 2 do
        [
          %{
            category: :relationship,
            content: String.trim(sentence),
            confidence: 0.7,
            entities: matched_people
          }
        ]
      else
        []
      end
    end)
  end

  defp extract_preferences(sentences) do
    patterns = ~w(prefer always\ use never\ use better\ to should\ always)

    Enum.flat_map(sentences, fn sentence ->
      lower = String.downcase(sentence)

      if Enum.any?(patterns, &String.contains?(lower, &1)) do
        [
          %{
            category: :preference,
            content: String.trim(sentence),
            confidence: 0.7,
            entities: extract_entities(sentence)
          }
        ]
      else
        []
      end
    end)
  end

  # ---------------------------------------------------------------------------
  # Private helpers
  # ---------------------------------------------------------------------------

  defp parse_memories(json_string) do
    # Strip markdown code fences if present
    cleaned =
      json_string
      |> String.trim()
      |> strip_code_fence()

    case Jason.decode(cleaned) do
      {:ok, list} when is_list(list) ->
        Enum.flat_map(list, &coerce_memory/1)

      {:ok, other} ->
        Logger.warning("MemoryExtractor: expected JSON array, got: #{inspect(other)}")
        []

      {:error, reason} ->
        Logger.warning(
          "MemoryExtractor: JSON decode failed: #{inspect(reason)}, raw: #{inspect(String.slice(json_string, 0, 200))}"
        )

        []
    end
  rescue
    err ->
      Logger.warning("MemoryExtractor parse_memories failed: #{inspect(err)}")
      []
  end

  defp strip_code_fence(text) do
    text
    |> String.replace(~r/^```(?:json)?\n?/, "")
    |> String.replace(~r/\n?```$/, "")
    |> String.trim()
  end

  defp coerce_memory(%{"category" => cat, "content" => content} = raw)
       when is_binary(cat) and is_binary(content) do
    confidence =
      case Map.get(raw, "confidence") do
        v when is_float(v) -> v
        v when is_integer(v) -> v / 1.0
        _ -> 0.5
      end

    entities =
      case Map.get(raw, "entities") do
        list when is_list(list) -> Enum.filter(list, &is_binary/1)
        _ -> []
      end

    [
      %{
        category: category_to_atom(cat),
        content: String.trim(content),
        confidence: confidence,
        entities: entities
      }
    ]
  end

  defp coerce_memory(other) do
    Logger.debug("MemoryExtractor: skipping malformed memory entry: #{inspect(other)}")
    []
  end

  defp category_to_atom(str) when is_binary(str) do
    normalized = String.downcase(String.trim(str))

    if normalized in @valid_categories do
      String.to_atom(normalized)
    else
      :context
    end
  end

  defp extract_sentences(text) do
    text
    |> String.split(~r/(?<=[.!?])(?:\s+|\n)/)
    |> Enum.map(&String.trim/1)
    |> Enum.reject(&(String.length(&1) < 10))
  end

  defp truncate(string, max) when byte_size(string) <= max, do: string

  defp truncate(string, max) do
    String.slice(string, 0, max) <> "..."
  end

  defp known_people do
    case Routing.load() do
      {:ok, topology} ->
        topology.endpoints
        |> Map.values()
        |> Enum.map(& &1.name)
        |> Enum.reject(&(&1 == "" or is_nil(&1)))

      {:error, _} ->
        []
    end
  rescue
    _ -> []
  end

  defp extract_proper_nouns(text) do
    # Match sequences of capitalised words (1-3 words) that aren't stopwords
    ~r/\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2})\b/
    |> Regex.scan(text, capture: :all_but_first)
    |> List.flatten()
    |> Enum.reject(fn noun ->
      words = String.split(noun, " ")
      Enum.all?(words, &(&1 in @stopwords))
    end)
    |> Enum.uniq()
  end

  defp deduplicate_by_content(memories) do
    memories
    |> Enum.uniq_by(fn m ->
      m.content
      |> String.downcase()
      |> String.trim()
    end)
  end
end
