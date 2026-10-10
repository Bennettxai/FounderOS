defmodule OptimalEngine.MemoryCore.FactPromoter do
  @moduledoc """
  Claim -> Fact promotion: the evidence-review boundary of the Truth
  Lifecycle. This module decides what is accepted as true.

  Promotion never deletes or overwrites the Claim. The Fact points back to
  accepted Claim IDs and source evidence, and the Claim row transitions to
  `status: "promoted"` / `review_status: "approved"`. Review is enforced by
  `ScoringPolicy.review_decision/2`: explicit actor approval (`:verifier_id`
  or `:actor_id`) or opt-in policy auto-promotion (`policy: :auto`).

  Both `promote/2` and `reject/2` reload the persisted Claim row and evaluate
  `lifecycle_state` from the database, never from the caller's (possibly
  stale) struct: an unpersisted Claim is `{:error, :claim_not_found}` and a
  Claim without a Source Package is `{:error, :source_lineage_required}`.
  The review transition itself is conditional (`WHERE lifecycle_state =
  'pending'`), so a Claim is reviewed exactly once even under concurrency.

  Facts are bitemporal (`valid_time_start`/`valid_time_end` plus transaction
  times). Promoting a Fact that updates or contradicts an existing accepted
  Fact for the same subject and action marks the old Fact superseded (closed
  transaction time, `supersedes`/`superseded_by` lineage, a `supersedes`
  relationship edge) and marks Memory Objects built on it superseded so
  retrieval stops returning them. Supersession only ever targets accepted
  Facts with an open transaction time; a closed Fact version is immutable
  and can never be re-closed or have its lineage rewritten. The whole
  promote sequence (claim review update, fact insert, supersession, edges,
  ledger) runs in one `OptimalEngine.Store.transaction/1`, so it is atomic
  and serialized. Every transition writes a derivation ledger entry.
  """

  alias OptimalEngine.Store, as: EngineStore

  alias OptimalEngine.MemoryCore.{
    Claim,
    DerivationLedgerEntry,
    Fact,
    ID,
    JSON,
    RelationshipEdge,
    ScoringPolicy,
    Store
  }

  @doc """
  Promote a Claim into a Fact.

  Accepts a `%Claim{}`, a bare claim attribute map, or a claim ID (the review
  queue path; requires `:workspace_id` in opts so the Claim can be loaded).
  Struct and map inputs are reloaded from the claims table before review:
  an unpersisted Claim returns `{:error, :claim_not_found}` and a Claim
  without source lineage returns `{:error, :source_lineage_required}`.

  Options:

    * `:verifier_id` / `:actor_id` - explicit reviewer approval
    * `policy: :auto` - allow policy auto-promotion above the confidence
      threshold (see `ScoringPolicy.auto_promote_threshold/0`)
    * `:supersedes` - explicit fact ID(s) this promotion supersedes (only
      accepted facts with an open transaction time are eligible)
    * `detect_supersession: false` - disable implicit same-subject detection
    * `:fact_text`, `:valid_time_start`, `:valid_time_end`, `:event_time`,
      `:aggregate_confidence`, `:aggregate_precision`, and the other fact
      fields as overrides

  Returns `{:ok, %Fact{}}` or `{:error, reason}` (see
  `ScoringPolicy.review_decision/2` for review errors).
  """
  @spec promote(Claim.t() | map() | String.t(), keyword()) :: {:ok, Fact.t()} | {:error, term()}
  def promote(claim_or_id, opts \\ [])

  def promote(claim_id, opts) when is_binary(claim_id) do
    with {:ok, workspace_id} <- require_workspace_id(opts),
         {:ok, claim} <- load_persisted_claim(workspace_id, claim_id, opts[:tenant_id]) do
      promote_persisted(claim, opts)
    end
  end

  def promote(%Claim{} = claim, opts) do
    with :ok <- require_source_lineage(claim),
         {:ok, persisted} <- reload_claim(claim) do
      promote_persisted(persisted, opts)
    end
  end

  def promote(claim, opts) when is_map(claim), do: promote(Claim.new(claim), opts)

  @doc """
  Reject a Claim in review.

  The persisted Claim row is reloaded first (stale structs cannot bypass
  review state): rejecting an already promoted Claim is
  `{:error, :claim_already_promoted}` and an unpersisted Claim is
  `{:error, :claim_not_found}`. The Claim row transitions to
  `status: "rejected"` / `review_status: "rejected"` and a
  `memory_core.reject_claim` ledger entry records the decision. Requires
  `:verifier_id` or `:actor_id`.
  """
  @spec reject(Claim.t() | map() | String.t(), keyword()) :: {:ok, Claim.t()} | {:error, term()}
  def reject(claim_or_id, opts \\ [])

  def reject(claim_id, opts) when is_binary(claim_id) do
    with {:ok, workspace_id} <- require_workspace_id(opts),
         {:ok, claim} <- load_persisted_claim(workspace_id, claim_id, opts[:tenant_id]) do
      reject_persisted(claim, opts)
    end
  end

  def reject(%Claim{} = claim, opts) do
    with {:ok, persisted} <- reload_claim(claim) do
      reject_persisted(persisted, opts)
    end
  end

  def reject(claim, opts) when is_map(claim), do: reject(Claim.new(claim), opts)

  defp promote_persisted(%Claim{} = claim, opts) do
    with :ok <- require_source_lineage(claim),
         {:ok, review} <- ScoringPolicy.review_decision(claim, opts) do
      do_promote(claim, review, opts)
    else
      {:error, {:contradicts_current_facts, fact_ids}} = error ->
        # Record `contradicts` edges from the claim to each blocking fact so the
        # contradiction is visible in the relationship graph even when promotion
        # is blocked. Additive only — never changes the promotion outcome.
        emit_contradicts_edges(claim, fact_ids, opts)
        error

      {:error, _} = error ->
        error
    end
  end

  defp reject_persisted(%Claim{lifecycle_state: "promoted"}, _opts),
    do: {:error, :claim_already_promoted}

  defp reject_persisted(%Claim{} = claim, opts) do
    reviewer = string_or_nil(Keyword.get(opts, :verifier_id) || Keyword.get(opts, :actor_id))

    if is_nil(reviewer) do
      {:error, :reviewer_required}
    else
      claim_ref = ref("claim", claim.id)

      ledger =
        DerivationLedgerEntry.new(
          "memory_core.reject_claim",
          "claim_review",
          [claim_ref],
          [claim_ref],
          tenant_id: claim.tenant_id,
          workspace_id: claim.workspace_id,
          source_package_links: source_refs(claim),
          evidence_links: [claim_ref],
          actor_id: Keyword.get(opts, :actor_id),
          evaluator_id: reviewer,
          scoring_policy_version: ScoringPolicy.version(),
          access_policy_id: claim.access_policy_id,
          security_labels: claim.security_labels,
          partition_ids: claim.partition_ids,
          metadata: %{review_basis: "actor_rejection", reason: Keyword.get(opts, :reason)}
        )

      EngineStore.transaction(fn txn ->
        with :ok <-
               Store.update_claim_review(claim.workspace_id, claim.id, "rejected", "rejected", txn),
             :ok <- txn_insert_derivation_entry(txn, ledger) do
          {:ok, Claim.new(%{claim | lifecycle_state: "rejected", review_status: "rejected"})}
        end
      end)
    end
  end

  defp do_promote(%Claim{} = claim, review, opts) do
    now = timestamp()
    fact_text = string_opt(opts, :fact_text, claim.claim_text)
    claim_ref = ref("claim", claim.id)
    source_ref = ref("source_package", claim.source_package_id)
    scores = ScoringPolicy.promotion_scores(claim, opts)

    fact_id =
      ID.content_id("fact", [
        claim.tenant_id,
        ":",
        claim.workspace_id,
        ":",
        claim.id,
        ":",
        fact_text
      ])

    EngineStore.transaction(fn txn ->
      with {:ok, superseded_facts} <- supersession_targets(txn, claim, fact_id, opts) do
        fact =
          build_fact(claim, review, scores, opts, %{
            id: fact_id,
            text: fact_text,
            now: now,
            supersedes: Enum.map(superseded_facts, & &1.id)
          })

        fact_ref = ref("fact", fact.id)

        ledger =
          DerivationLedgerEntry.new(
            "memory_core.promote_fact",
            "claim_to_fact",
            [claim_ref],
            [fact_ref],
            tenant_id: fact.tenant_id,
            workspace_id: fact.workspace_id,
            source_package_links: [source_ref],
            evidence_links: [source_ref, claim_ref],
            actor_id: Keyword.get(opts, :actor_id),
            evaluator_id: fact.verifier_id,
            confidence_delta: scores.confidence_delta,
            precision_delta: scores.precision_delta,
            scoring_policy_version: ScoringPolicy.version(),
            access_policy_id: fact.access_policy_id,
            security_labels: fact.security_labels,
            partition_ids: fact.partition_ids,
            metadata: %{
              verification_status: fact.verification_status,
              review_basis: review.review_basis,
              effective_threshold: review.effective_threshold,
              claim_confidence: review.claim_confidence,
              supersedes: fact.supersedes
            }
          )

        edge =
          RelationshipEdge.between(claim, {"claim", claim.id}, {"fact", fact.id}, "supports",
            confidence: fact.aggregate_confidence,
            precision_score: fact.aggregate_precision,
            evidence_links: [source_ref, claim_ref]
          )

        with :ok <-
               Store.update_claim_review(claim.workspace_id, claim.id, "promoted", "approved", txn),
             :ok <- txn_insert_fact(txn, fact),
             :ok <- txn_insert_relationship_edge(txn, edge),
             :ok <- txn_insert_derivation_entry(txn, ledger),
             :ok <- txn_supersede_facts(txn, fact, superseded_facts, review, opts, now) do
          {:ok, fact}
        end
      end
    end)
  end

  defp build_fact(%Claim{} = claim, review, scores, opts, built) do
    Fact.new(%{
      id: built.id,
      tenant_id: claim.tenant_id,
      workspace_id: claim.workspace_id,
      fact_text: built.text,
      fact_type: string_opt(opts, :fact_type, claim.claim_type || "assertion"),
      subject_anchor: string_or_nil(Keyword.get(opts, :subject_anchor) || claim.subject_anchor),
      action_class: string_or_nil(Keyword.get(opts, :action_class) || claim.action_class),
      object_anchor: string_or_nil(Keyword.get(opts, :object_anchor) || claim.object_anchor),
      scope: Keyword.get(opts, :scope, %{}),
      accepted_claim_ids: [claim.id],
      supporting_evidence_links: [
        ref("source_package", claim.source_package_id),
        ref("claim", claim.id)
      ],
      contradicting_evidence_links: Keyword.get(opts, :contradicting_evidence_links, []),
      verifier_id: review.verifier_id,
      verification_status: string_opt(opts, :verification_status, "verified"),
      aggregate_confidence: scores.confidence,
      aggregate_precision: scores.precision,
      raw_component_scores: scores.raw_component_scores,
      access_policy_id: claim.access_policy_id,
      security_labels: claim.security_labels,
      partition_ids: claim.partition_ids,
      lifecycle_state: string_opt(opts, :lifecycle_state, "accepted"),
      contradiction_status: string_opt(opts, :contradiction_status, "none"),
      event_time: serialize_time(Keyword.get(opts, :event_time)),
      valid_time_start:
        serialize_time(Keyword.get(opts, :valid_time_start) || claim.valid_time_start) ||
          built.now,
      valid_time_end: serialize_time(Keyword.get(opts, :valid_time_end) || claim.valid_time_end),
      transaction_time_start: built.now,
      transaction_time_end: nil,
      verification_time: serialize_time(Keyword.get(opts, :verification_time)) || built.now,
      stale_after: serialize_time(Keyword.get(opts, :stale_after)),
      metadata: Keyword.get(opts, :metadata, %{}),
      supersedes: built.supersedes
    })
  end

  # Supersession targets are read inside the promote transaction and are
  # restricted to accepted facts with an open transaction time: a closed
  # bitemporal fact version is immutable, so explicit `:supersedes` IDs that
  # point at already-superseded (or otherwise closed) facts are dropped.
  defp supersession_targets(txn, %Claim{} = claim, fact_id, opts) do
    explicit_ids =
      opts
      |> Keyword.get(:supersedes, [])
      |> List.wrap()
      |> Enum.map(&to_string/1)

    with {:ok, explicit} <- explicit_targets(txn, claim, explicit_ids),
         {:ok, implicit} <- implicit_targets(txn, claim, opts) do
      targets =
        (explicit ++ implicit)
        |> Enum.uniq_by(& &1.id)
        |> Enum.reject(&(&1.id == fact_id))

      {:ok, targets}
    end
  end

  defp explicit_targets(_txn, _claim, []), do: {:ok, []}

  defp explicit_targets(txn, claim, ids) do
    sql = """
    SELECT id, valid_time_end, metadata
    FROM facts
    WHERE workspace_id = ?1 AND tenant_id = ?2 AND id = ?3
      AND lifecycle_state = 'accepted' AND transaction_time_end IS NULL
    """

    Enum.reduce_while(ids, {:ok, []}, fn id, {:ok, acc} ->
      case EngineStore.txn_query(txn, sql, [claim.workspace_id, claim.tenant_id, id]) do
        {:ok, rows} -> {:cont, {:ok, acc ++ Enum.map(rows, &target_from_row/1)}}
        {:error, _} = error -> {:halt, error}
      end
    end)
  end

  defp implicit_targets(txn, claim, opts) do
    if Keyword.get(opts, :detect_supersession, true) and present?(claim.subject_anchor) and
         present?(claim.action_class) do
      sql = """
      SELECT id, valid_time_end, metadata
      FROM facts
      WHERE workspace_id = ?1 AND tenant_id = ?2
        AND subject_anchor = ?3 AND action_class = ?4
        AND lifecycle_state = 'accepted' AND transaction_time_end IS NULL
      ORDER BY created_at, id
      """

      with {:ok, rows} <-
             EngineStore.txn_query(txn, sql, [
               claim.workspace_id,
               claim.tenant_id,
               claim.subject_anchor,
               claim.action_class
             ]) do
        {:ok, Enum.map(rows, &target_from_row/1)}
      end
    else
      {:ok, []}
    end
  end

  defp target_from_row([id, valid_time_end, metadata]) do
    %{id: id, valid_time_end: valid_time_end, metadata: decode_map(metadata)}
  end

  defp txn_supersede_facts(_txn, _new_fact, [], _review, _opts, _now), do: :ok

  defp txn_supersede_facts(txn, new_fact, superseded_facts, review, opts, now) do
    Enum.reduce_while(superseded_facts, :ok, fn old_fact, :ok ->
      case txn_supersede_fact(txn, new_fact, old_fact, review, opts, now) do
        :ok -> {:cont, :ok}
        error -> {:halt, error}
      end
    end)
  end

  defp txn_supersede_fact(txn, new_fact, old_fact, review, opts, now) do
    metadata = Map.put(old_fact.metadata, "superseded_by", new_fact.id)
    new_fact_ref = ref("fact", new_fact.id)
    old_fact_ref = ref("fact", old_fact.id)

    edge =
      RelationshipEdge.between(new_fact, {"fact", new_fact.id}, {"fact", old_fact.id}, "supersedes",
        confidence: new_fact.aggregate_confidence,
        precision_score: new_fact.aggregate_precision,
        evidence_links: [new_fact_ref, old_fact_ref]
      )

    with :ok <- txn_close_superseded_fact(txn, new_fact.workspace_id, old_fact, metadata, now),
         :ok <- txn_insert_relationship_edge(txn, edge),
         {:ok, superseded_memory_ids} <-
           txn_supersede_memory_objects(txn, new_fact, old_fact.id, now) do
      ledger =
        DerivationLedgerEntry.new(
          "memory_core.supersede_fact",
          "fact_supersession",
          [new_fact_ref],
          [old_fact_ref],
          tenant_id: new_fact.tenant_id,
          workspace_id: new_fact.workspace_id,
          evidence_links: [new_fact_ref, old_fact_ref],
          actor_id: Keyword.get(opts, :actor_id),
          evaluator_id: review.verifier_id,
          scoring_policy_version: ScoringPolicy.version(),
          access_policy_id: new_fact.access_policy_id,
          security_labels: new_fact.security_labels,
          partition_ids: new_fact.partition_ids,
          metadata: %{
            superseded_by: new_fact.id,
            superseded_memory_object_ids: superseded_memory_ids
          }
        )

      txn_insert_derivation_entry(txn, ledger)
    end
  end

  # The UPDATE re-checks `lifecycle_state = 'accepted' AND
  # transaction_time_end IS NULL`, so a closed fact version can never be
  # re-closed or have its `superseded_by` lineage overwritten.
  defp txn_close_superseded_fact(txn, workspace_id, old_fact, metadata, now) do
    sql = """
    UPDATE facts
    SET lifecycle_state = 'superseded',
        contradiction_status = 'superseded',
        valid_time_end = ?3,
        transaction_time_end = ?4,
        metadata = ?5,
        updated_at = datetime('now')
    WHERE workspace_id = ?1 AND id = ?2
      AND lifecycle_state = 'accepted' AND transaction_time_end IS NULL
    """

    case EngineStore.txn_execute(txn, sql, [
           workspace_id,
           old_fact.id,
           old_fact.valid_time_end || now,
           now,
           JSON.map(metadata)
         ]) do
      {:ok, _changes} -> :ok
      {:error, _} = error -> error
    end
  end

  defp txn_supersede_memory_objects(txn, new_fact, fact_id, now) do
    select_sql = """
    SELECT id, fact_links
    FROM memory_objects
    WHERE workspace_id = ?1 AND tenant_id = ?2
      AND supersession_status = 'none' AND fact_links LIKE ?3
    """

    update_sql = """
    UPDATE memory_objects
    SET supersession_status = 'superseded',
        lifecycle_state = 'superseded',
        staleness_status = 'stale',
        transaction_time_end = ?3,
        updated_at = datetime('now')
    WHERE workspace_id = ?1 AND id = ?2 AND supersession_status = 'none'
    """

    with {:ok, rows} <-
           EngineStore.txn_query(txn, select_sql, [
             new_fact.workspace_id,
             new_fact.tenant_id,
             "%#{fact_id}%"
           ]) do
      rows
      |> Enum.filter(fn [_id, fact_links] -> links_include?(fact_links, fact_id) end)
      |> Enum.reduce_while({:ok, []}, fn [memory_id, _links], {:ok, acc} ->
        case EngineStore.txn_execute(txn, update_sql, [new_fact.workspace_id, memory_id, now]) do
          {:ok, _changes} -> {:cont, {:ok, [memory_id | acc]}}
          {:error, _} = error -> {:halt, error}
        end
      end)
    end
  end

  defp links_include?(fact_links, fact_id) do
    fact_links
    |> decode_list()
    |> Enum.any?(fn
      %{"id" => id} -> id == fact_id
      _ -> false
    end)
  end

  # -- transactional inserts (mirror MemoryCore.Store SQL, but run through
  # -- the opaque handle so the whole promote sequence is one transaction)

  defp txn_insert_fact(txn, %Fact{} = fact) do
    sql = """
    INSERT OR IGNORE INTO facts (
      id, tenant_id, workspace_id, fact_text, fact_type, subject_anchor,
      action_class, object_anchor, scope, accepted_claim_ids,
      supporting_evidence_links, contradicting_evidence_links, verifier_id,
      verification_status, aggregate_confidence, aggregate_precision,
      raw_component_scores, access_policy_id, security_labels, partition_ids,
      lifecycle_state, contradiction_status, event_time, valid_time_start,
      valid_time_end, transaction_time_start, transaction_time_end,
      verification_time, stale_after, metadata
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5, ?6,
      ?7, ?8, ?9, ?10,
      ?11, ?12, ?13,
      ?14, ?15, ?16,
      ?17, ?18, ?19, ?20,
      ?21, ?22, ?23, ?24,
      ?25, ?26, ?27,
      ?28, ?29, ?30
    )
    """

    txn_exec(txn, sql, [
      fact.id,
      fact.tenant_id,
      fact.workspace_id,
      fact.fact_text,
      fact.fact_type || "assertion",
      fact.subject_anchor,
      fact.action_class,
      fact.object_anchor,
      JSON.map(fact.scope),
      JSON.list(fact.accepted_claim_ids),
      JSON.list(fact.supporting_evidence_links),
      JSON.list(fact.contradicting_evidence_links),
      fact.verifier_id,
      fact.verification_status || "unverified",
      fact.aggregate_confidence || 0.5,
      fact.aggregate_precision || 0.5,
      JSON.map(fact.raw_component_scores),
      fact.access_policy_id,
      JSON.list(fact.security_labels),
      JSON.list(fact.partition_ids),
      fact.lifecycle_state || "candidate",
      fact.contradiction_status || "none",
      fact.event_time,
      fact.valid_time_start,
      fact.valid_time_end,
      fact.transaction_time_start || timestamp(),
      fact.transaction_time_end,
      fact.verification_time,
      fact.stale_after,
      JSON.map(fact.metadata)
    ])
  end

  defp txn_insert_relationship_edge(txn, %RelationshipEdge{} = edge) do
    sql = """
    INSERT OR IGNORE INTO relationship_edges (
      id, tenant_id, workspace_id, from_object_type, from_object_id,
      to_object_type, to_object_id, relationship_type, confidence,
      precision_score, evidence_links, access_policy_id, security_labels,
      partition_ids, lifecycle_state, valid_time_start, valid_time_end,
      transaction_time_start, transaction_time_end, metadata
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5,
      ?6, ?7, ?8, ?9,
      ?10, ?11, ?12, ?13,
      ?14, ?15, ?16, ?17,
      ?18, ?19, ?20
    )
    """

    txn_exec(txn, sql, [
      edge.id,
      edge.tenant_id,
      edge.workspace_id,
      edge.from_object_type,
      edge.from_object_id,
      edge.to_object_type,
      edge.to_object_id,
      edge.relationship_type,
      edge.confidence || 0.5,
      edge.precision_score || 0.5,
      JSON.list(edge.evidence_links),
      edge.access_policy_id,
      JSON.list(edge.security_labels),
      JSON.list(edge.partition_ids),
      edge.lifecycle_state || "current",
      edge.valid_time_start,
      edge.valid_time_end,
      edge.transaction_time_start || timestamp(),
      edge.transaction_time_end,
      JSON.map(edge.metadata)
    ])
  end

  defp txn_insert_derivation_entry(txn, %DerivationLedgerEntry{} = entry) do
    sql = """
    INSERT INTO derivation_ledger (
      id, tenant_id, workspace_id, activity_type, derivation_stage,
      input_object_links, output_object_links, source_package_links, evidence_links,
      actor_id, evaluator_id, parser_id, model_id, model_version, prompt_template_id,
      tool_call_links, confidence_delta, precision_delta, scoring_policy_version,
      access_policy_id, security_labels, partition_ids, lifecycle_state, replay_status,
      activity_time, transaction_time_start, transaction_time_end, audit_event_links,
      policy_version, metadata, created_at
    ) VALUES (
      ?1, ?2, ?3, ?4, ?5,
      ?6, ?7, ?8, ?9,
      ?10, ?11, ?12, ?13, ?14, ?15,
      ?16, ?17, ?18, ?19,
      ?20, ?21, ?22, ?23, ?24,
      ?25, ?26, ?27, ?28,
      ?29, ?30, ?31
    )
    """

    txn_exec(txn, sql, [
      entry.id,
      entry.tenant_id,
      entry.workspace_id,
      entry.activity_type,
      entry.derivation_stage,
      JSON.list(entry.input_object_links),
      JSON.list(entry.output_object_links),
      JSON.list(entry.source_package_links),
      JSON.list(entry.evidence_links),
      entry.actor_id,
      entry.evaluator_id,
      entry.parser_id,
      entry.model_id,
      entry.model_version,
      entry.prompt_template_id,
      JSON.list(entry.tool_call_links),
      entry.confidence_delta,
      entry.precision_delta,
      entry.scoring_policy_version,
      entry.access_policy_id,
      JSON.list(entry.security_labels),
      JSON.list(entry.partition_ids),
      entry.lifecycle_state,
      entry.replay_status,
      entry.activity_time,
      entry.transaction_time_start,
      entry.transaction_time_end,
      JSON.list(entry.audit_event_links),
      entry.policy_version,
      JSON.map(entry.metadata),
      entry.created_at
    ])
  end

  defp txn_exec(txn, sql, params) do
    case EngineStore.txn_execute(txn, sql, params) do
      {:ok, _changes} -> :ok
      {:error, _} = error -> error
    end
  end

  # -- claim loading: lifecycle state is always evaluated from the DB row

  defp reload_claim(%Claim{} = claim) do
    load_persisted_claim(
      string_or_nil(claim.workspace_id),
      string_or_nil(claim.id),
      claim.tenant_id
    )
  end

  defp load_persisted_claim(workspace_id, claim_id, tenant_id) do
    if is_nil(workspace_id) or is_nil(claim_id) do
      {:error, :claim_not_found}
    else
      opts =
        case string_or_nil(tenant_id) do
          nil -> []
          tenant -> [tenant_id: tenant]
        end

      case Store.get_claim(workspace_id, claim_id, opts) do
        {:ok, %Claim{} = persisted} -> {:ok, persisted}
        {:error, :not_found} -> {:error, :claim_not_found}
        {:error, _} = error -> error
      end
    end
  end

  defp require_workspace_id(opts) do
    case string_or_nil(Keyword.get(opts, :workspace_id)) do
      nil -> {:error, :workspace_id_required}
      workspace_id -> {:ok, workspace_id}
    end
  end

  defp require_source_lineage(%Claim{source_package_id: source_package_id})
       when is_binary(source_package_id) and source_package_id != "",
       do: :ok

  defp require_source_lineage(_claim), do: {:error, :source_lineage_required}

  defp source_refs(%Claim{source_package_id: nil}), do: []
  defp source_refs(%Claim{source_package_id: id}), do: [ref("source_package", id)]

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

  # Emit `contradicts` relationship edges from the claim to each blocking fact.
  # Runs outside any transaction; each INSERT OR IGNORE is safe to call alone.
  defp emit_contradicts_edges(%Claim{} = claim, fact_ids, opts) do
    claim_ref = ref("claim", claim.id)

    Enum.each(fact_ids, fn fact_id ->
      edge =
        RelationshipEdge.between(
          claim,
          {"claim", claim.id},
          {"fact", to_string(fact_id)},
          "contradicts",
          confidence: claim.aggregate_confidence || 0.5,
          precision_score: claim.aggregate_precision || 0.5,
          evidence_links: [claim_ref],
          metadata: %{promotion_blocked: true, actor_id: Keyword.get(opts, :actor_id)}
        )

      Store.insert_relationship_edge(edge)
    end)
  rescue
    _ -> :ok
  end

  defp decode_map(nil), do: %{}
  defp decode_map(""), do: %{}

  defp decode_map(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_map(decoded) -> decoded
      _ -> %{}
    end
  end

  defp decode_map(value) when is_map(value), do: value

  defp decode_list(nil), do: []
  defp decode_list(""), do: []

  defp decode_list(value) when is_binary(value) do
    case Jason.decode(value) do
      {:ok, decoded} when is_list(decoded) -> decoded
      {:ok, decoded} -> [decoded]
      _ -> []
    end
  end

  defp decode_list(value) when is_list(value), do: value
end
