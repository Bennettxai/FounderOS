defmodule OptimalEngine.Compliance.DSARTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Compliance.DSAR
  alias OptimalEngine.Identity.Principal

  setup do
    suffix = System.unique_integer([:positive])
    id = "user:dsar-#{suffix}@test"

    {:ok, _} =
      Principal.upsert(%{
        id: id,
        kind: :user,
        display_name: "DSAR Subject #{suffix}"
      })

    %{principal_id: id, suffix: suffix}
  end

  test "export/2 returns a full shape even when many sections are empty", %{principal_id: pid} do
    assert {:ok, export} = DSAR.export(pid)

    assert is_map(export.principal)
    assert export.principal.id == pid

    # Every documented section key is present
    for key <- [:contexts, :mentions, :memberships, :skills, :roles, :groups, :events] do
      assert Map.has_key?(export, key)
      assert is_list(Map.get(export, key))
    end

    assert is_binary(export.exported_at)
  end

  test "export/2 returns :not_found for an unknown principal" do
    assert {:error, :not_found} = DSAR.export("user:ghost-#{System.unique_integer([:positive])}")
  end
end
