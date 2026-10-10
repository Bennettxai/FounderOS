defmodule OptimalEngine.MemoryCore.MemoryObject do
  @moduledoc """
  Typed Memory Object for the Truth Lifecycle, plus its build verb.

  A Memory Object is the retrievable wrapper around an accepted Fact: a
  summary with salience, full source lineage (fact, claim, and source package
  links), temporal state (staleness and supersession), and separate
  confidence/precision scores. Fields map one-to-one onto the
  `memory_objects` table.

  `build_from_fact/2` is the lifecycle transition: it persists the Memory
  Object, writes `supports` (fact -> memory) and `derived_from`
  (memory -> fact) relationship edges, and records a derivation ledger entry.
  """

  alias OptimalEngine.MemoryCore.{
    DerivationLedgerEntry,
    Fact,
    ID,
    RelationshipEdge,
    ScoringPolicy,
    Store
  }

  @type t :: %__MODULE__{
          id: String.t() | nil,
          tenant_id: String.t(),
          workspace_id: String.t(),
          memory_type: String.t(),
          summary: String.t() | nil,
          subject_anchor: String.t() | nil,
          action_class: String.t() | nil,
          semantic_frame: map(),
          salience: float(),
          fact_links: [map()],
          claim_links: [map()],
          source_package_links: [map()],
          evidence_links: [map()],
          aggregate_confidence: float(),
          aggregate_precision: float(),
          access_policy_id: String.t() | nil,
          security_labels: [String.t()],
          partition_ids: [String.t()],
          lifecycle_state: String.t(),
          staleness_status: String.t(),
          supersession_status: String.t(),
          event_time: String.t() | nil,
          valid_time_start: String.t() | nil,
          valid_time_end: String.t() | nil,
          transaction_time_start: String.t() | nil,
          transaction_time_end: String.t() | nil,
          stale_after: String.t() | nil,
          metadata: map(),
          created_at: String.t() | nil,
          updated_at: String.t() | nil
        }

  defstruct [
    :id,
    :tenant_id,
    :workspace_id,
    :memory_type,
    :summary,
    :subject_anchor,
    :action_class,
    :semantic_frame,
    :salience,
    :fact_links,
    :claim_links,
    :source_package_links,
    :evidence_links,
    :aggregate_confidence,
    :aggregate_precision,
    :access_policy_id,
    :security_labels,
    :partition_ids,
    :lifecycle_state,
    :staleness_status,
    :supersession_status,
    :event_time,
    :valid_time_start,
    :valid_time_end,
    :transaction_time_start,
    :transaction_time_end,
    :stale_after,
    :metadata,
    :created_at,
    :updated_at
  ]

  @doc """
  Build a MemoryObject struct from an attribute map (atom keys).
  """
  @spec new(map()) :: t()
  def new(%__MODULE__{} = memory_object), do: memory_object

  def new(attrs) when is_map(attrs) do
    %__MODULE__{
      id: Map.get(attrs, :id),
      tenant_id: Map.get(attrs, :tenant_id, "default"),
      workspace_id: Map.get(attrs, :workspace_id, "default"),
      memory_type: Map.get(attrs, :memory_type, "general"),
      summary: Map.get(attrs, :summary),
      subject_anchor: Map.get(attrs, :subject_anchor),
      action_class: Map.get(attrs, :action_class),
      semantic_frame: Map.get(attrs, :semantic_frame) || %{},
      salience: Map.get(attrs, :salience) || 0.5,
      fact_links: Map.get(attrs, :fact_links) || [],
      claim_links: Map.get(attrs, :claim_links) || [],
      source_package_links: Map.get(attrs, :source_package_links) || [],
      evidence_links: Map.get(attrs, :evidence_links) || [],
      aggregate_confidence: Map.get(attrs, :aggregate_confidence) || 0.5,
      aggregate_precision: Map.get(attrs, :aggregate_precision) || 0.5,
      access_policy_id: Map.get(attrs, :access_policy_id),
      security_labels: Map.get(attrs, :security_labels) || [],
      partition_ids: Map.get(attrs, :partition_ids) || [],
      lifecycle_state: Map.get(attrs, :lifecycle_state, "candidate"),
      staleness_status: Map.get(attrs, :staleness_status, "current"),
      supersession_status: Map.get(attrs, :supersession_status, "none"),
      event_time: Map.get(attrs, :event_time),
      valid_time_start: Map.get(attrs, :valid_time_start),
      valid_time_end: Map.get(attrs, :valid_time_end),
      transaction_time_start: Map.get(attrs, :transaction_time_start),
      transaction_time_end: Map.get(attrs, :transaction_time_end),
      stale_after: Map.get(attrs, :stale_after),
      metadata: Map.get(attrs, :metadata) || %{},
      created_at: Map.get(attrs, :created_at),
      updated_at: Map.get(attrs, :updated_at)
    }
  end

  @doc """
  Build a Memory Object around an accepted Fact.

  Accepts a `%Fact{}` or a bare fact attribute map, but never trusts the
  caller's copy: the Fact is reloaded via `Store.get_fact/2` and must be a
  persisted Fact with `lifecycle_state: "accepted"` and at least one accepted
  Claim (source lineage). Returns `{:error, :fact_not_found}` when no such
  Fact row exists, and `{:error, :fact_not_accepted}` when the persisted Fact
  is not accepted or carries no accepted Claim lineage.

  Persists the Memory Object with source links back to the Fact, its
  accepted Claims, and the originating Source Packages; writes `supports`
  and `derived_from` relationship edges; and records a
  `memory_core.build_memory_object` derivation ledger entry. Returns
  `{:ok, %MemoryObject{}}`.
  """
  @spec build_from_fact(Fact.t() | map(), keyword()) :: {:ok, t()} | {:error, term()}
  def build_from_fact(fact, opts \\ [])

  def build_from_fact(%Fact{} = fact, opts) do
    with {:ok, fact} <- load_accepted_fact(fact) do
      do_build(fact, opts)
    end
  end

  def build_from_fact(fact, opts) when is_map(fact), do: build_from_fact(Fact.new(fact), opts)

  defp do_build(%Fact{} = fact, opts) do
    now = timestamp()
    summary = string_opt(opts, :summary, fact.fact_text)
    fact_ref = ref("fact", fact.id)
    claim_refs = Enum.map(fact.accepted_claim_ids, &ref("claim", &1))
    source_refs = source_refs_from_fact(fact)
    scores = ScoringPolicy.memory_scores(fact, opts)

    memory =
      new(%{
        id:
          ID.content_id("memobj", [
            fact.tenant_id,
            ":",
            fact.workspace_id,
            ":",
            fact.id,
            ":",
            summary
          ]),
        tenant_id: fact.tenant_id,
        workspace_id: fact.workspace_id,
        memory_type: string_opt(opts, :memory_type, "general"),
        summary: summary,
        subject_anchor: string_or_nil(Keyword.get(opts, :subject_anchor) || fact.subject_anchor),
        action_class: string_or_nil(Keyword.get(opts, :action_class) || fact.action_class),
        semantic_frame: Keyword.get(opts, :semantic_frame, %{}),
        salience: scores.salience,
        fact_links: [fact_ref],
        claim_links: claim_refs,
        source_package_links: source_refs,
        evidence_links: [fact_ref | claim_refs] ++ source_refs,
        aggregate_confidence: scores.confidence,
        aggregate_precision: scores.precision,
        access_policy_id: fact.access_policy_id,
        security_labels: fact.security_labels,
        partition_ids: fact.partition_ids,
        lifecycle_state: string_opt(opts, :lifecycle_state, "current"),
        staleness_status: string_opt(opts, :staleness_status, "current"),
        supersession_status: string_opt(opts, :supersession_status, "none"),
        event_time: serialize_time(Keyword.get(opts, :event_time) || fact.event_time),
        valid_time_start:
          serialize_time(Keyword.get(opts, :valid_time_start) || fact.valid_time_start),
        valid_time_end: serialize_time(Keyword.get(opts, :valid_time_end) || fact.valid_time_end),
        transaction_time_start: now,
        transaction_time_end: nil,
        stale_after: serialize_time(Keyword.get(opts, :stale_after) || fact.stale_after),
        metadata: Keyword.get(opts, :metadata, %{})
      })

    memory_ref = ref("memory_object", memory.id)

    ledger =
      DerivationLedgerEntry.new(
        "memory_core.build_memory_object",
        "fact_to_memory_object",
        [fact_ref],
        [memory_ref],
        tenant_id: memory.tenant_id,
        workspace_id: memory.workspace_id,
        source_package_links: source_refs,
        evidence_links: memory.evidence_links,
        actor_id: Keyword.get(opts, :actor_id),
        scoring_policy_version: ScoringPolicy.version(),
        access_policy_id: memory.access_policy_id,
        security_labels: memory.security_labels,
        partition_ids: memory.partition_ids,
        metadata: %{memory_type: memory.memory_type}
      )

    supports_edge =
      RelationshipEdge.between(fact, {"fact", fact.id}, {"memory_object", memory.id}, "supports",
        confidence: memory.aggregate_confidence,
        precision_score: memory.aggregate_precision,
        evidence_links: memory.evidence_links
      )

    derived_from_edge =
      RelationshipEdge.between(
        fact,
        {"memory_object", memory.id},
        {"fact", fact.id},
        "derived_from",
        confidence: memory.aggregate_confidence,
        precision_score: memory.aggregate_precision,
        evidence_links: [fact_ref]
      )

    # Emit a `part_of` edge when the caller declares that this Memory Object
    # is a component of a composite/parent Memory Object. Purely additive.
    part_of_edges =
      opts
      |> Keyword.get(:part_of_memory_ids, [])
      |> List.wrap()
      |> Enum.map(fn parent_id ->
        RelationshipEdge.between(
          fact,
          {"memory_object", memory.id},
          {"memory_object", to_string(parent_id)},
          "part_of",
          confidence: memory.aggregate_confidence,
          precision_score: memory.aggregate_precision,
          evidence_links: [fact_ref]
        )
      end)

    with :ok <- Store.insert_memory_object(memory),
         :ok <- Store.insert_relationship_edge(supports_edge),
         :ok <- Store.insert_relationship_edge(derived_from_edge),
         :ok <- insert_edges(part_of_edges),
         :ok <- Store.insert_derivation_entry(ledger) do
      {:ok, memory}
    end
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

  defp load_accepted_fact(%Fact{id: id, workspace_id: workspace_id})
       when not is_binary(id) or not is_binary(workspace_id),
       do: {:error, :fact_not_found}

  defp load_accepted_fact(%Fact{} = fact) do
    case Store.get_fact(fact.workspace_id, fact.id) do
      {:ok, %Fact{lifecycle_state: "accepted", accepted_claim_ids: [_ | _]} = stored} ->
        {:ok, stored}

      {:ok, %Fact{}} ->
        {:error, :fact_not_accepted}

      {:error, :not_found} ->
        {:error, :fact_not_found}

      {:error, _} = error ->
        error
    end
  end

  # Normalizes to canonical atom-key refs: the persisted Fact row decodes
  # its evidence links from JSON with string keys.
  defp source_refs_from_fact(%Fact{} = fact) do
    fact.supporting_evidence_links
    |> Enum.flat_map(fn
      %{type: "source_package", id: id} when is_binary(id) -> [id]
      %{"type" => "source_package", "id" => id} when is_binary(id) -> [id]
      _ -> []
    end)
    |> Enum.uniq()
    |> Enum.map(&ref("source_package", &1))
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
end
