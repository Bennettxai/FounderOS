defmodule OptimalEngine.Identity.ACLTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Identity.{ACL, Group, Principal, Role}

  setup do
    # Use unique ids per test to avoid interference between runs.
    suffix = System.unique_integer([:positive])

    principal = "user:ada-#{suffix}@sample.test"
    stranger = "user:bob-#{suffix}@sample.test"
    group = "group:sales-#{suffix}"
    resource = "optimal://nodes/test/#{suffix}.md"

    {:ok, _} =
      Principal.upsert(%{
        id: principal,
        kind: :user,
        display_name: "Ada #{suffix}"
      })

    {:ok, _} =
      Principal.upsert(%{
        id: stranger,
        kind: :user,
        display_name: "Bob #{suffix}"
      })

    {:ok, _} = Group.upsert(%{id: group, name: "sales-#{suffix}"})
    :ok = Group.add_member(principal, group)

    {:ok, principal: principal, stranger: stranger, group: group, resource: resource}
  end

  describe "can?/3 permissive default" do
    test "grants access when no ACL exists for the resource",
         %{principal: principal, resource: resource} do
      assert ACL.can?(principal, resource, :read)
    end
  end

  describe "direct principal grant" do
    test "allows the granted principal, denies the stranger",
         %{principal: principal, stranger: stranger, resource: resource} do
      :ok = ACL.grant(%{resource_uri: resource, principal_id: principal, permission: :read})

      assert ACL.can?(principal, resource, :read)
      refute ACL.can?(stranger, resource, :read)
    end

    test "read grant does not imply write grant",
         %{principal: principal, resource: resource} do
      :ok = ACL.grant(%{resource_uri: resource, principal_id: principal, permission: :read})

      refute ACL.can?(principal, resource, :write)
    end
  end

  describe "group grant" do
    test "allows principals in the granted group, denies those outside",
         %{principal: principal, stranger: stranger, group: group, resource: resource} do
      :ok = ACL.grant(%{resource_uri: resource, group_id: group, permission: :read})

      assert ACL.can?(principal, resource, :read)
      refute ACL.can?(stranger, resource, :read)
    end
  end

  describe "grant/1 validation" do
    test "rejects a grant with neither principal nor group", %{resource: resource} do
      assert {:error, :must_set_exactly_one_of_principal_or_group} =
               ACL.grant(%{resource_uri: resource, permission: :read})
    end

    test "rejects a grant with both principal and group",
         %{principal: principal, group: group, resource: resource} do
      assert {:error, :must_set_exactly_one_of_principal_or_group} =
               ACL.grant(%{
                 resource_uri: resource,
                 principal_id: principal,
                 group_id: group,
                 permission: :read
               })
    end
  end

  describe "Role.grant/1" do
    test "grants a role to a principal", %{principal: principal} do
      role_id = "role:test-#{System.unique_integer([:positive])}"
      {:ok, _} = Role.upsert(%{id: role_id, name: "test-role"})

      assert :ok = Role.grant(%{role_id: role_id, principal_id: principal})
      {:ok, roles} = Principal.roles(principal)
      assert role_id in roles
    end

    test "grants a role transitively via group membership",
         %{principal: principal, group: group} do
      role_id = "role:test-grp-#{System.unique_integer([:positive])}"
      {:ok, _} = Role.upsert(%{id: role_id, name: "test-grp-role"})

      assert :ok = Role.grant(%{role_id: role_id, group_id: group})
      {:ok, roles} = Principal.roles(principal)
      assert role_id in roles
    end
  end
end
