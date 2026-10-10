defmodule OptimalEngine.MemoryCore.KnowledgeLifecycle do
  @moduledoc """
  Deprecated facade for the Truth Lifecycle.

  The Source -> Claim -> Fact -> Memory Object transitions now live in
  domain-verb modules per the Memory Core cleanup rules:

    * `OptimalEngine.MemoryCore.ClaimExtractor.extract_from_source/2`
    * `OptimalEngine.MemoryCore.FactPromoter.promote/2` (and `reject/2`)
    * `OptimalEngine.MemoryCore.MemoryObject.build_from_fact/2`

  This module only delegates. It is NOT behavior-compatible with the legacy
  implementation it replaced:

    * `promote_claim_to_fact/2` now enforces evidence review. Pass
      `:verifier_id` or `:actor_id` (explicit reviewer approval) or opt in
      to auto-promotion with `policy: :auto`, gated by
      `OptimalEngine.MemoryCore.ScoringPolicy.auto_promote_threshold/0`.
      With no review options the call returns `{:error, :approval_required}`
      where the legacy module promoted unconditionally. Self-review (the
      verifier matching the Claim's own `evaluator_id`) is rejected unless
      `allow_self_review: true` is given.
    * `build_memory_object/2` now requires a persisted Fact with
      `lifecycle_state: "accepted"` and non-empty accepted Claim lineage,
      returning `{:error, :fact_not_found}` / `{:error, :fact_not_accepted}`
      otherwise.
    * All three entry points return typed structs (`%Claim{}`, `%Fact{}`,
      `%MemoryObject{}`) instead of plain maps: Access syntax
      (`claim[:id]`) and bare `Jason.encode!/1` on the results no longer
      work.

  Do not add new lifecycle logic here.
  """

  alias OptimalEngine.MemoryCore.{
    ClaimExtractor,
    DerivationLedgerEntry,
    FactPromoter,
    MemoryObject,
    RelationshipEdge,
    ScoringPolicy,
    Store
  }

  @deprecated "Use OptimalEngine.MemoryCore.ClaimExtractor.extract_from_source/2 (returns {:ok, %Claim{}})"
  defdelegate extract_claim(source_package, opts \\ []),
    to: ClaimExtractor,
    as: :extract_from_source

  @deprecated "Use OptimalEngine.MemoryCore.FactPromoter.promote/2 (requires :verifier_id/:actor_id or policy: :auto; returns {:ok, %Fact{}})"
  defdelegate promote_claim_to_fact(claim, opts \\ []), to: FactPromoter, as: :promote

  @deprecated "Use OptimalEngine.MemoryCore.MemoryObject.build_from_fact/2 (requires a persisted accepted Fact; returns {:ok, %MemoryObject{}})"
  defdelegate build_memory_object(fact, opts \\ []), to: MemoryObject, as: :build_from_fact

  def record_fact_supersession(new_fact, old_fact, opts \\ [])
      when is_map(new_fact) and is_map(old_fact) and is_list(opts) do
    workspace_id = Map.get(old_fact, :workspace_id) || "default"
    old_fact_id = Map.fetch!(old_fact, :id)
    new_fact_id = Map.fetch!(new_fact, :id)
    now = DateTime.utc_now() |> DateTime.truncate(:second) |> DateTime.to_iso8601()

    metadata =
      old_fact
      |> Map.get(:metadata, %{})
      |> Map.put("superseded_by", new_fact_id)
      |> Map.put("supersession_reason", Keyword.get(opts, :reason))

    edge =
      RelationshipEdge.between(
        new_fact,
        {"fact", new_fact_id},
        {"fact", old_fact_id},
        "supersedes",
        confidence: Map.get(new_fact, :aggregate_confidence, 0.5),
        precision_score: Map.get(new_fact, :aggregate_precision, 0.5),
        evidence_links: ["fact:#{new_fact_id}", "fact:#{old_fact_id}"]
      )

    ledger =
      DerivationLedgerEntry.new(
        "memory_core.supersede_fact",
        "fact_supersession",
        ["fact:#{new_fact_id}"],
        ["fact:#{old_fact_id}"],
        tenant_id: Map.get(new_fact, :tenant_id, Map.get(old_fact, :tenant_id, "default")),
        workspace_id: workspace_id,
        evidence_links: ["fact:#{new_fact_id}", "fact:#{old_fact_id}"],
        actor_id: Keyword.get(opts, :actor_id),
        evaluator_id: Keyword.get(opts, :evaluator_id),
        scoring_policy_version: ScoringPolicy.version(),
        metadata: %{reason: Keyword.get(opts, :reason), recorded_at: now}
      )

    with :ok <-
           Store.update_fact_supersession(workspace_id, old_fact_id, %{
             lifecycle_state: "superseded",
             contradiction_status: "superseded",
             superseded_by: new_fact_id,
             valid_time_end: Keyword.get(opts, :valid_time_end),
             transaction_time_end: now,
             metadata: metadata
           }),
         :ok <- Store.insert_relationship_edge(edge),
         :ok <- Store.insert_derivation_entry(ledger) do
      :ok
    end
  end
end
