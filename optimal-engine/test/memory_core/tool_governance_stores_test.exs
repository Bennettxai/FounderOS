defmodule OptimalEngine.MemoryCore.ToolGovernanceStoresTest do
  @moduledoc """
  Cluster proof: Tool governance + processor runs.

  Drives a REAL flow for each store in the cluster and asserts the store goes
  from 0 -> >=1 row with correct shape + provenance:

    * mcp_tool_definitions — ToolModelGovernance.register_mcp_tool_definition/1
    * tool_call_runs       — ToolModelGovernance.record_tool_call/3 (permission
                             + schema validation + audit) routes a real call
    * processor_runs       — Architecture.Apply.run/3 dispatches a field to its
                             registered processor and records the invocation
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Architecture.Apply
  alias OptimalEngine.Architecture.Architecture
  alias OptimalEngine.Architecture.Field
  alias OptimalEngine.MemoryCore.ToolModelGovernance
  alias OptimalEngine.Store

  defp unique_workspace, do: "tool-gov-stores-#{System.unique_integer([:positive])}"

  defp count(sql, params) do
    {:ok, [[n]]} = Store.raw_query(sql, params)
    n
  end

  test "mcp_tool_definitions: registration writes one governed catalog row (0 -> 1)" do
    workspace_id = unique_workspace()
    tool_name = "fs.read_dir"

    assert 0 ==
             count(
               "SELECT COUNT(*) FROM mcp_tool_definitions WHERE workspace_id = ?1 AND tool_name = ?2",
               [workspace_id, tool_name]
             )

    assert {:ok, definition} =
             ToolModelGovernance.register_mcp_tool_definition(
               workspace_id: workspace_id,
               tool_name: tool_name,
               required_privileges: ["fs:read"],
               allowed_partitions: ["sandbox"],
               input_schema: %{
                 "required" => ["path"],
                 "properties" => %{"path" => %{"type" => "string"}}
               },
               output_schema: %{"required" => ["entries"]}
             )

    assert 1 ==
             count(
               "SELECT COUNT(*) FROM mcp_tool_definitions WHERE workspace_id = ?1 AND tool_name = ?2",
               [workspace_id, tool_name]
             )

    # Shape + provenance.
    assert {:ok, [[stored_name, enabled, lifecycle, adapter]]} =
             Store.raw_query(
               "SELECT tool_name, enabled_state, lifecycle_state, protocol_adapter_id FROM mcp_tool_definitions WHERE id = ?1",
               [definition.id]
             )

    assert stored_name == tool_name
    assert enabled == "enabled"
    assert lifecycle == "active"
    assert adapter == "mcp"
  end

  test "tool_call_runs: record_tool_call routes an allowed call through governance (0 -> 1) and a rejected call (1 -> 2)" do
    workspace_id = unique_workspace()
    tool_name = "fs.read_dir"

    {:ok, definition} =
      ToolModelGovernance.register_mcp_tool_definition(
        workspace_id: workspace_id,
        tool_name: tool_name,
        required_privileges: ["fs:read"],
        allowed_partitions: ["sandbox"],
        input_schema: %{
          "required" => ["path"],
          "properties" => %{"path" => %{"type" => "string"}}
        }
      )

    assert 0 ==
             count(
               "SELECT COUNT(*) FROM tool_call_runs WHERE mcp_tool_definition_id = ?1",
               [definition.id]
             )

    # Allowed call: privilege satisfied, partition allowed, input schema valid.
    assert {:ok, allowed_run} =
             ToolModelGovernance.record_tool_call(
               tool_name,
               %{path: "/tmp"},
               workspace_id: workspace_id,
               actor_id: "agent:loop",
               granted_privileges: ["fs:read"],
               requested_partitions: ["sandbox"]
             )

    assert allowed_run.decision_state == "allowed"
    assert allowed_run.mcp_tool_definition_id == definition.id

    assert 1 ==
             count(
               "SELECT COUNT(*) FROM tool_call_runs WHERE mcp_tool_definition_id = ?1",
               [definition.id]
             )

    # Rejected call: missing privilege -> still recorded as a durable run row.
    assert {:ok, rejected_run} =
             ToolModelGovernance.record_tool_call(
               tool_name,
               %{path: "/tmp"},
               workspace_id: workspace_id,
               actor_id: "agent:loop",
               granted_privileges: [],
               requested_partitions: ["sandbox"]
             )

    assert rejected_run.decision_state == "rejected"
    assert rejected_run.rejection_reason =~ "missing:fs:read"

    assert 2 ==
             count(
               "SELECT COUNT(*) FROM tool_call_runs WHERE mcp_tool_definition_id = ?1",
               [definition.id]
             )

    # Provenance: each run links to a real persisted audit event in `events`.
    assert {:ok, [[decision_state, run_status, actor]]} =
             Store.raw_query(
               "SELECT decision_state, run_status, requesting_actor_id FROM tool_call_runs WHERE id = ?1",
               [allowed_run.id]
             )

    assert decision_state == "allowed"
    assert run_status == "approved"
    assert actor == "agent:loop"

    [audit_link | _] = allowed_run.audit_event_links
    assert %{type: "audit_event"} = audit_link

    assert 1 ==
             count("SELECT COUNT(*) FROM events WHERE id = ?1", [audit_link.id])
  end

  test "processor_runs: Apply.run dispatches a field to its processor and records the invocation (0 -> 1)" do
    tenant_id = "default"
    context_id = "ctx-#{System.unique_integer([:positive])}"
    arch_id = "ts-arch-#{System.unique_integer([:positive])}"

    arch =
      Architecture.new(
        id: arch_id,
        name: "metric_series",
        modality_primary: :time_series,
        fields: [
          %Field{
            name: :samples,
            modality: :time_series,
            processor: :ts_feature_extractor,
            required: true
          }
        ]
      )

    assert 0 ==
             count(
               "SELECT COUNT(*) FROM processor_runs WHERE context_id = ?1 AND architecture_id = ?2",
               [context_id, arch_id]
             )

    assert {:ok, outputs} =
             Apply.run(arch, %{samples: [1, 2, 3, 4, 5]},
               context_id: context_id,
               tenant_id: tenant_id
             )

    # The real processor ran and emitted features.
    assert %{kind: :features, value: %{count: 5, trend: 1}} = outputs.samples

    assert 1 ==
             count(
               "SELECT COUNT(*) FROM processor_runs WHERE context_id = ?1 AND architecture_id = ?2",
               [context_id, arch_id]
             )

    # Shape + provenance: status, processor + field captured, metadata records emit kind.
    assert {:ok, [[processor, field, status, metadata]]} =
             Store.raw_query(
               "SELECT processor, field, status, metadata FROM processor_runs WHERE context_id = ?1 AND architecture_id = ?2",
               [context_id, arch_id]
             )

    assert processor == "ts_feature_extractor"
    assert field == "samples"
    assert status == "success"
    assert %{"kind" => "features"} = Jason.decode!(metadata)
  end
end
