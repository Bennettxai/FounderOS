defmodule OptimalEngine.MemoryCoreBridgeTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{Memory, Store}

  defp ws, do: "memory-core-bridge-#{System.unique_integer([:positive])}"

  test "versioned memory creation also records source evidence and pending claim" do
    workspace_id = ws()

    content =
      "Bridge this memory into governed pending claims #{System.unique_integer([:positive])}"

    assert {:ok, mem} =
             Memory.create(%{
               content: content,
               workspace_id: workspace_id,
               audience: "engineering",
               metadata: %{source: "bridge-test"},
               actor_id: "user:test"
             })

    assert mem.content == content

    assert {:ok, [[source_id, source_uri, source_metadata]]} =
             Store.raw_query(
               """
               SELECT id, source_uri, metadata
               FROM source_packages
               WHERE workspace_id = ?1 AND raw_text = ?2 AND source_type = 'versioned_memory'
               """,
               [workspace_id, content]
             )

    assert source_uri == "memory://#{mem.id}"
    assert Jason.decode!(source_metadata)["versioned_memory_id"] == mem.id

    assert {:ok, [[claim_id, review_status, lifecycle_state, claim_metadata]]} =
             Store.raw_query(
               """
               SELECT id, review_status, lifecycle_state, metadata
               FROM claims
               WHERE workspace_id = ?1 AND source_package_id = ?2 AND claim_text = ?3
               """,
               [workspace_id, source_id, content]
             )

    assert review_status == "unreviewed"
    assert lifecycle_state == "pending"
    assert Jason.decode!(claim_metadata)["versioned_memory_id"] == mem.id

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM relationship_edges
               WHERE workspace_id = ?1
                 AND from_object_type = 'source_package'
                 AND from_object_id = ?2
                 AND to_object_type = 'claim'
                 AND to_object_id = ?3
                 AND relationship_type = 'supports'
               """,
               [workspace_id, source_id, claim_id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type = 'memory_core.extract_claim'
                 AND derivation_stage = 'source_package_to_claim'
               """,
               [workspace_id]
             )
  end

  test "versioned memory bridge does not auto-promote claims into facts" do
    workspace_id = ws()
    content = "This should stay pending until reviewed #{System.unique_integer([:positive])}"

    assert {:ok, _mem} =
             Memory.create(%{
               content: content,
               workspace_id: workspace_id
             })

    assert {:ok, [[1]]} =
             Store.raw_query("SELECT COUNT(*) FROM claims WHERE workspace_id = ?1", [workspace_id])

    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [workspace_id])

    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM memory_objects WHERE workspace_id = ?1", [
               workspace_id
             ])
  end
end
