defmodule OptimalEngine.Identity.Group do
  @moduledoc """
  Groups are named principal collections. Typically populated via SCIM from
  an IdP (Okta / Azure AD / Google Workspace) or created manually.

  Roles and ACLs can be granted to groups as shorthand for "every principal
  in this group."
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type source :: :local | :scim | :manual

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          name: String.t(),
          source: source(),
          created_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            name: nil,
            source: :local,
            created_at: nil

  @doc "Upsert a group. Idempotent on `id`."
  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(%{id: id, name: name} = attrs) when is_binary(id) and is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    source = Map.get(attrs, :source, :local)

    sql = """
    INSERT INTO groups (id, tenant_id, name, source)
    VALUES (?1, ?2, ?3, ?4)
    ON CONFLICT(id) DO UPDATE SET
      name   = excluded.name,
      source = excluded.source
    """

    case Store.raw_query(sql, [id, tenant_id, name, Atom.to_string(source)]) do
      {:ok, _} ->
        {:ok, %__MODULE__{id: id, tenant_id: tenant_id, name: name, source: source}}

      other ->
        other
    end
  end

  @doc "Add a principal to a group."
  @spec add_member(String.t(), String.t()) :: :ok | {:error, term()}
  def add_member(principal_id, group_id)
      when is_binary(principal_id) and is_binary(group_id) do
    case Store.raw_query(
           """
           INSERT OR IGNORE INTO principal_groups (principal_id, group_id)
           VALUES (?1, ?2)
           """,
           [principal_id, group_id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc "Remove a principal from a group."
  @spec remove_member(String.t(), String.t()) :: :ok | {:error, term()}
  def remove_member(principal_id, group_id)
      when is_binary(principal_id) and is_binary(group_id) do
    case Store.raw_query(
           "DELETE FROM principal_groups WHERE principal_id = ?1 AND group_id = ?2",
           [principal_id, group_id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end
end
