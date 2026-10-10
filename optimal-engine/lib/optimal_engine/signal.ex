defmodule OptimalEngine.Signal do
  @moduledoc """
  Core data model. Every piece of context in OptimalOS is a Signal.

  Signals are classified on 5 dimensions: S=(Mode, Genre, Type, Format, Structure)
  per Signal Theory (Alice H. Luna, Feb 2026).

  The struct is the canonical representation — it travels between processes,
  is persisted to SQLite, and is returned from searches.
  """

  @type mode :: :linguistic | :visual | :code | :data | :mixed
  @type signal_type :: :direct | :inform | :commit | :decide | :express
  @type format :: :markdown | :code | :json | :yaml | :unknown

  @type t :: %__MODULE__{
          id: String.t(),
          path: String.t(),
          title: String.t(),
          # Mode + Genre + Type + Format + Structure
          mode: mode(),
          genre: String.t(),
          type: signal_type(),
          format: format(),
          structure: String.t(),
          # Temporal
          created_at: DateTime.t(),
          modified_at: DateTime.t(),
          valid_from: DateTime.t() | nil,
          valid_until: DateTime.t() | nil,
          supersedes: String.t() | nil,
          # Classification
          node: String.t(),
          sn_ratio: float(),
          entities: [String.t()],
          # Tiered summaries
          l0_summary: String.t(),
          l1_description: String.t(),
          content: String.t(),
          # Routing
          routed_to: [String.t()],
          # Search score (transient, not persisted)
          score: float() | nil
        }

  defstruct [
    :id,
    :path,
    :title,
    :mode,
    :genre,
    :type,
    :format,
    :structure,
    :created_at,
    :modified_at,
    :valid_from,
    :valid_until,
    :supersedes,
    :node,
    :sn_ratio,
    :entities,
    :l0_summary,
    :l1_description,
    :content,
    :routed_to,
    :score
  ]

  @doc "Returns the list of valid signal modes."
  @spec valid_modes() :: [mode()]
  def valid_modes, do: [:linguistic, :visual, :code, :data, :mixed]

  @doc "Returns the list of valid signal types."
  @spec valid_types() :: [signal_type()]
  def valid_types, do: [:direct, :inform, :commit, :decide, :express]

  @doc "Returns the list of valid signal formats."
  @spec valid_formats() :: [format()]
  def valid_formats, do: [:markdown, :code, :json, :yaml, :unknown]

  @doc """
  Converts a Signal struct to a flat map suitable for SQLite insertion.
  DateTime fields are serialized to ISO8601 strings.
  """
  @spec to_row(t()) :: map()
  def to_row(%__MODULE__{} = s) do
    %{
      id: s.id,
      path: s.path,
      title: s.title,
      mode: to_string(s.mode || :linguistic),
      genre: s.genre || "note",
      type: to_string(s.type || :inform),
      format: to_string(s.format || :markdown),
      structure: s.structure || "",
      created_at: serialize_dt(s.created_at),
      modified_at: serialize_dt(s.modified_at),
      valid_from: serialize_dt(s.valid_from || s.created_at || s.modified_at || DateTime.utc_now()),
      valid_until: serialize_dt(s.valid_until),
      supersedes: s.supersedes,
      node: s.node || "inbox",
      sn_ratio: s.sn_ratio || 0.5,
      entities: Jason.encode!(s.entities || []),
      l0_summary: s.l0_summary || "",
      l1_description: s.l1_description || "",
      content: s.content || "",
      routed_to: Jason.encode!(s.routed_to || [])
    }
  end

  @doc "Reconstructs a Signal from a SQLite row (list of column values)."
  @spec from_row([term()]) :: t()
  def from_row(row) when is_list(row) do
    [
      id,
      path,
      title,
      mode,
      genre,
      type,
      format,
      structure,
      created_at,
      modified_at,
      valid_from,
      valid_until,
      supersedes,
      node,
      sn_ratio,
      entities_json,
      l0_summary,
      l1_description,
      content,
      routed_to_json
    ] = row

    %__MODULE__{
      id: id,
      path: path,
      title: title,
      mode: parse_atom(mode, :linguistic),
      genre: genre,
      type: parse_atom(type, :inform),
      format: parse_atom(format, :markdown),
      structure: structure || "",
      created_at: parse_dt(created_at),
      modified_at: parse_dt(modified_at),
      valid_from: parse_dt(valid_from),
      valid_until: parse_dt(valid_until),
      supersedes: supersedes,
      node: node,
      sn_ratio: sn_ratio || 0.5,
      entities: parse_json_list(entities_json),
      l0_summary: l0_summary || "",
      l1_description: l1_description || "",
      content: content || "",
      routed_to: parse_json_list(routed_to_json),
      score: nil
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
end
