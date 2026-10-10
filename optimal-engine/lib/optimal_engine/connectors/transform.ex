defmodule OptimalEngine.Connectors.Transform do
  @moduledoc """
  Shared helpers that turn a connector-specific payload into an
  engine `%Signal{}`.

  Every adapter's `transform/1` is free to do its own mapping, but
  most want the same 4 things:

    * a stable `id` synthesized from `(kind, external_id)`
    * the `source_uri` preserved for citation
    * a `modified_at` / `created_at` parsed from an ISO-8601 string
    * a default `mode` / `genre` / `format` pre-filled before the
      classifier re-stamps them downstream

  This module centralizes those so adapters stay short and uniform.
  """

  alias OptimalEngine.Signal

  @doc """
  Synthesize a deterministic signal id from `(connector_kind,
  external_id)`. Same inputs → same id — so retried syncs don't create
  duplicate rows.
  """
  @spec signal_id(atom(), String.t()) :: String.t()
  def signal_id(kind, external_id) when is_atom(kind) and is_binary(external_id) do
    raw = "#{kind}:#{external_id}"
    hash = :crypto.hash(:sha256, raw) |> Base.encode16(case: :lower) |> String.slice(0, 16)
    "#{kind}:#{hash}"
  end

  @doc """
  Build a skeleton Signal with connector-sane defaults. Adapters
  typically set `title`, `content`, and `node`, then merge additional
  fields specific to the source.
  """
  @spec new_signal(map()) :: Signal.t()
  def new_signal(attrs) when is_map(attrs) do
    now = DateTime.utc_now()

    %Signal{
      id: Map.get(attrs, :id),
      path: Map.get(attrs, :path, ""),
      title: Map.get(attrs, :title, ""),
      mode: Map.get(attrs, :mode, :linguistic),
      genre: Map.get(attrs, :genre, "note"),
      type: Map.get(attrs, :type, :inform),
      format: Map.get(attrs, :format, :markdown),
      structure: Map.get(attrs, :structure, "unstructured"),
      created_at: Map.get(attrs, :created_at, now),
      modified_at: Map.get(attrs, :modified_at, now),
      valid_from: Map.get(attrs, :valid_from),
      valid_until: Map.get(attrs, :valid_until),
      supersedes: Map.get(attrs, :supersedes),
      node: Map.get(attrs, :node, "inbox"),
      sn_ratio: Map.get(attrs, :sn_ratio, 0.5),
      entities: Map.get(attrs, :entities, []),
      l0_summary: Map.get(attrs, :l0_summary, ""),
      l1_description: Map.get(attrs, :l1_description, ""),
      content: Map.get(attrs, :content, ""),
      routed_to: Map.get(attrs, :routed_to, [])
    }
  end

  @doc """
  Parse an ISO-8601 string into a `DateTime`. Returns the given
  fallback (defaulting to the current UTC time) when parsing fails.
  """
  @spec parse_iso8601(String.t() | nil, DateTime.t() | nil) :: DateTime.t()
  def parse_iso8601(iso, fallback \\ nil)
  def parse_iso8601(nil, fallback), do: fallback || DateTime.utc_now()

  def parse_iso8601(iso, fallback) when is_binary(iso) do
    case DateTime.from_iso8601(iso) do
      {:ok, dt, _} -> dt
      _ -> fallback || DateTime.utc_now()
    end
  end

  @doc """
  Best-effort plain-text extraction from HTML or multi-line strings.
  Strips tags, collapses whitespace, trims.
  """
  @spec strip_html(String.t() | nil) :: String.t()
  def strip_html(nil), do: ""

  def strip_html(text) when is_binary(text) do
    text
    |> String.replace(~r/<[^>]+>/, " ")
    |> String.replace(~r/\s+/, " ")
    |> String.trim()
  end

  @doc "Build an `optimal://connectors/<kind>/<external_id>` URI."
  @spec source_uri(atom(), String.t()) :: String.t()
  def source_uri(kind, external_id) when is_atom(kind) and is_binary(external_id) do
    "optimal://connectors/#{kind}/#{external_id}"
  end
end
