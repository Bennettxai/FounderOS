defmodule OptimalEngine.Organization do
  @moduledoc """
  An operating or ownership entity inside a tenant.

  Tenants enforce hard data isolation. Organizations own workspaces inside
  that boundary, allowing one Optimal Engine account to model multiple legal
  entities, ventures, and personal operating scopes.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @default_id "default"
  @allowed_statuses [:active, :dormant, :archived]

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          slug: String.t(),
          name: String.t(),
          description: String.t() | nil,
          status: atom(),
          created_at: String.t() | nil,
          archived_at: String.t() | nil,
          metadata: map()
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            slug: nil,
            name: nil,
            description: nil,
            status: :active,
            created_at: nil,
            archived_at: nil,
            metadata: %{}

  def default_id, do: @default_id

  @spec create(map()) :: {:ok, t()} | {:error, term()}
  def create(%{slug: slug, name: name} = attrs) when is_binary(slug) and is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    id = Map.get(attrs, :id) || derive_id(tenant_id, slug)
    description = Map.get(attrs, :description)
    status = Map.get(attrs, :status, :active)
    metadata = Map.get(attrs, :metadata, %{})

    if status in @allowed_statuses do
      case Store.raw_query(
             "INSERT INTO organizations (id, tenant_id, slug, name, description, status, metadata) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
             [
               id,
               tenant_id,
               slug,
               name,
               description,
               Atom.to_string(status),
               Jason.encode!(metadata)
             ]
           ) do
        {:ok, _} -> get(id)
        other -> other
      end
    else
      {:error, {:invalid_status, status}}
    end
  end

  def create(_), do: {:error, :missing_required_fields}

  @spec get(String.t()) :: {:ok, t()} | {:error, :not_found}
  def get(id) when is_binary(id) do
    case Store.raw_query(
           "SELECT id, tenant_id, slug, name, description, status, created_at, archived_at, metadata FROM organizations WHERE id = ?1",
           [id]
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @spec list(keyword()) :: {:ok, [t()]} | {:error, term()}
  def list(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    status = Keyword.get(opts, :status, :active)

    {sql, params} =
      case status do
        :all ->
          {"SELECT id, tenant_id, slug, name, description, status, created_at, archived_at, metadata FROM organizations WHERE tenant_id = ?1 ORDER BY name",
           [tenant_id]}

        value when value in @allowed_statuses ->
          {"SELECT id, tenant_id, slug, name, description, status, created_at, archived_at, metadata FROM organizations WHERE tenant_id = ?1 AND status = ?2 ORDER BY name",
           [tenant_id, Atom.to_string(value)]}

        value ->
          throw({:invalid_status, value})
      end

    case Store.raw_query(sql, params) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  catch
    {:invalid_status, value} -> {:error, {:invalid_status, value}}
  end

  defp derive_id("default", slug), do: slug
  defp derive_id(tenant_id, slug), do: "#{tenant_id}:#{slug}"

  defp row_to_struct([
         id,
         tenant_id,
         slug,
         name,
         description,
         status,
         created_at,
         archived_at,
         metadata
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      slug: slug,
      name: name,
      description: description,
      status: String.to_atom(status),
      created_at: created_at,
      archived_at: archived_at,
      metadata: decode_json(metadata)
    }
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end
end
