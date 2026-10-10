defmodule OptimalEngine.Pipeline.IntentExtractor do
  @moduledoc """
  Stage 4 (b) of the ingestion pipeline — per-chunk intent extraction.

  Heuristics-first, Ollama-augmented-if-available. Returns one `%Intent{}`
  per chunk. If Ollama is reachable AND the heuristic confidence is below
  a threshold, we ask the local LLM to resolve among the 10 canonical
  intents; otherwise heuristics alone decide. This keeps the hot path
  fast + deterministic while allowing higher-quality inference on
  ambiguous chunks when capacity is available.

  Intent is NOT the same as `Classification.signal_type`. Type is the speech
  act (direct / inform / commit / decide / express). Intent is the goal —
  what the signal is *for*. They're correlated but not equivalent: a
  `:direct` speech-act can be a `:request_info` OR a `:commit_action`; the
  distinction matters at retrieval time when agents want to filter by goal.

  See `docs/architecture/ARCHITECTURE.md` §Stage 4 for the full enum.
  """

  alias OptimalEngine.Embed.Ollama
  alias OptimalEngine.Pipeline.Decomposer.{Chunk, ChunkTree}
  alias OptimalEngine.Pipeline.IntentExtractor.Intent

  @llm_threshold 0.7

  # Heuristic rule table. Order matters: first match wins. Each rule scores
  # a chunk between 0.0 and 1.0 via its regex match count.
  #
  # Pattern discipline: never put `\b` after a punctuation-ending token —
  # `\b` is a word↔non-word boundary, so `todo:\b` doesn't match "todo: "
  # (`:`→space is non-word→non-word). Punctuation-ending alternatives use
  # no trailing boundary; word-ending alternatives use `\b\b`.
  @rules [
    {:request_info,
     ~r/\?|\b(?:could you|can you|wondering|how do|what is|when will|who is)\b|(?:^|\W)(?:please\s+(?:send|share|let me know)|question:)/i},
    {:propose_decision,
     ~r/\b(?:we should|proposing|deciding between|vote on)\b|(?:^|\W)(?:let's\s+(?:go with|pick|choose|decide)|proposal:|recommend(?:ed)?\s+(?:we|you|they|approach))/i},
    {:commit_action,
     ~r/\b(?:i'll|we'll|i will|we will|going to|committing to|taking on)\b|(?:^|\W)(?:owning\s+(?:this|that)|action item|todo:|task:|by\s+(?:monday|tuesday|wednesday|thursday|friday|end of (?:week|day|month)|eod|eow))/i},
    {:express_concern,
     ~r/\b(?:worri(?:ed|es)|concern(?:ed|s|ing)?|risk(?:y|s|ed)?|blocker|blocked|issue|problem|alarming|dangerous|unsafe|unsustainable|warning|caution|red flag)\b/i},
    {:specify,
     ~r/\b(?:must|shall|required|requires?|mandator(?:y|ily)|acceptance criteria)\b|(?:^|\W)(?:requirement:|spec(?:ification)?:|contract:|constraint:|invariant:)/i},
    {:measure,
     ~r/(?:\d+(?:\.\d+)?(?:%|\s*(?:seconds?|minutes?|hours?|days?|weeks?|months?|ms|rpm|rps|req\/s|tps|tokens?|dollars?|\$|€|£|users?|customers?|conversions?|clicks?|hits?|errors?|failures?))|latency|throughput|count|total|average|median|p95|p99)/i},
    {:reflect,
     ~r/\b(?:looking back|in retrospect|retrospective|reviewing|post.?mortem|lessons learned|reflecting|examining past|hindsight)\b/i},
    {:narrate,
     ~r/(?:^|\W)(?:first,|next,|then,|after(?:wards)?,|finally,|timeline:)|\b(?:subsequently|meeting notes)\b|on\s+(?:\w+day|\d{4}-\d{2}-\d{2})/i},
    {:reference,
     ~r/^\s*(?:see|cf\.|cite)\b|(?:^|\W)(?:ref(?:erence)?s?:|related:|link:|source:|attached:)|\b(?:per\s+(?:the\s+)?(?:doc|spec|note|email|thread|ticket|pr|issue))\b/i}
    # :record_fact is the default, no rule needed
  ]

  @doc """
  Extract the intent of a single chunk.

  Returns `{:ok, %Intent{}}`. The returned intent always has a valid
  `:intent` atom from the canonical enum and a confidence in `0.0..1.0`.
  """
  @spec extract(Chunk.t(), keyword()) :: {:ok, Intent.t()}
  def extract(%Chunk{} = chunk, opts \\ []) do
    tenant_id = chunk.tenant_id
    text = chunk.text || ""

    {heuristic_intent, heuristic_conf, evidence} = apply_heuristics(text)

    {final_intent, final_conf} =
      if heuristic_conf < @llm_threshold and Keyword.get(opts, :ollama_augmentation, true) do
        ollama_refine(text, heuristic_intent, heuristic_conf)
      else
        {heuristic_intent, heuristic_conf}
      end

    {:ok,
     Intent.new(
       chunk_id: chunk.id,
       tenant_id: tenant_id,
       intent: final_intent,
       confidence: Float.round(final_conf, 3),
       evidence: evidence
     )}
  end

  @doc "Extract intents for every chunk in a ChunkTree."
  @spec extract_tree(ChunkTree.t(), keyword()) :: {:ok, [Intent.t()]}
  def extract_tree(%ChunkTree{chunks: chunks}, opts \\ []) do
    intents = Enum.map(chunks, fn c -> elem(extract(c, opts), 1) end)
    {:ok, intents}
  end

  # ─── heuristics ──────────────────────────────────────────────────────────

  defp apply_heuristics("") do
    {:record_fact, 0.3, nil}
  end

  defp apply_heuristics(text) do
    case Enum.find(@rules, fn {_intent, re} -> Regex.match?(re, text) end) do
      nil ->
        {:record_fact, 0.5, nil}

      {intent, re} ->
        match = Regex.run(re, text, return: :index)
        evidence = evidence_window(text, match)
        confidence = score_from_match_density(text, re)
        {intent, confidence, evidence}
    end
  end

  # Confidence scales with how many times the rule matches, capped at 0.95
  # for heuristics alone (we want Ollama to be able to push above this
  # when it agrees with confidence).
  defp score_from_match_density(text, re) do
    match_count = Regex.scan(re, text) |> length()
    base = 0.65
    additional = min(match_count - 1, 6) * 0.05
    Float.round(min(base + additional, 0.95), 2)
  end

  # Pull a ~80-char window around the first match so humans can eyeball
  # why the classifier landed where it did.
  defp evidence_window(_text, nil), do: nil
  defp evidence_window(_text, []), do: nil

  defp evidence_window(text, [{offset, length} | _]) do
    start = max(offset - 20, 0)
    finish = min(offset + length + 60, byte_size(text))
    slice = :binary.part(text, start, finish - start)
    String.trim(slice)
  end

  # ─── Ollama refinement ───────────────────────────────────────────────────

  # When heuristic confidence is below threshold, ask Ollama to pick one of
  # the 10 canonical intents. If Ollama isn't reachable or returns garbage,
  # we return the heuristic's guess unchanged (never crash, never block).
  defp ollama_refine(text, heuristic_intent, heuristic_conf) do
    if Ollama.available?() do
      case Ollama.generate(llm_prompt(text), system: llm_system_prompt(), timeout_ms: 5_000) do
        {:ok, response} ->
          case parse_llm_response(response) do
            {:ok, llm_intent, llm_conf} ->
              {llm_intent, Float.round(max(llm_conf, heuristic_conf), 3)}

            _ ->
              {heuristic_intent, heuristic_conf}
          end

        _ ->
          {heuristic_intent, heuristic_conf}
      end
    else
      {heuristic_intent, heuristic_conf}
    end
  end

  defp llm_system_prompt do
    """
    You classify text chunks by intent — what the author is trying to
    accomplish. Choose exactly ONE of these ten values:

    request_info | propose_decision | record_fact | express_concern |
    commit_action | reference | narrate | reflect | specify | measure

    Respond with a single JSON object, nothing else:
    {"intent": "<value>", "confidence": 0.0..1.0}
    """
  end

  defp llm_prompt(text) do
    # Cap to ~1500 chars to stay in a short round-trip.
    preview = text |> String.slice(0, 1500)
    "Classify this chunk:\n\n" <> preview
  end

  defp parse_llm_response(raw) when is_binary(raw) do
    # Find the first { ... } JSON blob in the response (LLMs sometimes wrap
    # in markdown or add prose).
    case Regex.run(~r/\{.*?\}/s, raw) do
      [json] ->
        case Jason.decode(json) do
          {:ok, %{"intent" => intent_str, "confidence" => c}}
          when is_binary(intent_str) and is_number(c) ->
            case safe_atom(intent_str) do
              nil -> :error
              intent -> {:ok, intent, 1.0 * c}
            end

          _ ->
            :error
        end

      _ ->
        :error
    end
  end

  defp parse_llm_response(_), do: :error

  defp safe_atom(str) do
    try do
      atom = String.to_existing_atom(str)
      if Intent.valid?(atom), do: atom, else: nil
    rescue
      ArgumentError -> nil
    end
  end
end
