defmodule OptimalEngine.Identity.ACL do
  @moduledoc """
  Access Control List primitive.

  An ACL row grants a permission (`:read | :write | :admin`) on a resource
  (identified by URI) to either a principal OR a group — never both.

  ACLs propagate by **intersection**: a wiki page's visibility is the
  intersection of visibilities of all cited chunks. This is enforced at
  curation time (Phase 7) and at retrieval time (Phase 8).

  ## Resource URI conventions

  - `"optimal://nodes/project-platform-launch/signals/2026-04-17-ed-call.md"`
  - `"chunk:{chunk_id}"` — per-chunk ACL
  - `"wiki:{slug}:{audience}"` — per-wiki-page ACL
  - `"tenant:{tenant_id}"` — tenant-wide grant
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type permission :: :read | :write | :admin

  @type t :: %__MODULE__{
          id: integer() | nil,
          tenant_id: String.t(),
          resource_uri: String.t(),
          principal_id: String.t() | nil,
          group_id: String.t() | nil,
          permission: permission(),
          granted_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            resource_uri: nil,
            principal_id: nil,
            group_id: nil,
            permission: :read,
            granted_at: nil

  @doc """
  Grant a permission on a resource to a principal or group. Exactly one of
  `principal_id` / `group_id` must be set.
  """
  @spec grant(map()) :: :ok | {:error, term()}
  def grant(%{resource_uri: uri, permission: perm} = attrs)
      when is_binary(uri) and perm in [:read, :write, :admin] do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    principal_id = Map.get(attrs, :principal_id)
    group_id = Map.get(attrs, :group_id)

    cond do
      is_binary(principal_id) and is_nil(group_id) ->
        do_insert(tenant_id, uri, principal_id, nil, perm)

      is_nil(principal_id) and is_binary(group_id) ->
        do_insert(tenant_id, uri, nil, group_id, perm)

      true ->
        {:error, :must_set_exactly_one_of_principal_or_group}
    end
  end

  defp do_insert(tenant_id, uri, principal_id, group_id, perm) do
    sql = """
    INSERT INTO acls (tenant_id, resource_uri, principal_id, group_id, permission)
    VALUES (?1, ?2, ?3, ?4, ?5)
    """

    case Store.raw_query(sql, [tenant_id, uri, principal_id, group_id, Atom.to_string(perm)]) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc """
  Returns `true` if the principal can `permission` the given resource URI.

  A principal `can?` a resource when **any** of the following is true:
    * an ACL grants it directly to the principal
    * an ACL grants it to any group the principal belongs to
    * NO ACL exists for the resource at all (permissive default for Phase 1)

  Phase 8 will tighten the default from permissive to deny-if-absent when the
  wiki-first retrieval path is fully wired. For now, the permissive default
  lets the 689 pre-Phase-1 tests continue to pass.
  """
  @spec can?(String.t(), String.t(), permission(), keyword()) :: boolean()
  def can?(principal_id, resource_uri, permission, opts \\ [])
      when is_binary(principal_id) and is_binary(resource_uri) and
             permission in [:read, :write, :admin] do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())

    case list_for_resource(tenant_id, resource_uri) do
      [] ->
        true

      acls ->
        {:ok, group_ids} = OptimalEngine.Identity.Principal.groups(principal_id)
        group_set = MapSet.new(group_ids)
        perm_str = Atom.to_string(permission)

        Enum.any?(acls, fn
          %{principal_id: ^principal_id, permission: ^perm_str} ->
            true

          %{group_id: gid, permission: ^perm_str} when is_binary(gid) ->
            MapSet.member?(group_set, gid)

          _ ->
            false
        end)
    end
  end

  @doc "Returns the list of ACL rows for a resource within a tenant."
  @spec list_for_resource(String.t(), String.t()) :: [map()]
  def list_for_resource(tenant_id, resource_uri)
      when is_binary(tenant_id) and is_binary(resource_uri) do
    case Store.raw_query(
           """
           SELECT principal_id, group_id, permission
           FROM acls
           WHERE tenant_id = ?1 AND resource_uri = ?2
           """,
           [tenant_id, resource_uri]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [principal_id, group_id, permission] ->
          %{principal_id: principal_id, group_id: group_id, permission: permission}
        end)

      _ ->
        []
    end
  end
end
