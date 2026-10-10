defmodule OptimalEngine.Memory.EncodingGate do
  @moduledoc """
  Decides whether a candidate memory is worth storing.

  This is the memory-specific quality gate. Signal intake already has S/N
  checks, but a memory write has a different failure mode: storing filler,
  rephrased duplicates, or tiny facts with no future retrieval value. The gate
  combines three cheap local signals:

    * novelty — inverse overlap with live memories in the same scope
    * salience — decision/preference/correction/date/number density
    * prediction_error — update/contradiction language against a similar memory

  The result is explainable and deterministic. It is intentionally heuristic so
  it can run in the hot path without an LLM.

  ## Policy adapter (Bridge rule)

  Per `docs/architecture/MEMORY-CORE-CODEBASE-FIT.md`, modules under
  `OptimalEngine.Memory` must not own governed memory objects or lifecycle
  transitions. This module is a transitional policy adapter: a pure decision
  function with no persistence and no lifecycle authority. It never creates,
  updates, or deletes memories, Claims, or Facts. Its verdict is applied only
  by governed callers — `OptimalEngine.Memory.remember/2` preserves every
  input as a `MemoryCore.SourcePackage` first (rejected inputs are kept
  quarantined, never dropped) and records the gate decision in the derivation
  ledger. Long term this heuristic belongs beside
  `OptimalEngine.MemoryCore.ScoringPolicy`.
  """

  @enforce_keys [:should_encode, :score, :novelty, :salience, :prediction_error, :reason]
  defstruct should_encode: false,
            score: 0.0,
            novelty: 0.0,
            salience: 0.0,
            prediction_error: 0.0,
            reason: "",
            similar_memory_id: nil,
            similar_memory_content: nil

  @type t :: %__MODULE__{
          should_encode: boolean(),
          score: float(),
          novelty: float(),
          salience: float(),
          prediction_error: float(),
          reason: String.t(),
          similar_memory_id: String.t() | nil,
          similar_memory_content: String.t() | nil
        }

  @noise ~w[
    ok okay k kk yes yeah yep yup no nah nope lol lmao haha hahaha nice cool
    thanks thx ty got it gotcha sure same idk gm gn brb ttyl wow
  ]

  @decision_words ~w[
    decided decision approved rejected chose choose committed commit owner owns
    deadline due pricing price budget architecture migration contract renewal
  ]

  @preference_words ~w[
    prefers prefer likes dislikes wants needs hates loves style format default
  ]

  @correction_words ~w[
    correction correct actually updated update now changed no longer instead
    supersedes replaces contradicts wrong
  ]

  @doc """
  Evaluate a candidate memory.

  Options:
    * `:candidates` — live memories in scope, usually from `Memory.list/1`
    * `:threshold` — minimum score to encode, default `0.30`
    * `:salience_floor` — hard floor for salience, default `0.10`
  """
  @spec evaluate(String.t(), keyword()) :: t()
  def evaluate(content, opts \\ []) when is_binary(content) do
    candidates = Keyword.get(opts, :candidates, [])
    threshold = Keyword.get(opts, :threshold, 0.30)
    salience_floor = Keyword.get(opts, :salience_floor, 0.10)

    {similarity, similar} = most_similar(content, candidates)
    novelty = Float.round(1.0 - similarity, 3)
    salience = salience(content)
    prediction_error = prediction_error(content, similarity)

    score =
      (0.30 * novelty + 0.40 * salience + 0.30 * prediction_error)
      |> clamp()
      |> Float.round(3)

    should_encode = salience >= salience_floor and score >= threshold

    %__MODULE__{
      should_encode: should_encode,
      score: score,
      novelty: novelty,
      salience: salience,
      prediction_error: prediction_error,
      reason: reason(should_encode, score, salience, novelty, prediction_error),
      similar_memory_id: similar && similar.id,
      similar_memory_content: similar && similar.content
    }
  end

  @doc false
  def similarity(a, b) when is_binary(a) and is_binary(b) do
    a_words = word_set(a)
    b_words = word_set(b)
    union = MapSet.union(a_words, b_words)

    if MapSet.size(union) == 0 do
      0.0
    else
      MapSet.size(MapSet.intersection(a_words, b_words)) / MapSet.size(union)
    end
  end

  defp most_similar(_content, []), do: {0.0, nil}

  defp most_similar(content, candidates) do
    candidates
    |> Enum.map(fn candidate -> {similarity(content, candidate.content || ""), candidate} end)
    |> Enum.max_by(fn {score, _candidate} -> score end, fn -> {0.0, nil} end)
  end

  defp salience(content) do
    words = words(content)
    normalized = String.downcase(content)

    cond do
      Enum.member?(@noise, String.trim(normalized)) ->
        0.0

      length(words) < 3 ->
        0.08

      true ->
        base = min(byte_size(content) / 240.0, 0.30)
        decision = keyword_score(words, @decision_words, 0.30)
        preference = keyword_score(words, @preference_words, 0.20)
        correction = keyword_score(words, @correction_words, 0.30)
        numbers = if Regex.match?(~r/\b\d{2,}|\$\d|\d+%|\bQ[1-4]\b/i, content), do: 0.18, else: 0.0

        dates =
          if Regex.match?(~r/\b(today|tomorrow|yesterday|week|month|year|20\d\d)\b/i, content),
            do: 0.12,
            else: 0.0

        (base + decision + preference + correction + numbers + dates)
        |> clamp()
        |> Float.round(3)
    end
  end

  defp prediction_error(content, similarity) do
    words = words(content)
    correction = keyword_score(words, @correction_words, 0.55)

    cond do
      correction > 0 and similarity >= 0.20 -> min(1.0, correction + similarity * 0.5)
      correction > 0 -> correction
      similarity >= 0.45 -> 0.20
      true -> 0.0
    end
    |> Float.round(3)
  end

  defp keyword_score(words, keywords, max_score) do
    hits = words |> MapSet.new() |> MapSet.intersection(MapSet.new(keywords)) |> MapSet.size()
    min(max_score, hits * (max_score / 2))
  end

  defp reason(false, _score, salience, _novelty, _prediction_error) when salience < 0.10 do
    "below salience floor"
  end

  defp reason(false, score, _salience, novelty, prediction_error) do
    "below encoding threshold score=#{score} novelty=#{novelty} prediction_error=#{prediction_error}"
  end

  defp reason(true, score, salience, novelty, prediction_error) do
    "encoded score=#{score} salience=#{salience} novelty=#{novelty} prediction_error=#{prediction_error}"
  end

  defp word_set(content), do: content |> words() |> MapSet.new()

  defp words(content) do
    Regex.scan(~r/[a-z0-9][a-z0-9_-]*/i, String.downcase(content))
    |> List.flatten()
    |> Enum.reject(&Enum.member?(@noise, &1))
  end

  defp clamp(n) when n < 0.0, do: 0.0
  defp clamp(n) when n > 1.0, do: 1.0
  defp clamp(n), do: n
end
