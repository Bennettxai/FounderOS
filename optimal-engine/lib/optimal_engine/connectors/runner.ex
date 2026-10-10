defmodule OptimalEngine.Connectors.Runner do
  @moduledoc """
  Executes a connector sync with state bookkeeping + structured
  error handling.

  Invariants:

    * One row in `connector_runs` per invocation, no matter how it ends
    * The `connectors.cursor` column is advanced **only** on success
    * `{:error, :fatal}` flips `connectors.enabled = 0` so it stops paging
    * Rate-limit errors sleep + retry up to `:max_retries` times
    * Transient errors back off exponentially (100ms → 200ms → … capped)

  This module doesn't know about specific adapters — it receives the
  module + row and drives the contract. The adapter's job is to map
  external APIs to `{:ok, signals, cursor}` / `{:error, reason}`.

  The `[Signal.t()]` returned by `sync/2` is handed straight to the
  intake pipeline (Phase 2 → 3 → 4 → …) via
  `OptimalEngine.Pipeline.Intake.ingest_signals/1`.
  """

  alias OptimalEngine.Connectors.{AssetIngest, Credential, Registry}
  alias OptimalEngine.MemoryCore.ToolModelGovernance
  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  require Logger

  @type connector_row :: %{
          id: String.t(),
          tenant_id: String.t(),
          workspace_id: String.t(),
          kind: String.t(),
          config: map(),
          cursor: String.t() | nil,
          enabled: boolean()
        }

  @type run_result :: %{
          connector_id: String.t(),
          workspace_id: String.t(),
          status: :success | :error | :disabled,
          signals: non_neg_integer(),
          errors: non_neg_integer(),
          cursor_before: String.t() | nil,
          cursor_after: String.t() | nil,
          reason: term() | nil
        }

  @type governed_run_result :: %{
          connector_id: String.t(),
          governance_run: map(),
          connector_result: map()
        }

  @default_max_retries 5
  @base_backoff_ms 100
  @max_backoff_ms 30_000

  @doc """
  Run one sync cycle for the connector identified by `connector_id`.

  Options:
    * `:max_retries` — cap on transient-error retries (default 5)
    * `:signal_sink` — callback that consumes the produced signals
      (default: no-op). Arity 1 receives `signals`; arity 2 receives
      `signals` plus a scope map `%{workspace_id, tenant_id, connector_id}`
      so the sink can land connector-ingested signals in the connector's
      workspace (e.g. `Pipeline.Intake.process(text, workspace_id: ...)`).
    * `:adapter` — adapter module override (test seam — production lookup
      goes through the compile-time `Registry`)
  """
  @spec run(String.t(), keyword()) :: {:ok, run_result()} | {:error, term()}
  def run(connector_id, opts \\ []) when is_binary(connector_id) do
    if governed_run?(opts) do
      run_governed(connector_id, Keyword.put(opts, :governed, true))
    else
      run_raw(connector_id, opts)
    end
  end

  defp run_raw(connector_id, opts) do
    with {:ok, row} <- fetch_connector(connector_id),
         :ok <- ensure_enabled(row),
         {:ok, mod} <- lookup_adapter(row.kind, opts),
         {:ok, state} <- init_adapter(mod, row.config) do
      run_id = start_run_row(row)

      result =
        row
        |> do_sync_with_retries(mod, state, opts)
        |> finalize(row, run_id, opts)

      {:ok, result}
    end
  end

  @doc """
  Run one sync cycle through the Memory Core governed tool-call surface.
  """
  @spec run_governed(String.t(), keyword()) ::
          {:ok, governed_run_result()} | {:error, term()}
  def run_governed(connector_id, opts \\ []) when is_binary(connector_id) do
    with {:ok, row} <- fetch_connector(connector_id),
         {:ok, mod} <- lookup_adapter(row.kind, opts),
         {:ok, _definition} <- register_governed_connector_tool(row, mod, opts) do
      executor = fn payload ->
        case run_raw(connector_id, connector_run_opts(opts)) do
          {:ok, result} -> {:ok, governed_connector_output(row, result, payload)}
          {:error, reason} -> {:error, reason}
        end
      end

      case ToolModelGovernance.execute_tool_call(
             connector_tool_name(row),
             governed_connector_input(row),
             executor,
             governed_connector_opts(row, opts)
           ) do
        {:ok, governance_run} ->
          {:ok,
           %{
             connector_id: row.id,
             governance_run: governance_run,
             connector_result: governance_run.output_payload
           }}

        {:error, _reason} = error ->
          error
      end
    end
  end

  # ─── sync loop with retry ───────────────────────────────────────────────

  defp do_sync_with_retries(row, mod, state, opts) do
    max_retries = Keyword.get(opts, :max_retries, @default_max_retries)
    do_sync_attempt(row, mod, state, 0, max_retries)
  end

  defp do_sync_attempt(row, mod, state, attempt, max_retries) do
    case mod.sync(state, row.cursor) do
      {:ok, signals, next_cursor} when is_list(signals) ->
        {:ok, signals, next_cursor, []}

      {:ok, %{signals: signals, cursor: next_cursor} = result} when is_list(signals) ->
        {:ok, signals, next_cursor, Map.get(result, :payloads, [])}

      {:error, {:rate_limited, retry_after_ms}} when attempt < max_retries ->
        Logger.info("Connector #{row.id}: rate-limited, sleeping #{retry_after_ms}ms")
        Process.sleep(retry_after_ms)
        do_sync_attempt(row, mod, state, attempt + 1, max_retries)

      {:error, :rate_limited} when attempt < max_retries ->
        Process.sleep(backoff(attempt))
        do_sync_attempt(row, mod, state, attempt + 1, max_retries)

      {:error, :auth_expired} ->
        {:error, :auth_expired}

      {:error, :transient} when attempt < max_retries ->
        Process.sleep(backoff(attempt))
        do_sync_attempt(row, mod, state, attempt + 1, max_retries)

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp backoff(attempt) do
    min(@base_backoff_ms * :math.pow(2, attempt), @max_backoff_ms)
    |> trunc()
  end

  # ─── finalize + persist state ────────────────────────────────────────────

  defp finalize({:ok, signals, next_cursor, payloads}, row, run_id, opts) do
    sink = Keyword.get(opts, :signal_sink, &default_sink/1)
    :ok = invoke_sink(sink, signals, row)
    %{assets: assets, errors: asset_errors} = preserve_payload_assets(row, payloads, opts)

    # Persist the cursor advance + audit completion in one transaction so
    # they succeed or fail together. Without this, a cursor could advance
    # while the audit row stayed in `'running'` — a silently-lost run that
    # no operator could diagnose.
    case transaction(fn ->
           advance_cursor(row.id, next_cursor)

           complete_run_row(
             run_id,
             :success,
             length(signals),
             length(asset_errors),
             row.cursor,
             next_cursor,
             asset_error_detail(asset_errors)
           )
         end) do
      :ok ->
        %{
          connector_id: row.id,
          workspace_id: row.workspace_id,
          status: :success,
          signals: length(signals),
          errors: length(asset_errors),
          assets: length(assets),
          asset_errors: length(asset_errors),
          cursor_before: row.cursor,
          cursor_after: next_cursor,
          reason: nil
        }

      :error ->
        # The sync itself worked, but we couldn't persist the state
        # change. Surface it so the operator sees a consistent failure
        # rather than a success report.
        complete_run_row(
          run_id,
          :error,
          length(signals),
          1,
          row.cursor,
          row.cursor,
          "persist_failed"
        )

        %{
          connector_id: row.id,
          workspace_id: row.workspace_id,
          status: :error,
          signals: length(signals),
          errors: 1,
          assets: length(assets),
          asset_errors: length(asset_errors),
          cursor_before: row.cursor,
          cursor_after: row.cursor,
          reason: :persist_failed
        }
    end
  end

  defp finalize({:error, :fatal}, row, run_id, _opts) do
    disable_connector(row.id)
    complete_run_row(run_id, :error, 0, 1, row.cursor, row.cursor, "fatal")

    %{
      connector_id: row.id,
      workspace_id: row.workspace_id,
      status: :disabled,
      signals: 0,
      errors: 1,
      assets: 0,
      asset_errors: 0,
      cursor_before: row.cursor,
      cursor_after: row.cursor,
      reason: :fatal
    }
  end

  defp finalize({:error, reason}, row, run_id, _opts) do
    complete_run_row(run_id, :error, 0, 1, row.cursor, row.cursor, inspect(reason))

    %{
      connector_id: row.id,
      workspace_id: row.workspace_id,
      status: :error,
      signals: 0,
      errors: 1,
      assets: 0,
      asset_errors: 0,
      cursor_before: row.cursor,
      cursor_after: row.cursor,
      reason: reason
    }
  end

  # ─── row I/O ─────────────────────────────────────────────────────────────

  defp fetch_connector(id) do
    case Store.raw_query(
           """
           SELECT id, tenant_id, workspace_id, kind, config, cursor, enabled
           FROM connectors WHERE id = ?1 LIMIT 1
           """,
           [id]
         ) do
      {:ok, [[id, tenant_id, workspace_id, kind, config_json, cursor, enabled]]} ->
        config = decode_config(config_json)

        {:ok,
         %{
           id: id,
           tenant_id: tenant_id,
           workspace_id: workspace_id || "default",
           kind: kind,
           config: config,
           cursor: cursor,
           enabled: enabled == 1
         }}

      {:ok, []} ->
        {:error, :not_found}

      other ->
        other
    end
  end

  defp ensure_enabled(%{enabled: true}), do: :ok
  defp ensure_enabled(%{enabled: false}), do: {:error, :disabled}

  # `String.to_existing_atom/1` (not `to_atom/1`): `kind_str` comes from the
  # `connectors.kind` column — operator-writable data. `to_atom/1` would
  # create a new atom on every junk value and eventually exhaust the atom
  # table. Every legitimate adapter atom is already loaded via the Registry.
  defp lookup_adapter(kind_str, opts) when is_binary(kind_str) do
    cond do
      mod = Keyword.get(opts, :adapter) ->
        {:ok, mod}

      resolver = Keyword.get(opts, :adapter_resolver) ->
        resolver.(kind_str)

      true ->
        Registry.fetch(String.to_existing_atom(kind_str))
    end
  rescue
    ArgumentError -> {:error, :unknown_kind}
  end

  defp init_adapter(mod, config) do
    config = maybe_decrypt_credentials(config)

    case mod.init(config) do
      {:ok, state} -> {:ok, state}
      {:error, _} = err -> err
      other -> {:error, {:bad_init_return, other}}
    end
  end

  defp register_governed_connector_tool(row, mod, opts) do
    ToolModelGovernance.register_mcp_tool_definition(
      tenant_id: row.tenant_id,
      workspace_id: governed_workspace_id(row, opts),
      tool_name: connector_tool_name(row),
      protocol_adapter_id: "mcp",
      implementation_type: "external_connector",
      registration_source: "connectors.runner",
      required_privileges:
        Keyword.get(opts, :required_privileges, [
          "connector:#{row.kind}:sync",
          "signal:ingest"
        ]),
      allowed_partitions: Keyword.get(opts, :allowed_partitions, []),
      input_schema: %{required: ["connector_id", "kind", "operation"]},
      output_schema: %{required: ["connector_id", "kind", "status", "signals", "errors"]},
      routing_policy: %{connector_id: row.id, kind: row.kind},
      timeout_policy: %{source: "connector_runner"},
      audit_policy: %{record_connector_run: true},
      metadata: %{
        connector_id: row.id,
        kind: row.kind,
        display_name: safe_adapter_value(mod, :display_name),
        auth_scheme: safe_adapter_value(mod, :auth_scheme)
      }
    )
  end

  defp governed_connector_opts(row, opts) do
    [
      tenant_id: row.tenant_id,
      workspace_id: governed_workspace_id(row, opts),
      protocol_adapter_id: "mcp",
      actor_id: Keyword.get(opts, :actor_id, "system:connector-runner"),
      active_memory_pool_id: Keyword.get(opts, :active_memory_pool_id),
      granted_privileges: Keyword.get(opts, :granted_privileges, []),
      requested_partitions: Keyword.get(opts, :requested_partitions, []),
      security_labels: Keyword.get(opts, :security_labels, []),
      partition_ids: Keyword.get(opts, :partition_ids, []),
      metadata: %{connector_id: row.id, kind: row.kind, operation: "sync"}
    ]
  end

  defp governed_connector_input(row) do
    %{
      connector_id: row.id,
      kind: row.kind,
      tenant_id: row.tenant_id,
      cursor_before: row.cursor,
      operation: "sync"
    }
  end

  defp governed_connector_output(row, result, _payload) do
    %{
      connector_id: row.id,
      kind: row.kind,
      status: result.status |> to_string(),
      signals: result.signals,
      errors: result.errors,
      assets: Map.get(result, :assets, 0),
      asset_errors: Map.get(result, :asset_errors, 0),
      cursor_before: result.cursor_before,
      cursor_after: result.cursor_after,
      reason: reason_to_string(result.reason)
    }
  end

  defp connector_tool_name(row), do: "connector.#{row.kind}.sync"

  defp governed_run?(opts) do
    Keyword.get(opts, :governed, false) == true or
      Keyword.has_key?(opts, :granted_privileges) or
      Keyword.has_key?(opts, :requested_partitions) or
      Keyword.has_key?(opts, :allowed_partitions)
  end

  defp connector_run_opts(opts) do
    Keyword.take(opts, [
      :max_retries,
      :signal_sink,
      :adapter,
      :adapter_resolver,
      :workspace_id,
      :actor_id,
      :security_labels,
      :partition_ids,
      :access_policy_id,
      :retention_class,
      :trust_label
    ])
  end

  defp preserve_payload_assets(_row, [], _opts), do: %{assets: [], errors: []}

  defp preserve_payload_assets(row, payloads, opts) when is_list(payloads) do
    Enum.reduce(payloads, %{assets: [], errors: []}, fn payload, acc ->
      external_id = payload_external_id(payload)

      case AssetIngest.preserve_payload_assets(row.kind, external_id, payload,
             tenant_id: row.tenant_id,
             workspace_id: Keyword.get(opts, :workspace_id, row.workspace_id),
             connector_id: row.id,
             actor_id: Keyword.get(opts, :actor_id, "system:connector-runner"),
             security_labels: Keyword.get(opts, :security_labels, []),
             partition_ids: Keyword.get(opts, :partition_ids, []),
             access_policy_id: Keyword.get(opts, :access_policy_id),
             retention_class: Keyword.get(opts, :retention_class),
             trust_label: Keyword.get(opts, :trust_label, "connector_unreviewed"),
             metadata: %{connector_run_payload_id: external_id}
           ) do
        {:ok, %{assets: assets, errors: errors}} ->
          %{assets: acc.assets ++ assets, errors: acc.errors ++ errors}
      end
    end)
  end

  defp preserve_payload_assets(_row, _payloads, _opts), do: %{assets: [], errors: []}

  defp payload_external_id(payload) when is_map(payload) do
    Map.get(payload, :id) ||
      Map.get(payload, "id") ||
      Map.get(payload, :external_id) ||
      Map.get(payload, "external_id") ||
      "payload-#{:erlang.unique_integer([:positive])}"
  end

  defp asset_error_detail([]), do: nil
  defp asset_error_detail(errors), do: Jason.encode!(%{asset_errors: errors})

  defp governed_workspace_id(row, opts) do
    Keyword.get(opts, :workspace_id, row.workspace_id || "default") |> to_string()
  end

  defp reason_to_string(nil), do: nil
  defp reason_to_string(reason) when is_binary(reason), do: reason
  defp reason_to_string(reason) when is_atom(reason), do: Atom.to_string(reason)
  defp reason_to_string(reason), do: inspect(reason)

  defp safe_adapter_value(mod, callback) do
    if function_exported?(mod, callback, 0), do: apply(mod, callback, []), else: nil
  rescue
    _ -> nil
  end

  defp maybe_decrypt_credentials(%{"credentials_ciphertext" => envelope} = config) do
    case Credential.decrypt(envelope) do
      {:ok, creds} -> Map.merge(config, creds)
      _ -> config
    end
  end

  defp maybe_decrypt_credentials(config), do: config

  defp decode_config(nil), do: %{}
  defp decode_config(""), do: %{}

  defp decode_config(json) when is_binary(json) do
    case Jason.decode(json) do
      {:ok, map} when is_map(map) -> map
      _ -> %{}
    end
  end

  defp start_run_row(row) do
    {:ok, _} =
      Store.raw_query(
        """
        INSERT INTO connector_runs (tenant_id, workspace_id, connector_id, cursor_before, status)
        VALUES (?1, ?2, ?3, ?4, 'running')
        """,
        [row.tenant_id, row.workspace_id, row.id, row.cursor]
      )

    # SQLite returns the last insert id via a small follow-up query
    {:ok, [[id]]} = Store.raw_query("SELECT last_insert_rowid()", [])
    id
  end

  defp complete_run_row(run_id, status, signals, errors, cur_before, cur_after, err_detail) do
    status_str =
      case status do
        :success -> "success"
        :error -> "error"
        :disabled -> "disabled"
      end

    Store.raw_query(
      """
      UPDATE connector_runs
      SET completed_at = datetime('now'),
          status       = ?1,
          signals_ingested = ?2,
          errors_encountered = ?3,
          cursor_before = ?4,
          cursor_after = ?5,
          error_detail = ?6
      WHERE id = ?7
      """,
      [status_str, signals, errors, cur_before, cur_after, err_detail, run_id]
    )
  end

  defp advance_cursor(connector_id, new_cursor) do
    Store.raw_query(
      "UPDATE connectors SET cursor = ?1 WHERE id = ?2",
      [new_cursor, connector_id]
    )
  end

  defp disable_connector(connector_id) do
    Store.raw_query(
      "UPDATE connectors SET enabled = 0 WHERE id = ?1",
      [connector_id]
    )
  end

  # Default sink: the connector runner is called from admin tooling
  # (CLI, schedulers) which doesn't care about ingest — callers that
  # want downstream processing pass `:signal_sink` explicitly.
  defp default_sink(_signals), do: :ok

  # Sinks come in two arities: legacy arity-1 `(signals)`, and arity-2
  # `(signals, scope)` where scope carries the connector's workspace so
  # downstream ingest attributes the signals to the right workspace.
  defp invoke_sink(sink, signals, row) when is_function(sink, 2) do
    sink.(signals, %{
      workspace_id: row.workspace_id,
      tenant_id: row.tenant_id,
      connector_id: row.id
    })
  end

  defp invoke_sink(sink, signals, _row) when is_function(sink, 1) do
    sink.(signals)
  end

  # Wrap a sequence of raw_query writes in a SQLite BEGIN/COMMIT so a
  # failure in any statement rolls back the lot. If the transaction
  # itself errors we log + return :error rather than raising, so the
  # caller can surface the issue through its normal result channel.
  defp transaction(fun) when is_function(fun, 0) do
    Store.raw_query("BEGIN IMMEDIATE", [])
    fun.()
    Store.raw_query("COMMIT", [])
    :ok
  rescue
    e ->
      Store.raw_query("ROLLBACK", [])
      Logger.error("[Runner] transaction aborted: #{Exception.message(e)}")
      :error
  end

  @doc """
  Upsert a connector row. This is the only place config JSON is
  written so we can enforce shape + encrypt credentials centrally.

  `attrs` may include `:workspace_id` (default `"default"`) — every signal
  the connector ingests is attributed to that workspace.
  """
  @spec upsert_row(map()) :: {:ok, String.t()} | {:error, term()}
  def upsert_row(%{id: id, kind: kind, config: config} = attrs)
      when is_binary(id) and is_atom(kind) and is_map(config) do
    tenant_id = Map.get(attrs, :tenant_id, Tenant.default_id())
    workspace_id = Map.get(attrs, :workspace_id, "default")
    enabled = if Map.get(attrs, :enabled, true), do: 1, else: 0

    sanitized = maybe_encrypt_credentials(config)

    sql = """
    INSERT INTO connectors (id, tenant_id, workspace_id, kind, config, cursor, enabled)
    VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
    ON CONFLICT(id) DO UPDATE SET
      workspace_id = excluded.workspace_id,
      config  = excluded.config,
      enabled = excluded.enabled
    """

    case Store.raw_query(sql, [
           id,
           tenant_id,
           workspace_id,
           Atom.to_string(kind),
           Jason.encode!(sanitized),
           Map.get(attrs, :cursor),
           enabled
         ]) do
      {:ok, _} -> {:ok, id}
      other -> other
    end
  end

  defp maybe_encrypt_credentials(%{"credentials" => creds} = config) when is_map(creds) do
    case Credential.encrypt(creds) do
      {:ok, envelope} ->
        config
        |> Map.delete("credentials")
        |> Map.put("credentials_ciphertext", envelope)

      {:error, _} ->
        # Key missing: keep plaintext for dev/test, but log loudly so
        # ops can't miss it in production.
        Logger.warning("Connector credentials stored in plaintext — CONNECTOR_KEY not configured.")

        config
    end
  end

  defp maybe_encrypt_credentials(config), do: config
end
