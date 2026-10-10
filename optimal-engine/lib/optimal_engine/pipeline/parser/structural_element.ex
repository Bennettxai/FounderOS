defmodule OptimalEngine.Pipeline.Parser.StructuralElement do
  @moduledoc """
  Represents a single structural boundary inside a `%ParsedDoc{}`: a heading,
  a page, a slide, a timestamp, a code block, a paragraph, a table row.

  The `kind` field discriminates the element type; `metadata` carries
  kind-specific extras (heading level, page number, slide notes, audio
  offset seconds, code block language, etc.).

  `offset` + `length` are byte positions into the parent `ParsedDoc.text`, so
  the Decomposer can compute parent-aware chunks cheaply.

  ## Kinds

  | kind           | metadata shape                              |
  |----------------|---------------------------------------------|
  | `:heading`     | `%{level: 1..6}`                            |
  | `:section`     | `%{title: String.t() \\| nil}`              |
  | `:page`        | `%{number: pos_integer()}`                  |
  | `:slide`       | `%{number: pos_integer(), notes: String.t() \\| nil}` |
  | `:timestamp`   | `%{seconds: float()}`                       |
  | `:code_block`  | `%{language: String.t() \\| nil}`           |
  | `:paragraph`   | `%{}`                                       |
  | `:table_row`   | `%{columns: [String.t()]}`                  |
  """

  @type kind ::
          :heading
          | :section
          | :page
          | :slide
          | :timestamp
          | :code_block
          | :paragraph
          | :table_row

  @type t :: %__MODULE__{
          kind: kind(),
          text: String.t(),
          offset: non_neg_integer(),
          length: non_neg_integer(),
          metadata: map()
        }

  defstruct kind: :paragraph,
            text: "",
            offset: 0,
            length: 0,
            metadata: %{}

  @doc "Build a StructuralElement with sensible defaults."
  @spec new(kind(), keyword()) :: t()
  def new(kind, fields) when is_atom(kind) and is_list(fields) do
    fields
    |> Keyword.put(:kind, kind)
    |> Keyword.put_new(:length, String.length(Keyword.get(fields, :text, "")))
    |> then(&struct(__MODULE__, &1))
  end
end
