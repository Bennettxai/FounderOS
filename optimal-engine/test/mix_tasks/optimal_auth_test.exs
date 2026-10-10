defmodule Mix.Tasks.Optimal.AuthTest do
  use ExUnit.Case, async: false

  import ExUnit.CaptureIO

  alias OptimalEngine.Auth.ApiKey

  @tenant_id "default"

  setup do
    Mix.Task.reenable("optimal.auth")
    :ok
  end

  test "mint prints a one-time API key" do
    name = "cli mint #{System.unique_integer([:positive])}"

    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Auth.run([
          "mint",
          "--name",
          name,
          "--workspace",
          "default:cli-test"
        ])
      end)

    assert output =~ "API key minted"
    assert output =~ "oe_"
    assert output =~ "default:cli-test"
  end

  test "mint --json returns machine-readable key payload" do
    name = "cli json #{System.unique_integer([:positive])}"

    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Auth.run(["mint", "--name", name, "--json"])
      end)

    assert {:ok, %{"id" => id, "key" => key, "tenant_id" => @tenant_id}} = Jason.decode(output)
    assert String.starts_with?(key, "oe_")

    assert {:ok, verified} = ApiKey.verify(key)
    assert verified.id == id
  end

  test "list shows minted key without secret" do
    name = "cli list #{System.unique_integer([:positive])}"
    {:ok, %{id: id, key: key}} = ApiKey.mint(%{tenant_id: @tenant_id, name: name})

    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Auth.run(["list", "--tenant", @tenant_id])
      end)

    assert output =~ id
    assert output =~ name
    refute output =~ key
  end

  test "revoke disables a key" do
    {:ok, %{id: id, key: key}} =
      ApiKey.mint(%{
        tenant_id: @tenant_id,
        name: "cli revoke #{System.unique_integer([:positive])}"
      })

    output =
      capture_io(fn ->
        Mix.Tasks.Optimal.Auth.run(["revoke", id])
      end)

    assert output =~ "Revoked #{id}"
    assert {:error, :revoked} = ApiKey.verify(key)
  end
end
