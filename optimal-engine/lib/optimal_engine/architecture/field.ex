defmodule OptimalEngine.Architecture.Field do
  @moduledoc """
  One field inside a data architecture.

  A data point is more than "text" — it's a composition of fields, each
  with its own **modality**, **shape**, and preferred **processor**. A
  clinical visit carries a patient id (structured), a free-text note
  (text), attached images (visual), and a timestamp (temporal); the
  engine stores all of it as one data point but dispatches each field
  to the right processor.

  ## Modality taxonomy

  The `:modality` atom tells the engine what kind of signal the field
  carries. Every modality maps to a default embedding family, a
  decomposition strategy, and a classifier preset.

      :text         linguistic content (prose, code comments)
      :code         source code (has its own tokenizer)
      :image        still visual (PNG/JPEG/SVG)
      :audio        waveform (speech or music)
      :video        time-indexed visual
      :time_series  numeric sequence sampled over time
      :table        structured rows + columns
      :structured   key-value record
      :graph        nodes + edges (as a first-class field type)
      :tensor       arbitrary-shape numeric array
      :geo          lat/lon or polygon
      :binary       opaque byte stream (fallback)

  ## Shape

  `:dims` is a list of integer or `:any` dimensions. A 768-d embedding
  → `[768]`. A 224×224 RGB image → `[3, 224, 224]`. A stream whose
  length varies → `[:any, 768]`. This is what lets the engine reject
  mis-shaped inputs before they hit a model that can't handle them.

  ## Processor hint

  `:processor` is the id of the preferred processor in the registry
  (see `OptimalEngine.Architecture.Processor`). A field without a
  hint falls back to the registry default for that modality.
  """

  @type modality ::
          :text
          | :code
          | :image
          | :audio
          | :video
          | :time_series
          | :table
          | :structured
          | :graph
          | :tensor
          | :geo
          | :binary

  @type dim :: non_neg_integer() | :any

  @type t :: %__MODULE__{
          name: atom(),
          modality: modality(),
          dims: [dim()],
          required: boolean(),
          processor: atom() | nil,
          description: String.t() | nil
        }

  @enforce_keys [:name, :modality]
  defstruct [
    :name,
    :modality,
    dims: [],
    required: false,
    processor: nil,
    description: nil
  ]

  @doc "Every modality recognized by the engine."
  @spec modalities() :: [modality()]
  def modalities do
    ~w(text code image audio video time_series table structured graph tensor geo binary)a
  end

  @doc """
  Shallow compatibility check between a declared field spec and a runtime
  value. Not full schema validation — a full validator lives in
  `Architecture.Apply.validate/2`.
  """
  @spec compatible?(t(), any()) :: boolean()
  def compatible?(%__MODULE__{required: true}, nil), do: false
  def compatible?(%__MODULE__{}, nil), do: true

  def compatible?(%__MODULE__{modality: :text}, v), do: is_binary(v)
  def compatible?(%__MODULE__{modality: :code}, v), do: is_binary(v)

  def compatible?(%__MODULE__{modality: :image}, v) do
    is_binary(v) or match?(%{path: _}, v) or match?(%{url: _}, v)
  end

  def compatible?(%__MODULE__{modality: :audio}, v) do
    is_binary(v) or match?(%{path: _}, v) or match?(%{url: _}, v)
  end

  def compatible?(%__MODULE__{modality: :video}, v) do
    is_binary(v) or match?(%{path: _}, v) or match?(%{url: _}, v)
  end

  def compatible?(%__MODULE__{modality: :time_series}, v), do: is_list(v)
  def compatible?(%__MODULE__{modality: :table}, v), do: is_list(v) or is_map(v)
  def compatible?(%__MODULE__{modality: :structured}, v), do: is_map(v)

  def compatible?(%__MODULE__{modality: :graph}, v) do
    is_map(v) and Map.has_key?(v, :nodes) and Map.has_key?(v, :edges)
  end

  def compatible?(%__MODULE__{modality: :tensor}, v), do: is_list(v) or is_binary(v)
  def compatible?(%__MODULE__{modality: :geo}, v), do: is_map(v) or is_tuple(v)
  def compatible?(%__MODULE__{modality: :binary}, v), do: is_binary(v)
  def compatible?(%__MODULE__{}, _), do: false
end
