defmodule OptimalEngine.Pipeline.Embedder.Embedding do
  @moduledoc """
  Per-chunk embedding row.

  Mirrors the `chunk_embeddings` table (Phase 5 migration 023). Every chunk
  produced by Stage 3 (Decomposer) becomes one row here with a 768-dim
  vector in the aligned nomic space — text via `nomic-embed-text`, image
  via `nomic-embed-vision`, audio via whisper-transcript→text-embed.

  Fields:

    * `chunk_id`   — FK to chunks.id
    * `tenant_id`  — tenant scope
    * `model`      — the embedding model identifier (e.g. `"nomic-embed-text"`)
    * `modality`   — original modality of the chunk (`:text | :image | :audio | :code | :data | :mixed`)
    * `dim`        — dimension of the vector (always 768 for v1 providers)
    * `vector`     — the float32 values
  """

  @type modality :: :text | :image | :audio | :video | :code | :data | :mixed

  @type t :: %__MODULE__{
          chunk_id: String.t(),
          tenant_id: String.t(),
          model: String.t(),
          modality: modality(),
          dim: non_neg_integer(),
          vector: [float()]
        }

  defstruct chunk_id: nil,
            tenant_id: "default",
            model: nil,
            modality: :text,
            dim: 768,
            vector: []

  @doc "Build an Embedding from fields, computing `dim` from `vector` length."
  @spec new(keyword()) :: t()
  def new(fields) when is_list(fields) do
    fields =
      Keyword.put_new_lazy(fields, :dim, fn ->
        fields |> Keyword.get(:vector, []) |> length()
      end)

    struct(__MODULE__, fields)
  end
end
