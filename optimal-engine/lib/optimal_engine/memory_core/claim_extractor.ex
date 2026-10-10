defmodule OptimalEngine.MemoryCore.ClaimExtractor do
  @moduledoc """
  Source Package -> Claim extraction for the Truth Lifecycle.

  Extraction is intentionally conservative: it does not pretend extraction is
  truth. Source text becomes an unreviewed, pending Claim that only the
  `OptimalEngine.MemoryCore.FactPromoter` review boundary can turn into a
  Fact. Every extraction persists the Source Package, the Claim, a
  `supports` relationship edge, and a derivation ledger entry.
  """

  alias OptimalEngine.MemoryCore.{
    Claim,
    DerivationLedgerEntry,
    ID,
    RelationshipEdge,
    ScoringPolicy,
    SourcePackage,
    Store
  }

  @doc """
  Extract a single Claim from a Source Package.

  Callers may pass structured anchors (`:subject_anchor`, `:action_class`,
  `:object_anchor`) and explicit scores; otherwise the source text becomes an
  unreviewed assertion claim scored by `ScoringPolicy.claim_scores/2`.

  Returns `{:ok, %Claim{}}` with `status: "pending"`.
  """
  @spec extract_from_source(SourcePackage.t(), keyword()) :: {:ok, Claim.t()} | {:error, term()}
  def extract_from_source(%SourcePackage{} = source_package, opts \\ []) do
    claim_text = string_opt(opts, :claim_text, source_package.raw_text)
    now = timestamp()
    scores = ScoringPolicy.claim_scores(source_package, opts)

    claim =
      Claim.new(%{
        id:
          ID.content_id("cl", [
            source_package.tenant_id,
            ":",
            source_package.workspace_id,
            ":",
            source_package.id,
            ":",
            claim_text
          ]),
        tenant_id: source_package.tenant_id,
        workspace_id: source_package.workspace_id,
        source_package_id: source_package.id,
        signal_id: string_or_nil(Keyword.get(opts, :signal_id)),
        claim_text: claim_text,
        claim_type: string_opt(opts, :claim_type, "assertion"),
        subject_anchor: string_or_nil(Keyword.get(opts, :subject_anchor)),
        action_class: string_or_nil(Keyword.get(opts, :action_class)),
        object_anchor: string_or_nil(Keyword.get(opts, :object_anchor)),
        semantic_frame: Keyword.get(opts, :semantic_frame, %{}),
        source_span: Keyword.get(opts, :source_span, %{}),
        extraction_run_id: string_or_nil(Keyword.get(opts, :extraction_run_id)),
        evaluator_id:
          string_or_nil(
            Keyword.get(opts, :evaluator_id) || Keyword.get(opts, :extracted_by) ||
              Keyword.get(opts, :actor_id)
          ),
        aggregate_confidence: scores.confidence,
        aggregate_precision: scores.precision,
        raw_component_scores: scores.raw_component_scores,
        calibration_dataset_version: string_or_nil(scores.calibration_dataset_version),
        access_policy_id: source_package.access_policy_id,
        security_labels: source_package.security_labels,
        partition_ids: source_package.partition_ids,
        lifecycle_state: string_opt(opts, :lifecycle_state, "pending"),
        review_status: string_opt(opts, :review_status, "unreviewed"),
        valid_time_start: serialize_time(Keyword.get(opts, :valid_time_start)),
        valid_time_end: serialize_time(Keyword.get(opts, :valid_time_end)),
        transaction_time_start: now,
        transaction_time_end: nil,
        stale_after: serialize_time(Keyword.get(opts, :stale_after)),
        metadata: Keyword.get(opts, :metadata, %{})
      })

    source_ref = ref("source_package", source_package.id)
    claim_ref = ref("claim", claim.id)

    ledger =
      DerivationLedgerEntry.new(
        "memory_core.extract_claim",
        "source_package_to_claim",
        [source_ref],
        [claim_ref],
        tenant_id: source_package.tenant_id,
        workspace_id: source_package.workspace_id,
        source_package_links: [source_ref],
        evidence_links: [source_ref],
        actor_id: Keyword.get(opts, :actor_id),
        evaluator_id: claim.evaluator_id,
        parser_id: string_opt(opts, :parser_id, "optimal_engine.memory_core.claim_extractor"),
        model_id: string_or_nil(Keyword.get(opts, :model_id)),
        model_version: string_or_nil(Keyword.get(opts, :model_version)),
        confidence_delta: claim.aggregate_confidence,
        precision_delta: claim.aggregate_precision,
        scoring_policy_version: ScoringPolicy.version(),
        access_policy_id: source_package.access_policy_id,
        security_labels: source_package.security_labels,
        partition_ids: source_package.partition_ids,
        metadata: %{claim_type: claim.claim_type}
      )

    edge =
      RelationshipEdge.between(
        source_package,
        {"source_package", source_package.id},
        {"claim", claim.id},
        "supports",
        confidence: claim.aggregate_confidence,
        precision_score: claim.aggregate_precision,
        evidence_links: [source_ref]
      )

    # Emit `depends_on` edges when the caller declares prerequisite claim IDs.
    depends_on_edges =
      opts
      |> Keyword.get(:depends_on_ids, [])
      |> List.wrap()
      |> Enum.map(fn prereq_id ->
        RelationshipEdge.between(
          source_package,
          {"claim", claim.id},
          {"claim", to_string(prereq_id)},
          "depends_on",
          confidence: claim.aggregate_confidence,
          precision_score: claim.aggregate_precision,
          evidence_links: [claim_ref]
        )
      end)

    # Detect conflicting accepted facts to emit `contradicts` edges.
    # A conflict exists when an accepted fact shares subject_anchor +
    # action_class with this claim but carries different object_anchor text.
    # This is purely additive and never blocks the extraction pipeline.
    contradicts_edges = build_contradicts_edges(claim, source_package)

    with :ok <- Store.insert_source_package(source_package),
         :ok <- Store.insert_claim(claim),
         :ok <- Store.insert_relationship_edge(edge),
         :ok <- insert_edges(depends_on_edges),
         :ok <- insert_edges(contradicts_edges),
         :ok <- Store.insert_derivation_entry(ledger) do
      {:ok, claim}
    end
  end

  # Query accepted facts for the same subject_anchor + action_class; emit a
  # `contradicts` edge for each fact whose object_anchor differs from ours.
  # Falls back to `{:ok, []}` on any error so the main pipeline is never blocked.
  defp build_contradicts_edges(%Claim{} = claim, %SourcePackage{} = source_package) do
    if present?(claim.subject_anchor) and present?(claim.action_class) do
      # Query WITHOUT object_anchor so we find facts with the SAME subject+action
      # but a DIFFERENT object — those are the conflicting ones.
      probe = %{
        workspace_id: claim.workspace_id,
        subject_anchor: claim.subject_anchor,
        action_class: claim.action_class,
        object_anchor: nil
      }

      case Store.list_current_facts_for_claim(probe) do
        {:ok, facts} ->
          claim_ref = ref("claim", claim.id)

          facts
          |> Enum.filter(fn fact ->
            present?(fact.object_anchor) and
              present?(claim.object_anchor) and
              fact.object_anchor != claim.object_anchor
          end)
          |> Enum.map(fn fact ->
            RelationshipEdge.between(
              source_package,
              {"claim", claim.id},
              {"fact", fact.id},
              "contradicts",
              confidence: claim.aggregate_confidence,
              precision_score: claim.aggregate_precision,
              evidence_links: [claim_ref]
            )
          end)

        _ ->
          []
      end
    else
      []
    end
  rescue
    _ -> []
  end

  defp insert_edges([]), do: :ok

  defp insert_edges(edges) do
    Enum.reduce_while(edges, :ok, fn edge, :ok ->
      case Store.insert_relationship_edge(edge) do
        :ok -> {:cont, :ok}
        {:error, _} = error -> {:halt, error}
      end
    end)
  end

  defp ref(type, id), do: DerivationLedgerEntry.object_ref(type, id)

  defp timestamp, do: DateTime.utc_now() |> DateTime.to_iso8601()

  defp serialize_time(nil), do: nil
  defp serialize_time(%DateTime{} = value), do: DateTime.to_iso8601(value)
  defp serialize_time(value) when is_binary(value), do: value
  defp serialize_time(value), do: to_string(value)

  defp string_opt(opts, key, default), do: Keyword.get(opts, key, default) |> to_string()

  defp string_or_nil(nil), do: nil
  defp string_or_nil(""), do: nil
  defp string_or_nil(value), do: to_string(value)

  defp present?(nil), do: false
  defp present?(""), do: false
  defp present?(_), do: true
end
