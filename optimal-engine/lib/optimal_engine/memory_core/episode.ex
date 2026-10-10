defmodule OptimalEngine.MemoryCore.Episode do
  @moduledoc """
  An Episode records a discrete event that occurred within a workspace — the
  durable episodic-memory primitive for the Memory Core.

  Episodes are created automatically by the intake pipeline whenever a source
  of type `transcript` or `meeting` is ingested. They are also the target for
  explicit episodic recording from other parts of the system (agent loops,
  connector pulls, user sessions).

  ## Fields

  - `id`           — `epi_<sha256>` content-addressed ID
  - `tenant_id`    — owning tenant
  - `workspace_id` — owning workspace
  - `node_id`      — optional: the workspace node this episode belongs to
  - `kind`         — episode kind (`transcript`, `meeting`, `event`, `intake`, etc.)
  - `occurred_at`  — when the event actually happened (valid time)
  - `summary`      — human-readable single-sentence summary
  - `provenance`   — JSON map linking back to the source objects
    (e.g. `%{"source_package_id" => "...", "signal_id" => "..."}`)
  - standard governance envelope: `tenant_id`, `workspace_id`, `security_labels`,
    `partition_ids`, `lifecycle_state`, `metadata`, `created_at`

  Construction is pure (`new/1`); persistence belongs to
  `OptimalEngine.MemoryCore.Store.insert_episode/1`.
  """

  alias OptimalEngine.MemoryCore.ID

  @type t :: %__MODULE__{}

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :node_id,
    :kind,
    :occurred_at,
    :summary,
    :provenance,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :metadata,
    :created_at
  ]

  @doc """
  Build an Episode struct from an attribute map.

  Requires `:kind` and `:summary`. All other fields have sensible defaults.
  Returns `{:ok, t()}` or `{:error, reason}`.
  """
  @spec new(map()) :: {:ok, t()} | {:error, term()}
  def new(%__MODULE__{} = ep), do: {:ok, ep}

  def new(attrs) when is_map(attrs) do
    kind = string_or_nil(Map.get(attrs, :kind))
    summary = string_or_nil(Map.get(attrs, :summary))

    cond do
      is_nil(kind) -> {:error, :kind_required}
      is_nil(summary) -> {:error, :summary_required}
      true -> {:ok, build(attrs, kind, summary)}
    end
  end

  defp build(attrs, kind, summary) do
    tenant_id = Map.get(attrs, :tenant_id, "default")
    workspace_id = Map.get(attrs, :workspace_id, "default")
    occurred_at = Map.get(attrs, :occurred_at) || DateTime.utc_now() |> DateTime.to_iso8601()

    %__MODULE__{
      id:
        Map.get(attrs, :id) ||
          ID.content_id("epi", [
            tenant_id,
            ":",
            workspace_id,
            ":",
            kind,
            ":",
            occurred_at,
            ":",
            summary
          ]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      node_id: string_or_nil(Map.get(attrs, :node_id)),
      kind: kind,
      occurred_at: occurred_at,
      summary: summary,
      provenance: Map.get(attrs, :provenance) || %{},
      security_labels: Map.get(attrs, :security_labels) || [],
      partition_ids: Map.get(attrs, :partition_ids) || [],
      lifecycle_state: Map.get(attrs, :lifecycle_state, "recorded"),
      metadata: Map.get(attrs, :metadata) || %{},
      created_at: Map.get(attrs, :created_at) || DateTime.utc_now() |> DateTime.to_iso8601()
    }
  end

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)
end
