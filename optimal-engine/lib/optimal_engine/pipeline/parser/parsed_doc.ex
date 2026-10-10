defmodule OptimalEngine.Pipeline.Parser.ParsedDoc do
  @moduledoc """
  The output contract for every parser backend in the pipeline.

  Stage 2 (`OptimalEngine.Pipeline.Parser`) consumes a `%RawSignal{}` (or a
  filesystem path) and produces exactly one `%ParsedDoc{}`. Stage 3
  (`OptimalEngine.Pipeline.Decomposer`) takes that `%ParsedDoc{}` and breaks
  it down into hierarchical chunks.

  Every backend — markdown, pdf, office, image, audio, video, and everything
  between — returns this same shape. That's what lets Stage 3 chunk anything
  without caring about the source format.

  ## Fields

  - `path` — original filesystem path, if the doc came from a file. `nil` for
    inline text.
  - `signal_id` — deterministic content-hash id (`"sha256:<hex>"`).
  - `text` — the extracted plain text. This is the primary input to Stage 3.
  - `structure` — a list of `%StructuralElement{}` preserving boundaries the
    parser observed (headings, pages, slides, timestamps, code blocks). Stage
    3 uses these to avoid splitting chunks across structural seams.
  - `assets` — list of `%Asset{}` references for binary content (images,
    audio, video) the parser preserved alongside the extracted text.
  - `modality` — primary modality of the source: `:text | :code | :data |
    :image | :audio | :video | :mixed`.
  - `metadata` — format-specific extras (encoding, page count, author, audio
    duration, image dimensions, etc.). Free-form map.
  - `warnings` — human-readable notes the parser attached when it had to
    degrade gracefully (e.g., "pdftotext not on PATH; returning metadata
    only").
  """

  alias OptimalEngine.Pipeline.Parser.{Asset, StructuralElement}

  @type modality :: :text | :code | :data | :image | :audio | :video | :mixed

  @type t :: %__MODULE__{
          path: String.t() | nil,
          signal_id: String.t() | nil,
          text: String.t(),
          structure: [StructuralElement.t()],
          assets: [Asset.t()],
          modality: modality(),
          metadata: map(),
          warnings: [String.t()]
        }

  defstruct path: nil,
            signal_id: nil,
            text: "",
            structure: [],
            assets: [],
            modality: :text,
            metadata: %{},
            warnings: []

  @doc "Build a ParsedDoc, hashing the text to derive `signal_id` when omitted."
  @spec new(keyword()) :: t()
  def new(fields) do
    fields = Keyword.put_new_lazy(fields, :signal_id, fn -> hash(fields[:text] || "") end)
    struct(__MODULE__, fields)
  end

  @doc "Appends a warning to an existing ParsedDoc."
  @spec warn(t(), String.t()) :: t()
  def warn(%__MODULE__{} = doc, message) when is_binary(message) do
    %{doc | warnings: doc.warnings ++ [message]}
  end

  defp hash(text) when is_binary(text) do
    "sha256:" <> (:crypto.hash(:sha256, text) |> Base.encode16(case: :lower))
  end
end
