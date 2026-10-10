defmodule OptimalEngine.Topology.NodeTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Topology.Node

  describe "upsert/1" do
    test "creates a node with defaults" do
      slug = "test-node-#{System.unique_integer([:positive])}"

      assert {:ok,
              %Node{
                slug: ^slug,
                workspace_id: "default",
                kind: :project,
                node_type_id: "default:project",
                style: :internal,
                status: :active
              }} =
               Node.upsert(%{slug: slug, name: "Test Project", kind: :project})

      assert {:ok, fetched} = Node.get_by_slug(slug)
      assert fetched.name == "Test Project"
      assert fetched.path == "nodes/#{slug}"
    end

    test "scopes natural identity by workspace, not tenant only" do
      slug = "same-project-#{System.unique_integer([:positive])}"

      assert {:ok, workspace_a} =
               Node.upsert(%{
                 slug: slug,
                 workspace_id: "workspace-a",
                 name: "Project in Workspace A",
                 kind: :project
               })

      assert {:ok, workspace_b} =
               Node.upsert(%{
                 slug: slug,
                 workspace_id: "workspace-b",
                 name: "Project in Workspace B",
                 kind: :project
               })

      assert workspace_a.id != workspace_b.id
      assert workspace_a.workspace_id == "workspace-a"
      assert workspace_b.workspace_id == "workspace-b"

      assert {:ok, fetched_a} = Node.get_by_slug(slug, workspace_id: "workspace-a")
      assert {:ok, fetched_b} = Node.get_by_slug(slug, workspace_id: "workspace-b")

      assert fetched_a.name == "Project in Workspace A"
      assert fetched_b.name == "Project in Workspace B"
    end

    test "rejects invalid kind via guard" do
      assert_raise FunctionClauseError, fn ->
        Node.upsert(%{slug: "x", name: "X", kind: :bogus})
      end
    end

    test "rejects invalid style with a soft error" do
      assert {:error, {:invalid_style, :weird}} =
               Node.upsert(%{
                 slug: "x-#{System.unique_integer([:positive])}",
                 name: "X",
                 kind: :unit,
                 style: :weird
               })
    end

    test "is idempotent — re-upsert updates name + description" do
      slug = "idem-#{System.unique_integer([:positive])}"

      assert {:ok, _} = Node.upsert(%{slug: slug, name: "Alpha", kind: :team})

      assert {:ok, _} =
               Node.upsert(%{
                 slug: slug,
                 name: "Alpha Renamed",
                 kind: :team,
                 description: "updated"
               })

      assert {:ok, %Node{name: "Alpha Renamed", description: "updated"}} = Node.get_by_slug(slug)
    end
  end

  describe "hierarchy" do
    setup do
      suffix = System.unique_integer([:positive])
      {:ok, parent} = Node.upsert(%{slug: "parent-#{suffix}", name: "Parent", kind: :unit})

      {:ok, child_a} =
        Node.upsert(%{
          slug: "child-a-#{suffix}",
          name: "Child A",
          kind: :team,
          parent_id: parent.id
        })

      {:ok, child_b} =
        Node.upsert(%{
          slug: "child-b-#{suffix}",
          name: "Child B",
          kind: :team,
          parent_id: parent.id
        })

      {:ok, grandchild} =
        Node.upsert(%{
          slug: "grand-#{suffix}",
          name: "GrandChild",
          kind: :project,
          parent_id: child_a.id
        })

      {:ok, parent: parent, child_a: child_a, child_b: child_b, grandchild: grandchild}
    end

    test "children/1 returns direct descendants", %{parent: parent, child_a: a, child_b: b} do
      assert {:ok, kids} = Node.children(parent.id)
      ids = Enum.map(kids, & &1.id)
      assert a.id in ids
      assert b.id in ids
      refute parent.id in ids
    end

    test "ancestors/1 walks up root→self ordered", %{parent: parent, child_a: a, grandchild: g} do
      assert {:ok, chain} = Node.ancestors(g.id)
      assert Enum.map(chain, & &1.id) == [parent.id, a.id, g.id]
    end

    test "rejects parent from another workspace" do
      suffix = System.unique_integer([:positive])

      {:ok, parent} =
        Node.upsert(%{
          workspace_id: "parent-workspace-#{suffix}",
          slug: "parent",
          name: "Parent",
          kind: :unit
        })

      parent_id = parent.id
      child_workspace_id = "child-workspace-#{suffix}"

      assert {:error, {:parent_not_found_in_workspace, ^parent_id, ^child_workspace_id}} =
               Node.upsert(%{
                 workspace_id: child_workspace_id,
                 slug: "child",
                 name: "Child",
                 kind: :project,
                 parent_id: parent.id
               })
    end
  end

  describe "list/1 filters" do
    test "filter by kind" do
      suffix = System.unique_integer([:positive])

      {:ok, _} =
        Node.upsert(%{
          slug: "list-u-#{suffix}",
          workspace_id: "list-workspace-a",
          name: "U",
          kind: :unit
        })

      {:ok, _} =
        Node.upsert(%{
          slug: "list-p-#{suffix}",
          workspace_id: "list-workspace-a",
          name: "P",
          kind: :project
        })

      {:ok, _} =
        Node.upsert(%{
          slug: "list-u-#{suffix}",
          workspace_id: "list-workspace-b",
          name: "U Other Workspace",
          kind: :unit
        })

      assert {:ok, units} = Node.list(kind: :unit, workspace_id: "list-workspace-a")
      assert Enum.any?(units, &(&1.slug == "list-u-#{suffix}"))
      refute Enum.any?(units, &(&1.slug == "list-p-#{suffix}"))
      assert Enum.all?(units, &(&1.workspace_id == "list-workspace-a"))
    end
  end
end
