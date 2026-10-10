defmodule OptimalEngine.MemoryCore.SourcePackageService do
  @moduledoc """
  Lifecycle operations for Source Packages.

  This module owns the first durable governed-memory flow:

      SourcePackage -> DerivationLedger -> Signal/Context

  `MemoryCore.Store` remains the persistence adapter. This module names the
  domain operation so intake does not need to know how the provenance write is
  assembled.
  """

  alias OptimalEngine.{Context, Signal}
  alias OptimalEngine.MemoryCore.{DerivationLedgerEntry, SourcePackage, Store}

  @spec record_workspace_edit(String.t(), keyword()) ::
          {:ok, SourcePackage.t()} | {:error, term()}
  def record_workspace_edit(raw_text, opts \\ []) when is_binary(raw_text) do
    source_package =
      SourcePackage.from_text(
        raw_text,
        Keyword.merge(
          [
            source_type: "workspace_edit",
            source_class: "text",
            source_system: "workspace_export",
            trust_label: "unreviewed"
          ],
          opts
        )
      )

    source_ref = DerivationLedgerEntry.object_ref("source_package", source_package.id)
    input_refs = edit_input_refs(opts)

    ledger_entry =
      DerivationLedgerEntry.new(
        "workspace_export.capture_edit",
        "projection_edit_to_source_package",
        input_refs,
        [source_ref],
        tenant_id: source_package.tenant_id,
        workspace_id: source_package.workspace_id,
        source_package_links: [source_ref],
        evidence_links: [source_ref],
        actor_id: Keyword.get(opts, :actor_id),
        parser_id: "optimal_engine.workspace_export",
        access_policy_id: source_package.access_policy_id,
        security_labels: source_package.security_labels,
        partition_ids: source_package.partition_ids,
        metadata: workspace_edit_metadata(opts)
      )

    with :ok <- Store.insert_source_package(source_package),
         :ok <- Store.insert_derivation_entry(ledger_entry) do
      {:ok, source_package}
    end
  end

  @spec record_ingested_signal(SourcePackage.t(), Signal.t(), Context.t(), keyword()) ::
          :ok | {:error, term()}
  def record_ingested_signal(
        %SourcePackage{} = source_package,
        %Signal{} = signal,
        %Context{} = context,
        opts \\ []
      ) do
    source_ref = DerivationLedgerEntry.object_ref("source_package", source_package.id)
    signal_ref = DerivationLedgerEntry.object_ref("signal", signal.id)
    context_ref = DerivationLedgerEntry.object_ref("context", context.id)

    ledger_entry =
      DerivationLedgerEntry.new(
        "intake.process",
        "source_to_signal_context",
        [source_ref],
        [signal_ref, context_ref],
        tenant_id: source_package.tenant_id,
        workspace_id: source_package.workspace_id,
        source_package_links: [source_ref],
        evidence_links: [source_ref],
        actor_id: Keyword.get(opts, :actor_id),
        parser_id: "optimal_engine.pipeline.intake",
        model_id: string_or_nil(Keyword.get(opts, :model_id)),
        model_version: string_or_nil(Keyword.get(opts, :model_version)),
        security_labels: source_package.security_labels,
        partition_ids: source_package.partition_ids,
        metadata: ledger_metadata(signal, context)
      )

    with :ok <- Store.insert_source_package(source_package),
         :ok <- Store.insert_derivation_entry(ledger_entry) do
      :ok
    end
  end

  defp ledger_metadata(%Signal{} = signal, %Context{} = context) do
    %{
      node: signal.node,
      genre: signal.genre,
      title: signal.title,
      routed_to: signal.routed_to || [],
      context_uri: context.uri
    }
  end

  defp edit_input_refs(opts) do
    []
    |> maybe_ref("export_record", Keyword.get(opts, :export_record_id))
    |> maybe_ref("projection_revision", Keyword.get(opts, :projection_revision_id))
    |> maybe_ref("node", Keyword.get(opts, :node_id))
  end

  defp maybe_ref(refs, _type, nil), do: refs
  defp maybe_ref(refs, _type, ""), do: refs

  defp maybe_ref(refs, type, id),
    do: refs ++ [DerivationLedgerEntry.object_ref(type, to_string(id))]

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)

  defp workspace_edit_metadata(opts) do
    %{
      edit_type: Keyword.get(opts, :edit_type, "source"),
      source_uri: Keyword.get(opts, :source_uri),
      surface_type: Keyword.get(opts, :surface_type),
      projection_uri: Keyword.get(opts, :projection_uri),
      export_record_id: Keyword.get(opts, :export_record_id),
      projection_revision_id: Keyword.get(opts, :projection_revision_id),
      node_id: Keyword.get(opts, :node_id)
    }
    |> Enum.reject(fn {_key, value} -> is_nil(value) end)
    |> Map.new()
  end
end
