defmodule OptimalEngine.MemoryCore.ClaimReviewTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.{Memory, MemoryCore, Store}
  alias OptimalEngine.MemoryCore.RetrievalCoordinator

  defp ws, do: "claim-review-#{System.unique_integer([:positive])}"

  test "pending claims from user-facing memory can be promoted to fact and memory object" do
    workspace_id = ws()
    content = "Claim review promotion test #{System.unique_integer([:positive])}"

    assert {:ok, _mem} =
             Memory.create(%{
               content: content,
               workspace_id: workspace_id,
               actor_id: "user:reviewer"
             })

    assert {:ok, [claim]} = MemoryCore.pending_claims(workspace_id: workspace_id)
    assert claim.claim_text == content
    assert claim.review_status == "unreviewed"

    assert {:ok, result} =
             MemoryCore.promote_claim(claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer",
               fact_text: "Accepted: #{content}",
               summary: "Remembered: #{content}",
               memory_type: "reviewed_note"
             )

    assert result.claim.review_status == "accepted"
    assert result.claim.lifecycle_state == "accepted"
    assert result.fact.fact_text == "Accepted: #{content}"
    assert result.memory_object.summary == "Remembered: #{content}"

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM facts WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, result.fact.id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM memory_objects WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, result.memory_object.id]
             )
  end

  test "rejected claims cannot be promoted" do
    workspace_id = ws()
    content = "Claim review rejection test #{System.unique_integer([:positive])}"

    assert {:ok, _mem} =
             Memory.create(%{
               content: content,
               workspace_id: workspace_id
             })

    assert {:ok, [claim]} = MemoryCore.pending_claims(workspace_id: workspace_id)

    assert {:ok, rejected} =
             MemoryCore.reject_claim(claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer"
             )

    assert rejected.review_status == "rejected"
    assert rejected.lifecycle_state == "rejected"

    assert {:error, :claim_rejected} =
             MemoryCore.promote_claim(claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer"
             )

    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [workspace_id])
  end

  test "review queue summarizes claims by review and lifecycle state" do
    workspace_id = ws()

    assert {:ok, _} =
             Memory.create(%{
               content: "Queue pending claim #{System.unique_integer([:positive])}",
               workspace_id: workspace_id
             })

    assert {:ok, _} =
             Memory.create(%{
               content: "Queue rejected claim #{System.unique_integer([:positive])}",
               workspace_id: workspace_id
             })

    assert {:ok, [claim_a, claim_b]} = MemoryCore.pending_claims(workspace_id: workspace_id)

    assert {:ok, _rejected} =
             MemoryCore.reject_claim(claim_b.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer"
             )

    assert {:ok, queue} = MemoryCore.claim_review_queue(workspace_id: workspace_id)

    assert queue.workspace_id == workspace_id
    assert queue.count == 2
    assert queue.review_counts["unreviewed"] == 1
    assert queue.review_counts["rejected"] == 1
    assert queue.lifecycle_counts["pending"] == 1
    assert queue.lifecycle_counts["rejected"] == 1
    assert Enum.any?(queue.claims, &(&1.id == claim_a.id))

    assert {:ok, pending_only} =
             MemoryCore.claim_review_queue(
               workspace_id: workspace_id,
               review_status: "unreviewed",
               lifecycle_state: "pending"
             )

    assert pending_only.count == 1
    assert hd(pending_only.claims).id == claim_a.id
  end

  test "stale claims are blocked from promotion by default" do
    workspace_id = ws()
    stale_after = DateTime.utc_now() |> DateTime.add(-60, :second) |> DateTime.to_iso8601()

    assert {:ok, claim} =
             extracted_claim(workspace_id,
               claim_text: "The deployment window was yesterday.",
               subject_anchor: "deployment",
               action_class: "scheduled_for",
               object_anchor: "yesterday",
               stale_after: stale_after
             )

    assert {:error, {:claim_stale, ^stale_after}} =
             MemoryCore.promote_claim(claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer"
             )

    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [workspace_id])
  end

  test "conflicting current facts require explicit supersession" do
    workspace_id = ws()

    assert {:ok, original_claim} =
             extracted_claim(workspace_id,
               claim_text: "The launch date is June 10.",
               subject_anchor: "project_launch",
               action_class: "date",
               object_anchor: "public_launch"
             )

    assert {:ok, original} =
             MemoryCore.promote_claim(original_claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer",
               fact_text: "The launch date is June 10.",
               summary: "Launch date is June 10."
             )

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               allowed_partitions: [],
               allowed_security_labels: []
             )

    assert package.refresh_state == "fresh"

    assert {:error, :context_package_not_stale} =
             MemoryCore.refresh_context_package(package.id, workspace_id: workspace_id)

    assert {:ok, replacement_claim} =
             extracted_claim(workspace_id,
               claim_text: "The launch date is June 17.",
               subject_anchor: "project_launch",
               action_class: "date",
               object_anchor: "public_launch"
             )

    assert {:error, {:contradicts_current_facts, [blocked_fact_id]}} =
             MemoryCore.promote_claim(replacement_claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer",
               fact_text: "The launch date is June 17."
             )

    assert blocked_fact_id == original.fact.id

    assert {:ok, replacement} =
             MemoryCore.promote_claim(replacement_claim.id,
               workspace_id: workspace_id,
               actor_id: "user:reviewer",
               fact_text: "The launch date is June 17.",
               summary: "Launch date moved to June 17.",
               supersedes_fact_id: original.fact.id,
               supersession_reason: "updated source evidence"
             )

    assert replacement.fact.lifecycle_state == "accepted"

    assert {:ok, [["superseded", "superseded"]]} =
             Store.raw_query(
               "SELECT lifecycle_state, contradiction_status FROM facts WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, original.fact.id]
             )

    assert {:ok, [["superseded", "stale", "superseded"]]} =
             Store.raw_query(
               "SELECT lifecycle_state, staleness_status, supersession_status FROM memory_objects WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, original.memory_object.id]
             )

    assert {:ok, [["stale", invalidation_reason]]} =
             Store.raw_query(
               "SELECT refresh_state, invalidation_reason FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, package.id]
             )

    assert invalidation_reason == "fact_superseded:#{original.fact.id}"

    assert {:ok, refreshed_package} =
             MemoryCore.refresh_context_package(package.id, workspace_id: workspace_id)

    assert refreshed_package.id != package.id
    assert refreshed_package.refresh_state == "fresh"
    assert refreshed_package.fact_links == [%{type: "fact", id: replacement.fact.id}]

    assert refreshed_package.memory_links == [
             %{type: "memory_object", id: replacement.memory_object.id}
           ]

    assert refreshed_package.metadata.refreshed_from_context_package_id == package.id
    assert refreshed_package.metadata.refreshed_from_invalidation_reason == invalidation_reason

    assert {:ok, [["refreshed"]]} =
             Store.raw_query(
               "SELECT refresh_state FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, package.id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM facts WHERE workspace_id = ?1 AND lifecycle_state = 'accepted'",
               [workspace_id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM relationship_edges WHERE workspace_id = ?1 AND relationship_type = 'supersedes' AND from_object_id = ?2 AND to_object_id = ?3",
               [workspace_id, replacement.fact.id, original.fact.id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM derivation_ledger WHERE workspace_id = ?1 AND activity_type = 'memory_core.supersede_fact'",
               [workspace_id]
             )
  end

  defp extracted_claim(workspace_id, opts) do
    source =
      MemoryCore.source_package_from_text(Keyword.fetch!(opts, :claim_text),
        workspace_id: workspace_id,
        source_type: "test_source",
        security_labels: ["internal"],
        partition_ids: ["test"]
      )

    MemoryCore.extract_claim(source, opts)
  end
end
