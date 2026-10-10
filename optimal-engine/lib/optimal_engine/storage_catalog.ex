defmodule OptimalEngine.StorageCatalog do
  @moduledoc """
  Read-only inventory of the engine's logical stores and indexes.

  This is the agent-facing map of where data lives. It exposes capabilities,
  backing technologies, and aggregate counts without exposing stored content,
  credentials, or raw database access.
  """

  alias OptimalEngine.Store

  @stores [
    %{
      id: "relational",
      purpose: "Canonical topology, identity, evidence, truth, and governance",
      technology: "SQLite",
      tables:
        ~w(organizations workspaces nodes source_packages contexts claims facts memory_objects episodes events workspace_storage_policies storage_provider_configs)
    },
    %{
      id: "full_text",
      purpose: "Exact text and phrase retrieval",
      technology: "SQLite FTS5",
      tables: ~w(contexts_fts memories_fts)
    },
    %{
      id: "vector",
      purpose: "Semantic similarity retrieval",
      technology: "SQLite BLOB embeddings with cosine search",
      tables: ~w(vectors chunk_embeddings)
    },
    %{
      id: "graph",
      purpose: "Relationship traversal and multi-hop reasoning",
      technology: "RocksDB projection with SQLite lineage",
      tables: ~w(edges relationship_edges node_relationships)
    },
    %{
      id: "assets",
      purpose: "Files, audio, video, OCR, transcripts, and visual extraction",
      technology: "Filesystem plus SQLite metadata",
      tables:
        ~w(assets asset_extractions asset_transcripts asset_ocr_spans asset_visual_observations)
    },
    %{
      id: "cache",
      purpose: "Hot context and retrieval acceleration",
      technology: "ETS and filesystem cache",
      tables: []
    },
    %{
      id: "jobs",
      purpose: "Durable background work, leases, retries, and dead letters",
      technology: "SQLite",
      tables: ~w(jobs dead_letter_jobs)
    },
    %{
      id: "replication",
      purpose: "Workspace-scoped mutation replay, idempotency, and replica progress",
      technology: "SQLite append-only ledger with optional NATS JetStream transport",
      tables: ~w(sync_mutations sync_cursors)
    },
    %{
      id: "metrics",
      purpose: "Live and historical operational measurements",
      technology: "Telemetry, Prometheus exposition, and SQLite history",
      tables: ~w(metric_samples)
    },
    %{
      id: "backups",
      purpose: "Backup inventory, verification, retention, and offsite status",
      technology: "SQLite catalog plus backup files",
      tables: ~w(backup_records)
    },
    %{
      id: "decomposition",
      purpose: "Checkpointed deterministic, recursive, and fallback decomposition runs",
      technology: "SQLite records plus optional DSPy RLM sidecar",
      tables: ~w(decomposition_runs)
    },
    %{
      id: "models",
      purpose: "Governed training examples, adaptation runs, and versioned model adapters",
      technology: "SQLite metadata plus trainer-neutral artifact storage",
      tables:
        ~w(training_examples training_runs model_adapters model_call_operations model_call_runs)
    },
    %{
      id: "secrets",
      purpose: "API authentication and encrypted connector credentials",
      technology: "bcrypt hashes and encrypted credential envelopes",
      tables: ~w(api_keys connectors)
    }
  ]

  @record_views %{
    "jobs" => {
      "SELECT id, workspace_id, kind, status, priority, run_at, attempts, max_attempts, last_error, updated_at FROM jobs WHERE workspace_id = ?1 ORDER BY updated_at DESC LIMIT ?2",
      ~w(id workspace_id kind status priority run_at attempts max_attempts last_error updated_at)
    },
    "metrics" => {
      "SELECT id, workspace_id, metric_name, metric_kind, value, labels, recorded_at FROM metric_samples WHERE workspace_id = ?1 ORDER BY recorded_at DESC LIMIT ?2",
      ~w(id workspace_id metric_name metric_kind value labels recorded_at)
    },
    "backups" => {
      "SELECT id, storage_provider, status, encrypted, offsite, size_bytes, checksum_sha256, integrity_status, created_at, verified_at, expires_at FROM backup_records ORDER BY created_at DESC LIMIT ?1",
      ~w(id storage_provider status encrypted offsite size_bytes checksum_sha256 integrity_status created_at verified_at expires_at)
    },
    "decomposition" => {
      "SELECT id, workspace_id, source_package_id, strategy, status, fallback_strategy, attempt, input_bytes, output_units, model_calls, iterations, error, created_at, completed_at FROM decomposition_runs WHERE workspace_id = ?1 ORDER BY created_at DESC LIMIT ?2",
      ~w(id workspace_id source_package_id strategy status fallback_strategy attempt input_bytes output_units model_calls iterations error created_at completed_at)
    },
    "replication" => {
      "SELECT sequence, id, workspace_id, device_id, actor_id, entity_type, entity_id, operation, payload_hash, idempotency_key, occurred_at, recorded_at FROM sync_mutations WHERE workspace_id = ?1 ORDER BY sequence DESC LIMIT ?2",
      ~w(sequence id workspace_id device_id actor_id entity_type entity_id operation payload_hash idempotency_key occurred_at recorded_at)
    },
    "models" => {
      "SELECT id, workspace_id, base_model, trainer, method, status, hardware, metrics, error, created_at, completed_at FROM training_runs WHERE workspace_id = ?1 ORDER BY created_at DESC LIMIT ?2",
      ~w(id workspace_id base_model trainer method status hardware metrics error created_at completed_at)
    }
  }

  @spec list() :: [map()]
  def list do
    Enum.map(@stores, fn store ->
      results = Map.new(store.tables, &{&1, count(&1)})

      counts =
        Map.new(results, fn
          {table, {:ok, count}} -> {table, count}
          {table, {:error, _reason}} -> {table, nil}
        end)

      store
      |> Map.put(:status, status(results))
      |> Map.put(:row_count, counts |> Map.values() |> Enum.reject(&is_nil/1) |> Enum.sum())
      |> Map.put(:table_counts, counts)
    end)
  end

  @spec get(String.t()) :: {:ok, map()} | {:error, :not_found}
  def get(id) when is_binary(id) do
    case Enum.find(list(), &(&1.id == id)) do
      nil -> {:error, :not_found}
      store -> {:ok, store}
    end
  end

  @spec records(String.t(), String.t() | nil, pos_integer()) ::
          {:ok, [map()]} | {:error, :not_found | :not_inspectable | :workspace_required | term()}
  def records(id, workspace_id, limit \\ 25) when is_binary(id) do
    limit = limit |> max(1) |> min(100)

    case Map.get(@record_views, id) do
      nil ->
        if Enum.any?(@stores, &(&1.id == id)),
          do: {:error, :not_inspectable},
          else: {:error, :not_found}

      {sql, fields} when id == "backups" ->
        query_records(sql, [limit], fields)

      {_sql, _fields} when not is_binary(workspace_id) or workspace_id == "" ->
        {:error, :workspace_required}

      {sql, fields} ->
        query_records(sql, [workspace_id, limit], fields)
    end
  end

  defp count(table) do
    case Store.raw_query("SELECT COUNT(*) FROM \"#{table}\"", []) do
      {:ok, [[count]]} when is_integer(count) -> {:ok, count}
      {:error, reason} -> {:error, reason}
      result -> {:error, {:unexpected_result, result}}
    end
  end

  defp query_records(sql, params, fields) do
    case Store.raw_query(sql, params) do
      {:ok, rows} -> {:ok, Enum.map(rows, &Map.new(Enum.zip(fields, &1)))}
      {:error, reason} -> {:error, reason}
    end
  end

  defp status(results) do
    if Enum.all?(results, fn {_table, result} -> match?({:ok, _count}, result) end),
      do: "available",
      else: "degraded"
  end
end
