defmodule OptimalEngine.HealthTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Health

  test "live?/0 returns true while the supervisor is running" do
    assert Health.live?()
  end

  test "ready/1 returns a status map with declared checks" do
    r = Health.ready(skip: [:embedder])
    assert is_map(r)
    assert Map.has_key?(r, :ok?)
    assert Map.has_key?(r, :checks)
    assert Map.has_key?(r, :degraded)

    # store + migrations should be :ok when supervisor is running
    assert r.checks.store == :ok
    assert r.checks.migrations == :ok
  end

  test "status/0 returns a known atom" do
    # Tests run with connectors registered but no CONNECTOR_KEY — the
    # credential check may be :error, so :down is a valid outcome here.
    assert Health.status() in [:up, :degraded, :down]
  end

  test "ready/1 honors the :skip option" do
    r = Health.ready(skip: [:embedder, :credential_key])
    refute Map.has_key?(r.checks, :embedder)
    refute Map.has_key?(r.checks, :credential_key)
  end
end
