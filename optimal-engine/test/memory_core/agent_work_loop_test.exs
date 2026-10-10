defmodule OptimalEngine.MemoryCore.AgentWorkLoopTest do
  @moduledoc """
  Agent Work Loop gate proof: pool opens -> Context Package loads -> governed
  tool output becomes an observation -> observation becomes a pending Claim ->
  a human or policy promotes or rejects it -> only promotion creates a Fact ->
  derivation ledger and audit record everything.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.ActiveMemoryPool
  alias OptimalEngine.MemoryCore.ClaimExtractor
  alias OptimalEngine.MemoryCore.FactPromoter
  alias OptimalEngine.MemoryCore.MemoryObject
  alias OptimalEngine.MemoryCore.RetrievalCoordinator
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.MemoryCore.ToolModelGovernance
  alias OptimalEngine.Store

  test "tool output becomes a pending claim that a human promotes into a fact" do
    workspace_id = unique_workspace()
    {:ok, _fact, _memory} = seed_accepted_memory(workspace_id)

    {:ok, pool} =
      ActiveMemoryPool.open(
        workspace_id: workspace_id,
        task_type: "launch_review",
        subject_anchor: "project_launch",
        member_links: [%{type: "human", id: "human:reviewer"}],
        agent_links: [%{type: "agent", id: "agent:loop"}],
        security_labels: ["internal"],
        partition_ids: ["project-launch"]
      )

    {:ok, package} =
      RetrievalCoordinator.retrieve("launch",
        workspace_id: workspace_id,
        actor_id: "agent:loop",
        active_memory_pool_id: pool.id,
        allowed_partitions: ["project-launch"],
        allowed_security_labels: ["internal"]
      )

    assert {:ok, loaded_pool} = ActiveMemoryPool.load_context_package(pool.id, package)
    assert %{type: "context_package", id: package.id} in loaded_pool.context_package_links

    {:ok, _tool} =
      ToolModelGovernance.register_mcp_tool_definition(
        workspace_id: workspace_id,
        tool_name: "calendar.read",
        required_privileges: ["calendar:read"],
        allowed_partitions: ["project-launch"],
        input_schema: %{
          "required" => ["calendar_id"],
          "properties" => %{"calendar_id" => %{"type" => "string"}}
        },
        output_schema: %{"required" => ["events"]}
      )

    assert {:ok, run} =
             ToolModelGovernance.execute_tool_call(
               "calendar.read",
               %{calendar_id: "primary"},
               fn _payload -> {:ok, %{events: [%{title: "Launch review"}]}} end,
               workspace_id: workspace_id,
               actor_id: "agent:loop",
               active_memory_pool_id: pool.id,
               granted_privileges: ["calendar:read"],
               requested_partitions: ["project-launch"],
               observation_text: "Calendar tool found a scheduled launch review event.",
               claim_text: "A launch review event is scheduled.",
               subject_anchor: "project_launch",
               action_class: "scheduled",
               object_anchor: "launch_review_event"
             )

    assert run.run_status == "completed"
    assert [%{type: "claim", id: claim_id}] = run.observation_links
    assert [%{type: "source_package", id: source_package_id}] = run.source_package_links

    assert [%{type: "audit_event", id: audit_event_id, kind: "tool.call.governed"}] =
             run.audit_event_links

    # The audit link carries the REAL persisted audit event id (the events
    # table's AUTOINCREMENT integer key), never a fabricated "audit_..." id:
    # the run joins to its audit event row by id.
    assert is_integer(audit_event_id)

    assert {:ok, [[event_kind, event_target_uri, event_metadata_json]]} =
             Store.raw_query(
               "SELECT kind, target_uri, metadata FROM events WHERE id = ?1",
               [audit_event_id]
             )

    assert event_kind == "tool.call.governed"
    assert event_target_uri == "tool:#{workspace_id}:#{run.id}"

    event_metadata = Jason.decode!(event_metadata_json)
    assert event_metadata["run_id"] == run.id
    assert event_metadata["run_type"] == "tool_call_run"
    assert event_metadata["workspace_id"] == workspace_id
    assert event_metadata["decision_state"] == "allowed"

    # The persisted run row stamps the same real audit event id.
    assert {:ok, [[run_links_json]]} =
             Store.raw_query(
               "SELECT audit_event_links FROM tool_call_runs WHERE id = ?1",
               [run.id]
             )

    assert [%{"type" => "audit_event", "id" => ^audit_event_id}] = Jason.decode!(run_links_json)

    # Raw tool output evidence is preserved as a Source Package linked to the run.
    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM source_packages WHERE workspace_id = ?1 AND id = ?2 AND source_type = 'tool_model_output'",
               [workspace_id, source_package_id]
             )

    # The observation is a pending Claim recording the tool as extractor,
    # carrying the governed run ID for lineage.
    {:ok, claim} = MemoryCoreStore.get_claim(workspace_id, claim_id)
    assert claim.status == "pending"
    assert claim.review_status == "unreviewed"
    assert claim.extracted_by == "tool:calendar.read"
    assert claim.metadata["origin_run_id"] == run.id
    assert claim.metadata["origin_run_type"] == "tool_call_run"

    # Tool output did NOT become a Fact without promotion.
    assert {:ok, [[0]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM facts WHERE workspace_id = ?1 AND accepted_claim_ids LIKE ?2",
               [workspace_id, "%#{claim_id}%"]
             )

    {:ok, refreshed_pool} = ActiveMemoryPool.get(pool.id)
    assert candidate?(refreshed_pool, claim_id)

    # Human review promotes the pending Claim through FactPromoter.
    assert {:ok, %{pool: reviewed_pool, fact: fact}} =
             ActiveMemoryPool.promote_observation(pool.id, claim_id,
               verifier_id: "human:reviewer",
               actor_id: "human:reviewer"
             )

    assert fact.accepted_claim_ids == [claim_id]
    assert fact.verifier_id == "human:reviewer"
    refute candidate?(reviewed_pool, claim_id)
    assert %{type: "fact", id: fact.id} in reviewed_pool.evidence_links

    {:ok, promoted_claim} = MemoryCoreStore.get_claim(workspace_id, claim_id)
    assert promoted_claim.status == "promoted"
    assert promoted_claim.review_status == "approved"

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM facts WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, fact.id]
             )

    # Lineage: the promotion wrote its derivation ledger entry.
    assert {:ok, [[1]]} =
             Store.raw_query(
               """
               SELECT COUNT(*) FROM derivation_ledger
               WHERE workspace_id = ?1
                 AND activity_type = 'memory_core.promote_fact'
                 AND output_object_links LIKE ?2
               """,
               [workspace_id, "%#{fact.id}%"]
             )

    # Audit recorded the governed run.
    assert {:ok, [[audit_count]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM events WHERE kind = 'tool.call.governed' AND metadata LIKE ?1",
               ["%#{workspace_id}%"]
             )

    assert audit_count >= 1
  end

  test "a rejected observation never becomes a fact" do
    workspace_id = unique_workspace()

    {:ok, pool} =
      ActiveMemoryPool.open(
        workspace_id: workspace_id,
        task_type: "incident_triage",
        subject_anchor: "outage"
      )

    {:ok, observation} =
      ActiveMemoryPool.publish_observation(
        pool.id,
        "Agent suspects the deploy caused the outage.",
        actor_id: "agent:loop",
        claim_text: "The deploy caused the outage.",
        subject_anchor: "outage",
        action_class: "caused_by",
        object_anchor: "deploy"
      )

    claim_id = observation.pending_claim.id
    assert observation.pending_claim.status == "pending"
    assert observation.pending_claim.extracted_by == "agent:loop"

    assert {:ok, %{pool: reviewed_pool, rejected_claim: rejected}} =
             ActiveMemoryPool.reject_observation(pool.id, claim_id,
               verifier_id: "human:reviewer",
               reason: "not enough evidence"
             )

    assert rejected.status == "rejected"
    assert rejected.review_status == "rejected"
    refute candidate?(reviewed_pool, claim_id)

    {:ok, stored} = MemoryCoreStore.get_claim(workspace_id, claim_id)
    assert stored.status == "rejected"
    assert stored.review_status == "rejected"

    # No Fact exists anywhere in the workspace.
    assert {:ok, [[0]]} =
             Store.raw_query("SELECT COUNT(*) FROM facts WHERE workspace_id = ?1", [workspace_id])

    # The rejection wrote its derivation ledger entry.
    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM derivation_ledger WHERE workspace_id = ?1 AND activity_type = 'memory_core.reject_claim'",
               [workspace_id]
             )

    # A rejected claim cannot come back through the pool review path...
    assert {:error, :not_a_promotion_candidate} =
             ActiveMemoryPool.promote_observation(pool.id, claim_id, verifier_id: "human:reviewer")

    # ...and the review boundary itself blocks it too.
    assert {:error, :claim_rejected} =
             FactPromoter.promote(claim_id,
               workspace_id: workspace_id,
               verifier_id: "human:reviewer"
             )
  end

  test "promotion requires reviewer approval or explicit auto-promotion policy" do
    workspace_id = unique_workspace()

    {:ok, pool} =
      ActiveMemoryPool.open(workspace_id: workspace_id, task_type: "review_policy_check")

    {:ok, low} =
      ActiveMemoryPool.publish_observation(pool.id, "Low-confidence observation about caching.",
        actor_id: "agent:loop",
        subject_anchor: "cache",
        action_class: "suspected_stale"
      )

    assert {:error, :approval_required} =
             ActiveMemoryPool.promote_observation(pool.id, low.pending_claim.id, [])

    assert {:error, :below_auto_promote_threshold} =
             ActiveMemoryPool.promote_observation(pool.id, low.pending_claim.id, policy: :auto)

    {:ok, still_pending} = MemoryCoreStore.get_claim(workspace_id, low.pending_claim.id)
    assert still_pending.status == "pending"

    {:ok, refreshed_pool} = ActiveMemoryPool.get(pool.id)
    assert candidate?(refreshed_pool, low.pending_claim.id)

    {:ok, high} =
      ActiveMemoryPool.publish_observation(
        pool.id,
        "High-confidence observation: deploy 42 finished successfully.",
        actor_id: "agent:loop",
        claim_text: "Deploy 42 finished successfully.",
        subject_anchor: "deploy_42",
        action_class: "completed",
        aggregate_confidence: 0.95,
        aggregate_precision: 0.9
      )

    assert {:ok, %{fact: fact}} =
             ActiveMemoryPool.promote_observation(pool.id, high.pending_claim.id, policy: :auto)

    assert fact.verifier_id =~ "policy:"
    assert fact.accepted_claim_ids == [high.pending_claim.id]
  end

  test "concurrent pool mutations are serialized: no update is ever lost" do
    workspace_id = unique_workspace()

    {:ok, pool} =
      ActiveMemoryPool.open(
        workspace_id: workspace_id,
        task_type: "concurrency_check",
        subject_anchor: "race"
      )

    observation_count = 8
    context_count = 4

    publish_jobs =
      for index <- 1..observation_count do
        fn ->
          ActiveMemoryPool.publish_observation(
            pool.id,
            "Concurrent observation #{index}: worker #{index} saw event #{index}.",
            actor_id: "agent:worker-#{index}",
            subject_anchor: "race",
            action_class: "observed"
          )
        end
      end

    context_jobs =
      for index <- 1..context_count do
        fn -> ActiveMemoryPool.load_context_package(pool.id, %{id: "ctx-concurrent-#{index}"}) end
      end

    results =
      (publish_jobs ++ context_jobs)
      |> Enum.map(&Task.async/1)
      |> Task.await_many(30_000)

    assert Enum.all?(results, &match?({:ok, _}, &1))

    observations = for {:ok, %{pending_claim: _} = observation} <- results, do: observation
    claim_ids = Enum.map(observations, & &1.pending_claim.id)
    source_ids = Enum.map(observations, & &1.source_package.id)

    assert length(Enum.uniq(claim_ids)) == observation_count
    assert length(Enum.uniq(source_ids)) == observation_count

    # Every concurrent writer's links survive: a lost (last-write-wins) update
    # would silently drop another writer's promotion candidate or source link.
    {:ok, refreshed} = ActiveMemoryPool.get(pool.id)

    for claim_id <- claim_ids, do: assert(candidate?(refreshed, claim_id))
    assert length(refreshed.promotion_candidate_links) == observation_count

    for source_id <- source_ids do
      assert Enum.any?(refreshed.source_package_links, fn link ->
               link_id(link) == source_id
             end)
    end

    for index <- 1..context_count do
      assert Enum.any?(refreshed.context_package_links, fn link ->
               link_id(link) == "ctx-concurrent-#{index}"
             end)
    end

    # Concurrent reviews on distinct claims must not lose each other's
    # decisions either: promote half, reject half, all at once.
    {to_promote, to_reject} = Enum.split(claim_ids, div(observation_count, 2))

    review_results =
      Enum.map(to_promote, fn claim_id ->
        Task.async(fn ->
          ActiveMemoryPool.promote_observation(pool.id, claim_id,
            verifier_id: "human:reviewer",
            actor_id: "human:reviewer"
          )
        end)
      end)
      |> Kernel.++(
        Enum.map(to_reject, fn claim_id ->
          Task.async(fn ->
            ActiveMemoryPool.reject_observation(pool.id, claim_id,
              verifier_id: "human:reviewer",
              reason: "concurrent review"
            )
          end)
        end)
      )
      |> Task.await_many(30_000)

    assert Enum.all?(review_results, &match?({:ok, _}, &1))

    {:ok, reviewed} = ActiveMemoryPool.get(pool.id)
    assert reviewed.promotion_candidate_links == []

    fact_ids = for {:ok, %{fact: fact}} <- review_results, do: fact.id
    assert length(fact_ids) == length(to_promote)

    for fact_id <- fact_ids do
      assert Enum.any?(reviewed.evidence_links, fn link -> link_id(link) == fact_id end)
    end

    # Every review decision is recorded on the pool; none was overwritten.
    decisions = Map.get(reviewed.metadata, "review_decisions", [])
    assert length(decisions) == observation_count

    decided_claim_ids = decisions |> Enum.map(&Map.get(&1, "claim_id")) |> Enum.sort()
    assert decided_claim_ids == Enum.sort(claim_ids)
  end

  test "closed pools refuse new context and observations" do
    workspace_id = unique_workspace()

    {:ok, pool} = ActiveMemoryPool.open(workspace_id: workspace_id, task_type: "short_lived")
    {:ok, _closed} = ActiveMemoryPool.close(pool.id, reason: "done")

    assert {:error, {:pool_not_open, "closed"}} =
             ActiveMemoryPool.publish_observation(pool.id, "Late observation.", [])

    assert {:error, {:pool_not_open, "closed"}} =
             ActiveMemoryPool.load_context_package(pool.id, %{id: "ctx-after-close"})
  end

  test "promote_observation only accepts claims linked to the pool" do
    workspace_id = unique_workspace()

    {:ok, pool} = ActiveMemoryPool.open(workspace_id: workspace_id, task_type: "scope_check")

    assert {:error, :not_a_promotion_candidate} =
             ActiveMemoryPool.promote_observation(pool.id, "cl-unrelated",
               verifier_id: "human:reviewer"
             )

    assert {:error, :not_a_promotion_candidate} =
             ActiveMemoryPool.reject_observation(pool.id, "cl-unrelated",
               verifier_id: "human:reviewer"
             )
  end

  test "governed dispatch rejects payloads that violate the registered schema types" do
    workspace_id = unique_workspace()
    test_pid = self()

    {:ok, _tool} =
      ToolModelGovernance.register_mcp_tool_definition(
        workspace_id: workspace_id,
        tool_name: "tickets.create",
        input_schema: %{
          "required" => ["title", "priority"],
          "properties" => %{
            "title" => %{"type" => "string"},
            "priority" => %{"type" => "integer"}
          }
        },
        output_schema: %{"required" => ["ticket_id"]}
      )

    assert {:error, {:rejected, rejected_run}} =
             ToolModelGovernance.execute_tool_call(
               "tickets.create",
               %{title: "Fix login", priority: "high"},
               fn _payload ->
                 send(test_pid, :should_not_execute)
                 {:ok, %{ticket_id: "T-1"}}
               end,
               workspace_id: workspace_id,
               actor_id: "agent:loop"
             )

    assert rejected_run.decision_state == "rejected"
    assert rejected_run.rejection_reason =~ "invalid_input:priority(expected integer)"
    refute_receive :should_not_execute

    # Rejected runs also stamp a REAL persisted audit event id.
    assert [%{type: "audit_event", id: rejected_audit_id, kind: "tool.call.governed"}] =
             rejected_run.audit_event_links

    assert is_integer(rejected_audit_id)

    assert {:ok, [[rejected_event_metadata]]} =
             Store.raw_query("SELECT metadata FROM events WHERE id = ?1", [rejected_audit_id])

    assert Jason.decode!(rejected_event_metadata)["decision_state"] == "rejected"

    assert {:ok, completed_run} =
             ToolModelGovernance.execute_tool_call(
               "tickets.create",
               %{title: "Fix login", priority: 2},
               fn _payload -> {:ok, %{ticket_id: "T-2"}} end,
               workspace_id: workspace_id,
               actor_id: "agent:loop"
             )

    assert completed_run.decision_state == "allowed"
    assert completed_run.run_status == "completed"

    # Each persisted run gets its own real audit event row.
    assert [%{type: "audit_event", id: completed_audit_id}] = completed_run.audit_event_links
    assert is_integer(completed_audit_id)
    assert completed_audit_id != rejected_audit_id

    assert {:ok, [[1]]} =
             Store.raw_query(
               "SELECT COUNT(*) FROM events WHERE id = ?1 AND kind = 'tool.call.governed'",
               [completed_audit_id]
             )
  end

  test "a completed run records when its observation could not be attached" do
    workspace_id = unique_workspace()

    {:ok, pool} = ActiveMemoryPool.open(workspace_id: workspace_id, task_type: "late_capture")
    {:ok, _closed} = ActiveMemoryPool.close(pool.id, reason: "done early")

    {:ok, _tool} =
      ToolModelGovernance.register_mcp_tool_definition(
        workspace_id: workspace_id,
        tool_name: "echo.say",
        input_schema: %{"required" => ["message"]},
        output_schema: %{"required" => ["echo"]}
      )

    assert {:ok, run} =
             ToolModelGovernance.execute_tool_call(
               "echo.say",
               %{message: "hi"},
               fn _payload -> {:ok, %{echo: "hi"}} end,
               workspace_id: workspace_id,
               actor_id: "agent:loop",
               active_memory_pool_id: pool.id,
               observation_text: "Echo replied."
             )

    assert run.run_status == "completed"
    assert run.observation_links == []
    assert run.metadata.observation_error =~ "pool_not_open"
  end

  defp unique_workspace, do: "agent-work-loop-test-#{System.unique_integer([:positive])}"

  defp candidate?(pool, claim_id) do
    Enum.any?(pool.promotion_candidate_links, fn link ->
      (Map.get(link, :type) || Map.get(link, "type")) == "claim" and
        (Map.get(link, :id) || Map.get(link, "id")) == claim_id
    end)
  end

  defp link_id(link), do: Map.get(link, :id) || Map.get(link, "id")

  defp seed_accepted_memory(workspace_id) do
    source =
      SourcePackage.from_text("The project launch was approved in the planning meeting.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        trust_label: "unreviewed",
        security_labels: ["internal"],
        partition_ids: ["project-launch"]
      )

    with {:ok, claim} <-
           ClaimExtractor.extract_from_source(source,
             claim_text: "The source states that the project launch was approved.",
             subject_anchor: "project_launch",
             action_class: "approved",
             object_anchor: "planning_meeting",
             aggregate_confidence: 0.72,
             aggregate_precision: 0.68,
             actor_id: "agent:seed"
           ),
         {:ok, fact} <-
           FactPromoter.promote(claim,
             fact_text: "The project launch was approved in the planning meeting.",
             verifier_id: "human:reviewer",
             aggregate_confidence: 0.9,
             aggregate_precision: 0.86
           ),
         {:ok, memory} <-
           MemoryObject.build_from_fact(fact,
             summary: "The launch approval is an accepted memory backed by the meeting source.",
             memory_type: "decision",
             salience: 0.8
           ) do
      {:ok, fact, memory}
    end
  end
end
