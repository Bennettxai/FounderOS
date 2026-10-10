defmodule OptimalEngine.OrganizationTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Organization
  alias OptimalEngine.Workspace

  test "one tenant can contain many organizations with independently filtered workspaces" do
    suffix = System.unique_integer([:positive])

    assert {:ok, first} =
             Organization.create(%{
               slug: "first-org-#{suffix}",
               name: "First Organization #{suffix}"
             })

    assert {:ok, second} =
             Organization.create(%{
               slug: "second-org-#{suffix}",
               name: "Second Organization #{suffix}"
             })

    assert {:ok, first_workspace} =
             Workspace.create(%{
               slug: "first-org-workspace-#{suffix}",
               name: "First Organization Workspace #{suffix}",
               organization_id: first.id
             })

    assert {:ok, second_workspace} =
             Workspace.create(%{
               slug: "second-org-workspace-#{suffix}",
               name: "Second Organization Workspace #{suffix}",
               organization_id: second.id
             })

    assert {:ok, first_workspaces} = Workspace.list(organization_id: first.id)
    assert Enum.any?(first_workspaces, &(&1.id == first_workspace.id))
    refute Enum.any?(first_workspaces, &(&1.id == second_workspace.id))

    assert {:ok, second_workspaces} = Workspace.list(organization_id: second.id)
    assert Enum.any?(second_workspaces, &(&1.id == second_workspace.id))
    refute Enum.any?(second_workspaces, &(&1.id == first_workspace.id))
  end

  test "workspace organization ownership can be reassigned explicitly" do
    suffix = System.unique_integer([:positive])

    {:ok, organization} =
      Organization.create(%{slug: "destination-#{suffix}", name: "Destination #{suffix}"})

    {:ok, workspace} =
      Workspace.create(%{slug: "movable-#{suffix}", name: "Movable #{suffix}"})

    assert workspace.organization_id == Organization.default_id()
    assert {:ok, moved} = Workspace.assign_organization(workspace.id, organization.id)
    assert moved.organization_id == organization.id
  end
end
