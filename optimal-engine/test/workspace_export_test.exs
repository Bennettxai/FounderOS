defmodule OptimalEngine.WorkspaceExportTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{WorkspaceExport, WorkspaceTopology}
  alias OptimalEngine.Store

  setup do
    suffix = System.unique_integer([:positive])
    workspace_id = "workspace-export-#{suffix}"
    tmp_dir = Path.join(System.tmp_dir!(), "workspace_export_#{suffix}")
    original_root = Application.get_env(:optimal_engine, :root_path)
    Application.put_env(:optimal_engine, :root_path, tmp_dir)

    on_exit(fn ->
      if original_root do
        Application.put_env(:optimal_engine, :root_path, original_root)
      else
        Application.delete_env(:optimal_engine, :root_path)
      end

      File.rm_rf(tmp_dir)
    end)

    {:ok, node} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_id,
        slug: "project-#{suffix}",
        name: "Project #{suffix}",
        kind: :project
      })

    %{workspace_id: workspace_id, node: node, tmp_dir: tmp_dir}
  end

  test "records export and projection revisions", %{workspace_id: workspace_id, node: node} do
    assert {:ok, export} =
             WorkspaceExport.record_export(%{
               workspace_id: workspace_id,
               source_object_type: "node",
               source_object_id: node.id,
               surface_type: "markdown",
               destination_uri: "file://nodes/#{node.slug}/context.md",
               object_version: "1",
               generated_by: "test"
             })

    assert export.workspace_id == workspace_id
    assert export.source_object_id == node.id
    assert export.surface_type == "markdown"

    assert {:ok, rev1} =
             WorkspaceExport.record_projection_revision(export.id, %{
               workspace_id: workspace_id,
               projection_uri: export.destination_uri,
               content: "# Project\n\nInitial content",
               source_object_links: [%{type: "node", id: node.id}]
             })

    assert rev1.revision_number == 1
    assert byte_size(rev1.content_hash) == 64
    assert [%{"type" => "node", "id" => node_id}] = rev1.source_object_links
    assert node_id == node.id

    assert {:ok, rev2} =
             WorkspaceExport.record_projection_revision(export.id, %{
               workspace_id: workspace_id,
               projection_uri: export.destination_uri,
               content: "# Project\n\nUpdated content"
             })

    assert rev2.revision_number == 2
    assert rev2.content_hash != rev1.content_hash
  end

  test "records link health and detects projection drift", %{workspace_id: workspace_id, node: node} do
    content = "# Project\n\nStable content"

    {:ok, export} =
      WorkspaceExport.record_export(%{
        workspace_id: workspace_id,
        source_object_type: "node",
        source_object_id: node.id,
        surface_type: "html",
        destination_uri: "file://nodes/#{node.slug}/index.html"
      })

    {:ok, revision} =
      WorkspaceExport.record_projection_revision(export.id, %{
        workspace_id: workspace_id,
        projection_uri: export.destination_uri,
        content: content
      })

    assert {:ok, :fresh} = WorkspaceExport.detect_drift(export.id, content)
    assert {:ok, :stale} = WorkspaceExport.detect_drift(export.id, content <> "\nChanged")

    assert {:ok, health} =
             WorkspaceExport.record_link_health(%{
               workspace_id: workspace_id,
               export_record_id: export.id,
               projection_revision_id: revision.id,
               broken_links: ["optimal://missing"],
               backlinks: ["optimal://nodes/source"],
               missing_references: [],
               stale_references: []
             })

    assert health.status == "issues"
    assert health.broken_links == ["optimal://missing"]
    assert health.backlinks == ["optimal://nodes/source"]
  end

  test "invalidates projections that depend on a node", %{workspace_id: workspace_id, node: node} do
    content = "# Project\n\nContent that came from a node."

    {:ok, export} =
      WorkspaceExport.record_export(%{
        workspace_id: workspace_id,
        source_object_type: "node",
        source_object_id: node.id,
        surface_type: "markdown",
        destination_uri: "file://nodes/#{node.slug}/context.md"
      })

    {:ok, revision} =
      WorkspaceExport.record_projection_revision(export.id, %{
        workspace_id: workspace_id,
        projection_uri: export.destination_uri,
        content: content,
        source_object_links: [%{type: "node", id: node.id}]
      })

    assert {:ok, result} =
             WorkspaceExport.invalidate_for_node(node.id,
               workspace_id: workspace_id,
               reason: "node updated"
             )

    assert result.export_record_ids == [export.id]
    assert result.projection_revision_count == 1
    assert result.link_health_record_count == 1

    assert {:ok, stale_export} = WorkspaceExport.get_export(export.id, workspace_id: workspace_id)
    assert stale_export.lifecycle_state == "stale"

    assert {:ok, stale_revision} =
             WorkspaceExport.get_projection_revision(revision.id, workspace_id: workspace_id)

    assert stale_revision.lifecycle_state == "stale"

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM link_health_records WHERE workspace_id = ?1 AND export_record_id = ?2 AND status = 'stale' AND stale_references LIKE ?3",
               [workspace_id, export.id, "%#{node.id}%"]
             )
  end

  test "captures content edits as source packages", %{workspace_id: workspace_id, node: node} do
    {:ok, export} =
      WorkspaceExport.record_export(%{
        workspace_id: workspace_id,
        source_object_type: "node",
        source_object_id: node.id,
        surface_type: "markdown",
        destination_uri: "file://nodes/#{node.slug}/context.md"
      })

    assert {:ok, capture} =
             WorkspaceExport.capture_edit(export.id, %{
               content: "Human added new project context.",
               actor_id: "human:test"
             })

    assert capture.kind == :source_package
    assert capture.source_package.workspace_id == workspace_id
    assert capture.source_package.source_type == "workspace_edit"
    assert capture.source_package.source_uri == export.destination_uri
    assert capture.source_package.raw_text == "Human added new project context."
    assert capture.topology_change_request == nil

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM derivation_ledger WHERE workspace_id = ?1 AND activity_type = 'workspace_export.capture_edit'",
               [workspace_id]
             )
  end

  test "captures topology edits as pending topology change requests", %{
    workspace_id: workspace_id,
    node: node
  } do
    {:ok, export} =
      WorkspaceExport.record_export(%{
        workspace_id: workspace_id,
        source_object_type: "node",
        source_object_id: node.id,
        surface_type: "markdown",
        destination_uri: "file://nodes/#{node.slug}/context.md"
      })

    assert {:ok, capture} =
             WorkspaceExport.capture_edit(export.id, %{
               edit_type: :topology,
               content: "Change this project owner.",
               actor_id: "agent:test",
               request_type: "change_node_owner",
               proposed_payload: %{
                 node_id: node.id,
                 owner_id: "principal:new-owner"
               }
             })

    assert capture.kind == :topology_change_request
    assert capture.source_package == nil
    assert request = capture.topology_change_request
    assert request.workspace_id == workspace_id
    assert request.request_type == "change_node_owner"
    assert request.target_object_type == "node"
    assert request.target_object_id == node.id
    assert request.review_status == "pending"
    assert request.lifecycle_state == "open"
    assert request.proposed_payload["owner_id"] == "principal:new-owner"
  end

  test "can route captured content edits back through signal intake", %{
    workspace_id: workspace_id,
    node: node,
    tmp_dir: tmp_dir
  } do
    {:ok, export} =
      WorkspaceExport.record_export(%{
        workspace_id: workspace_id,
        source_object_type: "node",
        source_object_id: node.id,
        surface_type: "markdown",
        destination_uri: "file://nodes/#{node.slug}/context.md"
      })

    assert {:ok, capture} =
             WorkspaceExport.capture_edit(export.id, %{
               content: "Routed workspace edit for project state.",
               actor_id: "human:test",
               route: true,
               node: "inbox",
               title: "Routed Workspace Edit"
             })

    assert capture.kind == :source_package
    assert capture.intake_result.signal.title == "Routed Workspace Edit"
    assert capture.intake_result.context.workspace_id == workspace_id

    [relative_path] = capture.intake_result.files_written
    assert File.exists?(Path.join(tmp_dir, relative_path))

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM contexts WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, capture.intake_result.context.id]
             )
  end
end
