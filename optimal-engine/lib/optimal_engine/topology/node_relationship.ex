defmodule OptimalEngine.Topology.NodeRelationship do
  @moduledoc """
  Workspace-scoped topology relationship between two Nodes.

  These are structure relationships for the workspace itself: parent-like
  containment, dependency, ownership, reference, collaboration, and similar
  links. They are separate from Memory Core relationship edges, which describe
  evidentiary, semantic, temporal, causal, and procedural memory relationships.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Topology.Node

  @type relationship_type ::
          :parent_child
          | :depends_on
          | :blocks
          | :references
          | :collaborates_with
          | :sequence_before
          | :instance_of
          | :owns
          | :participates_in
          | :supports

  @allowed_relationships [
    :parent_child,
    :depends_on,
    :blocks,
    :references,
    :collaborates_with,
    :sequence_before,
    :instance_of,
    :owns,
    :participates_in,
    :supports
  ]

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          source_node_id: String.t(),
          target_node_id: String.t(),
          relationship_type: relationship_type(),
          direction: String.t(),
          strength: float(),
          valid_time_start: String.t() | nil,
          valid_time_end: String.t() | nil,
          lifecycle_state: String.t(),
          metadata: map(),
          created_by: String.t() | nil,
          created_at: String.t() | nil,
          updated_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            workspace_id: "default",
            source_node_id: nil,
            target_node_id: nil,
            relationship_type: :references,
            direction: "directed",
            strength: 1.0,
            valid_time_start: nil,
            valid_time_end: nil,
            lifecycle_state: "active",
            metadata: %{},
            created_by: nil,
            created_at: nil,
            updated_at: nil

  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(
        %{
          source_node_id: source_node_id,
          target_node_id: target_node_id,
          relationship_type: relationship_type
        } = attrs
      )
      when is_binary(source_node_id) and is_binary(target_node_id) and
             relationship_type in @allowed_relationships do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    relationship_type_string = Atom.to_string(relationship_type)

    id =
      Map.get(attrs, :id) ||
        deterministic_id(workspace_id, source_node_id, target_node_id, relationship_type_string)

    direction = Map.get(attrs, :direction, "directed")
    strength = Map.get(attrs, :strength, 1.0)
    valid_time_start = Map.get(attrs, :valid_time_start)
    valid_time_end = Map.get(attrs, :valid_time_end)
    lifecycle_state = Map.get(attrs, :lifecycle_state, "active")
    metadata = Map.get(attrs, :metadata, %{})
    created_by = Map.get(attrs, :created_by)

    sql = """
    INSERT INTO node_relationships (
      id, tenant_id, workspace_id, source_node_id, target_node_id,
      relationship_type, direction, strength, valid_time_start, valid_time_end,
      lifecycle_state, metadata, created_by
    ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13)
    ON CONFLICT(workspace_id, source_node_id, target_node_id, relationship_type)
    DO UPDATE SET
      direction = excluded.direction,
      strength = excluded.strength,
      valid_time_start = excluded.valid_time_start,
      valid_time_end = excluded.valid_time_end,
      lifecycle_state = excluded.lifecycle_state,
      metadata = excluded.metadata,
      updated_at = datetime('now')
    """

    with :ok <- validate_node_scope(source_node_id, tenant_id, workspace_id, :source),
         :ok <- validate_node_scope(target_node_id, tenant_id, workspace_id, :target),
         :ok <-
           Store.raw_execute(sql, [
             id,
             tenant_id,
             workspace_id,
             source_node_id,
             target_node_id,
             relationship_type_string,
             direction,
             strength,
             valid_time_start,
             valid_time_end,
             lifecycle_state,
             Jason.encode!(metadata),
             created_by
           ]) do
      get(id, tenant_id: tenant_id, workspace_id: workspace_id)
    end
  end

  def upsert(%{relationship_type: relationship_type}) do
    {:error, {:invalid_relationship_type, relationship_type}}
  end

  @spec get(String.t(), keyword()) :: {:ok, t()} | {:error, :not_found}
  def get(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM node_relationships WHERE id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @spec for_node(String.t(), keyword()) :: {:ok, [t()]} | {:error, term()}
  def for_node(node_id, opts \\ []) when is_binary(node_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    direction = Keyword.get(opts, :direction, :both)

    {where, params} =
      case direction do
        :outgoing ->
          {["tenant_id = ?1", "workspace_id = ?2", "source_node_id = ?3"],
           [tenant_id, workspace_id, node_id]}

        :incoming ->
          {["tenant_id = ?1", "workspace_id = ?2", "target_node_id = ?3"],
           [tenant_id, workspace_id, node_id]}

        :both ->
          {["tenant_id = ?1", "workspace_id = ?2", "(source_node_id = ?3 OR target_node_id = ?3)"],
           [tenant_id, workspace_id, node_id]}
      end

    sql =
      "SELECT " <>
        select_columns() <>
        " FROM node_relationships WHERE " <> Enum.join(where, " AND ") <> " ORDER BY created_at"

    case Store.raw_query(sql, params) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  end

  defp deterministic_id(workspace_id, source_node_id, target_node_id, relationship_type) do
    hash =
      :crypto.hash(
        :sha256,
        "#{workspace_id}|#{source_node_id}|#{target_node_id}|#{relationship_type}"
      )
      |> Base.encode16(case: :lower)
      |> binary_part(0, 24)

    "node_rel_#{hash}"
  end

  defp validate_node_scope(node_id, tenant_id, workspace_id, side) do
    case Node.get(node_id, tenant_id: tenant_id, workspace_id: workspace_id) do
      {:ok, _node} -> :ok
      {:error, :not_found} -> {:error, {:node_not_found_in_workspace, side, node_id, workspace_id}}
      other -> other
    end
  end

  defp select_columns do
    "id, tenant_id, workspace_id, source_node_id, target_node_id, relationship_type, direction, strength, valid_time_start, valid_time_end, lifecycle_state, metadata, created_by, created_at, updated_at"
  end

  defp row_to_struct([
         id,
         tenant_id,
         workspace_id,
         source_node_id,
         target_node_id,
         relationship_type,
         direction,
         strength,
         valid_time_start,
         valid_time_end,
         lifecycle_state,
         metadata,
         created_by,
         created_at,
         updated_at
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      source_node_id: source_node_id,
      target_node_id: target_node_id,
      relationship_type: safe_atom(relationship_type, :references),
      direction: direction,
      strength: strength || 1.0,
      valid_time_start: valid_time_start,
      valid_time_end: valid_time_end,
      lifecycle_state: lifecycle_state,
      metadata: decode_json(metadata),
      created_by: created_by,
      created_at: created_at,
      updated_at: updated_at
    }
  end

  defp workspace_clause(nil, _position), do: ""
  defp workspace_clause(_workspace_id, position), do: " AND workspace_id = ?#{position}"

  defp scoped_params(params, nil), do: params
  defp scoped_params(params, workspace_id), do: params ++ [workspace_id]

  defp safe_atom(nil, fallback), do: fallback

  defp safe_atom(value, fallback) when is_binary(value) do
    try do
      String.to_existing_atom(value)
    rescue
      ArgumentError -> fallback
    end
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end
end
