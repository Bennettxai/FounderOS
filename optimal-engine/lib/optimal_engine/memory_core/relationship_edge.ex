defmodule OptimalEngine.MemoryCore.RelationshipEdge do
  @moduledoc """
  Typed graph link between governed Memory Core objects.

  Fields map one-to-one onto the `relationship_edges` table. Edge IDs are
  content-derived from `(workspace, from, to, relationship_type)` so the same
  logical link is idempotent per workspace. Relationship types in use:
  `"supports"`, `"derived_from"`, and `"supersedes"`.
  """

  alias OptimalEngine.MemoryCore.ID

  @type t :: %__MODULE__{
          id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          from_object_type: String.t() | nil,
          from_object_id: String.t() | nil,
          to_object_type: String.t() | nil,
          to_object_id: String.t() | nil,
          relationship_type: String.t() | nil,
          confidence: float(),
          precision_score: float(),
          evidence_links: [map()],
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          lifecycle_state: String.t(),
          valid_time_start: String.t() | nil,
          valid_time_end: String.t() | nil,
          transaction_time_start: String.t() | nil,
          transaction_time_end: String.t() | nil,
          metadata: map(),
          created_at: String.t() | nil
        }

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :from_object_type,
    :from_object_id,
    :to_object_type,
    :to_object_id,
    :relationship_type,
    :confidence,
    :precision_score,
    :evidence_links,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :valid_time_start,
    :valid_time_end,
    :transaction_time_start,
    :transaction_time_end,
    :metadata,
    :created_at
  ]

  @doc """
  Build a RelationshipEdge struct from an attribute map (atom keys).
  """
  @spec new(map()) :: t()
  def new(%__MODULE__{} = edge), do: edge

  def new(attrs) when is_map(attrs) do
    %__MODULE__{
      id: Map.get(attrs, :id),
      tenant_id: Map.get(attrs, :tenant_id, "default"),
      workspace_id: Map.get(attrs, :workspace_id, "default"),
      from_object_type: Map.get(attrs, :from_object_type),
      from_object_id: Map.get(attrs, :from_object_id),
      to_object_type: Map.get(attrs, :to_object_type),
      to_object_id: Map.get(attrs, :to_object_id),
      relationship_type: Map.get(attrs, :relationship_type),
      confidence: Map.get(attrs, :confidence) || 0.5,
      precision_score: Map.get(attrs, :precision_score) || 0.5,
      evidence_links: Map.get(attrs, :evidence_links) || [],
      access_policy_id: Map.get(attrs, :access_policy_id),
      security_labels: Map.get(attrs, :security_labels) || [],
      partition_ids: Map.get(attrs, :partition_ids) || [],
      lifecycle_state: Map.get(attrs, :lifecycle_state, "current"),
      valid_time_start: Map.get(attrs, :valid_time_start),
      valid_time_end: Map.get(attrs, :valid_time_end),
      transaction_time_start: Map.get(attrs, :transaction_time_start),
      transaction_time_end: Map.get(attrs, :transaction_time_end),
      metadata: Map.get(attrs, :metadata) || %{},
      created_at: Map.get(attrs, :created_at)
    }
  end

  @doc """
  Build a typed edge between two governed objects.

  `scope` supplies tenancy and security inheritance (any map or struct with
  `workspace_id`, `tenant_id`, and optional `access_policy_id`,
  `security_labels`, `partition_ids`).
  """
  @spec between(map(), {String.t(), String.t()}, {String.t(), String.t()}, String.t(), keyword()) ::
          t()
  def between(scope, {from_type, from_id}, {to_type, to_id}, relationship_type, opts \\ []) do
    new(%{
      id:
        ID.content_id("edge", [
          Map.get(scope, :workspace_id, "default"),
          ":",
          from_type,
          ":",
          from_id,
          ":",
          to_type,
          ":",
          to_id,
          ":",
          relationship_type
        ]),
      tenant_id: Map.get(scope, :tenant_id, "default"),
      workspace_id: Map.get(scope, :workspace_id, "default"),
      from_object_type: from_type,
      from_object_id: from_id,
      to_object_type: to_type,
      to_object_id: to_id,
      relationship_type: relationship_type,
      confidence: Keyword.get(opts, :confidence, 0.5),
      precision_score: Keyword.get(opts, :precision_score, 0.5),
      evidence_links: Keyword.get(opts, :evidence_links, []),
      access_policy_id: Map.get(scope, :access_policy_id),
      security_labels: Map.get(scope, :security_labels) || [],
      partition_ids: Map.get(scope, :partition_ids) || [],
      lifecycle_state: "current",
      transaction_time_start: timestamp(),
      metadata: Keyword.get(opts, :metadata, %{})
    })
  end

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()
end
