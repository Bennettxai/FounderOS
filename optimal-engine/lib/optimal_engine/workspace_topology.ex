defmodule OptimalEngine.WorkspaceTopology do
  @moduledoc """
  Lifecycle facade for workspace topology.

  This is the entry point callers should use when they are defining the shape
  of a workspace: Nodes, Node Types, and Node Relationships. The lower-level
  modules still own their persistence details, but this facade names the
  workspace operation instead of exposing raw tables.
  """

  alias OptimalEngine.Store
  alias OptimalEngine.Storage.PolicyStore
  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Topology.{Node, NodeMember, NodeRelationship, NodeType}
  alias OptimalEngine.Workspace
  alias OptimalEngine.Workspace.Filesystem, as: WorkspaceFilesystem
  alias OptimalEngine.WorkspaceExport

  @standard_node_types [
    {"entity", "Entity", "Business, organization, institution, or operating entity."},
    {"division", "Division", "Major operating branch inside an organization or workspace."},
    {"department", "Department", "Functional area inside an entity."},
    {"team", "Team", "Group of people or agents working together."},
    {"role", "Role", "Named responsibility or position held by a person or agent."},
    {"project", "Project", "Bounded initiative with a target outcome."},
    {"process", "Process", "Repeatable sequence of work with defined inputs and outputs."},
    {"task", "Task", "Assignable unit of work with a completion state."},
    {"operation", "Operation", "Ongoing process or business function."},
    {"learning", "Learning", "Knowledge acquisition or capability development."},
    {"person", "Person", "Individual human, agent, partner, customer, or stakeholder."},
    {"customer", "Customer", "Client account or customer organization receiving ongoing value."},
    {"product", "Product", "Product, platform, system, or offer."},
    {"partnership", "Partnership",
     "Collaboration, joint venture, or external working relationship."},
    {"context", "Context", "Reference context with a lifecycle and relationships."}
  ]

  @type topology_change_request :: %{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          request_type: String.t(),
          target_object_type: String.t(),
          target_object_id: String.t() | nil,
          proposed_payload: map(),
          reason: String.t() | nil,
          requested_by: String.t() | nil,
          review_status: String.t(),
          lifecycle_state: String.t(),
          audit_event_links: [map()],
          created_at: String.t() | nil,
          updated_at: String.t() | nil
        }

  @type setup_result :: %{
          workspace: Workspace.t(),
          path: String.t(),
          node_types: [NodeType.t()],
          nodes: [Node.t()],
          relationships: [NodeRelationship.t()],
          exports: [map()],
          projections: [map()],
          rhythm_paths: [String.t()],
          agent_sop_path: String.t() | nil
        }

  @doc "Create a workspace and seed the standard Node Types it needs to operate."
  @spec create_workspace(map()) :: {:ok, Workspace.t()} | {:error, term()}
  def create_workspace(attrs) when is_map(attrs) do
    with {:ok, workspace} <- Workspace.create(attrs),
         :ok <- ensure_standard_node_types(workspace),
         {:ok, _policy} <- ensure_storage_policy(workspace, attrs) do
      {:ok, workspace}
    end
  end

  @doc """
  Create or refresh a workspace operating topology.

  This is the onboarding entry point for humans and agent CLIs. It creates the
  workspace if needed, ensures standard Node Types, optionally creates starter
  Nodes, writes the human-operable markdown/rhythm/agent files, and records
  those files as governed projections.
  """
  @spec setup_workspace(map()) :: {:ok, setup_result()} | {:error, term()}
  def setup_workspace(%{slug: slug, name: name} = attrs)
      when is_binary(slug) and is_binary(name) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    root = Map.get(attrs, :root, Application.get_env(:optimal_engine, :root_path, File.cwd!()))
    create_starter_nodes? = Map.get(attrs, :starter_nodes, true)
    write_files? = Map.get(attrs, :write_files, true)
    write_rhythm? = Map.get(attrs, :write_rhythm, true)
    write_agent_sop? = Map.get(attrs, :write_agent_sop, true)

    with {:ok, workspace} <- get_or_create_workspace(attrs),
         {:ok, _policy} <- ensure_storage_policy(workspace, attrs),
         :ok <- ensure_standard_node_types(workspace),
         {:ok, node_types} <- list_node_types(tenant_id: tenant_id, workspace_id: workspace.id),
         {:ok, nodes} <- maybe_create_setup_nodes(workspace, attrs, create_starter_nodes?),
         {:ok, relationships} <- create_setup_relationships(workspace, nodes, attrs),
         {:ok, path} <- WorkspaceFilesystem.provision(workspace.slug, root: root),
         {:ok, projection_result} <-
           maybe_write_setup_files(workspace, path, nodes,
             write_files: write_files?,
             write_rhythm: write_rhythm?,
             write_agent_sop: write_agent_sop?,
             actor_id: Map.get(attrs, :actor_id) || Map.get(attrs, :created_by) || "optimal.setup"
           ) do
      {:ok,
       %{
         workspace: workspace,
         path: path,
         node_types: node_types,
         nodes: nodes,
         relationships: relationships,
         exports: projection_result.exports,
         projections: projection_result.projections,
         rhythm_paths: projection_result.rhythm_paths,
         agent_sop_path: projection_result.agent_sop_path
       }}
    end
  end

  @doc "Fetch a workspace by id."
  @spec get_workspace(String.t()) :: {:ok, Workspace.t()} | {:error, :not_found}
  def get_workspace(id), do: Workspace.get(id)

  defp ensure_storage_policy(workspace, attrs) do
    case PolicyStore.get_policy(workspace.id) do
      {:ok, policy} ->
        {:ok, policy}

      {:error, :not_found} ->
        use_cases = Map.get(attrs, :storage_use_cases, ["desktop_local"])

        PolicyStore.put_policy(workspace.id, use_cases,
          actor_id: Map.get(attrs, :actor_id) || Map.get(attrs, :created_by) || "workspace.create"
        )

      {:error, reason} ->
        {:error, reason}
    end
  end

  @doc "Ensure a workspace has the standard Node Types."
  @spec ensure_standard_node_types(Workspace.t() | map()) :: :ok | {:error, term()}
  def ensure_standard_node_types(%{id: workspace_id, tenant_id: tenant_id}) do
    @standard_node_types
    |> Enum.reduce_while(:ok, fn {slug, name, description}, :ok ->
      case create_node_type(%{
             tenant_id: tenant_id,
             workspace_id: workspace_id,
             slug: slug,
             name: name,
             category: "standard",
             description: description
           }) do
        {:ok, _type} -> {:cont, :ok}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
  end

  @doc "Create or update a Node Type inside a workspace."
  @spec create_node_type(map()) :: {:ok, NodeType.t()} | {:error, term()}
  def create_node_type(attrs), do: NodeType.upsert(attrs)

  @doc "Create or update a Node inside a workspace."
  @spec create_node(map()) :: {:ok, Node.t()} | {:error, term()}
  def create_node(attrs), do: Node.upsert(attrs)

  @doc "Return a nested Node tree for a workspace."
  @spec node_tree(keyword()) :: {:ok, [map()]} | {:error, term()}
  def node_tree(opts \\ []), do: Node.tree(opts)

  @doc "Fetch a Node Type by slug within a workspace."
  @spec get_node_type(String.t(), keyword()) :: {:ok, NodeType.t()} | {:error, :not_found}
  def get_node_type(slug, opts \\ []), do: NodeType.get_by_slug(slug, opts)

  @doc "List Node Types in a workspace."
  @spec list_node_types(keyword()) :: {:ok, [NodeType.t()]} | {:error, term()}
  def list_node_types(opts \\ []), do: NodeType.list(opts)

  @doc "Create or update a typed relationship between two Nodes."
  @spec link_nodes(String.t(), String.t(), NodeRelationship.relationship_type(), keyword()) ::
          {:ok, NodeRelationship.t()} | {:error, term()}
  def link_nodes(source_node_id, target_node_id, relationship_type, opts \\ [])
      when is_binary(source_node_id) and is_binary(target_node_id) and is_atom(relationship_type) do
    opts
    |> Map.new()
    |> Map.merge(%{
      source_node_id: source_node_id,
      target_node_id: target_node_id,
      relationship_type: relationship_type
    })
    |> NodeRelationship.upsert()
  end

  @doc "Return topology relationships touching a Node."
  @spec relationships_for_node(String.t(), keyword()) ::
          {:ok, [NodeRelationship.t()]} | {:error, term()}
  def relationships_for_node(node_id, opts \\ []), do: NodeRelationship.for_node(node_id, opts)

  @doc "Attach a human, agent, team, service account, or tool principal to a Node."
  @spec add_node_member(String.t(), String.t(), keyword()) :: :ok | {:error, term()}
  def add_node_member(node_id, principal_id, opts \\ []) do
    NodeMember.add(node_id, principal_id, opts)
  end

  @doc "List active members of a Node."
  @spec node_members(String.t(), keyword()) :: {:ok, [map()]} | {:error, term()}
  def node_members(node_id, opts \\ []) do
    NodeMember.members_of(node_id, opts)
  end

  @doc "List active Nodes for a principal."
  @spec nodes_for_principal(String.t(), keyword()) :: {:ok, [map()]} | {:error, term()}
  def nodes_for_principal(principal_id, opts \\ []) do
    NodeMember.nodes_of(principal_id, opts)
  end

  @doc "Return the standard starter Nodes used by `setup_workspace/1`."
  @spec starter_nodes() :: [map()]
  def starter_nodes do
    [
      %{
        slug: "inbox",
        name: "Inbox",
        kind: :context,
        description: "Default intake area for unsorted sources, notes, and pending routing."
      },
      %{
        slug: "first-project",
        name: "First Project",
        kind: :project,
        description: "Starter project Node. Rename or replace with the first active initiative."
      },
      %{
        slug: "operating-rhythm",
        name: "Operating Rhythm",
        kind: :operation,
        description: "Daily, weekly, and monthly cadence for keeping the workspace current."
      },
      %{
        slug: "primary-user",
        name: "Primary User",
        kind: :person,
        description: "The first human/operator represented in this workspace."
      },
      %{
        slug: "knowledge-base",
        name: "Knowledge Base",
        kind: :learning,
        description: "Research, references, frameworks, and reusable learning context."
      }
    ]
  end

  @doc """
  Record a proposed topology mutation that still needs review.

  This is the write path for agent-suggested or projection-edit-suggested
  workspace structure changes. It creates a pending request instead of silently
  mutating durable topology.
  """
  @spec propose_change(map()) :: {:ok, topology_change_request()} | {:error, term()}
  def propose_change(%{request_type: request_type, target_object_type: target_object_type} = attrs)
      when is_binary(request_type) and is_binary(target_object_type) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    target_object_id = Map.get(attrs, :target_object_id)
    proposed_payload = Map.get(attrs, :proposed_payload, %{})
    reason = Map.get(attrs, :reason)
    requested_by = Map.get(attrs, :requested_by) || Map.get(attrs, :actor_id)
    review_status = Map.get(attrs, :review_status, "pending")
    lifecycle_state = Map.get(attrs, :lifecycle_state, "open")
    audit_event_links = Map.get(attrs, :audit_event_links, [])

    id =
      Map.get(attrs, :id) ||
        deterministic_id("topology_change", [
          workspace_id,
          request_type,
          target_object_type,
          target_object_id || "",
          Jason.encode!(proposed_payload),
          Integer.to_string(System.unique_integer([:positive]))
        ])

    sql = """
    INSERT INTO topology_change_requests (
      id, tenant_id, workspace_id, request_type, target_object_type,
      target_object_id, proposed_payload, reason, requested_by, review_status,
      lifecycle_state, audit_event_links
    ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
    """

    with :ok <-
           Store.raw_execute(sql, [
             id,
             tenant_id,
             workspace_id,
             request_type,
             target_object_type,
             target_object_id,
             Jason.encode!(proposed_payload),
             reason,
             requested_by,
             review_status,
             lifecycle_state,
             Jason.encode!(audit_event_links)
           ]) do
      get_change_request(id, tenant_id: tenant_id, workspace_id: workspace_id)
    end
  end

  @doc "Fetch a topology change request by id."
  @spec get_change_request(String.t(), keyword()) ::
          {:ok, topology_change_request()} | {:error, :not_found}
  def get_change_request(id, opts \\ []) when is_binary(id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)

    case Store.raw_query(
           "SELECT id, tenant_id, workspace_id, request_type, target_object_type, target_object_id, proposed_payload, reason, requested_by, review_status, lifecycle_state, audit_event_links, created_at, updated_at FROM topology_change_requests WHERE id = ?1 AND tenant_id = ?2" <>
             workspace_clause(workspace_id, 3),
           scoped_params([id, tenant_id], workspace_id)
         ) do
      {:ok, [row]} -> {:ok, row_to_change_request(row)}
      {:ok, []} -> {:error, :not_found}
      other -> other
    end
  end

  @doc """
  Review a pending topology change.

  `decision` may be `:approve` or `:reject`. By default, approval records the
  review decision only. Pass `apply: true` to apply supported request types:

    * `create_node`
    * `link_nodes`
    * `add_node_member` / `change_node_owner`

  Unsupported request types remain reviewed but unapplied.
  """
  @spec review_change_request(String.t(), :approve | :reject, keyword()) ::
          {:ok, %{request: topology_change_request(), applied: term() | nil}} | {:error, term()}
  def review_change_request(id, decision, opts \\ [])
      when is_binary(id) and decision in [:approve, :reject] do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    reviewed_by = Keyword.get(opts, :reviewed_by) || Keyword.get(opts, :actor_id)
    apply? = Keyword.get(opts, :apply, false)
    review_status = if decision == :approve, do: "approved", else: "rejected"
    lifecycle_state = if decision == :approve, do: "reviewed", else: "closed"

    with {:ok, request} <- get_change_request(id, tenant_id: tenant_id, workspace_id: workspace_id),
         {:ok, applied} <- maybe_apply_change(request, decision, apply?),
         :ok <-
           Store.raw_execute(
             "UPDATE topology_change_requests SET review_status = ?1, lifecycle_state = ?2, reviewed_by = ?3, reviewed_at = datetime('now'), updated_at = datetime('now') WHERE id = ?4 AND tenant_id = ?5",
             [review_status, lifecycle_state, reviewed_by, id, tenant_id]
           ),
         {:ok, reviewed} <- get_change_request(id, tenant_id: tenant_id, workspace_id: workspace_id) do
      {:ok, %{request: reviewed, applied: applied}}
    end
  end

  defp maybe_apply_change(_request, :reject, _apply?), do: {:ok, nil}
  defp maybe_apply_change(_request, :approve, false), do: {:ok, nil}

  defp maybe_apply_change(%{request_type: "create_node"} = request, :approve, true) do
    payload = request.proposed_payload

    attrs =
      %{
        tenant_id: request.tenant_id,
        workspace_id: request.workspace_id,
        slug: fetch_payload!(payload, "slug"),
        name: fetch_payload!(payload, "name"),
        kind: parse_kind(Map.get(payload, "kind", "project")),
        parent_id: Map.get(payload, "parent_id"),
        description: Map.get(payload, "description"),
        metadata: Map.get(payload, "metadata", %{})
      }

    create_node(attrs)
  end

  defp maybe_apply_change(%{request_type: "link_nodes"} = request, :approve, true) do
    payload = request.proposed_payload

    link_nodes(
      fetch_payload!(payload, "source_node_id"),
      fetch_payload!(payload, "target_node_id"),
      parse_relationship_type(Map.get(payload, "relationship_type", "references")),
      tenant_id: request.tenant_id,
      workspace_id: request.workspace_id,
      metadata: Map.get(payload, "metadata", %{})
    )
  end

  defp maybe_apply_change(%{request_type: request_type} = request, :approve, true)
       when request_type in ["add_node_member", "change_node_owner"] do
    payload = request.proposed_payload
    node_id = Map.get(payload, "node_id") || request.target_object_id
    principal_id = Map.get(payload, "principal_id") || Map.get(payload, "owner_id")

    membership =
      if request_type == "change_node_owner",
        do: :owner,
        else: parse_membership(Map.get(payload, "membership", "internal"))

    case add_node_member(node_id, principal_id,
           tenant_id: request.tenant_id,
           workspace_id: request.workspace_id,
           membership: membership,
           role: Map.get(payload, "role")
         ) do
      :ok -> {:ok, %{node_id: node_id, principal_id: principal_id, membership: membership}}
      other -> other
    end
  end

  defp maybe_apply_change(_request, :approve, true), do: {:ok, nil}

  defp get_or_create_workspace(%{slug: slug} = attrs) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())

    case Workspace.get_by_slug(slug, tenant_id) do
      {:ok, workspace} ->
        {:ok, workspace}

      {:error, :not_found} ->
        create_workspace(attrs)

      other ->
        other
    end
  end

  defp maybe_create_setup_nodes(workspace, attrs, create_starter_nodes?) do
    custom_nodes =
      attrs
      |> Map.get(:nodes, [])
      |> normalize_setup_node_specs()

    starter_nodes =
      if create_starter_nodes? do
        starter_nodes()
      else
        []
      end

    node_specs =
      (starter_nodes ++ custom_nodes)
      |> unique_by_slug()

    Enum.reduce_while(node_specs, {:ok, []}, fn spec, {:ok, acc} ->
      node_attrs =
        spec
        |> Map.put(:tenant_id, workspace.tenant_id)
        |> Map.put(:workspace_id, workspace.id)
        |> Map.put_new(:path, "nodes/#{spec.slug}")

      case create_node(node_attrs) do
        {:ok, node} -> {:cont, {:ok, [node | acc]}}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
    |> case do
      {:ok, nodes} -> {:ok, Enum.reverse(nodes)}
      other -> other
    end
  end

  defp normalize_setup_node_specs(specs) when is_list(specs) do
    Enum.map(specs, fn spec ->
      spec
      |> normalize_keys()
      |> Map.put_new(:kind, :project)
      |> Map.update!(:kind, &parse_kind/1)
      |> Map.put_new(:description, "Custom setup Node.")
    end)
  end

  defp normalize_setup_node_specs(_), do: []

  defp normalize_keys(map) when is_map(map) do
    map
    |> Enum.map(fn
      {key, value} when is_binary(key) -> {normalize_key(key), value}
      pair -> pair
    end)
    |> Map.new()
  end

  defp normalize_key("slug"), do: :slug
  defp normalize_key("name"), do: :name
  defp normalize_key("kind"), do: :kind
  defp normalize_key("description"), do: :description
  defp normalize_key("path"), do: :path
  defp normalize_key("metadata"), do: :metadata
  defp normalize_key("source"), do: :source
  defp normalize_key("target"), do: :target
  defp normalize_key("relationship_type"), do: :relationship_type
  defp normalize_key(other), do: other

  defp unique_by_slug(specs) do
    specs
    |> Enum.reverse()
    |> Enum.uniq_by(& &1.slug)
    |> Enum.reverse()
  end

  defp create_setup_relationships(_workspace, [], _attrs), do: {:ok, []}

  defp create_setup_relationships(workspace, nodes, attrs) do
    node_by_slug = Map.new(nodes, &{&1.slug, &1})

    default_relationships = [
      {"primary-user", "first-project", :participates_in, %{role: "operator"}},
      {"operating-rhythm", "first-project", :supports, %{cadence: "daily_weekly"}},
      {"knowledge-base", "first-project", :references, %{purpose: "research_context"}},
      {"inbox", "first-project", :supports, %{purpose: "unrouted_inputs"}}
    ]

    relationship_specs =
      default_relationships ++ Map.get(attrs, :relationships, [])

    Enum.reduce_while(relationship_specs, {:ok, []}, fn spec, {:ok, acc} ->
      with {:ok, source_slug, target_slug, relationship_type, metadata} <-
             normalize_relationship_spec(spec),
           {:ok, source} <- fetch_setup_node(node_by_slug, source_slug),
           {:ok, target} <- fetch_setup_node(node_by_slug, target_slug),
           {:ok, relationship} <-
             link_nodes(source.id, target.id, relationship_type,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id,
               metadata: metadata,
               created_by: Map.get(attrs, :actor_id) || "optimal.setup"
             ) do
        {:cont, {:ok, [relationship | acc]}}
      else
        {:error, :missing_setup_node} -> {:cont, {:ok, acc}}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
    |> case do
      {:ok, relationships} -> {:ok, Enum.reverse(relationships)}
      other -> other
    end
  end

  defp normalize_relationship_spec({source, target, type, metadata}),
    do: {:ok, source, target, type, metadata}

  defp normalize_relationship_spec(%{} = spec) do
    spec = normalize_keys(spec)

    {:ok, Map.fetch!(spec, :source), Map.fetch!(spec, :target),
     parse_relationship_type(Map.get(spec, :relationship_type, :references)),
     Map.get(spec, :metadata, %{})}
  end

  defp fetch_setup_node(node_by_slug, slug) do
    case Map.fetch(node_by_slug, slug) do
      {:ok, node} -> {:ok, node}
      :error -> {:error, :missing_setup_node}
    end
  end

  defp maybe_write_setup_files(workspace, workspace_path, nodes, opts) do
    if Keyword.get(opts, :write_files, true) == false do
      {:ok, %{exports: [], projections: [], rhythm_paths: [], agent_sop_path: nil}}
    else
      actor_id = Keyword.fetch!(opts, :actor_id)

      with {:ok, node_projection_result} <-
             write_node_projection_files(workspace, workspace_path, nodes, actor_id),
           {:ok, rhythm_paths} <-
             maybe_write_rhythm_files(workspace_path, Keyword.get(opts, :write_rhythm, true)),
           {:ok, agent_sop_path} <-
             maybe_write_agent_sop(
               workspace,
               workspace_path,
               Keyword.get(opts, :write_agent_sop, true)
             ) do
        {:ok,
         %{
           exports: node_projection_result.exports,
           projections: node_projection_result.projections,
           rhythm_paths: rhythm_paths,
           agent_sop_path: agent_sop_path
         }}
      end
    end
  end

  defp write_node_projection_files(workspace, workspace_path, nodes, actor_id) do
    Enum.reduce_while(nodes, {:ok, %{exports: [], projections: []}}, fn node, {:ok, acc} ->
      node_dir = Path.join([workspace_path, "nodes", node.slug])
      context_path = Path.join(node_dir, "context.md")
      signal_path = Path.join(node_dir, "signal.md")

      with :ok <- create_node_projection_dirs(node_dir),
           :ok <- write_if_missing(context_path, node_context_markdown(node)),
           :ok <- write_if_missing(signal_path, node_signal_markdown(node)),
           {:ok, context_projection} <-
             record_node_projection(
               workspace,
               workspace_path,
               node,
               context_path,
               "context",
               actor_id
             ),
           {:ok, signal_projection} <-
             record_node_projection(
               workspace,
               workspace_path,
               node,
               signal_path,
               "signal",
               actor_id
             ) do
        {:cont,
         {:ok,
          %{
            exports: [context_projection.export, signal_projection.export | acc.exports],
            projections: [
              context_projection.projection,
              signal_projection.projection | acc.projections
            ]
          }}}
      else
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
    |> case do
      {:ok, result} ->
        {:ok,
         %{
           exports: Enum.reverse(result.exports),
           projections: Enum.reverse(result.projections)
         }}

      other ->
        other
    end
  end

  defp create_node_projection_dirs(node_dir) do
    [
      "signals",
      "sources",
      "decisions",
      "workflows",
      "packages",
      "exports"
    ]
    |> Enum.reduce_while(:ok, fn child, :ok ->
      case File.mkdir_p(Path.join(node_dir, child)) do
        :ok -> {:cont, :ok}
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
  end

  defp record_node_projection(workspace, workspace_path, node, path, role, actor_id) do
    destination_uri = "file://#{Path.relative_to(path, workspace_path)}"
    content = File.read!(path)

    with {:ok, export} <-
           WorkspaceExport.record_export(%{
             tenant_id: workspace.tenant_id,
             workspace_id: workspace.id,
             source_object_type: "node",
             source_object_id: node.id,
             surface_type: "markdown",
             destination_uri: destination_uri,
             generated_by: actor_id,
             metadata: %{projection_role: role, node_slug: node.slug}
           }),
         {:ok, projection} <-
           WorkspaceExport.record_projection_revision(export.id, %{
             tenant_id: workspace.tenant_id,
             workspace_id: workspace.id,
             projection_uri: destination_uri,
             content: content,
             source_object_links: [%{type: "node", id: node.id}],
             created_by: actor_id,
             metadata: %{projection_role: role, node_slug: node.slug}
           }) do
      {:ok, %{export: export, projection: projection}}
    end
  end

  defp maybe_write_rhythm_files(_workspace_path, false), do: {:ok, []}

  defp maybe_write_rhythm_files(workspace_path, true) do
    rhythm_dir = Path.join(workspace_path, "rhythm")
    daily_dir = Path.join(rhythm_dir, "daily")
    weekly_dir = Path.join(rhythm_dir, "weekly")

    paths = [
      Path.join(rhythm_dir, "README.md"),
      Path.join(daily_dir, ".gitkeep"),
      Path.join(weekly_dir, ".gitkeep")
    ]

    with :ok <- File.mkdir_p(daily_dir),
         :ok <- File.mkdir_p(weekly_dir),
         :ok <- write_if_missing(Enum.at(paths, 0), rhythm_readme()) do
      File.touch!(Enum.at(paths, 1))
      File.touch!(Enum.at(paths, 2))
      {:ok, paths}
    end
  end

  defp maybe_write_agent_sop(_workspace, _workspace_path, false), do: {:ok, nil}

  defp maybe_write_agent_sop(workspace, workspace_path, true) do
    path = Path.join(workspace_path, "AGENTS.md")

    with :ok <- write_if_missing(path, agent_sop_markdown(workspace, workspace_path)) do
      {:ok, path}
    end
  end

  defp write_if_missing(path, content) do
    if File.exists?(path) do
      :ok
    else
      File.write(path, content)
    end
  end

  defp node_context_markdown(node) do
    """
    # #{node.name}

    ## Identity

    - Node ID: `#{node.id}`
    - Node type: `#{node.kind}`
    - Purpose: #{node.description || "Define this Node's purpose."}
    - Status: `#{node.status}`

    ## Relationships

    Add important parent, child, dependency, ownership, and collaboration links here.

    ## Stable Context

    Durable facts and reviewed memory for this Node should appear here as projections.
    Human edits are re-ingested as source evidence before becoming durable truth.

    ## Node Files

    - `signals/` holds time-based updates and classified input for this Node.
    - `sources/` holds local evidence or source references waiting for ingestion.
    - `decisions/` holds decision projections backed by Claims and Facts.
    - `workflows/` holds workflow and procedure projections.
    - `packages/` holds receiver/channel bundles for this Node.
    - `exports/` holds generated views and loose output artifacts for this Node.

    Packages are deliverable bundles, often zipped, prepared for a person,
    team, channel, or external system. Node-specific packages stay inside the
    owning Node. Only cross-node packages should live at workspace scope.
    """
  end

  defp node_signal_markdown(node) do
    """
    # #{node.name} Signal

    ## Current Focus

    - Define the current focus for this Node.

    ## Blockers

    - None recorded.

    ## Open Decisions

    - None recorded.
    """
  end

  defp rhythm_readme do
    """
    # Operating Rhythm

    Use this folder for daily and weekly operating state.

    Rhythm is working state, not durable truth by itself:

    ```text
    daily note -> Source Package -> Signal -> Claim/Fact/Memory when reviewed
    ```

    Suggested cadence:

    - daily: focus, blockers, decisions, follow-ups
    - weekly: review active Nodes, stale context, open Claims, next priorities
    - monthly: review topology, policies, workflows, and Skill Packages
    """
  end

  defp agent_sop_markdown(workspace, workspace_path) do
    """
    # Agent SOP

    Workspace: `#{workspace.id}`

    Agents working in this workspace must:

    1. Load workspace and Node scope before acting.
    2. Retrieve governed context before answering project or memory questions.
    3. Treat markdown as a projection or source input, not automatic truth.
    4. Record observations as pending Claims before promotion.
    5. Use registered tools and preserve audit/source links.

    Useful commands:

    ```bash
    mix optimal.topology --workspace #{workspace.id}
    mix optimal.rag "what changed this week?" --workspace #{workspace.id} --trace
    mix optimal.ingest_workspace #{workspace_path}
    ```
    """
  end

  defp row_to_change_request([
         id,
         tenant_id,
         workspace_id,
         request_type,
         target_object_type,
         target_object_id,
         proposed_payload,
         reason,
         requested_by,
         review_status,
         lifecycle_state,
         audit_event_links,
         created_at,
         updated_at
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      request_type: request_type,
      target_object_type: target_object_type,
      target_object_id: target_object_id,
      proposed_payload: decode_map(proposed_payload),
      reason: reason,
      requested_by: requested_by,
      review_status: review_status,
      lifecycle_state: lifecycle_state,
      audit_event_links: decode_list(audit_event_links),
      created_at: created_at,
      updated_at: updated_at
    }
  end

  defp fetch_payload!(payload, key) do
    case Map.fetch(payload, key) do
      {:ok, value} when value not in [nil, ""] -> value
      _ -> raise ArgumentError, "missing required topology payload field #{inspect(key)}"
    end
  end

  defp parse_kind(value) when is_atom(value), do: value
  defp parse_kind(value) when is_binary(value), do: String.to_existing_atom(value)

  defp parse_relationship_type(value) when is_atom(value), do: value
  defp parse_relationship_type(value) when is_binary(value), do: String.to_existing_atom(value)

  defp parse_membership(value) when is_atom(value), do: value
  defp parse_membership(value) when is_binary(value), do: String.to_existing_atom(value)

  defp deterministic_id(prefix, parts) do
    hash =
      :crypto.hash(:sha256, Enum.join(parts, "|"))
      |> Base.encode16(case: :lower)
      |> binary_part(0, 24)

    "#{prefix}_#{hash}"
  end

  defp workspace_clause(nil, _position), do: ""
  defp workspace_clause(_workspace_id, position), do: " AND workspace_id = ?#{position}"

  defp scoped_params(params, nil), do: params
  defp scoped_params(params, workspace_id), do: params ++ [workspace_id]

  defp decode_map(nil), do: %{}
  defp decode_map(""), do: %{}

  defp decode_map(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end

  defp decode_list(nil), do: []
  defp decode_list(""), do: []

  defp decode_list(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_list(decoded) -> decoded
      _ -> []
    end
  end
end
