defmodule OptimalEngine.MemoryCore.WorkflowSkillPromotionTest do
  @moduledoc """
  Capture-to-promote proof for procedural knowledge:

  recording two similar Workflow Traces of the same family promotes the pattern
  into a Generalized Workflow, a Procedural Memory Object, and a draft (disabled)
  Skill Package, each linked back to every supporting trace.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.MemoryCore.Store
  alias OptimalEngine.MemoryCore.WorkflowSkill
  alias OptimalEngine.Topology.Skill

  test "two similar traces promote into a skill package linked to the traces" do
    workspace_id = unique_workspace()
    family = "ingest_signal"

    steps = [
      %{action: "search", target: "kb"},
      %{action: "classify"},
      %{action: "write_signal"}
    ]

    {:ok, _t1} =
      WorkflowSkill.record_trace(steps,
        workflow_family: family,
        workspace_id: workspace_id,
        subject_anchor: "pricing",
        evidence_links: [%{type: "signal", id: "sig:one"}],
        auto_promote: false
      )

    {:ok, _t2} =
      WorkflowSkill.record_trace(steps,
        workflow_family: family,
        workspace_id: workspace_id,
        subject_anchor: "pricing",
        evidence_links: [%{type: "signal", id: "sig:two"}],
        auto_promote: false
      )

    assert {:ok, traces} = Store.list_workflow_traces(workspace_id, workflow_family: family)
    assert length(traces) == 2

    assert {:ok, result} =
             WorkflowSkill.promote_repeated(family,
               workspace_id: workspace_id,
               subject_anchor: "pricing"
             )

    %{skill_package: pkg, procedure: proc, workflow: wf, traces: promoted} = result

    # Promotion folded BOTH traces into one generalized workflow.
    assert length(promoted) == 2
    assert length(wf.workflow_trace_links) == 2

    # Skill packages start disabled and in draft until governance enables them.
    assert pkg.enabled_state == "disabled"
    assert pkg.review_status == "draft"
    assert pkg.skill_package_name == family

    # Chain links back: skill -> procedure -> workflow.
    proc_ref = %{type: "procedural_memory_object", id: proc.id}
    assert proc_ref in pkg.procedural_memory_links

    wf_ref = %{type: "generalized_workflow", id: wf.id}
    assert wf_ref in proc.generalized_workflow_links

    # Everything is persisted and listable through the store.
    assert {:ok, [_ | _] = pkgs} = Store.list_skill_packages(workspace_id)
    assert Enum.any?(pkgs, &(&1.id == pkg.id))
    assert {:ok, [_ | _]} = Store.list_generalized_workflows(workspace_id)
    assert {:ok, [_ | _]} = Store.list_procedural_memory_objects(workspace_id)
  end

  test "promotion registers the terminal capability in the skills registry" do
    workspace_id = unique_workspace()
    # Unique family => unique skill name => provable 0 -> 1 in the skills registry.
    family = "register_capability_flow_#{System.unique_integer([:positive])}"
    steps = [%{action: "a"}, %{action: "b"}]

    # 0: capability does not exist yet.
    assert {:ok, before} = Skill.list()
    refute Enum.any?(before, &(&1.name == family))

    {:ok, _t1} =
      WorkflowSkill.record_trace(steps,
        workflow_family: family,
        workspace_id: workspace_id,
        auto_promote: false
      )

    {:ok, _t2} =
      WorkflowSkill.record_trace(steps,
        workflow_family: family,
        workspace_id: workspace_id,
        auto_promote: false
      )

    assert {:ok, %{skill_package: pkg}} =
             WorkflowSkill.promote_repeated(family, workspace_id: workspace_id)

    # >=1: the real promotion flow registered a named, tenant-scoped capability.
    assert {:ok, after_skills} = Skill.list()
    registered = Enum.filter(after_skills, &(&1.name == family))
    assert length(registered) == 1

    [skill] = registered
    assert skill.kind == :domain
    assert skill.tenant_id == pkg.tenant_id
    assert skill.description =~ pkg.id
  end

  test "a single trace stays below threshold" do
    workspace_id = unique_workspace()

    {:ok, _t} =
      WorkflowSkill.record_trace([%{action: "do_thing"}],
        workflow_family: "lonely_workflow",
        workspace_id: workspace_id,
        auto_promote: false
      )

    assert {:ok, :below_threshold} =
             WorkflowSkill.promote_repeated("lonely_workflow", workspace_id: workspace_id)

    assert {:ok, []} = Store.list_skill_packages(workspace_id)
  end

  test "record_trace auto-promotes once the threshold is reached" do
    workspace_id = unique_workspace()
    family = "auto_promote_flow"
    steps = [%{action: "a"}, %{action: "b"}]

    {:ok, _} =
      WorkflowSkill.record_trace(steps, workflow_family: family, workspace_id: workspace_id)

    {:ok, second} =
      WorkflowSkill.record_trace(steps, workflow_family: family, workspace_id: workspace_id)

    assert Map.has_key?(second, :promoted_skill_package)
    assert {:ok, [_ | _]} = Store.list_skill_packages(workspace_id)
  end

  test "record_trace requires a workflow_family and at least one step" do
    assert {:error, :workflow_family_required} = WorkflowSkill.record_trace([%{action: "x"}], [])

    assert {:error, :no_steps} =
             WorkflowSkill.record_trace([], workflow_family: "empty")
  end

  defp unique_workspace,
    do: "workflow-skill-promotion-test-#{System.unique_integer([:positive])}"
end
