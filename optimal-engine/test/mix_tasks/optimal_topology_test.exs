defmodule Mix.Tasks.Optimal.TopologyTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  alias OptimalEngine.Topology.Node
  alias OptimalEngine.WorkspaceInitiation

  setup do
    suffix = System.unique_integer([:positive])
    tmp_dir = Path.join(System.tmp_dir!(), "optimal_topology_#{suffix}")
    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)
    File.mkdir_p!(tmp_dir)

    on_exit(fn ->
      Mix.Task.reenable("optimal.topology")

      if original_root do
        Application.put_env(:optimal_engine, :root_path, original_root)
      else
        Application.delete_env(:optimal_engine, :root_path)
      end

      File.rm_rf(tmp_dir)
    end)

    %{suffix: suffix, tmp_dir: tmp_dir}
  end

  test "CLI approves and applies a pending topology change", %{suffix: suffix, tmp_dir: tmp_dir} do
    slug = "topology-review-#{suffix}"

    assert {:ok, result} =
             WorkspaceInitiation.initiate(%{
               slug: slug,
               name: "Topology Review #{suffix}",
               root: tmp_dir,
               dump_text: """
               Projects:
               - Client Launch
               """,
               review_only: true,
               actor_id: "agent:test-topology"
             })

    workspace_id = result.setup.workspace.id

    request =
      Enum.find(result.topology_change_requests, &(&1.proposed_payload["slug"] == "client-launch"))

    assert request
    assert {:error, :not_found} = Node.get_by_slug("client-launch", workspace_id: workspace_id)

    output =
      capture_io(fn ->
        Mix.Task.reenable("optimal.topology")

        Mix.Tasks.Optimal.Topology.run([
          "approve",
          request.id,
          "--workspace",
          workspace_id,
          "--apply"
        ])
      end)

    assert output =~ "Topology change approved"
    assert output =~ "Applied:"
    assert {:ok, _node} = Node.get_by_slug("client-launch", workspace_id: workspace_id)
  end
end
