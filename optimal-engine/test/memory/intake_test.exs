defmodule OptimalEngine.Memory.IntakeTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Memory

  defp ws, do: "test-intake-#{:erlang.unique_integer([:positive])}"

  describe "remember/2 governed intake" do
    test "preserves low-salience filler as rejected source evidence" do
      workspace_id = ws()

      assert {:ok, result} = Memory.remember(%{content: "ok", workspace_id: workspace_id})

      assert result.action == :rejected
      assert result.memory == nil
      assert result.source_package.quarantine_state == "rejected"
      assert result.source_package.raw_text == "ok"
      assert result.pending_claim == nil
      assert result.gate.should_encode == false

      assert get_in(result.source_package.metadata, ["memory_intake", "path"]) ==
               "rejected_source_package"

      assert {:ok, [[1]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT COUNT(*) FROM source_packages WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, result.source_package.id]
               )

      assert {:ok, [[1]]} =
               OptimalEngine.Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1
                   AND activity_type = 'memory.remember'
                   AND derivation_stage = 'source_rejected'
                 """,
                 [workspace_id]
               )
    end

    test "salient memory input becomes a pending source-backed claim" do
      workspace_id = ws()

      assert {:ok, result} =
               Memory.remember(%{
                 content: "Decision: Finance lead owns pricing renewal for Q4 at $2000.",
                 workspace_id: workspace_id
               })

      assert result.action == :pending_claim
      assert result.gate.should_encode == true
      assert result.memory == nil
      assert result.source_package.workspace_id == workspace_id
      assert result.source_package.quarantine_state == "clear"
      assert result.pending_claim.workspace_id == workspace_id
      assert result.pending_claim.source_package_id == result.source_package.id
      assert result.pending_claim.claim_text =~ "Finance lead"

      assert get_in(result.source_package.metadata, ["memory_intake", "gate", "should_encode"]) ==
               true

      assert get_in(result.source_package.metadata, ["memory_intake", "path"]) ==
               "memory_core_pending_claim"

      # The source -> claim lifecycle transition must be ledgered.
      assert {:ok, [[1]]} =
               OptimalEngine.Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1
                   AND activity_type = 'memory_core.extract_claim'
                   AND derivation_stage = 'source_package_to_claim'
                 """,
                 [workspace_id]
               )
    end

    test "force bypass still goes through the governed source-first path" do
      workspace_id = ws()

      assert {:ok, result} =
               Memory.remember(%{content: "ok", workspace_id: workspace_id}, force: true)

      assert result.action == :pending_claim
      assert result.gate.should_encode == false
      assert result.memory == nil
      assert result.source_package.quarantine_state == "clear"
      assert result.pending_claim.source_package_id == result.source_package.id
      assert result.pending_claim.claim_text == "ok"
      assert [] = Memory.list(workspace_id: workspace_id, limit: 10)
    end

    test "does not write versioned memories for duplicate governed input" do
      workspace_id = ws()

      assert {:ok, first} =
               Memory.remember(%{
                 content: "Decision: Finance lead owns pricing renewal for Q4 at $2000.",
                 workspace_id: workspace_id
               })

      assert {:ok, second} =
               Memory.remember(%{
                 content: "Decision: Finance lead owns pricing renewal for Q4 at $2000.",
                 workspace_id: workspace_id
               })

      assert first.action == :pending_claim
      assert second.action == :pending_claim
      assert first.source_package.id == second.source_package.id
      assert first.pending_claim.id == second.pending_claim.id
      assert first.memory == nil
      assert second.memory == nil

      assert [] = Memory.list(workspace_id: workspace_id, limit: 10)
    end

    test "corrections are pending claims, not direct memory updates" do
      workspace_id = ws()

      assert {:ok, first} =
               Memory.remember(%{
                 content: "Decision: Finance lead owns pricing renewal for Q4.",
                 workspace_id: workspace_id
               })

      assert {:ok, second} =
               Memory.remember(%{
                 content:
                   "Correction: Operations lead now owns pricing renewal for Q4 instead of Finance lead.",
                 workspace_id: workspace_id
               })

      assert first.action == :pending_claim
      assert second.action == :pending_claim
      assert first.pending_claim.id != second.pending_claim.id
      assert second.pending_claim.claim_text =~ "Operations lead"
      assert [] = Memory.list(workspace_id: workspace_id, limit: 10)
    end
  end
end
