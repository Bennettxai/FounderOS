defmodule OptimalEngine.Architecture.Architecture do
  @moduledoc """
  A **DataArchitecture** describes the shape of one class of data point —
  a clinical visit, a sales call, a code commit, an IoT sensor window,
  a satellite tile — as a composition of typed `Field`s plus the
  processor bindings that turn each field into something retrievable.

  The engine is **model-agnostic**: an architecture doesn't say "use
  GPT-4"; it says "apply a text-embedding processor to the `note`
  field". Swap the processor, the data-point shape is unchanged.

  ## Why this exists

  Most retrieval systems bake in one data model ("a chunk of text").
  Real organizations produce every dimension — transcripts and scans
  and CSV exports and code diffs and time-series telemetry. A
  universal storage layer needs a schema that can talk about any of
  them without losing the **granularity** (what to chunk, at what
  scale) or the **processing contract** (which model/algorithm
  owns each field).

  ## Struct

      %Architecture{
        id:           "text_signal.v1",
        name:         "text_signal",
        version:      1,
        description:  "Free-text signal (note, transcript, doc)",
        modality_primary: :text,
        fields:       [%Field{}, ...],
        granularity:  [:document, :section, :paragraph, :sentence],
        retention:    :default,
        metadata:     %{...}
      }

  See `OptimalEngine.Architecture.Architectures.*` for built-in
  definitions and `OptimalEngine.Architecture.Registry` for lookup.
  """

  alias OptimalEngine.Architecture.Field

  @type granularity :: atom()

  @type t :: %__MODULE__{
          id: String.t(),
          name: String.t(),
          version: pos_integer(),
          description: String.t() | nil,
          modality_primary: Field.modality(),
          fields: [Field.t()],
          granularity: [granularity()],
          retention: atom(),
          metadata: map()
        }

  @enforce_keys [:id, :name, :modality_primary]
  defstruct id: nil,
            name: nil,
            version: 1,
            description: nil,
            modality_primary: :text,
            fields: [],
            granularity: [:document],
            retention: :default,
            metadata: %{}

  @doc "Build an architecture with sensible defaults."
  @spec new(keyword()) :: t()
  def new(attrs) do
    attrs = Map.new(attrs)
    name = Map.fetch!(attrs, :name)
    version = Map.get(attrs, :version, 1)
    id = Map.get(attrs, :id, "#{name}.v#{version}")

    struct!(__MODULE__, Map.merge(%{id: id, version: version}, attrs))
  end

  @doc "Look up a field by its atom name."
  @spec field(t(), atom()) :: Field.t() | nil
  def field(%__MODULE__{fields: fields}, name) when is_atom(name) do
    Enum.find(fields, fn f -> f.name == name end)
  end

  @doc "Required field names."
  @spec required_fields(t()) :: [atom()]
  def required_fields(%__MODULE__{fields: fields}) do
    fields |> Enum.filter(& &1.required) |> Enum.map(& &1.name)
  end

  @doc "Round-trip to a JSON-friendly map for persistence in `architectures.spec`."
  @spec to_spec(t()) :: map()
  def to_spec(%__MODULE__{} = arch) do
    %{
      "id" => arch.id,
      "name" => arch.name,
      "version" => arch.version,
      "description" => arch.description,
      "modality_primary" => Atom.to_string(arch.modality_primary),
      "granularity" => Enum.map(arch.granularity, &Atom.to_string/1),
      "retention" => Atom.to_string(arch.retention),
      "fields" =>
        Enum.map(arch.fields, fn f ->
          %{
            "name" => Atom.to_string(f.name),
            "modality" => Atom.to_string(f.modality),
            "dims" => Enum.map(f.dims, &format_dim/1),
            "required" => f.required,
            "processor" => if(f.processor, do: Atom.to_string(f.processor)),
            "description" => f.description
          }
        end),
      "metadata" => arch.metadata
    }
  end

  defp format_dim(:any), do: "any"
  defp format_dim(n) when is_integer(n), do: n
end
