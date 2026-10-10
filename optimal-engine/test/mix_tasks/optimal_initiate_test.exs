defmodule Mix.Tasks.Optimal.InitiateTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  alias OptimalEngine.Store
  alias OptimalEngine.Workspace

  setup do
    suffix = System.unique_integer([:positive])
    tmp_dir = Path.join(System.tmp_dir!(), "optimal_initiate_#{suffix}")
    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)
    File.mkdir_p!(tmp_dir)

    on_exit(fn ->
      Mix.Task.reenable("optimal.initiate")

      if original_root do
        Application.put_env(:optimal_engine, :root_path, original_root)
      else
        Application.delete_env(:optimal_engine, :root_path)
      end

      File.rm_rf(tmp_dir)
    end)

    %{suffix: suffix, tmp_dir: tmp_dir}
  end

  test "CLI initiates a workspace from a dump file and applies conservative proposals", %{
    suffix: suffix,
    tmp_dir: tmp_dir
  } do
    slug = "cli-initiation-#{suffix}"
    dump_path = Path.join(tmp_dir, "setup.md")

    File.write!(dump_path, """
    Projects:
    - Founder Dashboard

    People:
    - Operator

    Integrations:
    - Google Workspace
    - Fathom MCP
    """)

    output =
      capture_io(fn ->
        Mix.Task.reenable("optimal.initiate")

        Mix.Tasks.Optimal.Initiate.run([
          slug,
          "--name",
          "CLI Initiation #{suffix}",
          "--root",
          tmp_dir,
          "--dump",
          dump_path
        ])
      end)

    assert output =~ "Optimal workspace initiated"
    assert output =~ "Source Package:"
    assert output =~ "Setup Claim:"
    assert output =~ "Applied:"
    assert output =~ "Accepted Nodes:"
    assert output =~ "project:founder-dashboard"
    assert output =~ "mcp_or_api:google_workspace"
    assert output =~ "fathom"
    assert output =~ "Inspect accepted Nodes"
    assert output =~ "Confirm integration scopes"

    assert {:ok, workspace} = Workspace.get_by_slug(slug, "default")

    assert {:ok, [[source_count]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM source_packages WHERE workspace_id = ?1 AND source_type = 'workspace_initiation_dump'",
               [workspace.id]
             )

    assert source_count == 1

    assert {:ok, [[approved_count]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM topology_change_requests WHERE workspace_id = ?1 AND review_status = 'approved'",
               [workspace.id]
             )

    assert approved_count >= 2
  end
end
