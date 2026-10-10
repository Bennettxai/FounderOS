defmodule OptimalEngine.Memory.DedupTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.Versioned, as: Memory

  # Each test gets an isolated workspace so concurrent / sequential tests
  # don't pollute each other via the dedup index.
  defp ws, do: "dedup-ws-#{:erlang.unique_integer([:positive])}"

  # ---------------------------------------------------------------------------
  # Default policy: return_existing
  # ---------------------------------------------------------------------------

  describe "dedup — return_existing (default)" do
    test "same content + workspace + audience → second create returns first id" do
      workspace_id = ws()

      {:ok, first} =
        Memory.create(%{content: "hello world", workspace_id: workspace_id})

      {:ok, second} =
        Memory.create(%{content: "hello world", workspace_id: workspace_id})

      assert second.id == first.id
      assert second.was_existing == true
    end

    test "was_existing is false on a genuinely new memory" do
      workspace_id = ws()
      {:ok, mem} = Memory.create(%{content: "brand new memory", workspace_id: workspace_id})
      assert mem.was_existing == false
    end

    test "content is case- and whitespace-normalised before hashing" do
      workspace_id = ws()

      {:ok, first} =
        Memory.create(%{content: "Hello World", workspace_id: workspace_id})

      # Lowercased + leading/trailing whitespace stripped → same hash
      {:ok, second} =
        Memory.create(%{content: "  hello world  ", workspace_id: workspace_id})

      assert second.id == first.id
      assert second.was_existing == true
    end

    test "different audience → not deduplicated (separate rows)" do
      workspace_id = ws()

      {:ok, eng} =
        Memory.create(%{
          content: "shared content",
          workspace_id: workspace_id,
          audience: "engineering"
        })

      {:ok, sales} =
        Memory.create(%{
          content: "shared content",
          workspace_id: workspace_id,
          audience: "sales"
        })

      refute eng.id == sales.id
      assert eng.was_existing == false
      assert sales.was_existing == false
    end

    test "different workspace → not deduplicated (separate rows)" do
      ws1 = ws()
      ws2 = ws()

      {:ok, mem1} = Memory.create(%{content: "same content", workspace_id: ws1})
      {:ok, mem2} = Memory.create(%{content: "same content", workspace_id: ws2})

      refute mem1.id == mem2.id
      assert mem1.was_existing == false
      assert mem2.was_existing == false
    end

    test "same content but existing is_forgotten=true → new memory created" do
      workspace_id = ws()

      {:ok, first} = Memory.create(%{content: "to be forgotten", workspace_id: workspace_id})
      :ok = Memory.forget(first.id)

      # After forgetting, the dedup check should not find the forgotten row
      {:ok, second} = Memory.create(%{content: "to be forgotten", workspace_id: workspace_id})

      refute second.id == first.id
      assert second.was_existing == false
    end

    test "same content but existing is_latest=false → new memory created" do
      workspace_id = ws()

      {:ok, v1} = Memory.create(%{content: "versioned content", workspace_id: workspace_id})
      # Update creates v2, demoting v1 to is_latest=false
      {:ok, _v2} = Memory.update(v1.id, %{content: "versioned content updated"})

      # v1 is no longer latest — a fresh create with same content as v1 should insert new row
      {:ok, fresh} =
        Memory.create(%{content: "versioned content", workspace_id: workspace_id})

      # fresh is a new row (v1 is not live)
      assert fresh.was_existing == false
      refute fresh.id == v1.id
    end
  end

  # ---------------------------------------------------------------------------
  # Policy: always_insert
  # ---------------------------------------------------------------------------

  describe "dedup — always_insert policy" do
    test "same content with always_insert → both rows created" do
      workspace_id = ws()

      {:ok, first} =
        Memory.create(%{
          content: "duplicate me",
          workspace_id: workspace_id,
          dedup: "always_insert"
        })

      {:ok, second} =
        Memory.create(%{
          content: "duplicate me",
          workspace_id: workspace_id,
          dedup: "always_insert"
        })

      # Two distinct rows
      refute first.id == second.id
      assert first.was_existing == false
      assert second.was_existing == false
    end

    test "always_insert accepts atom policy" do
      workspace_id = ws()

      {:ok, first} =
        Memory.create(%{
          content: "atom policy",
          workspace_id: workspace_id,
          dedup: :always_insert
        })

      {:ok, second} =
        Memory.create(%{
          content: "atom policy",
          workspace_id: workspace_id,
          dedup: :always_insert
        })

      refute first.id == second.id
    end
  end

  # ---------------------------------------------------------------------------
  # Policy: bump_version
  # ---------------------------------------------------------------------------

  describe "dedup — bump_version policy" do
    test "same content with bump_version → existing gets updated to v2" do
      workspace_id = ws()

      {:ok, v1} =
        Memory.create(%{
          content: "bump this",
          workspace_id: workspace_id
        })

      assert v1.version == 1
      assert v1.is_latest == true

      {:ok, v2} =
        Memory.create(%{
          content: "bump this",
          workspace_id: workspace_id,
          dedup: "bump_version"
        })

      # v2 should be a new version of the same logical memory
      assert v2.version == 2
      assert v2.parent_memory_id == v1.id
      assert v2.is_latest == true

      # v1 should be demoted
      {:ok, v1_stale} = Memory.get(v1.id)
      assert v1_stale.is_latest == false
    end

    test "bump_version accepts atom policy" do
      workspace_id = ws()

      {:ok, v1} =
        Memory.create(%{content: "bump atom", workspace_id: workspace_id})

      {:ok, v2} =
        Memory.create(%{
          content: "bump atom",
          workspace_id: workspace_id,
          dedup: :bump_version
        })

      assert v2.version == 2
      assert v2.parent_memory_id == v1.id
    end
  end
end
