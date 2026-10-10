defmodule OptimalEngine.WorkspaceTopologyTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Identity.Principal
  alias OptimalEngine.WorkspaceTopology

  setup do
    suffix = System.unique_integer([:positive])
    tmp_dir = Path.join(System.tmp_dir!(), "workspace_topology_#{suffix}")
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

    :ok
  end

  test "creates workspaces with standard node types" do
    suffix = System.unique_integer([:positive])

    assert {:ok, workspace} =
             WorkspaceTopology.create_workspace(%{
               slug: "operating-space-#{suffix}",
               name: "Operating Space #{suffix}",
               description: "Workspace topology test"
             })

    assert workspace.id != nil

    assert {:ok, node_types} =
             WorkspaceTopology.list_node_types(workspace_id: workspace.id)

    slugs = Enum.map(node_types, & &1.slug) |> Enum.sort()
    assert "project" in slugs
    assert "person" in slugs
    assert "operation" in slugs
    assert "context" in slugs
    assert "division" in slugs
    assert "role" in slugs
    assert "process" in slugs
    assert "task" in slugs
    assert "customer" in slugs

    assert {:ok, type} = WorkspaceTopology.get_node_type("project", workspace_id: workspace.id)
    assert type.workspace_id == workspace.id
  end

  test "different workspaces can contain same project slug independently" do
    suffix = System.unique_integer([:positive])

    {:ok, workspace_a} =
      WorkspaceTopology.create_workspace(%{
        slug: "workspace-a-#{suffix}",
        name: "Workspace A #{suffix}"
      })

    {:ok, workspace_b} =
      WorkspaceTopology.create_workspace(%{
        slug: "workspace-b-#{suffix}",
        name: "Workspace B #{suffix}"
      })

    {:ok, project_a} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_a.id,
        slug: "shared-project",
        name: "Shared Project A",
        kind: :project
      })

    {:ok, project_b} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_b.id,
        slug: "shared-project",
        name: "Shared Project B",
        kind: :project
      })

    assert project_a.id != project_b.id
    assert project_a.workspace_id == workspace_a.id
    assert project_b.workspace_id == workspace_b.id
  end

  test "builds parent child hierarchy and returns workspace node tree" do
    suffix = System.unique_integer([:positive])
    workspace_id = "topology-tree-#{suffix}"

    {:ok, entity} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_id,
        slug: "entity",
        name: "Entity",
        kind: :entity
      })

    {:ok, department} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_id,
        slug: "department",
        name: "Department",
        kind: :department,
        parent_id: entity.id
      })

    {:ok, project} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_id,
        slug: "project",
        name: "Project",
        kind: :project,
        parent_id: department.id
      })

    assert {:ok, ancestors} =
             OptimalEngine.Topology.Node.ancestors(project.id, workspace_id: workspace_id)

    assert Enum.map(ancestors, & &1.id) == [entity.id, department.id, project.id]

    assert {:ok, [root]} = WorkspaceTopology.node_tree(workspace_id: workspace_id)
    assert root.node.id == entity.id
    assert [department_item] = root.children
    assert department_item.node.id == department.id
    assert [project_item] = department_item.children
    assert project_item.node.id == project.id
  end

  test "creates workspace-scoped node types" do
    suffix = System.unique_integer([:positive])
    workspace_id = "topology-type-workspace-#{suffix}"

    assert {:ok, type} =
             WorkspaceTopology.create_node_type(%{
               workspace_id: workspace_id,
               slug: "client-program",
               name: "Client Program",
               description: "Custom client program node type",
               metadata: %{review_cadence: "weekly"}
             })

    assert type.workspace_id == workspace_id
    assert type.slug == "client-program"
    assert type.metadata["review_cadence"] == "weekly"

    assert {:ok, fetched} =
             WorkspaceTopology.get_node_type("client-program", workspace_id: workspace_id)

    assert fetched.id == type.id
  end

  test "creates and reads typed node relationships inside a workspace" do
    suffix = System.unique_integer([:positive])
    workspace_id = "topology-rel-workspace-#{suffix}"

    assert {:ok, project} =
             WorkspaceTopology.create_node(%{
               workspace_id: workspace_id,
               slug: "project-#{suffix}",
               name: "Project #{suffix}",
               kind: :project
             })

    assert {:ok, person} =
             WorkspaceTopology.create_node(%{
               workspace_id: workspace_id,
               slug: "person-#{suffix}",
               name: "Person #{suffix}",
               kind: :person
             })

    assert {:ok, relationship} =
             WorkspaceTopology.link_nodes(person.id, project.id, :participates_in,
               workspace_id: workspace_id,
               metadata: %{role: "owner"}
             )

    assert relationship.workspace_id == workspace_id
    assert relationship.source_node_id == person.id
    assert relationship.target_node_id == project.id
    assert relationship.relationship_type == :participates_in
    assert relationship.metadata["role"] == "owner"

    assert {:ok, outgoing} =
             WorkspaceTopology.relationships_for_node(person.id,
               workspace_id: workspace_id,
               direction: :outgoing
             )

    assert Enum.map(outgoing, & &1.id) == [relationship.id]

    assert {:ok, incoming} =
             WorkspaceTopology.relationships_for_node(project.id,
               workspace_id: workspace_id,
               direction: :incoming
             )

    assert Enum.map(incoming, & &1.id) == [relationship.id]
  end

  test "rejects node relationships that cross workspace scope" do
    suffix = System.unique_integer([:positive])
    workspace_a = "topology-rel-a-#{suffix}"
    workspace_b = "topology-rel-b-#{suffix}"

    {:ok, source} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_a,
        slug: "source",
        name: "Source",
        kind: :project
      })

    {:ok, target} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_b,
        slug: "target",
        name: "Target",
        kind: :project
      })

    target_id = target.id

    assert {:error, {:node_not_found_in_workspace, :target, ^target_id, ^workspace_a}} =
             WorkspaceTopology.link_nodes(source.id, target.id, :depends_on,
               workspace_id: workspace_a
             )
  end

  test "reviews and applies create_node topology changes" do
    suffix = System.unique_integer([:positive])
    workspace_id = "topology-review-create-#{suffix}"

    assert {:ok, request} =
             WorkspaceTopology.propose_change(%{
               workspace_id: workspace_id,
               request_type: "create_node",
               target_object_type: "node",
               proposed_payload: %{
                 slug: "new-project",
                 name: "New Project",
                 kind: "project",
                 metadata: %{source: "test"}
               },
               requested_by: "agent:test"
             })

    assert request.review_status == "pending"

    assert {:ok, reviewed} =
             WorkspaceTopology.review_change_request(request.id, :approve,
               workspace_id: workspace_id,
               reviewed_by: "human:test",
               apply: true
             )

    assert reviewed.request.review_status == "approved"
    assert reviewed.applied.slug == "new-project"
    assert reviewed.applied.workspace_id == workspace_id

    assert {:ok, node} =
             OptimalEngine.Topology.Node.get_by_slug("new-project", workspace_id: workspace_id)

    assert node.id == reviewed.applied.id
  end

  test "rejects topology changes without applying them" do
    suffix = System.unique_integer([:positive])
    workspace_id = "topology-review-reject-#{suffix}"

    {:ok, request} =
      WorkspaceTopology.propose_change(%{
        workspace_id: workspace_id,
        request_type: "create_node",
        target_object_type: "node",
        proposed_payload: %{slug: "bad-node", name: "Bad Node", kind: "project"}
      })

    assert {:ok, reviewed} =
             WorkspaceTopology.review_change_request(request.id, :reject,
               workspace_id: workspace_id,
               reviewed_by: "human:test",
               apply: true
             )

    assert reviewed.request.review_status == "rejected"
    assert reviewed.applied == nil

    assert {:error, :not_found} =
             OptimalEngine.Topology.Node.get_by_slug("bad-node", workspace_id: workspace_id)
  end

  test "applies owner change requests as workspace-scoped node membership" do
    suffix = System.unique_integer([:positive])
    workspace_id = "topology-review-owner-#{suffix}"

    {:ok, node} =
      WorkspaceTopology.create_node(%{
        workspace_id: workspace_id,
        slug: "owner-project",
        name: "Owner Project",
        kind: :project
      })

    {:ok, principal} =
      Principal.upsert(%{
        id: "principal:owner-#{suffix}",
        kind: :user,
        display_name: "Owner #{suffix}"
      })

    {:ok, request} =
      WorkspaceTopology.propose_change(%{
        workspace_id: workspace_id,
        request_type: "change_node_owner",
        target_object_type: "node",
        target_object_id: node.id,
        proposed_payload: %{owner_id: principal.id}
      })

    assert {:ok, reviewed} =
             WorkspaceTopology.review_change_request(request.id, :approve,
               workspace_id: workspace_id,
               reviewed_by: "human:test",
               apply: true
             )

    assert reviewed.request.review_status == "approved"
    assert reviewed.applied.membership == :owner

    assert {:ok, [member]} = WorkspaceTopology.node_members(node.id, workspace_id: workspace_id)
    assert member.workspace_id == workspace_id
    assert member.principal_id == principal.id
    assert member.membership == :owner
  end
end
