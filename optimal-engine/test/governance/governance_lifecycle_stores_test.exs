defmodule OptimalEngine.GovernanceLifecycleStoresTest do
  @moduledoc """
  Drives each store in the "governance lifecycle + eval + misc" cluster from
  0 -> >=1 row via its REAL public flow, asserting shape + provenance:

    * retention_policies        Compliance.add_retention_policy/1 (new caller)
    * legal_holds               Compliance.place_legal_hold/4
    * topology_change_requests  WorkspaceTopology.propose_change/1 (governed edit)
    * evaluation_runs           Evaluation.record_run/1
    * evaluation_cases          Evaluation.record_case/2
    * audiences                 Governance.Audience.create/1 (new caller)
    * architectures             Architecture.register/2
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.{Architecture, Compliance, Evaluation, Store, WorkspaceTopology}
  alias OptimalEngine.Architecture.Architecture, as: Arch
  alias OptimalEngine.Governance.Audience

  defp count(table, tenant) do
    {:ok, [[n]]} =
      Store.raw_query("SELECT COUNT(*) FROM #{table} WHERE tenant_id = ?1", [tenant])

    n
  end

  setup do
    {:ok, suffix: System.unique_integer([:positive])}
  end

  test "retention_policies: add_retention_policy/1 goes 0 -> 1 with correct shape", %{
    suffix: suffix
  } do
    tenant = "gov-ret-#{suffix}"
    assert count("retention_policies", tenant) == 0

    assert {:ok, id} =
             Compliance.add_retention_policy(%{
               tenant_id: tenant,
               scope_type: "genre",
               scope_value: "transcript",
               ttl_days: 30,
               action: "archive"
             })

    assert is_integer(id)
    assert count("retention_policies", tenant) == 1

    {:ok, [[scope_type, scope_value, ttl, action]]} =
      Store.raw_query(
        "SELECT scope_type, scope_value, ttl_days, action FROM retention_policies WHERE tenant_id = ?1",
        [tenant]
      )

    assert scope_type == "genre"
    assert scope_value == "transcript"
    assert ttl == 30
    assert action == "archive"
  end

  test "legal_holds: place_legal_hold/4 goes 0 -> 1 with provenance", %{suffix: suffix} do
    tenant = "gov-hold-#{suffix}"
    assert count("legal_holds", tenant) == 0

    signal_id = "sig-#{suffix}"

    assert {:ok, hold_id} =
             Compliance.place_legal_hold(signal_id, "operator-#{suffix}", "litigation hold",
               tenant_id: tenant
             )

    assert is_integer(hold_id)
    assert count("legal_holds", tenant) == 1
    assert Compliance.LegalHold.held?(signal_id, tenant)

    [hold] = Compliance.active_legal_holds(tenant)
    assert hold.signal_id == signal_id
    assert hold.held_by == "operator-#{suffix}"
    assert hold.reason == "litigation hold"
  end

  test "topology_change_requests: propose_change/1 (governed edit) goes 0 -> 1", %{suffix: suffix} do
    tenant = "gov-tcr-#{suffix}"
    workspace = "ws-#{suffix}"
    assert count("topology_change_requests", tenant) == 0

    assert {:ok, request} =
             WorkspaceTopology.propose_change(%{
               tenant_id: tenant,
               workspace_id: workspace,
               request_type: "create_node",
               target_object_type: "node",
               proposed_payload: %{"slug" => "new-node-#{suffix}", "name" => "New Node"},
               reason: "exercise governed edit path",
               requested_by: "agent-#{suffix}"
             })

    assert request.review_status == "pending"
    assert request.lifecycle_state == "open"
    assert request.requested_by == "agent-#{suffix}"
    assert count("topology_change_requests", tenant) == 1

    # close the loop: governed review transitions lifecycle_state
    assert {:ok, %{request: reviewed}} =
             WorkspaceTopology.review_change_request(request.id, :approve,
               tenant_id: tenant,
               workspace_id: workspace,
               reviewed_by: "reviewer-#{suffix}"
             )

    assert reviewed.review_status == "approved"
  end

  test "evaluation_runs + evaluation_cases: record_run/1 + record_case/2 go 0 -> 1", %{
    suffix: suffix
  } do
    tenant = "gov-eval-#{suffix}"
    assert count("evaluation_runs", tenant) == 0
    assert count("evaluation_cases", tenant) == 0

    assert {:ok, run} =
             Evaluation.record_run(%{
               tenant_id: tenant,
               benchmark_name: "governance-smoke",
               question_count: 1,
               answer_model: "test-model",
               judge_model: "test-judge",
               created_by: "agent-#{suffix}"
             })

    assert run.tenant_id == tenant
    assert run.benchmark_name == "governance-smoke"
    assert count("evaluation_runs", tenant) == 1

    assert {:ok, ec} =
             Evaluation.record_case(run.id, %{
               case_id: "q1",
               question: "what is the governance lifecycle?",
               expected_answer: "policies + holds + reviews",
               actual_answer: "policies + holds + reviews",
               scores: %{"correctness" => 1.0},
               status: "passed"
             })

    assert ec.evaluation_run_id == run.id
    assert ec.case_id == "q1"
    assert count("evaluation_cases", tenant) == 1

    # provenance: case links back to its run
    {:ok, [[linked_run]]} =
      Store.raw_query("SELECT evaluation_run_id FROM evaluation_cases WHERE id = ?1", [ec.id])

    assert linked_run == run.id
  end

  test "audiences: Audience.create/1 goes 0 -> 1 with role bindings", %{suffix: suffix} do
    tenant = "gov-aud-#{suffix}"
    assert count("audiences", tenant) == 0

    assert {:ok, audience} =
             Audience.create(%{
               tenant_id: tenant,
               name: "developers",
               role_ids: ["role-dev-#{suffix}", "role-lead-#{suffix}"],
               description: "Receivers who decode specs"
             })

    assert audience.name == "developers"
    assert count("audiences", tenant) == 1

    assert {:ok, fetched} = Audience.get(audience.id, tenant_id: tenant)
    assert fetched.role_ids == ["role-dev-#{suffix}", "role-lead-#{suffix}"]
    assert [%{name: "developers"}] = Audience.list(tenant_id: tenant)
  end

  test "architectures: Architecture.register/2 goes 0 -> 1 with spec", %{suffix: suffix} do
    tenant = "gov-arch-#{suffix}"
    assert count("architectures", tenant) == 0

    arch =
      Arch.new(
        name: "governance-doc-#{suffix}",
        version: 1,
        description: "An architecture doc ingested for the governance cluster",
        modality_primary: :text
      )

    assert :ok = Architecture.register(arch, tenant_id: tenant)
    assert count("architectures", tenant) == 1

    {:ok, [[name, modality, spec]]} =
      Store.raw_query(
        "SELECT name, modality_primary, spec FROM architectures WHERE tenant_id = ?1",
        [tenant]
      )

    assert name == "governance-doc-#{suffix}"
    assert modality == "text"
    assert {:ok, _decoded} = Jason.decode(spec)
  end
end
