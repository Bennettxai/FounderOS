defmodule OptimalEngine.MemoryCore.SpineTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.ActiveMemoryPool
  alias OptimalEngine.MemoryCore.ClaimExtractor
  alias OptimalEngine.MemoryCore.FactPromoter
  alias OptimalEngine.MemoryCore.MemoryObject
  alias OptimalEngine.MemoryCore.RetrievalCoordinator
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.MemoryCore.ToolModelGovernance
  alias OptimalEngine.MemoryCore.WorkflowSkill
  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Pipeline.Intake
  alias OptimalEngine.Store

  setup do
    tmp_dir = System.tmp_dir!() |> Path.join("optimal_memory_core_test_#{:rand.uniform(99_999)}")
    File.mkdir_p!(tmp_dir)

    original_root = Application.get_env(:optimal_engine, :root_path)
    original_cross_ref_files = Application.get_env(:optimal_engine, :write_cross_ref_files)

    Application.put_env(:optimal_engine, :root_path, tmp_dir)
    Application.put_env(:optimal_engine, :write_cross_ref_files, false)

    for folder <- ~w[
          entity-company product-customer-portal operation-delivery operation-weekly-review 05-operations
          project-platform-launch 07-community learning-research-library
          09-inbox team 11-finance 12-program
        ] do
      File.mkdir_p!(Path.join([tmp_dir, folder, "signals"]))
    end

    on_exit(fn ->
      Application.put_env(:optimal_engine, :root_path, original_root)
      Application.put_env(:optimal_engine, :write_cross_ref_files, original_cross_ref_files)
      File.rm_rf!(tmp_dir)
    end)

    %{tmp_dir: tmp_dir}
  end

  test "migration creates the Memory Core object tables" do
    required_tables = [
      "source_packages",
      "claims",
      "facts",
      "memory_objects",
      "memory_detail_objects",
      "relationship_edges",
      "derivation_ledger",
      "workflow_traces",
      "generalized_workflows",
      "procedural_memory_objects",
      "skill_packages",
      "active_memory_pools",
      "context_packages",
      "model_call_operations",
      "mcp_tool_definitions",
      "model_call_runs",
      "tool_call_runs"
    ]

    for table <- required_tables do
      {:ok, [[^table]]} =
        Store.raw_query("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?1", [
          table
        ])
    end
  end

  test "SourcePackage.from_text/2 is workspace-scoped and deterministic" do
    raw_text = "Memory Core source package deterministic test #{System.unique_integer()}"

    source_a =
      SourcePackage.from_text(raw_text,
        tenant_id: "tenant-a",
        workspace_id: "workspace-a",
        source_type: "manual",
        security_labels: ["internal"],
        partition_ids: ["Project Y"]
      )

    source_b =
      SourcePackage.from_text(raw_text,
        tenant_id: "tenant-a",
        workspace_id: "workspace-a"
      )

    source_c =
      SourcePackage.from_text(raw_text,
        tenant_id: "tenant-a",
        workspace_id: "workspace-b"
      )

    assert source_a.id == source_b.id
    assert source_a.id != source_c.id
    assert source_a.content_hash == source_b.content_hash
    assert source_a.security_labels == ["internal"]
    assert source_a.partition_ids == ["Project Y"]
  end

  test "typed store inserts Source Packages idempotently" do
    raw_text = "Memory Core typed store insert test #{System.unique_integer()}"

    source_package =
      SourcePackage.from_text(raw_text,
        tenant_id: "default",
        workspace_id: "memory-core-spine-test",
        trust_label: "reviewed"
      )

    assert :ok = MemoryCoreStore.insert_source_package(source_package)
    assert :ok = MemoryCoreStore.insert_source_package(source_package)

    {:ok, [[count, trust_label]]} =
      Store.raw_query(
        "SELECT COUNT(*), MAX(trust_label) FROM source_packages WHERE id = ?1",
        [source_package.id]
      )

    assert count == 1
    assert trust_label == "reviewed"
  end

  test "accepted intake records source evidence and derivation ledger" do
    raw_text = "Memory Core intake provenance test #{System.unique_integer()}"

    {:ok, result} =
      Intake.process(raw_text,
        node: "founder",
        title: "Memory Core Spine Test",
        workspace_id: "memory-core-intake-test"
      )

    {:ok, [[source_id, content_hash]]} =
      Store.raw_query(
        """
        SELECT id, content_hash
        FROM source_packages
        WHERE workspace_id = ?1 AND raw_text = ?2
        """,
        ["memory-core-intake-test", raw_text]
      )

    assert is_binary(source_id)
    assert byte_size(content_hash) == 64

    {:ok, [[ledger_count]]} =
      Store.raw_query(
        """
        SELECT COUNT(*)
        FROM derivation_ledger
        WHERE workspace_id = ?1
          AND activity_type = 'intake.process'
          AND derivation_stage = 'source_to_signal_context'
          AND input_object_links LIKE ?2
          AND output_object_links LIKE ?3
        """,
        [
          "memory-core-intake-test",
          "%#{source_id}%",
          "%#{result.signal.id}%"
        ]
      )

    assert ledger_count >= 1
  end

  test "knowledge lifecycle separates source, claim, fact, and memory object" do
    workspace_id = "memory-core-lifecycle-test-#{System.unique_integer([:positive])}"

    source =
      SourcePackage.from_text("The project launch was approved in the planning meeting.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        trust_label: "unreviewed",
        security_labels: ["internal"],
        partition_ids: ["project-launch"]
      )

    assert {:ok, claim} =
             ClaimExtractor.extract_from_source(source,
               claim_text: "The source states that the project launch was approved.",
               subject_anchor: "project_launch",
               action_class: "approved",
               object_anchor: "planning_meeting",
               aggregate_confidence: 0.72,
               aggregate_precision: 0.68,
               actor_id: "agent:test"
             )

    assert claim.lifecycle_state == "pending"
    assert claim.review_status == "unreviewed"
    assert claim.source_package_id == source.id

    assert {:ok, fact} =
             FactPromoter.promote(claim,
               fact_text: "The project launch was approved in the planning meeting.",
               verifier_id: "human:reviewer",
               valid_time_start: "2026-06-02T00:00:00Z",
               aggregate_confidence: 0.9,
               aggregate_precision: 0.86
             )

    assert fact.lifecycle_state == "accepted"
    assert fact.verification_status == "verified"
    assert fact.accepted_claim_ids == [claim.id]

    assert {:ok, memory} =
             MemoryObject.build_from_fact(fact,
               summary:
                 "The project launch approval is an accepted memory backed by the planning meeting source.",
               memory_type: "decision",
               salience: 0.8
             )

    assert memory.lifecycle_state == "current"
    assert memory.memory_type == "decision"
    assert memory.fact_links == [%{type: "fact", id: fact.id}]

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM source_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, source.id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query("SELECT COUNT(*) FROM claims WHERE workspace_id = ?1 AND id = ?2", [
               workspace_id,
               claim.id
             ])

    assert {:ok, [[1]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1 AND id = ?2", [
               workspace_id,
               fact.id
             ])

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM memory_objects WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, memory.id]
             )

    assert {:ok, [[3]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM relationship_edges WHERE workspace_id = ?1 AND relationship_type = 'supports'",
               [workspace_id]
             )

    assert {:ok, [[3]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM derivation_ledger WHERE workspace_id = ?1 AND activity_type IN ('memory_core.extract_claim', 'memory_core.promote_fact', 'memory_core.build_memory_object')",
               [workspace_id]
             )
  end

  test "retrieval coordinator assembles a governed context package" do
    workspace_id = "memory-core-retrieval-test-#{System.unique_integer([:positive])}"

    {:ok, fact, memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"]
             )

    assert package.request_intent == "recall"
    assert package.requesting_actor_id == "agent:test"
    assert package.fact_links == [%{type: "fact", id: fact.id}]
    assert package.memory_links == [%{type: "memory_object", id: memory.id}]
    assert package.filtered_object_summary.returned_facts == 1
    assert package.filtered_object_summary.returned_memory_objects == 1
    assert package.filtered_object_summary.redacted_or_filtered_objects == 0
    assert length(package.facts) == 1
    assert length(package.memory_objects) == 1

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, package.id]
             )
  end

  test "retrieval coordinator batch refreshes stale context packages" do
    workspace_id = "memory-core-batch-refresh-test-#{System.unique_integer([:positive])}"

    {:ok, fact, _memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"]
             )

    assert :ok =
             MemoryCoreStore.invalidate_context_packages_for_object("fact", fact.id,
               workspace_id: workspace_id,
               reason: "batch_refresh_test"
             )

    assert {:ok, batch} = MemoryCore.refresh_stale_context_packages(workspace_id: workspace_id)
    assert batch.stale_context_package_ids == [package.id]
    assert batch.errors == []
    assert [refreshed_package] = batch.refreshed_context_packages
    assert refreshed_package.id != package.id
    assert refreshed_package.refresh_state == "fresh"

    assert {:ok, [["refreshed"]]} =
             Store.raw_query(
               "SELECT refresh_state FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, package.id]
             )

    assert {:ok, empty_batch} =
             MemoryCore.refresh_stale_context_packages(workspace_id: workspace_id)

    assert empty_batch.stale_context_package_ids == []
    assert empty_batch.refreshed_context_packages == []
    assert empty_batch.errors == []
  end

  test "retrieval coordinator filters unauthorized partitions before packaging" do
    workspace_id = "memory-core-retrieval-filter-test-#{System.unique_integer([:positive])}"

    {:ok, _fact, _memory} = create_accepted_memory(workspace_id, partition_ids: ["restricted"])

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["public"],
               allowed_security_labels: ["internal"]
             )

    assert package.fact_links == []
    assert package.memory_links == []
    assert package.filtered_object_summary.candidate_facts == 1
    assert package.filtered_object_summary.candidate_memory_objects == 1
    assert package.filtered_object_summary.redacted_or_filtered_objects == 2

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, package.id]
             )
  end

  test "retrieval coordinator applies structured subject/action/object filters" do
    workspace_id = "memory-core-structured-retrieval-test-#{System.unique_integer([:positive])}"

    {:ok, fact, memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    assert {:ok, matching_package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"],
               subject_anchor: "project_launch",
               action_class: "approved",
               object_anchor: "planning_meeting"
             )

    assert matching_package.retrieval_plan.path == "memory_core_structured_lookup"
    assert matching_package.retrieval_plan.structured_filters.subject_anchor == "project_launch"
    assert matching_package.fact_links == [%{type: "fact", id: fact.id}]
    assert matching_package.memory_links == [%{type: "memory_object", id: memory.id}]

    assert {:ok, mismatched_package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"],
               subject_anchor: "other_subject"
             )

    assert mismatched_package.fact_links == []
    assert mismatched_package.memory_links == []
    assert mismatched_package.filtered_object_summary.returned_facts == 0
    assert mismatched_package.filtered_object_summary.returned_memory_objects == 0
  end

  test "retrieval coordinator returns governed asset extraction projections", %{tmp_dir: tmp_dir} do
    workspace_id = "memory-core-retrieval-extraction-test-#{System.unique_integer([:positive])}"
    source_path = Path.join(tmp_dir, "retrieval-audio.wav")
    File.write!(source_path, "RIFF....WAVEretrieval-extraction-test")

    assert {:ok, %{asset: asset}} =
             MemoryCore.store_asset_file(source_path,
               workspace_id: workspace_id,
               actor_id: "user:retrieval",
               security_labels: ["internal"],
               partition_ids: ["project-launch"]
             )

    assert {:ok, run} =
             MemoryCore.run_asset_adapter(asset.id, :openai_whisper,
               workspace_id: workspace_id,
               actor_id: "user:retrieval",
               command: "printf",
               args: ["Launch transcript confirms approval"],
               adapter_role: :audio_transcription,
               confidence: 0.81,
               precision: 0.76
             )

    assert run.status == "completed"

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("Launch transcript",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"]
             )

    assert [%{type: "asset_extraction", id: extraction_id}] =
             Enum.filter(package.returned_object_links, &(&1.type == "asset_extraction"))

    assert package.filtered_object_summary.candidate_asset_extractions == 1
    assert package.filtered_object_summary.returned_asset_extractions == 1
    assert [%{id: ^extraction_id, extraction_type: "transcript"}] = package.asset_extractions
    assert package.package_confidence_summary.count == 1
    assert %{type: "asset", id: asset.id} in package.evidence_links

    assert {:ok, restricted_package} =
             RetrievalCoordinator.retrieve("Launch transcript",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["other-project"],
               allowed_security_labels: ["internal"]
             )

    assert restricted_package.asset_extractions == []
    assert restricted_package.filtered_object_summary.candidate_asset_extractions == 1
    assert restricted_package.filtered_object_summary.returned_asset_extractions == 0
    assert restricted_package.filtered_object_summary.redacted_or_filtered_objects == 1
  end

  test "retrieval coordinator excludes stale facts and memory objects before packaging" do
    workspace_id = "memory-core-retrieval-stale-test-#{System.unique_integer([:positive])}"
    stale_after = DateTime.utc_now() |> DateTime.add(-60, :second) |> DateTime.to_iso8601()

    {:ok, _fact, _memory} =
      create_accepted_memory(workspace_id,
        partition_ids: ["project-launch"],
        fact_opts: [stale_after: stale_after],
        memory_opts: [stale_after: stale_after]
      )

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"]
             )

    assert package.fact_links == []
    assert package.memory_links == []
    assert package.filtered_object_summary.candidate_facts == 0
    assert package.filtered_object_summary.candidate_memory_objects == 0
  end

  test "retrieval coordinator excludes superseded memory objects" do
    workspace_id = "memory-core-retrieval-superseded-test-#{System.unique_integer([:positive])}"

    {:ok, fact, memory} =
      create_accepted_memory(workspace_id,
        partition_ids: ["project-launch"],
        memory_opts: [supersession_status: "superseded"]
      )

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"]
             )

    assert package.fact_links == [%{type: "fact", id: fact.id}]
    assert package.memory_links == []
    assert memory.supersession_status == "superseded"
    assert package.filtered_object_summary.candidate_facts == 1
    assert package.filtered_object_summary.candidate_memory_objects == 0
  end

  test "active memory pool loads context and publishes observations as pending claims" do
    workspace_id = "memory-core-pool-test-#{System.unique_integer([:positive])}"

    {:ok, fact, _memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    assert {:ok, pool} =
             ActiveMemoryPool.open(
               workspace_id: workspace_id,
               task_type: "project_review",
               subject_anchor: "project_launch",
               member_links: [%{type: "human", id: "human:reviewer"}],
               agent_links: [%{type: "agent", id: "agent:test"}],
               security_labels: ["internal"],
               partition_ids: ["project-launch"]
             )

    assert pool.lifecycle_state == "open"
    assert pool.refresh_state == "fresh"

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               active_memory_pool_id: pool.id,
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"]
             )

    assert {:ok, loaded_pool} = ActiveMemoryPool.load_context_package(pool.id, package)
    assert %{type: "context_package", id: package.id} in loaded_pool.context_package_links
    assert loaded_pool.refresh_state == "fresh"

    assert {:ok, skipped_refresh} = MemoryCore.refresh_pool_context_packages(pool.id)
    assert skipped_refresh.refreshed_context_packages == []
    assert skipped_refresh.skipped_context_package_ids == [package.id]

    assert :ok =
             MemoryCoreStore.invalidate_context_packages_for_object("fact", fact.id,
               workspace_id: workspace_id,
               reason: "pool_refresh_test"
             )

    assert {:ok, refreshed} = MemoryCore.refresh_pool_context_packages(pool.id)
    assert [refreshed_package] = refreshed.refreshed_context_packages
    assert refreshed_package.id != package.id
    assert refreshed_package.refresh_state == "fresh"

    assert %{type: "context_package", id: refreshed_package.id} in refreshed.pool.context_package_links

    assert {:ok, observation} =
             ActiveMemoryPool.publish_observation(
               pool.id,
               "Reviewer observed that launch approval still needs a budget follow-up.",
               actor_id: "agent:test",
               claim_text: "The launch approval needs a budget follow-up.",
               subject_anchor: "project_launch",
               action_class: "needs_follow_up",
               object_anchor: "budget",
               aggregate_confidence: 0.61,
               aggregate_precision: 0.58
             )

    assert observation.pending_claim.lifecycle_state == "pending"
    assert observation.pending_claim.review_status == "unreviewed"
    assert observation.pool.refresh_state == "dirty"

    assert %{type: "claim", id: observation.pending_claim.id} in observation.pool.promotion_candidate_links

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM claims WHERE workspace_id = ?1 AND id = ?2 AND lifecycle_state = 'pending'",
               [workspace_id, observation.pending_claim.id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM source_packages WHERE workspace_id = ?1 AND id = ?2 AND source_type = 'pool_observation'",
               [workspace_id, observation.source_package.id]
             )

    assert {:ok, closed_pool} = ActiveMemoryPool.close(pool.id, reason: "test complete")
    assert closed_pool.lifecycle_state == "closed"
    assert closed_pool.archive_state == "archived"
    assert is_binary(closed_pool.closed_at)
  end

  test "workflow and skill lifecycle turns observed work into a draft skill package" do
    workspace_id = "memory-core-workflow-test-#{System.unique_integer([:positive])}"

    {:ok, _fact, _memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    {:ok, pool} =
      ActiveMemoryPool.open(
        workspace_id: workspace_id,
        task_type: "launch_review",
        subject_anchor: "project_launch",
        member_links: [%{type: "human", id: "human:reviewer"}],
        agent_links: [%{type: "agent", id: "agent:test"}],
        security_labels: ["internal"],
        partition_ids: ["project-launch"]
      )

    {:ok, package} =
      RetrievalCoordinator.retrieve("launch",
        workspace_id: workspace_id,
        actor_id: "agent:test",
        active_memory_pool_id: pool.id,
        allowed_partitions: ["project-launch"],
        allowed_security_labels: ["internal"]
      )

    {:ok, _loaded_pool} = ActiveMemoryPool.load_context_package(pool.id, package)

    {:ok, observation} =
      ActiveMemoryPool.publish_observation(
        pool.id,
        "Launch review used source-backed context, checked pending budget work, and recorded the follow-up.",
        actor_id: "agent:test",
        claim_text: "Launch review needs a budget follow-up.",
        subject_anchor: "project_launch",
        action_class: "reviewed",
        object_anchor: "budget_follow_up"
      )

    assert {:ok, trace} =
             WorkflowSkill.capture_trace_from_pool(pool.id,
               workflow_family: "launch_review",
               action_class: "reviewed",
               outcome: "budget_follow_up_recorded",
               step_links: [
                 %{type: "step", id: "load_context"},
                 %{type: "step", id: "record_observation"}
               ],
               actor_id: "agent:test"
             )

    assert trace.workflow_family == "launch_review"
    assert %{type: "claim", id: observation.pending_claim.id} in trace.output_links

    assert {:ok, workflow} =
             WorkflowSkill.generalize_workflow(trace,
               applicability_conditions: %{node_type: "project"},
               validation_score: 0.35,
               actor_id: "agent:test"
             )

    assert workflow.lifecycle_state == "candidate"
    assert workflow.validation_status == "unvalidated"

    assert {:ok, procedure} =
             WorkflowSkill.create_procedural_memory(workflow,
               capability_name: "launch_review_procedure",
               required_privileges: ["memory:read", "claims:create"],
               actor_id: "agent:test"
             )

    assert procedure.capability_name == "launch_review_procedure"
    assert procedure.lifecycle_state == "candidate"

    assert {:ok, skill} =
             WorkflowSkill.package_skill(procedure,
               skill_package_name: "launch_review_skill",
               input_contract: %{requires: ["project_node_id"]},
               output_contract: %{creates: ["pending_claims"]},
               execution_policy: %{requires_review: true},
               actor_id: "agent:test"
             )

    assert skill.skill_package_name == "launch_review_skill"
    assert skill.review_status == "draft"
    assert skill.enabled_state == "disabled"

    for {table, id} <- [
          {"workflow_traces", trace.id},
          {"generalized_workflows", workflow.id},
          {"procedural_memory_objects", procedure.id},
          {"skill_packages", skill.id}
        ] do
      assert {:ok, [[1]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM #{table} WHERE workspace_id = ?1 AND id = ?2",
                 [
                   workspace_id,
                   id
                 ]
               )
    end

    assert {:ok, [[ledger_count]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type IN (
                   'memory_core.capture_workflow_trace',
                   'memory_core.generalize_workflow',
                   'memory_core.create_procedural_memory',
                   'memory_core.package_skill'
                 )
               """,
               [workspace_id]
             )

    assert ledger_count == 4
  end

  test "retrieval coordinator packages workflow candidates and approved enabled skills" do
    workspace_id = "memory-core-workflow-retrieval-test-#{System.unique_integer([:positive])}"

    {:ok, _fact, _memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    {:ok, pool} =
      ActiveMemoryPool.open(
        workspace_id: workspace_id,
        task_type: "launch_review",
        subject_anchor: "project_launch",
        member_links: [%{type: "human", id: "human:reviewer"}],
        agent_links: [%{type: "agent", id: "agent:test"}],
        security_labels: ["internal"],
        partition_ids: ["project-launch"]
      )

    {:ok, trace} =
      WorkflowSkill.capture_trace_from_pool(pool.id,
        workflow_family: "launch_review",
        action_class: "reviewed",
        outcome: "budget_follow_up_recorded",
        step_links: [
          %{type: "step", id: "load_context"},
          %{type: "step", id: "record_observation"}
        ],
        actor_id: "agent:test"
      )

    {:ok, workflow} =
      WorkflowSkill.generalize_workflow(trace,
        applicability_conditions: %{node_type: "project"},
        validation_score: 0.45,
        actor_id: "agent:test"
      )

    {:ok, procedure} =
      WorkflowSkill.create_procedural_memory(workflow,
        capability_name: "launch_review_procedure",
        required_privileges: ["memory:read", "claims:create"],
        actor_id: "agent:test"
      )

    {:ok, draft_skill} =
      WorkflowSkill.package_skill(procedure,
        skill_package_name: "launch_review_draft_skill",
        actor_id: "agent:test"
      )

    assert draft_skill.review_status == "draft"
    assert draft_skill.enabled_state == "disabled"

    assert {:ok, draft_package} =
             RetrievalCoordinator.retrieve("launch_review",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"],
               workflow_family: "launch_review",
               task_family: "launch_review"
             )

    assert draft_package.workflow_links == [%{type: "generalized_workflow", id: workflow.id}]
    assert draft_package.skill_package_links == []
    assert draft_package.filtered_object_summary.returned_workflows == 1
    assert draft_package.filtered_object_summary.returned_skill_packages == 0
    assert "workflow_skill.eligibility_filter" in draft_package.retrieval_plan.executed_paths

    {:ok, approved_skill} =
      WorkflowSkill.package_skill(procedure,
        skill_package_name: "launch_review_approved_skill",
        review_status: "approved",
        enabled_state: "enabled",
        input_contract: %{requires: ["project_node_id"]},
        output_contract: %{creates: ["pending_claims"]},
        execution_policy: %{requires_review: true},
        actor_id: "agent:test"
      )

    assert {:ok, package} =
             RetrievalCoordinator.retrieve("launch_review",
               workspace_id: workspace_id,
               actor_id: "agent:test",
               allowed_partitions: ["project-launch"],
               allowed_security_labels: ["internal"],
               workflow_family: "launch_review",
               task_family: "launch_review"
             )

    assert package.workflow_links == [%{type: "generalized_workflow", id: workflow.id}]
    assert package.skill_package_links == [%{type: "skill_package", id: approved_skill.id}]
    assert package.filtered_object_summary.returned_workflows == 1
    assert package.filtered_object_summary.returned_skill_packages == 1
    assert [%{skill_package_name: "launch_review_approved_skill"}] = package.skill_packages

    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM context_packages
               WHERE workspace_id = ?1
                 AND id = ?2
                 AND workflow_links != '[]'
                 AND skill_package_links != '[]'
               """,
               [workspace_id, package.id]
             )
  end

  test "tool and model governance records allowed and rejected calls" do
    workspace_id = "memory-core-tool-model-test-#{System.unique_integer([:positive])}"

    {:ok, _fact, _memory} = create_accepted_memory(workspace_id, partition_ids: ["project-launch"])

    {:ok, pool} =
      ActiveMemoryPool.open(
        workspace_id: workspace_id,
        task_type: "tool_model_review",
        subject_anchor: "project_launch",
        agent_links: [%{type: "agent", id: "agent:test"}],
        security_labels: ["internal"],
        partition_ids: ["project-launch"]
      )

    assert {:ok, tool_definition} =
             ToolModelGovernance.register_mcp_tool_definition(
               workspace_id: workspace_id,
               tool_name: "calendar.read",
               implementation_type: "external_connector",
               required_privileges: ["calendar:read"],
               allowed_partitions: ["project-launch"],
               input_schema: %{required: ["calendar_id"]},
               output_schema: %{required: ["events"]},
               security_labels: ["internal"],
               partition_ids: ["project-launch"]
             )

    assert {:ok, model_operation} =
             ToolModelGovernance.register_model_call_operation(
               workspace_id: workspace_id,
               function_name: "summarize_context",
               model_task_type: "summarization",
               model_id: "test-model",
               required_privileges: ["model:summarize"],
               allowed_partitions: ["project-launch"],
               input_schema: %{required: ["context"]},
               output_contract: %{required: ["summary"]},
               security_labels: ["internal"],
               partition_ids: ["project-launch"]
             )

    assert {:ok, rejected_tool_run} =
             ToolModelGovernance.record_tool_call(
               "calendar.read",
               %{calendar_id: "primary"},
               workspace_id: workspace_id,
               actor_id: "agent:test",
               granted_privileges: [],
               requested_partitions: ["project-launch"]
             )

    assert rejected_tool_run.decision_state == "rejected"
    assert rejected_tool_run.rejection_reason =~ "missing:calendar:read"

    assert [%{type: "audit_event", kind: "tool.call.governed"}] =
             rejected_tool_run.audit_event_links

    assert {:ok, invalid_tool_output_run} =
             ToolModelGovernance.record_tool_call(
               "calendar.read",
               %{calendar_id: "primary"},
               workspace_id: workspace_id,
               actor_id: "agent:test",
               active_memory_pool_id: pool.id,
               granted_privileges: ["calendar:read"],
               requested_partitions: ["project-launch"],
               output_payload: %{},
               observation_text: "Calendar tool returned an invalid empty payload.",
               claim_text: "Calendar returned an invalid empty payload."
             )

    assert invalid_tool_output_run.decision_state == "allowed"
    assert invalid_tool_output_run.run_status == "output_rejected"
    assert invalid_tool_output_run.rejection_reason =~ "missing_output:events"
    assert invalid_tool_output_run.observation_links == []
    assert invalid_tool_output_run.source_package_links == []

    assert [%{type: "audit_event", kind: "tool.call.governed"}] =
             invalid_tool_output_run.audit_event_links

    assert {:ok, allowed_tool_run} =
             ToolModelGovernance.record_tool_call(
               "calendar.read",
               %{calendar_id: "primary"},
               workspace_id: workspace_id,
               actor_id: "agent:test",
               active_memory_pool_id: pool.id,
               granted_privileges: ["calendar:read"],
               requested_partitions: ["project-launch"],
               output_payload: %{events: [%{title: "Launch review"}]},
               observation_text: "Calendar tool returned a launch review event.",
               claim_text: "Calendar contains a launch review event.",
               subject_anchor: "project_launch",
               action_class: "scheduled",
               object_anchor: "launch_review_event"
             )

    assert allowed_tool_run.decision_state == "allowed"
    assert allowed_tool_run.run_status == "completed"
    assert allowed_tool_run.mcp_tool_definition_id == tool_definition.id
    assert [%{type: "audit_event", kind: "tool.call.governed"}] = allowed_tool_run.audit_event_links
    assert [%{type: "claim", id: pending_claim_id}] = allowed_tool_run.observation_links

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM claims WHERE workspace_id = ?1 AND id = ?2 AND review_status = 'unreviewed'",
               [workspace_id, pending_claim_id]
             )

    assert {:ok, allowed_model_run} =
             ToolModelGovernance.record_model_call(
               "summarize_context",
               %{context: "Launch context"},
               workspace_id: workspace_id,
               actor_id: "agent:test",
               granted_privileges: ["model:summarize"],
               requested_partitions: ["project-launch"],
               output_payload: %{summary: "Launch is ready for review."}
             )

    assert allowed_model_run.decision_state == "allowed"
    assert allowed_model_run.model_call_operation_id == model_operation.id

    assert [%{type: "audit_event", kind: "model.call.governed"}] =
             allowed_model_run.audit_event_links

    assert {:ok, invalid_model_output_run} =
             ToolModelGovernance.record_model_call(
               "summarize_context",
               %{context: "Launch context"},
               workspace_id: workspace_id,
               actor_id: "agent:test",
               granted_privileges: ["model:summarize"],
               requested_partitions: ["project-launch"],
               output_payload: %{}
             )

    assert invalid_model_output_run.decision_state == "allowed"
    assert invalid_model_output_run.run_status == "output_rejected"
    assert invalid_model_output_run.rejection_reason =~ "missing_output:summary"
    assert invalid_model_output_run.observation_links == []

    assert [%{type: "audit_event", kind: "model.call.governed"}] =
             invalid_model_output_run.audit_event_links

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM mcp_tool_definitions WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, tool_definition.id]
             )

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM model_call_operations WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, model_operation.id]
             )

    assert {:ok, [[3]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM tool_call_runs WHERE workspace_id = ?1 AND mcp_tool_definition_id = ?2",
               [workspace_id, tool_definition.id]
             )

    assert {:ok, [[2]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM model_call_runs WHERE workspace_id = ?1 AND model_call_operation_id = ?2",
               [workspace_id, model_operation.id]
             )

    # Every governed run must join to its audit event by the REAL persisted
    # event id (Agent Work Loop gate: "Audit records the run") — no synthetic
    # ids, no fuzzy metadata-echo lookups.
    governed_runs = [
      {rejected_tool_run, "tool.call.governed"},
      {invalid_tool_output_run, "tool.call.governed"},
      {allowed_tool_run, "tool.call.governed"},
      {allowed_model_run, "model.call.governed"},
      {invalid_model_output_run, "model.call.governed"}
    ]

    for {run, kind} <- governed_runs do
      assert [%{type: "audit_event", id: audit_event_id, kind: ^kind}] = run.audit_event_links
      assert is_integer(audit_event_id)

      assert {:ok, [[1]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*)
                 FROM events
                 WHERE id = ?1
                   AND tenant_id = 'default'
                   AND workspace_id = ?2
                   AND kind = ?3
                   AND metadata LIKE ?4
                 """,
                 [audit_event_id, workspace_id, kind, "%#{run.id}%"]
               )
    end

    assert {:ok, [[5]]} =
             Store.raw_query(
               """
               SELECT COUNT(*)
               FROM events
               WHERE tenant_id = 'default'
                 AND workspace_id = ?1
                 AND kind IN ('tool.call.governed', 'model.call.governed')
               """,
               [workspace_id]
             )
  end

  test "tool governance execute helper blocks, runs, and records tool execution" do
    workspace_id = "memory-core-tool-execute-test-#{System.unique_integer([:positive])}"
    test_pid = self()

    assert {:ok, _tool_definition} =
             ToolModelGovernance.register_mcp_tool_definition(
               workspace_id: workspace_id,
               tool_name: "workspace.notify",
               implementation_type: "internal_adapter",
               required_privileges: ["workspace:notify"],
               allowed_partitions: ["workspace"],
               input_schema: %{required: ["message"]},
               output_schema: %{required: ["delivered"]}
             )

    assert {:error, {:rejected, rejected_run}} =
             ToolModelGovernance.execute_tool_call(
               "workspace.notify",
               %{message: "Blocked"},
               fn _payload ->
                 send(test_pid, :should_not_execute)
                 {:ok, %{delivered: true}}
               end,
               workspace_id: workspace_id,
               actor_id: "agent:test",
               granted_privileges: [],
               requested_partitions: ["workspace"]
             )

    assert rejected_run.decision_state == "rejected"
    refute_receive :should_not_execute

    assert {:ok, completed_run} =
             ToolModelGovernance.execute_tool_call(
               "workspace.notify",
               %{message: "Allowed"},
               fn payload ->
                 send(test_pid, {:executed, payload.message})
                 {:ok, %{delivered: true}}
               end,
               workspace_id: workspace_id,
               actor_id: "agent:test",
               granted_privileges: ["workspace:notify"],
               requested_partitions: ["workspace"]
             )

    assert completed_run.decision_state == "allowed"
    assert completed_run.run_status == "completed"
    assert_receive {:executed, "Allowed"}

    assert {:error, {:execution_failed, :boom, failed_run}} =
             ToolModelGovernance.execute_tool_call(
               "workspace.notify",
               %{message: "Fail"},
               fn _payload -> {:error, :boom} end,
               workspace_id: workspace_id,
               actor_id: "agent:test",
               granted_privileges: ["workspace:notify"],
               requested_partitions: ["workspace"]
             )

    assert failed_run.decision_state == "allowed"
    assert failed_run.run_status == "failed"
    assert failed_run.rejection_reason =~ "execution_failed"

    assert {:ok, [[3]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM tool_call_runs WHERE workspace_id = ?1 AND tool_name = 'workspace.notify'",
               [workspace_id]
             )
  end

  defp create_accepted_memory(workspace_id, opts) do
    partition_ids = Keyword.fetch!(opts, :partition_ids)
    fact_opts = Keyword.get(opts, :fact_opts, [])
    memory_opts = Keyword.get(opts, :memory_opts, [])

    source =
      SourcePackage.from_text("The project launch was approved in the planning meeting.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        trust_label: "unreviewed",
        security_labels: ["internal"],
        partition_ids: partition_ids
      )

    with {:ok, claim} <-
           ClaimExtractor.extract_from_source(source,
             claim_text: "The source states that the project launch was approved.",
             subject_anchor: "project_launch",
             action_class: "approved",
             object_anchor: "planning_meeting",
             aggregate_confidence: 0.72,
             aggregate_precision: 0.68,
             actor_id: "agent:test"
           ),
         {:ok, fact} <-
           FactPromoter.promote(
             claim,
             [
               fact_text: "The project launch was approved in the planning meeting.",
               verifier_id: "human:reviewer",
               aggregate_confidence: 0.9,
               aggregate_precision: 0.86
             ] ++ fact_opts
           ),
         {:ok, memory} <-
           MemoryObject.build_from_fact(
             fact,
             [
               summary:
                 "The project launch approval is an accepted memory backed by the planning meeting source.",
               memory_type: "decision",
               salience: 0.8
             ] ++ memory_opts
           ) do
      {:ok, fact, memory}
    end
  end
end
