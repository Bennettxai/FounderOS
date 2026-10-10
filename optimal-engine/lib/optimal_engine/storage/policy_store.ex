defmodule OptimalEngine.Storage.PolicyStore do
  @moduledoc """
  Workspace-scoped storage policy and replication-ledger persistence.

  Policies contain use-case choices and provider identifiers only. Credentials
  stay in environment variables or a secret manager and are referenced by
  name. Sync mutations are append-only, idempotent, and explicitly scoped.
  """

  alias OptimalEngine.Storage.{Planner, Providers}
  alias OptimalEngine.Store

  @spec get_policy(String.t()) :: {:ok, map()} | {:error, :not_found | term()}
  def get_policy(workspace_id) when is_binary(workspace_id) do
    sql = """
    SELECT workspace_id, tenant_id, organization_id, use_cases,
           provider_overrides, policy_version, created_by, created_at, updated_at
    FROM workspace_storage_policies
    WHERE workspace_id = ?1
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, [row]} -> {:ok, policy_from_row(row)}
      {:ok, []} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec put_policy(String.t(), [String.t()], keyword()) :: {:ok, map()} | {:error, term()}
  def put_policy(workspace_id, use_cases, opts \\ [])
      when is_binary(workspace_id) and is_list(use_cases) do
    overrides = Keyword.get(opts, :provider_overrides, %{})

    with {:ok, _plan} <- Planner.plan(use_cases),
         :ok <- validate_overrides(overrides),
         {:ok, tenant_id, organization_id} <- workspace_scope(workspace_id),
         :ok <-
           Store.raw_execute(
             """
             INSERT INTO workspace_storage_policies (
               workspace_id, tenant_id, organization_id, use_cases,
               provider_overrides, policy_version, created_by
             ) VALUES (?1, ?2, ?3, ?4, ?5, 1, ?6)
             ON CONFLICT(workspace_id) DO UPDATE SET
               tenant_id = excluded.tenant_id,
               organization_id = excluded.organization_id,
               use_cases = excluded.use_cases,
               provider_overrides = excluded.provider_overrides,
               policy_version = workspace_storage_policies.policy_version + 1,
               created_by = excluded.created_by,
               updated_at = datetime('now')
             """,
             [
               workspace_id,
               tenant_id,
               organization_id,
               Jason.encode!(Enum.uniq(use_cases)),
               Jason.encode!(overrides),
               Keyword.get(opts, :actor_id)
             ]
           ) do
      get_policy(workspace_id)
    end
  end

  @spec append_mutation(map()) :: {:ok, map()} | {:error, term()}
  def append_mutation(attrs) when is_map(attrs) do
    required = ~w(workspace_id device_id entity_type entity_id operation)a

    with :ok <- require_fields(attrs, required),
         :ok <- validate_operation(attrs.operation),
         {:ok, tenant_id, organization_id} <- workspace_scope(attrs.workspace_id) do
      payload = Map.get(attrs, :payload)
      payload_json = if is_nil(payload), do: nil, else: Jason.encode!(payload)
      payload_hash = hash(payload_json || "")
      id = Map.get(attrs, :id, UUID.uuid4())
      occurred_at = Map.get(attrs, :occurred_at, DateTime.utc_now() |> DateTime.to_iso8601())
      idempotency_key = Map.get(attrs, :idempotency_key, id)

      case Store.raw_execute(
             """
             INSERT OR IGNORE INTO sync_mutations (
               id, tenant_id, organization_id, workspace_id, device_id, actor_id,
               entity_type, entity_id, operation, payload, payload_hash,
               idempotency_key, occurred_at
             ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13)
             """,
             [
               id,
               tenant_id,
               organization_id,
               attrs.workspace_id,
               attrs.device_id,
               Map.get(attrs, :actor_id),
               attrs.entity_type,
               attrs.entity_id,
               to_string(attrs.operation),
               payload_json,
               payload_hash,
               idempotency_key,
               occurred_at
             ]
           ) do
        :ok -> get_mutation_by_idempotency_key(idempotency_key)
        {:error, reason} -> {:error, reason}
      end
    end
  end

  @spec mutations_after(String.t(), non_neg_integer(), pos_integer()) ::
          {:ok, [map()]} | {:error, term()}
  def mutations_after(workspace_id, sequence, limit \\ 100)
      when is_binary(workspace_id) and is_integer(sequence) and sequence >= 0 do
    limit = limit |> max(1) |> min(1_000)

    case Store.raw_query(
           """
           SELECT sequence, id, tenant_id, organization_id, workspace_id,
                  device_id, actor_id, entity_type, entity_id, operation,
                  payload, payload_hash, idempotency_key, occurred_at, recorded_at
           FROM sync_mutations
           WHERE workspace_id = ?1 AND sequence > ?2
           ORDER BY sequence ASC
           LIMIT ?3
           """,
           [workspace_id, sequence, limit]
         ) do
      {:ok, rows} -> {:ok, Enum.map(rows, &mutation_from_row/1)}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec advance_cursor(String.t(), String.t(), non_neg_integer()) :: :ok | {:error, term()}
  def advance_cursor(workspace_id, replica_id, sequence)
      when is_binary(workspace_id) and is_binary(replica_id) and is_integer(sequence) and
             sequence >= 0 do
    with {:ok, tenant_id, _organization_id} <- workspace_scope(workspace_id) do
      Store.raw_execute(
        """
        INSERT INTO sync_cursors (tenant_id, workspace_id, replica_id, last_sequence)
        VALUES (?1, ?2, ?3, ?4)
        ON CONFLICT(tenant_id, workspace_id, replica_id) DO UPDATE SET
          last_sequence = MAX(sync_cursors.last_sequence, excluded.last_sequence),
          updated_at = datetime('now')
        """,
        [tenant_id, workspace_id, replica_id, sequence]
      )
    end
  end

  defp get_mutation_by_idempotency_key(key) do
    case Store.raw_query(
           """
           SELECT sequence, id, tenant_id, organization_id, workspace_id,
                  device_id, actor_id, entity_type, entity_id, operation,
                  payload, payload_hash, idempotency_key, occurred_at, recorded_at
           FROM sync_mutations WHERE idempotency_key = ?1
           """,
           [key]
         ) do
      {:ok, [row]} -> {:ok, mutation_from_row(row)}
      {:ok, []} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  defp workspace_scope(workspace_id) do
    case Store.raw_query(
           "SELECT tenant_id, organization_id FROM workspaces WHERE id = ?1 AND status = 'active'",
           [workspace_id]
         ) do
      {:ok, [[tenant_id, organization_id]]} -> {:ok, tenant_id, organization_id}
      {:ok, []} -> {:error, :workspace_not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  defp validate_overrides(overrides) when is_map(overrides) do
    known = MapSet.new(Providers.provider_ids())
    unknown = overrides |> Map.keys() |> Enum.reject(&MapSet.member?(known, to_string(&1)))
    if unknown == [], do: :ok, else: {:error, {:unknown_providers, unknown}}
  end

  defp validate_overrides(_), do: {:error, :invalid_provider_overrides}

  defp require_fields(attrs, fields) do
    missing = Enum.reject(fields, &(Map.has_key?(attrs, &1) and not is_nil(Map.get(attrs, &1))))
    if missing == [], do: :ok, else: {:error, {:missing_fields, missing}}
  end

  defp validate_operation(operation) when operation in [:create, :update, :delete], do: :ok
  defp validate_operation(operation) when operation in ["create", "update", "delete"], do: :ok
  defp validate_operation(_), do: {:error, :invalid_operation}

  defp policy_from_row([
         workspace_id,
         tenant_id,
         organization_id,
         use_cases,
         provider_overrides,
         policy_version,
         created_by,
         created_at,
         updated_at
       ]) do
    %{
      workspace_id: workspace_id,
      tenant_id: tenant_id,
      organization_id: organization_id,
      use_cases: decode_json(use_cases, []),
      provider_overrides: decode_json(provider_overrides, %{}),
      policy_version: policy_version,
      created_by: created_by,
      created_at: created_at,
      updated_at: updated_at
    }
  end

  defp mutation_from_row([
         sequence,
         id,
         tenant_id,
         organization_id,
         workspace_id,
         device_id,
         actor_id,
         entity_type,
         entity_id,
         operation,
         payload,
         payload_hash,
         idempotency_key,
         occurred_at,
         recorded_at
       ]) do
    %{
      sequence: sequence,
      id: id,
      tenant_id: tenant_id,
      organization_id: organization_id,
      workspace_id: workspace_id,
      device_id: device_id,
      actor_id: actor_id,
      entity_type: entity_type,
      entity_id: entity_id,
      operation: operation,
      payload: decode_json(payload, nil),
      payload_hash: payload_hash,
      idempotency_key: idempotency_key,
      occurred_at: occurred_at,
      recorded_at: recorded_at
    }
  end

  defp decode_json(nil, fallback), do: fallback

  defp decode_json(value, fallback) do
    case Jason.decode(value) do
      {:ok, decoded} -> decoded
      _ -> fallback
    end
  end

  defp hash(value), do: :crypto.hash(:sha256, value) |> Base.encode16(case: :lower)
end
