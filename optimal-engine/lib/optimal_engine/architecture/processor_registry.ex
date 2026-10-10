defmodule OptimalEngine.Architecture.ProcessorRegistry do
  @moduledoc """
  Registry of available **processors** — the things that actually
  process a field (embed it, classify it, extract entities from it,
  transcribe it, score it for anomalies, …).

  Each processor declares its `id/0` atom, `modality/0` it consumes,
  and `emits/0` output kinds. Adding a new processor — whether it
  wraps a local Ollama model, a remote vision API, a classical
  rule-based algorithm, or an agent workflow — means implementing
  `OptimalEngine.Architecture.Processor` and appending to
  `@processors` below.

  The engine's **existing** pipeline stages (Embedder, Classifier,
  EntityExtractor, Decomposer) are processors too. They live under
  `Architecture.Processors.*` as thin wrappers pointing at the
  Phase 2-6 implementations.
  """

  alias OptimalEngine.Architecture.Processors

  @processors [
    Processors.TextEmbedder,
    Processors.ImageEmbedder,
    Processors.AudioEmbedder,
    Processors.VideoEmbedder,
    Processors.CodeEmbedder,
    Processors.TsFeatureExtractor,
    Processors.VlmCaptioner
  ]

  @doc "All registered processor modules."
  @spec all() :: [module()]
  def all, do: @processors

  @doc "Look up a processor module by its `id/0` atom."
  @spec fetch(atom()) :: {:ok, module()} | {:error, :unknown_processor}
  def fetch(id) when is_atom(id) do
    Enum.find(@processors, fn mod -> mod.id() == id end)
    |> case do
      nil -> {:error, :unknown_processor}
      mod -> {:ok, mod}
    end
  end

  @doc "Summary triples for CLI listings."
  @spec summary() :: [{atom(), atom(), [atom()]}]
  def summary do
    Enum.map(@processors, fn mod -> {mod.id(), mod.modality(), mod.emits()} end)
  end
end
