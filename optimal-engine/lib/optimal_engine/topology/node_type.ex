defmodule OptimalEngine.Topology.NodeType do
  @moduledoc """
  Workspace-scoped Node Type records.

  A Node Type gives a Node durable meaning inside a workspace. Examples:
  project, person, product, operation, context. Workspaces can use the seeded
  standard types or add custom types without changing the core Node table.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          slug: String.t(),
          name: String.t(),
          category: String.t(),
          description: String.t() | nil,
          metadata: map(),
          lifecycle_state: String.t(),
          created_by: String.t() | nil,
          created_at: String.t() | nil,
          updated_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            workspace_id: "default",
            slug: nil,
            name: nil,
            category: "custom",
            description: nil,
            metadata: %{},
            lifecycle_state: "active",
            created_by: nil,
            created_at: nil,
            updated_at: nil

  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(%{slug: slug, name: name} = attrs) when is_binary(slug) and is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    id = Map.get(attrs, :id) || "#{workspace_id}:#{slug}"
    category = Map.get(attrs, :category, "custom")
    description = Map.get(attrs, :description)
    metadata = Map.get(attrs, :metadata, %{})
    lifecycle_state = Map.get(attrs, :lifecycle_state, "active")
    created_by = Map.get(attrs, :created_by)

    sql = """
    INSERT INTO node_types (
      id, tenant_id, workspace_id, slug, name, category, description,
      metadata, lifecycle_state, created_by
    ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10)
    ON CONFLICT(id) DO UPDATE SET
      name = excluded.name,
      category = excluded.category,
      description = excluded.description,
      metadata = excluded.metadata,
      lifecycle_state = excluded.lifecycle_state,
      updated_at = datetime('now')
    """

    case Store.raw_execute(sql, [
           id,
           tenant_id,
           workspace_id,
           slug,
           name,
           category,
           description,
           Jason.encode!(metadata),
           lifecycle_state,
           created_by
         ]) do
      :ok ->
        get(id, tenant_id: tenant_id, workspace_id: workspace_id)

      other ->
        other
    end
  end

  @spec get(String.t(), keyword()) :: {:ok, t()} | {:error, :not_found}
  def get(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM node_types WHERE id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @spec get_by_slug(String.t(), keyword()) :: {:ok, t()} | {:error, :not_found}
  def get_by_slug(slug, opts \\ []) when is_binary(slug) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM node_types WHERE slug = ?1 AND tenant_id = ?2 AND workspace_id = ?3",
           [slug, tenant_id, workspace_id]
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @spec list(keyword()) :: {:ok, [t()]} | {:error, term()}
  def list(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM node_types WHERE tenant_id = ?1 AND workspace_id = ?2 ORDER BY slug",
           [tenant_id, workspace_id]
         ) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  end

  defp select_columns do
    "id, tenant_id, workspace_id, slug, name, category, description, metadata, lifecycle_state, created_by, created_at, updated_at"
  end

  defp row_to_struct([
         id,
         tenant_id,
         workspace_id,
         slug,
         name,
         category,
         description,
         metadata,
         lifecycle_state,
         created_by,
         created_at,
         updated_at
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      slug: slug,
      name: name,
      category: category,
      description: description,
      metadata: decode_json(metadata),
      lifecycle_state: lifecycle_state,
      created_by: created_by,
      created_at: created_at,
      updated_at: updated_at
    }
  end

  defp workspace_clause(nil, _position), do: ""
  defp workspace_clause(_workspace_id, position), do: " AND workspace_id = ?#{position}"

  defp scoped_params(params, nil), do: params
  defp scoped_params(params, workspace_id), do: params ++ [workspace_id]

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end
end
