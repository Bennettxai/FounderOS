defmodule OptimalEngine.Memory.Store.ETSTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Store.ETS

  setup do
    :ets.delete_all_objects(:optimal_engine_memory_store)
    :ets.delete_all_objects(:optimal_engine_memory_collections)
    :ok
  end

  describe "put/4 and get/2" do
    test "stores and retrieves an entry" do
      assert :ok = ETS.put("col", "key1", "value1", %{})
      assert {:ok, entry} = ETS.get("col", "key1")
      assert entry.key == "key1"
      assert entry.value == "value1"
      assert %DateTime{} = entry.metadata.created_at
      assert entry.metadata.access_count == 1
    end

    test "preserves custom metadata" do
      meta = %{tags: ["a", "b"]}
      ETS.put("col", "k", "v", meta)
      {:ok, entry} = ETS.get("col", "k")
      assert "a" in entry.metadata.tags
    end

    test "overwrites existing entry" do
      ETS.put("col", "k", "v1", %{})
      ETS.put("col", "k", "v2", %{})
      {:ok, entry} = ETS.get("col", "k")
      assert entry.value == "v2"
    end
  end

  describe "get/2" do
    test "returns not_found for missing key" do
      assert {:error, :not_found} = ETS.get("col", "missing")
    end

    test "increments access_count on each get" do
      ETS.put("col", "k", "v", %{})
      {:ok, e1} = ETS.get("col", "k")
      {:ok, e2} = ETS.get("col", "k")
      assert e2.metadata.access_count == e1.metadata.access_count + 1
    end
  end

  describe "delete/2" do
    test "removes an entry" do
      ETS.put("col", "k", "v", %{})
      assert :ok = ETS.delete("col", "k")
      assert {:error, :not_found} = ETS.get("col", "k")
    end

    test "is idempotent" do
      assert :ok = ETS.delete("col", "nonexistent")
    end
  end

  describe "list/2" do
    test "lists all entries in a collection" do
      ETS.put("col", "k1", "v1", %{})
      ETS.put("col", "k2", "v2", %{})
      ETS.put("other", "k3", "v3", %{})

      {:ok, entries} = ETS.list("col")
      assert length(entries) == 2
      keys = Enum.map(entries, & &1.key)
      assert "k1" in keys
      assert "k2" in keys
    end

    test "respects limit option" do
      for i <- 1..10, do: ETS.put("col", "k#{i}", "v#{i}", %{})
      {:ok, entries} = ETS.list("col", limit: 3)
      assert length(entries) == 3
    end

    test "returns empty for unknown collection" do
      {:ok, entries} = ETS.list("nonexistent")
      assert entries == []
    end
  end

  describe "search/2" do
    test "finds by value content" do
      ETS.put("col", "k1", "Elixir is great", %{})
      ETS.put("col", "k2", "Rust is fast", %{})

      {:ok, matches} = ETS.search("col", "elixir")
      assert length(matches) == 1
      assert hd(matches).key == "k1"
    end

    test "finds by key content" do
      ETS.put("col", "elixir-guide", "some content", %{})
      {:ok, matches} = ETS.search("col", "elixir")
      assert length(matches) == 1
    end

    test "finds by tags" do
      ETS.put("col", "k1", "content", %{tags: ["elixir", "otp"]})
      {:ok, matches} = ETS.search("col", "otp")
      assert length(matches) == 1
    end

    test "case-insensitive search" do
      ETS.put("col", "k1", "ELIXIR", %{})
      {:ok, matches} = ETS.search("col", "elixir")
      assert length(matches) == 1
    end

    test "multi-term search matches any term" do
      ETS.put("col", "k1", "Elixir guide", %{})
      ETS.put("col", "k2", "Rust guide", %{})
      ETS.put("col", "k3", "Python guide", %{})

      {:ok, matches} = ETS.search("col", "elixir rust")
      keys = Enum.map(matches, & &1.key)
      assert "k1" in keys
      assert "k2" in keys
      refute "k3" in keys
    end
  end

  describe "collections/0" do
    test "lists all collection names" do
      ETS.put("alpha", "k", "v", %{})
      ETS.put("beta", "k", "v", %{})

      {:ok, cols} = ETS.collections()
      assert "alpha" in cols
      assert "beta" in cols
    end

    test "returns empty when no collections exist" do
      {:ok, cols} = ETS.collections()
      assert cols == []
    end
  end
end
