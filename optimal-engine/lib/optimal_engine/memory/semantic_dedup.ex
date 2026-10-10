defmodule OptimalEngine.Memory.SemanticDedup do
  @moduledoc """
  Advisory near-duplicate analysis for candidate memory content.

  `Memory.Versioned.create/1` already handles exact content hashes. This module
  catches the cases hashes cannot: paraphrases, expansions, and updates. It
  returns one of three advisory actions:

    * `:add` — no similar live memory carries the content
    * `:skip` — an existing memory already carries the fact
    * `:update` — the candidate appears to supersede the closest existing memory

  ## Policy adapter (Bridge rule)

  Per `docs/architecture/MEMORY-CORE-CODEBASE-FIT.md`, modules under
  `OptimalEngine.Memory` must not own governed memory objects or lifecycle
  transitions. This module is a transitional policy adapter: a pure decision
  function with no persistence and no lifecycle authority. Its actions are
  advisory signals only — they must NEVER be applied as direct writes to
  versioned memories (no `Memory.update/2` supersession, no silent skips that
  drop input). Governed intake is source-first: `OptimalEngine.Memory.remember/2`
  preserves every input as a `MemoryCore.SourcePackage`, dedup of governed
  truth is content-addressed (Source Package and Claim IDs), and supersession
  is decided at the `MemoryCore.FactPromoter` review boundary with derivation
  ledger entries.
  """

  alias OptimalEngine.Memory.EncodingGate

  @enforce_keys [:action, :reason, :similarity]
  defstruct action: :add,
            reason: "",
            similarity: 0.0,
            existing: nil

  @type action :: :add | :skip | :update

  @type t :: %__MODULE__{
          action: action(),
          reason: String.t(),
          similarity: float(),
          existing: map() | nil
        }

  @update_cues ~r/\b(actually|updated?|now|changed|no longer|instead|supersedes|replaces|correction|correct|wrong|contradicts)\b/i

  @doc """
  Decide add/update/skip for `content` against scoped live `candidates`.
  """
  @spec decide(String.t(), [map()], keyword()) :: t()
  def decide(content, candidates, opts \\ []) when is_binary(content) and is_list(candidates) do
    skip_threshold = Keyword.get(opts, :skip_threshold, 0.82)
    update_threshold = Keyword.get(opts, :update_threshold, 0.42)

    {similarity, existing} = closest(content, candidates)
    similarity = Float.round(similarity, 3)

    cond do
      is_nil(existing) ->
        %__MODULE__{action: :add, similarity: 0.0, reason: "no similar live memory"}

      normalized(content) == normalized(existing.content || "") ->
        %__MODULE__{action: :skip, existing: existing, similarity: 1.0, reason: "exact duplicate"}

      similarity >= skip_threshold and not update_cue?(content) ->
        %__MODULE__{
          action: :skip,
          existing: existing,
          similarity: similarity,
          reason: "semantic duplicate"
        }

      similarity >= update_threshold and update_cue?(content) ->
        %__MODULE__{
          action: :update,
          existing: existing,
          similarity: similarity,
          reason: "candidate appears to supersede similar memory"
        }

      similarity >= 0.60 and expands?(content, existing.content || "") ->
        %__MODULE__{
          action: :update,
          existing: existing,
          similarity: similarity,
          reason: "candidate expands similar memory"
        }

      true ->
        %__MODULE__{
          action: :add,
          existing: existing,
          similarity: similarity,
          reason: "distinct enough to store separately"
        }
    end
  end

  defp closest(_content, []), do: {0.0, nil}

  defp closest(content, candidates) do
    candidates
    |> Enum.map(fn candidate ->
      {EncodingGate.similarity(content, candidate.content || ""), candidate}
    end)
    |> Enum.max_by(fn {score, _candidate} -> score end, fn -> {0.0, nil} end)
  end

  defp update_cue?(content), do: Regex.match?(@update_cues, content)

  defp expands?(content, existing) do
    byte_size(content) > byte_size(existing) * 1.25
  end

  defp normalized(content) do
    content
    |> String.downcase()
    |> String.trim()
    |> String.replace(~r/\s+/, " ")
  end
end
