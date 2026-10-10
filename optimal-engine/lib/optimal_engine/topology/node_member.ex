defmodule OptimalEngine.Topology.NodeMember do
  @moduledoc """
  Ties a principal to a node with an explicit membership kind.

  A principal can be a member of many nodes; a node has many members.
  Membership is time-bounded (`started_at` + `ended_at`) so historical
  queries ("who was on the sales team on 2026-03-15?") are answerable.

  Membership kinds:

    * `:owner`    — operates the node; can add members, edit metadata
    * `:internal` — employee / contributor inside the company
    * `:external` — client, partner, vendor; outside the company
    * `:observer` — read-only visibility (auditors, AI agents)
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Topology.Node

  @type membership :: :owner | :internal | :external | :observer

  @allowed_memberships [:owner, :internal, :external, :observer]

  @doc """
  Add a principal to a node with a membership kind. Idempotent on
  `(node_id, principal_id, membership)`.

  Options:
    * `:membership` — default `:internal`
    * `:role`       — optional string ("lead", "designer", "CSM", …)
    * `:tenant_id`  — defaults to default tenant
    * `:workspace_id` — defaults to `"default"`
  """
  @spec add(String.t(), String.t(), keyword()) :: :ok | {:error, term()}
  def add(node_id, principal_id, opts \\ [])
      when is_binary(node_id) and is_binary(principal_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    membership = Keyword.get(opts, :membership, :internal)
    role = Keyword.get(opts, :role)

    if membership not in @allowed_memberships do
      {:error, {:invalid_membership, membership}}
    else
      with {:ok, resolved_workspace_id} <- resolve_node_workspace(node_id, tenant_id, workspace_id) do
        sql = """
        INSERT OR IGNORE INTO node_members (
          tenant_id, workspace_id, node_id, principal_id, membership, role
        )
        VALUES (?1, ?2, ?3, ?4, ?5, ?6)
        """

        case Store.raw_query(sql, [
               tenant_id,
               resolved_workspace_id,
               node_id,
               principal_id,
               Atom.to_string(membership),
               role
             ]) do
          {:ok, _} -> :ok
          other -> other
        end
      end
    end
  end

  @doc "Remove (end) a membership. Sets `ended_at` rather than deleting."
  @spec remove(String.t(), String.t(), keyword()) :: :ok | {:error, term()}
  def remove(node_id, principal_id, opts \\ [])
      when is_binary(node_id) and is_binary(principal_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    membership = Keyword.get(opts, :membership)

    {sql, params} =
      remove_query(node_id, principal_id, tenant_id, workspace_id, membership)

    case Store.raw_query(sql, params) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc """
  Returns every active member of the node, optionally filtered by
  membership kind.
  """
  @spec members_of(String.t(), keyword()) :: {:ok, [map()]}
  def members_of(node_id, opts \\ []) when is_binary(node_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    kind_filter = Keyword.get(opts, :membership)

    {where, params} =
      {["node_id = ?1", "tenant_id = ?2", "ended_at IS NULL"], [node_id, tenant_id]}

    {where, params} =
      case workspace_id do
        nil ->
          {where, params}

        workspace_id ->
          {where ++ ["workspace_id = ?#{length(params) + 1}"], params ++ [workspace_id]}
      end

    {where, params} =
      case kind_filter do
        nil ->
          {where, params}

        m when m in @allowed_memberships ->
          {where ++ ["membership = ?#{length(params) + 1}"], params ++ [Atom.to_string(m)]}
      end

    sql = """
    SELECT workspace_id, principal_id, membership, role, started_at, ended_at
    FROM node_members
    WHERE #{Enum.join(where, " AND ")}
    ORDER BY started_at
    """

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        {:ok,
         Enum.map(rows, fn [workspace_id, pid, m, role, started_at, ended_at] ->
           %{
             workspace_id: workspace_id,
             principal_id: pid,
             membership: safe_atom(m, :internal),
             role: role,
             started_at: started_at,
             ended_at: ended_at
           }
         end)}

      other ->
        other
    end
  end

  @doc """
  Returns every active node the principal is a member of, optionally filtered
  by membership kind.
  """
  @spec nodes_of(String.t(), keyword()) :: {:ok, [map()]}
  def nodes_of(principal_id, opts \\ []) when is_binary(principal_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    kind_filter = Keyword.get(opts, :membership)

    {where, params} =
      {["principal_id = ?1", "tenant_id = ?2", "ended_at IS NULL"], [principal_id, tenant_id]}

    {where, params} =
      case workspace_id do
        nil ->
          {where, params}

        workspace_id ->
          {where ++ ["workspace_id = ?#{length(params) + 1}"], params ++ [workspace_id]}
      end

    {where, params} =
      case kind_filter do
        nil ->
          {where, params}

        m when m in @allowed_memberships ->
          {where ++ ["membership = ?#{length(params) + 1}"], params ++ [Atom.to_string(m)]}
      end

    sql = """
    SELECT workspace_id, node_id, membership, role, started_at, ended_at
    FROM node_members
    WHERE #{Enum.join(where, " AND ")}
    ORDER BY started_at
    """

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        {:ok,
         Enum.map(rows, fn [workspace_id, nid, m, role, started_at, ended_at] ->
           %{
             workspace_id: workspace_id,
             node_id: nid,
             membership: safe_atom(m, :internal),
             role: role,
             started_at: started_at,
             ended_at: ended_at
           }
         end)}

      other ->
        other
    end
  end

  defp remove_query(node_id, principal_id, tenant_id, workspace_id, membership) do
    {where, params} =
      {["node_id = ?1", "principal_id = ?2", "tenant_id = ?3", "ended_at IS NULL"],
       [node_id, principal_id, tenant_id]}

    {where, params} =
      case workspace_id do
        nil ->
          {where, params}

        workspace_id ->
          {where ++ ["workspace_id = ?#{length(params) + 1}"], params ++ [workspace_id]}
      end

    {where, params} =
      case membership do
        nil ->
          {where, params}

        m when m in @allowed_memberships ->
          {where ++ ["membership = ?#{length(params) + 1}"], params ++ [Atom.to_string(m)]}
      end

    {"""
     UPDATE node_members
     SET ended_at = datetime('now')
     WHERE #{Enum.join(where, " AND ")}
     """, params}
  end

  defp resolve_node_workspace(node_id, tenant_id, nil) do
    case Node.get(node_id, tenant_id: tenant_id) do
      {:ok, node} -> {:ok, node.workspace_id}
      {:error, :not_found} -> {:error, {:node_not_found, node_id}}
      other -> other
    end
  end

  defp resolve_node_workspace(node_id, tenant_id, workspace_id) do
    case Node.get(node_id, tenant_id: tenant_id, workspace_id: workspace_id) do
      {:ok, node} -> {:ok, node.workspace_id}
      {:error, :not_found} -> {:error, {:node_not_found_in_workspace, node_id, workspace_id}}
      other -> other
    end
  end

  defp safe_atom(nil, fallback), do: fallback

  defp safe_atom(str, fallback) when is_binary(str) do
    try do
      String.to_existing_atom(str)
    rescue
      ArgumentError -> fallback
    end
  end
end
