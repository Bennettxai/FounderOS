defmodule OptimalEngine.Pipeline.Classifier.Classification do
  @moduledoc """
  Per-chunk Signal Theory classification row.

  Mirrors the `classifications` table (Phase 1 migration 005):

      %Classification{
        chunk_id, tenant_id,
        mode, genre, signal_type, format, structure,  # the 5 Signal Theory dimensions
        sn_ratio, confidence
      }

  The Classifier produces one of these per chunk in the tree. Confidence is
  `0.0..1.0` and reflects how sure the classifier is about the inferred
  dimensions (higher when heuristics agree + Ollama concurs; lower when only
  heuristics fire, or a dimension was pinned from parser metadata).
  """

  @type mode :: :linguistic | :visual | :code | :data | :mixed | nil
  @type genre :: atom() | nil
  @type signal_type :: :direct | :inform | :commit | :decide | :express | nil
  @type format :: :markdown | :code | :json | :yaml | :csv | :html | :text | :unknown | nil
  @type structure :: atom() | nil

  @type t :: %__MODULE__{
          chunk_id: String.t(),
          tenant_id: String.t(),
          mode: mode(),
          genre: genre(),
          signal_type: signal_type(),
          format: format(),
          structure: structure(),
          sn_ratio: float() | nil,
          confidence: float() | nil
        }

  defstruct chunk_id: nil,
            tenant_id: "default",
            mode: nil,
            genre: nil,
            signal_type: nil,
            format: nil,
            structure: nil,
            sn_ratio: nil,
            confidence: nil

  @doc "Build a Classification with sensible defaults."
  @spec new(keyword()) :: t()
  def new(fields) when is_list(fields), do: struct(__MODULE__, fields)
end
