defmodule OptimalEngine.Topology.Node do
  @moduledoc """
  An organizational unit inside a tenant's workspace — a team, a project, an
  external entity, a domain, a named person, etc.

  Nodes form a tree (`parent_id`). They're the anchor for every signal the
  engine ingests: routing rules point at them, wiki pages are organized by
  them, retrieval boosts chunks rooted in the caller's nodes.

  Before Phase 3.5 these existed as freeform strings on `contexts.node`;
  after Phase 3.5 they're first-class rows with kind + style + path +
  status + versionable metadata.

  See `docs/architecture/WORKSPACE.md` for the full model.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @type kind ::
          :unit
          | :team
          | :project
          | :entity
          | :domain
          | :department
          | :operation
          | :learning
          | :person
          | :product
          | :partnership
          | :context
  @type style :: :internal | :external | :mixed
  @type status :: :active | :archived | :draft

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          slug: String.t(),
          name: String.t(),
          kind: kind(),
          node_type_id: String.t() | nil,
          parent_id: String.t() | nil,
          description: String.t() | nil,
          style: style(),
          status: status(),
          path: String.t(),
          metadata: map(),
          lifecycle_state: String.t(),
          created_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: Tenant.default_id(),
            workspace_id: "default",
            slug: nil,
            name: nil,
            kind: :unit,
            node_type_id: nil,
            parent_id: nil,
            description: nil,
            style: :internal,
            status: :active,
            path: "",
            metadata: %{},
            lifecycle_state: "active",
            created_at: nil

  @allowed_kinds [
    :unit,
    :team,
    :project,
    :entity,
    :domain,
    :department,
    :operation,
    :learning,
    :person,
    :product,
    :partnership,
    :context
  ]
  @allowed_styles [:internal, :external, :mixed]
  @allowed_statuses [:active, :archived, :draft]

  @doc """
  Upsert a node. `slug` + `workspace_id` is the natural key; `id` defaults to
  `"{tenant_id}:{workspace_id}:{slug}"` for a deterministic, readable identifier.
  """
  @spec upsert(map()) :: {:ok, t()} | {:error, term()}
  def upsert(%{slug: slug, name: name, kind: kind} = attrs)
      when is_binary(slug) and is_binary(name) and kind in @allowed_kinds do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    id = Map.get(attrs, :id) || "#{tenant_id}:#{workspace_id}:#{slug}"
    parent_id = Map.get(attrs, :parent_id)
    description = Map.get(attrs, :description)
    style = Map.get(attrs, :style, :internal)
    status = Map.get(attrs, :status, :active)
    kind_string = Atom.to_string(kind)
    node_type_id = Map.get(attrs, :node_type_id, "#{workspace_id}:#{kind_string}")
    path = Map.get(attrs, :path, "nodes/#{slug}")
    metadata = Map.get(attrs, :metadata, %{})
    lifecycle_state = Map.get(attrs, :lifecycle_state, Atom.to_string(status))

    cond do
      style not in @allowed_styles ->
        {:error, {:invalid_style, style}}

      status not in @allowed_statuses ->
        {:error, {:invalid_status, status}}

      true ->
        with :ok <- validate_parent_scope(parent_id, tenant_id, workspace_id) do
          upsert_scoped_node(%{
            id: id,
            tenant_id: tenant_id,
            workspace_id: workspace_id,
            slug: slug,
            name: name,
            kind_string: kind_string,
            kind: kind,
            node_type_id: node_type_id,
            parent_id: parent_id,
            description: description,
            style: style,
            status: status,
            path: path,
            metadata: metadata,
            lifecycle_state: lifecycle_state
          })
        end
    end
  end

  defp upsert_scoped_node(%{
         id: id,
         tenant_id: tenant_id,
         workspace_id: workspace_id,
         slug: slug,
         name: name,
         kind_string: kind_string,
         kind: kind,
         node_type_id: node_type_id,
         parent_id: parent_id,
         description: description,
         style: style,
         status: status,
         path: path,
         metadata: metadata,
         lifecycle_state: lifecycle_state
       }) do
    sql = """
    INSERT INTO nodes (id, tenant_id, workspace_id, slug, name, kind, node_type_id,
                       parent_id, description, style, status, path, metadata,
                       lifecycle_state)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14)
    ON CONFLICT(id) DO UPDATE SET
      workspace_id     = excluded.workspace_id,
      name             = excluded.name,
      kind             = excluded.kind,
      node_type_id     = excluded.node_type_id,
      parent_id        = excluded.parent_id,
      description      = excluded.description,
      style            = excluded.style,
      status           = excluded.status,
      path             = excluded.path,
      metadata         = excluded.metadata,
      lifecycle_state  = excluded.lifecycle_state,
      updated_at       = datetime('now')
    """

    params = [
      id,
      tenant_id,
      workspace_id,
      slug,
      name,
      kind_string,
      node_type_id,
      parent_id,
      description,
      Atom.to_string(style),
      Atom.to_string(status),
      path,
      Jason.encode!(metadata),
      lifecycle_state
    ]

    case Store.raw_query(sql, params) do
      {:ok, _} ->
        {:ok,
         %__MODULE__{
           id: id,
           tenant_id: tenant_id,
           workspace_id: workspace_id,
           slug: slug,
           name: name,
           kind: kind,
           node_type_id: node_type_id,
           parent_id: parent_id,
           description: description,
           style: style,
           status: status,
           path: path,
           metadata: metadata,
           lifecycle_state: lifecycle_state
         }}

      other ->
        other
    end
  end

  @doc "Fetch a node by id, tenant-scoped and optionally workspace-scoped."
  @spec get(String.t(), String.t() | keyword()) :: {:ok, t()} | {:error, :not_found}
  def get(id, opts \\ []) when is_binary(id) do
    {tenant_id, workspace_id} = scope_from_arg(opts)

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM nodes WHERE id = ?1 AND tenant_id = ?2" <> workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Fetch by slug within a workspace."
  @spec get_by_slug(String.t(), String.t() | keyword()) :: {:ok, t()} | {:error, :not_found}
  def get_by_slug(slug, opts \\ []) when is_binary(slug) do
    {tenant_id, workspace_id} = scope_from_arg(opts)

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM nodes WHERE slug = ?1 AND tenant_id = ?2" <> workspace_clause(workspace_id, 3),
           scoped_params([slug, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_struct(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc "Lists all nodes in a tenant, optionally filtered by kind and/or status."
  @spec list(keyword()) :: {:ok, [t()]}
  def list(opts \\ []) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    {clauses, params} = {["tenant_id = ?1"], [tenant_id]}

    {clauses, params} =
      case workspace_id do
        nil ->
          {clauses, params}

        workspace_id ->
          {clauses ++ ["workspace_id = ?#{length(params) + 1}"], params ++ [workspace_id]}
      end

    {clauses, params} =
      case Keyword.get(opts, :kind) do
        nil ->
          {clauses, params}

        k when is_atom(k) ->
          {clauses ++ ["kind = ?#{length(params) + 1}"], params ++ [Atom.to_string(k)]}
      end

    {clauses, params} =
      case Keyword.get(opts, :status) do
        nil ->
          {clauses, params}

        s when is_atom(s) ->
          {clauses ++ ["status = ?#{length(params) + 1}"], params ++ [Atom.to_string(s)]}
      end

    sql =
      "SELECT " <>
        select_columns() <>
        " FROM nodes WHERE " <> Enum.join(clauses, " AND ") <> " ORDER BY slug"

    case Store.raw_query(sql, params) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  end

  @doc "Returns the direct children of a node (one level down)."
  @spec children(String.t(), keyword()) :: {:ok, [t()]}
  def children(node_id, opts \\ []) when is_binary(node_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT " <>
             select_columns() <>
             " FROM nodes WHERE parent_id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3) <> " ORDER BY slug",
           scoped_params([node_id, tenant_id], workspace_id)
         ) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_struct/1)}
      other -> other
    end
  end

  @doc "Walks up the parent chain. Returns ancestors ordered root→self."
  @spec ancestors(String.t(), keyword()) :: {:ok, [t()]}
  def ancestors(node_id, opts \\ []) when is_binary(node_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    do_ancestors(node_id, tenant_id, workspace_id, [])
  end

  @doc """
  Returns a nested tree of Nodes for a tenant/workspace.

  Each tree item is `%{node: %Node{}, children: [...]}`. Root items are Nodes
  with no parent or with a parent that is not present in the current scope.
  """
  @spec tree(keyword()) :: {:ok, [map()]} | {:error, term()}
  def tree(opts \\ []) do
    with {:ok, nodes} <- list(opts) do
      children_by_parent = Enum.group_by(nodes, & &1.parent_id)
      ids = MapSet.new(nodes, & &1.id)

      roots =
        nodes
        |> Enum.filter(fn node ->
          is_nil(node.parent_id) or not MapSet.member?(ids, node.parent_id)
        end)
        |> Enum.sort_by(& &1.slug)

      {:ok, Enum.map(roots, &tree_item(&1, children_by_parent))}
    end
  end

  # Ancestors are built by walking parent pointers up from the starting node.
  # Each recursive call prepends the newly-seen node, so the accumulator
  # naturally ends up in root→self order. Do NOT reverse at the end.
  defp do_ancestors(nil, _tenant_id, _workspace_id, acc), do: {:ok, acc}

  defp do_ancestors(id, tenant_id, workspace_id, acc) do
    case get(id, tenant_id: tenant_id, workspace_id: workspace_id) do
      {:ok, node} -> do_ancestors(node.parent_id, tenant_id, workspace_id, [node | acc])
      {:error, :not_found} -> {:ok, acc}
    end
  end

  defp tree_item(node, children_by_parent) do
    children =
      children_by_parent
      |> Map.get(node.id, [])
      |> Enum.sort_by(& &1.slug)
      |> Enum.map(&tree_item(&1, children_by_parent))

    %{node: node, children: children}
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp validate_parent_scope(nil, _tenant_id, _workspace_id), do: :ok

  defp validate_parent_scope(parent_id, tenant_id, workspace_id) do
    case get(parent_id, tenant_id: tenant_id, workspace_id: workspace_id) do
      {:ok, _parent} -> :ok
      {:error, :not_found} -> {:error, {:parent_not_found_in_workspace, parent_id, workspace_id}}
      other -> other
    end
  end

  defp select_columns,
    do:
      "id, tenant_id, workspace_id, slug, name, kind, node_type_id, parent_id, description, style, status, path, metadata, lifecycle_state, created_at"

  defp row_to_struct([
         id,
         tenant_id,
         workspace_id,
         slug,
         name,
         kind,
         node_type_id,
         parent_id,
         description,
         style,
         status,
         path,
         metadata,
         lifecycle_state,
         created_at
       ]) do
    %__MODULE__{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      slug: slug,
      name: name,
      kind: safe_atom(kind, :unit),
      node_type_id: node_type_id,
      parent_id: parent_id,
      description: description,
      style: safe_atom(style, :internal),
      status: safe_atom(status, :active),
      path: path,
      metadata: decode_json(metadata),
      lifecycle_state: lifecycle_state || Atom.to_string(safe_atom(status, :active)),
      created_at: created_at
    }
  end

  defp scope_from_arg(opts) when is_list(opts),
    do: {Keyword.get(opts, :tenant_id, Tenant.default_id()), Keyword.get(opts, :workspace_id)}

  defp scope_from_arg(tenant_id) when is_binary(tenant_id), do: {tenant_id, nil}

  defp workspace_clause(nil, _position), do: ""
  defp workspace_clause(_workspace_id, position), do: " AND workspace_id = ?#{position}"

  defp scoped_params(params, nil), do: params
  defp scoped_params(params, workspace_id), do: params ++ [workspace_id]

  defp safe_atom(nil, fallback), do: fallback

  defp safe_atom(str, fallback) when is_binary(str) do
    try do
      String.to_existing_atom(str)
    rescue
      ArgumentError -> fallback
    end
  end

  defp decode_json(nil), do: %{}
  defp decode_json(""), do: %{}

  defp decode_json(s) when is_binary(s) do
    case Jason.decode(s) do
      {:ok, m} -> m
      _ -> %{}
    end
  end
end
