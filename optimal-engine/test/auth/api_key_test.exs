defmodule OptimalEngine.Auth.ApiKeyTest do
  @moduledoc """
  Unit tests for the ApiKey lifecycle: mint → verify → revoke → delete.
  bcrypt cost is set to 4 in test.exs so hashing is fast.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Auth.ApiKey

  # All tests use the "default" tenant which is seeded by migration 015.
  @tenant_id "default"

  describe "mint/1" do
    test "returns id, secret, and oe_-prefixed key" do
      assert {:ok, result} = ApiKey.mint(%{tenant_id: @tenant_id, name: "test key"})
      assert is_binary(result.id)
      assert is_binary(result.secret)
      assert String.starts_with?(result.key, "oe_")
      assert String.contains?(result.key, result.id)
      assert String.contains?(result.key, result.secret)
    end

    test "stores the key — list returns it without secret" do
      assert {:ok, %{id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "listable key"})
      assert {:ok, keys} = ApiKey.list(@tenant_id)
      ids = Enum.map(keys, & &1.id)
      assert id in ids
      # No key in the list struct has a hashed_secret field exposed
      key = Enum.find(keys, &(&1.id == id))
      refute Map.has_key?(key, :hashed_secret)
    end

    test "stores custom scopes and workspace_scope" do
      scopes = ["read:memory", "write:memory"]
      ws_scope = ["default:engineering"]

      assert {:ok, %{id: id}} =
               ApiKey.mint(%{
                 tenant_id: @tenant_id,
                 name: "scoped key",
                 scopes: scopes,
                 workspace_scope: ws_scope
               })

      assert {:ok, keys} = ApiKey.list(@tenant_id)
      key = Enum.find(keys, &(&1.id == id))
      assert key.scopes == scopes
      assert key.workspace_scope == ws_scope
    end

    test "stores expires_at when provided" do
      future = "2099-01-01T00:00:00Z"

      assert {:ok, %{id: id}} =
               ApiKey.mint(%{
                 tenant_id: @tenant_id,
                 name: "expiring key",
                 expires_at: future
               })

      assert {:ok, keys} = ApiKey.list(@tenant_id)
      key = Enum.find(keys, &(&1.id == id))
      assert key.expires_at == future
    end
  end

  describe "verify/1" do
    test "returns {:ok, key} for a valid token" do
      assert {:ok, %{key: token, id: id}} =
               ApiKey.mint(%{tenant_id: @tenant_id, name: "verify ok"})

      assert {:ok, %ApiKey{id: ^id, tenant_id: @tenant_id}} = ApiKey.verify(token)
    end

    test "returns {:error, :invalid} for a garbage token" do
      assert {:error, :invalid} = ApiKey.verify("not-a-real-token")
    end

    test "returns {:error, :invalid} for oe_ prefix but wrong id length" do
      assert {:error, :invalid} = ApiKey.verify("oe_tooshort_secret")
    end

    test "returns {:error, :invalid} for a correct-format token with wrong secret" do
      assert {:ok, %{key: token}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "wrong secret"})
      # Replace the secret portion with gibberish
      [prefix_and_id | _] = String.split(token, "_", parts: 3)

      forged =
        prefix_and_id <> "_" <> String.slice(token, 0, 8) <> "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxfake"

      assert {:error, :invalid} = ApiKey.verify(forged)
    end

    test "returns {:error, :revoked} for a revoked key" do
      assert {:ok, %{id: id, key: token}} =
               ApiKey.mint(%{tenant_id: @tenant_id, name: "to revoke"})

      assert :ok = ApiKey.revoke(id)
      assert {:error, :revoked} = ApiKey.verify(token)
    end

    test "returns {:error, :expired} for an expired key" do
      past = "2000-01-01T00:00:00Z"

      assert {:ok, %{key: token}} =
               ApiKey.mint(%{tenant_id: @tenant_id, name: "expired key", expires_at: past})

      assert {:error, :expired} = ApiKey.verify(token)
    end

    test "accepts a key with a future expiry" do
      future = "2099-12-31T23:59:59Z"

      assert {:ok, %{key: token, id: id}} =
               ApiKey.mint(%{tenant_id: @tenant_id, name: "future expiry", expires_at: future})

      assert {:ok, %ApiKey{id: ^id}} = ApiKey.verify(token)
    end
  end

  describe "revoke/1" do
    test "soft-revokes a key — it no longer appears in list" do
      assert {:ok, %{id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "to be revoked"})
      assert :ok = ApiKey.revoke(id)
      assert {:ok, keys} = ApiKey.list(@tenant_id)
      ids = Enum.map(keys, & &1.id)
      refute id in ids
    end

    test "is idempotent — revoking twice returns :ok" do
      assert {:ok, %{id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "revoke twice"})
      assert :ok = ApiKey.revoke(id)
      assert :ok = ApiKey.revoke(id)
    end
  end

  describe "delete/1" do
    test "hard-deletes a key — verify returns :invalid afterward" do
      assert {:ok, %{id: id, key: token}} =
               ApiKey.mint(%{tenant_id: @tenant_id, name: "to delete"})

      assert :ok = ApiKey.delete(id)
      assert {:error, :invalid} = ApiKey.verify(token)
    end

    test "delete of non-existent key returns :ok (no-op)" do
      assert :ok = ApiKey.delete("nonexistentid1234567890ab")
    end
  end

  describe "record_usage/1" do
    test "is a no-op that does not crash" do
      assert {:ok, %{id: id}} = ApiKey.mint(%{tenant_id: @tenant_id, name: "usage tracking"})
      # Fire and forget — just assert it doesn't crash
      assert :ok = ApiKey.record_usage(id)
      # Give the Task a moment to settle (it's async)
      Process.sleep(50)
    end
  end
end
