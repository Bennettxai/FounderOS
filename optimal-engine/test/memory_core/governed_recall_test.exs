defmodule OptimalEngine.MemoryCore.GovernedRecallTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.ClaimExtractor
  alias OptimalEngine.MemoryCore.ContextPackage
  alias OptimalEngine.MemoryCore.FactPromoter
  alias OptimalEngine.MemoryCore.ID
  alias OptimalEngine.MemoryCore.MemoryObject
  alias OptimalEngine.MemoryCore.RetrievalCoordinator
  alias OptimalEngine.MemoryCore.RetrievalPackage
  alias OptimalEngine.MemoryCore.ScopeEnvelope
  alias OptimalEngine.MemoryCore.SourcePackage
  alias OptimalEngine.MemoryCore.Store, as: MemoryCoreStore
  alias OptimalEngine.Store

  defp unique_workspace(prefix), do: "#{prefix}-#{System.unique_integer([:positive])}"

  defp insert_fact!(workspace_id, attrs) do
    fact =
      Map.merge(
        %{
          id: ID.random_id("fact"),
          tenant_id: "default",
          workspace_id: workspace_id,
          fact_text: "The project launch was approved.",
          subject_anchor: "project_launch",
          action_class: "approved",
          lifecycle_state: "accepted",
          contradiction_status: "none",
          aggregate_confidence: 0.5,
          aggregate_precision: 0.5,
          security_labels: ["internal"],
          partition_ids: ["public"],
          supporting_evidence_links: []
        },
        attrs
      )

    :ok = MemoryCoreStore.insert_fact(fact)
    fact
  end

  defp seed_accepted_memory(workspace_id, opts \\ []) do
    partition_ids = Keyword.get(opts, :partition_ids, ["project-launch"])
    security_labels = Keyword.get(opts, :security_labels, ["internal"])

    source =
      SourcePackage.from_text("The project launch was approved in the planning meeting.",
        workspace_id: workspace_id,
        source_type: "meeting_note",
        trust_label: "unreviewed",
        security_labels: security_labels,
        partition_ids: partition_ids
      )

    {:ok, claim} =
      ClaimExtractor.extract_from_source(source,
        claim_text: "The source states that the project launch was approved.",
        subject_anchor: "project_launch",
        action_class: "approved",
        object_anchor: "planning_meeting",
        aggregate_confidence: 0.72,
        aggregate_precision: 0.68,
        actor_id: "agent:test"
      )

    {:ok, fact} =
      FactPromoter.promote(claim,
        fact_text: "The project launch was approved in the planning meeting.",
        verifier_id: "human:reviewer",
        aggregate_confidence: 0.9,
        aggregate_precision: 0.86
      )

    {:ok, memory} =
      MemoryObject.build_from_fact(fact,
        summary: "The project launch approval is an accepted memory backed by the source.",
        memory_type: "decision",
        salience: 0.8
      )

    %{source: source, claim: claim, fact: fact, memory: memory}
  end

  describe "retrieve_package/2 — authorization during candidate expansion" do
    test "workspace scope is applied inside the candidate queries, not as a post-filter" do
      workspace_a = unique_workspace("governed-recall-ws-a")
      workspace_b = unique_workspace("governed-recall-ws-b")

      fact_a = insert_fact!(workspace_a, %{})
      _fact_b = insert_fact!(workspace_b, %{aggregate_confidence: 0.99})

      assert {:ok, %RetrievalPackage{} = package} =
               RetrievalCoordinator.retrieve_package("launch",
                 workspace_id: workspace_a,
                 actor_id: "agent:test",
                 allowed_partitions: ["public"],
                 allowed_security_labels: ["internal"]
               )

      assert Enum.map(package.facts, & &1.id) == [fact_a.id]
      assert Enum.all?(package.facts, &(&1.workspace_id == workspace_a))

      # The other workspace's row never entered the candidate set — the
      # candidate count only sees workspace A.
      assert package.filtered_summary.candidate_facts == 1
      assert package.filtered_summary.returned_facts == 1
      assert package.filtered_summary.redacted_or_filtered_objects == 0

      assert package.retrieval_plan.filters.workspace_id == workspace_a

      assert package.retrieval_plan.filters.authorization.applied ==
               "during_candidate_expansion"
    end

    test "authorized objects are not crowded out by higher-ranked unauthorized candidates" do
      workspace_id = unique_workspace("governed-recall-crowding")

      for n <- 1..5 do
        insert_fact!(workspace_id, %{
          fact_text: "Restricted launch detail #{n}.",
          partition_ids: ["restricted"],
          aggregate_confidence: 0.99
        })
      end

      authorized_fact =
        insert_fact!(workspace_id, %{
          fact_text: "Public launch summary.",
          partition_ids: ["public"],
          aggregate_confidence: 0.4
        })

      assert {:ok, package} =
               RetrievalCoordinator.retrieve_package("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:test",
                 allowed_partitions: ["public"],
                 allowed_security_labels: ["internal"],
                 limit: 2
               )

      assert Enum.map(package.facts, & &1.id) == [authorized_fact.id]
      assert package.filtered_summary.returned_facts == 1
      assert package.filtered_summary.candidate_facts == 6
      assert package.filtered_summary.redacted_or_filtered_objects == 5
    end

    test "filtered_summary carries reason classes and redacted object links" do
      workspace_id = unique_workspace("governed-recall-reasons")
      seeded = seed_accepted_memory(workspace_id, partition_ids: ["restricted"])

      secret_fact =
        insert_fact!(workspace_id, %{
          fact_text: "Secret launch budget.",
          partition_ids: ["restricted"],
          security_labels: ["secret"]
        })

      assert {:ok, package} =
               RetrievalCoordinator.retrieve_package("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:test",
                 allowed_partitions: ["public"],
                 allowed_security_labels: ["internal"]
               )

      assert package.facts == []
      assert package.memory_objects == []
      assert package.filtered_summary.redacted_or_filtered_objects == 3
      assert package.filtered_summary.reason_classes["partition_filtered"] == 3
      assert package.filtered_summary.reason_classes["security_label_filtered"] == 1

      redacted_by_id = Map.new(package.redacted_object_links, &{&1.id, &1})
      assert redacted_by_id[seeded.fact.id].type == "fact"
      assert redacted_by_id[seeded.fact.id].reasons == ["partition_filtered"]
      assert redacted_by_id[seeded.memory.id].type == "memory_object"

      assert redacted_by_id[secret_fact.id].reasons == [
               "partition_filtered",
               "security_label_filtered"
             ]
    end

    test "accepts a ScopeEnvelope as the second argument" do
      workspace_id = unique_workspace("governed-recall-scope")

      # Unlabeled, unpartitioned objects: the bare ScopeEnvelope carries no
      # authorization envelope, and the empty envelope fails closed.
      seeded = seed_accepted_memory(workspace_id, partition_ids: [], security_labels: [])

      scope = ScopeEnvelope.resolve(workspace_id: workspace_id, actor_id: "agent:scope")

      assert {:ok, %RetrievalPackage{} = package} =
               RetrievalCoordinator.retrieve_package("launch", scope)

      assert package.workspace_id == workspace_id
      assert package.scope.actor_id == "agent:scope"
      assert Enum.map(package.facts, & &1.id) == [seeded.fact.id]

      assert {:ok, %ContextPackage{} = context_package} =
               RetrievalCoordinator.retrieve("launch", scope)

      assert context_package.workspace_id == workspace_id
      assert context_package.requesting_actor_id == "agent:scope"
    end
  end

  describe "retrieve/2 and build_context_package/2 — Context Package projection" do
    test "returns a persisted ContextPackage with citations, freshness, and ledger lineage" do
      workspace_id = unique_workspace("governed-recall-package")
      seeded = seed_accepted_memory(workspace_id)

      assert {:ok, %ContextPackage{} = package} =
               RetrievalCoordinator.retrieve("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:test",
                 allowed_partitions: ["project-launch"],
                 allowed_security_labels: ["internal"]
               )

      assert package.fact_links == [%{type: "fact", id: seeded.fact.id}]
      assert package.memory_links == [%{type: "memory_object", id: seeded.memory.id}]

      # Facts carry source links back to the Source Package.
      [fact] = package.facts
      assert Enum.any?(fact.source_package_links, &(link_id(&1) == seeded.source.id))
      assert Enum.any?(package.source_package_links, &(link_id(&1) == seeded.source.id))

      # Freshness metadata.
      assert package.refresh_state == "fresh"
      assert is_binary(package.refresh_time)
      assert package.time_mode == "current_valid"

      # Authorization snapshot applied during candidate expansion.
      assert package.authorization_envelope.applied_during_candidate_expansion == true
      assert package.authorization_envelope.actor_id == "agent:test"

      # Token-budget-aware sections with inline citations.
      assert package.sections.facts =~ seeded.fact.fact_text
      assert package.sections.facts =~ seeded.source.id
      assert package.sections.summary =~ "Facts: 1 returned of 1 candidates"
      assert package.sections.total_tokens <= package.sections.token_budget
      refute package.sections.truncated?

      # Persisted row.
      assert {:ok, [[1]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, package.id]
               )

      # Derivation ledger entry: source, processor, actor, policy.
      assert {:ok, [[1]]} =
               Store.raw_query(
                 """
                 SELECT COUNT(*)
                 FROM derivation_ledger
                 WHERE workspace_id = ?1
                   AND activity_type = 'memory_core.assemble_context_package'
                   AND derivation_stage = 'retrieval_package_to_context_package'
                   AND actor_id = 'agent:test'
                   AND parser_id = 'optimal_engine.memory_core.retrieval_coordinator'
                   AND policy_version = 'governed_recall_v1'
                   AND output_object_links LIKE ?2
                   AND source_package_links LIKE ?3
                 """,
                 [workspace_id, "%#{package.id}%", "%#{seeded.source.id}%"]
               )
    end

    test "build_context_package/2 projects a RetrievalPackage into a ContextPackage" do
      workspace_id = unique_workspace("governed-recall-projection")
      seeded = seed_accepted_memory(workspace_id)

      {:ok, retrieval_package} =
        RetrievalCoordinator.retrieve_package("launch",
          workspace_id: workspace_id,
          actor_id: "agent:test",
          allowed_partitions: ["project-launch"],
          allowed_security_labels: ["internal"]
        )

      assert {:ok, %ContextPackage{} = package} =
               RetrievalCoordinator.build_context_package(retrieval_package,
                 request_id: "req_test",
                 detail_depth: 2
               )

      assert package.returned_object_links ==
               RetrievalPackage.returned_object_links(retrieval_package)

      assert package.filtered_object_summary == retrieval_package.filtered_summary
      assert package.retrieval_plan == retrieval_package.retrieval_plan
      assert package.request_id == "req_test"
      assert package.detail_depth == 2
      assert package.metadata.retrieval_package_id == retrieval_package.id
      assert [%{id: fact_id}] = package.fact_links
      assert fact_id == seeded.fact.id

      assert {:ok, [[1]]} =
               Store.raw_query(
                 "SELECT COUNT(*) FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, package.id]
               )
    end

    test "policy-excluded objects keep redacted links on the persisted package" do
      workspace_id = unique_workspace("governed-recall-redaction")
      seeded = seed_accepted_memory(workspace_id, partition_ids: ["restricted"])

      assert {:ok, package} =
               RetrievalCoordinator.retrieve("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:test",
                 allowed_partitions: ["public"],
                 allowed_security_labels: ["internal"]
               )

      assert package.fact_links == []
      assert package.filtered_object_summary.redacted_or_filtered_objects == 2

      redacted_ids = Enum.map(package.redacted_object_links, & &1.id)
      assert seeded.fact.id in redacted_ids
      assert seeded.memory.id in redacted_ids

      assert {:ok, [[stored_links]]} =
               Store.raw_query(
                 "SELECT redacted_object_links FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, package.id]
               )

      assert stored_links =~ seeded.fact.id
      assert stored_links =~ "partition_filtered"
    end
  end

  describe "fail-closed authorization envelope" do
    test "an empty envelope excludes security-labeled and partitioned objects" do
      workspace_id = unique_workspace("governed-recall-fail-closed")

      labeled =
        insert_fact!(workspace_id, %{
          fact_text: "Labeled launch detail.",
          security_labels: ["internal"],
          partition_ids: []
        })

      partitioned =
        insert_fact!(workspace_id, %{
          fact_text: "Partitioned launch detail.",
          security_labels: [],
          partition_ids: ["restricted"]
        })

      open =
        insert_fact!(workspace_id, %{
          fact_text: "Open launch note.",
          security_labels: [],
          partition_ids: []
        })

      # No allowed_partitions / allowed_security_labels: fail closed.
      assert {:ok, package} =
               RetrievalCoordinator.retrieve_package("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:test"
               )

      assert Enum.map(package.facts, & &1.id) == [open.id]
      assert package.filtered_summary.candidate_facts == 3
      assert package.filtered_summary.returned_facts == 1
      assert package.filtered_summary.redacted_or_filtered_objects == 2

      redacted_by_id = Map.new(package.redacted_object_links, &{&1.id, &1})
      assert redacted_by_id[labeled.id].reasons == ["security_label_filtered"]
      assert redacted_by_id[partitioned.id].reasons == ["partition_filtered"]

      assert package.retrieval_plan.filters.authorization.empty_envelope_behavior ==
               "fail_closed"
    end

    test "an explicit envelope still admits matching labeled and partitioned objects" do
      workspace_id = unique_workspace("governed-recall-explicit-envelope")

      fact =
        insert_fact!(workspace_id, %{
          security_labels: ["internal"],
          partition_ids: ["restricted"]
        })

      assert {:ok, package} =
               RetrievalCoordinator.retrieve_package("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:test",
                 allowed_partitions: ["restricted"],
                 allowed_security_labels: ["internal"]
               )

      assert Enum.map(package.facts, & &1.id) == [fact.id]
      assert package.filtered_summary.redacted_or_filtered_objects == 0
    end
  end

  describe "tenant scoping" do
    test "candidate SQL excludes other tenants' rows that share a workspace id" do
      workspace_id = unique_workspace("governed-recall-tenant")

      fact_a =
        insert_fact!(workspace_id, %{
          tenant_id: "tenant-a",
          security_labels: [],
          partition_ids: []
        })

      _fact_b =
        insert_fact!(workspace_id, %{
          tenant_id: "tenant-b",
          security_labels: [],
          partition_ids: [],
          aggregate_confidence: 0.99
        })

      assert {:ok, package} =
               RetrievalCoordinator.retrieve_package("launch",
                 tenant_id: "tenant-a",
                 workspace_id: workspace_id,
                 actor_id: "agent:test"
               )

      assert Enum.map(package.facts, & &1.id) == [fact_a.id]
      assert package.tenant_id == "tenant-a"
      assert package.retrieval_plan.filters.tenant_id == "tenant-a"

      # Tenant B's row never entered the candidate set: it is scoped out,
      # not redacted.
      assert package.filtered_summary.candidate_facts == 1
      assert package.filtered_summary.redacted_or_filtered_objects == 0
    end

    test "ScopeEnvelope derives a tenant-scoped default workspace id" do
      scope = ScopeEnvelope.resolve(tenant_id: "exampleorg")
      assert scope.workspace_id == "exampleorg:default"
      assert :workspace_id in scope.unresolved

      # The default tenant keeps the bare "default" workspace.
      assert ScopeEnvelope.resolve([]).workspace_id == "default"

      # An explicit workspace always wins over the derived default.
      assert ScopeEnvelope.resolve(tenant_id: "exampleorg", workspace_id: "ws-9").workspace_id ==
               "ws-9"

      assert ScopeEnvelope.default_workspace_id("default") == "default"
      assert ScopeEnvelope.default_workspace_id("exampleorg") == "exampleorg:default"
    end

    test "Store readers apply the tenant predicate when a tenant is given" do
      workspace_id = unique_workspace("governed-recall-store-tenant")
      fact = insert_fact!(workspace_id, %{tenant_id: "tenant-a"})

      # Workspace-only reads keep working (single-workspace compatibility).
      assert {:ok, _} = MemoryCoreStore.get_fact(workspace_id, fact.id)

      assert {:ok, _} = MemoryCoreStore.get_fact(workspace_id, fact.id, tenant_id: "tenant-a")

      assert {:error, :not_found} =
               MemoryCoreStore.get_fact(workspace_id, fact.id, tenant_id: "tenant-b")

      assert {:ok, [%{id: listed_id}]} =
               MemoryCoreStore.list_facts(workspace_id, tenant_id: "tenant-a")

      assert listed_id == fact.id
      assert {:ok, []} = MemoryCoreStore.list_facts(workspace_id, tenant_id: "tenant-b")
    end
  end

  defp link_id(%{"id" => id}), do: id
  defp link_id(%{id: id}), do: id
  defp link_id(_), do: nil
end
