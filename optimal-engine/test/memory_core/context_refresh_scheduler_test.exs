defmodule OptimalEngine.MemoryCore.ContextRefreshSchedulerTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.ContextRefreshScheduler
  alias OptimalEngine.MemoryCore.KnowledgeLifecycle
  alias OptimalEngine.MemoryCore.RetrievalCoordinator
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Store

  test "application scheduler is disabled in tests" do
    assert %{enabled: false} = ContextRefreshScheduler.status()
  end

  test "run_once refreshes stale context packages for a workspace" do
    workspace_id = "memory-core-scheduler-test-#{System.unique_integer([:positive])}"
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
               reason: "scheduler_refresh_test"
             )

    assert {:ok, summary} =
             ContextRefreshScheduler.run_once(
               workspace_id: workspace_id,
               tenant_id: "default",
               actor_id: "system:test-scheduler",
               batch_limit: 10
             )

    assert summary.refreshed_count == 1
    assert summary.error_count == 0
    assert summary.stale_context_package_ids == [package.id]
    assert [workspace_result] = summary.workspace_results
    assert workspace_result.workspace_id == workspace_id
    assert workspace_result.refreshed_count == 1

    assert {:ok, [["refreshed"]]} =
             Store.raw_query(
               "SELECT refresh_state FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
               [workspace_id, package.id]
             )
  end

  test "force_run executes through a scheduler process even when periodic ticks are disabled" do
    workspace_id = "memory-core-force-scheduler-test-#{System.unique_integer([:positive])}"
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
               reason: "scheduler_force_refresh_test"
             )

    server = :"context_refresh_scheduler_#{System.unique_integer([:positive])}"

    assert {:ok, pid} =
             ContextRefreshScheduler.start_link(
               name: server,
               enabled: false,
               tenant_id: "default",
               workspace_ids: [workspace_id],
               batch_limit: 10
             )

    assert {:ok, summary} = ContextRefreshScheduler.force_run(nil, server: server)
    assert summary.refreshed_count == 1
    assert summary.stale_context_package_ids == [package.id]

    GenServer.stop(pid)
  end

  defp create_accepted_memory(workspace_id, opts) do
    partition_ids = Keyword.fetch!(opts, :partition_ids)

    source =
      SourcePackage.from_text("The project launch was approved in the planning meeting.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        trust_label: "unreviewed",
        security_labels: ["internal"],
        partition_ids: partition_ids
      )

    with {:ok, claim} <-
           KnowledgeLifecycle.extract_claim(source,
             claim_text: "The source states that the project launch was approved.",
             subject_anchor: "project_launch",
             action_class: "approved",
             object_anchor: "planning_meeting",
             aggregate_confidence: 0.72,
             aggregate_precision: 0.68,
             actor_id: "agent:test"
           ),
         {:ok, fact} <-
           KnowledgeLifecycle.promote_claim_to_fact(claim,
             fact_text: "The project launch was approved in the planning meeting.",
             verifier_id: "human:reviewer",
             aggregate_confidence: 0.9,
             aggregate_precision: 0.86
           ),
         {:ok, memory} <-
           KnowledgeLifecycle.build_memory_object(fact,
             summary:
               "The project launch approval is an accepted memory backed by the planning meeting source.",
             memory_type: "decision",
             salience: 0.8
           ) do
      {:ok, fact, memory}
    end
  end
end
