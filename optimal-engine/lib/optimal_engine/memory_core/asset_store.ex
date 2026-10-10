defmodule OptimalEngine.MemoryCore.AssetStore do
  @moduledoc """
  Governed storage for raw multimodal artifacts.

  Parsers can extract text, OCR, transcripts, frames, or summaries, but the raw
  file is still the evidence. This module preserves that file in the workspace
  asset store, writes an `assets` row, creates a Source Package for the raw
  artifact, and records the derivation link.
  """

  alias OptimalEngine.{Store, Workspace}

  alias OptimalEngine.MemoryCore.{
    DerivationLedgerEntry,
    ClaimExtractor,
    ID,
    JSON,
    SourcePackage
  }

  alias OptimalEngine.Pipeline.MultimodalToolRegistry
  alias OptimalEngine.Pipeline.Parser.Asset
  alias OptimalEngine.Workspace.Filesystem

  @spec store_file(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def store_file(path, opts \\ []) when is_binary(path) do
    with {:ok, asset} <- Asset.from_path(path, opts),
         {:ok, result} <- store_asset(asset, opts) do
      {:ok, result}
    end
  end

  @spec store_asset(Asset.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def store_asset(asset, opts \\ [])

  def store_asset(%Asset{path: path} = asset, opts) when is_binary(path) do
    with {:ok, storage_path} <- copy_asset(asset, opts),
         {:ok, source_package} <- record_source_package(asset, storage_path, opts),
         asset_id = asset_id(asset, source_package),
         :ok <- upsert_asset(asset_id, asset, storage_path, source_package, opts),
         :ok <- record_derivation(asset_id, asset, source_package, opts),
         {:ok, stored_asset} <- get(asset_id, opts) do
      {:ok, %{asset: stored_asset, source_package: source_package}}
    end
  end

  def store_asset(%Asset{} = _asset, _opts), do: {:error, :asset_path_required}

  @spec record_adapter_run(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def record_adapter_run(asset_id, opts \\ []) when is_binary(asset_id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    with {:ok, asset} <- get(asset_id, tenant_id: tenant_id, workspace_id: workspace_id),
         {:ok, tool} <- adapter_tool(opts),
         {:ok, run} <- build_adapter_run(asset, tool, opts),
         :ok <- insert_adapter_run(run),
         :ok <- record_adapter_derivation(run, asset, tool, opts) do
      {:ok, run}
    end
  end

  @spec list_adapter_runs(String.t(), keyword()) :: {:ok, [map()]} | {:error, term()}
  def list_adapter_runs(asset_id, opts \\ []) when is_binary(asset_id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, tenant_id, workspace_id, asset_id, source_package_id, adapter_id,
           adapter_role, modality, status, started_at, completed_at, input_hash,
           output_hash, output_text, output_ref, model_id, model_version,
           confidence, precision, error_reason, security_labels, partition_ids,
           metadata, derivation_ledger_id, created_by, created_at
    FROM asset_adapter_runs
    WHERE tenant_id = ?1 AND workspace_id = ?2 AND asset_id = ?3
    ORDER BY created_at DESC
    """

    case Store.raw_query(sql, [tenant_id, workspace_id, asset_id]) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_adapter_run/1)}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec get_adapter_run(String.t(), keyword()) :: {:ok, map()} | {:error, :not_found | term()}
  def get_adapter_run(run_id, opts \\ []) when is_binary(run_id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, tenant_id, workspace_id, asset_id, source_package_id, adapter_id,
           adapter_role, modality, status, started_at, completed_at, input_hash,
           output_hash, output_text, output_ref, model_id, model_version,
           confidence, precision, error_reason, security_labels, partition_ids,
           metadata, derivation_ledger_id, created_by, created_at
    FROM asset_adapter_runs
    WHERE tenant_id = ?1 AND workspace_id = ?2 AND id = ?3
    """

    case Store.raw_query(sql, [tenant_id, workspace_id, run_id]) do
      {:ok, [row]} -> {:ok, row_to_adapter_run(row)}
      {:ok, []} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec claim_from_adapter_run(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def claim_from_adapter_run(run_id, opts \\ []) when is_binary(run_id) do
    with {:ok, run} <- get_adapter_run(run_id, opts),
         :ok <- claimable_adapter_run?(run) do
      source_package =
        SourcePackage.from_text(run.output_text,
          tenant_id: run.tenant_id,
          workspace_id: run.workspace_id,
          source_type: "adapter_output",
          source_class: run.modality || "text",
          source_system: run.adapter_id,
          source_uri: "optimal://assets/#{run.asset_id}/adapter-runs/#{run.id}",
          trust_label: Keyword.get(opts, :trust_label, "derived_unreviewed"),
          access_policy_id: Keyword.get(opts, :access_policy_id),
          security_labels: run.security_labels,
          partition_ids: run.partition_ids,
          actor_id: Keyword.get(opts, :actor_id) || run.created_by,
          metadata:
            Map.merge(run.metadata || %{}, %{
              asset_id: run.asset_id,
              raw_source_package_id: run.source_package_id,
              adapter_run_id: run.id,
              adapter_id: run.adapter_id,
              adapter_role: run.adapter_role,
              output_hash: run.output_hash,
              derived_from: "asset_adapter_run"
            })
        )

      with {:ok, claim} <-
             ClaimExtractor.extract_from_source(source_package,
               claim_text: Keyword.get(opts, :claim_text, run.output_text),
               claim_type: Keyword.get(opts, :claim_type, "adapter_output"),
               subject_anchor: Keyword.get(opts, :subject_anchor),
               action_class: Keyword.get(opts, :action_class),
               object_anchor: Keyword.get(opts, :object_anchor),
               extraction_run_id: run.id,
               evaluator_id: Keyword.get(opts, :evaluator_id),
               aggregate_confidence:
                 Keyword.get(opts, :aggregate_confidence, run.confidence || 0.5),
               aggregate_precision: Keyword.get(opts, :aggregate_precision, run.precision || 0.5),
               actor_id: Keyword.get(opts, :actor_id) || run.created_by,
               metadata: %{
                 adapter_run_id: run.id,
                 asset_id: run.asset_id,
                 adapter_id: run.adapter_id,
                 output_hash: run.output_hash
               }
             ) do
        {:ok, %{adapter_run: run, source_package: source_package, pending_claim: claim}}
      end
    end
  end

  @spec record_asset_extraction(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def record_asset_extraction(run_id, opts \\ []) when is_binary(run_id) do
    with {:ok, run} <- get_adapter_run(run_id, opts),
         :ok <- completed_adapter_run?(run),
         {:ok, extraction} <- build_asset_extraction(run, opts),
         :ok <- insert_asset_extraction(extraction),
         {:ok, projection} <- insert_typed_extraction_projection(extraction, opts),
         :ok <- record_extraction_derivation(extraction, run, opts) do
      {:ok, %{extraction: extraction, projection: projection}}
    end
  end

  @spec list_asset_extractions(String.t(), keyword()) :: {:ok, [map()]} | {:error, term()}
  def list_asset_extractions(asset_id, opts \\ []) when is_binary(asset_id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
           extraction_type, modality, content_text, content_ref, content_hash,
           confidence, precision, security_labels, partition_ids, metadata,
           derivation_ledger_id, created_by, created_at
    FROM asset_extractions
    WHERE tenant_id = ?1 AND workspace_id = ?2 AND asset_id = ?3
    ORDER BY created_at DESC
    """

    case Store.raw_query(sql, [tenant_id, workspace_id, asset_id]) do
      {:ok, rows} -> {:ok, Enum.map(rows, &row_to_asset_extraction/1)}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec get_asset_extraction(String.t(), keyword()) :: {:ok, map()} | {:error, :not_found | term()}
  def get_asset_extraction(extraction_id, opts \\ []) when is_binary(extraction_id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
           extraction_type, modality, content_text, content_ref, content_hash,
           confidence, precision, security_labels, partition_ids, metadata,
           derivation_ledger_id, created_by, created_at
    FROM asset_extractions
    WHERE tenant_id = ?1 AND workspace_id = ?2 AND id = ?3
    """

    case Store.raw_query(sql, [tenant_id, workspace_id, extraction_id]) do
      {:ok, [row]} -> {:ok, row_to_asset_extraction(row)}
      {:ok, []} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec claim_from_asset_extraction(String.t(), keyword()) :: {:ok, map()} | {:error, term()}
  def claim_from_asset_extraction(extraction_id, opts \\ []) when is_binary(extraction_id) do
    with {:ok, extraction} <- get_asset_extraction(extraction_id, opts),
         :ok <- claimable_extraction?(extraction) do
      source_package =
        SourcePackage.from_text(extraction.content_text,
          tenant_id: extraction.tenant_id,
          workspace_id: extraction.workspace_id,
          source_type: "asset_extraction",
          source_class: extraction.extraction_type,
          source_system: "asset_extraction_projection",
          source_uri: "optimal://assets/#{extraction.asset_id}/extractions/#{extraction.id}",
          trust_label: Keyword.get(opts, :trust_label, "derived_unreviewed"),
          security_labels: extraction.security_labels,
          partition_ids: extraction.partition_ids,
          actor_id: Keyword.get(opts, :actor_id) || extraction.created_by,
          metadata:
            Map.merge(extraction.metadata || %{}, %{
              asset_id: extraction.asset_id,
              adapter_run_id: extraction.adapter_run_id,
              extraction_id: extraction.id,
              extraction_type: extraction.extraction_type,
              content_hash: extraction.content_hash,
              derived_from: "asset_extraction"
            })
        )

      with {:ok, claim} <-
             ClaimExtractor.extract_from_source(source_package,
               claim_text: Keyword.get(opts, :claim_text, extraction.content_text),
               claim_type: Keyword.get(opts, :claim_type, extraction.extraction_type),
               subject_anchor: Keyword.get(opts, :subject_anchor),
               action_class: Keyword.get(opts, :action_class),
               object_anchor: Keyword.get(opts, :object_anchor),
               extraction_run_id: extraction.id,
               evaluator_id: Keyword.get(opts, :evaluator_id),
               aggregate_confidence:
                 Keyword.get(opts, :aggregate_confidence, extraction.confidence || 0.5),
               aggregate_precision:
                 Keyword.get(opts, :aggregate_precision, extraction.precision || 0.5),
               actor_id: Keyword.get(opts, :actor_id) || extraction.created_by,
               metadata: %{
                 adapter_run_id: extraction.adapter_run_id,
                 asset_id: extraction.asset_id,
                 extraction_id: extraction.id,
                 extraction_type: extraction.extraction_type,
                 content_hash: extraction.content_hash
               }
             ) do
        {:ok, %{asset_extraction: extraction, source_package: source_package, pending_claim: claim}}
      end
    end
  end

  @spec get(String.t(), keyword()) :: {:ok, map()} | {:error, :not_found | term()}
  def get(asset_id, opts \\ []) when is_binary(asset_id) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    sql = """
    SELECT id, tenant_id, workspace_id, content_type, size_bytes, storage_path,
           created_at, content_hash, modality, source_package_id, original_path,
           trust_label, retention_class, access_policy_id, security_labels,
           partition_ids, metadata
    FROM assets
    WHERE id = ?1 AND tenant_id = ?2 AND workspace_id = ?3
    """

    case Store.raw_query(sql, [asset_id, tenant_id, workspace_id]) do
      {:ok, [row]} -> {:ok, row_to_asset(row)}
      {:ok, []} -> {:error, :not_found}
      {:error, reason} -> {:error, reason}
    end
  end

  defp adapter_tool(opts) do
    adapter_id = Keyword.fetch!(opts, :adapter_id)
    normalized = normalize_adapter_id(adapter_id)

    case MultimodalToolRegistry.get(normalized) do
      nil -> {:error, {:unknown_adapter, adapter_id}}
      tool -> {:ok, tool}
    end
  end

  defp claimable_adapter_run?(%{status: "completed", output_text: output_text})
       when is_binary(output_text) do
    if String.trim(output_text) == "" do
      {:error, :adapter_output_empty}
    else
      :ok
    end
  end

  defp claimable_adapter_run?(%{status: status}), do: {:error, {:adapter_run_not_completed, status}}

  defp completed_adapter_run?(%{status: "completed"}), do: :ok
  defp completed_adapter_run?(%{status: status}), do: {:error, {:adapter_run_not_completed, status}}

  defp claimable_extraction?(%{content_text: content_text}) when is_binary(content_text) do
    if String.trim(content_text) == "" do
      {:error, :asset_extraction_empty}
    else
      :ok
    end
  end

  defp build_asset_extraction(run, opts) do
    now = timestamp()
    extraction_type = normalize_extraction_type(Keyword.fetch!(opts, :extraction_type))
    content_text = Keyword.get(opts, :content_text, run.output_text || "")
    content_ref = Keyword.get(opts, :content_ref)

    content_hash =
      Keyword.get(opts, :content_hash) || derived_output_hash(content_text, content_ref)

    extraction = %{
      id: Keyword.get(opts, :id, ID.random_id("aext")),
      tenant_id: run.tenant_id,
      workspace_id: run.workspace_id,
      asset_id: run.asset_id,
      source_package_id: run.source_package_id,
      adapter_run_id: run.id,
      extraction_type: extraction_type,
      modality: Keyword.get(opts, :modality, run.modality),
      content_text: content_text,
      content_ref: content_ref,
      content_hash: content_hash,
      confidence: Keyword.get(opts, :confidence, run.confidence),
      precision: Keyword.get(opts, :precision, run.precision),
      security_labels: Keyword.get(opts, :security_labels, run.security_labels),
      partition_ids: Keyword.get(opts, :partition_ids, run.partition_ids),
      metadata: Keyword.get(opts, :metadata, %{}),
      derivation_ledger_id: nil,
      created_by: Keyword.get(opts, :actor_id) || run.created_by,
      created_at: now
    }

    with :ok <- supported_extraction_type?(extraction_type),
         :ok <- validate_extraction_payload(extraction, opts) do
      {:ok, extraction}
    end
  end

  defp normalize_extraction_type(value) do
    value
    |> to_string()
    |> String.downcase()
    |> String.replace("-", "_")
  end

  defp supported_extraction_type?(type)
       when type in ["transcript", "ocr_span", "visual_observation", "embedding_ref"],
       do: :ok

  defp supported_extraction_type?(type), do: {:error, {:unsupported_extraction_type, type}}

  defp validate_extraction_payload(
         %{extraction_type: "embedding_ref", content_ref: content_ref},
         opts
       ) do
    embedding_ref = Keyword.get(opts, :embedding_ref) || content_ref

    cond do
      not is_binary(Keyword.get(opts, :embedding_model_id)) ->
        {:error, :embedding_model_id_required}

      not is_binary(embedding_ref) or String.trim(embedding_ref) == "" ->
        {:error, :embedding_ref_required}

      true ->
        :ok
    end
  end

  defp validate_extraction_payload(%{content_text: content_text}, _opts)
       when is_binary(content_text) do
    if String.trim(content_text) == "" do
      {:error, :asset_extraction_text_required}
    else
      :ok
    end
  end

  defp insert_asset_extraction(extraction) do
    sql = """
    INSERT INTO asset_extractions (
      id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
      extraction_type, modality, content_text, content_ref, content_hash,
      confidence, precision, security_labels, partition_ids, metadata,
      derivation_ledger_id, created_by, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11,
      ?12, ?13, ?14, ?15, ?16,
      ?17, ?18, ?19
    )
    """

    Store.raw_execute(sql, [
      extraction.id,
      extraction.tenant_id,
      extraction.workspace_id,
      extraction.asset_id,
      extraction.source_package_id,
      extraction.adapter_run_id,
      extraction.extraction_type,
      extraction.modality,
      extraction.content_text,
      extraction.content_ref,
      extraction.content_hash,
      extraction.confidence,
      extraction.precision,
      JSON.list(extraction.security_labels),
      JSON.list(extraction.partition_ids),
      JSON.map(extraction.metadata),
      extraction.derivation_ledger_id,
      extraction.created_by,
      extraction.created_at
    ])
  end

  defp insert_typed_extraction_projection(%{extraction_type: "transcript"} = extraction, opts) do
    projection =
      Map.merge(extraction, %{
        language: Keyword.get(opts, :language),
        speaker: Keyword.get(opts, :speaker),
        start_ms: Keyword.get(opts, :start_ms),
        end_ms: Keyword.get(opts, :end_ms)
      })

    sql = """
    INSERT INTO asset_transcripts (
      id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
      extraction_id, language, speaker, transcript_text, start_ms, end_ms,
      confidence, precision, security_labels, partition_ids, metadata,
      derivation_ledger_id, created_by, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11, ?12,
      ?13, ?14, ?15, ?16, ?17,
      ?18, ?19, ?20
    )
    """

    with :ok <-
           Store.raw_execute(sql, [
             projection.id,
             projection.tenant_id,
             projection.workspace_id,
             projection.asset_id,
             projection.source_package_id,
             projection.adapter_run_id,
             projection.id,
             projection.language,
             projection.speaker,
             projection.content_text,
             projection.start_ms,
             projection.end_ms,
             projection.confidence,
             projection.precision,
             JSON.list(projection.security_labels),
             JSON.list(projection.partition_ids),
             JSON.map(projection.metadata),
             projection.derivation_ledger_id,
             projection.created_by,
             projection.created_at
           ]) do
      {:ok, projection}
    end
  end

  defp insert_typed_extraction_projection(%{extraction_type: "ocr_span"} = extraction, opts) do
    projection =
      Map.merge(extraction, %{
        page_number: Keyword.get(opts, :page_number),
        bbox: Keyword.get(opts, :bbox, %{})
      })

    sql = """
    INSERT INTO asset_ocr_spans (
      id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
      extraction_id, page_number, span_text, bbox, confidence, precision,
      security_labels, partition_ids, metadata, derivation_ledger_id,
      created_by, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11, ?12,
      ?13, ?14, ?15, ?16,
      ?17, ?18
    )
    """

    with :ok <-
           Store.raw_execute(sql, [
             projection.id,
             projection.tenant_id,
             projection.workspace_id,
             projection.asset_id,
             projection.source_package_id,
             projection.adapter_run_id,
             projection.id,
             projection.page_number,
             projection.content_text,
             JSON.map(projection.bbox),
             projection.confidence,
             projection.precision,
             JSON.list(projection.security_labels),
             JSON.list(projection.partition_ids),
             JSON.map(projection.metadata),
             projection.derivation_ledger_id,
             projection.created_by,
             projection.created_at
           ]) do
      {:ok, projection}
    end
  end

  defp insert_typed_extraction_projection(
         %{extraction_type: "visual_observation"} = extraction,
         opts
       ) do
    projection =
      Map.merge(extraction, %{
        observation_type: Keyword.get(opts, :observation_type, "caption"),
        region: Keyword.get(opts, :region, %{}),
        frame_time_ms: Keyword.get(opts, :frame_time_ms)
      })

    sql = """
    INSERT INTO asset_visual_observations (
      id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
      extraction_id, observation_type, observation_text, region, frame_time_ms,
      confidence, precision, security_labels, partition_ids, metadata,
      derivation_ledger_id, created_by, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11,
      ?12, ?13, ?14, ?15, ?16,
      ?17, ?18, ?19
    )
    """

    with :ok <-
           Store.raw_execute(sql, [
             projection.id,
             projection.tenant_id,
             projection.workspace_id,
             projection.asset_id,
             projection.source_package_id,
             projection.adapter_run_id,
             projection.id,
             projection.observation_type,
             projection.content_text,
             JSON.map(projection.region),
             projection.frame_time_ms,
             projection.confidence,
             projection.precision,
             JSON.list(projection.security_labels),
             JSON.list(projection.partition_ids),
             JSON.map(projection.metadata),
             projection.derivation_ledger_id,
             projection.created_by,
             projection.created_at
           ]) do
      {:ok, projection}
    end
  end

  defp insert_typed_extraction_projection(%{extraction_type: "embedding_ref"} = extraction, opts) do
    projection =
      Map.merge(extraction, %{
        embedding_model_id: Keyword.fetch!(opts, :embedding_model_id),
        embedding_model_version: Keyword.get(opts, :embedding_model_version),
        embedding_ref: Keyword.get(opts, :embedding_ref) || extraction.content_ref,
        embedding_dim: Keyword.get(opts, :embedding_dim),
        embedding_space: Keyword.get(opts, :embedding_space),
        target_ref: Keyword.get(opts, :target_ref)
      })

    sql = """
    INSERT INTO asset_embedding_refs (
      id, tenant_id, workspace_id, asset_id, source_package_id, adapter_run_id,
      extraction_id, embedding_model_id, embedding_model_version, embedding_ref,
      embedding_dim, embedding_space, target_ref, confidence, precision,
      security_labels, partition_ids, metadata, derivation_ledger_id,
      created_by, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10,
      ?11, ?12, ?13, ?14, ?15,
      ?16, ?17, ?18, ?19,
      ?20, ?21
    )
    """

    with :ok <-
           Store.raw_execute(sql, [
             projection.id,
             projection.tenant_id,
             projection.workspace_id,
             projection.asset_id,
             projection.source_package_id,
             projection.adapter_run_id,
             projection.id,
             projection.embedding_model_id,
             projection.embedding_model_version,
             projection.embedding_ref,
             projection.embedding_dim,
             projection.embedding_space,
             projection.target_ref,
             projection.confidence,
             projection.precision,
             JSON.list(projection.security_labels),
             JSON.list(projection.partition_ids),
             JSON.map(projection.metadata),
             projection.derivation_ledger_id,
             projection.created_by,
             projection.created_at
           ]) do
      {:ok, projection}
    end
  end

  defp insert_typed_extraction_projection(%{extraction_type: extraction_type}, _opts) do
    {:error, {:unsupported_extraction_type, extraction_type}}
  end

  defp record_extraction_derivation(extraction, run, opts) do
    asset_ref = DerivationLedgerEntry.object_ref("asset", extraction.asset_id)
    run_ref = DerivationLedgerEntry.object_ref("asset_adapter_run", run.id)
    extraction_ref = DerivationLedgerEntry.object_ref("asset_extraction", extraction.id)

    source_refs =
      if is_binary(extraction.source_package_id) do
        [DerivationLedgerEntry.object_ref("source_package", extraction.source_package_id)]
      else
        []
      end

    ledger =
      DerivationLedgerEntry.new(
        "asset_extraction.#{extraction.extraction_type}",
        "adapter_run_to_asset_extraction",
        [run_ref],
        [extraction_ref],
        tenant_id: extraction.tenant_id,
        workspace_id: extraction.workspace_id,
        source_package_links: source_refs,
        evidence_links: [asset_ref, run_ref | source_refs],
        actor_id: Keyword.get(opts, :actor_id),
        parser_id: "optimal_engine.memory_core.asset_store",
        model_id: run.model_id,
        model_version: run.model_version,
        confidence_delta: extraction.confidence,
        precision_delta: extraction.precision,
        security_labels: extraction.security_labels,
        partition_ids: extraction.partition_ids,
        metadata: %{
          asset_id: extraction.asset_id,
          adapter_run_id: run.id,
          extraction_id: extraction.id,
          extraction_type: extraction.extraction_type,
          content_hash: extraction.content_hash
        }
      )

    case OptimalEngine.MemoryCore.Store.insert_derivation_entry(ledger) do
      :ok -> attach_extraction_derivation(extraction, ledger.id)
      {:error, reason} -> {:error, reason}
    end
  end

  defp attach_extraction_derivation(extraction, ledger_id) do
    type_table =
      case extraction.extraction_type do
        "transcript" -> "asset_transcripts"
        "ocr_span" -> "asset_ocr_spans"
        "visual_observation" -> "asset_visual_observations"
        "embedding_ref" -> "asset_embedding_refs"
      end

    with :ok <-
           Store.raw_execute(
             "UPDATE asset_extractions SET derivation_ledger_id = ?1 WHERE id = ?2",
             [ledger_id, extraction.id]
           ) do
      Store.raw_execute(
        "UPDATE #{type_table} SET derivation_ledger_id = ?1 WHERE id = ?2",
        [ledger_id, extraction.id]
      )
    end
  end

  defp normalize_adapter_id(id) when is_atom(id), do: id

  defp normalize_adapter_id(id) when is_binary(id) do
    case MultimodalToolRegistry.get(id) do
      %{id: tool_id} -> tool_id
      nil -> :unknown
    end
  end

  defp build_adapter_run(asset, tool, opts) do
    now = timestamp()
    output_text = Keyword.get(opts, :output_text, "")
    output_ref = Keyword.get(opts, :output_ref)
    status = Keyword.get(opts, :status, "completed")
    input_hash = Keyword.get(opts, :input_hash, asset.content_hash)
    output_hash = Keyword.get(opts, :output_hash) || derived_output_hash(output_text, output_ref)

    run = %{
      id: Keyword.get(opts, :id, ID.random_id("aar")),
      tenant_id: asset.tenant_id,
      workspace_id: asset.workspace_id,
      asset_id: asset.id,
      source_package_id: asset.source_package_id,
      adapter_id: Atom.to_string(tool.id),
      adapter_role: adapter_role(tool, opts),
      modality: Keyword.get(opts, :modality, asset.modality),
      status: status,
      started_at: Keyword.get(opts, :started_at, now),
      completed_at: Keyword.get(opts, :completed_at, if(status == "running", do: nil, else: now)),
      input_hash: input_hash,
      output_hash: output_hash,
      output_text: output_text,
      output_ref: output_ref,
      model_id: Keyword.get(opts, :model_id),
      model_version: Keyword.get(opts, :model_version),
      confidence: Keyword.get(opts, :confidence),
      precision: Keyword.get(opts, :precision),
      error_reason: Keyword.get(opts, :error_reason),
      security_labels: Keyword.get(opts, :security_labels, asset.security_labels),
      partition_ids: Keyword.get(opts, :partition_ids, asset.partition_ids),
      metadata: Keyword.get(opts, :metadata, %{}),
      derivation_ledger_id: nil,
      created_by: Keyword.get(opts, :actor_id),
      created_at: now
    }

    {:ok, run}
  end

  defp adapter_role(tool, opts) do
    case Keyword.get(opts, :adapter_role) do
      nil -> tool.roles |> List.first() |> to_string()
      role -> to_string(role)
    end
  end

  defp derived_output_hash("", nil), do: nil

  defp derived_output_hash(output_text, output_ref) do
    material = [output_text || "", ":", output_ref || ""]
    "sha256:" <> ID.sha256(IO.iodata_to_binary(material))
  end

  defp insert_adapter_run(run) do
    sql = """
    INSERT INTO asset_adapter_runs (
      id, tenant_id, workspace_id, asset_id, source_package_id, adapter_id,
      adapter_role, modality, status, started_at, completed_at, input_hash,
      output_hash, output_text, output_ref, model_id, model_version,
      confidence, precision, error_reason, security_labels, partition_ids,
      metadata, derivation_ledger_id, created_by, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11, ?12,
      ?13, ?14, ?15, ?16, ?17,
      ?18, ?19, ?20, ?21, ?22,
      ?23, ?24, ?25, ?26
    )
    """

    Store.raw_execute(sql, [
      run.id,
      run.tenant_id,
      run.workspace_id,
      run.asset_id,
      run.source_package_id,
      run.adapter_id,
      run.adapter_role,
      run.modality,
      run.status,
      run.started_at,
      run.completed_at,
      run.input_hash,
      run.output_hash,
      run.output_text,
      run.output_ref,
      run.model_id,
      run.model_version,
      run.confidence,
      run.precision,
      run.error_reason,
      JSON.list(run.security_labels),
      JSON.list(run.partition_ids),
      JSON.map(run.metadata),
      run.derivation_ledger_id,
      run.created_by,
      run.created_at
    ])
  end

  defp record_adapter_derivation(run, asset, tool, opts) do
    asset_ref = DerivationLedgerEntry.object_ref("asset", asset.id)
    run_ref = DerivationLedgerEntry.object_ref("asset_adapter_run", run.id)

    source_refs =
      if is_binary(asset.source_package_id) do
        [DerivationLedgerEntry.object_ref("source_package", asset.source_package_id)]
      else
        []
      end

    ledger =
      DerivationLedgerEntry.new(
        "asset_adapter.#{Atom.to_string(tool.id)}",
        "asset_to_derived_multimodal_output",
        [asset_ref],
        [run_ref],
        tenant_id: asset.tenant_id,
        workspace_id: asset.workspace_id,
        source_package_links: source_refs,
        evidence_links: [asset_ref | source_refs],
        actor_id: Keyword.get(opts, :actor_id),
        parser_id: "optimal_engine.pipeline.multimodal_tool_registry",
        model_id: run.model_id,
        model_version: run.model_version,
        confidence_delta: run.confidence,
        precision_delta: run.precision,
        security_labels: run.security_labels,
        partition_ids: run.partition_ids,
        metadata: %{
          adapter_run_id: run.id,
          adapter_id: run.adapter_id,
          adapter_role: run.adapter_role,
          status: run.status,
          output_hash: run.output_hash
        }
      )

    case OptimalEngine.MemoryCore.Store.insert_derivation_entry(ledger) do
      :ok -> attach_derivation_ledger(run.id, ledger.id)
      {:error, reason} -> {:error, reason}
    end
  end

  defp attach_derivation_ledger(run_id, ledger_id) do
    Store.raw_execute(
      "UPDATE asset_adapter_runs SET derivation_ledger_id = ?1 WHERE id = ?2",
      [ledger_id, run_id]
    )
  end

  defp copy_asset(%Asset{} = asset, opts) do
    root = Application.get_env(:optimal_engine, :root_path, File.cwd!())
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    workspace_slug = workspace_slug(workspace_id)
    workspace_path = Filesystem.path(root, workspace_slug)
    hash_slug = String.replace_prefix(asset.hash, "sha256:", "")
    shard = String.slice(hash_slug, 0, 2) || "00"
    ext = asset.path |> Path.extname() |> String.downcase()
    target_dir = Path.join([workspace_path, "assets", shard])
    target_path = Path.join(target_dir, "#{hash_slug}#{ext}")

    with :ok <- File.mkdir_p(target_dir),
         :ok <- maybe_copy(asset.path, target_path) do
      {:ok, target_path}
    end
  end

  defp maybe_copy(source_path, target_path) do
    if File.exists?(target_path) do
      :ok
    else
      File.cp(source_path, target_path)
    end
  end

  defp record_source_package(%Asset{} = asset, storage_path, opts) do
    tenant_id = Keyword.get(opts, :tenant_id, "default")
    workspace_id = Keyword.get(opts, :workspace_id, "default")

    source_package =
      SourcePackage.from_artifact(
        storage_path,
        asset.hash,
        source_type: Keyword.get(opts, :source_type, "file"),
        source_class: Atom.to_string(asset.modality),
        source_system: Keyword.get(opts, :source_system, "asset_store"),
        source_uri: Keyword.get(opts, :source_uri, asset.path),
        verbatim_archive_uri: storage_path,
        trust_label: Keyword.get(opts, :trust_label, "unreviewed"),
        retention_class: Keyword.get(opts, :retention_class, "standard"),
        access_policy_id: Keyword.get(opts, :access_policy_id),
        security_labels: Keyword.get(opts, :security_labels, []),
        partition_ids: Keyword.get(opts, :partition_ids, []),
        tenant_id: tenant_id,
        workspace_id: workspace_id,
        actor_id: Keyword.get(opts, :actor_id),
        raw_text: Keyword.get(opts, :extracted_text, ""),
        metadata:
          Map.merge(asset.metadata || %{}, %{
            asset_id: ID.content_id("asset", [tenant_id, ":", workspace_id, ":", asset.hash]),
            content_hash: asset.hash,
            content_type: asset.content_type,
            modality: Atom.to_string(asset.modality),
            original_path: asset.path,
            storage_path: storage_path,
            size_bytes: asset.size
          })
      )

    case OptimalEngine.MemoryCore.Store.insert_source_package(source_package) do
      :ok -> {:ok, source_package}
      {:error, reason} -> {:error, reason}
    end
  end

  defp asset_id(%Asset{} = asset, %SourcePackage{} = source_package) do
    ID.content_id("asset", [
      source_package.tenant_id,
      ":",
      source_package.workspace_id,
      ":",
      asset.hash
    ])
  end

  defp upsert_asset(
         asset_id,
         %Asset{} = asset,
         storage_path,
         %SourcePackage{} = source_package,
         opts
       ) do
    sql = """
    INSERT INTO assets (
      id, tenant_id, workspace_id, content_type, size_bytes, storage_path,
      content_hash, modality, source_package_id, original_path, trust_label,
      retention_class, access_policy_id, security_labels, partition_ids, metadata
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10, ?11,
      ?12, ?13, ?14, ?15, ?16
    )
    ON CONFLICT(id) DO UPDATE SET
      tenant_id = excluded.tenant_id,
      workspace_id = excluded.workspace_id,
      content_type = excluded.content_type,
      size_bytes = excluded.size_bytes,
      storage_path = excluded.storage_path,
      content_hash = excluded.content_hash,
      modality = excluded.modality,
      source_package_id = excluded.source_package_id,
      original_path = excluded.original_path,
      trust_label = excluded.trust_label,
      retention_class = excluded.retention_class,
      access_policy_id = excluded.access_policy_id,
      security_labels = excluded.security_labels,
      partition_ids = excluded.partition_ids,
      metadata = excluded.metadata
    """

    Store.raw_execute(sql, [
      asset_id,
      source_package.tenant_id,
      source_package.workspace_id,
      asset.content_type,
      asset.size,
      storage_path,
      asset.hash,
      Atom.to_string(asset.modality),
      source_package.id,
      asset.path,
      Keyword.get(opts, :trust_label, "unreviewed"),
      Keyword.get(opts, :retention_class, "standard"),
      Keyword.get(opts, :access_policy_id),
      JSON.list(Keyword.get(opts, :security_labels, [])),
      JSON.list(Keyword.get(opts, :partition_ids, [])),
      JSON.map(Map.merge(asset.metadata || %{}, Keyword.get(opts, :metadata, %{})))
    ])
  end

  defp record_derivation(asset_id, %Asset{} = asset, %SourcePackage{} = source_package, opts) do
    source_ref = DerivationLedgerEntry.object_ref("source_package", source_package.id)
    asset_ref = DerivationLedgerEntry.object_ref("asset", asset_id)

    ledger =
      DerivationLedgerEntry.new(
        "asset_store.preserve_file",
        "raw_file_to_source_package_asset",
        [],
        [source_ref, asset_ref],
        tenant_id: source_package.tenant_id,
        workspace_id: source_package.workspace_id,
        source_package_links: [source_ref],
        evidence_links: [source_ref, asset_ref],
        actor_id: Keyword.get(opts, :actor_id),
        parser_id: "optimal_engine.memory_core.asset_store",
        access_policy_id: source_package.access_policy_id,
        security_labels: source_package.security_labels,
        partition_ids: source_package.partition_ids,
        metadata: %{
          asset_id: asset_id,
          content_hash: asset.hash,
          modality: Atom.to_string(asset.modality)
        }
      )

    OptimalEngine.MemoryCore.Store.insert_derivation_entry(ledger)
  end

  defp workspace_slug(workspace_id) do
    case Workspace.get(workspace_id) do
      {:ok, workspace} -> workspace.slug
      _ -> workspace_id
    end
  end

  defp row_to_asset([
         id,
         tenant_id,
         workspace_id,
         content_type,
         size_bytes,
         storage_path,
         created_at,
         content_hash,
         modality,
         source_package_id,
         original_path,
         trust_label,
         retention_class,
         access_policy_id,
         security_labels,
         partition_ids,
         metadata
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      content_type: content_type,
      size_bytes: size_bytes,
      storage_path: storage_path,
      created_at: created_at,
      content_hash: content_hash,
      modality: modality,
      source_package_id: source_package_id,
      original_path: original_path,
      trust_label: trust_label,
      retention_class: retention_class,
      access_policy_id: access_policy_id,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      metadata: decode_map(metadata)
    }
  end

  defp row_to_adapter_run([
         id,
         tenant_id,
         workspace_id,
         asset_id,
         source_package_id,
         adapter_id,
         adapter_role,
         modality,
         status,
         started_at,
         completed_at,
         input_hash,
         output_hash,
         output_text,
         output_ref,
         model_id,
         model_version,
         confidence,
         precision,
         error_reason,
         security_labels,
         partition_ids,
         metadata,
         derivation_ledger_id,
         created_by,
         created_at
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      asset_id: asset_id,
      source_package_id: source_package_id,
      adapter_id: adapter_id,
      adapter_role: adapter_role,
      modality: modality,
      status: status,
      started_at: started_at,
      completed_at: completed_at,
      input_hash: input_hash,
      output_hash: output_hash,
      output_text: output_text,
      output_ref: output_ref,
      model_id: model_id,
      model_version: model_version,
      confidence: confidence,
      precision: precision,
      error_reason: error_reason,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      metadata: decode_map(metadata),
      derivation_ledger_id: derivation_ledger_id,
      created_by: created_by,
      created_at: created_at
    }
  end

  defp row_to_asset_extraction([
         id,
         tenant_id,
         workspace_id,
         asset_id,
         source_package_id,
         adapter_run_id,
         extraction_type,
         modality,
         content_text,
         content_ref,
         content_hash,
         confidence,
         precision,
         security_labels,
         partition_ids,
         metadata,
         derivation_ledger_id,
         created_by,
         created_at
       ]) do
    %{
      id: id,
      tenant_id: tenant_id,
      workspace_id: workspace_id,
      asset_id: asset_id,
      source_package_id: source_package_id,
      adapter_run_id: adapter_run_id,
      extraction_type: extraction_type,
      modality: modality,
      content_text: content_text,
      content_ref: content_ref,
      content_hash: content_hash,
      confidence: confidence,
      precision: precision,
      security_labels: decode_list(security_labels),
      partition_ids: decode_list(partition_ids),
      metadata: decode_map(metadata),
      derivation_ledger_id: derivation_ledger_id,
      created_by: created_by,
      created_at: created_at
    }
  end

  defp decode_list(nil), do: []

  defp decode_list(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, list} when is_list(list) -> list
      _ -> []
    end
  end

  defp decode_map(nil), do: %{}

  defp decode_map(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, map} when is_map(map) -> map
      _ -> %{}
    end
  end

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()
end
