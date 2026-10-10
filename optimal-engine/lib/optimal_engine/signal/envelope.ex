defmodule OptimalEngine.Signal.Envelope do
  @moduledoc """
  The core Signal struct — a CloudEvents v1.0.2 envelope fused with Signal Theory dimensions.

  Every event in the MIOSA ecosystem is a Signal. A Signal carries:

  - **CloudEvents v1.0.2** required and optional attributes for interoperability
  - **Signal Theory Mode + Genre + Type + Format + Structure** dimensions for quality classification
  - **MIOSA extensions** for agent context, causality chains, and correlation

  ## CloudEvents v1.0.2 Compliance

  Required attributes: `id`, `source`, `type`, `specversion`.
  Optional attributes: `time`, `subject`, `data`, `datacontenttype`, `dataschema`.

  ## Signal Theory Dimensions

  | Dimension | Field | Values |
  |-----------|-------|--------|
  | Mode (M) | `signal_mode` | `:linguistic`, `:visual`, `:code`, `:mixed` |
  | Genre (G) | `signal_genre` | `:spec`, `:report`, `:pr`, `:adr`, `:brief`, `:chat`, `:error`, `:progress`, `:alert` |
  | Type (T) | `signal_type` | `:direct`, `:inform`, `:commit`, `:decide`, `:express` |
  | Format (F) | `signal_format` | `:markdown`, `:code`, `:json`, `:cli`, `:diff`, `:table` |
  | Structure (W) | `signal_structure` | Genre-specific template key (atom) |

  ## Submodules

  - `OptimalEngine.Signal.Envelope.Validation` — field validation and Signal Theory constraint checks
  - `OptimalEngine.Signal.Envelope.Builder` — CloudEvents serialization and deserialization
  """

  alias __MODULE__, as: Signal
  alias OptimalEngine.Signal.Envelope.Builder
  alias OptimalEngine.Signal.Envelope.Validation

  @type signal_mode :: :linguistic | :visual | :code | :mixed
  @type signal_genre ::
          :spec | :report | :pr | :adr | :brief | :chat | :error | :progress | :alert
  @type signal_type :: :direct | :inform | :commit | :decide | :express
  @type signal_format :: :markdown | :code | :json | :cli | :diff | :table
  @type agent_tier :: :elite | :specialist | :utility

  @type t :: %Signal{
          # CloudEvents v1.0.2 required
          id: String.t() | nil,
          source: String.t() | nil,
          type: String.t() | nil,
          specversion: String.t(),
          # CloudEvents v1.0.2 optional
          time: DateTime.t() | nil,
          subject: String.t() | nil,
          data: term(),
          datacontenttype: String.t(),
          dataschema: String.t() | nil,
          # Signal Theory extensions
          signal_mode: signal_mode() | nil,
          signal_genre: signal_genre() | nil,
          signal_type: signal_type() | nil,
          signal_format: signal_format() | nil,
          signal_structure: atom() | nil,
          signal_sn_ratio: float() | nil,
          # MIOSA extensions
          agent_id: String.t() | nil,
          agent_tier: agent_tier() | nil,
          session_id: String.t() | nil,
          parent_id: String.t() | nil,
          correlation_id: String.t() | nil,
          extensions: map()
        }

  defstruct [
    # CloudEvents v1.0.2 required
    :id,
    :source,
    :type,
    # CloudEvents v1.0.2 optional
    :time,
    :subject,
    :data,
    :dataschema,
    # Signal Theory extensions
    :signal_mode,
    :signal_genre,
    :signal_type,
    :signal_format,
    :signal_structure,
    :signal_sn_ratio,
    # MIOSA extensions
    :agent_id,
    :agent_tier,
    :session_id,
    :parent_id,
    :correlation_id,
    # Defaults (must come last)
    specversion: "1.0.2",
    datacontenttype: "application/json",
    extensions: %{}
  ]

  # ── Construction ──────────────────────────────────────────────────

  @doc """
  Creates a new signal with the given type and options.

  Auto-generates `id` (UUID), `time` (UTC now), and `specversion` ("1.0.2").

  ## Options

  All struct fields can be passed as keyword options. At minimum, `source` should
  be provided for a well-formed CloudEvents signal.

  ## Examples

      iex> {:ok, signal} = OptimalEngine.Signal.Envelope.new("miosa.test.event", source: "/test")
      iex> signal.type
      "miosa.test.event"
      iex> signal.specversion
      "1.0.2"

      iex> {:ok, signal} = OptimalEngine.Signal.Envelope.new("miosa.test.data", source: "/test", data: %{key: "value"})
      iex> signal.data
      %{key: "value"}
  """
  @spec new(String.t(), keyword()) :: {:ok, t()} | {:error, term()}
  def new(type, opts \\ []) when is_binary(type) do
    signal = %Signal{
      id: Keyword.get(opts, :id, UUID.uuid4()),
      source: Keyword.get(opts, :source),
      type: type,
      specversion: "1.0.2",
      time: Keyword.get(opts, :time, DateTime.utc_now()),
      subject: Keyword.get(opts, :subject),
      data: Keyword.get(opts, :data),
      datacontenttype: Keyword.get(opts, :datacontenttype, "application/json"),
      dataschema: Keyword.get(opts, :dataschema),
      signal_mode: Keyword.get(opts, :signal_mode),
      signal_genre: Keyword.get(opts, :signal_genre),
      signal_type: Keyword.get(opts, :signal_type),
      signal_format: Keyword.get(opts, :signal_format),
      signal_structure: Keyword.get(opts, :signal_structure),
      signal_sn_ratio: Keyword.get(opts, :signal_sn_ratio),
      agent_id: Keyword.get(opts, :agent_id),
      agent_tier: Keyword.get(opts, :agent_tier),
      session_id: Keyword.get(opts, :session_id),
      parent_id: Keyword.get(opts, :parent_id),
      correlation_id: Keyword.get(opts, :correlation_id),
      extensions: Keyword.get(opts, :extensions, %{})
    }

    case Validation.validate(signal) do
      :ok -> {:ok, signal}
      {:error, _} = err -> err
    end
  end

  @doc """
  Like `new/2` but raises on validation failure.

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test.event", source: "/test")
      iex> is_binary(signal.id)
      true
  """
  @spec new!(String.t(), keyword()) :: t()
  def new!(type, opts \\ []) do
    case new(type, opts) do
      {:ok, signal} -> signal
      {:error, reason} -> raise ArgumentError, "invalid signal: #{inspect(reason)}"
    end
  end

  @doc """
  Applies Signal Theory classification to a signal.

  ## Options

  - `:mode` - Signal mode (`:linguistic`, `:visual`, `:code`, `:mixed`)
  - `:genre` - Signal genre (`:spec`, `:report`, `:pr`, `:adr`, `:brief`, `:chat`, `:error`, `:progress`, `:alert`)
  - `:type` - Signal type (`:direct`, `:inform`, `:commit`, `:decide`, `:express`)
  - `:format` - Signal format (`:markdown`, `:code`, `:json`, `:cli`, `:diff`, `:table`)
  - `:structure` - Genre-specific template key (atom)
  - `:sn_ratio` - Signal-to-noise ratio estimate (0.0 to 1.0)

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test", source: "/test")
      iex> {:ok, classified} = OptimalEngine.Signal.Envelope.classify(signal, mode: :code, genre: :spec, type: :inform, format: :markdown)
      iex> classified.signal_mode
      :code
  """
  @spec classify(t(), keyword()) :: {:ok, t()} | {:error, term()}
  def classify(%Signal{} = signal, opts) when is_list(opts) do
    classified = %Signal{
      signal
      | signal_mode: Keyword.get(opts, :mode, signal.signal_mode),
        signal_genre: Keyword.get(opts, :genre, signal.signal_genre),
        signal_type: Keyword.get(opts, :type, signal.signal_type),
        signal_format: Keyword.get(opts, :format, signal.signal_format),
        signal_structure: Keyword.get(opts, :structure, signal.signal_structure),
        signal_sn_ratio: Keyword.get(opts, :sn_ratio, signal.signal_sn_ratio)
    }

    case Validation.validate(classified) do
      :ok -> {:ok, classified}
      {:error, _} = err -> err
    end
  end

  @doc """
  Creates a child signal linked to the parent via `parent_id` and `correlation_id`.

  The child inherits the parent's `correlation_id` (or uses the parent's `id` if none set),
  `source`, `session_id`, `agent_id`, and `agent_tier`.

  ## Examples

      iex> parent = OptimalEngine.Signal.Envelope.new!("miosa.parent", source: "/test")
      iex> {:ok, child} = OptimalEngine.Signal.Envelope.chain(parent, "miosa.child", data: %{step: 2})
      iex> child.parent_id == parent.id
      true
  """
  @spec chain(t(), String.t(), keyword()) :: {:ok, t()} | {:error, term()}
  def chain(%Signal{} = parent, child_type, opts \\ []) when is_binary(child_type) do
    correlation = parent.correlation_id || parent.id

    child_opts =
      Keyword.merge(
        [
          source: parent.source,
          parent_id: parent.id,
          correlation_id: correlation,
          session_id: parent.session_id,
          agent_id: parent.agent_id,
          agent_tier: parent.agent_tier
        ],
        opts
      )

    new(child_type, child_opts)
  end

  # ── Validation delegates ──────────────────────────────────────────
  # Full docs in `OptimalEngine.Signal.Envelope.Validation`.

  @doc "See `OptimalEngine.Signal.Envelope.Validation.validate/1`."
  defdelegate validate(signal), to: Validation

  @doc "See `OptimalEngine.Signal.Envelope.Validation.shannon_check/2`."
  defdelegate shannon_check(signal, max_bytes), to: Validation

  @doc "See `OptimalEngine.Signal.Envelope.Validation.ashby_check/1`."
  defdelegate ashby_check(signal), to: Validation

  @doc "See `OptimalEngine.Signal.Envelope.Validation.beer_check/1`."
  defdelegate beer_check(signal), to: Validation

  @doc "See `OptimalEngine.Signal.Envelope.Validation.wiener_check/2`."
  defdelegate wiener_check(signal, acknowledged_ids), to: Validation

  # ── Builder delegates ─────────────────────────────────────────────
  # Full docs in `OptimalEngine.Signal.Envelope.Builder`.

  @doc "See `OptimalEngine.Signal.Envelope.Builder.to_cloud_event/1`."
  defdelegate to_cloud_event(signal), to: Builder

  @doc "See `OptimalEngine.Signal.Envelope.Builder.from_cloud_event/1`."
  defdelegate from_cloud_event(map), to: Builder
end
