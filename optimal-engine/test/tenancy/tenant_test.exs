defmodule OptimalEngine.Tenancy.TenantTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Tenancy.Tenant

  describe "default tenant" do
    test "is seeded by migration 015" do
      assert {:ok, %Tenant{id: "default", name: "Default Tenant"}} = Tenant.get("default")
    end

    test "default_id/0 returns the reserved id" do
      assert Tenant.default_id() == "default"
    end
  end

  describe "create/1" do
    test "inserts a new tenant with all attributes" do
      id = "test-tenant-#{System.unique_integer([:positive])}"

      assert {:ok, %Tenant{id: ^id, name: "Test Corp", plan: "enterprise"}} =
               Tenant.create(%{
                 id: id,
                 name: "Test Corp",
                 plan: "enterprise",
                 region: "us-east-1",
                 metadata: %{"industry" => "saas"}
               })

      assert {:ok, fetched} = Tenant.get(id)
      assert fetched.plan == "enterprise"
      assert fetched.region == "us-east-1"
      assert fetched.metadata == %{"industry" => "saas"}
    end
  end

  describe "get/1" do
    test "returns :not_found for unknown tenant" do
      assert {:error, :not_found} = Tenant.get("does-not-exist-xyz")
    end
  end
end
