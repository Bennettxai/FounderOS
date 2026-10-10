defmodule OptimalEngine.Storage.ProvidersTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Storage.{Planner, Providers}

  test "provider ids are unique and credentials are never returned" do
    providers = Providers.list()
    ids = Enum.map(providers, & &1.id)

    assert length(ids) == length(Enum.uniq(ids))
    assert "postgres" in ids
    assert "s3" in ids
    assert "nats_jetstream" in ids
    assert "valkey" in ids
    assert "qdrant" in ids
    assert "duckdb" in ids
    assert "openbao" in ids
    assert "fractal" in ids

    encoded = Jason.encode!(providers)
    refute encoded =~ "secret_access_key"
    refute encoded =~ "token_value"
  end

  test "local workspace plan uses embedded providers" do
    assert {:ok, plan} = Planner.plan(["desktop_local"])

    ids = Enum.map(plan.providers, & &1.id)
    assert ids == ~w(sqlite fts5 embedded_vectors rocksdb filesystem ets)
    assert plan.ready
    assert Enum.all?(plan.providers, &(&1.lifecycle_state == "active"))
  end

  test "configured external providers remain unverified until probed" do
    System.put_env("OPTIMAL_QDRANT_URL", "http://127.0.0.1:1")
    on_exit(fn -> System.delete_env("OPTIMAL_QDRANT_URL") end)

    assert {:ok, provider} = Providers.get("qdrant")
    assert provider.configured
    assert provider.lifecycle_state == "configured_unverified"

    assert {:ok, provider} = Providers.get("qdrant", probe: true)
    assert provider.lifecycle_state == "unavailable"
  end

  test "plans combine use cases without duplicate providers" do
    assert {:ok, plan} = Planner.plan(["cloud_team", "multi_device", "media_archive"])
    ids = Enum.map(plan.providers, & &1.id)

    assert length(ids) == length(Enum.uniq(ids))
    assert "postgres" in ids
    assert "s3" in ids
    assert "openbao" in ids
    assert "nats_jetstream" in ids
    assert "valkey" in ids
  end

  test "unknown use cases are rejected" do
    assert {:error, {:unknown_use_cases, ["made_up"]}} = Planner.plan(["made_up"])
  end
end
