defmodule OptimalEngine.MemoryCore.PromotionSchedulerTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{Memory, Store}
  alias OptimalEngine.MemoryCore.PromotionScheduler

  defp ws, do: "promotion-sched-#{System.unique_integer([:positive])}"

  # Seed a genuine pending Claim through the real Source → Signal → Claim
  # pipeline (Memory.create), then set its confidence so the autonomous
  # scheduler's threshold gate is exercised deterministically.
  defp seed_claim(workspace_id, confidence) do
    content = "Promotion scheduler claim #{System.unique_integer([:positive])}"

    assert {:ok, _mem} =
             Memory.create(%{
               content: content,
               workspace_id: workspace_id,
               actor_id: "user:seed"
             })

    assert {:ok, [claim]} =
             OptimalEngine.MemoryCore.pending_claims(workspace_id: workspace_id)

    assert :ok =
             Store.raw_execute(
               "UPDATE claims SET aggregate_confidence = ?1 WHERE id = ?2",
               [confidence, claim.id]
             )

    %{claim | aggregate_confidence: confidence}
  end

  test "auto_threshold promotes a high-confidence pending claim to fact + memory object" do
    workspace_id = ws()
    high = seed_claim(workspace_id, 0.95)

    assert {:ok, summary} =
             PromotionScheduler.run_once(
               tenant_id: "default",
               workspace_ids: [workspace_id],
               mode: :auto_threshold
             )

    assert summary.mode == :auto_threshold
    assert summary.promoted_count == 1
    assert summary.error_count == 0
    assert [fact_id] = summary.promoted_fact_ids
    assert [memory_id] = summary.promoted_memory_object_ids

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM facts WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, fact_id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM memory_objects WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, memory_id]
             )

    assert {:ok, [[0]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM claims WHERE id = ?1 AND review_status = 'unreviewed'",
               [high.id]
             )
  end

  test "auto_threshold skips claims below the per-workspace threshold" do
    workspace_id = ws()
    _low = seed_claim(workspace_id, 0.50)

    assert {:ok, summary} =
             PromotionScheduler.run_once(
               tenant_id: "default",
               workspace_ids: [workspace_id],
               mode: :auto_threshold
             )

    assert summary.promoted_count == 0
    assert summary.skipped_count == 1

    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [workspace_id])
  end

  test "manual mode is a no-op even for high-confidence claims" do
    workspace_id = ws()
    _high = seed_claim(workspace_id, 0.99)

    assert {:ok, summary} =
             PromotionScheduler.run_once(
               tenant_id: "default",
               workspace_ids: [workspace_id],
               mode: :manual
             )

    assert summary.mode == :manual
    assert summary.promoted_count == 0

    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [workspace_id])
  end
end
