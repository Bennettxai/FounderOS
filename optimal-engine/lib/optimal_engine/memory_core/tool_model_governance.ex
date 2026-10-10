defmodule OptimalEngine.MemoryCore.ToolModelGovernance do
  @moduledoc """
  First Tool/Model governance runtime slice.

  This module does not execute external tools or models. It proves the control
  plane:

  * definitions are registered as governed catalog objects;
  * attempted calls are validated before they are allowed;
  * allowed and rejected attempts are recorded as durable run rows;
  * optional outputs can become Active Memory Pool observations.
  """

  alias OptimalEngine.Audit.Event, as: AuditEvent
  alias OptimalEngine.Store, as: EngineStore

  alias OptimalEngine.MemoryCore.{
    ActiveMemoryPool,
    ID,
    Store
  }

  require Logger

  @spec register_model_call_operation(keyword()) :: {:ok, map()} | {:error, term()}
  def register_model_call_operation(opts) when is_list(opts) do
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = string_opt(opts, :workspace_id, "default")
    function_name = string_opt(opts, :function_name, nil)
    model_task_type = string_opt(opts, :model_task_type, "generation")

    operation = %{
      id:
        ID.content_id("mco", [
          tenant_id,
          ":",
          workspace_id,
          ":",
          function_name,
          ":",
          model_task_type
        ]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      function_name: function_name,
      model_task_type: model_task_type,
      execution_mode: string_opt(opts, :execution_mode, "sync"),
      risk_class: string_opt(opts, :risk_class, "low"),
      model_id: string_or_nil(Keyword.get(opts, :model_id)),
      model_version: string_or_nil(Keyword.get(opts, :model_version)),
      prompt_template_id: string_or_nil(Keyword.get(opts, :prompt_template_id)),
      input_schema: Keyword.get(opts, :input_schema, %{}),
      output_contract: Keyword.get(opts, :output_contract, %{}),
      execution_policy: Keyword.get(opts, :execution_policy, %{}),
      storage_target: Keyword.get(opts, :storage_target, %{}),
      prompt_policy_id: string_or_nil(Keyword.get(opts, :prompt_policy_id)),
      cost_policy: Keyword.get(opts, :cost_policy, %{}),
      expected_confidence_behavior: Keyword.get(opts, :expected_confidence_behavior, %{}),
      required_privileges: list_opt(opts, :required_privileges),
      allowed_partitions: list_opt(opts, :allowed_partitions),
      lifecycle_state: string_opt(opts, :lifecycle_state, "active"),
      rollout_state: string_opt(opts, :rollout_state, "enabled"),
      suspension_reason: string_or_nil(Keyword.get(opts, :suspension_reason)),
      retirement_status: string_opt(opts, :retirement_status, "active"),
      transaction_time_start: timestamp(),
      access_policy_id: string_or_nil(Keyword.get(opts, :access_policy_id)),
      security_labels: list_opt(opts, :security_labels),
      partition_ids: list_opt(opts, :partition_ids),
      metadata: Keyword.get(opts, :metadata, %{})
    }

    with :ok <- Store.insert_model_call_operation(operation) do
      {:ok, operation}
    end
  end

  @spec register_mcp_tool_definition(keyword()) :: {:ok, map()} | {:error, term()}
  def register_mcp_tool_definition(opts) when is_list(opts) do
    tenant_id = string_opt(opts, :tenant_id, "default")
    workspace_id = string_opt(opts, :workspace_id, "default")
    tool_name = string_opt(opts, :tool_name, nil)
    protocol_adapter_id = string_opt(opts, :protocol_adapter_id, "mcp")

    definition = %{
      id:
        ID.content_id("mcp_tool", [
          tenant_id,
          ":",
          workspace_id,
          ":",
          protocol_adapter_id,
          ":",
          tool_name
        ]),
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      tool_name: tool_name,
      protocol_adapter_id: protocol_adapter_id,
      implementation_type: string_opt(opts, :implementation_type, "external"),
      enabled_state: string_opt(opts, :enabled_state, "enabled"),
      registration_source: string_or_nil(Keyword.get(opts, :registration_source)),
      documentation_links: list_opt(opts, :documentation_links),
      required_privileges: list_opt(opts, :required_privileges),
      allowed_partitions: list_opt(opts, :allowed_partitions),
      input_schema: Keyword.get(opts, :input_schema, %{}),
      output_schema: Keyword.get(opts, :output_schema, %{}),
      execution_policy: Keyword.get(opts, :execution_policy, %{}),
      routing_policy: Keyword.get(opts, :routing_policy, %{}),
      timeout_policy: Keyword.get(opts, :timeout_policy, %{}),
      cost_policy: Keyword.get(opts, :cost_policy, %{}),
      audit_policy: Keyword.get(opts, :audit_policy, %{}),
      lifecycle_state: string_opt(opts, :lifecycle_state, "active"),
      suspension_reason: string_or_nil(Keyword.get(opts, :suspension_reason)),
      retirement_status: string_opt(opts, :retirement_status, "active"),
      transaction_time_start: timestamp(),
      access_policy_id: string_or_nil(Keyword.get(opts, :access_policy_id)),
      security_labels: list_opt(opts, :security_labels),
      partition_ids: list_opt(opts, :partition_ids),
      policy_version: string_or_nil(Keyword.get(opts, :policy_version)),
      metadata: Keyword.get(opts, :metadata, %{})
    }

    with :ok <- Store.insert_mcp_tool_definition(definition) do
      {:ok, definition}
    end
  end

  @spec record_model_call(String.t(), map(), keyword()) :: {:ok, map()} | {:error, term()}
  def record_model_call(function_name, input_payload, opts \\ [])
      when is_binary(function_name) and is_map(input_payload) do
    workspace_id = string_opt(opts, :workspace_id, "default")

    case Store.get_model_call_operation(workspace_id, function_name) do
      {:ok, operation} ->
        decision = decide(operation, input_payload, opts, :model)
        run = build_model_run(operation, input_payload, decision, opts)
        persist_run(run, decision, opts, :model)

      {:error, reason} ->
        {:error, reason}
    end
  end

  @spec record_tool_call(String.t(), map(), keyword()) :: {:ok, map()} | {:error, term()}
  def record_tool_call(tool_name, input_payload, opts \\ [])
      when is_binary(tool_name) and is_map(input_payload) do
    workspace_id = string_opt(opts, :workspace_id, "default")
    protocol_adapter_id = string_opt(opts, :protocol_adapter_id, "mcp")

    case Store.get_mcp_tool_definition(workspace_id, tool_name, protocol_adapter_id) do
      {:ok, definition} ->
        decision = decide(definition, input_payload, opts, :tool)
        run = build_tool_run(definition, input_payload, decision, opts)
        persist_run(run, decision, opts, :tool)

      {:error, reason} ->
        {:error, reason}
    end
  end

  @spec execute_tool_call(String.t(), map(), function(), keyword()) ::
          {:ok, map()} | {:error, term()}
  def execute_tool_call(tool_name, input_payload, executor, opts \\ [])
      when is_binary(tool_name) and is_map(input_payload) and is_function(executor) do
    workspace_id = string_opt(opts, :workspace_id, "default")
    protocol_adapter_id = string_opt(opts, :protocol_adapter_id, "mcp")

    case Store.get_mcp_tool_definition(workspace_id, tool_name, protocol_adapter_id) do
      {:ok, definition} ->
        decision = decide(definition, input_payload, opts, :tool)

        if decision.allowed? do
          execute_allowed_tool(definition, input_payload, executor, opts)
        else
          run = build_tool_run(definition, input_payload, decision, opts)

          case persist_run(run, decision, opts, :tool) do
            {:ok, rejected_run} -> {:error, {:rejected, rejected_run}}
            other -> other
          end
        end

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp persist_run(run, decision, opts, kind) do
    run =
      if decision.allowed? and run.run_status == "completed" do
        attach_observation(run, opts, kind)
      else
        run
      end
      |> attach_audit_event(kind)

    insert_result =
      case kind do
        :model -> Store.insert_model_call_run(run)
        :tool -> Store.insert_tool_call_run(run)
      end

    with :ok <- insert_result do
      {:ok, run}
    end
  end

  defp execute_allowed_tool(definition, input_payload, executor, opts) do
    decision = decide(definition, input_payload, opts, :tool)
    started_at = System.monotonic_time(:millisecond)

    case invoke_executor(executor, input_payload) do
      {:ok, output_payload} ->
        exec_opts =
          opts
          |> Keyword.put(:output_payload, output_payload)
          |> Keyword.put(:latency_ms, elapsed_ms(started_at, opts))

        run = build_tool_run(definition, input_payload, decision, exec_opts)

        case persist_run(run, decision, exec_opts, :tool) do
          {:ok, %{run_status: "output_rejected"} = output_rejected_run} ->
            {:error, {:output_rejected, output_rejected_run}}

          other ->
            other
        end

      {:error, reason} ->
        failed_opts =
          opts
          |> Keyword.put(:output_payload, %{error: inspect(reason)})
          |> Keyword.put(:latency_ms, elapsed_ms(started_at, opts))

        run =
          definition
          |> build_tool_run(input_payload, decision, failed_opts)
          |> Map.put(:run_status, "failed")
          |> Map.put(:rejection_reason, "execution_failed:#{inspect(reason)}")

        case persist_run(run, decision, failed_opts, :tool) do
          {:ok, failed_run} -> {:error, {:execution_failed, reason, failed_run}}
          other -> other
        end
    end
  end

  defp invoke_executor(executor, input_payload) do
    result =
      try do
        case :erlang.fun_info(executor, :arity) do
          {:arity, 0} -> executor.()
          {:arity, 1} -> executor.(input_payload)
          {:arity, _} -> {:error, :unsupported_executor_arity}
        end
      rescue
        exception -> {:error, {exception.__struct__, Exception.message(exception)}}
      catch
        kind, reason -> {:error, {kind, reason}}
      end

    normalize_executor_result(result)
  end

  defp normalize_executor_result({:ok, payload}) when is_map(payload), do: {:ok, payload}
  defp normalize_executor_result({:ok, payload}), do: {:ok, %{result: inspect(payload)}}
  defp normalize_executor_result(:ok), do: {:ok, %{result: "ok"}}
  defp normalize_executor_result(payload) when is_map(payload), do: {:ok, payload}
  defp normalize_executor_result({:error, reason}), do: {:error, reason}
  defp normalize_executor_result(other), do: {:ok, %{result: inspect(other)}}

  defp elapsed_ms(started_at, opts) do
    Keyword.get(opts, :latency_ms) || max(System.monotonic_time(:millisecond) - started_at, 0)
  end

  # Tool/model output never becomes a Fact directly: it is published into the
  # Active Memory Pool as an observation, which preserves the raw output as a
  # Source Package and creates a pending Claim. The run row links forward to
  # the observation Claim, and the Claim/Source Package carry the run ID back.
  # Observation failures are recorded on the run instead of being swallowed.
  defp attach_observation(run, opts, kind) do
    observation_text = Keyword.get(opts, :observation_text)
    pool_id = Map.get(run, :active_memory_pool_id)

    cond do
      is_nil(observation_text) or observation_text == "" ->
        run

      is_nil(pool_id) ->
        run

      true ->
        case ActiveMemoryPool.publish_observation(pool_id, observation_text,
               actor_id: Map.get(run, :requesting_actor_id),
               extracted_by: run_identity(run, kind),
               source_type: "tool_model_output",
               claim_text: Keyword.get(opts, :claim_text, observation_text),
               subject_anchor: Keyword.get(opts, :subject_anchor),
               action_class: Keyword.get(opts, :action_class),
               object_anchor: Keyword.get(opts, :object_anchor),
               observation_kind: "tool_model_output",
               source_metadata: run_link_metadata(run, kind),
               claim_metadata: run_link_metadata(run, kind)
             ) do
          {:ok, observation} ->
            run
            |> Map.put(:source_package_links, [
              %{type: "source_package", id: observation.source_package.id}
            ])
            |> Map.put(:observation_links, [%{type: "claim", id: observation.pending_claim.id}])

          {:error, reason} ->
            Map.put(run, :metadata, Map.put(run.metadata, :observation_error, inspect(reason)))
        end
    end
  end

  defp run_identity(run, :tool), do: "tool:#{run.tool_name}"
  defp run_identity(run, :model), do: "model:#{run.function_name}"

  defp run_link_metadata(run, kind),
    do: %{origin_run_type: "#{kind}_call_run", origin_run_id: run.id}

  defp decide(definition, input_payload, opts, kind) do
    state_check = state_check(definition, kind)

    privilege_check =
      subset_check(
        Map.get(definition, :required_privileges, []),
        list_opt(opts, :granted_privileges)
      )

    partition_check =
      subset_check(
        list_opt(opts, :requested_partitions),
        Map.get(definition, :allowed_partitions, [])
      )

    input_check =
      payload_schema_check(
        input_payload,
        Map.get(definition, :input_schema, %{}),
        "input"
      )

    checks = [
      state_check,
      privilege_check,
      partition_check,
      input_check
    ]

    reasons =
      checks
      |> Enum.reject(&elem(&1, 0))
      |> Enum.map(&elem(&1, 1))

    %{
      allowed?: reasons == [],
      rejection_reason: Enum.join(reasons, "; "),
      checks: %{
        state: elem(state_check, 0),
        privileges: elem(privilege_check, 0),
        partitions: elem(partition_check, 0),
        input_schema: elem(input_check, 0)
      }
    }
  end

  defp state_check(%{lifecycle_state: "active", rollout_state: "enabled"}, :model), do: {true, nil}
  defp state_check(%{lifecycle_state: "active", enabled_state: "enabled"}, :tool), do: {true, nil}
  defp state_check(_definition, _kind), do: {false, "definition_not_enabled"}

  defp subset_check([], _available), do: {true, nil}

  defp subset_check(required, available) do
    missing = Enum.reject(required, &(&1 in available))

    case missing do
      [] -> {true, nil}
      _ -> {false, "missing:#{Enum.join(missing, ",")}"}
    end
  end

  # Registered-schema validation: required keys plus JSON-Schema style
  # property type checks ("string", "integer", "number", "boolean", "object",
  # "array", "null"). `prefix` is "input" or "output"; reasons render as
  # "missing_input:key" / "invalid_output:key(expected type)".
  defp payload_schema_check(payload, schema, prefix) do
    payload = stringify_keys(payload)

    required =
      (Map.get(schema, "required") || Map.get(schema, :required) || [])
      |> Enum.map(&to_string/1)

    missing = Enum.reject(required, &Map.has_key?(payload, &1))

    type_violations =
      schema
      |> schema_property_types()
      |> Enum.flat_map(fn {name, expected_type} ->
        case Map.fetch(payload, name) do
          {:ok, value} ->
            if type_matches?(expected_type, value),
              do: [],
              else: ["invalid_#{prefix}:#{name}(expected #{expected_type})"]

          :error ->
            []
        end
      end)

    reasons =
      case missing do
        [] -> type_violations
        _ -> ["missing_#{prefix}:#{Enum.join(missing, ",")}" | type_violations]
      end

    case reasons do
      [] -> {true, nil}
      _ -> {false, Enum.join(reasons, "; ")}
    end
  end

  defp schema_property_types(schema) do
    (Map.get(schema, "properties") || Map.get(schema, :properties) || %{})
    |> Enum.flat_map(fn {name, spec} ->
      case property_type(spec) do
        nil -> []
        type -> [{to_string(name), type}]
      end
    end)
  end

  defp property_type(spec) when is_map(spec) do
    case Map.get(spec, "type") || Map.get(spec, :type) do
      nil -> nil
      type -> to_string(type)
    end
  end

  defp property_type(_spec), do: nil

  defp type_matches?("string", value), do: is_binary(value)
  defp type_matches?("integer", value), do: is_integer(value)
  defp type_matches?("number", value), do: is_number(value)
  defp type_matches?("boolean", value), do: is_boolean(value)
  defp type_matches?("object", value), do: is_map(value)
  defp type_matches?("array", value), do: is_list(value)
  defp type_matches?("null", value), do: is_nil(value)
  defp type_matches?(_type, _value), do: true

  defp stringify_keys(payload), do: Map.new(payload, fn {key, value} -> {to_string(key), value} end)

  defp build_model_run(operation, input_payload, decision, opts) do
    output_payload = Keyword.get(opts, :output_payload, %{})

    %{
      id: ID.random_id("modelrun"),
      tenant_id: operation.tenant_id,
      workspace_id: operation.workspace_id,
      model_call_operation_id: operation.id,
      function_name: operation.function_name,
      requesting_actor_id: string_or_nil(Keyword.get(opts, :actor_id)),
      active_memory_pool_id: string_or_nil(Keyword.get(opts, :active_memory_pool_id)),
      decision_state: decision_state(decision),
      run_status: run_status(decision, output_payload, opts),
      rejection_reason: reject_reason(decision),
      input_payload: input_payload,
      output_payload: output_payload,
      input_hash: payload_hash(input_payload),
      output_hash: payload_hash(output_payload),
      required_privileges: operation.required_privileges,
      granted_privileges: list_opt(opts, :granted_privileges),
      requested_partitions: list_opt(opts, :requested_partitions),
      allowed_partitions: operation.allowed_partitions,
      policy_decision: decision,
      latency_ms: Keyword.get(opts, :latency_ms),
      cost_units: Keyword.get(opts, :cost_units),
      source_package_links: [],
      observation_links: [],
      audit_event_links: [],
      access_policy_id: operation.access_policy_id,
      security_labels: operation.security_labels,
      partition_ids: operation.partition_ids,
      metadata: Keyword.get(opts, :metadata, %{})
    }
    |> apply_output_validation(operation.output_contract, opts, :output_contract)
  end

  defp build_tool_run(definition, input_payload, decision, opts) do
    output_payload = Keyword.get(opts, :output_payload, %{})

    %{
      id: ID.random_id("toolrun"),
      tenant_id: definition.tenant_id,
      workspace_id: definition.workspace_id,
      mcp_tool_definition_id: definition.id,
      tool_name: definition.tool_name,
      requesting_actor_id: string_or_nil(Keyword.get(opts, :actor_id)),
      active_memory_pool_id: string_or_nil(Keyword.get(opts, :active_memory_pool_id)),
      decision_state: decision_state(decision),
      run_status: run_status(decision, output_payload, opts),
      rejection_reason: reject_reason(decision),
      input_payload: input_payload,
      output_payload: output_payload,
      input_hash: payload_hash(input_payload),
      output_hash: payload_hash(output_payload),
      required_privileges: definition.required_privileges,
      granted_privileges: list_opt(opts, :granted_privileges),
      requested_partitions: list_opt(opts, :requested_partitions),
      allowed_partitions: definition.allowed_partitions,
      policy_decision: decision,
      latency_ms: Keyword.get(opts, :latency_ms),
      cost_units: Keyword.get(opts, :cost_units),
      source_package_links: [],
      observation_links: [],
      audit_event_links: [],
      access_policy_id: definition.access_policy_id,
      security_labels: definition.security_labels,
      partition_ids: definition.partition_ids,
      metadata: Keyword.get(opts, :metadata, %{})
    }
    |> apply_output_validation(definition.output_schema, opts, :output_schema)
  end

  defp decision_state(%{allowed?: true}), do: "allowed"
  defp decision_state(_decision), do: "rejected"

  defp run_status(%{allowed?: false}, _output_payload, _opts), do: "rejected"

  defp run_status(_decision, _output_payload, opts),
    do: if(output_payload_provided?(opts), do: "completed", else: "approved")

  defp apply_output_validation(%{decision_state: "rejected"} = run, _schema, _opts, _check_key),
    do: run

  defp apply_output_validation(run, schema, opts, check_key) do
    cond do
      not output_payload_provided?(opts) ->
        put_output_check(run, check_key, true)

      true ->
        case payload_schema_check(run.output_payload, schema || %{}, "output") do
          {true, _} ->
            put_output_check(run, check_key, true)

          {false, reason} ->
            run
            |> Map.put(:run_status, "output_rejected")
            |> Map.put(:rejection_reason, append_reason(run.rejection_reason, reason))
            |> put_output_check(check_key, false)
        end
    end
  end

  defp put_output_check(run, check_key, value) do
    checks = run.policy_decision |> Map.get(:checks, %{}) |> Map.put(check_key, value)
    Map.put(run, :policy_decision, Map.put(run.policy_decision, :checks, checks))
  end

  defp append_reason(nil, reason), do: reason
  defp append_reason("", reason), do: reason
  defp append_reason(existing, reason), do: existing <> "; " <> reason

  defp output_payload_provided?(opts), do: Keyword.has_key?(opts, :output_payload)

  defp reject_reason(%{allowed?: true}), do: nil
  defp reject_reason(%{rejection_reason: reason}), do: reason

  # The audit event is inserted FIRST so the run row carries the REAL
  # persisted audit event id — `INSERT ... RETURNING id` resolves the events
  # table's AUTOINCREMENT primary key in the same statement, so the run can be
  # joined to its audit event by id (Agent Work Loop gate: "Audit records the
  # run"). No fabricated/synthetic ids are ever stamped into
  # audit_event_links: if the audit write fails, the failure is recorded on
  # the run's metadata and the link list stays empty.
  defp attach_audit_event(run, kind) do
    case insert_audit_event(run, kind) do
      {:ok, audit_event_id} ->
        audit_link = %{
          type: "audit_event",
          id: audit_event_id,
          kind: audit_kind(kind)
        }

        Map.update(run, :audit_event_links, [audit_link], fn links -> links ++ [audit_link] end)

      {:error, reason} ->
        Logger.warning(
          "[ToolModelGovernance] Failed to record audit event for run #{run.id}: #{inspect(reason)}"
        )

        Map.put(run, :metadata, Map.put(run.metadata, :audit_link_error, inspect(reason)))
    end
  end

  defp insert_audit_event(run, kind) do
    event =
      AuditEvent.new(audit_kind(kind),
        tenant_id: run.tenant_id,
        principal: run.requesting_actor_id || "system",
        target_uri: "#{kind}:#{run.workspace_id}:#{run.id}",
        latency_ms: run.latency_ms,
        metadata: %{
          workspace_id: run.workspace_id,
          run_id: run.id,
          run_type: "#{kind}_call_run",
          decision_state: run.decision_state,
          run_status: run.run_status,
          rejection_reason: run.rejection_reason
        }
      )

    sql = """
    INSERT INTO events (tenant_id, workspace_id, ts, principal, kind, target_uri, latency_ms, metadata)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)
    RETURNING id
    """

    params = [
      event.tenant_id,
      run.workspace_id,
      event.ts,
      event.principal,
      event.kind,
      event.target_uri,
      event.latency_ms,
      Jason.encode!(event.metadata)
    ]

    case EngineStore.raw_query(sql, params) do
      {:ok, [[audit_event_id]]} when is_integer(audit_event_id) -> {:ok, audit_event_id}
      {:ok, other} -> {:error, {:unexpected_audit_insert_result, other}}
      {:error, reason} -> {:error, reason}
    end
  end

  defp audit_kind(kind), do: "#{kind}.call.governed"

  defp payload_hash(payload), do: Jason.encode!(payload) |> ID.sha256()

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()

  defp list_opt(opts, key), do: Keyword.get(opts, key, []) |> List.wrap() |> Enum.map(&to_string/1)

  defp string_opt(opts, key, nil) do
    case Keyword.get(opts, key) do
      nil -> nil
      value -> to_string(value)
    end
  end

  defp string_opt(opts, key, default), do: Keyword.get(opts, key, default) |> to_string()

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)
end
