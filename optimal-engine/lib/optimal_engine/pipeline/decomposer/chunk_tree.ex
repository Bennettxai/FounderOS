defmodule OptimalEngine.Pipeline.Decomposer.ChunkTree do
  @moduledoc """
  The output of `OptimalEngine.Pipeline.Decomposer.decompose/2`.

  A flat list of `%Chunk{}` plus the root (document-scale) chunk's id. The
  tree structure is encoded in `parent_id` on each chunk — not in nested
  data — so persistence is one-shot (bulk insert into the `chunks` table)
  and re-traversal is a simple `Enum.filter/2`.

  Use the helper functions (`at_scale/2`, `children_of/2`, `parent_of/2`,
  `counts/1`) to navigate. The `Decomposer` guarantees:

    * exactly one chunk at `:document` scale (`root_chunk_id` is its id)
    * every non-document chunk has a valid `parent_id`
    * chunks are ordered by `(scale, offset_bytes, id)` within a scale
  """

  alias OptimalEngine.Pipeline.Decomposer.Chunk

  @type t :: %__MODULE__{
          root_chunk_id: String.t(),
          chunks: [Chunk.t()]
        }

  defstruct root_chunk_id: nil, chunks: []

  @doc "Returns every chunk at a given scale in order."
  @spec at_scale(t(), Chunk.scale()) :: [Chunk.t()]
  def at_scale(%__MODULE__{chunks: chunks}, scale) when is_atom(scale) do
    Enum.filter(chunks, &(&1.scale == scale))
  end

  @doc "Returns the direct children of a chunk id (one scale below)."
  @spec children_of(t(), String.t()) :: [Chunk.t()]
  def children_of(%__MODULE__{chunks: chunks}, parent_id) when is_binary(parent_id) do
    Enum.filter(chunks, &(&1.parent_id == parent_id))
  end

  @doc "Returns the parent chunk of the given id, or nil."
  @spec parent_of(t(), String.t()) :: Chunk.t() | nil
  def parent_of(%__MODULE__{chunks: chunks}, child_id) when is_binary(child_id) do
    case Enum.find(chunks, &(&1.id == child_id)) do
      nil -> nil
      %Chunk{parent_id: nil} -> nil
      %Chunk{parent_id: pid} -> Enum.find(chunks, &(&1.id == pid))
    end
  end

  @doc "Returns per-scale counts as a map."
  @spec counts(t()) :: %{required(Chunk.scale()) => non_neg_integer()}
  def counts(%__MODULE__{chunks: chunks}) do
    Enum.reduce(
      [:document, :section, :paragraph, :chunk],
      %{},
      fn scale, acc ->
        Map.put(acc, scale, Enum.count(chunks, &(&1.scale == scale)))
      end
    )
  end

  @doc """
  Returns the root (`:document`-scale) chunk's text — byte-identical to the
  original `%ParsedDoc.text` since the document chunk covers the whole thing.
  This is the "reassembly" guarantee: one read from the root chunk
  reconstructs the source.
  """
  @spec reassemble(t()) :: String.t()
  def reassemble(%__MODULE__{chunks: chunks, root_chunk_id: rid}) do
    case Enum.find(chunks, &(&1.id == rid)) do
      nil -> ""
      %Chunk{text: text} -> text
    end
  end
end
