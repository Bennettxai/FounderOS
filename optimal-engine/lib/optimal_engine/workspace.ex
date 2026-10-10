defmodule OptimalEngine.Workspace do
  @moduledoc """
  A **workspace** is a knowledge base owned by an organization inside a tenant.

  An organization can hold many workspaces — e.g. "Engineering Brain",
  "Sales Brain", "M&A Brain", "Personal". Each workspace has its own
  curated wiki, its own connectors, its own audiences, and its own
  set of typed hierarchical nodes scoped by `node.workspace_id`.

  Multiplicity rules:

  - Every signal-bearing row carries a `workspace_id`.
  - Default deployments use the singleton workspace `"default"`
    seeded by migration 026.
  - A principal in a tenant can belong to multiple workspaces via
    `workspace_members` with role ∈ `:owner | :member | :viewer`.

  Tenant isolation is the absolute boundary; workspace is a soft scope
  inside it. There's no cross-workspace leakage by default — every
  query that scopes by workspace must pass `workspace_id`.

  This module is the facade over the `workspaces` and `workspace_members`
  tables created in migration 026.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Organization
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Workspace.Config
  alias OptimalEngine.Workspace.Filesystem

  @default_id "default"
  @allowed_statuses [:active, :archived]
  @allowed_roles [:owner, :member, :viewer]

  @type status :: :active | :archived
  @type role :: :owner | :member | :viewer

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          organization_id: String.t(),
          slug: String.t(),
          name: String.t(),
          description: String.t() | nil,
          status: status(),
          created_at: String.t() | nil,
          archived_at: String.t() | nil,
          metadata: map()
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            organization_id: Organization.default_id(),
            slug: nil,
            name: nil,
            description: nil,
            status: :active,
            created_at: nil,
            archived_at: nil,
            metadata: %{}

  # ── Defaults ────────────────────────────────────────────────────────────

  @doc "Returns the reserved default workspace id."
  @spec default_id() :: String.t()
  def default_id, do: @default_id

  # ── Lookup ──────────────────────────────────────────────────────────────

  @doc "Returns the workspace with the given id, or `{:error, :not_found}`."
  @spec get(String.t()) :: {:ok, t()} | {:error, :not_found}
  def get(id) when is_binary(id) do
    case Store.raw_query(
           select_sql() <> " WHERE id = ?1",
           [id]
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Returns the workspace identified by `(tenant_id, slug)`."
  @spec get_by_slug(String.t(), String.t()) :: {:ok, t()} | {:error, :not_found}
  def get_by_slug(slug, tenant_id) when is_binary(slug) and is_binary(tenant_id) do
    case Store.raw_query(
           select_sql() <> " WHERE tenant_id = ?1 AND slug = ?2",
           [tenant_id, slug]
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  Lists workspaces in a tenant. Options:
    * `:status`    — `:active` (default) | `:archived` | `:all`
    * `:tenant_id` — defaults to the default tenant
    * `:limit`     — max rows (default 50)
    * `:offset`    — row offset for pagination (default 0)
  """
  @spec list(keyword()) :: {:ok, [t()]} | {:error, term()}
  def list(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    organization_id = Keyword.get(opts, :organization_id)
    status = Keyword.get(opts, :status, :active)
    limit = Keyword.get(opts, :limit, 50)
    offset = Keyword.get(opts, :offset, 0)

    {where, params} = list_scope(tenant_id, organization_id, status)

    sql =
      select_sql() <>
        where <> " ORDER BY name LIMIT ?#{length(params) + 1} OFFSET ?#{length(params) + 2}"

    params = params ++ [limit, offset]

    case Store.raw_query(sql, params) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  catch
    {:invalid_status, s} -> {:error, {:invalid_status, s}}
  end

  @doc """
  Counts workspaces in a tenant. Accepts the same `:tenant_id` and `:status` opts as `list/1`.
  """
  @spec count(keyword()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    organization_id = Keyword.get(opts, :organization_id)
    status = Keyword.get(opts, :status, :active)

    {where, params} = list_scope(tenant_id, organization_id, status)
    sql = "SELECT COUNT(*) FROM workspaces" <> where

    case Store.raw_query(sql, params) do
      {:ok, [[n]]} -> {:ok, n}
      {:ok, []} -> {:ok, 0}
      other -> other
    end
  catch
    {:invalid_status, s} -> {:error, {:invalid_status, s}}
  end

  # ── Mutations ───────────────────────────────────────────────────────────

  @doc """
  Creates a workspace. Required: `slug`, `name`. Optional: `tenant_id`
  (defaults to default tenant), `description`, `metadata`. Slug becomes
  part of the id (`tenant_id:slug` for non-default tenants, plain
  `default` for the singleton default-tenant default-workspace case).
  """
  @spec create(map()) :: {:ok, t()} | {:error, term()}
  def create(%{slug: slug, name: name} = attrs) when is_binary(slug) and is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    organization_id = Map.get(attrs, :organization_id, Organization.default_id())
    description = Map.get(attrs, :description)
    metadata = Map.get(attrs, :metadata, %{})
    id = Map.get(attrs, :id) || derive_id(tenant_id, slug)

    sql = """
    INSERT INTO workspaces (id, tenant_id, organization_id, slug, name, description, status, metadata)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, 'active', ?7)
    """

    params = [id, tenant_id, organization_id, slug, name, description, Jason.encode!(metadata)]

    with {:ok, organization} <- Organization.get(organization_id),
         true <- organization.tenant_id == tenant_id || {:error, :organization_tenant_mismatch},
         {:ok, _} <- Store.raw_query(sql, params) do
      root = Application.get_env(:optimal_engine, :root_path, File.cwd!())

      case Filesystem.provision(slug, root: root) do
        {:ok, _path} ->
          case Config.put(slug, Config.defaults(), root) do
            :ok ->
              :ok

            {:error, reason} ->
              require Logger

              Logger.warning(
                "[Workspace.create] config write failed for #{slug}: #{inspect(reason)}"
              )
          end

        {:error, reason} ->
          require Logger
          Logger.warning("[Workspace.create] FS provision failed for #{slug}: #{inspect(reason)}")
      end

      {:ok,
       %__MODULE__{
         id: id,
         tenant_id: tenant_id,
         organization_id: organization_id,
         slug: slug,
         name: name,
         description: description,
         status: :active,
         metadata: metadata
       }}
    else
      false -> {:error, :organization_tenant_mismatch}
      other -> other
    end
  end

  def create(_), do: {:error, :missing_required_fields}

  @doc """
  Updates a workspace's mutable fields: `:name`, `:description`,
  `:metadata`. Slug + tenant + id are immutable.
  """
  @spec update(String.t(), map()) :: {:ok, t()} | {:error, term()}
  def update(id, attrs) when is_binary(id) and is_map(attrs) do
    with {:ok, current} <- get(id) do
      name = Map.get(attrs, :name, current.name)
      description = Map.get(attrs, :description, current.description)
      metadata = Map.get(attrs, :metadata, current.metadata)

      case Store.raw_query(
             "UPDATE workspaces SET name = ?1, description = ?2, metadata = ?3 WHERE id = ?4",
             [name, description, Jason.encode!(metadata), id]
           ) do
        {:ok, _} ->
          {:ok, %{current | name: name, description: description, metadata: metadata}}

        other ->
          other
      end
    end
  end

  @doc "Moves a workspace to another organization in the same tenant."
  @spec assign_organization(String.t(), String.t()) :: {:ok, t()} | {:error, term()}
  def assign_organization(id, organization_id) when is_binary(id) and is_binary(organization_id) do
    with {:ok, workspace} <- get(id),
         {:ok, organization} <- Organization.get(organization_id),
         true <-
           organization.tenant_id == workspace.tenant_id || {:error, :organization_tenant_mismatch},
         {:ok, _} <-
           Store.raw_query("UPDATE workspaces SET organization_id = ?1 WHERE id = ?2", [
             organization_id,
             id
           ]) do
      get(id)
    else
      false -> {:error, :organization_tenant_mismatch}
      other -> other
    end
  end

  @doc "Soft-deletes a workspace by setting `status = 'archived'`."
  @spec archive(String.t()) :: :ok | {:error, term()}
  def archive(id) when is_binary(id) do
    case Store.raw_query(
           "UPDATE workspaces SET status = 'archived', archived_at = datetime('now') WHERE id = ?1",
           [id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  # ── Membership ──────────────────────────────────────────────────────────

  @doc """
  Grants a principal access to a workspace at a role. Role defaults to
  `:member`.
  """
  @spec add_member(String.t(), String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def add_member(workspace_id, principal_id, opts \\ []) do
    role = Keyword.get(opts, :role, :member)
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())

    if role not in @allowed_roles do
      {:error, {:invalid_role, role}}
    else
      sql = """
      INSERT INTO workspace_members (tenant_id, workspace_id, principal_id, role)
      VALUES (?1, ?2, ?3, ?4)
      ON CONFLICT(workspace_id, principal_id) DO UPDATE SET
        role = excluded.role,
        ended_at = NULL
      """

      case Store.raw_query(sql, [tenant_id, workspace_id, principal_id, Atom.to_string(role)]) do
        {:ok, _} ->
          {:ok,
           %{
             tenant_id: tenant_id,
             workspace_id: workspace_id,
             principal_id: principal_id,
             role: role
           }}

        other ->
          other
      end
    end
  end

  @doc "Removes a principal from a workspace (sets ended_at)."
  @spec remove_member(String.t(), String.t()) :: :ok | {:error, term()}
  def remove_member(workspace_id, principal_id) do
    case Store.raw_query(
           "UPDATE workspace_members SET ended_at = datetime('now') WHERE workspace_id = ?1 AND principal_id = ?2",
           [workspace_id, principal_id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc "Lists active members of a workspace as `{principal_id, role}` tuples."
  @spec members_of(String.t()) :: {:ok, [{String.t(), atom()}]} | {:error, term()}
  def members_of(workspace_id) when is_binary(workspace_id) do
    case Store.raw_query(
           "SELECT principal_id, role FROM workspace_members WHERE workspace_id = ?1 AND ended_at IS NULL ORDER BY role, principal_id",
           [workspace_id]
         ) do
      {:ok, rows} ->
        {:ok, Enum.map(rows, fn [pid, role] -> {pid, String.to_atom(role)} end)}

      other ->
        other
    end
  end

  @doc """
  Lists the workspaces a principal can access (active, not-ended) as
  `{workspace, role}` tuples, ordered by name.
  """
  @spec workspaces_of(String.t()) :: {:ok, [{t(), atom()}]} | {:error, term()}
  def workspaces_of(principal_id) when is_binary(principal_id) do
    sql = """
    SELECT w.id, w.tenant_id, w.organization_id, w.slug, w.name, w.description, w.status,
           w.created_at, w.archived_at, w.metadata, m.role
    FROM workspaces w
    INNER JOIN workspace_members m ON m.workspace_id = w.id
    WHERE m.principal_id = ?1 AND m.ended_at IS NULL AND w.status = 'active'
    ORDER BY w.name
    """

    case Store.raw_query(sql, [principal_id]) do
      {:ok, rows} ->
        {:ok,
         Enum.map(rows, fn row ->
           {ws_row, [role]} = Enum.split(row, 10)
           {row_to_struct(ws_row), String.to_atom(role)}
         end)}

      other ->
        other
    end
  end

  # ── Helpers ─────────────────────────────────────────────────────────────

  # The default-tenant default-workspace gets the bare id "default" for
  # backwards compat with rows backfilled in migration 026. Everything
  # else gets `<tenant_id>:<slug>`.
  defp derive_id(tenant_id, slug) do
    if tenant_id == Tenant.default_id() and slug == @default_id,
      do: @default_id,
      else: "#{tenant_id}:#{slug}"
  end

  defp select_sql do
    "SELECT id, tenant_id, organization_id, slug, name, description, status, created_at, archived_at, metadata FROM workspaces"
  end

  defp list_scope(tenant_id, organization_id, status) do
    {clauses, params} =
      if organization_id,
        do: {["tenant_id = ?1", "organization_id = ?2"], [tenant_id, organization_id]},
        else: {["tenant_id = ?1"], [tenant_id]}

    case status do
      :all ->
        {" WHERE " <> Enum.join(clauses, " AND "), params}

      s when s in @allowed_statuses ->
        position = length(params) + 1

        {" WHERE " <> Enum.join(clauses ++ ["status = ?#{position}"], " AND "),
         params ++ [Atom.to_string(s)]}

      other ->
        throw({:invalid_status, other})
    end
  end

  defp row_to_struct([
         id,
         tenant_id,
         organization_id,
         slug,
         name,
         description,
         status,
         created_at,
         archived_at,
         meta_json
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      organization_id: organization_id,
      slug: slug,
      name: name,
      description: description,
      status: String.to_atom(status),
      created_at: created_at,
      archived_at: archived_at,
      metadata: decode_json(meta_json)
    }
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(s) when is_binary(s) do
    case Jason.decode(s) do
      {:ok, m} when is_map(m) -> m
      _ -> %{}
    end
  end
end
