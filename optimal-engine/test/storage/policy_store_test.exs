defmodule OptimalEngine.Storage.PolicyStoreTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Storage.PolicyStore
  alias OptimalEngine.WorkspaceTopology

  setup do
    suffix = System.unique_integer([:positive])

    {:ok, workspace} =
      WorkspaceTopology.create_workspace(%{
        slug: "storage-policy-#{suffix}",
        name: "Storage Policy #{suffix}"
      })

    %{workspace: workspace}
  end

  test "persists versioned workspace storage policy", %{workspace: workspace} do
    assert {:ok, initial} = PolicyStore.get_policy(workspace.id)
    assert initial.policy_version == 1
    assert initial.use_cases == ["desktop_local"]

    assert {:ok, first} =
             PolicyStore.put_policy(workspace.id, ["desktop_local"], actor_id: "test")

    assert first.policy_version == 2
    assert first.use_cases == ["desktop_local"]

    assert {:ok, second} =
             PolicyStore.put_policy(workspace.id, ["desktop_local", "analytics"], actor_id: "test")

    assert second.policy_version == 3
    assert second.use_cases == ["desktop_local", "analytics"]
  end

  test "rejects unknown provider overrides", %{workspace: workspace} do
    assert {:error, {:unknown_providers, [:imaginary]}} =
             PolicyStore.put_policy(workspace.id, ["desktop_local"],
               provider_overrides: %{imaginary: "enabled"}
             )
  end

  test "append-only mutations are idempotent and workspace scoped", %{workspace: workspace} do
    attrs = %{
      workspace_id: workspace.id,
      device_id: "device-a",
      entity_type: "memory_object",
      entity_id: "memory-1",
      operation: :update,
      payload: %{"title" => "Updated"},
      idempotency_key: "mutation-#{workspace.id}"
    }

    assert {:ok, first} = PolicyStore.append_mutation(attrs)
    assert {:ok, second} = PolicyStore.append_mutation(attrs)
    assert first.sequence == second.sequence

    assert {:ok, [mutation]} = PolicyStore.mutations_after(workspace.id, 0)
    assert mutation.workspace_id == workspace.id
    assert mutation.payload == %{"title" => "Updated"}

    assert :ok = PolicyStore.advance_cursor(workspace.id, "replica-b", mutation.sequence)
    assert :ok = PolicyStore.advance_cursor(workspace.id, "replica-b", 0)
  end
end
