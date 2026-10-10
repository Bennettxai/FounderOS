defmodule OptimalEngine.MemoryCore.TruthLifecycleTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.Claim
  alias OptimalEngine.MemoryCore.ClaimExtractor
  alias OptimalEngine.MemoryCore.Fact
  alias OptimalEngine.MemoryCore.FactPromoter
  alias OptimalEngine.MemoryCore.KnowledgeLifecycle
  alias OptimalEngine.MemoryCore.MemoryObject
  alias OptimalEngine.MemoryCore.RelationshipEdge
  alias OptimalEngine.MemoryCore.ScoringPolicy
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Store

  defp unique_workspace(prefix), do: "#{prefix}-#{System.unique_integer([:positive])}"

  defp build_source(workspace_id, raw_text, opts \\ []) do
    SourcePackage.from_text(
      raw_text,
      Keyword.merge(
        [
          workspace_id: workspace_id,
          source_type: "meeting_note",
          trust_label: "unreviewed",
          security_labels: ["internal"],
          partition_ids: ["project-launch"]
        ],
        opts
      )
    )
  end

  defp extract_launch_claim(source, opts \\ []) do
    ClaimExtractor.extract_from_source(
      source,
      Keyword.merge(
        [
          claim_text: "The source states that the project launch was approved.",
          subject_anchor: "project_launch",
          action_class: "approved",
          object_anchor: "planning_meeting",
          aggregate_confidence: 0.72,
          aggregate_precision: 0.68,
          actor_id: "agent:test"
        ],
        opts
      )
    )
  end

  describe "ClaimExtractor.extract_from_source/2" do
    test "returns a pending typed Claim with contract aliases in sync" do
      workspace_id = unique_workspace("truth-extract")
      source = build_source(workspace_id, "The project launch was approved.")

      assert {:ok, %Claim{} = claim} = extract_launch_claim(source)

      assert claim.workspace_id == workspace_id
      assert claim.tenant_id == "default"
      assert claim.source_package_id == source.id
      assert claim.status == "pending"
      assert claim.lifecycle_state == "pending"
      assert claim.review_status == "unreviewed"
      assert claim.subject == "project_launch"
      assert claim.subject == claim.subject_anchor
      assert claim.predicate == "approved"
      assert claim.predicate == claim.action_class
      assert claim.object_value == "planning_meeting"
      assert claim.object_value == claim.object_anchor
      assert claim.confidence == 0.72
      assert claim.confidence == claim.aggregate_confidence
      assert claim.aggregate_precision == 0.68
      assert claim.extracted_by == "agent:test"

      assert {:ok, [[1]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM claims WHERE workspace_id = ?1 AND id = ?2 AND lifecycle_state = 'pending'",
                 [workspace_id, claim.id]
               )

      assert {:ok, [[version]]} =
               Store.raw_query(
                 """
                 SELECT scoring_policy_version FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.extract_claim'
                 """,
                 [workspace_id]
               )

      assert version == ScoringPolicy.version()
    end
  end

  describe "ScoringPolicy" do
    test "keeps confidence and precision as separate versioned scores" do
      assert ScoringPolicy.version() =~ "scoring_policy"

      source = build_source(unique_workspace("truth-scoring"), "Launch was approved.")

      bare = ScoringPolicy.claim_scores(source, [])
      assert_in_delta bare.confidence, 0.55, 0.0001
      assert_in_delta bare.precision, 0.55, 0.0001

      anchored =
        ScoringPolicy.claim_scores(source,
          subject_anchor: "project_launch",
          action_class: "approved",
          object_anchor: "planning_meeting"
        )

      assert_in_delta anchored.confidence, 0.55, 0.0001
      assert_in_delta anchored.precision, 0.70, 0.0001
      assert anchored.precision != anchored.confidence
      assert anchored.raw_component_scores["anchor_count"] == 3

      claim = Claim.new(%{aggregate_confidence: 0.6, aggregate_precision: 0.7})
      promoted = ScoringPolicy.promotion_scores(claim, aggregate_confidence: 0.9)
      assert_in_delta promoted.confidence, 0.9, 0.0001
      assert_in_delta promoted.precision, 0.7, 0.0001
      assert_in_delta promoted.confidence_delta, 0.3, 0.0001
      assert_in_delta promoted.precision_delta, 0.0, 0.0001
    end

    test "ignores caller threshold overrides and exposes the effective threshold" do
      low = Claim.new(%{aggregate_confidence: 0.5, aggregate_precision: 0.5})

      # A caller-supplied :auto_promote_threshold must NOT lower the gate.
      assert {:error, :below_auto_promote_threshold} =
               ScoringPolicy.review_decision(low, policy: :auto, auto_promote_threshold: 0.0)

      high = Claim.new(%{aggregate_confidence: 0.9, aggregate_precision: 0.8})

      assert {:ok, review} = ScoringPolicy.review_decision(high, policy: :auto)
      assert review.review_basis == "policy_auto_promotion"
      assert review.verifier_id == "policy:#{ScoringPolicy.version()}"
      assert review.effective_threshold == ScoringPolicy.auto_promote_threshold()
      assert review.claim_confidence == 0.9

      assert {:ok, %{effective_threshold: threshold, claim_confidence: 0.9}} =
               ScoringPolicy.review_decision(high, verifier_id: "human:reviewer")

      assert threshold == ScoringPolicy.auto_promote_threshold()
    end

    test "rejects reviewer self-approval unless explicitly allowed" do
      claim = Claim.new(%{aggregate_confidence: 0.9, evaluator_id: "agent:author"})

      assert {:error, :self_review_not_allowed} =
               ScoringPolicy.review_decision(claim, verifier_id: "agent:author")

      assert {:error, :self_review_not_allowed} =
               ScoringPolicy.review_decision(claim, actor_id: "agent:author")

      assert {:ok, %{verifier_id: "agent:author", review_basis: "actor_approval"}} =
               ScoringPolicy.review_decision(claim,
                 verifier_id: "agent:author",
                 allow_self_review: true
               )

      assert {:ok, %{verifier_id: "human:reviewer", review_basis: "actor_approval"}} =
               ScoringPolicy.review_decision(claim, verifier_id: "human:reviewer")
    end
  end

  describe "FactPromoter.promote/2 evidence review" do
    test "refuses promotion without actor approval or policy opt-in" do
      workspace_id = unique_workspace("truth-review")
      source = build_source(workspace_id, "Launch approval pending review #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:error, :approval_required} = FactPromoter.promote(claim, [])

      assert {:error, :below_auto_promote_threshold} =
               FactPromoter.promote(claim, policy: :auto)

      assert {:ok, [[0]]} =
               Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [
                 workspace_id
               ])
    end

    test "refuses promotion when a claim has no source package lineage" do
      workspace_id = unique_workspace("truth-lineage")

      bare_claim =
        Claim.new(%{
          id: "claim_without_source",
          tenant_id: "default",
          workspace_id: workspace_id,
          source_package_id: nil,
          claim_text: "A bare claim should never become a Fact.",
          aggregate_confidence: 0.95,
          aggregate_precision: 0.9
        })

      assert {:error, :source_lineage_required} =
               FactPromoter.promote(bare_claim, verifier_id: "human:reviewer")

      assert {:ok, [[0]]} =
               Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [
                 workspace_id
               ])
    end

    test "refuses promotion of a claim that was never persisted" do
      workspace_id = unique_workspace("truth-unpersisted")

      unpersisted =
        Claim.new(%{
          id: "claim_never_persisted",
          tenant_id: "default",
          workspace_id: workspace_id,
          source_package_id: "sp_fake",
          claim_text: "An unpersisted claim must not become a Fact.",
          aggregate_confidence: 0.95,
          aggregate_precision: 0.9
        })

      assert {:error, :claim_not_found} =
               FactPromoter.promote(unpersisted, verifier_id: "human:reviewer")

      # The by-ID path fails the same way.
      assert {:error, :claim_not_found} =
               FactPromoter.promote("claim_never_persisted",
                 workspace_id: workspace_id,
                 verifier_id: "human:reviewer"
               )

      assert {:ok, [[0]]} =
               Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [
                 workspace_id
               ])
    end

    test "auto-promotes through policy above the confidence threshold" do
      workspace_id = unique_workspace("truth-auto")
      source = build_source(workspace_id, "Launch approval high confidence #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source, aggregate_confidence: 0.9)

      assert {:ok, %Fact{} = fact} = FactPromoter.promote(claim, policy: :auto)
      assert fact.verifier_id == "policy:#{ScoringPolicy.version()}"

      assert {:ok, [["policy_auto_promotion"]]} =
               Store.raw_query(
                 """
                 SELECT json_extract(metadata, '$.review_basis') FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.promote_fact'
                 """,
                 [workspace_id]
               )
    end

    test "records bitemporal state and transitions the claim to promoted" do
      workspace_id = unique_workspace("truth-promote")
      source = build_source(workspace_id, "Launch approval bitemporal #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:ok, %Fact{} = fact} =
               FactPromoter.promote(claim,
                 verifier_id: "human:reviewer",
                 aggregate_confidence: 0.9,
                 aggregate_precision: 0.86
               )

      assert fact.lifecycle_state == "accepted"
      assert fact.verification_status == "verified"
      assert fact.verifier_id == "human:reviewer"
      assert fact.accepted_claim_ids == [claim.id]
      assert is_binary(fact.valid_time_start)
      assert is_binary(fact.transaction_time_start)
      assert is_binary(fact.verification_time)
      assert is_nil(fact.transaction_time_end)
      assert fact.aggregate_confidence == 0.9
      assert fact.aggregate_precision == 0.86

      assert {:ok, [["promoted", "approved"]]} =
               Store.raw_query(
                 "SELECT lifecycle_state, review_status FROM claims WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, claim.id]
               )

      assert {:ok, [%RelationshipEdge{relationship_type: "supports"}]} =
               MemoryCoreStore.list_relationship_edges(workspace_id,
                 from_object_id: claim.id,
                 to_object_id: fact.id
               )

      assert {:ok, [[ScoringPolicy.version()]]} ==
               Store.raw_query(
                 """
                 SELECT scoring_policy_version FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.promote_fact'
                 """,
                 [workspace_id]
               )

      # The promotion ledger records the effective review threshold and the
      # claim confidence that produced the decision, for replay.
      assert {:ok, [[threshold, claim_confidence]]} =
               Store.raw_query(
                 """
                 SELECT json_extract(metadata, '$.effective_threshold'),
                        json_extract(metadata, '$.claim_confidence')
                 FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.promote_fact'
                 """,
                 [workspace_id]
               )

      assert threshold == ScoringPolicy.auto_promote_threshold()
      assert claim_confidence == claim.aggregate_confidence
    end

    test "promotes a pending claim loaded by ID (review queue path)" do
      workspace_id = unique_workspace("truth-queue")
      source = build_source(workspace_id, "Launch approval queue #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:error, :workspace_id_required} =
               FactPromoter.promote(claim.id, verifier_id: "human:reviewer")

      assert {:ok, %Fact{} = fact} =
               FactPromoter.promote(claim.id,
                 workspace_id: workspace_id,
                 verifier_id: "human:reviewer"
               )

      assert fact.accepted_claim_ids == [claim.id]

      assert {:error, :claim_already_promoted} =
               FactPromoter.promote(claim.id,
                 workspace_id: workspace_id,
                 verifier_id: "human:reviewer"
               )
    end
  end

  describe "FactPromoter.reject/2" do
    test "closes the review loop with a rejected claim and ledger entry" do
      workspace_id = unique_workspace("truth-reject")
      source = build_source(workspace_id, "Launch approval rejected #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:error, :reviewer_required} = FactPromoter.reject(claim, [])

      assert {:ok, %Claim{} = rejected} =
               FactPromoter.reject(claim, verifier_id: "human:reviewer", reason: "no evidence")

      assert rejected.status == "rejected"
      assert rejected.review_status == "rejected"

      assert {:ok, [["rejected", "rejected"]]} =
               Store.raw_query(
                 "SELECT lifecycle_state, review_status FROM claims WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, claim.id]
               )

      assert {:ok, [[1]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.reject_claim'
                   AND derivation_stage = 'claim_review'
                 """,
                 [workspace_id]
               )

      assert {:error, :claim_rejected} =
               FactPromoter.promote(claim.id,
                 workspace_id: workspace_id,
                 verifier_id: "human:reviewer"
               )
    end
  end

  describe "stale-struct review hardening" do
    test "double-promoting a stale claim struct is rejected" do
      workspace_id = unique_workspace("truth-stale-promote")
      source = build_source(workspace_id, "Launch approval stale promote #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:ok, %Fact{}} = FactPromoter.promote(claim, verifier_id: "human:reviewer")

      # The caller's struct is now stale (still says "pending"), but review
      # state must be evaluated from the persisted row.
      assert claim.lifecycle_state == "pending"

      assert {:error, :claim_already_promoted} =
               FactPromoter.promote(claim, verifier_id: "human:reviewer")

      assert {:ok, [[1]]} =
               Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [
                 workspace_id
               ])

      assert {:ok, [[1]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.promote_fact'
                 """,
                 [workspace_id]
               )
    end

    test "rejecting a stale claim struct after promotion is refused" do
      workspace_id = unique_workspace("truth-stale-reject")
      source = build_source(workspace_id, "Launch approval stale reject #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:ok, %Fact{} = fact} = FactPromoter.promote(claim, verifier_id: "human:reviewer")
      assert claim.lifecycle_state == "pending"

      assert {:error, :claim_already_promoted} =
               FactPromoter.reject(claim, verifier_id: "human:reviewer", reason: "stale struct")

      # The accepted Fact must never end up pointing at a rejected claim.
      assert {:ok, [["promoted", "approved"]]} =
               Store.raw_query(
                 "SELECT lifecycle_state, review_status FROM claims WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, claim.id]
               )

      assert {:ok, %Fact{lifecycle_state: "accepted"}} =
               MemoryCoreStore.get_fact(workspace_id, fact.id)

      assert {:ok, [[0]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.reject_claim'
                 """,
                 [workspace_id]
               )
    end

    test "rejecting an unpersisted claim struct is refused" do
      workspace_id = unique_workspace("truth-stale-missing")

      unpersisted =
        Claim.new(%{
          id: "claim_reject_missing",
          tenant_id: "default",
          workspace_id: workspace_id,
          source_package_id: "sp_fake",
          claim_text: "Never persisted."
        })

      assert {:error, :claim_not_found} =
               FactPromoter.reject(unpersisted, verifier_id: "human:reviewer")
    end

    test "re-rejecting an already reviewed claim returns :claim_already_reviewed" do
      workspace_id = unique_workspace("truth-stale-rereject")
      source = build_source(workspace_id, "Launch approval re-reject #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:ok, %Claim{}} = FactPromoter.reject(claim, verifier_id: "human:reviewer")

      assert {:error, :claim_already_reviewed} =
               FactPromoter.reject(claim, verifier_id: "human:reviewer")

      # The conditional review update guards the Store layer directly too.
      assert {:error, :claim_already_reviewed} =
               MemoryCoreStore.update_claim_review(workspace_id, claim.id, "promoted", "approved")

      assert {:ok, [["rejected", "rejected"]]} =
               Store.raw_query(
                 "SELECT lifecycle_state, review_status FROM claims WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, claim.id]
               )
    end
  end

  describe "supersession" do
    test "promoting an updated fact supersedes the prior fact and its memory objects" do
      workspace_id = unique_workspace("truth-supersede")

      old_source = build_source(workspace_id, "Launch was approved for June #{workspace_id}.")

      {:ok, old_claim} =
        extract_launch_claim(old_source, claim_text: "The launch was approved for June.")

      {:ok, old_fact} = FactPromoter.promote(old_claim, verifier_id: "human:reviewer")
      {:ok, old_memory} = MemoryObject.build_from_fact(old_fact, memory_type: "decision")

      new_source = build_source(workspace_id, "Launch was moved to July #{workspace_id}.")

      {:ok, new_claim} =
        extract_launch_claim(new_source, claim_text: "The launch was moved to July.")

      assert {:ok, %Fact{} = new_fact} =
               FactPromoter.promote(new_claim, verifier_id: "human:reviewer")

      assert new_fact.supersedes == [old_fact.id]
      assert new_fact.metadata["supersedes"] == [old_fact.id]

      assert {:ok, %Fact{} = stored_old_fact} =
               MemoryCoreStore.get_fact(workspace_id, old_fact.id)

      assert stored_old_fact.lifecycle_state == "superseded"
      assert stored_old_fact.superseded_by == new_fact.id
      assert is_binary(stored_old_fact.transaction_time_end)
      assert is_binary(stored_old_fact.valid_time_end)

      assert {:ok, [%RelationshipEdge{} = edge]} =
               MemoryCoreStore.list_relationship_edges(workspace_id,
                 relationship_type: "supersedes"
               )

      assert edge.from_object_id == new_fact.id
      assert edge.to_object_id == old_fact.id

      assert {:ok, %MemoryObject{} = stored_old_memory} =
               MemoryCoreStore.get_memory_object(workspace_id, old_memory.id)

      assert stored_old_memory.supersession_status == "superseded"
      assert is_binary(stored_old_memory.transaction_time_end)

      assert {:ok, [[1]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.supersede_fact'
                   AND derivation_stage = 'fact_supersession'
                 """,
                 [workspace_id]
               )

      assert {:ok, [%Fact{id: open_fact_id}]} =
               MemoryCoreStore.list_facts(workspace_id,
                 lifecycle_state: "accepted",
                 open_transaction_only: true
               )

      assert open_fact_id == new_fact.id
    end

    test "explicit supersedes option closes a named fact without anchor matching" do
      workspace_id = unique_workspace("truth-supersede-explicit")

      old_source = build_source(workspace_id, "Budget cap is 10k #{workspace_id}.")

      {:ok, old_claim} =
        extract_launch_claim(old_source,
          claim_text: "Budget cap is 10k.",
          subject_anchor: "budget",
          action_class: "capped",
          object_anchor: nil
        )

      {:ok, old_fact} = FactPromoter.promote(old_claim, verifier_id: "human:reviewer")

      new_source = build_source(workspace_id, "Budget cap raised to 20k #{workspace_id}.")

      {:ok, new_claim} =
        extract_launch_claim(new_source,
          claim_text: "Budget cap is 20k.",
          subject_anchor: nil,
          action_class: nil,
          object_anchor: nil
        )

      assert {:ok, %Fact{} = new_fact} =
               FactPromoter.promote(new_claim,
                 verifier_id: "human:reviewer",
                 supersedes: old_fact.id
               )

      assert new_fact.supersedes == [old_fact.id]

      assert {:ok, %Fact{lifecycle_state: "superseded"}} =
               MemoryCoreStore.get_fact(workspace_id, old_fact.id)
    end

    test "explicit supersession cannot re-close an already superseded fact" do
      workspace_id = unique_workspace("truth-supersede-closed")

      old_source = build_source(workspace_id, "Budget cap is 10k #{workspace_id}.")

      {:ok, old_claim} =
        extract_launch_claim(old_source,
          claim_text: "Budget cap is 10k.",
          subject_anchor: "budget",
          action_class: "capped",
          object_anchor: nil
        )

      {:ok, old_fact} = FactPromoter.promote(old_claim, verifier_id: "human:reviewer")

      mid_source = build_source(workspace_id, "Budget cap is 20k #{workspace_id}.")

      {:ok, mid_claim} =
        extract_launch_claim(mid_source,
          claim_text: "Budget cap is 20k.",
          subject_anchor: "budget",
          action_class: "capped",
          object_anchor: nil
        )

      {:ok, mid_fact} = FactPromoter.promote(mid_claim, verifier_id: "human:reviewer")

      assert {:ok, %Fact{} = closed_old} = MemoryCoreStore.get_fact(workspace_id, old_fact.id)
      assert closed_old.lifecycle_state == "superseded"
      assert closed_old.superseded_by == mid_fact.id
      closed_transaction_end = closed_old.transaction_time_end
      assert is_binary(closed_transaction_end)

      late_source = build_source(workspace_id, "Budget cap is 30k #{workspace_id}.")

      {:ok, late_claim} =
        extract_launch_claim(late_source,
          claim_text: "Budget cap is 30k.",
          subject_anchor: nil,
          action_class: nil,
          object_anchor: nil
        )

      # Explicitly naming the already-closed fact must NOT re-close it or
      # rewrite its supersession lineage: closed bitemporal versions are
      # immutable.
      assert {:ok, %Fact{} = late_fact} =
               FactPromoter.promote(late_claim,
                 verifier_id: "human:reviewer",
                 supersedes: old_fact.id
               )

      assert late_fact.supersedes == []

      assert {:ok, %Fact{} = reread_old} = MemoryCoreStore.get_fact(workspace_id, old_fact.id)
      assert reread_old.lifecycle_state == "superseded"
      assert reread_old.superseded_by == mid_fact.id
      assert reread_old.transaction_time_end == closed_transaction_end
    end
  end

  describe "MemoryObject.build_from_fact/2" do
    test "writes source lineage links, derived_from edge, and ledger entry" do
      workspace_id = unique_workspace("truth-memory")
      source = build_source(workspace_id, "Launch approval memory #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)
      {:ok, fact} = FactPromoter.promote(claim, verifier_id: "human:reviewer")

      assert {:ok, %MemoryObject{} = memory} =
               MemoryObject.build_from_fact(fact,
                 summary: "Launch approval is an accepted memory.",
                 memory_type: "decision",
                 salience: 0.8
               )

      assert memory.fact_links == [%{type: "fact", id: fact.id}]
      assert memory.claim_links == [%{type: "claim", id: claim.id}]
      assert memory.source_package_links == [%{type: "source_package", id: source.id}]
      assert memory.salience == 0.8
      assert memory.lifecycle_state == "current"
      assert memory.supersession_status == "none"

      assert {:ok, [%RelationshipEdge{relationship_type: "derived_from"}]} =
               MemoryCoreStore.list_relationship_edges(workspace_id,
                 relationship_type: "derived_from",
                 from_object_id: memory.id,
                 to_object_id: fact.id
               )

      assert {:ok, [[1]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.build_memory_object'
                   AND scoring_policy_version = ?2
                 """,
                 [workspace_id, ScoringPolicy.version()]
               )
    end

    test "refuses to build memory from an unpersisted or unaccepted fact" do
      workspace_id = unique_workspace("truth-memory-gate")

      # Never persisted: even an "accepted"-looking struct must be rejected.
      unpersisted =
        Fact.new(%{
          id: "fact_never_promoted",
          tenant_id: "default",
          workspace_id: workspace_id,
          fact_text: "Unpersisted text must not become memory.",
          lifecycle_state: "accepted",
          accepted_claim_ids: ["cl_fake"]
        })

      assert {:error, :fact_not_found} = MemoryObject.build_from_fact(unpersisted)

      # Bare map with no id at all.
      assert {:error, :fact_not_found} =
               MemoryObject.build_from_fact(%{
                 workspace_id: workspace_id,
                 fact_text: "Bare map without lineage."
               })

      # Persisted but still a candidate (never promoted through review).
      candidate =
        Fact.new(%{
          id: "fact_candidate_#{System.unique_integer([:positive])}",
          tenant_id: "default",
          workspace_id: workspace_id,
          fact_text: "Candidate fact awaiting review.",
          lifecycle_state: "candidate",
          accepted_claim_ids: ["cl_pending"]
        })

      assert :ok = MemoryCoreStore.insert_fact(candidate)
      assert {:error, :fact_not_accepted} = MemoryObject.build_from_fact(candidate)

      # Persisted and "accepted" but with no accepted claim lineage.
      no_lineage =
        Fact.new(%{
          id: "fact_no_lineage_#{System.unique_integer([:positive])}",
          tenant_id: "default",
          workspace_id: workspace_id,
          fact_text: "Accepted fact without claim lineage.",
          lifecycle_state: "accepted",
          accepted_claim_ids: []
        })

      assert :ok = MemoryCoreStore.insert_fact(no_lineage)
      assert {:error, :fact_not_accepted} = MemoryObject.build_from_fact(no_lineage)

      assert {:ok, [[0]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM memory_objects WHERE workspace_id = ?1",
                 [workspace_id]
               )
    end

    test "rechecks the persisted fact so stale structs cannot build memory after supersession" do
      workspace_id = unique_workspace("truth-memory-stale")

      old_source = build_source(workspace_id, "Launch set for June #{workspace_id}.")
      {:ok, old_claim} = extract_launch_claim(old_source, claim_text: "Launch set for June.")
      {:ok, old_fact} = FactPromoter.promote(old_claim, verifier_id: "human:reviewer")

      new_source = build_source(workspace_id, "Launch moved to July #{workspace_id}.")
      {:ok, new_claim} = extract_launch_claim(new_source, claim_text: "Launch moved to July.")
      {:ok, _new_fact} = FactPromoter.promote(new_claim, verifier_id: "human:reviewer")

      # old_fact's struct still says "accepted", but the persisted row is
      # superseded: building memory from the stale copy must fail.
      assert old_fact.lifecycle_state == "accepted"
      assert {:error, :fact_not_accepted} = MemoryObject.build_from_fact(old_fact)
    end
  end

  describe "MemoryCore.Store typed reads" do
    test "round-trips claims, facts, and memory objects by workspace" do
      workspace_id = unique_workspace("truth-store")
      source = build_source(workspace_id, "Launch approval store #{workspace_id}.")
      {:ok, claim} = extract_launch_claim(source)

      assert {:ok, %Claim{} = stored_claim} = MemoryCoreStore.get_claim(workspace_id, claim.id)
      assert stored_claim.claim_text == claim.claim_text
      assert stored_claim.status == "pending"
      assert stored_claim.subject == "project_launch"
      assert stored_claim.security_labels == ["internal"]

      assert {:ok, [%Claim{id: pending_id}]} =
               MemoryCoreStore.list_claims(workspace_id, review_status: "unreviewed")

      assert pending_id == claim.id

      assert {:error, :not_found} = MemoryCoreStore.get_claim(workspace_id, "cl_missing")
      assert {:error, :not_found} = MemoryCoreStore.get_claim("other-workspace", claim.id)

      {:ok, fact} = FactPromoter.promote(claim, verifier_id: "human:reviewer")
      assert {:ok, %Fact{} = stored_fact} = MemoryCoreStore.get_fact(workspace_id, fact.id)
      assert stored_fact.fact_text == fact.fact_text
      assert stored_fact.accepted_claim_ids == [claim.id]

      {:ok, memory} = MemoryObject.build_from_fact(fact)

      assert {:ok, %MemoryObject{summary: summary}} =
               MemoryCoreStore.get_memory_object(workspace_id, memory.id)

      assert summary == memory.summary

      assert {:ok, [%MemoryObject{id: memory_id}]} =
               MemoryCoreStore.list_memory_objects(workspace_id, fact_link_id: fact.id)

      assert memory_id == memory.id
    end
  end

  describe "KnowledgeLifecycle deprecated delegation" do
    test "legacy entry points still run the full lifecycle" do
      workspace_id = unique_workspace("truth-legacy")
      source = build_source(workspace_id, "Launch approval legacy #{workspace_id}.")

      assert {:ok, %Claim{} = claim} =
               KnowledgeLifecycle.extract_claim(source,
                 claim_text: "Legacy claim text.",
                 actor_id: "agent:test"
               )

      assert {:ok, %Fact{} = fact} =
               KnowledgeLifecycle.promote_claim_to_fact(claim, verifier_id: "human:reviewer")

      assert {:ok, %MemoryObject{}} = KnowledgeLifecycle.build_memory_object(fact)

      assert {:ok, [[3]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type IN
                   ('memory_core.extract_claim', 'memory_core.promote_fact', 'memory_core.build_memory_object')
                 """,
                 [workspace_id]
               )
    end
  end
end
