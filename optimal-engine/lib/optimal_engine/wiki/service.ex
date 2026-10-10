defmodule OptimalEngine.Wiki.Service do
  @moduledoc """
  Lifecycle service for markdown wiki projections.

  The low-level wiki modules already know how to store pages, render
  directives, and check citation hygiene. This service connects that tier to
  workspace topology: Nodes become stored wiki pages, written markdown files,
  export records, projection revisions, and link-health records.
  """

  alias OptimalEngine.Tenancy.Tenant
  alias OptimalEngine.Topology.Node
  alias OptimalEngine.Wiki
  alias OptimalEngine.Wiki.Page
  alias OptimalEngine.Workspace
  alias OptimalEngine.Workspace.Filesystem, as: WorkspaceFilesystem
  alias OptimalEngine.WorkspaceExport
  alias OptimalEngine.WorkspaceTopology

  @default_schema %{
    "required_sections" => ["Summary", "Open threads", "Related"],
    "required_frontmatter" => ["slug", "audience", "version", "last_curated"],
    "max_bytes" => 50_000
  }

  @type render_result :: %{
          page: Page.t(),
          markdown: String.t(),
          path: String.t(),
          export: WorkspaceExport.export_record(),
          projection: WorkspaceExport.projection_revision(),
          link_health: WorkspaceExport.link_health_record(),
          integrity: Wiki.Integrity.report()
        }

  @doc """
  Render one Node into the workspace wiki.

  The node can be addressed by id or slug. When a slug is supplied,
  `:workspace_id` should be supplied so the lookup stays scoped.
  """
  @spec render_node_page(String.t(), keyword()) :: {:ok, render_result()} | {:error, term()}
  def render_node_page(node_id_or_slug, opts \\ []) when is_binary(node_id_or_slug) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id)
    audience = Keyword.get(opts, :audience, "default")
    actor_id = Keyword.get(opts, :actor_id, "wiki.service")

    with {:ok, node} <- get_node(node_id_or_slug, tenant_id, workspace_id),
         {:ok, workspace} <- get_workspace(node.workspace_id),
         {:ok, workspace_path} <- provision_workspace(workspace, opts),
         {:ok, relationships} <-
           WorkspaceTopology.relationships_for_node(node.id,
             tenant_id: tenant_id,
             workspace_id: node.workspace_id
           ),
         {:ok, page} <- build_node_page(node, workspace, relationships, audience, actor_id),
         :ok <- Wiki.put(page),
         {:ok, result} <-
           persist_projection(page, workspace, workspace_path,
             source_object_type: "node",
             source_object_id: node.id,
             source_object_links: [%{type: "node", id: node.id}],
             projection_role: "node_wiki",
             actor_id: actor_id
           ) do
      {:ok, result}
    end
  end

  @doc """
  Render the workspace's Node tree into a single wiki page.
  """
  @spec render_workspace_tree(String.t(), keyword()) :: {:ok, render_result()} | {:error, term()}
  def render_workspace_tree(workspace_id, opts \\ []) when is_binary(workspace_id) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    audience = Keyword.get(opts, :audience, "default")
    actor_id = Keyword.get(opts, :actor_id, "wiki.service")

    with {:ok, workspace} <- get_workspace(workspace_id),
         {:ok, workspace_path} <- provision_workspace(workspace, opts),
         {:ok, tree} <-
           WorkspaceTopology.node_tree(tenant_id: tenant_id, workspace_id: workspace.id),
         {:ok, page} <- build_tree_page(workspace, tree, audience, actor_id),
         :ok <- Wiki.put(page),
         {:ok, result} <-
           persist_projection(page, workspace, workspace_path,
             source_object_type: "workspace",
             source_object_id: workspace.id,
             source_object_links: tree_source_links(tree),
             projection_role: "workspace_tree_wiki",
             actor_id: actor_id
           ) do
      {:ok, result}
    end
  end

  @doc """
  Check a stored wiki page and record link-health for its latest projection
  when one exists.
  """
  @spec check_page(String.t(), keyword()) ::
          {:ok, %{page: Page.t(), integrity: map(), link_health: map() | nil}} | {:error, term()}
  def check_page(slug, opts \\ []) when is_binary(slug) do
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    audience = Keyword.get(opts, :audience, "default")

    with {:ok, page} <- Wiki.latest(tenant_id, slug, audience, workspace_id) do
      integrity = verify_page(page)
      link_health = maybe_record_latest_link_health(page, integrity)
      {:ok, %{page: page, integrity: integrity, link_health: link_health}}
    end
  end

  defp get_node(id_or_slug, tenant_id, workspace_id) do
    node_opts =
      [tenant_id: tenant_id] ++ if(workspace_id, do: [workspace_id: workspace_id], else: [])

    case Node.get(id_or_slug, node_opts) do
      {:ok, node} ->
        {:ok, node}

      {:error, :not_found} ->
        Node.get_by_slug(id_or_slug, node_opts)
    end
  end

  defp get_workspace(workspace_id) do
    Workspace.get(workspace_id)
  end

  defp provision_workspace(workspace, opts) do
    root = Keyword.get(opts, :root, Application.get_env(:optimal_engine, :root_path, File.cwd!()))
    WorkspaceFilesystem.provision(workspace.slug, root: root)
  end

  defp build_node_page(node, workspace, relationships, audience, actor_id) do
    slug = "node-#{node.slug}"
    now = DateTime.utc_now() |> DateTime.to_iso8601()
    citation = "optimal://nodes/#{node.id}"

    body = """
    # #{node.name}

    ## Summary

    #{node.name} is a #{node.kind} Node in workspace #{workspace.slug}. It exists to hold purpose, state, context, relationships, decisions, and projections for this area of work. {{cite: #{citation}}}

    ## Identity

    - Node ID: `#{node.id}` {{cite: #{citation}}}
    - Slug: `#{node.slug}` {{cite: #{citation}}}
    - Type: `#{node.kind}` {{cite: #{citation}}}
    - Status: `#{node.status}` {{cite: #{citation}}}
    - Lifecycle state: `#{node.lifecycle_state}` {{cite: #{citation}}}

    ## Purpose

    #{node.description || "No explicit purpose has been written yet."} {{cite: #{citation}}}

    ## Current state

    This page is a generated wiki projection. Durable topology remains in the workspace/node tables, while this markdown file is the human-operable view. {{cite: #{citation}}}

    ## Relationships

    #{relationship_lines(relationships, node)}

    ## Open threads

    - Confirm owner, review cadence, and current blockers if they are not already represented in Node metadata. {{cite: #{citation}}}

    ## Related

    #{related_lines(relationships, node)}
    """

    version = next_version(node.tenant_id, node.workspace_id, slug, audience)

    page =
      Page.new(
        tenant_id: node.tenant_id,
        workspace_id: node.workspace_id,
        slug: slug,
        audience: audience,
        version: version,
        frontmatter: %{
          "slug" => slug,
          "audience" => audience,
          "version" => version,
          "workspace_id" => node.workspace_id,
          "node_id" => node.id,
          "node_slug" => node.slug,
          "source_object_type" => "node",
          "source_object_id" => node.id,
          "last_curated" => now,
          "curated_by" => actor_id
        },
        body: String.trim_trailing(body) <> "\n",
        last_curated: now,
        curated_by: actor_id
      )

    {:ok, page}
  end

  defp build_tree_page(workspace, tree, audience, actor_id) do
    slug = "workspace-tree"
    now = DateTime.utc_now() |> DateTime.to_iso8601()
    citation = "optimal://workspaces/#{workspace.id}"

    body = """
    # #{workspace.name} Node Tree

    ## Summary

    This page shows the current workspace topology as a generated wiki projection. Workspace and Node rows remain the canonical topology state. {{cite: #{citation}}}

    ## Node tree

    #{tree_lines(tree)}

    ## Open threads

    - Review whether each Node has owner, purpose, status, and relationship coverage. {{cite: #{citation}}}

    ## Related

    - Workspace: `#{workspace.id}` {{cite: #{citation}}}
    """

    version = next_version(workspace.tenant_id, workspace.id, slug, audience)

    page =
      Page.new(
        tenant_id: workspace.tenant_id,
        workspace_id: workspace.id,
        slug: slug,
        audience: audience,
        version: version,
        frontmatter: %{
          "slug" => slug,
          "audience" => audience,
          "version" => version,
          "workspace_id" => workspace.id,
          "source_object_type" => "workspace",
          "source_object_id" => workspace.id,
          "last_curated" => now,
          "curated_by" => actor_id
        },
        body: String.trim_trailing(body) <> "\n",
        last_curated: now,
        curated_by: actor_id
      )

    {:ok, page}
  end

  defp next_version(tenant_id, workspace_id, slug, audience) do
    case Wiki.latest(tenant_id, slug, audience, workspace_id) do
      {:ok, page} -> page.version + 1
      {:error, :not_found} -> 1
    end
  end

  defp persist_projection(page, workspace, workspace_path, opts) do
    markdown = Wiki.to_markdown(page)

    root =
      if workspace.slug == "default" do
        workspace_path
      else
        Path.dirname(workspace_path)
      end

    path = WorkspaceFilesystem.wiki_path(root, workspace.slug, page.slug)
    destination_uri = "file://#{Path.relative_to(path, workspace_path)}"
    actor_id = Keyword.fetch!(opts, :actor_id)

    with :ok <- File.mkdir_p(Path.dirname(path)),
         :ok <- File.write(path, markdown),
         {:ok, export} <-
           WorkspaceExport.record_export(%{
             tenant_id: workspace.tenant_id,
             workspace_id: workspace.id,
             source_object_type: Keyword.fetch!(opts, :source_object_type),
             source_object_id: Keyword.fetch!(opts, :source_object_id),
             surface_type: "wiki_markdown",
             destination_uri: destination_uri,
             object_version: Integer.to_string(page.version),
             generated_by: actor_id,
             metadata: %{
               wiki_slug: page.slug,
               audience: page.audience,
               projection_role: Keyword.fetch!(opts, :projection_role)
             }
           }),
         {:ok, projection} <-
           WorkspaceExport.record_projection_revision(export.id, %{
             tenant_id: workspace.tenant_id,
             workspace_id: workspace.id,
             projection_uri: destination_uri,
             content: markdown,
             source_object_links: Keyword.fetch!(opts, :source_object_links),
             created_by: actor_id,
             metadata: %{wiki_slug: page.slug, audience: page.audience}
           }) do
      integrity = verify_page(page)

      {:ok, link_health} =
        record_link_health(page, export.id, projection.id, integrity)

      {:ok,
       %{
         page: page,
         markdown: markdown,
         path: path,
         export: export,
         projection: projection,
         link_health: link_health,
         integrity: integrity
       }}
    end
  end

  defp verify_page(page) do
    Wiki.verify_against_schema(page, @default_schema, resolve_uri: &resolve_uri/1)
  end

  defp maybe_record_latest_link_health(page, integrity) do
    # The deterministic export id mirrors WorkspaceExport.record_export/1. If
    # that export does not exist, checking the page should still succeed.
    destination_uri = "file://.wiki/#{page.slug}.md"

    case WorkspaceExport.record_export(%{
           tenant_id: page.tenant_id,
           workspace_id: page.workspace_id,
           source_object_type: Map.get(page.frontmatter, "source_object_type", "wiki_page"),
           source_object_id: Map.get(page.frontmatter, "source_object_id", page.slug),
           surface_type: "wiki_markdown",
           destination_uri: destination_uri,
           metadata: %{wiki_slug: page.slug, check_only: true}
         }) do
      {:ok, export} ->
        {:ok, health} = record_link_health(page, export.id, nil, integrity)
        health

      _ ->
        nil
    end
  end

  defp record_link_health(page, export_id, projection_id, integrity) do
    broken =
      integrity.issues
      |> Enum.filter(&(&1.kind == :broken_citation))
      |> Enum.map(& &1.message)

    missing =
      integrity.issues
      |> Enum.reject(&(&1.severity == :info))
      |> Enum.map(& &1.message)

    WorkspaceExport.record_link_health(%{
      tenant_id: page.tenant_id,
      workspace_id: page.workspace_id,
      export_record_id: export_id,
      projection_revision_id: projection_id,
      broken_links: broken,
      missing_references: missing,
      backlinks: wikilinks(page.body),
      stale_references: [],
      metadata: %{
        wiki_slug: page.slug,
        integrity_ok: integrity.ok?,
        issue_count: length(integrity.issues)
      }
    })
  end

  defp resolve_uri("optimal://nodes/" <> node_id) do
    case Node.get(node_id) do
      {:ok, _node} -> :ok
      {:error, reason} -> {:error, reason}
    end
  end

  defp resolve_uri("optimal://workspaces/" <> workspace_id) do
    case Workspace.get(workspace_id) do
      {:ok, _workspace} -> :ok
      {:error, reason} -> {:error, reason}
    end
  end

  defp resolve_uri("optimal://" <> _), do: :ok
  defp resolve_uri(_uri), do: :ok

  defp relationship_lines([], node) do
    "- No topology relationships are currently attached to [[node-#{node.slug}]]. {{cite: optimal://nodes/#{node.id}}}"
  end

  defp relationship_lines(relationships, node) do
    relationships
    |> Enum.map(fn rel ->
      related_id =
        if rel.source_node_id == node.id do
          rel.target_node_id
        else
          rel.source_node_id
        end

      "- `#{rel.relationship_type}` with `#{related_id}` {{cite: optimal://nodes/#{node.id}}}"
    end)
    |> Enum.join("\n")
  end

  defp related_lines([], node), do: "- [[node-#{node.slug}]] {{cite: optimal://nodes/#{node.id}}}"

  defp related_lines(relationships, node) do
    relationships
    |> Enum.map(fn rel ->
      related_id =
        if rel.source_node_id == node.id do
          rel.target_node_id
        else
          rel.source_node_id
        end

      "- `#{related_id}` via `#{rel.relationship_type}` {{cite: optimal://nodes/#{node.id}}}"
    end)
    |> Enum.join("\n")
  end

  defp tree_lines([]), do: "- No Nodes have been defined yet."

  defp tree_lines(tree) do
    tree
    |> Enum.flat_map(&tree_item_lines(&1, 0))
    |> Enum.join("\n")
  end

  defp tree_item_lines(%{node: node, children: children}, depth) do
    indent = String.duplicate("  ", depth)

    line =
      "#{indent}- [[node-#{node.slug}]] - #{node.name} (`#{node.kind}`) {{cite: optimal://nodes/#{node.id}}}"

    [line | Enum.flat_map(children, &tree_item_lines(&1, depth + 1))]
  end

  defp tree_source_links(tree) do
    tree
    |> Enum.flat_map(&tree_nodes/1)
    |> Enum.map(&%{type: "node", id: &1.id})
  end

  defp tree_nodes(%{node: node, children: children}) do
    [node | Enum.flat_map(children, &tree_nodes/1)]
  end

  defp wikilinks(body) do
    Regex.scan(~r/\[\[([^\]]+)\]\]/, body)
    |> Enum.map(fn [_, slug] -> slug end)
  end
end
