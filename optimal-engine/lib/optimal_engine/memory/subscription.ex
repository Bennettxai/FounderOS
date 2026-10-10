defmodule OptimalEngine.Memory.Subscription do
  @moduledoc """
  Subscription describes what an agent (or human principal) wants pushed
  to them proactively. The Surfacer GenServer matches incoming events
  against active subscriptions and pushes envelopes through whatever
  channels the subscriber is connected on (SSE today; webhook later).

  Scopes:
    * `:workspace` — anything in this workspace
    * `:node`      — only signals/pages anchored to a specific node
    * `:topic`     — entity-based (`scope_value` = entity name or wiki slug)
    * `:audience`  — only audience-aware variants for that audience

  Categories follow Engramme's "Questions in the Wild" taxonomy
  (March 2026), reframed for enterprise. Stored as a JSON list so the
  taxonomy can grow without a migration.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Workspace

  @allowed_scopes [:workspace, :node, :topic, :audience]

  # Enterprise-relevant subset of Engramme's 18 categories.
  # Skipped: passwords, own_account_ids, household_objects, personal_task_knowledge.
  @categories [
    :recent_actions,
    :autobiographical_past,
    :contacts,
    :schedules,
    :ownership,
    :open_tasks,
    :tip_of_tongue,
    :professional_knowledge,
    :file_locations,
    :procedures,
    :event_locations,
    :factual,
    :contradictions,
    :unassigned
  ]

  @type scope :: :workspace | :node | :topic | :audience
  @type category :: atom()

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          principal_id: String.t() | nil,
          scope: scope(),
          scope_value: String.t() | nil,
          categories: [category()],
          activity: String.t() | nil,
          status: :active | :paused,
          created_at: String.t() | nil,
          metadata: map()
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            workspace_id: Workspace.default_id(),
            principal_id: nil,
            scope: :workspace,
            scope_value: nil,
            categories: [],
            activity: nil,
            status: :active,
            created_at: nil,
            metadata: %{}

  @doc "All categories the surfacer recognizes."
  @spec categories() :: [category()]
  def categories, do: @categories

  # ── CRUD ───────────────────────────────────────────────────────────────

  @doc """
  Creates a subscription. Required: `workspace_id`. All else optional.
  Default scope is `:workspace` (everything in the workspace surfaces);
  default categories are all of them.
  """
  @spec create(map()) :: {:ok, t()} | {:error, term()}
  def create(attrs) when is_map(attrs) do
    workspace_id = Map.get(attrs, :workspace_id, Workspace.default_id())
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    principal_id = Map.get(attrs, :principal_id)
    scope = Map.get(attrs, :scope, :workspace)
    scope_value = Map.get(attrs, :scope_value)
    categories = Map.get(attrs, :categories, @categories)
    activity = Map.get(attrs, :activity)
    metadata = Map.get(attrs, :metadata, %{})
    id = Map.get(attrs, :id) || derive_id(workspace_id, principal_id, scope, scope_value)

    cond do
      scope not in @allowed_scopes ->
        {:error, {:invalid_scope, scope}}

      not Enum.all?(categories, &(&1 in @categories)) ->
        {:error, {:invalid_category, Enum.find(categories, &(&1 not in @categories))}}

      true ->
        sql = """
        INSERT INTO surfacing_subscriptions (
          id, tenant_id, workspace_id, principal_id, scope, scope_value,
          categories, activity, status, metadata
        ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, 'active', ?9)
        ON CONFLICT(id) DO UPDATE SET
          principal_id = excluded.principal_id,
          scope = excluded.scope,
          scope_value = excluded.scope_value,
          categories = excluded.categories,
          activity = excluded.activity,
          status = 'active',
          metadata = excluded.metadata
        """

        params = [
          id,
          tenant_id,
          workspace_id,
          principal_id,
          Atom.to_string(scope),
          scope_value,
          encode_categories(categories),
          activity,
          Jason.encode!(metadata)
        ]

        case Store.raw_query(sql, params) do
          {:ok, _} ->
            {:ok,
             %__MODULE__{
               id: id,
               tenant_id: tenant_id,
               workspace_id: workspace_id,
               principal_id: principal_id,
               scope: scope,
               scope_value: scope_value,
               categories: categories,
               activity: activity,
               status: :active,
               metadata: metadata
             }}

          other ->
            other
        end
    end
  end

  @doc """
  Lists subscriptions for a workspace.

  Options:
    - `:workspace_id` — defaults to the default workspace
    - `:status` — `:active` (default) | `:paused` | `:all`
    - `:limit`  — max rows (default 50)
    - `:offset` — row offset for pagination (default 0)
  """
  @spec list(keyword()) :: {:ok, [t()]} | {:error, term()}
  def list(opts \\ []) do
    workspace_id = Keyword.get(opts, :workspace_id, Workspace.default_id())
    status = Keyword.get(opts, :status, :active)
    limit = Keyword.get(opts, :limit, 50)
    offset = Keyword.get(opts, :offset, 0)

    {sql, params} =
      case status do
        :all ->
          {"SELECT id, tenant_id, workspace_id, principal_id, scope, scope_value, categories, activity, status, created_at, metadata FROM surfacing_subscriptions WHERE workspace_id = ?1 ORDER BY created_at DESC LIMIT ?2 OFFSET ?3",
           [workspace_id, limit, offset]}

        :active ->
          {"SELECT id, tenant_id, workspace_id, principal_id, scope, scope_value, categories, activity, status, created_at, metadata FROM surfacing_subscriptions WHERE workspace_id = ?1 AND status = 'active' ORDER BY created_at DESC LIMIT ?2 OFFSET ?3",
           [workspace_id, limit, offset]}

        :paused ->
          {"SELECT id, tenant_id, workspace_id, principal_id, scope, scope_value, categories, activity, status, created_at, metadata FROM surfacing_subscriptions WHERE workspace_id = ?1 AND status = 'paused' ORDER BY created_at DESC LIMIT ?2 OFFSET ?3",
           [workspace_id, limit, offset]}
      end

    case Store.raw_query(sql, params) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  end

  @doc """
  Counts subscriptions for a workspace. Accepts the same `:workspace_id` and `:status` opts as `list/1`.
  """
  @spec count(keyword()) :: {:ok, non_neg_integer()} | {:error, term()}
  def count(opts \\ []) do
    workspace_id = Keyword.get(opts, :workspace_id, Workspace.default_id())
    status = Keyword.get(opts, :status, :active)

    {sql, params} =
      case status do
        :all ->
          {"SELECT COUNT(*) FROM surfacing_subscriptions WHERE workspace_id = ?1", [workspace_id]}

        :active ->
          {"SELECT COUNT(*) FROM surfacing_subscriptions WHERE workspace_id = ?1 AND status = 'active'",
           [workspace_id]}

        :paused ->
          {"SELECT COUNT(*) FROM surfacing_subscriptions WHERE workspace_id = ?1 AND status = 'paused'",
           [workspace_id]}
      end

    case Store.raw_query(sql, params) do
      {:ok, [[n]]} -> {:ok, n}
      {:ok, []} -> {:ok, 0}
      other -> other
    end
  end

  @doc "Lists ALL active subscriptions across workspaces — used by Surfacer."
  @spec list_all_active() :: [t()]
  def list_all_active do
    case Store.raw_query(
           "SELECT id, tenant_id, workspace_id, principal_id, scope, scope_value, categories, activity, status, created_at, metadata FROM surfacing_subscriptions WHERE status = 'active'",
           []
         ) do
      {:ok, rows} -> Enum.map(rows, &row_to_struct/1)
      _ -> []
    end
  end

  @doc "Pauses a subscription."
  @spec pause(String.t()) :: :ok | {:error, term()}
  def pause(id) when is_binary(id) do
    case Store.raw_query(
           "UPDATE surfacing_subscriptions SET status = 'paused', paused_at = datetime('now') WHERE id = ?1",
           [id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc "Re-activates a paused subscription."
  @spec resume(String.t()) :: :ok | {:error, term()}
  def resume(id) when is_binary(id) do
    case Store.raw_query(
           "UPDATE surfacing_subscriptions SET status = 'active', paused_at = NULL WHERE id = ?1",
           [id]
         ) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  @doc "Deletes a subscription."
  @spec delete(String.t()) :: :ok | {:error, term()}
  def delete(id) when is_binary(id) do
    case Store.raw_query("DELETE FROM surfacing_subscriptions WHERE id = ?1", [id]) do
      {:ok, _} -> :ok
      other -> other
    end
  end

  # ── Helpers ─────────────────────────────────────────────────────────────

  defp derive_id(workspace_id, principal_id, scope, scope_value) do
    bits = [workspace_id, principal_id || "anon", Atom.to_string(scope), scope_value || "*"]
    "sub:" <> Enum.join(bits, ":")
  end

  defp encode_categories(cats) do
    cats |> Enum.map(&Atom.to_string/1) |> Jason.encode!()
  end

  defp decode_categories(json) do
    case Jason.decode(json) do
      {:ok, list} when is_list(list) -> Enum.map(list, &String.to_atom/1)
      _ -> []
    end
  end

  defp row_to_struct([
         id,
         tenant_id,
         workspace_id,
         principal_id,
         scope,
         scope_value,
         cats,
         activity,
         status,
         created_at,
         meta
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      principal_id: principal_id,
      scope: String.to_atom(scope),
      scope_value: scope_value,
      categories: decode_categories(cats),
      activity: activity,
      status: String.to_atom(status),
      created_at: created_at,
      metadata: decode_json(meta)
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
