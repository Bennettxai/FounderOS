defmodule OptimalEngine.Connectors.RegistryTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Connectors.Registry

  test "all/0 lists every adapter module" do
    mods = Registry.all()
    assert length(mods) == 15
    # Every module responds to kind/0
    assert Enum.all?(mods, fn m -> is_atom(m.kind()) end)
  end

  test "kinds/0 returns 14 unique adapter atoms" do
    kinds = Registry.kinds()
    assert length(kinds) == 15
    assert length(Enum.uniq(kinds)) == 15
    assert :slack in kinds
    assert :gmail in kinds
    assert :github in kinds
    assert :hubspot in kinds
  end

  test "fetch/1 returns the adapter module for a known kind" do
    assert {:ok, OptimalEngine.Connectors.Adapters.Slack} = Registry.fetch(:slack)
    assert {:ok, OptimalEngine.Connectors.Adapters.Linear} = Registry.fetch(:linear)
  end

  test "fetch/1 returns :unknown_kind for an unregistered atom" do
    assert {:error, :unknown_kind} = Registry.fetch(:nonexistent)
  end

  test "summary/0 emits (kind, name, auth) triples" do
    summary = Registry.summary()
    assert length(summary) == 15
    assert Enum.all?(summary, fn {k, n, a} -> is_atom(k) and is_binary(n) and is_atom(a) end)

    {:slack, name, auth} =
      Enum.find(summary, fn {k, _, _} -> k == :slack end)

    assert name == "Slack"
    assert auth == :oauth2
  end
end
