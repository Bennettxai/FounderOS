defmodule OptimalEngine.Pipeline.Decomposer.Chunk do
  @moduledoc """
  A single unit in the hierarchical decomposition of a `%ParsedDoc{}`.

  Chunks exist at four fixed scales:

    * `:document` — exactly one per signal; covers the full text
    * `:section`  — derived from parser-reported headings / pages / slides
    * `:paragraph`— derived from parser-reported paragraph boundaries
    * `:chunk`    — sliding-window retrieval unit within each paragraph

  Every non-document chunk has `parent_id` pointing at the chunk one scale
  above, forming a 4-level tree. The chunk ID is deterministic + idempotent:
  `"{signal_id}:<prefix>-<index>"` where `prefix ∈ {doc, sec, par, chk}` and
  index is the 0-based position within that scale for the signal.

  Fields mirror the `chunks` table created by Phase 1 migration 005.
  """

  @type scale :: :document | :section | :paragraph | :chunk

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          signal_id: String.t(),
          parent_id: String.t() | nil,
          scale: scale(),
          offset_bytes: non_neg_integer(),
          length_bytes: non_neg_integer(),
          text: String.t(),
          modality: atom(),
          asset_ref: String.t() | nil,
          classification_level: String.t()
        }

  defstruct id: nil,
            tenant_id: "default",
            workspace_id: "default",
            signal_id: nil,
            parent_id: nil,
            scale: :chunk,
            offset_bytes: 0,
            length_bytes: 0,
            text: "",
            modality: :text,
            asset_ref: nil,
            classification_level: "internal"

  @scale_prefix %{
    document: "doc",
    section: "sec",
    paragraph: "par",
    chunk: "chk"
  }

  @doc "Build a deterministic chunk ID."
  @spec build_id(String.t(), scale(), non_neg_integer()) :: String.t()
  def build_id(signal_id, scale, index)
      when is_binary(signal_id) and is_atom(scale) and is_integer(index) do
    "#{signal_id}:#{Map.fetch!(@scale_prefix, scale)}-#{index}"
  end

  @doc "Convenience constructor with sensible defaults."
  @spec new(keyword()) :: t()
  def new(fields) when is_list(fields) do
    fields =
      Keyword.put_new_lazy(fields, :length_bytes, fn ->
        fields |> Keyword.get(:text, "") |> byte_size()
      end)

    struct(__MODULE__, fields)
  end
end
