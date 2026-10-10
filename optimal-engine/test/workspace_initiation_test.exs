defmodule OptimalEngine.WorkspaceInitiationTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Store
  alias OptimalEngine.Topology.Node
  alias OptimalEngine.WorkspaceInitiation

  setup do
    suffix = System.unique_integer([:positive])
    tmp_dir = Path.join(System.tmp_dir!(), "workspace_initiation_#{suffix}")
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

    %{suffix: suffix, tmp_dir: tmp_dir}
  end

  test "initiates a workspace from a messy dump and applies conservative topology", %{
    suffix: suffix,
    tmp_dir: tmp_dir
  } do
    slug = "initiation-workspace-#{suffix}"

    dump = """
    # Setup dump

    Companies:
    - Example Company

    Projects:
    - Launch Plan - ship the first customer portal

    People:
    - Founder
    - Support Lead

    Operations:
    - Weekly Review

    Products:
    - Customer Portal

    Integrations:
    - Google Workspace
    - Fathom MCP
    - GitHub
    - custom API for billing
    """

    assert {:ok, result} =
             WorkspaceInitiation.initiate(%{
               slug: slug,
               name: "Initiation Workspace #{suffix}",
               root: tmp_dir,
               dump_text: dump,
               actor_id: "agent:test-initiation"
             })

    workspace_id = result.setup.workspace.id

    assert result.source_package.raw_text == dump
    assert result.source_package.source_type == "workspace_initiation_dump"
    assert result.setup_claim.review_status == "unreviewed"
    assert result.setup_claim.lifecycle_state == "pending"

    assert {:ok, persisted_claim} =
             MemoryCoreStore.get_claim(result.setup_claim.id, workspace_id: workspace_id)

    assert persisted_claim.review_status == "unreviewed"

    proposed = MapSet.new(result.proposed_nodes, &{&1.kind, &1.slug})
    assert {"entity", "example-company"} in proposed
    assert {"project", "launch-plan"} in proposed
    assert {"person", "founder"} in proposed
    assert {"operation", "weekly-review"} in proposed
    assert {"product", "customer-portal"} in proposed

    assert length(result.topology_change_requests) >= 5
    assert length(result.applied_topology_changes) == length(result.topology_change_requests)
    assert length(result.accepted_nodes) >= 5

    assert Enum.all?(
             result.applied_topology_changes,
             &(&1.request.review_status == "approved")
           )

    assert {:ok, _entity} = Node.get_by_slug("example-company", workspace_id: workspace_id)
    assert {:ok, _project} = Node.get_by_slug("launch-plan", workspace_id: workspace_id)
    assert {:ok, _person} = Node.get_by_slug("founder", workspace_id: workspace_id)
    assert {:ok, _starter} = Node.get_by_slug("inbox", workspace_id: workspace_id)

    assert File.exists?(Path.join([tmp_dir, slug, "nodes", "launch-plan", "context.md"]))

    integration_slugs = MapSet.new(result.proposed_integrations, & &1.slug)
    assert "google_workspace" in integration_slugs
    assert "fathom" in integration_slugs
    assert "github" in integration_slugs
    assert "custom_api" in integration_slugs

    assert Enum.all?(result.registered_tool_definitions, &(&1.enabled_state == "disabled"))
    assert Enum.all?(result.registered_tool_definitions, &(&1.lifecycle_state == "draft"))

    assert {:ok, [[tool_count]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM mcp_tool_definitions WHERE workspace_id = ?1 AND enabled_state = 'disabled'",
               [workspace_id]
             )

    assert tool_count >= 4

    assert Enum.any?(result.open_questions, &String.contains?(&1, "Which proposed Nodes"))
    assert Enum.any?(result.open_questions, &String.contains?(&1, "credentials"))
    assert Enum.any?(result.next_actions, &String.contains?(&1, "Confirm integration scopes"))
    assert Enum.any?(result.next_actions, &String.contains?(&1, "Inspect accepted Nodes"))
  end

  test "review-only initiation leaves topology proposals pending", %{
    suffix: suffix,
    tmp_dir: tmp_dir
  } do
    slug = "review-only-workspace-#{suffix}"

    dump = """
    Projects:
    - Launch Plan

    People:
    - Operator
    """

    assert {:ok, result} =
             WorkspaceInitiation.initiate(%{
               slug: slug,
               name: "Review Only Workspace #{suffix}",
               root: tmp_dir,
               dump_text: dump,
               review_only: true,
               actor_id: "agent:test-initiation"
             })

    workspace_id = result.setup.workspace.id

    assert result.applied_topology_changes == []
    assert result.accepted_nodes == []
    assert Enum.all?(result.topology_change_requests, &(&1.review_status == "pending"))
    assert {:error, :not_found} = Node.get_by_slug("launch-plan", workspace_id: workspace_id)
    assert {:ok, _starter} = Node.get_by_slug("inbox", workspace_id: workspace_id)
    assert Enum.any?(result.next_actions, &String.contains?(&1, "Review proposals"))
  end

  test "candidate inference stays conservative for vague text" do
    dump = "I have a lot going on. Need help organizing everything and planning better."

    assert [] = WorkspaceInitiation.infer_node_candidates(dump)
    assert [] = WorkspaceInitiation.infer_integration_candidates(dump)
  end
end
