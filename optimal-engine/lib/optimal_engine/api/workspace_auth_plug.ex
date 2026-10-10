defmodule OptimalEngine.API.WorkspaceAuthPlug do
  @moduledoc """
  Workspace authorization for the HTTP API.

  Runs AFTER `OptimalEngine.API.AuthPlug` (which authenticates the caller and
  assigns `:current_tenant` / `:current_api_key`) and BEFORE `:dispatch`, so
  no workspace-parameterized handler can run for a workspace the caller is
  not allowed to read.

  For every request it:

  1. Resolves EVERY workspace id the request names — the `:id` path segment
     on `/api/workspaces/:id*`, the `"workspace"` body field, and the
     `"workspace"` query parameter. Authorizing all of them (not just the one
     a given handler happens to read) means a body/query mismatch can never
     smuggle a second, unauthorized workspace past the check. When a
     workspace-scoped path names none, the implicit `"default"` workspace is
     authorized instead — preserving single-workspace behavior while keeping
     it tenant-checked.
  2. Enforces, for each requested workspace id:
     * **tenant ownership** — a workspace registered to another tenant (or an
       unregistered `"<tenant>:<slug>"` id with a foreign tenant prefix)
       returns **404** `{"error":"workspace not found"}`. Fails closed:
       foreign workspaces are indistinguishable from absent ones.
     * **the API key's `workspace_scope`** — the id must be listed or the
       scope must contain `"*"`, otherwise **403**
       `{"error":"workspace_scope_denied"}`. Anonymous mode
       (`auth_required: false`, no key) has no key scope to enforce.
  3. Assigns the primary requested workspace (path > body > query >
     `"default"`) to `conn.assigns[:workspace_id]` — closing the loop with
     `AuthPlug`'s documented workspace-scope contract, which reads that
     assign.

  Unregistered workspace ids without a tenant prefix (single-tenant
  deployments using ad hoc workspace ids) pass the tenant check for
  compatibility; per-row `workspace_id` filtering in the handlers still
  scopes the data, and keyed callers remain bound by `workspace_scope`.
  """

  @behaviour Plug

  import Plug.Conn

  alias OptimalEngine.Workspace

  # Endpoint families that serve workspace-scoped data. Requests under these
  # paths are authorized even when no workspace is named (implicit "default").
  # Paths outside these families are only checked when they explicitly name a
  # workspace.
  @workspace_scoped_roots ~w(activity batch graph grep health memory node
                             optimal profile rag recall render search signals
                             subscriptions wiki workspace workspaces)

  @impl true
  def init(opts), do: opts

  @impl true
  def call(conn, _opts) do
    conn = fetch_query_params(conn)
    requested = requested_workspaces(conn)

    cond do
      requested == [] and not workspace_scoped_path?(conn) ->
        conn

      Enum.any?(requested, &(not is_binary(&1))) ->
        halt_json(conn, 400, "invalid workspace")

      true ->
        authorize_all(conn, if(requested == [], do: ["default"], else: requested))
    end
  end

  # ---------------------------------------------------------------------------
  # Requested-workspace resolution
  # ---------------------------------------------------------------------------

  defp requested_workspaces(conn) do
    [path_workspace(conn), body_workspace(conn), query_workspace(conn)]
    |> Enum.reject(&(&1 in [nil, ""]))
    |> Enum.uniq()
  end

  # /api/workspaces/:id, /api/workspaces/:id/config, /api/workspaces/:id/archive…
  defp path_workspace(%{path_info: ["api", "workspaces", id | _]}), do: URI.decode(id)

  defp path_workspace(%{path_info: ["api", "storage", "workspaces", id | _]}),
    do: URI.decode(id)

  defp path_workspace(_conn), do: nil

  defp body_workspace(%{body_params: %Plug.Conn.Unfetched{}}), do: nil
  defp body_workspace(%{body_params: params}) when is_map(params), do: Map.get(params, "workspace")
  defp body_workspace(_conn), do: nil

  defp query_workspace(conn), do: Map.get(conn.query_params, "workspace")

  # GET/POST /api/workspaces (list/create) are tenant-parameterized, not
  # workspace-parameterized — only the /:id subroutes name a workspace, and
  # those are caught by path_workspace/1 above.
  defp workspace_scoped_path?(%{path_info: ["api", "workspaces"]}), do: false
  defp workspace_scoped_path?(%{path_info: ["api", root | _]}), do: root in @workspace_scoped_roots
  defp workspace_scoped_path?(_conn), do: false

  # ---------------------------------------------------------------------------
  # Authorization
  # ---------------------------------------------------------------------------

  defp authorize_all(conn, [primary | _] = workspace_ids) do
    tenant_id = conn.assigns[:current_tenant] || "default"
    api_key = conn.assigns[:current_api_key]
    conn = assign(conn, :workspace_id, primary)

    Enum.reduce_while(workspace_ids, conn, fn workspace_id, conn ->
      case authorize(workspace_id, tenant_id, api_key) do
        :ok -> {:cont, conn}
        {:error, :foreign_tenant} -> {:halt, halt_json(conn, 404, "workspace not found")}
        {:error, :scope_denied} -> {:halt, halt_json(conn, 403, "workspace_scope_denied")}
      end
    end)
  end

  # Tenant ownership first: a foreign tenant's workspace is always 404 —
  # indistinguishable from a workspace that does not exist — regardless of the
  # key's workspace_scope. Scope mismatches inside the caller's own tenant
  # are 403.
  defp authorize(workspace_id, tenant_id, api_key) do
    with :ok <- check_tenant_ownership(workspace_id, tenant_id) do
      check_key_scope(workspace_id, api_key)
    end
  end

  defp check_tenant_ownership(workspace_id, tenant_id) do
    case Workspace.get(workspace_id) do
      {:ok, workspace} ->
        if workspace.tenant_id == tenant_id, do: :ok, else: {:error, :foreign_tenant}

      {:error, :not_found} ->
        # Unregistered id. The "<tenant>:<slug>" id convention
        # (Workspace.derive_id) still names an owner — fail closed on a
        # foreign prefix. Bare unregistered ids pass for single-tenant
        # compatibility (handlers still workspace-filter every row).
        case String.split(workspace_id, ":", parts: 2) do
          [prefix, _slug] when prefix != tenant_id -> {:error, :foreign_tenant}
          _ -> :ok
        end

      _other ->
        # Store unavailable — fail closed.
        {:error, :foreign_tenant}
    end
  end

  defp check_key_scope(_workspace_id, nil), do: :ok

  defp check_key_scope(workspace_id, api_key) do
    scope = api_key.workspace_scope || []

    if "*" in scope or workspace_id in scope do
      :ok
    else
      {:error, :scope_denied}
    end
  end

  # ---------------------------------------------------------------------------
  # Error responses
  # ---------------------------------------------------------------------------

  defp halt_json(conn, status, error) do
    conn
    |> put_resp_content_type("application/json")
    |> send_resp(status, Jason.encode!(%{error: error}))
    |> halt()
  end
end
