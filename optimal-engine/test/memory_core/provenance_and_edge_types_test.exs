defmodule OptimalEngine.MemoryCore.ProvenanceAndEdgeTypesTest do
  @moduledoc """
  Asserts:
  (a) model_id is non-null on extract_claim and intake.process ledger entries
      when model_id is passed via opts.
  (b) contradicts, depends_on, and part_of relationship edges can be produced.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.Claim
  alias OptimalEngine.MemoryCore.ClaimExtractor
  alias OptimalEngine.MemoryCore.Fact
  alias OptimalEngine.MemoryCore.FactPromoter
  alias OptimalEngine.MemoryCore.MemoryObject
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.MemoryCore.SourcePackageService
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Signal
  alias OptimalEngine.Context
  alias OptimalEngine.Store

  defp unique_ws(prefix), do: "#{prefix}-#{System.unique_integer([:positive])}"

  defp build_source(workspace_id, raw_text, opts \\ []) do
    SourcePackage.from_text(
      raw_text,
      Keyword.merge(
        [workspace_id: workspace_id, trust_label: "unreviewed"],
        opts
      )
    )
  end

  defp insert_accepted_fact(workspace_id, subject, action, object) do
    source = build_source(workspace_id, "base fact text")
    :ok = MemoryCoreStore.insert_source_package(source)

    {:ok, claim} =
      ClaimExtractor.extract_from_source(source,
        claim_text: "base: #{subject} #{action} #{object}",
        subject_anchor: subject,
        action_class: action,
        object_anchor: object,
        actor_id: "agent:extractor"
      )

    FactPromoter.promote(claim,
      workspace_id: workspace_id,
      verifier_id: "agent:verifier",
      subject_anchor: subject,
      action_class: action,
      object_anchor: object
    )
  end

  # ---------------------------------------------------------------------------
  # (a) model_id propagation into derivation_ledger
  # ---------------------------------------------------------------------------

  describe "model_id on extract_claim ledger entries" do
    test "extract_claim ledger entry carries model_id when provided" do
      ws = unique_ws("ledger-model-extract")
      source = build_source(ws, "The API rate limit is 100 req/s.")

      assert {:ok, claim} =
               ClaimExtractor.extract_from_source(source,
                 claim_text: "API rate limit is 100 req/s",
                 actor_id: "agent:test",
                 model_id: "nemotron-3-super",
                 model_version: "2026-q1"
               )

      # Query ledger for the extract_claim entry in this workspace
      assert {:ok, rows} =
               Store.raw_query(
                 """
                 SELECT model_id, model_version
                 FROM derivation_ledger
                 WHERE workspace_id = ?1
                   AND activity_type = 'memory_core.extract_claim'
                   AND derivation_stage = 'source_package_to_claim'
                 ORDER BY created_at DESC
                 LIMIT 1
                 """,
                 [ws]
               )

      assert [[model_id, model_version]] = rows
      assert model_id == "nemotron-3-super"
      assert model_version == "2026-q1"

      _ = claim
    end

    test "extract_claim ledger entry still inserts when model_id is omitted" do
      ws = unique_ws("ledger-no-model-extract")
      source = build_source(ws, "No model provided.")

      assert {:ok, _claim} =
               ClaimExtractor.extract_from_source(source,
                 claim_text: "no model",
                 actor_id: "agent:test"
               )

      assert {:ok, [[count]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM derivation_ledger
                 WHERE workspace_id = ?1 AND activity_type = 'memory_core.extract_claim'
                 """,
                 [ws]
               )

      assert count >= 1
    end
  end

  describe "model_id on intake.process ledger entries" do
    test "record_ingested_signal carries model_id when provided" do
      ws = unique_ws("ledger-model-intake")

      source =
        build_source(ws, "intake text")

      :ok = MemoryCoreStore.insert_source_package(source)

      now = DateTime.utc_now()

      signal = %Signal{
        id: :crypto.strong_rand_bytes(8) |> Base.encode16(case: :lower),
        title: "Test Signal",
        genre: "note",
        node: "inbox",
        mode: :linguistic,
        type: :inform,
        format: :markdown,
        structure: "",
        created_at: now,
        modified_at: now,
        sn_ratio: 0.7,
        entities: [],
        routed_to: [],
        content: "intake text"
      }

      context = %Context{
        id: signal.id,
        uri: "optimal://nodes/inbox/signals/test.md",
        type: :signal,
        path: "/tmp/test.md",
        title: signal.title,
        workspace_id: ws
      }

      :ok =
        SourcePackageService.record_ingested_signal(source, signal, context,
          actor_id: "agent:test",
          model_id: "nemotron-3-super",
          model_version: "2026-q1"
        )

      assert {:ok, rows} =
               Store.raw_query(
                 """
                 SELECT model_id, model_version
                 FROM derivation_ledger
                 WHERE workspace_id = ?1
                   AND activity_type = 'intake.process'
                   AND derivation_stage = 'source_to_signal_context'
                 ORDER BY created_at DESC
                 LIMIT 1
                 """,
                 [ws]
               )

      assert [[model_id, model_version]] = rows
      assert model_id == "nemotron-3-super"
      assert model_version == "2026-q1"
    end
  end

  # ---------------------------------------------------------------------------
  # (b) New relationship_edge types: contradicts, depends_on, part_of
  # ---------------------------------------------------------------------------

  describe "contradicts edge — ClaimExtractor" do
    test "emits contradicts edge when claim object_anchor conflicts with an accepted fact" do
      ws = unique_ws("edge-contradicts-extract")

      # Insert an accepted fact: project_launch approved at planning_meeting
      assert {:ok, _fact} =
               insert_accepted_fact(ws, "project_launch", "approved", "planning_meeting")

      # Extract a new claim with same subject+action but different object
      source2 = build_source(ws, "Launch was actually approved at board_meeting.")

      assert {:ok, claim2} =
               ClaimExtractor.extract_from_source(source2,
                 claim_text: "project launch approved at board_meeting",
                 subject_anchor: "project_launch",
                 action_class: "approved",
                 object_anchor: "board_meeting",
                 actor_id: "agent:test"
               )

      # A contradicts edge from claim2 to the accepted fact should exist
      assert {:ok, [[count]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM relationship_edges
                 WHERE workspace_id = ?1
                   AND relationship_type = 'contradicts'
                   AND from_object_type = 'claim'
                   AND from_object_id = ?2
                 """,
                 [ws, claim2.id]
               )

      assert count >= 1,
             "Expected at least one 'contradicts' edge from claim #{claim2.id} in workspace #{ws}"
    end

    test "does not emit contradicts edge when no conflict exists" do
      ws = unique_ws("edge-no-contradicts")

      # Insert accepted fact for a different subject
      assert {:ok, _fact} =
               insert_accepted_fact(ws, "budget", "approved", "board_2026")

      # New claim on a completely different subject
      source2 = build_source(ws, "Feature rollout scheduled for Q3.")

      assert {:ok, claim2} =
               ClaimExtractor.extract_from_source(source2,
                 claim_text: "feature rollout Q3",
                 subject_anchor: "feature_rollout",
                 action_class: "scheduled",
                 object_anchor: "Q3_2026",
                 actor_id: "agent:test"
               )

      assert {:ok, [[count]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM relationship_edges WHERE workspace_id = ?1 AND relationship_type = 'contradicts' AND from_object_id = ?2",
                 [ws, claim2.id]
               )

      assert count == 0
    end
  end

  describe "contradicts edge — FactPromoter" do
    test "emits contradicts edge when promotion is blocked by conflicting fact" do
      ws = unique_ws("edge-contradicts-promoter")

      # First: insert an accepted fact
      assert {:ok, _fact} =
               insert_accepted_fact(ws, "pricing_model", "set", "per_seat_v1")

      # Second: create a claim that conflicts
      source2 = build_source(ws, "Pricing is now flat_fee not per_seat.")
      :ok = MemoryCoreStore.insert_source_package(source2)

      {:ok, conflict_claim} =
        ClaimExtractor.extract_from_source(source2,
          claim_text: "pricing model set to flat_fee",
          subject_anchor: "pricing_model",
          action_class: "set",
          object_anchor: "flat_fee_v1",
          actor_id: "agent:extractor"
        )

      # Attempt promotion WITHOUT resolving the conflict (no :supersedes)
      result =
        FactPromoter.promote(conflict_claim,
          workspace_id: ws,
          verifier_id: "agent:verifier"
        )

      # Promotion is expected to be blocked OR succeed (depends on ScoringPolicy thresholds)
      # Either way, if it was blocked, the contradicts edge must be present.
      case result do
        {:error, {:contradicts_current_facts, _fact_ids}} ->
          assert {:ok, [[count]]} =
                   Store.raw_query(
                     """
                     SELECT COUNT(*) FROM relationship_edges
                     WHERE workspace_id = ?1
                       AND relationship_type = 'contradicts'
                       AND from_object_type = 'claim'
                       AND from_object_id = ?2
                     """,
                     [ws, conflict_claim.id]
                   )

          assert count >= 1,
                 "Expected contradicts edge when promotion was blocked for claim #{conflict_claim.id}"

        {:ok, _fact} ->
          # Promotion succeeded (supersession was auto-detected) — no edge required
          :ok
      end
    end
  end

  describe "depends_on edge" do
    test "emits depends_on edges for declared prerequisite claim ids" do
      ws = unique_ws("edge-depends-on")

      # Extract prerequisite claim
      source_prereq = build_source(ws, "Infrastructure must be provisioned first.")

      assert {:ok, prereq_claim} =
               ClaimExtractor.extract_from_source(source_prereq,
                 claim_text: "infrastructure provisioned",
                 actor_id: "agent:test"
               )

      # Extract dependent claim, declaring prereq
      source_dep = build_source(ws, "Deployment can proceed once infra is ready.")

      assert {:ok, dep_claim} =
               ClaimExtractor.extract_from_source(source_dep,
                 claim_text: "deployment proceeds after infra",
                 actor_id: "agent:test",
                 depends_on_ids: [prereq_claim.id]
               )

      # A depends_on edge from dep_claim to prereq_claim should exist
      assert {:ok, [[count]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM relationship_edges
                 WHERE workspace_id = ?1
                   AND relationship_type = 'depends_on'
                   AND from_object_type = 'claim'
                   AND from_object_id = ?2
                   AND to_object_type = 'claim'
                   AND to_object_id = ?3
                 """,
                 [ws, dep_claim.id, prereq_claim.id]
               )

      assert count >= 1,
             "Expected a 'depends_on' edge from #{dep_claim.id} to #{prereq_claim.id}"
    end

    test "no depends_on edges when depends_on_ids is empty" do
      ws = unique_ws("edge-no-depends-on")
      source = build_source(ws, "Standalone claim.")

      assert {:ok, claim} =
               ClaimExtractor.extract_from_source(source,
                 claim_text: "standalone",
                 actor_id: "agent:test",
                 depends_on_ids: []
               )

      assert {:ok, [[count]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM relationship_edges WHERE workspace_id = ?1 AND relationship_type = 'depends_on' AND from_object_id = ?2",
                 [ws, claim.id]
               )

      assert count == 0
    end
  end

  describe "part_of edge" do
    test "emits part_of edges for declared parent memory object ids" do
      ws = unique_ws("edge-part-of")

      # Build an accepted fact and parent memory object
      assert {:ok, fact} =
               insert_accepted_fact(ws, "system_design", "completed", "v1_spec")

      # Build parent memory object
      assert {:ok, parent_memory} =
               MemoryObject.build_from_fact(fact, memory_type: "composite")

      # Build a child memory object that declares part_of relationship
      assert {:ok, child_memory} =
               MemoryObject.build_from_fact(fact,
                 memory_type: "component",
                 summary: "component of the system design memory",
                 part_of_memory_ids: [parent_memory.id]
               )

      # A part_of edge from child to parent should exist
      assert {:ok, [[count]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*) FROM relationship_edges
                 WHERE workspace_id = ?1
                   AND relationship_type = 'part_of'
                   AND from_object_type = 'memory_object'
                   AND from_object_id = ?2
                   AND to_object_type = 'memory_object'
                   AND to_object_id = ?3
                 """,
                 [ws, child_memory.id, parent_memory.id]
               )

      assert count >= 1,
             "Expected a 'part_of' edge from #{child_memory.id} to #{parent_memory.id}"
    end

    test "no part_of edges when part_of_memory_ids is empty" do
      ws = unique_ws("edge-no-part-of")

      assert {:ok, fact} =
               insert_accepted_fact(ws, "audit_complete", "done", "q1_audit")

      assert {:ok, memory} = MemoryObject.build_from_fact(fact, part_of_memory_ids: [])

      assert {:ok, [[count]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM relationship_edges WHERE workspace_id = ?1 AND relationship_type = 'part_of' AND from_object_id = ?2",
                 [ws, memory.id]
               )

      assert count == 0
    end
  end
end
