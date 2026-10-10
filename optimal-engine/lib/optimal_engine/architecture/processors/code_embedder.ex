defmodule OptimalEngine.Architecture.Processors.CodeEmbedder do
  @moduledoc """
  Code embedding — delegates to the text embedder for now.

  A dedicated code encoder (StarCoder-embed, CodeBERT, UniXCoder)
  lives in Phase 15. Until then the text model gives us 768-d
  vectors aligned with the rest of the space; retrieval quality on
  code will improve when the dedicated model lands.
  """

  @behaviour OptimalEngine.Architecture.Processor

  alias OptimalEngine.Architecture.Processors.TextEmbedder

  @impl true
  def id, do: :code_embedder

  @impl true
  def modality, do: :code

  @impl true
  def emits, do: [:embedding]

  @impl true
  def init(_config), do: {:ok, %{}}

  @impl true
  def process(field, value, state) do
    with {:ok, output} <- TextEmbedder.process(field, value, state) do
      {:ok, put_in(output, [:metadata, :via], :text_embedder_fallback)}
    end
  end
end
