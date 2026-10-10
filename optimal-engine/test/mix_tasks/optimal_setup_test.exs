defmodule Mix.Tasks.Optimal.SetupTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  alias OptimalEngine.Store
  alias OptimalEngine.Topology.Node
  alias OptimalEngine.Workspace
  alias OptimalEngine.WorkspaceTopology

  setup do
    suffix = System.unique_integer([:positive])
    tmp_dir = Path.join(System.tmp_dir!(), "optimal_setup_#{suffix}")
    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)
    File.mkdir_p!(tmp_dir)

    on_exit(fn ->
      Mix.Task.reenable("optimal.setup")

      if original_root do
        Application.put_env(:optimal_engine, :root_path, original_root)
      else
        Application.delete_env(:optimal_engine, :root_path)
      end

      File.rm_rf(tmp_dir)
    end)

    %{suffix: suffix, tmp_dir: tmp_dir}
  end

  test "setup creates the workspace topology and markdown operating surface", %{
    suffix: suffix,
    tmp_dir: tmp_dir
  } do
    slug = "setup-workspace-#{suffix}"
    workspace_path = Path.join(tmp_dir, slug)

    output =
      capture_io(fn ->
        Mix.Task.reenable("optimal.setup")

        Mix.Tasks.Optimal.Setup.run([
          slug,
          "--name",
          "Setup Workspace #{suffix}",
          "--root",
          tmp_dir
        ])
      end)

    assert output =~ "Optimal workspace ready"
    assert output =~ "mix optimal.topology --workspace default:#{slug}"

    assert {:ok, workspace} = Workspace.get_by_slug(slug, "default")
    assert workspace.id == "default:#{slug}"
    assert workspace.name == "Setup Workspace #{suffix}"

    assert {:ok, node_types} =
             WorkspaceTopology.list_node_types(workspace_id: workspace.id)

    assert "project" in Enum.map(node_types, & &1.slug)
    assert "person" in Enum.map(node_types, & &1.slug)
    assert "operation" in Enum.map(node_types, & &1.slug)

    assert {:ok, nodes} = Node.list(workspace_id: workspace.id)
    node_slugs = Enum.map(nodes, & &1.slug)

    assert "inbox" in node_slugs
    assert "first-project" in node_slugs
    assert "operating-rhythm" in node_slugs
    assert "primary-user" in node_slugs
    assert "knowledge-base" in node_slugs

    assert File.exists?(Path.join(workspace_path, "nodes/first-project/context.md"))
    assert File.exists?(Path.join(workspace_path, "nodes/first-project/signal.md"))
    assert File.dir?(Path.join(workspace_path, "nodes/first-project/signals"))
    assert File.dir?(Path.join(workspace_path, "nodes/first-project/sources"))
    assert File.dir?(Path.join(workspace_path, "nodes/first-project/decisions"))
    assert File.dir?(Path.join(workspace_path, "nodes/first-project/workflows"))
    assert File.dir?(Path.join(workspace_path, "nodes/first-project/packages"))
    assert File.dir?(Path.join(workspace_path, "nodes/first-project/exports"))
    assert File.exists?(Path.join(workspace_path, "rhythm/README.md"))
    assert File.exists?(Path.join(workspace_path, "rhythm/daily/.gitkeep"))
    assert File.exists?(Path.join(workspace_path, "rhythm/weekly/.gitkeep"))
    assert File.exists?(Path.join(workspace_path, "AGENTS.md"))

    assert {:ok, [[10]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM export_records WHERE workspace_id = ?1 AND source_object_type = 'node'",
               [workspace.id]
             )

    assert {:ok, [[10]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM projection_revisions WHERE workspace_id = ?1",
               [workspace.id]
             )

    assert {:ok, [[relationship_count]]} =
             Store.raw_query("SELECT COUNT(*) FROM node_relationships WHERE workspace_id = ?1", [
               workspace.id
             ])

    assert relationship_count >= 4
  end

  test "setup can create an explicit custom topology without starter nodes", %{
    suffix: suffix,
    tmp_dir: tmp_dir
  } do
    slug = "custom-workspace-#{suffix}"

    capture_io(fn ->
      Mix.Task.reenable("optimal.setup")

      Mix.Tasks.Optimal.Setup.run([
        slug,
        "--root",
        tmp_dir,
        "--no-starter-nodes",
        "--node",
        "product:customer-portal:Customer Portal",
        "--node",
        "project:launch-plan:Launch Plan"
      ])
    end)

    assert {:ok, workspace} = Workspace.get_by_slug(slug, "default")
    assert {:ok, nodes} = Node.list(workspace_id: workspace.id)

    assert Enum.map(nodes, & &1.slug) == ["customer-portal", "launch-plan"]
    assert Enum.map(nodes, & &1.kind) == [:product, :project]
  end
end
