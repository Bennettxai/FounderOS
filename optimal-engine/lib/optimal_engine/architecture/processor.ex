defmodule OptimalEngine.Architecture.Processor do
  @moduledoc """
  Behaviour every data processor implements.

  A **processor** is the generic word for "thing that takes a field
  value and emits something the engine can index, retrieve, or reason
  over". It is deliberately **model-agnostic**:

    * A text-embedder that calls Ollama is a processor.
    * A vision model (CLIP, SigLIP) is a processor.
    * A time-series anomaly detector is a processor.
    * A classical regex + symbolic rule engine is a processor.
    * An agent workflow that negotiates with external services is
      a processor.

  The engine doesn't care which category a processor belongs to — it
  cares that the processor can be initialized, can process a field,
  and can declare what it emits.

  ## Callbacks

      id/0            stable atom the architecture refers to
      modality/0      the modality this processor consumes
      emits/0         atom(s) this processor produces
                      (e.g. `:embedding`, `:entities`, `:classification`)
      init/1          hydrate runtime state from config
      process/3       given (field, value, state) produce output

  ## Output shape

  `process/3` returns `{:ok, %{kind: emit_kind, value: any, metadata: map}}`
  or `{:error, reason}`. The `Apply` module persists each successful
  emission into `processor_runs` plus the appropriate downstream
  table (embeddings, entities, classifications, etc).

  ## Why not re-use Pipeline?

  The Phase 2-6 Pipeline stages (Parser, Decomposer, Classifier,
  Embedder, Clusterer) are hard-wired for text signals with the
  engine's built-in granularities. Processors generalize that to
  any field type + any model. Pipeline stages are implemented as
  processors in the registry — they're the defaults, not the only
  option.
  """

  alias OptimalEngine.Architecture.Field

  @type emit_kind ::
          :embedding
          | :classification
          | :entities
          | :summary
          | :transcription
          | :ocr
          | :tags
          | :features
          | :score
          | :anomaly
          | :caption
          | :segmentation
          | :noop

  @type output :: %{
          required(:kind) => emit_kind(),
          required(:value) => any(),
          optional(:metadata) => map()
        }

  @callback id() :: atom()
  @callback modality() :: Field.modality() | :any
  @callback emits() :: [emit_kind()]
  @callback init(config :: map()) :: {:ok, state :: term()} | {:error, term()}
  @callback process(field :: Field.t(), value :: any(), state :: term()) ::
              {:ok, output()} | {:error, term()}

  @optional_callbacks [init: 1]
end
