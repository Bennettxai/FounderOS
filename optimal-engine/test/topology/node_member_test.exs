defmodule OptimalEngine.Topology.NodeMemberTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Identity.Principal
  alias OptimalEngine.Topology.{Node, NodeMember}

  setup do
    suffix = System.unique_integer([:positive])

    {:ok, node} =
      Node.upsert(%{slug: "nm-test-#{suffix}", name: "NM Test", kind: :team})

    {:ok, internal_p} =
      Principal.upsert(%{
        id: "user:internal-#{suffix}",
        kind: :user,
        display_name: "Internal #{suffix}"
      })

    {:ok, external_p} =
      Principal.upsert(%{
        id: "user:external-#{suffix}",
        kind: :user,
        display_name: "External #{suffix}"
      })

    {:ok, agent_p} =
      Principal.upsert(%{
        id: "agent:observer-#{suffix}",
        kind: :agent,
        display_name: "Observer #{suffix}"
      })

    {:ok, node: node, internal: internal_p, external: external_p, agent: agent_p}
  end

  test "add + members_of works for all membership kinds",
       %{node: node, internal: internal, external: external, agent: agent} do
    :ok = NodeMember.add(node.id, internal.id, membership: :internal, role: "lead")
    :ok = NodeMember.add(node.id, external.id, membership: :external)
    :ok = NodeMember.add(node.id, agent.id, membership: :observer)

    assert {:ok, members} = NodeMember.members_of(node.id)
    assert length(members) == 3

    by_kind = Enum.group_by(members, & &1.membership)
    assert length(by_kind[:internal]) == 1
    assert length(by_kind[:external]) == 1
    assert length(by_kind[:observer]) == 1

    assert hd(by_kind[:internal]).role == "lead"
  end

  test "members_of with membership filter",
       %{node: node, internal: internal, external: external} do
    :ok = NodeMember.add(node.id, internal.id, membership: :internal)
    :ok = NodeMember.add(node.id, external.id, membership: :external)

    assert {:ok, [member]} = NodeMember.members_of(node.id, membership: :external)
    assert member.principal_id == external.id
  end

  test "nodes_of returns every active node for a principal",
       %{internal: p} do
    suffix = System.unique_integer([:positive])

    {:ok, node_a} = Node.upsert(%{slug: "nm-a-#{suffix}", name: "A", kind: :team})
    {:ok, node_b} = Node.upsert(%{slug: "nm-b-#{suffix}", name: "B", kind: :project})

    :ok = NodeMember.add(node_a.id, p.id, membership: :internal)
    :ok = NodeMember.add(node_b.id, p.id, membership: :owner)

    assert {:ok, memberships} = NodeMember.nodes_of(p.id)
    node_ids = Enum.map(memberships, & &1.node_id) |> Enum.sort()
    assert node_a.id in node_ids
    assert node_b.id in node_ids
  end

  test "remove/3 closes a membership by setting ended_at",
       %{node: node, internal: p} do
    :ok = NodeMember.add(node.id, p.id, membership: :internal)
    assert {:ok, [_active]} = NodeMember.members_of(node.id, membership: :internal)

    :ok = NodeMember.remove(node.id, p.id, membership: :internal)
    assert {:ok, []} = NodeMember.members_of(node.id, membership: :internal)
  end

  test "add is idempotent on (node, principal, membership)",
       %{node: node, internal: p} do
    :ok = NodeMember.add(node.id, p.id, membership: :internal, role: "first")
    :ok = NodeMember.add(node.id, p.id, membership: :internal, role: "second")

    assert {:ok, [single]} = NodeMember.members_of(node.id, membership: :internal)
    # INSERT OR IGNORE keeps the first row
    assert single.role == "first"
  end

  test "memberships are workspace-scoped for nodes and principals",
       %{internal: p} do
    suffix = System.unique_integer([:positive])
    workspace_a = "membership-a-#{suffix}"
    workspace_b = "membership-b-#{suffix}"

    {:ok, node_a} =
      Node.upsert(%{
        workspace_id: workspace_a,
        slug: "shared-project",
        name: "Shared Project A",
        kind: :project
      })

    {:ok, node_b} =
      Node.upsert(%{
        workspace_id: workspace_b,
        slug: "shared-project",
        name: "Shared Project B",
        kind: :project
      })

    :ok = NodeMember.add(node_a.id, p.id, workspace_id: workspace_a, membership: :owner)
    :ok = NodeMember.add(node_b.id, p.id, workspace_id: workspace_b, membership: :observer)

    assert {:ok, [member_a]} = NodeMember.members_of(node_a.id, workspace_id: workspace_a)
    assert member_a.workspace_id == workspace_a
    assert member_a.membership == :owner

    assert {:ok, []} = NodeMember.members_of(node_a.id, workspace_id: workspace_b)

    assert {:ok, [node_membership_a]} = NodeMember.nodes_of(p.id, workspace_id: workspace_a)
    assert node_membership_a.node_id == node_a.id
    assert node_membership_a.membership == :owner

    assert {:ok, [node_membership_b]} = NodeMember.nodes_of(p.id, workspace_id: workspace_b)
    assert node_membership_b.node_id == node_b.id
    assert node_membership_b.membership == :observer
  end

  test "add resolves workspace from the node instead of defaulting incorrectly",
       %{internal: p} do
    suffix = System.unique_integer([:positive])
    workspace_id = "membership-resolve-#{suffix}"

    {:ok, node} =
      Node.upsert(%{
        workspace_id: workspace_id,
        slug: "resolved-node",
        name: "Resolved Node",
        kind: :project
      })

    assert :ok = NodeMember.add(node.id, p.id, membership: :owner)

    assert {:ok, [member]} = NodeMember.members_of(node.id, workspace_id: workspace_id)
    assert member.workspace_id == workspace_id
    assert member.membership == :owner

    assert {:ok, []} = NodeMember.members_of(node.id, workspace_id: "default")
  end

  test "add rejects explicit workspace mismatch",
       %{internal: p} do
    suffix = System.unique_integer([:positive])

    {:ok, node} =
      Node.upsert(%{
        workspace_id: "membership-home-#{suffix}",
        slug: "home-node",
        name: "Home Node",
        kind: :project
      })

    wrong_workspace_id = "wrong-workspace-#{suffix}"

    assert {:error, {:node_not_found_in_workspace, node_id, ^wrong_workspace_id}} =
             NodeMember.add(node.id, p.id,
               workspace_id: wrong_workspace_id,
               membership: :owner
             )

    assert node_id == node.id
  end
end
