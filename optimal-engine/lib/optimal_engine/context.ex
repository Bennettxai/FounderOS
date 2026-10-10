defmodule OptimalEngine.Context do
  @moduledoc """
  Universal context unit for OptimalEngine.

  Every piece of ingested content — regardless of origin — is a Context. The three
  base types mirror tiered context systems's model:

  - `:resource`  — Static knowledge: docs, PDFs, specs, manuals, API references
  - `:memory`    — Dynamic learned facts from conversations or agent observations
  - `:skill`     — Callable tool/function definitions
  - `:signal`    — OptimalOS extension: classified content with full Mode + Genre + Type + Format + Structure dimensions

  Only `:signal` contexts carry a populated `signal` field. All other types leave
  it `nil`. This lets the engine ingest anything while preserving the full Signal
  Theory classification for content that deserves it.

  ## URI scheme

  Every context is addressable via an `optimal://` URI:

      optimal://resources/{path}          — static knowledge
      optimal://user/memories/{path}      — user-learned facts
      optimal://agent/skills/{path}       — callable tools
      optimal://agent/memories/{path}     — agent-learned patterns
      optimal://nodes/{node-id}/{path}    — org node content (12 folders)
      optimal://sessions/{id}/{path}      — conversation history
      optimal://inbox/{path}              — unrouted content

  ## Tiered summaries

  Every context has three content tiers:
  - `l0_abstract`  — ~100 tokens: a single-line machine-readable descriptor
  - `l1_overview`  — ~1-2K tokens: enough for retrieval decisions
  - `content`      — full content (L2)
  """

  alias OptimalEngine.Signal

  @type context_type :: :resource | :memory | :skill | :signal

  @type t :: %__MODULE__{
          id: String.t(),
          uri: String.t(),
          type: context_type(),
          path: String.t() | nil,
          title: String.t(),
          content: String.t(),
          # Tiered summaries
          l0_abstract: String.t(),
          l1_overview: String.t(),
          # Signal classification (populated only when type == :signal)
          signal: Signal.t() | nil,
          # Classification metadata
          node: String.t() | nil,
          sn_ratio: float(),
          entities: [String.t()],
          # Temporal
          created_at: DateTime.t(),
          modified_at: DateTime.t(),
          valid_from: DateTime.t() | nil,
          valid_until: DateTime.t() | nil,
          supersedes: String.t() | nil,
          # Routing
          routed_to: [String.t()],
          # Workspace scoping (soft scope within a tenant)
          workspace_id: String.t(),
          # Extra
          metadata: map(),
          # Search score (transient — not persisted)
          score: float() | nil
        }

  defstruct [
    :id,
    :uri,
    :type,
    :path,
    :title,
    :content,
    :l0_abstract,
    :l1_overview,
    :signal,
    :node,
    :created_at,
    :modified_at,
    :valid_from,
    :valid_until,
    :supersedes,
    :score,
    sn_ratio: 0.5,
    entities: [],
    routed_to: [],
    workspace_id: "default",
    metadata: %{}
  ]

  @doc "Returns all valid context types."
  @spec valid_types() :: [context_type()]
  def valid_types, do: [:resource, :memory, :skill, :signal]

  @doc """
  Converts a Context struct to a flat map suitable for SQLite insertion.
  Signal dimensions are extracted from the embedded signal (if present).
  """
  @spec to_row(t()) :: map()
  def to_row(%__MODULE__{} = ctx) do
    base = to_row_base(ctx)
    signal_dims = signal_dimensions(ctx.signal)
    Map.merge(base, signal_dims)
  end

  defp to_row_base(ctx) do
    Map.merge(to_row_identity(ctx), to_row_temporal(ctx))
  end

  # credo:disable-for-next-line Credo.Check.Refactor.CyclomaticComplexity
  defp to_row_identity(ctx) do
    %{
      id: ctx.id,
      uri: ctx.uri || "",
      type: to_string(ctx.type || :resource),
      path: ctx.path,
      title: ctx.title || "",
      l0_abstract: ctx.l0_abstract || "",
      l1_overview: ctx.l1_overview || "",
      content: ctx.content || "",
      node: ctx.node || "inbox",
      sn_ratio: ctx.sn_ratio || 0.5,
      entities: Jason.encode!(ctx.entities || []),
      workspace_id: ctx.workspace_id || "default"
    }
  end

  defp to_row_temporal(ctx) do
    %{
      created_at: serialize_dt(ctx.created_at),
      modified_at: serialize_dt(ctx.modified_at),
      valid_from: serialize_dt(ctx.valid_from),
      valid_until: serialize_dt(ctx.valid_until),
      supersedes: ctx.supersedes,
      routed_to: Jason.encode!(ctx.routed_to || []),
      metadata: Jason.encode!(ctx.metadata || %{})
    }
  end

  # Signal dimensions — NULL for non-signal types
  defp signal_dimensions(nil) do
    %{mode: nil, genre: nil, signal_type: nil, format: nil, structure: nil}
  end

  defp signal_dimensions(sig) do
    %{
      mode: to_string(sig.mode || :linguistic),
      genre: sig.genre || "note",
      signal_type: to_string(sig.type || :inform),
      format: to_string(sig.format || :markdown),
      structure: sig.structure || ""
    }
  end

  @doc """
  Reconstructs a Context from a SQLite row (list of column values).
  Column order must match `context_columns/0` in the Store.
  """
  @spec from_row([term()]) :: t()
  def from_row(row) when is_list(row) do
    [
      id,
      uri,
      type,
      path,
      title,
      l0_abstract,
      l1_overview,
      content,
      mode,
      genre,
      signal_type,
      format,
      structure,
      node,
      sn_ratio,
      entities_json,
      created_at,
      modified_at,
      valid_from,
      valid_until,
      supersedes,
      routed_to_json,
      metadata_json,
      workspace_id
    ] = row

    ctx_type = parse_context_type(type)

    parsed =
      parse_common_fields(
        id: id,
        path: path,
        title: title,
        l0_abstract: l0_abstract,
        l1_overview: l1_overview,
        content: content,
        node: node,
        sn_ratio: sn_ratio,
        entities_json: entities_json,
        created_at: created_at,
        modified_at: modified_at,
        valid_from: valid_from,
        valid_until: valid_until,
        supersedes: supersedes,
        routed_to_json: routed_to_json
      )

    signal =
      if ctx_type == :signal do
        build_signal_from_row(parsed, mode, genre, signal_type, format, structure)
      else
        nil
      end

    %__MODULE__{
      id: id,
      uri: uri || "",
      type: ctx_type,
      path: path,
      title: parsed.title,
      content: parsed.content,
      l0_abstract: parsed.l0_abstract,
      l1_overview: parsed.l1_overview,
      signal: signal,
      node: parsed.node,
      sn_ratio: parsed.sn_ratio,
      entities: parsed.entities,
      created_at: parsed.created_at,
      modified_at: parsed.modified_at,
      valid_from: parsed.valid_from,
      valid_until: parsed.valid_until,
      supersedes: supersedes,
      routed_to: parsed.routed_to,
      workspace_id: workspace_id || "default",
      metadata: parse_json_map(metadata_json),
      score: nil
    }
  end

  # Parse the common (non-signal) fields from a row.
  # Accepts a keyword list to avoid the arity-15 credo violation.
  defp parse_common_fields(fields) do
    %{
      id: fields[:id],
      path: fields[:path],
      title: fields[:title] || "",
      content: fields[:content] || "",
      l0_abstract: fields[:l0_abstract] || "",
      l1_overview: fields[:l1_overview] || "",
      node: fields[:node] || "inbox",
      sn_ratio: fields[:sn_ratio] || 0.5,
      entities: parse_json_list(fields[:entities_json]),
      created_at: parse_dt(fields[:created_at]),
      modified_at: parse_dt(fields[:modified_at]),
      valid_from: parse_dt(fields[:valid_from]),
      valid_until: parse_dt(fields[:valid_until]),
      supersedes: fields[:supersedes],
      routed_to: parse_json_list(fields[:routed_to_json])
    }
  end

  defp build_signal_from_row(p, mode, genre, signal_type, format, structure) do
    %Signal{
      id: p.id,
      path: p.path,
      title: p.title,
      mode: parse_atom(mode, :linguistic),
      genre: genre || "note",
      type: parse_atom(signal_type, :inform),
      format: parse_atom(format, :markdown),
      structure: structure || "",
      created_at: p.created_at,
      modified_at: p.modified_at,
      valid_from: p.valid_from,
      valid_until: p.valid_until,
      supersedes: p.supersedes,
      node: p.node,
      sn_ratio: p.sn_ratio,
      entities: p.entities,
      l0_summary: p.l0_abstract,
      l1_description: p.l1_overview,
      content: p.content,
      routed_to: p.routed_to,
      score: nil
    }
  end

  @doc """
  Converts a Context to a Signal struct for backward compatibility.
  Returns `nil` if the context has no signal classification.
  """
  @spec to_signal(t()) :: Signal.t() | nil
  def to_signal(%__MODULE__{type: :signal, signal: %Signal{} = sig}), do: sig

  def to_signal(%__MODULE__{} = ctx) do
    # Build a minimal Signal from context fields for non-signal types
    %Signal{
      id: ctx.id,
      path: ctx.path,
      title: ctx.title,
      mode: :linguistic,
      genre: "note",
      type: :inform,
      format: :markdown,
      structure: "",
      created_at: ctx.created_at,
      modified_at: ctx.modified_at,
      valid_from: ctx.valid_from,
      valid_until: ctx.valid_until,
      supersedes: ctx.supersedes,
      node: ctx.node || "inbox",
      sn_ratio: ctx.sn_ratio,
      entities: ctx.entities,
      l0_summary: ctx.l0_abstract,
      l1_description: ctx.l1_overview,
      content: ctx.content,
      routed_to: ctx.routed_to,
      score: ctx.score
    }
  end

  @doc """
  Builds a Context from a Signal struct (for migration / compatibility).
  Sets type to :signal and populates the signal field.
  """
  @spec from_signal(Signal.t()) :: t()
  def from_signal(%Signal{} = sig) do
    uri = "optimal://nodes/#{sig.node || "inbox"}/#{Path.basename(sig.path || "")}"

    %__MODULE__{
      id: sig.id,
      uri: uri,
      type: :signal,
      path: sig.path,
      title: sig.title,
      content: sig.content || "",
      l0_abstract: sig.l0_summary || "",
      l1_overview: sig.l1_description || "",
      signal: sig,
      node: sig.node,
      sn_ratio: sig.sn_ratio || 0.5,
      entities: sig.entities || [],
      created_at: sig.created_at,
      modified_at: sig.modified_at,
      valid_from: sig.valid_from,
      valid_until: sig.valid_until,
      supersedes: sig.supersedes,
      routed_to: sig.routed_to || [],
      metadata: %{},
      score: sig.score
    }
  end

  # --- Private helpers ---

  defp serialize_dt(nil), do: nil
  defp serialize_dt(%DateTime{} = dt), do: DateTime.to_iso8601(dt)

  defp parse_dt(nil), do: nil
  defp parse_dt(""), do: nil

  defp parse_dt(str) when is_binary(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _} -> dt
      _ -> nil
    end
  end

  defp parse_context_type("resource"), do: :resource
  defp parse_context_type("memory"), do: :memory
  defp parse_context_type("skill"), do: :skill
  defp parse_context_type("signal"), do: :signal
  defp parse_context_type(_), do: :resource

  defp parse_atom(nil, default), do: default
  defp parse_atom("", default), do: default

  defp parse_atom(str, _default) when is_binary(str) do
    String.to_existing_atom(str)
  rescue
    ArgumentError -> String.to_atom(str)
  end

  defp parse_json_list(nil), do: []
  defp parse_json_list(""), do: []

  defp parse_json_list(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, list} when is_list(list) -> list
      _ -> []
    end
  end

  defp parse_json_map(nil), do: %{}
  defp parse_json_map(""), do: %{}

  defp parse_json_map(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, map} when is_map(map) -> map
      _ -> %{}
    end
  end
end
