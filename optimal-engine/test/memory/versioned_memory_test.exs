defmodule OptimalEngine.Memory.VersionedTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Versioned, as: Memory

  # Each test uses a fresh workspace_id to avoid cross-test pollution without
  # needing full DB teardown (SQLite in-process; other tests share the same DB).
  defp ws, do: "test-ws-#{:erlang.unique_integer([:positive])}"

  # ---------------------------------------------------------------------------
  # create/1
  # ---------------------------------------------------------------------------

  describe "create/1" do
    test "creates a memory with required content" do
      workspace_id = ws()
      assert {:ok, mem} = Memory.create(%{content: "hello world", workspace_id: workspace_id})

      assert mem.content == "hello world"
      assert mem.version == 1
      assert mem.is_latest == true
      assert mem.is_forgotten == false
      assert mem.is_static == false
      assert mem.audience == "default"
      assert String.starts_with?(mem.id, "mem_")
      assert mem.root_memory_id == mem.id
    end

    test "creates with optional fields" do
      workspace_id = ws()

      assert {:ok, mem} =
               Memory.create(%{
                 content: "citation memory",
                 workspace_id: workspace_id,
                 is_static: true,
                 audience: "engineering",
                 citation_uri: "https://example.com/doc",
                 source_chunk_id: "chunk:abc123",
                 metadata: %{"source" => "test"}
               })

      assert mem.is_static == true
      assert mem.audience == "engineering"
      assert mem.citation_uri == "https://example.com/doc"
      assert mem.source_chunk_id == "chunk:abc123"
      assert mem.metadata == %{"source" => "test"}
    end

    test "returns error when content is missing" do
      assert {:error, :missing_required_fields} = Memory.create(%{workspace_id: ws()})
    end

    test "returns error when content is empty string" do
      assert {:error, :missing_required_fields} = Memory.create(%{content: "", workspace_id: ws()})
    end

    test "assigns default workspace and tenant" do
      assert {:ok, mem} = Memory.create(%{content: "defaults"})
      assert mem.workspace_id == "default"
      assert mem.tenant_id == "default"
    end
  end

  # ---------------------------------------------------------------------------
  # get/1
  # ---------------------------------------------------------------------------

  describe "get/1" do
    test "retrieves a created memory by id" do
      workspace_id = ws()
      {:ok, created} = Memory.create(%{content: "findme", workspace_id: workspace_id})
      assert {:ok, fetched} = Memory.get(created.id)
      assert fetched.id == created.id
      assert fetched.content == "findme"
    end

    test "returns not_found for unknown id" do
      assert {:error, :not_found} = Memory.get("mem_does_not_exist")
    end

    test "deserializes booleans correctly from SQLite integers" do
      workspace_id = ws()

      {:ok, created} =
        Memory.create(%{content: "bool check", workspace_id: workspace_id, is_static: true})

      {:ok, fetched} = Memory.get(created.id)
      assert fetched.is_static == true
      assert fetched.is_forgotten == false
      assert fetched.is_latest == true
    end
  end

  # ---------------------------------------------------------------------------
  # list/1
  # ---------------------------------------------------------------------------

  describe "list/1" do
    test "returns only latest, non-forgotten memories in workspace by default" do
      workspace_id = ws()
      {:ok, m1} = Memory.create(%{content: "first", workspace_id: workspace_id})
      {:ok, m2} = Memory.create(%{content: "second", workspace_id: workspace_id})

      {:ok, ids} =
        Memory.list(workspace_id: workspace_id)
        |> then(fn {:ok, mems} -> {:ok, Enum.map(mems, & &1.id)} end)

      assert m1.id in ids
      assert m2.id in ids
    end

    test "filters by audience" do
      workspace_id = ws()

      {:ok, eng} =
        Memory.create(%{content: "eng", workspace_id: workspace_id, audience: "engineering"})

      {:ok, _sales} =
        Memory.create(%{content: "sales", workspace_id: workspace_id, audience: "sales"})

      {:ok, result} = Memory.list(workspace_id: workspace_id, audience: "engineering")
      ids = Enum.map(result, & &1.id)
      assert eng.id in ids
      refute Enum.any?(result, &(&1.audience == "sales"))
    end

    test "excludes forgotten memories by default" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "forget me", workspace_id: workspace_id})
      :ok = Memory.forget(mem.id)

      {:ok, result} = Memory.list(workspace_id: workspace_id)
      refute Enum.any?(result, &(&1.id == mem.id))
    end

    test "includes forgotten memories when include_forgotten: true" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "include forgotten", workspace_id: workspace_id})
      :ok = Memory.forget(mem.id)

      {:ok, result} = Memory.list(workspace_id: workspace_id, include_forgotten: true)
      assert Enum.any?(result, &(&1.id == mem.id))
    end

    test "excludes old versions by default" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1", workspace_id: workspace_id})
      {:ok, _v2} = Memory.update(v1.id, %{content: "v2"})

      {:ok, result} = Memory.list(workspace_id: workspace_id)
      refute Enum.any?(result, &(&1.id == v1.id))
    end

    test "includes old versions when include_old_versions: true" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1 old", workspace_id: workspace_id})
      {:ok, _v2} = Memory.update(v1.id, %{content: "v2 new"})

      {:ok, result} = Memory.list(workspace_id: workspace_id, include_old_versions: true)
      assert Enum.any?(result, &(&1.id == v1.id))
    end

    test "respects limit" do
      workspace_id = ws()
      for i <- 1..5, do: Memory.create(%{content: "mem #{i}", workspace_id: workspace_id})
      {:ok, result} = Memory.list(workspace_id: workspace_id, limit: 2)
      assert length(result) <= 2
    end

    test "workspace isolation — does not return memories from another workspace" do
      ws1 = ws()
      ws2 = ws()
      {:ok, mem_ws1} = Memory.create(%{content: "in ws1", workspace_id: ws1})
      {:ok, _mem_ws2} = Memory.create(%{content: "in ws2", workspace_id: ws2})

      {:ok, result} = Memory.list(workspace_id: ws1)
      ids = Enum.map(result, & &1.id)
      assert mem_ws1.id in ids
      refute Enum.any?(result, &(&1.workspace_id == ws2))
    end
  end

  # ---------------------------------------------------------------------------
  # update/2 — versioning
  # ---------------------------------------------------------------------------

  describe "update/2" do
    test "creates v2 with parent_memory_id set to v1's id" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "original", workspace_id: workspace_id})
      {:ok, v2} = Memory.update(v1.id, %{content: "updated"})

      assert v2.version == 2
      assert v2.parent_memory_id == v1.id
      assert v2.root_memory_id == v1.id
      assert v2.content == "updated"
      assert v2.is_latest == true
    end

    test "demotes v1 is_latest to false" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1", workspace_id: workspace_id})
      {:ok, _v2} = Memory.update(v1.id, %{content: "v2"})

      {:ok, stale_v1} = Memory.get(v1.id)
      assert stale_v1.is_latest == false
    end

    test "creates an :updates relation from v2 to v1" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1", workspace_id: workspace_id})
      {:ok, v2} = Memory.update(v1.id, %{content: "v2"})

      {:ok, rels} = Memory.relations(v2.id)
      assert Enum.any?(rels, &(&1.relation == :updates and &1.target_memory_id == v1.id))
    end

    test "chains version correctly across 3 updates" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1", workspace_id: workspace_id})
      {:ok, v2} = Memory.update(v1.id, %{content: "v2"})
      {:ok, v3} = Memory.update(v2.id, %{content: "v3"})

      assert v3.version == 3
      assert v3.parent_memory_id == v2.id
      assert v3.root_memory_id == v1.id
    end

    test "update inherits workspace, audience from parent when not specified" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1", workspace_id: workspace_id, audience: "eng"})
      {:ok, v2} = Memory.update(v1.id, %{content: "v2 content"})

      assert v2.audience == "eng"
      assert v2.workspace_id == workspace_id
    end
  end

  # ---------------------------------------------------------------------------
  # extend/2
  # ---------------------------------------------------------------------------

  describe "extend/2" do
    test "creates child memory with :extends relation" do
      workspace_id = ws()
      {:ok, parent} = Memory.create(%{content: "parent", workspace_id: workspace_id})
      {:ok, child} = Memory.extend(parent.id, %{content: "extended", workspace_id: workspace_id})

      {:ok, rels} = Memory.relations(child.id)
      assert Enum.any?(rels, &(&1.relation == :extends and &1.target_memory_id == parent.id))
    end

    test "source keeps is_latest = true after extend" do
      workspace_id = ws()
      {:ok, parent} = Memory.create(%{content: "parent", workspace_id: workspace_id})
      {:ok, _child} = Memory.extend(parent.id, %{content: "extension", workspace_id: workspace_id})

      {:ok, parent_after} = Memory.get(parent.id)
      assert parent_after.is_latest == true
    end

    test "child has is_latest = true" do
      workspace_id = ws()
      {:ok, parent} = Memory.create(%{content: "parent", workspace_id: workspace_id})
      {:ok, child} = Memory.extend(parent.id, %{content: "extension", workspace_id: workspace_id})

      {:ok, child_fetched} = Memory.get(child.id)
      assert child_fetched.is_latest == true
    end
  end

  # ---------------------------------------------------------------------------
  # derive/2
  # ---------------------------------------------------------------------------

  describe "derive/2" do
    test "creates derived memory with :derives relation" do
      workspace_id = ws()
      {:ok, source} = Memory.create(%{content: "source", workspace_id: workspace_id})

      {:ok, derived} =
        Memory.derive(source.id, %{content: "derived insight", workspace_id: workspace_id})

      {:ok, rels} = Memory.relations(derived.id)
      assert Enum.any?(rels, &(&1.relation == :derives and &1.target_memory_id == source.id))
    end

    test "source keeps is_latest after derive" do
      workspace_id = ws()
      {:ok, source} = Memory.create(%{content: "source data", workspace_id: workspace_id})
      {:ok, _derived} = Memory.derive(source.id, %{content: "derived", workspace_id: workspace_id})

      {:ok, source_after} = Memory.get(source.id)
      assert source_after.is_latest == true
    end
  end

  # ---------------------------------------------------------------------------
  # forget/2
  # ---------------------------------------------------------------------------

  describe "forget/2" do
    test "sets is_forgotten flag on the row" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "to forget", workspace_id: workspace_id})
      :ok = Memory.forget(mem.id)

      {:ok, fetched} = Memory.get(mem.id)
      assert fetched.is_forgotten == true
    end

    test "stores reason when provided" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "forget with reason", workspace_id: workspace_id})
      :ok = Memory.forget(mem.id, reason: "outdated")

      {:ok, fetched} = Memory.get(mem.id)
      assert fetched.forget_reason == "outdated"
    end

    test "stores forget_after when provided" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "timed forget", workspace_id: workspace_id})
      ts = "2030-01-01T00:00:00Z"
      :ok = Memory.forget(mem.id, forget_after: ts)

      {:ok, fetched} = Memory.get(mem.id)
      assert fetched.forget_after == ts
    end

    test "does not remove the row (soft delete only)" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "soft delete", workspace_id: workspace_id})
      :ok = Memory.forget(mem.id)

      assert {:ok, _} = Memory.get(mem.id)
    end

    test "list excludes forgotten by default but get still returns it" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "excluded from list", workspace_id: workspace_id})
      :ok = Memory.forget(mem.id)

      {:ok, list_result} = Memory.list(workspace_id: workspace_id)
      refute Enum.any?(list_result, &(&1.id == mem.id))

      assert {:ok, _} = Memory.get(mem.id)
    end
  end

  # ---------------------------------------------------------------------------
  # versions/1
  # ---------------------------------------------------------------------------

  describe "versions/1" do
    test "returns single-element chain for a memory with no updates" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "solo", workspace_id: workspace_id})
      {:ok, chain} = Memory.versions(mem.id)
      assert length(chain) == 1
      assert hd(chain).id == mem.id
    end

    test "returns chain ordered v1 → v2 → v3" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1", workspace_id: workspace_id})
      {:ok, v2} = Memory.update(v1.id, %{content: "v2"})
      {:ok, v3} = Memory.update(v2.id, %{content: "v3"})

      {:ok, chain} = Memory.versions(v3.id)
      versions = Enum.map(chain, & &1.version)
      assert versions == [1, 2, 3]
      ids = Enum.map(chain, & &1.id)
      assert ids == [v1.id, v2.id, v3.id]
    end

    test "versions/1 works when called on v1 of a chain" do
      workspace_id = ws()
      {:ok, v1} = Memory.create(%{content: "v1 root", workspace_id: workspace_id})
      {:ok, v2} = Memory.update(v1.id, %{content: "v2 update"})

      {:ok, chain_from_v1} = Memory.versions(v1.id)
      {:ok, chain_from_v2} = Memory.versions(v2.id)
      assert Enum.map(chain_from_v1, & &1.id) == Enum.map(chain_from_v2, & &1.id)
    end
  end

  # ---------------------------------------------------------------------------
  # relations/1
  # ---------------------------------------------------------------------------

  describe "relations/1" do
    test "returns empty list for a memory with no relations" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "isolated", workspace_id: workspace_id})
      {:ok, rels} = Memory.relations(mem.id)
      assert rels == []
    end

    test "returns outbound and inbound relations" do
      workspace_id = ws()
      {:ok, m1} = Memory.create(%{content: "m1", workspace_id: workspace_id})
      {:ok, m2} = Memory.create(%{content: "m2", workspace_id: workspace_id})
      {:ok, _} = Memory.update(m1.id, %{content: "m1 v2"})

      # m2 extends m1 — adds outbound rel from m2 to m1
      {:ok, child} = Memory.extend(m1.id, %{content: "extends m1", workspace_id: workspace_id})

      {:ok, rels_m1} = Memory.relations(m1.id)
      # m1 should appear as target in the :updates relation (v2 → m1)
      # and as target in the :extends relation (child → m1)
      relation_types = Enum.map(rels_m1, & &1.relation)
      assert :updates in relation_types or :extends in relation_types

      {:ok, rels_child} = Memory.relations(child.id)
      assert Enum.any?(rels_child, &(&1.relation == :extends and &1.direction == :outbound))

      _ = m2
    end

    test "relation direction is correct" do
      workspace_id = ws()
      {:ok, source} = Memory.create(%{content: "source", workspace_id: workspace_id})
      {:ok, derived} = Memory.derive(source.id, %{content: "derived", workspace_id: workspace_id})

      {:ok, source_rels} = Memory.relations(source.id)
      inbound = Enum.filter(source_rels, &(&1.direction == :inbound))
      assert Enum.any?(inbound, &(&1.relation == :derives))

      {:ok, derived_rels} = Memory.relations(derived.id)
      outbound = Enum.filter(derived_rels, &(&1.direction == :outbound))
      assert Enum.any?(outbound, &(&1.relation == :derives))
    end
  end

  # ---------------------------------------------------------------------------
  # delete/1 — hard delete
  # ---------------------------------------------------------------------------

  describe "delete/1" do
    test "hard delete removes the memory row" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "to delete", workspace_id: workspace_id})
      :ok = Memory.delete(mem.id)
      assert {:error, :not_found} = Memory.get(mem.id)
    end

    test "hard delete cascades to relation rows" do
      workspace_id = ws()
      {:ok, parent} = Memory.create(%{content: "parent", workspace_id: workspace_id})
      {:ok, child} = Memory.extend(parent.id, %{content: "child", workspace_id: workspace_id})

      # Verify relation exists before delete
      {:ok, rels_before} = Memory.relations(child.id)
      assert length(rels_before) > 0

      # Delete child — relation should cascade
      :ok = Memory.delete(child.id)

      # parent still exists, its relations should no longer include the deleted child
      {:ok, rels_after} = Memory.relations(parent.id)

      refute Enum.any?(
               rels_after,
               &(&1.source_memory_id == child.id or &1.target_memory_id == child.id)
             )
    end

    test "hard delete differs from soft forget — get returns not_found" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "contrast", workspace_id: workspace_id})

      # soft forget — row survives
      :ok = Memory.forget(mem.id)
      assert {:ok, _} = Memory.get(mem.id)

      # hard delete — row gone
      :ok = Memory.delete(mem.id)
      assert {:error, :not_found} = Memory.get(mem.id)
    end
  end
end
