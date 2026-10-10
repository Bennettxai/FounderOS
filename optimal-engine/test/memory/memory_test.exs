defmodule OptimalEngine.MemoryTest do
  use ExUnit.Case, async: false

  setup do
    # Clean ETS tables between tests
    :ets.delete_all_objects(:optimal_engine_memory_store)
    :ets.delete_all_objects(:optimal_engine_memory_collections)
    :ok
  end

  describe "store/3" do
    test "stores a value and recalls it" do
      assert :ok = OptimalEngine.Memory.store("decisions", "d1", "Use ETS")
      assert {:ok, entry} = OptimalEngine.Memory.recall("decisions", "d1")
      assert entry.value == "Use ETS"
      assert entry.key == "d1"
    end

    test "stores with tags" do
      assert :ok =
               OptimalEngine.Memory.store("patterns", "p1", "GenServer", tags: ["elixir", "otp"])

      {:ok, entry} = OptimalEngine.Memory.recall("patterns", "p1")
      assert "elixir" in entry.metadata.tags
      assert "otp" in entry.metadata.tags
    end

    test "stores complex values" do
      value = %{title: "Decision", reason: "Performance", score: 9.5}
      assert :ok = OptimalEngine.Memory.store("decisions", "d2", value)
      {:ok, entry} = OptimalEngine.Memory.recall("decisions", "d2")
      assert entry.value == value
    end
  end

  describe "recall/2" do
    test "returns error for missing key" do
      assert {:error, :not_found} = OptimalEngine.Memory.recall("decisions", "nonexistent")
    end

    test "increments access count on recall" do
      OptimalEngine.Memory.store("stats", "s1", "data")
      {:ok, e1} = OptimalEngine.Memory.recall("stats", "s1")
      assert e1.metadata.access_count == 1
      {:ok, e2} = OptimalEngine.Memory.recall("stats", "s1")
      assert e2.metadata.access_count == 2
    end
  end

  describe "search/2" do
    test "finds entries by keyword in value" do
      OptimalEngine.Memory.store("tips", "t1", "Use pattern matching in Elixir")
      OptimalEngine.Memory.store("tips", "t2", "Go is fast for networking")

      {:ok, matches} = OptimalEngine.Memory.search("tips", "elixir")
      assert length(matches) == 1
      assert hd(matches).key == "t1"
    end

    test "finds entries by keyword in key" do
      OptimalEngine.Memory.store("tips", "elixir-tip", "Some tip")
      {:ok, matches} = OptimalEngine.Memory.search("tips", "elixir")
      assert length(matches) == 1
    end

    test "finds entries by tag" do
      OptimalEngine.Memory.store("tips", "t1", "tip", tags: ["rust"])
      {:ok, matches} = OptimalEngine.Memory.search("tips", "rust")
      assert length(matches) == 1
    end

    test "returns empty for no matches" do
      OptimalEngine.Memory.store("tips", "t1", "hello")
      {:ok, matches} = OptimalEngine.Memory.search("tips", "zzzznotfound")
      assert matches == []
    end
  end

  describe "forget/2" do
    test "deletes a memory entry" do
      OptimalEngine.Memory.store("tmp", "k", "v")
      assert {:ok, _} = OptimalEngine.Memory.recall("tmp", "k")
      assert :ok = OptimalEngine.Memory.forget("tmp", "k")
      assert {:error, :not_found} = OptimalEngine.Memory.recall("tmp", "k")
    end
  end

  describe "collections/0" do
    test "lists all collections" do
      OptimalEngine.Memory.store("alpha", "k", "v")
      OptimalEngine.Memory.store("beta", "k", "v")
      {:ok, cols} = OptimalEngine.Memory.collections()
      assert "alpha" in cols
      assert "beta" in cols
    end
  end

  describe "export/2 and import_collection/2" do
    @tag :tmp_dir
    test "round-trips a collection through JSON", %{tmp_dir: tmp_dir} do
      OptimalEngine.Memory.store("export_test", "k1", "value1", tags: ["tag1"])
      OptimalEngine.Memory.store("export_test", "k2", "value2")

      path = Path.join(tmp_dir, "export_test.json")
      assert :ok = OptimalEngine.Memory.export("export_test", path)
      assert File.exists?(path)

      # Clear and reimport
      OptimalEngine.Memory.forget("export_test", "k1")
      OptimalEngine.Memory.forget("export_test", "k2")

      assert :ok = OptimalEngine.Memory.import_collection("export_reimport", path)
      {:ok, entry} = OptimalEngine.Memory.recall("export_reimport", "k1")
      assert entry.value == "value1"
    end
  end
end
