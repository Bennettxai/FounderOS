defmodule OptimalEngine.Wiki.ServiceTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Store
  alias OptimalEngine.Topology.Node
  alias OptimalEngine.Wiki
  alias OptimalEngine.Wiki.Service, as: WikiService
  alias OptimalEngine.Workspace
  alias OptimalEngine.WorkspaceExport
  alias OptimalEngine.WorkspaceTopology

  setup do
    suffix = System.unique_integer([:positive])
    tmp_dir = Path.join(System.tmp_dir!(), "wiki_service_#{suffix}")
    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)
    File.mkdir_p!(tmp_dir)

    on_exit(fn ->
      if original_root do
        Application.put_env(:optimal_engine, :root_path, original_root)
      else
        Application.delete_env(:optimal_engine, :root_path)
      end

      File.rm_rf(tmp_dir)
    end)

    slug = "wiki-service-#{suffix}"

    {:ok, setup_result} =
      WorkspaceTopology.setup_workspace(%{
        slug: slug,
        name: "Wiki Service #{suffix}",
        root: tmp_dir,
        actor_id: "test"
      })

    {:ok, project} =
      Node.get_by_slug("first-project",
        tenant_id: setup_result.workspace.tenant_id,
        workspace_id: setup_result.workspace.id
      )

    %{
      suffix: suffix,
      tmp_dir: tmp_dir,
      workspace: setup_result.workspace,
      workspace_path: setup_result.path,
      project: project
    }
  end

  test "renders a node into stored wiki, markdown projection, revision, and link health", %{
    tmp_dir: tmp_dir,
    workspace: workspace,
    workspace_path: workspace_path,
    project: project
  } do
    assert {:ok, result} =
             WikiService.render_node_page(project.slug,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id,
               root: tmp_dir,
               actor_id: "test"
             )

    assert result.page.slug == "node-#{project.slug}"
    assert result.page.workspace_id == workspace.id
    assert result.integrity.ok?
    assert result.export.source_object_type == "node"
    assert result.export.source_object_id == project.id
    assert result.export.surface_type == "wiki_markdown"
    assert result.projection.source_object_links == [%{"type" => "node", "id" => project.id}]
    assert result.link_health.status == "healthy"

    assert File.exists?(Path.join([workspace_path, ".wiki", "node-#{project.slug}.md"]))
    assert result.markdown =~ "source_object_type: node"
    assert result.markdown =~ "{{cite: optimal://nodes/#{project.id}}}"

    assert {:ok, stored} =
             Wiki.latest(workspace.tenant_id, result.page.slug, "default", workspace.id)

    assert stored.version == 1
    assert stored.body =~ "## Summary"
    assert stored.body =~ "## Open threads"
    assert stored.body =~ "## Related"

    assert {:ok, :fresh} = WorkspaceExport.detect_drift(result.export.id, result.markdown)
  end

  test "rendering the same node creates a new wiki version and projection revision", %{
    tmp_dir: tmp_dir,
    workspace: workspace,
    project: project
  } do
    assert {:ok, first} =
             WikiService.render_node_page(project.id,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id,
               root: tmp_dir,
               actor_id: "test"
             )

    assert {:ok, second} =
             WikiService.render_node_page(project.id,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id,
               root: tmp_dir,
               actor_id: "test"
             )

    assert first.page.version == 1
    assert second.page.version == 2
    assert first.export.id == second.export.id
    assert first.projection.revision_number == 1
    assert second.projection.revision_number == 2
  end

  test "renders workspace tree as a wiki page linked to node projections", %{
    tmp_dir: tmp_dir,
    workspace: workspace,
    workspace_path: workspace_path,
    project: project
  } do
    assert {:ok, result} =
             WikiService.render_workspace_tree(workspace.id,
               tenant_id: workspace.tenant_id,
               root: tmp_dir,
               actor_id: "test"
             )

    assert result.page.slug == "workspace-tree"
    assert result.export.source_object_type == "workspace"
    assert result.export.source_object_id == workspace.id
    assert result.projection.source_object_links |> Enum.any?(&(&1["id"] == project.id))
    assert result.markdown =~ "[[node-#{project.slug}]]"
    assert File.exists?(Path.join([workspace_path, ".wiki", "workspace-tree.md"]))
    assert result.integrity.ok?
  end

  test "check_page verifies a stored page and records health", %{
    tmp_dir: tmp_dir,
    workspace: workspace,
    project: project
  } do
    assert {:ok, render} =
             WikiService.render_node_page(project.id,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id,
               root: tmp_dir,
               actor_id: "test"
             )

    assert {:ok, checked} =
             WikiService.check_page(render.page.slug,
               tenant_id: workspace.tenant_id,
               workspace_id: workspace.id
             )

    assert checked.page.slug == render.page.slug
    assert checked.integrity.ok?
    assert checked.link_health.status == "healthy"

    assert {:ok, [[count]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM link_health_records WHERE workspace_id = ?1 AND metadata LIKE ?2",
               [workspace.id, "%#{render.page.slug}%"]
             )

    assert count >= 2
  end

  test "project lives inside a workspace, not beside it", %{workspace: workspace, project: project} do
    assert {:ok, fetched_workspace} = Workspace.get(workspace.id)
    assert fetched_workspace.id == workspace.id
    assert project.workspace_id == workspace.id
    assert project.kind == :project
  end
end
