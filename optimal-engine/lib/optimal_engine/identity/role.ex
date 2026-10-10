defmodule OptimalEngine.Identity.Role do
  @moduledoc """
  A role is a named capability set within a tenant. Examples: `sales`,
  `legal:hold`, `audit:read`, `engineering`, `exec-brief`.

  Roles are granted to principals (directly) or to groups (transitively).
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          name: String.t(),
          description: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            name: nil,
            description: nil

  @doc "Upsert a role."
  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(%{id: id, name: name} = attrs) when is_binary(id) and is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    description = Map.get(attrs, :description)

    sql = """
    INSERT INTO roles (id, tenant_id, name, description)
    VALUES (?1, ?2, ?3, ?4)
    ON CONFLICT(id) DO UPDATE SET
      name        = excluded.name,
      description = excluded.description
    """

    case Store.raw_query(sql, [id, tenant_id, name, description]) do
      {:ok, _} ->
        {:ok, %__MODULE__{id: id, tenant_id: tenant_id, name: name, description: description}}

      other ->
        other
    end
  end

  @doc """
  Grant a role to a principal or group. Exactly one of `principal_id` /
  `group_id` must be set.
  """
  @spec grant(map()) :: :ok | {:error, term()}
  def grant(%{role_id: role_id} = attrs) when is_binary(role_id) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    principal_id = Map.get(attrs, :principal_id)
    group_id = Map.get(attrs, :group_id)

    cond do
      is_binary(principal_id) and is_nil(group_id) ->
        do_grant(tenant_id, role_id, principal_id: principal_id)

      is_nil(principal_id) and is_binary(group_id) ->
        do_grant(tenant_id, role_id, group_id: group_id)

      true ->
        {:error, :must_set_exactly_one_of_principal_or_group}
    end
  end

  defp do_grant(tenant_id, role_id, principal_id: principal_id) do
    sql = """
    INSERT INTO role_grants (tenant_id, principal_id, role_id)
    VALUES (?1, ?2, ?3)
    """

    case Store.raw_query(sql, [tenant_id, principal_id, role_id]) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  defp do_grant(tenant_id, role_id, group_id: group_id) do
    sql = """
    INSERT INTO role_grants (tenant_id, group_id, role_id)
    VALUES (?1, ?2, ?3)
    """

    case Store.raw_query(sql, [tenant_id, group_id, role_id]) do
      {:ok, _} -> :ok
      other -> other
    end
  end
end
